package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeExternalRevenueRequestUsesMinorUnitsAndWhitelistedFields(t *testing.T) {
	record, err := normalizeExternalRevenueRequest(externalRevenueMutationRequest{
		Source:          " XIANQU ",
		ExternalOrderNo: " order-1001 ",
		AmountMinor:     "1205",
		Currency:        "cny",
		OccurredAt:      1_725_148_800,
		Note:            " private receipt ",
	}, 7, 1_725_148_900)
	assert.Nil(t, record)
	assert.EqualError(t, err, "unsupported external revenue source")

	record, err = normalizeExternalRevenueRequest(externalRevenueMutationRequest{
		Source:          " XIANyu ",
		ExternalOrderNo: " order-1001 ",
		AmountMinor:     "1205",
		Currency:        "cny",
		OccurredAt:      1_725_148_800,
		Note:            " private receipt ",
	}, 7, 1_725_148_900)
	require.NoError(t, err)
	require.NotNil(t, record)
	assert.Equal(t, int64(1205), record.AmountMinor)
	assert.Equal(t, "CNY", record.Currency)
	require.NotNil(t, record.ExternalOrderNo)
	assert.Equal(t, "order-1001", *record.ExternalOrderNo)
	assert.Equal(t, 7, record.CreatedBy)
	assert.Equal(t, "private receipt", record.Note)
}

func TestNormalizeExternalRevenueRequestRejectsInvalidReportingData(t *testing.T) {
	base := externalRevenueMutationRequest{
		Source:      "other",
		AmountMinor: "100",
		Currency:    "CNY",
		OccurredAt:  1_725_148_800,
	}

	_, err := normalizeExternalRevenueRequest(base, 1, 1_725_148_900)
	assert.EqualError(t, err, "source label is required for other revenue")

	base.SourceLabel = "offline"
	base.AmountMinor = "-1"
	_, err = normalizeExternalRevenueRequest(base, 1, 1_725_148_900)
	assert.EqualError(t, err, "amount must be a positive minor-unit integer within the supported range")
}
