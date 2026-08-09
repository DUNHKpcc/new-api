package controller

import (
	"testing"

	"github.com/Calcium-Ion/go-epay/epay"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseEpayPaidAmountMinor(t *testing.T) {
	tests := []struct {
		name      string
		money     string
		wantMinor int64
		wantError bool
	}{
		{name: "two decimals", money: "12.34", wantMinor: 1234},
		{name: "minimum amount", money: "0.01", wantMinor: 1},
		{name: "trailing zeros", money: "12.3400", wantMinor: 1234},
		{name: "zero", money: "0", wantError: true},
		{name: "negative", money: "-1.00", wantError: true},
		{name: "fractional minor unit", money: "1.001", wantError: true},
		{name: "not a number", money: "invalid", wantError: true},
		{name: "out of range", money: "92233720368547758.08", wantError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			amountMinor, err := parseEpayPaidAmountMinor(test.money)
			if test.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.wantMinor, amountMinor)
		})
	}
}

func TestApplyEpayVerifiedPaymentOnlySetsPaymentEvidence(t *testing.T) {
	topUp := &model.TopUp{Amount: 10, Money: 73}
	verifyInfo := &epay.VerifyRes{TradeNo: "epay-provider-1", Money: "72.50"}

	actualMoney, err := applyEpayVerifiedPayment(topUp, verifyInfo)
	require.NoError(t, err)
	assert.Equal(t, int64(7250), topUp.PaidAmountMinor)
	assert.Equal(t, "CNY", topUp.PaidCurrency)
	assert.Equal(t, "epay-provider-1", topUp.ProviderTradeNo)
	assert.Equal(t, int64(10), topUp.Amount)
	assert.Equal(t, float64(73), topUp.Money)
	assert.Equal(t, "72.5", actualMoney.String())
}
