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

func TestUserInsertUsesPersistedPaymentComplianceWhenRuntimeSnapshotIsStale(t *testing.T) {
	db := setupAffiliateReferralTestDB(t)
	setAffiliateSignupRewardTestSetting(t, true, 17, 9, 12)

	// Simulate another instance confirming compliance in the database while this
	// process still has the old runtime snapshot.
	operation_setting.GetPaymentSetting().ComplianceConfirmed = false
	operation_setting.GetPaymentSetting().ComplianceTermsVersion = ""

	inviter := createAffiliateReferralTestUser(t, db, "db-compliance-inviter", "db-compliance", common.UserStatusEnabled)
	referred := &User{
		Username:    "db-compliance-referred",
		Password:    "password123",
		DisplayName: "db-compliance-referred",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
	}

	require.NoError(t, referred.Insert(inviter.Id))
	var storedInviter User
	require.NoError(t, db.First(&storedInviter, inviter.Id).Error)
	assert.Equal(t, 17, storedInviter.AffQuota)
	var storedReferred User
	require.NoError(t, db.First(&storedReferred, referred.Id).Error)
	assert.Equal(t, 9, storedReferred.Quota)
}

func TestUserInsertFailsClosedWhenPersistedPaymentComplianceIsNotCurrent(t *testing.T) {
	db := setupAffiliateReferralTestDB(t)
	setAffiliateSignupRewardTestSetting(t, true, 17, 9, 12)
	setAffiliatePaymentComplianceOptions(t, false)

	// A stale true runtime snapshot must not authorize a one-time reward.
	operation_setting.GetPaymentSetting().ComplianceConfirmed = true
	operation_setting.GetPaymentSetting().ComplianceTermsVersion = operation_setting.CurrentComplianceTermsVersion

	inviter := createAffiliateReferralTestUser(t, db, "stale-compliance-inviter", "stale-compliance", common.UserStatusEnabled)
	referred := &User{
		Username:    "stale-compliance-referred",
		Password:    "password123",
		DisplayName: "stale-compliance-referred",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
	}

	require.NoError(t, referred.Insert(inviter.Id))
	var storedInviter User
	require.NoError(t, db.First(&storedInviter, inviter.Id).Error)
	assert.Zero(t, storedInviter.AffQuota)
	var rewardCount int64
	require.NoError(t, db.Model(&AffiliateSignupReward{}).Count(&rewardCount).Error)
	assert.Zero(t, rewardCount)
}

func TestUpdateOptionsBulkRollsBackComplianceBatch(t *testing.T) {
	db := setupAffiliateReferralTestDB(t)
	previousPaymentSetting := *operation_setting.GetPaymentSetting()
	operation_setting.GetPaymentSetting().ComplianceConfirmed = false
	operation_setting.GetPaymentSetting().ComplianceTermsVersion = ""
	t.Cleanup(func() { *operation_setting.GetPaymentSetting() = previousPaymentSetting })

	require.NoError(t, UpdateOptionsBulk(map[string]string{
		PaymentComplianceConfirmedOptionKey:    "false",
		PaymentComplianceTermsVersionOptionKey: "",
	}))
	injectedErr := errors.New("injected compliance option failure")
	callbackName := "test:fail_compliance_terms_update"
	require.NoError(t, db.Callback().Update().Before("gorm:update").Register(callbackName, func(tx *gorm.DB) {
		option, ok := tx.Statement.Dest.(*Option)
		if ok && option.Key == PaymentComplianceTermsVersionOptionKey {
			tx.AddError(injectedErr)
		}
	}))
	t.Cleanup(func() { require.NoError(t, db.Callback().Update().Remove(callbackName)) })

	err := UpdateOptionsBulk(map[string]string{
		PaymentComplianceConfirmedOptionKey:    "true",
		PaymentComplianceTermsVersionOptionKey: operation_setting.CurrentComplianceTermsVersion,
	})
	require.ErrorIs(t, err, injectedErr)

	var options []Option
	require.NoError(t, db.Where(commonKeyCol+" IN ?", []string{
		PaymentComplianceConfirmedOptionKey,
		PaymentComplianceTermsVersionOptionKey,
	}).Find(&options).Error)
	values := make(map[string]string, len(options))
	for _, option := range options {
		values[option.Key] = option.Value
	}
	assert.Equal(t, "false", values[PaymentComplianceConfirmedOptionKey])
	assert.Empty(t, values[PaymentComplianceTermsVersionOptionKey])
	assert.False(t, operation_setting.IsPaymentComplianceConfirmed())
}

func TestUpdateEpayCurrencyUsesPersistedAffiliateSetting(t *testing.T) {
	db := setupAffiliateReferralTestDB(t)
	previousAffiliateSetting := operation_setting.GetAffiliateSettingSnapshot()
	previousPaymentSetting := *operation_setting.GetPaymentSetting()
	t.Cleanup(func() {
		require.NoError(t, operation_setting.SetAffiliateSetting(previousAffiliateSetting))
		*operation_setting.GetPaymentSetting() = previousPaymentSetting
	})

	persisted := operation_setting.DefaultAffiliateSetting()
	persisted.CommissionEnabled = true
	encoded, err := operation_setting.MarshalAffiliateSetting(persisted)
	require.NoError(t, err)
	require.NoError(t, db.Create(&[]Option{
		{Key: operation_setting.AffiliateSettingOptionKey, Value: encoded},
		{Key: EpayCurrencyOptionKey, Value: "CNY"},
	}).Error)

	runtimeSetting := persisted
	runtimeSetting.CommissionEnabled = false
	require.NoError(t, operation_setting.SetAffiliateSetting(runtimeSetting))
	operation_setting.GetPaymentSetting().EpayCurrency = "USD"
	currency, err := GetEpayCurrencySnapshot()
	require.NoError(t, err)
	assert.Equal(t, "CNY", currency)

	err = UpdateOption(EpayCurrencyOptionKey, "USD")
	require.ErrorIs(t, err, ErrAffiliateEpayCurrencyConflict)
	var stored Option
	require.NoError(t, db.Where(commonKeyCol+" = ?", EpayCurrencyOptionKey).First(&stored).Error)
	assert.Equal(t, "CNY", stored.Value)
}

func TestUpdateOptionRejectsProtectedKeyVariants(t *testing.T) {
	setupAffiliateReferralTestDB(t)
	for _, testCase := range []struct {
		name  string
		key   string
		value string
	}{
		{name: "affiliate case variant", key: "affiliatesetting", value: `{}`},
		{name: "affiliate trailing space", key: operation_setting.AffiliateSettingOptionKey + " ", value: `{}`},
		{name: "Epay currency case variant", key: "PAYMENT_SETTING.EPAY_CURRENCY", value: "USD"},
		{name: "compliance canonical", key: PaymentComplianceConfirmedOptionKey, value: "true"},
		{name: "compliance case variant", key: "PAYMENT_SETTING.COMPLIANCE_CONFIRMED", value: "true"},
		{name: "legacy reward case variant", key: "quotaforinviter", value: "100"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			err := UpdateOption(testCase.key, testCase.value)
			assert.ErrorIs(t, err, ErrProtectedOptionKey)
		})
	}

	err := UpdateOptionsBulk(map[string]string{
		"PAYMENT_SETTING.COMPLIANCE_CONFIRMED": "true",
	})
	assert.ErrorIs(t, err, ErrProtectedOptionKey)
}
