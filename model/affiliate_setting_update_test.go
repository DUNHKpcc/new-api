package model

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func prepareAffiliateSettingUpdateTest(t *testing.T) (User, operation_setting.AffiliateSetting) {
	t.Helper()
	truncateTables(t)
	previous := operation_setting.GetAffiliateSettingSnapshot()
	t.Cleanup(func() { require.NoError(t, operation_setting.SetAffiliateSetting(previous)) })
	current := operation_setting.DefaultAffiliateSetting()
	require.NoError(t, operation_setting.SetAffiliateSetting(current))
	encoded, err := operation_setting.MarshalAffiliateSetting(current)
	require.NoError(t, err)
	require.NoError(t, DB.Create(&Option{Key: operation_setting.AffiliateSettingOptionKey, Value: encoded}).Error)
	setAffiliatePaymentComplianceOptions(t, true)
	operator := User{Username: "affiliate-setting-root", AffCode: "affiliate-setting-root"}
	require.NoError(t, DB.Create(&operator).Error)
	return operator, current
}

func TestUpdateAffiliateSettingOptionUsesPersistedPaymentCompliance(t *testing.T) {
	operator, current := prepareAffiliateSettingUpdateTest(t)
	setAffiliatePaymentComplianceOptions(t, false)
	previousPaymentSetting := *operation_setting.GetPaymentSetting()
	operation_setting.GetPaymentSetting().ComplianceConfirmed = true
	operation_setting.GetPaymentSetting().ComplianceTermsVersion = operation_setting.CurrentComplianceTermsVersion
	t.Cleanup(func() { *operation_setting.GetPaymentSetting() = previousPaymentSetting })

	next := current
	next.CommissionEnabled = true
	next.CommissionRateBPS = 2500
	_, err := UpdateAffiliateSettingOption(current.Version, next, operator.Id)
	assert.ErrorIs(t, err, ErrPaymentComplianceRequired)

	var option Option
	require.NoError(t, DB.Where(commonKeyCol+" = ?", operation_setting.AffiliateSettingOptionKey).First(&option).Error)
	stored, err := operation_setting.ParseAffiliateSettingJSON(option.Value)
	require.NoError(t, err)
	assert.Equal(t, current, stored)
}

func TestUpdateAffiliateSettingOptionUsesVersionCASAndAtomicAudit(t *testing.T) {
	operator, current := prepareAffiliateSettingUpdateTest(t)
	next := current
	next.CommissionEnabled = true
	next.CommissionRateBPS = 2500

	result, err := UpdateAffiliateSettingOption(current.Version, next, operator.Id)
	require.NoError(t, err)
	assert.Equal(t, current.Version, result.Previous.Version)
	assert.Equal(t, current.Version+1, result.Current.Version)
	assert.Equal(t, int64(2500), operation_setting.GetAffiliateSettingSnapshot().CommissionRateBPS)

	var change AffiliateConfigChange
	require.NoError(t, DB.First(&change).Error)
	assert.Equal(t, current.Version, change.VersionBefore)
	assert.Equal(t, current.Version+1, change.VersionAfter)
	var event AffiliateOutboxEvent
	require.NoError(t, DB.Where("action = ?", "affiliate.config_changed").First(&event).Error)

	stale := next
	stale.CommissionRateBPS = 5000
	_, err = UpdateAffiliateSettingOption(current.Version, stale, operator.Id)
	assert.ErrorIs(t, err, ErrOptionVersionConflict)
	assert.Equal(t, int64(2500), operation_setting.GetAffiliateSettingSnapshot().CommissionRateBPS)
	var changeCount int64
	require.NoError(t, DB.Model(&AffiliateConfigChange{}).Count(&changeCount).Error)
	assert.Equal(t, int64(1), changeCount)
}

func TestUpdateAffiliateSettingOptionRollsBackWhenOutboxConflicts(t *testing.T) {
	operator, current := prepareAffiliateSettingUpdateTest(t)
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		_, err := EnqueueAffiliateEventWithTx(
			tx,
			"affiliate.config_changed:2",
			operator.Id,
			"affiliate.config_changed",
			LogTypeManage,
			"Conflicting event",
			map[string]interface{}{"version_after": int64(2)},
			100,
		)
		return err
	}))
	next := current
	next.CommissionEnabled = true
	next.CommissionRateBPS = 1000

	_, err := UpdateAffiliateSettingOption(current.Version, next, operator.Id)
	assert.ErrorIs(t, err, ErrAffiliateOutboxConflict)
	var option Option
	require.NoError(t, DB.Where(commonKeyCol+" = ?", operation_setting.AffiliateSettingOptionKey).First(&option).Error)
	stored, err := operation_setting.ParseAffiliateSettingJSON(option.Value)
	require.NoError(t, err)
	assert.Equal(t, current, stored)
	assert.Equal(t, current, operation_setting.GetAffiliateSettingSnapshot())
	var changeCount int64
	require.NoError(t, DB.Model(&AffiliateConfigChange{}).Count(&changeCount).Error)
	assert.Zero(t, changeCount)
}
