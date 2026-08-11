package model

import (
	"errors"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestCompleteEpayTopUpRollsBackEveryAffiliateWriteBoundary(t *testing.T) {
	testCases := []struct {
		name         string
		callbackKind string
		targetTable  string
		startingDebt int64
	}{
		{name: "commission create", callbackKind: "create", targetTable: "affiliate_commissions"},
		{name: "commission outbox create", callbackKind: "create", targetTable: "affiliate_outbox_events"},
		{name: "commission debt update", callbackKind: "update", targetTable: "affiliate_profiles", startingDebt: 1_000},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			truncateTables(t)
			useAffiliateSettingForTopUpTest(t, 0)
			agent, referred := createAffiliateUsersAndProfile(t, 100)
			if testCase.startingDebt > 0 {
				require.NoError(t, DB.Model(&AffiliateProfile{}).
					Where("user_id = ?", agent.Id).
					Update("commission_debt_quota", testCase.startingDebt).Error)
			}
			order := createAffiliateEpayOrder(t, referred, "settlement-failure-"+testCase.name, 5_000)

			callbackName := "test:fail_settlement_" + testCase.callbackKind + "_" + testCase.targetTable
			injected := errors.New("injected affiliate settlement failure")
			callback := func(tx *gorm.DB) {
				if tx.Statement.Table == testCase.targetTable {
					tx.AddError(injected)
				}
			}
			var remove func() error
			if testCase.callbackKind == "create" {
				require.NoError(t, DB.Callback().Create().Before("gorm:create").Register(callbackName, callback))
				remove = func() error { return DB.Callback().Create().Remove(callbackName) }
			} else {
				require.NoError(t, DB.Callback().Update().Before("gorm:update").Register(callbackName, callback))
				remove = func() error { return DB.Callback().Update().Remove(callbackName) }
			}
			callbackInstalled := true
			t.Cleanup(func() {
				if callbackInstalled {
					_ = remove()
				}
			})

			input := EpaySettlement{
				TradeNo: order.TradeNo, ProviderTradeNo: "provider-" + testCase.name,
				PaidAmountMinor: 5_000, PaymentMethod: "alipay", CompletedAt: 200,
			}
			_, err := CompleteEpayTopUp(input)
			require.Error(t, err)

			var reloadedTopUp TopUp
			require.NoError(t, DB.First(&reloadedTopUp, order.Id).Error)
			assert.Equal(t, common.TopUpStatusPending, reloadedTopUp.Status)
			assert.Nil(t, reloadedTopUp.ProviderTradeNo)
			var reloadedReferred User
			require.NoError(t, DB.First(&reloadedReferred, referred.Id).Error)
			assert.Equal(t, referred.Quota, reloadedReferred.Quota)
			var profile AffiliateProfile
			require.NoError(t, DB.First(&profile, agent.Id).Error)
			assert.Equal(t, testCase.startingDebt, profile.CommissionDebtQuota)
			var commissionCount int64
			require.NoError(t, DB.Model(&AffiliateCommission{}).Count(&commissionCount).Error)
			assert.Zero(t, commissionCount)
			var outboxCount int64
			require.NoError(t, DB.Model(&AffiliateOutboxEvent{}).Count(&outboxCount).Error)
			assert.Zero(t, outboxCount)

			require.NoError(t, remove())
			callbackInstalled = false
			_, err = CompleteEpayTopUp(input)
			require.NoError(t, err)
			require.NoError(t, DB.First(&reloadedReferred, referred.Id).Error)
			assert.Equal(t, referred.Quota+order.QuotaAmount, reloadedReferred.Quota)
		})
	}
}

func TestReverseAffiliateCommissionDoesNotCreateDebtBeforeTransfer(t *testing.T) {
	for _, waitDays := range []int64{0, 7} {
		name := "available"
		if waitDays > 0 {
			name = "pending"
		}
		t.Run(name, func(t *testing.T) {
			truncateTables(t)
			useAffiliateSettingForTopUpTest(t, waitDays)
			agent, referred := createAffiliateUsersAndProfile(t, 100)
			order := createAffiliateEpayOrder(t, referred, "reverse-before-transfer-"+name, 5_000)
			settlement, err := CompleteEpayTopUp(EpaySettlement{
				TradeNo: order.TradeNo, ProviderTradeNo: "provider-reverse-before-transfer-" + name,
				PaidAmountMinor: 5_000, PaymentMethod: "alipay", CompletedAt: 200,
			})
			require.NoError(t, err)

			result, err := ReverseAffiliateCommission(
				settlement.CommissionID,
				agent.Id,
				"commission correction",
				300,
			)
			require.NoError(t, err)
			assert.Zero(t, result.Reversal.DebtAddedQuota)
			assert.Equal(t, AffiliateCommissionStatusReversed, result.Commission.Status)
			var profile AffiliateProfile
			require.NoError(t, DB.First(&profile, agent.Id).Error)
			assert.Zero(t, profile.CommissionDebtQuota)
			var reloadedAgent User
			require.NoError(t, DB.First(&reloadedAgent, agent.Id).Error)
			assert.Equal(t, agent.Quota, reloadedAgent.Quota)
		})
	}
}
