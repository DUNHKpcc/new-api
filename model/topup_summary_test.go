package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func providerTradeNo(value string) *string {
	return &value
}

func TestGetVerifiedTopUpPaymentTotals(t *testing.T) {
	truncateTables(t)

	topups := []TopUp{
		{
			UserId: 101, TradeNo: "epay-cny-1", PaymentProvider: PaymentProviderEpay,
			Status: common.TopUpStatusSuccess, PaidAmountMinor: 1200, PaidCurrency: "CNY",
			ProviderTradeNo: providerTradeNo("provider-1"), CompletionSource: CompletionSourceWebhook, CreateTime: 100,
		},
		{
			UserId: 101, TradeNo: "epay-cny-2", PaymentProvider: PaymentProviderEpay,
			Status: common.TopUpStatusSuccess, PaidAmountMinor: 34, PaidCurrency: "CNY",
			ProviderTradeNo: providerTradeNo("provider-2"), CompletionSource: CompletionSourceWebhook, CreateTime: 200,
		},
		{
			UserId: 101, TradeNo: "epay-usd", PaymentProvider: PaymentProviderEpay,
			Status: common.TopUpStatusSuccess, PaidAmountMinor: 500, PaidCurrency: "USD",
			ProviderTradeNo: providerTradeNo("provider-3"), CompletionSource: CompletionSourceWebhook, CreateTime: 300,
		},
		{
			UserId: 101, TradeNo: "pending", PaymentProvider: PaymentProviderEpay,
			Status: common.TopUpStatusPending, PaidAmountMinor: 9999, PaidCurrency: "CNY",
			ProviderTradeNo: providerTradeNo("provider-pending"), CompletionSource: CompletionSourceWebhook, CreateTime: 400,
		},
		{
			UserId: 101, TradeNo: "manual-completion", PaymentProvider: PaymentProviderEpay,
			Status: common.TopUpStatusSuccess, Money: 88, CreateTime: 500,
		},
		{
			UserId: 101, TradeNo: "missing-provider-reference", PaymentProvider: PaymentProviderEpay,
			Status: common.TopUpStatusSuccess, CompletionSource: CompletionSourceWebhook, PaidAmountMinor: 8800, PaidCurrency: "CNY",
			CreateTime: 510,
		},
		{
			UserId: 101, TradeNo: "missing-paid-amount", PaymentProvider: PaymentProviderEpay,
			Status: common.TopUpStatusSuccess, CompletionSource: CompletionSourceWebhook, PaidCurrency: "CNY",
			ProviderTradeNo: providerTradeNo("provider-missing-amount"), CreateTime: 520,
		},
		{
			UserId: 101, TradeNo: "missing-currency", PaymentProvider: PaymentProviderEpay,
			Status: common.TopUpStatusSuccess, CompletionSource: CompletionSourceWebhook, PaidAmountMinor: 8800,
			ProviderTradeNo: providerTradeNo("provider-missing-currency"), CreateTime: 530,
		},
		{
			UserId: 101, TradeNo: "stripe", PaymentProvider: PaymentProviderStripe,
			Status: common.TopUpStatusSuccess, PaidAmountMinor: 700, PaidCurrency: "USD",
			ProviderTradeNo: providerTradeNo("stripe-provider"), CompletionSource: CompletionSourceWebhook, CreateTime: 600,
		},
		{
			UserId: 202, TradeNo: "other-user", PaymentProvider: PaymentProviderEpay,
			Status: common.TopUpStatusSuccess, PaidAmountMinor: 900, PaidCurrency: "CNY",
			ProviderTradeNo: providerTradeNo("provider-other"), CompletionSource: CompletionSourceWebhook, CreateTime: 50,
		},
		{
			UserId: 101, TradeNo: "admin-with-evidence", PaymentProvider: PaymentProviderEpay,
			Status: common.TopUpStatusSuccess, PaidAmountMinor: 9900, PaidCurrency: "CNY",
			ProviderTradeNo: providerTradeNo("provider-admin"), CompletionSource: CompletionSourceAdmin, CreateTime: 610,
		},
	}
	require.NoError(t, DB.Create(&topups).Error)

	epayTotals, err := GetVerifiedTopUpPaymentTotals(101, PaymentProviderEpay)
	require.NoError(t, err)
	require.Len(t, epayTotals, 2)
	assert.Equal(t, VerifiedTopUpPaymentTotal{
		PaymentProvider: PaymentProviderEpay,
		Currency:        "CNY",
		AmountMinor:     1234,
	}, epayTotals[0])
	assert.Equal(t, int64(500), epayTotals[1].AmountMinor)
	assert.Equal(t, "USD", epayTotals[1].Currency)
	encoded, err := common.Marshal(epayTotals[0])
	require.NoError(t, err)
	assert.Contains(t, string(encoded), `"amount_minor":"1234"`)
	topUpJSON, err := common.Marshal(topups[0])
	require.NoError(t, err)
	assert.NotContains(t, string(topUpJSON), "paid_amount_minor")
	assert.NotContains(t, string(topUpJSON), "provider_trade_no")

	allTotals, err := GetVerifiedTopUpPaymentTotals(101)
	require.NoError(t, err)
	require.Len(t, allTotals, 3)
	assert.Equal(t, PaymentProviderStripe, allTotals[2].PaymentProvider)
}
