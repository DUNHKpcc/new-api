package service

import (
	"testing"

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
	assert.Equal(t, int64(180), result.ModelsHistory.Points[2].Tokens)
	assert.Equal(t, int64(250), result.VendorShareHistory.Vendors[1].Total)
	assert.Equal(t, int64(250), result.VendorShareHistory.Points[1].Tokens)
	assert.InDelta(t, 0.625, result.VendorShareHistory.Points[1].Share, 0.0001)
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
