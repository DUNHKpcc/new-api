package operation_setting

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

const (
	RankingDisplayOptionKey      = "RankingDisplayConfig"
	RankingDisplayConfigVersion  = 1
	MaxRankingDisplayModels      = 100
	MaxRankingDisplayAddedTokens = int64(9_007_199_254_740_991)
	MaxRankingDisplayConfigBytes = 256 * 1024
)

type RankingDisplayConfig struct {
	Version int                             `json:"version"`
	Enabled bool                            `json:"enabled"`
	Periods map[string]RankingDisplayPeriod `json:"periods"`
}

type RankingDisplayPeriod struct {
	Adjustments map[string]int64 `json:"adjustments"`
}

func DefaultRankingDisplayConfig() RankingDisplayConfig {
	return RankingDisplayConfig{
		Version: RankingDisplayConfigVersion,
		Periods: make(map[string]RankingDisplayPeriod),
	}
}

func ParseRankingDisplayConfig(value string) (RankingDisplayConfig, error) {
	if len(value) > MaxRankingDisplayConfigBytes {
		return RankingDisplayConfig{}, fmt.Errorf("ranking display configuration is too large")
	}
	if strings.TrimSpace(value) == "" {
		return DefaultRankingDisplayConfig(), nil
	}

	var config RankingDisplayConfig
	if err := common.UnmarshalJsonStr(value, &config); err != nil {
		return RankingDisplayConfig{}, fmt.Errorf("invalid ranking display configuration: %w", err)
	}
	if err := ValidateRankingDisplayConfig(config); err != nil {
		return RankingDisplayConfig{}, err
	}
	if config.Periods == nil {
		config.Periods = make(map[string]RankingDisplayPeriod)
	}
	return config, nil
}

func ValidateRankingDisplayConfig(config RankingDisplayConfig) error {
	if config.Version != RankingDisplayConfigVersion {
		return fmt.Errorf("ranking display configuration version must be %d", RankingDisplayConfigVersion)
	}
	if len(config.Periods) > 4 {
		return fmt.Errorf("ranking display configuration has too many periods")
	}

	for period, periodConfig := range config.Periods {
		switch period {
		case "today", "week", "month", "year":
		default:
			return fmt.Errorf("invalid ranking display period: %s", period)
		}
		if len(periodConfig.Adjustments) > MaxRankingDisplayModels {
			return fmt.Errorf("ranking display period %s has too many models", period)
		}
		for modelName, addedTokens := range periodConfig.Adjustments {
			trimmedName := strings.TrimSpace(modelName)
			if trimmedName == "" || trimmedName != modelName || len(modelName) > 64 {
				return fmt.Errorf("invalid ranking display model name")
			}
			if addedTokens < 0 || addedTokens > MaxRankingDisplayAddedTokens {
				return fmt.Errorf("invalid added token value for model %s", modelName)
			}
		}
	}
	return nil
}
