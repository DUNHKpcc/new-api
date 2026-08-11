package model

import (
	"errors"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestAffiliateSettingOptionPublishesOnlyValidatedCompleteSnapshots(t *testing.T) {
	originalSetting := operation_setting.GetAffiliateSettingSnapshot()
	common.OptionMapRWMutex.Lock()
	originalOptions := common.OptionMap
	common.OptionMap = make(map[string]string)
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		require.NoError(t, operation_setting.SetAffiliateSetting(originalSetting))
		common.OptionMapRWMutex.Lock()
		common.OptionMap = originalOptions
		common.OptionMapRWMutex.Unlock()
	})

	updated := originalSetting
	updated.CommissionEnabled = true
	updated.CommissionRateBPS = 5_000
	encoded, err := operation_setting.MarshalAffiliateSetting(updated)
	require.NoError(t, err)
	require.NoError(t, updateOptionMap(operation_setting.AffiliateSettingOptionKey, encoded))

	snapshot := operation_setting.GetAffiliateSettingSnapshot()
	assert.True(t, snapshot.CommissionEnabled)
	assert.Equal(t, int64(5_000), snapshot.CommissionRateBPS)

	require.Error(t, updateOptionMap(operation_setting.AffiliateSettingOptionKey, `{"version":1}`))
	assert.Equal(t, snapshot, operation_setting.GetAffiliateSettingSnapshot())
}

func TestValidateOptionValueRejectsInvalidAffiliateSettingJSON(t *testing.T) {
	err := validateOptionValue(operation_setting.AffiliateSettingOptionKey, `{"version":1}`)
	assert.Error(t, err)
}

func TestLoadOptionsDoesNotReplaceAffiliateSnapshotWhenQueryFails(t *testing.T) {
	original := operation_setting.GetAffiliateSettingSnapshot()
	protected := original
	protected.CommissionEnabled = true
	protected.CommissionRateBPS = 4_321
	require.NoError(t, operation_setting.SetAffiliateSetting(protected))
	t.Cleanup(func() { require.NoError(t, operation_setting.SetAffiliateSetting(original)) })

	callbackName := "test:fail_affiliate_option_query"
	require.NoError(t, DB.Callback().Query().Before("gorm:query").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement.Table == "options" {
			tx.AddError(errors.New("injected option query failure"))
		}
	}))
	t.Cleanup(func() { require.NoError(t, DB.Callback().Query().Remove(callbackName)) })

	require.Error(t, loadOptionsFromDatabase())
	assert.Equal(t, protected, operation_setting.GetAffiliateSettingSnapshot())
}
