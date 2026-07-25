package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

const DesktopAgentAutoGroup = "auto"

type DesktopAgentSetting struct {
	ClaudeGroup string `json:"claude_group"`
	CodexGroup  string `json:"codex_group"`
}

var desktopAgentSetting = DesktopAgentSetting{
	ClaudeGroup: DesktopAgentAutoGroup,
	CodexGroup:  DesktopAgentAutoGroup,
}

func init() {
	config.GlobalConfig.Register("desktop_agent_setting", &desktopAgentSetting)
}

func GetDesktopAgentSetting() *DesktopAgentSetting {
	return &desktopAgentSetting
}
