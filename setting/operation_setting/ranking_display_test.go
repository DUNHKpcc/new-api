package operation_setting

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRankingDisplayConfigAcceptsNonNegativeModelAdjustments(t *testing.T) {
	config, err := ParseRankingDisplayConfig(`{
		"version": 1,
		"enabled": true,
		"periods": {
			"week": {"adjustments": {"gpt-5": 12000000, "claude": 0}}
		}
	}`)

	require.NoError(t, err)
	assert.True(t, config.Enabled)
	assert.Equal(t, int64(12000000), config.Periods["week"].Adjustments["gpt-5"])
	assert.Equal(t, int64(0), config.Periods["week"].Adjustments["claude"])
}

func TestParseRankingDisplayConfigRejectsInvalidPeriodAndNegativeAdjustments(t *testing.T) {
	testCases := []string{
		`{"version":1,"enabled":true,"periods":{"quarter":{"adjustments":{"gpt-5":1}}}}`,
		`{"version":1,"enabled":true,"periods":{"week":{"adjustments":{"gpt-5":-1}}}}`,
		`{"version":2,"enabled":true,"periods":{}}`,
	}

	for _, value := range testCases {
		_, err := ParseRankingDisplayConfig(value)
		assert.Error(t, err)
	}
}

func TestParseRankingDisplayConfigRejectsOversizedPayload(t *testing.T) {
	_, err := ParseRankingDisplayConfig(strings.Repeat(" ", MaxRankingDisplayConfigBytes+1))

	assert.Error(t, err)
}
