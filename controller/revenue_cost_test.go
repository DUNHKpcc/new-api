package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeRevenueCostRequestNormalizesSelectedCategoryAndCurrency(t *testing.T) {
	record, err := normalizeRevenueCostRequest(revenueCostMutationRequest{
		Month:       " 2026-08 ",
		Category:    " SERVER ",
		Description: "  monthly host  ",
		AmountMinor: "1205",
		Currency:    "cny",
	}, 7, 1_725_148_900)
	require.NoError(t, err)
	require.NotNil(t, record)
	assert.Equal(t, "2026-08", record.Month)
	assert.Equal(t, "server", record.Category)
	assert.Equal(t, "monthly host", record.Description)
	assert.Equal(t, int64(1205), record.AmountMinor)
	assert.Equal(t, "CNY", record.Currency)
	assert.Equal(t, 7, record.CreatedBy)
}

func TestNormalizeRevenueCostRequestRejectsInvalidCostDimensions(t *testing.T) {
	base := revenueCostMutationRequest{
		Month:       "2026-08",
		Category:    "server",
		AmountMinor: "100",
		Currency:    "CNY",
	}

	_, err := normalizeRevenueCostRequest(base, 1, 1_725_148_900)
	assert.NoError(t, err)

	base.Month = "2026-8"
	_, err = normalizeRevenueCostRequest(base, 1, 1_725_148_900)
	assert.EqualError(t, err, "month must use YYYY-MM format")

	base.Month = "2026-08"
	base.Category = "hosting"
	_, err = normalizeRevenueCostRequest(base, 1, 1_725_148_900)
	assert.EqualError(t, err, "unsupported revenue cost category")

	base.Category = "server"
	base.AmountMinor = "-1"
	_, err = normalizeRevenueCostRequest(base, 1, 1_725_148_900)
	assert.EqualError(t, err, "amount must be a non-negative minor-unit integer within the supported range")
}
