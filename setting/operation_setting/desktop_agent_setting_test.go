package operation_setting

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDesktopAgentSettingExposesIndependentAdminConfigKeys(t *testing.T) {
	previous := desktopAgentSetting
	t.Cleanup(func() {
		desktopAgentSetting = previous
	})

	require.NoError(t, config.UpdateConfigFromMap(&desktopAgentSetting, map[string]string{
		"claude_group": "claude-group",
		"codex_group":  "codex-group",
		"gift_plan_id": "42",
	}))

	exported := config.GlobalConfig.ExportAllConfigs()
	assert.Equal(t, "claude-group", exported["desktop_agent_setting.claude_group"])
	assert.Equal(t, "codex-group", exported["desktop_agent_setting.codex_group"])
	assert.Equal(t, "42", exported["desktop_agent_setting.gift_plan_id"])
}
