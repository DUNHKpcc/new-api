package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

const DesktopAgentAutoGroup = "auto"

type DesktopAgentSetting struct {
	ClaudeGroup string `json:"claude_group"`
	CodexGroup  string `json:"codex_group"`
	GiftPlanId  int    `json:"gift_plan_id"`
}

var desktopAgentSetting = DesktopAgentSetting{
	ClaudeGroup: DesktopAgentAutoGroup,
	CodexGroup:  DesktopAgentAutoGroup,
	GiftPlanId:  0,
}

func init() {
	config.GlobalConfig.Register("desktop_agent_setting", &desktopAgentSetting)
}

func GetDesktopAgentSetting() *DesktopAgentSetting {
	return &desktopAgentSetting
}
