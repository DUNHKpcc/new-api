package model

import (
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaymentProviderTopUpContractMatrix(t *testing.T) {
	previousQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 1000
	t.Cleanup(func() {
		common.QuotaPerUnit = previousQuotaPerUnit
	})

	providers := []struct {
		name          string
		provider      string
		expectedQuota int
		complete      func(string) error
	}{
		{
			name:          "stripe",
			provider:      PaymentProviderStripe,
			expectedQuota: 9990,
			complete: func(tradeNo string) error {
				return Recharge(tradeNo, "cus_contract", "127.0.0.1")
			},
		},
		{
			name:          "creem",
			provider:      PaymentProviderCreem,
			expectedQuota: 2,
			complete: func(tradeNo string) error {
				return RechargeCreem(tradeNo, "contract@example.com", "Contract User", "127.0.0.1")
			},
		},
		{
			name:          "waffo",
			provider:      PaymentProviderWaffo,
			expectedQuota: 2000,
			complete: func(tradeNo string) error {
				return RechargeWaffo(tradeNo, "127.0.0.1")
			},
		},
		{
			name:          "waffo pancake",
			provider:      PaymentProviderWaffoPancake,
			expectedQuota: 2000,
			complete:      RechargeWaffoPancake,
		},
	}

	for index, provider := range providers {
		t.Run(provider.name, func(t *testing.T) {
			truncateTables(t)
			userID := 700 + index
			tradeNo := fmt.Sprintf("provider-contract-%d", index)
			insertUserForPaymentGuardTest(t, userID, 10)
			insertTopUpForPaymentGuardTest(t, tradeNo, userID, provider.provider)

			require.NoError(t, provider.complete(tradeNo))
			assert.Equal(t, 10+provider.expectedQuota, getUserQuotaForPaymentGuardTest(t, userID))
			assert.Equal(t, common.TopUpStatusSuccess, getTopUpStatusForPaymentGuardTest(t, tradeNo))

			_ = provider.complete(tradeNo)
			assert.Equal(t, 10+provider.expectedQuota, getUserQuotaForPaymentGuardTest(t, userID),
				"duplicate delivery must never credit quota twice")
		})
	}
}

func TestPaymentProviderTerminalStatusRejectsOutOfOrderSuccess(t *testing.T) {
	previousQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 1000
	t.Cleanup(func() {
		common.QuotaPerUnit = previousQuotaPerUnit
	})

	providers := []struct {
		name     string
		provider string
		complete func(string) error
	}{
		{
			name:     "stripe expired before success",
			provider: PaymentProviderStripe,
			complete: func(tradeNo string) error {
				return Recharge(tradeNo, "cus_contract", "127.0.0.1")
			},
		},
		{
			name:     "creem failed before success",
			provider: PaymentProviderCreem,
			complete: func(tradeNo string) error {
				return RechargeCreem(tradeNo, "", "", "127.0.0.1")
			},
		},
		{
			name:     "waffo expired before success",
			provider: PaymentProviderWaffo,
			complete: func(tradeNo string) error {
				return RechargeWaffo(tradeNo, "127.0.0.1")
			},
		},
		{
			name:     "waffo pancake failed before success",
			provider: PaymentProviderWaffoPancake,
			complete: RechargeWaffoPancake,
		},
	}

	for index, provider := range providers {
		t.Run(provider.name, func(t *testing.T) {
			truncateTables(t)
			userID := 800 + index
			tradeNo := fmt.Sprintf("provider-out-of-order-%d", index)
			insertUserForPaymentGuardTest(t, userID, 10)
			insertTopUpForPaymentGuardTest(t, tradeNo, userID, provider.provider)
			targetStatus := common.TopUpStatusExpired
			if index%2 == 1 {
				targetStatus = common.TopUpStatusFailed
			}
			require.NoError(t, UpdatePendingTopUpStatus(tradeNo, provider.provider, targetStatus))

			require.Error(t, provider.complete(tradeNo))
			assert.Equal(t, 10, getUserQuotaForPaymentGuardTest(t, userID))
			assert.Equal(t, targetStatus, getTopUpStatusForPaymentGuardTest(t, tradeNo))
		})
	}
}
