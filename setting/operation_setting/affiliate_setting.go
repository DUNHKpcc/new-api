package operation_setting

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/QuantumNous/new-api/common"
)

const (
	AffiliateSettingOptionKey            = "AffiliateSetting"
	MaxAffiliateCommissionRateBPS  int64 = 10_000
	MaxAffiliateCommissionWaitDays int64 = 3_650
)

// AffiliateSetting controls the independent signup-reward and top-up-commission programs.
// Amounts are integer quota or minor-currency units; float values are intentionally unsupported.
type AffiliateSetting struct {
	RegistrationRewardEnabled   bool  `json:"registration_reward_enabled"`
	InviterRewardQuota          int64 `json:"inviter_reward_quota"`
	InviteeRewardQuota          int64 `json:"invitee_reward_quota"`
	CommissionEnabled           bool  `json:"commission_enabled"`
	QualificationThresholdMinor int64 `json:"qualification_threshold_minor"`
	CommissionRateBPS           int64 `json:"commission_rate_bps"`
	CommissionWaitDays          int64 `json:"commission_wait_days"`
	Version                     int64 `json:"version"`
}

var affiliateSettingSnapshot atomic.Pointer[AffiliateSetting]

func init() {
	setting := DefaultAffiliateSetting()
	affiliateSettingSnapshot.Store(&setting)
}

func DefaultAffiliateSetting() AffiliateSetting {
	return AffiliateSetting{
		RegistrationRewardEnabled:   true,
		InviterRewardQuota:          0,
		InviteeRewardQuota:          0,
		CommissionEnabled:           false,
		QualificationThresholdMinor: 10_000,
		CommissionRateBPS:           0,
		CommissionWaitDays:          7,
		Version:                     1,
	}
}

// NormalizeAffiliateSetting returns canonical storage values after validating the complete setting.
func NormalizeAffiliateSetting(setting AffiliateSetting) (AffiliateSetting, error) {
	if err := ValidateAffiliateSetting(setting); err != nil {
		return AffiliateSetting{}, err
	}
	return setting, nil
}

// MarshalAffiliateSetting serializes one complete, canonical configuration snapshot.
func MarshalAffiliateSetting(setting AffiliateSetting) (string, error) {
	normalized, err := NormalizeAffiliateSetting(setting)
	if err != nil {
		return "", err
	}
	encoded, err := common.Marshal(normalized)
	if err != nil {
		return "", fmt.Errorf("序列化代理配置失败: %w", err)
	}
	return string(encoded), nil
}

// ParseAffiliateSettingJSON rejects partial, unknown, or invalid configuration snapshots.
func ParseAffiliateSettingJSON(value string) (AffiliateSetting, error) {
	rawValue := json.RawMessage(strings.TrimSpace(value))
	if common.GetJsonType(rawValue) != "object" {
		return AffiliateSetting{}, errors.New("代理配置必须是 JSON 对象")
	}

	var fields map[string]json.RawMessage
	if err := common.Unmarshal(rawValue, &fields); err != nil {
		return AffiliateSetting{}, fmt.Errorf("解析代理配置失败: %w", err)
	}
	for _, field := range affiliateSettingJSONFields {
		if _, ok := fields[field]; !ok {
			return AffiliateSetting{}, fmt.Errorf("代理配置缺少字段 %q", field)
		}
	}
	for field := range fields {
		if _, ok := affiliateSettingJSONFieldSet[field]; !ok {
			return AffiliateSetting{}, fmt.Errorf("代理配置包含未知字段 %q", field)
		}
	}

	var setting AffiliateSetting
	if err := common.Unmarshal(rawValue, &setting); err != nil {
		return AffiliateSetting{}, fmt.Errorf("解析代理配置失败: %w", err)
	}
	return NormalizeAffiliateSetting(setting)
}

func ValidateAffiliateSettingJSON(value string) error {
	_, err := ParseAffiliateSettingJSON(value)
	return err
}

// LoadAffiliateSettingJSON validates the complete snapshot before publishing it.
func LoadAffiliateSettingJSON(value string) error {
	setting, err := ParseAffiliateSettingJSON(value)
	if err != nil {
		return err
	}
	return SetAffiliateSetting(setting)
}

// AffiliateSettingFromLegacyRewardQuotas creates the V1 default from the old reward options.
func AffiliateSettingFromLegacyRewardQuotas(inviterQuota, inviteeQuota int64) (AffiliateSetting, error) {
	setting := DefaultAffiliateSetting()
	setting.InviterRewardQuota = inviterQuota
	setting.InviteeRewardQuota = inviteeQuota
	return NormalizeAffiliateSetting(setting)
}

// ValidateAffiliateSetting verifies all fields because this configuration controls financial rewards.
func ValidateAffiliateSetting(setting AffiliateSetting) error {
	if setting.InviterRewardQuota < 0 {
		return errors.New("邀请人固定奖励额度不能为负数")
	}
	if setting.InviterRewardQuota > int64(common.MaxQuota) {
		return errors.New("邀请人固定奖励额度超出系统额度上限")
	}
	if setting.InviteeRewardQuota < 0 {
		return errors.New("被邀请人固定奖励额度不能为负数")
	}
	if setting.InviteeRewardQuota > int64(common.MaxQuota) {
		return errors.New("被邀请人固定奖励额度超出系统额度上限")
	}
	if setting.QualificationThresholdMinor < 0 {
		return errors.New("代理资格门槛不能为负数")
	}
	if setting.CommissionRateBPS < 0 || setting.CommissionRateBPS > MaxAffiliateCommissionRateBPS {
		return errors.New("代理佣金费率必须在 0 到 10000 个基点之间")
	}
	if setting.CommissionWaitDays < 0 || setting.CommissionWaitDays > MaxAffiliateCommissionWaitDays {
		return errors.New("代理佣金等待天数必须在 0 到 3650 天之间")
	}
	if setting.Version < 1 {
		return errors.New("代理配置版本必须大于或等于 1")
	}
	return nil
}

// GetAffiliateSettingSnapshot returns an immutable-by-copy view for concurrent readers.
func GetAffiliateSettingSnapshot() AffiliateSetting {
	setting := affiliateSettingSnapshot.Load()
	if setting == nil {
		return DefaultAffiliateSetting()
	}
	return *setting
}

// SetAffiliateSetting validates, normalizes, and atomically publishes a new runtime snapshot.
func SetAffiliateSetting(setting AffiliateSetting) error {
	normalized, err := NormalizeAffiliateSetting(setting)
	if err != nil {
		return err
	}

	affiliateSettingSnapshot.Store(&normalized)
	return nil
}

var affiliateSettingJSONFields = []string{
	"registration_reward_enabled",
	"inviter_reward_quota",
	"invitee_reward_quota",
	"commission_enabled",
	"qualification_threshold_minor",
	"commission_rate_bps",
	"commission_wait_days",
	"version",
}

var affiliateSettingJSONFieldSet = func() map[string]struct{} {
	fields := make(map[string]struct{}, len(affiliateSettingJSONFields))
	for _, field := range affiliateSettingJSONFields {
		fields[field] = struct{}{}
	}
	// Accept snapshots written by early V1 builds; currency is now derived from Epay.
	fields["qualification_currency"] = struct{}{}
	return fields
}()
