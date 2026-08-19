package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAllTopUpsFiltersByCompletionRangeBeforePagination(t *testing.T) {
	truncateTables(t)

	topUps := []TopUp{
		{TradeNo: "july-order", CompleteTime: 1_722_470_400, Status: common.TopUpStatusSuccess},
		{TradeNo: "august-first", CompleteTime: 1_722_556_800, Status: common.TopUpStatusSuccess},
		{TradeNo: "august-second", CompleteTime: 1_725_148_799, Status: common.TopUpStatusSuccess},
		{TradeNo: "september-order", CompleteTime: 1_725_148_800, Status: common.TopUpStatusSuccess},
	}
	require.NoError(t, DB.Create(&topUps).Error)

	pageInfo := &common.PageInfo{Page: 1, PageSize: 20}
	records, total, err := GetAllTopUps(pageInfo, 1_722_556_800, 1_725_148_800)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	require.Len(t, records, 2)
	assert.Equal(t, "august-second", records[0].TradeNo)
	assert.Equal(t, "august-first", records[1].TradeNo)
}

func TestSearchAllTopUpsCombinesKeywordAndCompletionRange(t *testing.T) {
	truncateTables(t)

	topUps := []TopUp{
		{TradeNo: "august-target", CompleteTime: 1_722_556_800, Status: common.TopUpStatusSuccess},
		{TradeNo: "august-other", CompleteTime: 1_722_556_801, Status: common.TopUpStatusSuccess},
		{TradeNo: "july-target", CompleteTime: 1_722_470_400, Status: common.TopUpStatusSuccess},
	}
	require.NoError(t, DB.Create(&topUps).Error)

	pageInfo := &common.PageInfo{Page: 1, PageSize: 20}
	records, total, err := SearchAllTopUps(
		"%target%",
		pageInfo,
		1_722_556_800,
		1_725_148_800,
	)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, records, 1)
	assert.Equal(t, "august-target", records[0].TradeNo)
}
