package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildEpayOrderPricingUsesExactOrderSnapshots(t *testing.T) {
	originalPrice := operation_setting.Price
	originalQuotaPerUnit := common.QuotaPerUnit
	originalDisplayType := operation_setting.GetGeneralSetting().QuotaDisplayType
	originalDiscounts := operation_setting.GetPaymentSetting().AmountDiscount
	originalRatios := common.TopupGroupRatio2JSONString()
	t.Cleanup(func() {
		operation_setting.Price = originalPrice
		common.QuotaPerUnit = originalQuotaPerUnit
		operation_setting.GetGeneralSetting().QuotaDisplayType = originalDisplayType
		operation_setting.GetPaymentSetting().AmountDiscount = originalDiscounts
		require.NoError(t, common.UpdateTopupGroupRatioByJSONString(originalRatios))
	})

	operation_setting.Price = 7.3
	common.QuotaPerUnit = 500_000
	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeUSD
	operation_setting.GetPaymentSetting().AmountDiscount = map[int]float64{10: 0.9}
	require.NoError(t, common.UpdateTopupGroupRatioByJSONString(`{"vip":1.2}`))

	pricing, err := buildEpayOrderPricing(10, "vip")
	require.NoError(t, err)
	assert.Equal(t, "78.84", pricing.PayMoney)
	assert.Equal(t, int64(7884), pricing.ExpectedAmountMinor)
	assert.Equal(t, 5_000_000, pricing.QuotaAmount)
	assert.Equal(t, "7.3", pricing.UnitPriceSnapshot)
	assert.Equal(t, "500000", pricing.QuotaPerUnitSnapshot)
	assert.Equal(t, "1.2", pricing.TopUpGroupRatioSnapshot)
	assert.Equal(t, "0.9", pricing.AmountDiscountSnapshot)
}

func TestBuildEpayOrderPricingPreservesTokenQuota(t *testing.T) {
	originalPrice := operation_setting.Price
	originalQuotaPerUnit := common.QuotaPerUnit
	originalDisplayType := operation_setting.GetGeneralSetting().QuotaDisplayType
	originalDiscounts := operation_setting.GetPaymentSetting().AmountDiscount
	t.Cleanup(func() {
		operation_setting.Price = originalPrice
		common.QuotaPerUnit = originalQuotaPerUnit
		operation_setting.GetGeneralSetting().QuotaDisplayType = originalDisplayType
		operation_setting.GetPaymentSetting().AmountDiscount = originalDiscounts
	})

	operation_setting.Price = 7.3
	common.QuotaPerUnit = 500_000
	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeTokens
	operation_setting.GetPaymentSetting().AmountDiscount = map[int]float64{}

	pricing, err := buildEpayOrderPricing(750_000, "default")
	require.NoError(t, err)
	assert.Equal(t, "10.95", pricing.PayMoney)
	assert.Equal(t, int64(1095), pricing.ExpectedAmountMinor)
	assert.Equal(t, 750_000, pricing.QuotaAmount)
}

func TestBuildEpayOrderPricingRejectsInvalidFinancialInputs(t *testing.T) {
	originalPrice := operation_setting.Price
	originalQuotaPerUnit := common.QuotaPerUnit
	t.Cleanup(func() {
		operation_setting.Price = originalPrice
		common.QuotaPerUnit = originalQuotaPerUnit
	})

	operation_setting.Price = 0
	common.QuotaPerUnit = 500_000
	_, err := buildEpayOrderPricing(10, "default")
	assert.Error(t, err)

	operation_setting.Price = 7.3
	common.QuotaPerUnit = 0
	_, err = buildEpayOrderPricing(10, "default")
	assert.Error(t, err)
}
