package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLegacyEpayMigrationBlocksPendingOrdersAndVerifiesHistoricalEvidence(t *testing.T) {
	truncateTables(t)
	user := User{
		Username: "legacy-epay-migration", AffCode: "legacy-epay-migration",
		Status: common.UserStatusEnabled,
	}
	require.NoError(t, DB.Create(&user).Error)
	legacyPending := TopUp{
		UserId: user.Id, Amount: 10, Money: 73, TradeNo: "legacy-epay-pending",
		PaymentMethod: "alipay", PaymentProvider: PaymentProviderEpay,
		Status: common.TopUpStatusPending,
	}
	partialPending := TopUp{
		UserId: user.Id, Amount: 10, Money: 73, TradeNo: "legacy-epay-partial",
		PaymentMethod: "alipay", PaymentProvider: PaymentProviderEpay,
		ExpectedAmountMinor: 1, Status: common.TopUpStatusPending,
	}
	require.NoError(t, DB.Create(&legacyPending).Error)
	require.NoError(t, DB.Create(&partialPending).Error)

	err := ensureNoLegacyPendingEpayOrders()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "2 legacy pending Epay orders")
	require.NoError(t, DB.Model(&TopUp{}).
		Where("id IN ?", []int{legacyPending.Id, partialPending.Id}).
		Update("status", "closed").Error)
	require.NoError(t, ensureNoLegacyPendingEpayOrders())

	historicalProviderTrade := "legacy-epay-provider-success"
	historicalSuccess := TopUp{
		UserId: user.Id, TradeNo: "legacy-epay-success", Money: 43.6905,
		PaymentMethod: "alipay", PaymentProvider: PaymentProviderEpay,
		PaidAmountMinor: 4_369, PaidCurrency: "CNY", ProviderTradeNo: &historicalProviderTrade,
		Status: common.TopUpStatusSuccess,
	}
	historicalMismatchProviderTrade := "legacy-epay-provider-mismatch"
	historicalMismatch := TopUp{
		UserId: user.Id, TradeNo: "legacy-epay-success-mismatch", Money: 30,
		PaymentMethod: "alipay", PaymentProvider: PaymentProviderEpay,
		PaidAmountMinor: 2_500, PaidCurrency: "CNY", ProviderTradeNo: &historicalMismatchProviderTrade,
		Status: common.TopUpStatusSuccess,
	}
	historicalWithoutEvidence := TopUp{
		UserId: user.Id, TradeNo: "legacy-epay-success-no-evidence", Money: 25,
		PaymentMethod: "alipay", PaymentProvider: PaymentProviderEpay,
		PaidAmountMinor: 2_500, PaidCurrency: "CNY", Status: common.TopUpStatusSuccess,
	}
	require.NoError(t, DB.Create(&[]TopUp{
		historicalSuccess,
		historicalMismatch,
		historicalWithoutEvidence,
	}).Error)
	require.NoError(t, DB.Model(&TopUp{}).
		Where("trade_no = ?", historicalSuccess.TradeNo).
		UpdateColumn("completion_source", nil).Error)

	result, err := backfillLegacyEpayEvidence()
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.SuccessBackfilled)
	assert.Equal(t, int64(1), result.SuccessBlocked)

	var migratedHistorical TopUp
	require.NoError(t, DB.Where("trade_no = ?", historicalSuccess.TradeNo).First(&migratedHistorical).Error)
	assert.Equal(t, CompletionSourceWebhook, migratedHistorical.CompletionSource)
	var mismatchedHistorical TopUp
	require.NoError(t, DB.Where("trade_no = ?", historicalMismatch.TradeNo).First(&mismatchedHistorical).Error)
	assert.Empty(t, mismatchedHistorical.CompletionSource)
	var unverifiedHistorical TopUp
	require.NoError(t, DB.Where("trade_no = ?", historicalWithoutEvidence.TradeNo).First(&unverifiedHistorical).Error)
	assert.Empty(t, unverifiedHistorical.CompletionSource)
	totals, err := GetVerifiedTopUpPaymentTotals(user.Id, PaymentProviderEpay)
	require.NoError(t, err)
	require.Len(t, totals, 1)
	assert.Equal(t, int64(4_369), totals[0].AmountMinor)

	replay, err := backfillLegacyEpayEvidence()
	require.NoError(t, err)
	assert.Zero(t, replay.SuccessBackfilled)
	assert.Zero(t, replay.SuccessBlocked)
	var marker Option
	require.NoError(t, DB.Where(commonKeyCol+" = ?", legacyEpayEvidenceMigrationMarker).First(&marker).Error)
	assert.Equal(t, "1", marker.Value)
}

func TestLegacyEpayPreflightAcceptsCompletePendingSnapshot(t *testing.T) {
	truncateTables(t)
	user := User{Username: "current-epay-pending", AffCode: "current-epay-pending"}
	require.NoError(t, DB.Create(&user).Error)
	createPendingEpayTopUp(t, "current-epay-pending", user.Id)
	require.NoError(t, ensureNoLegacyPendingEpayOrders())
	require.NoError(t, DB.Model(&TopUp{}).
		Where("trade_no = ?", "current-epay-pending").
		UpdateColumn("paid_currency", nil).Error)
	require.Error(t, ensureNoLegacyPendingEpayOrders())
}
