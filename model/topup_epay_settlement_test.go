package model

import (
	"errors"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func createPendingEpayTopUp(t *testing.T, tradeNo string, userID int) TopUp {
	t.Helper()
	topUp := TopUp{
		UserId:                  userID,
		Amount:                  10,
		Money:                   73,
		TradeNo:                 tradeNo,
		PaymentMethod:           "alipay",
		PaymentProvider:         PaymentProviderEpay,
		ExpectedAmountMinor:     7300,
		PaidCurrency:            "CNY",
		QuotaAmount:             5_000_000,
		UnitPriceSnapshot:       "7.3",
		QuotaPerUnitSnapshot:    "500000",
		TopUpGroupRatioSnapshot: "1",
		AmountDiscountSnapshot:  "1",
		CommissionEligible:      true,
		CreateTime:              100,
		Status:                  common.TopUpStatusPending,
	}
	require.NoError(t, DB.Create(&topUp).Error)
	return topUp
}

func TestCompleteEpayTopUpCreditsExactlyOnce(t *testing.T) {
	truncateTables(t)
	user := User{Username: "epay-user", AffCode: "epay-user", Quota: 100}
	require.NoError(t, DB.Create(&user).Error)
	topUp := createPendingEpayTopUp(t, "epay-order-1", user.Id)

	input := EpaySettlement{
		TradeNo:         topUp.TradeNo,
		ProviderTradeNo: "provider-1",
		PaidAmountMinor: 7300,
		PaymentMethod:   "alipay",
		CompletedAt:     200,
	}
	first, err := CompleteEpayTopUp(input)
	require.NoError(t, err)
	assert.False(t, first.AlreadyCompleted)
	assert.Equal(t, topUp.QuotaAmount, first.QuotaAdded)
	require.NoError(t, DB.Create(&Option{
		Key: operation_setting.AffiliateSettingOptionKey, Value: `{}`,
	}).Error)

	second, err := CompleteEpayTopUp(input)
	require.NoError(t, err)
	assert.True(t, second.AlreadyCompleted)
	assert.Zero(t, second.QuotaAdded)
	input.PaymentMethod = "wxpay"
	_, err = CompleteEpayTopUp(input)
	assert.ErrorIs(t, err, ErrPaymentMethodMismatch)

	var reloadedTopUp TopUp
	require.NoError(t, DB.First(&reloadedTopUp, topUp.Id).Error)
	assert.Equal(t, common.TopUpStatusSuccess, reloadedTopUp.Status)
	assert.Equal(t, CompletionSourceWebhook, reloadedTopUp.CompletionSource)
	assert.Equal(t, int64(200), reloadedTopUp.CompleteTime)
	assert.Equal(t, int64(7300), reloadedTopUp.PaidAmountMinor)
	require.NotNil(t, reloadedTopUp.ProviderTradeNo)
	assert.Equal(t, "provider-1", *reloadedTopUp.ProviderTradeNo)

	var reloadedUser User
	require.NoError(t, DB.First(&reloadedUser, user.Id).Error)
	assert.Equal(t, 100+topUp.QuotaAmount, reloadedUser.Quota)
}

func TestCompleteEpayTopUpRejectsInvalidEvidenceWithoutMutation(t *testing.T) {
	testCases := []struct {
		name   string
		mutate func(*EpaySettlement)
		want   error
	}{
		{
			name: "payment method mismatch",
			mutate: func(input *EpaySettlement) {
				input.PaymentMethod = "wxpay"
			},
			want: ErrPaymentMethodMismatch,
		},
		{
			name: "missing provider trade number",
			mutate: func(input *EpaySettlement) {
				input.ProviderTradeNo = ""
			},
			want: ErrEpayProviderTradeNoRequired,
		},
		{
			name: "zero amount",
			mutate: func(input *EpaySettlement) {
				input.PaidAmountMinor = 0
			},
			want: ErrEpayPaidAmountInvalid,
		},
		{
			name: "amount mismatch",
			mutate: func(input *EpaySettlement) {
				input.PaidAmountMinor = 7299
			},
			want: ErrEpayPaidAmountMismatch,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			truncateTables(t)
			user := User{Username: "epay-" + testCase.name, AffCode: "aff-" + testCase.name, Quota: 50}
			require.NoError(t, DB.Create(&user).Error)
			topUp := createPendingEpayTopUp(t, "order-"+testCase.name, user.Id)
			input := EpaySettlement{
				TradeNo:         topUp.TradeNo,
				ProviderTradeNo: "provider-" + testCase.name,
				PaidAmountMinor: topUp.ExpectedAmountMinor,
				PaymentMethod:   "alipay",
				CompletedAt:     200,
			}
			testCase.mutate(&input)

			_, err := CompleteEpayTopUp(input)
			assert.ErrorIs(t, err, testCase.want)

			var reloadedTopUp TopUp
			require.NoError(t, DB.First(&reloadedTopUp, topUp.Id).Error)
			assert.Equal(t, common.TopUpStatusPending, reloadedTopUp.Status)
			assert.Zero(t, reloadedTopUp.PaidAmountMinor)
			assert.Nil(t, reloadedTopUp.ProviderTradeNo)

			var reloadedUser User
			require.NoError(t, DB.First(&reloadedUser, user.Id).Error)
			assert.Equal(t, 50, reloadedUser.Quota)
		})
	}
}

func TestCompleteEpayTopUpRollsBackOrderWhenQuotaUpdateFails(t *testing.T) {
	truncateTables(t)
	user := User{Username: "epay-rollback", AffCode: "epay-rollback", Quota: 50}
	require.NoError(t, DB.Create(&user).Error)
	topUp := createPendingEpayTopUp(t, "epay-rollback-order", user.Id)

	callbackName := "test:fail_epay_quota_update"
	require.NoError(t, DB.Callback().Update().Before("gorm:update").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement.Table == "users" {
			tx.AddError(errors.New("injected quota update failure"))
		}
	}))
	t.Cleanup(func() {
		require.NoError(t, DB.Callback().Update().Remove(callbackName))
	})

	_, err := CompleteEpayTopUp(EpaySettlement{
		TradeNo:         topUp.TradeNo,
		ProviderTradeNo: "provider-rollback",
		PaidAmountMinor: topUp.ExpectedAmountMinor,
		PaymentMethod:   "alipay",
		CompletedAt:     200,
	})
	require.Error(t, err)

	var reloadedTopUp TopUp
	require.NoError(t, DB.First(&reloadedTopUp, topUp.Id).Error)
	assert.Equal(t, common.TopUpStatusPending, reloadedTopUp.Status)
	assert.Nil(t, reloadedTopUp.ProviderTradeNo)

	var reloadedUser User
	require.NoError(t, DB.First(&reloadedUser, user.Id).Error)
	assert.Equal(t, 50, reloadedUser.Quota)
}

func TestCompleteEpayTopUpRejectsProviderTradeReuse(t *testing.T) {
	truncateTables(t)
	user := User{Username: "epay-provider-reuse", AffCode: "epay-provider-reuse"}
	require.NoError(t, DB.Create(&user).Error)
	first := createPendingEpayTopUp(t, "epay-provider-order-1", user.Id)
	second := createPendingEpayTopUp(t, "epay-provider-order-2", user.Id)

	settle := func(topUp TopUp) error {
		_, err := CompleteEpayTopUp(EpaySettlement{
			TradeNo:         topUp.TradeNo,
			ProviderTradeNo: "provider-reused",
			PaidAmountMinor: topUp.ExpectedAmountMinor,
			PaymentMethod:   "alipay",
			CompletedAt:     200,
		})
		return err
	}
	require.NoError(t, settle(first))
	require.Error(t, settle(second))

	var reloadedUser User
	require.NoError(t, DB.First(&reloadedUser, user.Id).Error)
	assert.Equal(t, first.QuotaAmount, reloadedUser.Quota)
	var reloadedSecond TopUp
	require.NoError(t, DB.First(&reloadedSecond, second.Id).Error)
	assert.Equal(t, common.TopUpStatusPending, reloadedSecond.Status)
}

func TestManualCompleteTopUpRejectsMissingOrOverflowingUser(t *testing.T) {
	t.Run("quota overflow", func(t *testing.T) {
		truncateTables(t)
		user := User{Username: "manual-overflow-user", AffCode: "manual-overflow", Quota: common.MaxQuota}
		require.NoError(t, DB.Create(&user).Error)
		topUp := createPendingEpayTopUp(t, "manual-overflow-order", user.Id)

		assert.ErrorIs(t, ManualCompleteTopUp(topUp.TradeNo, "127.0.0.1"), ErrEpayQuotaOverflow)
		var reloaded TopUp
		require.NoError(t, DB.First(&reloaded, topUp.Id).Error)
		assert.Equal(t, common.TopUpStatusPending, reloaded.Status)
	})

	t.Run("missing user", func(t *testing.T) {
		truncateTables(t)
		topUp := createPendingEpayTopUp(t, "manual-missing-user-order", 999_999)

		require.Error(t, ManualCompleteTopUp(topUp.TradeNo, "127.0.0.1"))
		var reloaded TopUp
		require.NoError(t, DB.First(&reloaded, topUp.Id).Error)
		assert.Equal(t, common.TopUpStatusPending, reloaded.Status)
	})
}
