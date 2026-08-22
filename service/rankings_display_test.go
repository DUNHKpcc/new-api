package service

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyRankingDisplayConfigAddsTokensWithoutMutatingLiveSnapshot(t *testing.T) {
	previousRank := 2
	live := &RankingsResponse{
		DataMode: RankingDataModeLive,
		Models: []RankedModel{
			{Rank: 1, ModelName: "model-a", Vendor: "Vendor A", TotalTokens: 100, Share: 0.4, previousTokens: 100},
			{Rank: 2, PreviousRank: &previousRank, ModelName: "model-b", Vendor: "Vendor B", TotalTokens: 50, Share: 0.2, previousTokens: 100},
		},
		Vendors: []RankedVendor{
			{Rank: 1, Vendor: "Vendor A", TotalTokens: 150, TopModel: "model-a", previousTokens: 150},
			{Rank: 2, Vendor: "Vendor B", TotalTokens: 100, TopModel: "model-b", previousTokens: 100},
		},
		ModelsHistory: ModelHistorySeries{
			Models: []ModelHistoryModel{{Name: "model-a", Total: 100}, {Name: "model-b", Total: 50}},
			Points: []ModelHistoryPoint{
				{Ts: "2026-08-20T00:00:00Z", Label: "Aug 20", Model: "model-b", Tokens: 20},
				{Ts: "2026-08-21T00:00:00Z", Label: "Aug 21", Model: "model-a", Tokens: 100},
				{Ts: "2026-08-21T00:00:00Z", Label: "Aug 21", Model: "model-b", Tokens: 30},
			},
		},
		VendorShareHistory: VendorShareSeries{
			Vendors: []VendorShareVendor{{Name: "Vendor A", Total: 150}, {Name: "Vendor B", Total: 100}},
			Points: []VendorSharePoint{
				{Ts: "2026-08-21T00:00:00Z", Label: "Aug 21", Vendor: "Vendor A", Tokens: 150, Share: 0.6},
				{Ts: "2026-08-21T00:00:00Z", Label: "Aug 21", Vendor: "Vendor B", Tokens: 100, Share: 0.4},
			},
		},
	}
	config := operation_setting.RankingDisplayConfig{
		Version: operation_setting.RankingDisplayConfigVersion,
		Enabled: true,
		Periods: map[string]operation_setting.RankingDisplayPeriod{
			"week": {Adjustments: map[string]int64{"model-b": 150}},
		},
	}

	result, err := applyRankingDisplayConfig(live, config, "week")

	require.NoError(t, err)
	assert.Equal(t, RankingDataModeAdjusted, result.DataMode)
	require.Len(t, result.Models, 2)
	assert.Equal(t, "model-b", result.Models[0].ModelName)
	assert.Equal(t, int64(200), result.Models[0].TotalTokens)
	assert.Equal(t, 1, result.Models[0].Rank)
	assert.InDelta(t, float64(200)/400, result.Models[0].Share, 0.0001)
	assert.Equal(t, float64(100), result.Models[0].GrowthPct)
	assert.Equal(t, int64(250), result.Vendors[0].TotalTokens)
	assert.Equal(t, "Vendor B", result.Vendors[0].Vendor)
	assert.Equal(t, float64(150), result.Vendors[0].GrowthPct)
	assert.Equal(t, int64(200), result.ModelsHistory.Models[1].Total)
	assert.Equal(t, int64(250), result.VendorShareHistory.Vendors[1].Total)
	modelBHistoryTotal := int64(0)
	for _, point := range result.ModelsHistory.Points {
		if point.Model == "model-b" {
			modelBHistoryTotal += point.Tokens
		}
	}
	assert.Equal(t, int64(200), modelBHistoryTotal)
	vendorBHistoryTotal := int64(0)
	for _, point := range result.VendorShareHistory.Points {
		if point.Vendor == "Vendor B" {
			vendorBHistoryTotal += point.Tokens
		}
	}
	assert.Equal(t, int64(250), vendorBHistoryTotal)
	assert.Equal(t, int64(50), live.Models[1].TotalTokens)
	assert.Equal(t, int64(30), live.ModelsHistory.Points[2].Tokens)
	assert.Equal(t, int64(100), live.VendorShareHistory.Points[1].Tokens)
	assert.Equal(t, RankingDataModeLive, live.DataMode)
}

func TestApplyRankingDisplayConfigWithNoAddedTokensKeepsLiveTotals(t *testing.T) {
	live := &RankingsResponse{
		DataMode: RankingDataModeLive,
		Models:   []RankedModel{{Rank: 1, ModelName: "model-a", Vendor: "Vendor A", TotalTokens: 100}},
		Vendors:  []RankedVendor{{Rank: 1, Vendor: "Vendor A", TotalTokens: 100}},
	}
	config := operation_setting.RankingDisplayConfig{
		Version: operation_setting.RankingDisplayConfigVersion,
		Enabled: true,
		Periods: map[string]operation_setting.RankingDisplayPeriod{
			"week": {Adjustments: map[string]int64{"model-a": 0}},
		},
	}

	result, err := applyRankingDisplayConfig(live, config, "week")

	require.NoError(t, err)
	assert.Same(t, live, result)
	assert.Equal(t, int64(100), result.Models[0].TotalTokens)
	assert.Equal(t, RankingDataModeLive, result.DataMode)
}

func TestApplyRankingDisplayConfigAggregatesDailyRecordsIntoHistoryBuckets(t *testing.T) {
	previousRank := 1
	weekConfig, err := rankingConfig("week")
	require.NoError(t, err)
	currentStart, _, err := rankingDateRange("2026-08-19")
	require.NoError(t, err)
	currentEnd := currentStart + 4*24*60*60 - 1
	live := &RankingsResponse{
		DataMode: RankingDataModeLive,
		Models: []RankedModel{
			{Rank: 1, PreviousRank: &previousRank, ModelName: "model-a", Vendor: "Vendor A", TotalTokens: 100, previousTokens: 80},
			{Rank: 2, ModelName: "model-b", Vendor: "Vendor B", TotalTokens: 50, previousTokens: 100},
		},
		Vendors: []RankedVendor{
			{Rank: 1, Vendor: "Vendor A", TotalTokens: 100, TopModel: "model-a", previousTokens: 80},
			{Rank: 2, Vendor: "Vendor B", TotalTokens: 50, TopModel: "model-b", previousTokens: 100},
		},
		ModelsHistory: ModelHistorySeries{
			Models: []ModelHistoryModel{{Name: "model-a", Vendor: "Vendor A", Total: 100}, {Name: "model-b", Vendor: "Vendor B", Total: 50}},
			Points: []ModelHistoryPoint{
				{Ts: "2026-08-20T00:00:00Z", Label: "Aug 20", Model: "model-a", Vendor: "Vendor A", Tokens: 40},
				{Ts: "2026-08-21T00:00:00Z", Label: "Aug 21", Model: "model-a", Vendor: "Vendor A", Tokens: 60},
			},
		},
		VendorShareHistory: VendorShareSeries{
			Vendors: []VendorShareVendor{{Name: "Vendor A", Total: 100}, {Name: "Vendor B", Total: 50}},
			Points: []VendorSharePoint{
				{Ts: "2026-08-20T00:00:00Z", Label: "Aug 20", Vendor: "Vendor A", Tokens: 40, Share: 1},
				{Ts: "2026-08-21T00:00:00Z", Label: "Aug 21", Vendor: "Vendor A", Tokens: 60, Share: 1},
			},
		},
		rankingMeta: &rankingSnapshotMeta{
			config:                weekConfig,
			currentStart:          currentStart,
			currentEnd:            currentEnd,
			previousStart:         currentStart - 7*24*60*60,
			previousEnd:           currentStart - 1,
			previousTokensByModel: map[string]int64{"model-a": 80, "model-b": 100},
		},
	}
	config := operation_setting.RankingDisplayConfig{
		Version: operation_setting.RankingDisplayConfigVersion,
		Enabled: true,
		DailyRecords: map[string]operation_setting.RankingDisplayDay{
			"2026-08-18": {Adjustments: map[string]int64{"model-a": 20}},
			"2026-08-20": {Adjustments: map[string]int64{"model-a": 30}},
			"2026-08-21": {Adjustments: map[string]int64{"model-a": 70}},
		},
	}

	result, err := applyRankingDisplayConfig(live, config, "week")

	require.NoError(t, err)
	assert.Equal(t, int64(200), result.Models[0].TotalTokens)
	assert.Equal(t, float64(100), result.Models[0].GrowthPct)
	assert.Equal(t, int64(70), result.ModelsHistory.Points[0].Tokens)
	assert.Equal(t, int64(130), result.ModelsHistory.Points[1].Tokens)
	assert.Equal(t, int64(70), result.VendorShareHistory.Points[0].Tokens)
	assert.Equal(t, int64(130), result.VendorShareHistory.Points[1].Tokens)
	assert.Equal(t, 2, result.ModelsHistory.Buckets)
	assert.Equal(t, 2, result.VendorShareHistory.Buckets)
}

func TestApplyRankingDisplayConfigDistributesTodayAdjustmentsAcrossTimeline(t *testing.T) {
	todayConfig, err := rankingConfig("today")
	require.NoError(t, err)
	currentStart, _, err := rankingDateRange("2026-08-22")
	require.NoError(t, err)
	currentEnd := currentStart + 4*60*60 - 1
	live := &RankingsResponse{
		DataMode: RankingDataModeLive,
		Models:   []RankedModel{{Rank: 1, ModelName: "model-a", Vendor: "Vendor A", TotalTokens: 1}},
		Vendors:  []RankedVendor{{Rank: 1, Vendor: "Vendor A", TotalTokens: 1, TopModel: "model-a"}},
		ModelsHistory: ModelHistorySeries{
			Models: []ModelHistoryModel{{Name: "model-a", Vendor: "Vendor A", Total: 1}},
			Points: []ModelHistoryPoint{{
				Ts: "2026-08-22T01:00:00Z", Label: "01:00", Model: "model-a", Vendor: "Vendor A", Tokens: 1,
			}},
		},
		VendorShareHistory: VendorShareSeries{
			Vendors: []VendorShareVendor{{Name: "Vendor A", Total: 1, Share: 1}},
			Points: []VendorSharePoint{{
				Ts: "2026-08-22T01:00:00Z", Label: "01:00", Vendor: "Vendor A", Tokens: 1, Share: 1,
			}},
		},
		rankingMeta: &rankingSnapshotMeta{
			config:       todayConfig,
			currentStart: currentStart,
			currentEnd:   currentEnd,
		},
	}
	config := operation_setting.RankingDisplayConfig{
		Version: operation_setting.RankingDisplayConfigVersion,
		Enabled: true,
		Periods: map[string]operation_setting.RankingDisplayPeriod{
			"today": {Adjustments: map[string]int64{"model-a": 100}},
		},
		DailyRecords: map[string]operation_setting.RankingDisplayDay{
			"2026-08-22": {Adjustments: map[string]int64{"model-a": 20}},
		},
	}

	result, err := applyRankingDisplayConfig(live, config, "today")

	require.NoError(t, err)
	assert.Equal(t, int64(121), result.Models[0].TotalTokens)
	assert.Equal(t, int64(121), result.ModelsHistory.Models[0].Total)
	assert.Equal(t, 4, result.ModelsHistory.Buckets)
	assert.Equal(t, 4, result.VendorShareHistory.Buckets)
	modelPoints := make(map[string]int64)
	for _, point := range result.ModelsHistory.Points {
		if point.Model == "model-a" {
			modelPoints[point.Ts] += point.Tokens
		}
	}
	assert.Len(t, modelPoints, 4)
	modelHistoryTotal := int64(0)
	for _, tokens := range modelPoints {
		modelHistoryTotal += tokens
	}
	assert.Equal(t, int64(121), modelHistoryTotal)
	vendorHistoryTotal := int64(0)
	for _, point := range result.VendorShareHistory.Points {
		vendorHistoryTotal += point.Tokens
	}
	assert.Equal(t, int64(121), vendorHistoryTotal)
	assert.Equal(t, time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC).Format(time.RFC3339), result.ModelsHistory.Points[0].Ts)
}

func TestApplyRankingDisplayConfigRejectsTotalsAboveSafeIntegerRange(t *testing.T) {
	live := &RankingsResponse{
		DataMode: RankingDataModeLive,
		Models: []RankedModel{{
			Rank: 1, ModelName: "model-a", Vendor: "Vendor A",
			TotalTokens: operation_setting.MaxRankingDisplayAddedTokens,
		}},
		Vendors: []RankedVendor{{
			Rank: 1, Vendor: "Vendor A",
			TotalTokens: operation_setting.MaxRankingDisplayAddedTokens,
		}},
	}
	config := operation_setting.RankingDisplayConfig{
		Version: operation_setting.RankingDisplayConfigVersion,
		Enabled: true,
		Periods: map[string]operation_setting.RankingDisplayPeriod{
			"week": {Adjustments: map[string]int64{"model-a": 1}},
		},
	}

	result, err := applyRankingDisplayConfig(live, config, "week")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, operation_setting.MaxRankingDisplayAddedTokens, live.Models[0].TotalTokens)
}

func TestGetRankingsSnapshotForDateRejectsInvalidDate(t *testing.T) {
	_, err := GetRankingsSnapshotForDate("today", "2026-02-30")

	assert.Error(t, err)
}
