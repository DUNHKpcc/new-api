package operation_setting

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultAffiliateSetting(t *testing.T) {
	setting := DefaultAffiliateSetting()

	assert.True(t, setting.RegistrationRewardEnabled)
	assert.Zero(t, setting.InviterRewardQuota)
	assert.Zero(t, setting.InviteeRewardQuota)
	assert.False(t, setting.CommissionEnabled)
	assert.Equal(t, int64(10_000), setting.QualificationThresholdMinor)
	assert.Zero(t, setting.CommissionRateBPS)
	assert.Equal(t, int64(7), setting.CommissionWaitDays)
	assert.Equal(t, int64(1), setting.Version)
}

func TestSetAffiliateSettingNormalizesAndSnapshots(t *testing.T) {
	original := GetAffiliateSettingSnapshot()
	t.Cleanup(func() {
		require.NoError(t, SetAffiliateSetting(original))
	})

	updated := original
	updated.CommissionRateBPS = 5_000
	require.NoError(t, SetAffiliateSetting(updated))

	snapshot := GetAffiliateSettingSnapshot()
	assert.Equal(t, int64(5_000), snapshot.CommissionRateBPS)
	snapshot.CommissionRateBPS = 1
	assert.Equal(t, int64(5_000), GetAffiliateSettingSnapshot().CommissionRateBPS)
}

func TestAffiliateSettingJSONRoundTripUsesCanonicalSnapshot(t *testing.T) {
	setting := DefaultAffiliateSetting()
	setting.CommissionRateBPS = 5_000

	encoded, err := MarshalAffiliateSetting(setting)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"registration_reward_enabled": true,
		"inviter_reward_quota": 0,
		"invitee_reward_quota": 0,
		"commission_enabled": false,
		"qualification_threshold_minor": 10000,
		"commission_rate_bps": 5000,
		"commission_wait_days": 7,
		"version": 1
	}`, encoded)

	decoded, err := ParseAffiliateSettingJSON(encoded)
	require.NoError(t, err)
	assert.Equal(t, int64(5_000), decoded.CommissionRateBPS)
}

func TestParseAffiliateSettingJSONAcceptsLegacyCurrencyField(t *testing.T) {
	encoded, err := MarshalAffiliateSetting(DefaultAffiliateSetting())
	require.NoError(t, err)
	legacy := strings.TrimSuffix(encoded, "}") + `,"qualification_currency":"CNY"}`

	setting, err := ParseAffiliateSettingJSON(legacy)
	require.NoError(t, err)
	canonical, err := MarshalAffiliateSetting(setting)
	require.NoError(t, err)
	assert.NotContains(t, canonical, "qualification_currency")
}

func TestParseAffiliateSettingJSONRejectsPartialOrInvalidSnapshots(t *testing.T) {
	valid, err := MarshalAffiliateSetting(DefaultAffiliateSetting())
	require.NoError(t, err)

	testCases := []struct {
		name  string
		value string
	}{
		{
			name:  "not an object",
			value: `[]`,
		},
		{
			name:  "missing version",
			value: `{"registration_reward_enabled":true}`,
		},
		{
			name:  "unknown field",
			value: valid[:len(valid)-1] + `,"unexpected":true}`,
		},
		{
			name: "fractional financial value",
			value: `{
				"registration_reward_enabled":true,
				"inviter_reward_quota":0,
				"invitee_reward_quota":0,
				"commission_enabled":false,
				"qualification_threshold_minor":10000,
				"qualification_currency":"CNY",
				"commission_rate_bps":0.5,
				"commission_wait_days":7,
				"version":1
			}`,
		},
		{
			name: "invalid version",
			value: `{
				"registration_reward_enabled":true,
				"inviter_reward_quota":0,
				"invitee_reward_quota":0,
				"commission_enabled":false,
				"qualification_threshold_minor":10000,
				"qualification_currency":"CNY",
				"commission_rate_bps":0,
				"commission_wait_days":7,
				"version":0
			}`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Error(t, ValidateAffiliateSettingJSON(testCase.value))
		})
	}
}

func TestAffiliateSettingFromLegacyRewardQuotas(t *testing.T) {
	setting, err := AffiliateSettingFromLegacyRewardQuotas(123, 456)
	require.NoError(t, err)
	assert.True(t, setting.RegistrationRewardEnabled)
	assert.Equal(t, int64(123), setting.InviterRewardQuota)
	assert.Equal(t, int64(456), setting.InviteeRewardQuota)
	assert.Equal(t, int64(1), setting.Version)

	_, err = AffiliateSettingFromLegacyRewardQuotas(-1, 0)
	assert.Error(t, err)
}

func TestValidateAffiliateSettingRejectsInvalidValues(t *testing.T) {
	valid := DefaultAffiliateSetting()
	testCases := []struct {
		name   string
		mutate func(*AffiliateSetting)
	}{
		{
			name: "negative inviter reward quota",
			mutate: func(setting *AffiliateSetting) {
				setting.InviterRewardQuota = -1
			},
		},
		{
			name: "inviter reward quota above maximum",
			mutate: func(setting *AffiliateSetting) {
				setting.InviterRewardQuota = int64(common.MaxQuota) + 1
			},
		},
		{
			name: "negative invitee reward quota",
			mutate: func(setting *AffiliateSetting) {
				setting.InviteeRewardQuota = -1
			},
		},
		{
			name: "invitee reward quota above maximum",
			mutate: func(setting *AffiliateSetting) {
				setting.InviteeRewardQuota = int64(common.MaxQuota) + 1
			},
		},
		{
			name: "negative qualification threshold",
			mutate: func(setting *AffiliateSetting) {
				setting.QualificationThresholdMinor = -1
			},
		},
		{
			name: "negative commission rate",
			mutate: func(setting *AffiliateSetting) {
				setting.CommissionRateBPS = -1
			},
		},
		{
			name: "commission rate above maximum",
			mutate: func(setting *AffiliateSetting) {
				setting.CommissionRateBPS = 10_001
			},
		},
		{
			name: "negative commission wait days",
			mutate: func(setting *AffiliateSetting) {
				setting.CommissionWaitDays = -1
			},
		},
		{
			name: "commission wait days above maximum",
			mutate: func(setting *AffiliateSetting) {
				setting.CommissionWaitDays = MaxAffiliateCommissionWaitDays + 1
			},
		},
		{
			name: "invalid version",
			mutate: func(setting *AffiliateSetting) {
				setting.Version = 0
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			setting := valid
			testCase.mutate(&setting)
			assert.Error(t, ValidateAffiliateSetting(setting))
		})
	}
}
