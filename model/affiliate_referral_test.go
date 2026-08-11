package model

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupAffiliateReferralTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	previousDB := DB
	previousDatabaseType := common.MainDatabaseType()
	previousRedisEnabled := common.RedisEnabled
	dsn := fmt.Sprintf("file:affiliate-referral-%d?mode=memory&cache=shared&_busy_timeout=5000", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	// SQLite has one writer. A single pooled connection makes the concurrency
	// regression deterministic while still exercising competing goroutines.
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(
		&User{},
		&Option{},
		&AffiliateReferral{},
		&AffiliateSignupReward{},
		&AffiliateSignupRewardTransfer{},
		&AffiliateOutboxEvent{},
	))

	DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	common.RedisEnabled = false
	require.NoError(t, db.Create(&[]Option{
		{Key: PaymentComplianceConfirmedOptionKey, Value: "true"},
		{Key: PaymentComplianceTermsVersionOptionKey, Value: operation_setting.CurrentComplianceTermsVersion},
	}).Error)
	t.Cleanup(func() {
		DB = previousDB
		common.SetMainDatabaseType(previousDatabaseType)
		common.RedisEnabled = previousRedisEnabled
		_ = sqlDB.Close()
	})
	return db
}

func createAffiliateReferralTestUser(t *testing.T, db *gorm.DB, username, affCode string, status int) *User {
	t.Helper()
	user := &User{
		Username:    username,
		Password:    "unused-password",
		DisplayName: username,
		Status:      status,
		Role:        common.RoleCommonUser,
		Group:       "default",
		AffCode:     affCode,
	}
	require.NoError(t, db.Create(user).Error)
	return user
}

func setAffiliateSignupRewardTestSetting(t *testing.T, enabled bool, inviterQuota, inviteeQuota, version int64) {
	t.Helper()
	previous := operation_setting.GetAffiliateSettingSnapshot()
	previousPaymentSetting := *operation_setting.GetPaymentSetting()
	updated := previous
	updated.RegistrationRewardEnabled = enabled
	updated.InviterRewardQuota = inviterQuota
	updated.InviteeRewardQuota = inviteeQuota
	updated.Version = version
	require.NoError(t, operation_setting.SetAffiliateSetting(updated))
	encoded, err := operation_setting.MarshalAffiliateSetting(updated)
	require.NoError(t, err)
	require.NoError(t, UpdateOption(operation_setting.AffiliateSettingOptionKey, encoded))
	operation_setting.GetPaymentSetting().ComplianceConfirmed = true
	operation_setting.GetPaymentSetting().ComplianceTermsVersion = operation_setting.CurrentComplianceTermsVersion
	setAffiliatePaymentComplianceOptions(t, true)
	t.Cleanup(func() {
		require.NoError(t, operation_setting.SetAffiliateSetting(previous))
		*operation_setting.GetPaymentSetting() = previousPaymentSetting
	})
}

func setAffiliatePaymentComplianceOptions(t *testing.T, confirmed bool) {
	t.Helper()
	termsVersion := ""
	if confirmed {
		termsVersion = operation_setting.CurrentComplianceTermsVersion
	}
	require.NoError(t, UpdateOptionsBulk(map[string]string{
		PaymentComplianceConfirmedOptionKey:    fmt.Sprintf("%t", confirmed),
		PaymentComplianceTermsVersionOptionKey: termsVersion,
	}))
}

func TestUserInsertCreatesAffiliateLedgerInCreationTransaction(t *testing.T) {
	db := setupAffiliateReferralTestDB(t)
	setAffiliateSignupRewardTestSetting(t, true, 13, 5, 8)
	previousInviterQuota := common.QuotaForInviter
	previousInviteeQuota := common.QuotaForInvitee
	common.QuotaForInviter = 101
	common.QuotaForInvitee = 103
	t.Cleanup(func() {
		common.QuotaForInviter = previousInviterQuota
		common.QuotaForInvitee = previousInviteeQuota
	})
	inviter := createAffiliateReferralTestUser(t, db, "insert-inviter", "legacy", common.UserStatusEnabled)
	referred := &User{
		Username:    "insert-referred",
		Password:    "password123",
		DisplayName: "insert-referred",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
	}

	require.NoError(t, referred.Insert(inviter.Id))
	assert.Len(t, referred.AffCode, AffiliateCodeLength)

	var storedInviter User
	require.NoError(t, db.First(&storedInviter, inviter.Id).Error)
	assert.Equal(t, 1, storedInviter.AffCount)
	assert.Equal(t, 13, storedInviter.AffQuota)
	assert.Equal(t, 13, storedInviter.AffHistoryQuota)
	var storedReferred User
	require.NoError(t, db.First(&storedReferred, referred.Id).Error)
	assert.Equal(t, inviter.Id, storedReferred.InviterId)
	assert.Equal(t, 5, storedReferred.Quota)
	var rewardCount int64
	require.NoError(t, db.Model(&AffiliateSignupReward{}).Count(&rewardCount).Error)
	assert.EqualValues(t, 2, rewardCount)
}

func TestUserInsertCountsReferralWithoutRewardsBeforePaymentCompliance(t *testing.T) {
	db := setupAffiliateReferralTestDB(t)
	setAffiliateSignupRewardTestSetting(t, true, 13, 5, 8)
	operation_setting.GetPaymentSetting().ComplianceConfirmed = false
	operation_setting.GetPaymentSetting().ComplianceTermsVersion = ""
	setAffiliatePaymentComplianceOptions(t, false)
	inviter := createAffiliateReferralTestUser(t, db, "compliance-inviter", "compliance", common.UserStatusEnabled)
	referred := &User{
		Username:    "compliance-user",
		Password:    "password123",
		DisplayName: "compliance-user",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
	}

	require.NoError(t, referred.Insert(inviter.Id))
	var storedInviter User
	require.NoError(t, db.First(&storedInviter, inviter.Id).Error)
	assert.Equal(t, 1, storedInviter.AffCount)
	assert.Zero(t, storedInviter.AffQuota)
	assert.Zero(t, storedInviter.AffHistoryQuota)
	var storedReferred User
	require.NoError(t, db.First(&storedReferred, referred.Id).Error)
	assert.Equal(t, inviter.Id, storedReferred.InviterId)
	assert.Zero(t, storedReferred.Quota)
	var rewardCount int64
	require.NoError(t, db.Model(&AffiliateSignupReward{}).Count(&rewardCount).Error)
	assert.Zero(t, rewardCount)
}

func TestUserInsertUsesDatabaseAffiliateRewardSnapshot(t *testing.T) {
	db := setupAffiliateReferralTestDB(t)
	setAffiliateSignupRewardTestSetting(t, true, 1, 0, 2)
	persisted := operation_setting.GetAffiliateSettingSnapshot()
	persisted.InviterRewardQuota = 17
	persisted.Version = 9
	encoded, err := operation_setting.MarshalAffiliateSetting(persisted)
	require.NoError(t, err)
	require.NoError(t, db.Model(&Option{}).
		Where(commonKeyCol+" = ?", operation_setting.AffiliateSettingOptionKey).
		Update("value", encoded).Error)
	inviter := createAffiliateReferralTestUser(t, db, "db-setting-inviter", "dbsetting", common.UserStatusEnabled)
	referred := &User{
		Username: "db-setting-referred", Password: "password123", DisplayName: "db-setting-referred",
		Role: common.RoleCommonUser, Status: common.UserStatusEnabled,
	}

	require.NoError(t, referred.Insert(inviter.Id))
	var storedInviter User
	require.NoError(t, db.First(&storedInviter, inviter.Id).Error)
	assert.Equal(t, 17, storedInviter.AffQuota)
	var reward AffiliateSignupReward
	require.NoError(t, db.Where("beneficiary_role = ?", AffiliateSignupRewardRoleInviter).First(&reward).Error)
	assert.Equal(t, int64(9), reward.ConfigVersion)
}

func TestUserInsertRollsBackWhenAffiliateRewardFails(t *testing.T) {
	db := setupAffiliateReferralTestDB(t)
	setAffiliateSignupRewardTestSetting(t, true, 13, 5, 8)
	inviter := createAffiliateReferralTestUser(t, db, "insert-rollback-inviter", "rollback", common.UserStatusEnabled)
	require.NoError(t, db.Migrator().DropTable(&AffiliateSignupReward{}))
	referred := &User{
		Username:    "insert-rollback-referred",
		Password:    "password123",
		DisplayName: "insert-rollback-referred",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
	}

	require.Error(t, referred.Insert(inviter.Id))
	var userCount int64
	require.NoError(t, db.Model(&User{}).Where("username = ?", referred.Username).Count(&userCount).Error)
	assert.Zero(t, userCount)
	var referralCount int64
	require.NoError(t, db.Model(&AffiliateReferral{}).Count(&referralCount).Error)
	assert.Zero(t, referralCount)
	var storedInviter User
	require.NoError(t, db.First(&storedInviter, inviter.Id).Error)
	assert.Zero(t, storedInviter.AffCount)
	assert.Zero(t, storedInviter.AffQuota)
}

func TestAffiliateCodeGenerationRetriesCollisionAndPreservesLegacyCodes(t *testing.T) {
	db := setupAffiliateReferralTestDB(t)
	createAffiliateReferralTestUser(t, db, "code-owner", "COLLIDE1", common.UserStatusEnabled)
	newUser := &User{Username: "new-code-owner", Password: "password123", AffCode: "ignored"}
	candidates := []string{"COLLIDE1", "NEWCODE1"}
	nextCandidate := 0
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return createUserWithUniqueAffiliateCode(tx, newUser, func() string {
			candidate := candidates[nextCandidate]
			nextCandidate++
			return candidate
		})
	}))
	assert.Equal(t, "NEWCODE1", newUser.AffCode)
	assert.Equal(t, 2, nextCandidate)

	legacyOwner := createAffiliateReferralTestUser(t, db, "legacy-code-owner", "A1b2", common.UserStatusEnabled)
	code, err := ensureUserAffiliateCodeWithGenerator(legacyOwner.Id, func() string {
		return "UNUSED01"
	})
	require.NoError(t, err)
	assert.Equal(t, "A1b2", code)
}

func TestResolveAffiliateInviterIdByCode(t *testing.T) {
	db := setupAffiliateReferralTestDB(t)
	enabled := createAffiliateReferralTestUser(t, db, "resolve-enabled", "old4", common.UserStatusEnabled)
	createAffiliateReferralTestUser(t, db, "resolve-disabled", "disabled", common.UserStatusDisabled)

	id, err := ResolveAffiliateInviterIdByCode(" old4 ")
	require.NoError(t, err)
	assert.Equal(t, enabled.Id, id)
	_, err = ResolveAffiliateInviterIdByCode("missing")
	assert.ErrorIs(t, err, ErrAffiliateInviterNotFound)
	_, err = ResolveAffiliateInviterIdByCode("disabled")
	assert.ErrorIs(t, err, ErrAffiliateInviterDisabled)
	id, err = ResolveAffiliateInviterIdByCode("  ")
	require.NoError(t, err)
	assert.Zero(t, id)
}

func TestApplyAffiliateReferralWithTxCountsZeroRewardReferral(t *testing.T) {
	db := setupAffiliateReferralTestDB(t)
	inviter := createAffiliateReferralTestUser(t, db, "zero-reward-inviter", "zero-inviter", common.UserStatusEnabled)
	referred := createAffiliateReferralTestUser(t, db, "zero-reward-referred", "zero-referred", common.UserStatusEnabled)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		_, err := ApplyAffiliateReferralWithTx(tx, referred, inviter.Id, AffiliateSignupRewardConfigSnapshot{
			Enabled:            false,
			InviterRewardQuota: 700,
			InviteeRewardQuota: 300,
			ConfigVersion:      4,
		})
		return err
	}))

	var storedInviter User
	require.NoError(t, db.First(&storedInviter, inviter.Id).Error)
	assert.Equal(t, 1, storedInviter.AffCount)
	assert.Zero(t, storedInviter.AffQuota)
	assert.Zero(t, storedInviter.AffHistoryQuota)

	var storedReferred User
	require.NoError(t, db.First(&storedReferred, referred.Id).Error)
	assert.Equal(t, inviter.Id, storedReferred.InviterId)
	assert.Zero(t, storedReferred.Quota)

	var referralCount int64
	require.NoError(t, db.Model(&AffiliateReferral{}).Count(&referralCount).Error)
	assert.EqualValues(t, 1, referralCount)
	var rewardCount int64
	require.NoError(t, db.Model(&AffiliateSignupReward{}).Count(&rewardCount).Error)
	assert.Zero(t, rewardCount)
	var outboxEvents []AffiliateOutboxEvent
	require.NoError(t, db.Find(&outboxEvents).Error)
	require.Len(t, outboxEvents, 1)
	assert.Equal(t, AffiliateEventActionReferralBound, outboxEvents[0].Action)
	assert.Equal(t, inviter.Id, outboxEvents[0].UserId)
}

func TestApplyAffiliateReferralWithTxGrantsBothRewards(t *testing.T) {
	db := setupAffiliateReferralTestDB(t)
	inviter := createAffiliateReferralTestUser(t, db, "reward-inviter", "reward-inviter", common.UserStatusEnabled)
	referred := createAffiliateReferralTestUser(t, db, "reward-referred", "reward-referred", common.UserStatusEnabled)
	require.NoError(t, db.Model(inviter).Updates(map[string]interface{}{
		"quota":       900,
		"aff_count":   2,
		"aff_quota":   5,
		"aff_history": 7,
	}).Error)
	require.NoError(t, db.Model(referred).Update("quota", 100).Error)

	config := AffiliateSignupRewardConfigSnapshot{
		Enabled:            true,
		InviterRewardQuota: 30,
		InviteeRewardQuota: 20,
		ConfigVersion:      9,
	}
	var referral *AffiliateReferral
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var err error
		referral, err = ApplyAffiliateReferralWithTx(tx, referred, inviter.Id, config)
		return err
	}))
	require.NotNil(t, referral)
	assert.Equal(t, inviter.Id, referral.InviterUserId)
	assert.Equal(t, referred.Id, referral.ReferredUserId)
	assert.Equal(t, inviter.AffCode, referral.AffCodeSnapshot)

	var storedInviter User
	require.NoError(t, db.First(&storedInviter, inviter.Id).Error)
	assert.Equal(t, 900, storedInviter.Quota)
	assert.Equal(t, 3, storedInviter.AffCount)
	assert.Equal(t, 35, storedInviter.AffQuota)
	assert.Equal(t, 37, storedInviter.AffHistoryQuota)

	var storedReferred User
	require.NoError(t, db.First(&storedReferred, referred.Id).Error)
	assert.Equal(t, inviter.Id, storedReferred.InviterId)
	assert.Equal(t, 120, storedReferred.Quota)

	var rewards []AffiliateSignupReward
	require.NoError(t, db.Order("beneficiary_role ASC").Find(&rewards).Error)
	require.Len(t, rewards, 2)
	rewardByRole := map[string]AffiliateSignupReward{}
	for _, reward := range rewards {
		rewardByRole[reward.BeneficiaryRole] = reward
	}
	assert.Equal(t, int64(30), rewardByRole[AffiliateSignupRewardRoleInviter].RewardQuota)
	assert.Equal(t, inviter.Id, rewardByRole[AffiliateSignupRewardRoleInviter].BeneficiaryUserId)
	assert.Equal(t, int64(20), rewardByRole[AffiliateSignupRewardRoleInvitee].RewardQuota)
	assert.Equal(t, referred.Id, rewardByRole[AffiliateSignupRewardRoleInvitee].BeneficiaryUserId)
	for _, reward := range rewards {
		assert.Equal(t, referral.Id, reward.ReferralId)
		assert.Equal(t, AffiliateSignupRewardStatusGranted, reward.Status)
		assert.Equal(t, int64(9), reward.ConfigVersion)
	}
	var outboxEvents []AffiliateOutboxEvent
	require.NoError(t, db.Order("id asc").Find(&outboxEvents).Error)
	require.Len(t, outboxEvents, 3)
	assert.Equal(t, AffiliateEventActionReferralBound, outboxEvents[0].Action)
	assert.Equal(t, AffiliateEventActionSignupRewardGranted, outboxEvents[1].Action)
	assert.Equal(t, AffiliateEventActionSignupRewardGranted, outboxEvents[2].Action)
	for _, event := range outboxEvents {
		assert.NotContains(t, event.Payload, inviter.Username)
		assert.NotContains(t, event.Payload, referred.Username)
	}
}

func TestApplyAffiliateReferralWithTxIsIdempotent(t *testing.T) {
	db := setupAffiliateReferralTestDB(t)
	inviter := createAffiliateReferralTestUser(t, db, "idempotent-inviter", "idempotent-inviter", common.UserStatusEnabled)
	referred := createAffiliateReferralTestUser(t, db, "idempotent-referred", "idempotent-referred", common.UserStatusEnabled)
	firstConfig := AffiliateSignupRewardConfigSnapshot{Enabled: true, InviterRewardQuota: 11, InviteeRewardQuota: 7, ConfigVersion: 2}
	secondConfig := AffiliateSignupRewardConfigSnapshot{Enabled: true, InviterRewardQuota: 999, InviteeRewardQuota: 888, ConfigVersion: 3}

	for _, config := range []AffiliateSignupRewardConfigSnapshot{firstConfig, secondConfig} {
		require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
			_, err := ApplyAffiliateReferralWithTx(tx, referred, inviter.Id, config)
			return err
		}))
	}

	var storedInviter User
	require.NoError(t, db.First(&storedInviter, inviter.Id).Error)
	assert.Equal(t, 1, storedInviter.AffCount)
	assert.Equal(t, 11, storedInviter.AffQuota)
	assert.Equal(t, 11, storedInviter.AffHistoryQuota)
	var storedReferred User
	require.NoError(t, db.First(&storedReferred, referred.Id).Error)
	assert.Equal(t, inviter.Id, storedReferred.InviterId)
	assert.Equal(t, 7, storedReferred.Quota)
	var referralCount int64
	require.NoError(t, db.Model(&AffiliateReferral{}).Count(&referralCount).Error)
	assert.EqualValues(t, 1, referralCount)
	var rewardCount int64
	require.NoError(t, db.Model(&AffiliateSignupReward{}).Count(&rewardCount).Error)
	assert.EqualValues(t, 2, rewardCount)
	var outboxCount int64
	require.NoError(t, db.Model(&AffiliateOutboxEvent{}).Count(&outboxCount).Error)
	assert.EqualValues(t, 3, outboxCount)
}

func TestApplyAffiliateReferralWithTxRejectsInvalidInviters(t *testing.T) {
	tests := []struct {
		name      string
		prepareID func(t *testing.T, db *gorm.DB, referred *User) int
		wantErr   error
	}{
		{
			name: "missing",
			prepareID: func(_ *testing.T, _ *gorm.DB, _ *User) int {
				return 999999
			},
			wantErr: ErrAffiliateInviterNotFound,
		},
		{
			name: "disabled",
			prepareID: func(t *testing.T, db *gorm.DB, _ *User) int {
				return createAffiliateReferralTestUser(t, db, "disabled-inviter", "disabled-inviter", common.UserStatusDisabled).Id
			},
			wantErr: ErrAffiliateInviterDisabled,
		},
		{
			name: "self",
			prepareID: func(_ *testing.T, _ *gorm.DB, referred *User) int {
				return referred.Id
			},
			wantErr: ErrAffiliateSelfReferral,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := setupAffiliateReferralTestDB(t)
			referred := createAffiliateReferralTestUser(t, db, "invalid-referred", "invalid-referred", common.UserStatusEnabled)
			inviterID := test.prepareID(t, db, referred)
			err := db.Transaction(func(tx *gorm.DB) error {
				_, err := ApplyAffiliateReferralWithTx(tx, referred, inviterID, AffiliateSignupRewardConfigSnapshot{Enabled: true, InviterRewardQuota: 5, InviteeRewardQuota: 3, ConfigVersion: 1})
				return err
			})
			assert.ErrorIs(t, err, test.wantErr)

			var storedReferred User
			require.NoError(t, db.First(&storedReferred, referred.Id).Error)
			assert.Zero(t, storedReferred.InviterId)
			var referralCount int64
			require.NoError(t, db.Model(&AffiliateReferral{}).Count(&referralCount).Error)
			assert.Zero(t, referralCount)
			var rewardCount int64
			require.NoError(t, db.Model(&AffiliateSignupReward{}).Count(&rewardCount).Error)
			assert.Zero(t, rewardCount)
		})
	}
}

func TestApplyAffiliateReferralWithTxRejectsInvalidConfigVersion(t *testing.T) {
	db := setupAffiliateReferralTestDB(t)
	inviter := createAffiliateReferralTestUser(t, db, "invalid-config-inviter", "invalid-config-inviter", common.UserStatusEnabled)
	referred := createAffiliateReferralTestUser(t, db, "invalid-config-referred", "invalid-config-referred", common.UserStatusEnabled)

	err := db.Transaction(func(tx *gorm.DB) error {
		_, err := ApplyAffiliateReferralWithTx(tx, referred, inviter.Id, AffiliateSignupRewardConfigSnapshot{
			Enabled:       true,
			ConfigVersion: 0,
		})
		return err
	})
	assert.ErrorIs(t, err, ErrAffiliateRewardConfigInvalid)
}

func TestApplyAffiliateReferralWithTxRejectsRewardBalanceOverflow(t *testing.T) {
	tests := []struct {
		name           string
		inviterUpdates map[string]interface{}
		referredQuota  int
		inviterReward  int64
		inviteeReward  int64
	}{
		{
			name:           "inviter available quota",
			inviterUpdates: map[string]interface{}{"aff_quota": common.MaxQuota},
			inviterReward:  1,
		},
		{
			name:           "inviter historical quota",
			inviterUpdates: map[string]interface{}{"aff_history": common.MaxQuota},
			inviterReward:  1,
		},
		{
			name:          "invitee main quota",
			referredQuota: common.MaxQuota,
			inviteeReward: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := setupAffiliateReferralTestDB(t)
			inviter := createAffiliateReferralTestUser(t, db, "overflow-inviter", "overflow-inviter", common.UserStatusEnabled)
			referred := createAffiliateReferralTestUser(t, db, "overflow-referred", "overflow-referred", common.UserStatusEnabled)
			if len(test.inviterUpdates) > 0 {
				require.NoError(t, db.Model(inviter).Updates(test.inviterUpdates).Error)
			}
			if test.referredQuota > 0 {
				require.NoError(t, db.Model(referred).Update("quota", test.referredQuota).Error)
			}

			err := db.Transaction(func(tx *gorm.DB) error {
				_, err := ApplyAffiliateReferralWithTx(tx, referred, inviter.Id, AffiliateSignupRewardConfigSnapshot{
					Enabled:            true,
					InviterRewardQuota: test.inviterReward,
					InviteeRewardQuota: test.inviteeReward,
					ConfigVersion:      1,
				})
				return err
			})
			assert.ErrorIs(t, err, ErrAffiliateRewardBalanceOverflow)

			var storedInviter User
			require.NoError(t, db.First(&storedInviter, inviter.Id).Error)
			assert.Zero(t, storedInviter.AffCount)
			var storedReferred User
			require.NoError(t, db.First(&storedReferred, referred.Id).Error)
			assert.Zero(t, storedReferred.InviterId)
			var referralCount int64
			require.NoError(t, db.Model(&AffiliateReferral{}).Count(&referralCount).Error)
			assert.Zero(t, referralCount)
			var rewardCount int64
			require.NoError(t, db.Model(&AffiliateSignupReward{}).Count(&rewardCount).Error)
			assert.Zero(t, rewardCount)
		})
	}
}

func TestApplyAffiliateReferralWithTxRollsBackWhenRewardWriteFails(t *testing.T) {
	db := setupAffiliateReferralTestDB(t)
	inviter := createAffiliateReferralTestUser(t, db, "rollback-inviter", "rollback-inviter", common.UserStatusEnabled)
	require.NoError(t, db.Migrator().DropTable(&AffiliateSignupReward{}))

	referred := &User{
		Username:    "rollback-referred",
		Password:    "unused-password",
		DisplayName: "rollback-referred",
		Status:      common.UserStatusEnabled,
		Role:        common.RoleCommonUser,
		Group:       "default",
		AffCode:     "rollback-referred",
	}
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(referred).Error; err != nil {
			return err
		}
		_, err := ApplyAffiliateReferralWithTx(tx, referred, inviter.Id, AffiliateSignupRewardConfigSnapshot{Enabled: true, InviterRewardQuota: 5, InviteeRewardQuota: 3, ConfigVersion: 1})
		return err
	})
	require.Error(t, err)

	var storedInviter User
	require.NoError(t, db.First(&storedInviter, inviter.Id).Error)
	assert.Zero(t, storedInviter.AffCount)
	assert.Zero(t, storedInviter.AffQuota)
	assert.Zero(t, storedInviter.AffHistoryQuota)
	var referredCount int64
	require.NoError(t, db.Model(&User{}).Where("username = ?", referred.Username).Count(&referredCount).Error)
	assert.Zero(t, referredCount)
	var referralCount int64
	require.NoError(t, db.Model(&AffiliateReferral{}).Count(&referralCount).Error)
	assert.Zero(t, referralCount)
}

func TestApplyAffiliateReferralWithTxRollsBackWhenOutboxWriteFails(t *testing.T) {
	db := setupAffiliateReferralTestDB(t)
	inviter := createAffiliateReferralTestUser(t, db, "outbox-rollback-inviter", "outbox-rollback-inviter", common.UserStatusEnabled)
	referred := createAffiliateReferralTestUser(t, db, "outbox-rollback-user", "outbox-rollback-user", common.UserStatusEnabled)
	require.NoError(t, db.Migrator().DropTable(&AffiliateOutboxEvent{}))

	err := db.Transaction(func(tx *gorm.DB) error {
		_, err := ApplyAffiliateReferralWithTx(tx, referred, inviter.Id, AffiliateSignupRewardConfigSnapshot{
			Enabled:       false,
			ConfigVersion: 1,
		})
		return err
	})
	require.Error(t, err)

	var storedInviter User
	require.NoError(t, db.First(&storedInviter, inviter.Id).Error)
	assert.Zero(t, storedInviter.AffCount)
	var storedReferred User
	require.NoError(t, db.First(&storedReferred, referred.Id).Error)
	assert.Zero(t, storedReferred.InviterId)
	var referralCount int64
	require.NoError(t, db.Model(&AffiliateReferral{}).Count(&referralCount).Error)
	assert.Zero(t, referralCount)
}

func TestApplyAffiliateReferralWithTxSQLiteConcurrentRegistrations(t *testing.T) {
	db := setupAffiliateReferralTestDB(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(8)
	inviter := createAffiliateReferralTestUser(t, db, "concurrent-inviter", "concurrent-inviter", common.UserStatusEnabled)
	const registrations = 12
	referredUsers := make([]*User, 0, registrations)
	for i := 0; i < registrations; i++ {
		referredUsers = append(referredUsers, createAffiliateReferralTestUser(
			t,
			db,
			fmt.Sprintf("concurrent-referred-%d", i),
			fmt.Sprintf("concurrent-code-%d", i),
			common.UserStatusEnabled,
		))
	}

	start := make(chan struct{})
	errorsByRegistration := make([]error, registrations)
	var waitGroup sync.WaitGroup
	for i := range referredUsers {
		waitGroup.Add(1)
		go func(index int) {
			defer waitGroup.Done()
			<-start
			errorsByRegistration[index] = db.Transaction(func(tx *gorm.DB) error {
				_, err := ApplyAffiliateReferralWithTx(tx, referredUsers[index], inviter.Id, AffiliateSignupRewardConfigSnapshot{
					Enabled:            true,
					InviterRewardQuota: 7,
					InviteeRewardQuota: 3,
					ConfigVersion:      1,
				})
				return err
			})
		}(i)
	}
	close(start)
	waitGroup.Wait()
	successfulRegistrations := 0
	for _, registrationErr := range errorsByRegistration {
		if registrationErr == nil {
			successfulRegistrations++
			continue
		}
		assert.Contains(t, strings.ToLower(registrationErr.Error()), "locked")
	}
	require.Greater(t, successfulRegistrations, 0)

	var storedInviter User
	require.NoError(t, db.First(&storedInviter, inviter.Id).Error)
	assert.Equal(t, successfulRegistrations, storedInviter.AffCount)
	assert.Equal(t, successfulRegistrations*7, storedInviter.AffQuota)
	assert.Equal(t, successfulRegistrations*7, storedInviter.AffHistoryQuota)
	var referralCount int64
	require.NoError(t, db.Model(&AffiliateReferral{}).Count(&referralCount).Error)
	assert.EqualValues(t, successfulRegistrations, referralCount)
	var rewardCount int64
	require.NoError(t, db.Model(&AffiliateSignupReward{}).Count(&rewardCount).Error)
	assert.EqualValues(t, successfulRegistrations*2, rewardCount)
	for index, referred := range referredUsers {
		var stored User
		require.NoError(t, db.First(&stored, referred.Id).Error)
		if errorsByRegistration[index] == nil {
			assert.Equal(t, inviter.Id, stored.InviterId)
			assert.Equal(t, 3, stored.Quota)
		} else {
			assert.Zero(t, stored.InviterId)
			assert.Zero(t, stored.Quota)
		}
	}
}

func TestApplyAffiliateReferralWithTxRejectsConflictingReentry(t *testing.T) {
	db := setupAffiliateReferralTestDB(t)
	firstInviter := createAffiliateReferralTestUser(t, db, "first-inviter", "first-inviter", common.UserStatusEnabled)
	secondInviter := createAffiliateReferralTestUser(t, db, "second-inviter", "second-inviter", common.UserStatusEnabled)
	referred := createAffiliateReferralTestUser(t, db, "conflict-referred", "conflict-referred", common.UserStatusEnabled)
	config := AffiliateSignupRewardConfigSnapshot{Enabled: true, InviterRewardQuota: 5, InviteeRewardQuota: 2, ConfigVersion: 1}

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		_, err := ApplyAffiliateReferralWithTx(tx, referred, firstInviter.Id, config)
		return err
	}))
	err := db.Transaction(func(tx *gorm.DB) error {
		_, err := ApplyAffiliateReferralWithTx(tx, referred, secondInviter.Id, config)
		return err
	})
	assert.True(t, errors.Is(err, ErrAffiliateReferralConflict))

	var storedFirst User
	require.NoError(t, db.First(&storedFirst, firstInviter.Id).Error)
	assert.Equal(t, 1, storedFirst.AffCount)
	assert.Equal(t, 5, storedFirst.AffQuota)
	var storedSecond User
	require.NoError(t, db.First(&storedSecond, secondInviter.Id).Error)
	assert.Zero(t, storedSecond.AffCount)
	assert.Zero(t, storedSecond.AffQuota)
}

func TestAffiliateLedgerUniqueConstraints(t *testing.T) {
	db := setupAffiliateReferralTestDB(t)
	firstInviter := createAffiliateReferralTestUser(t, db, "constraint-first", "constraint-first", common.UserStatusEnabled)
	secondInviter := createAffiliateReferralTestUser(t, db, "constraint-second", "constraint-second", common.UserStatusEnabled)
	referred := createAffiliateReferralTestUser(t, db, "constraint-referred", "constraint-referred", common.UserStatusEnabled)

	referral := &AffiliateReferral{
		InviterUserId:   firstInviter.Id,
		ReferredUserId:  referred.Id,
		AffCodeSnapshot: firstInviter.AffCode,
		BoundAt:         common.GetTimestamp(),
	}
	require.NoError(t, db.Create(referral).Error)
	assert.Error(t, db.Create(&AffiliateReferral{
		InviterUserId:   secondInviter.Id,
		ReferredUserId:  referred.Id,
		AffCodeSnapshot: secondInviter.AffCode,
		BoundAt:         common.GetTimestamp(),
	}).Error)

	reward := AffiliateSignupReward{
		ReferralId:        referral.Id,
		BeneficiaryUserId: firstInviter.Id,
		BeneficiaryRole:   AffiliateSignupRewardRoleInviter,
		RewardQuota:       10,
		Status:            AffiliateSignupRewardStatusGranted,
		ConfigVersion:     1,
		CreatedAt:         common.GetTimestamp(),
	}
	require.NoError(t, db.Create(&reward).Error)
	reward.Id = 0
	assert.Error(t, db.Create(&reward).Error)
	reward.Id = 0
	reward.BeneficiaryUserId = referred.Id
	reward.BeneficiaryRole = AffiliateSignupRewardRoleInvitee
	require.NoError(t, db.Create(&reward).Error)

	transfer := AffiliateSignupRewardTransfer{
		UserId:           firstInviter.Id,
		IdempotencyKey:   "same-request",
		TransferredQuota: 5,
		AffQuotaBefore:   10,
		AffQuotaAfter:    5,
		MainQuotaBefore:  0,
		MainQuotaAfter:   5,
		CreatedAt:        common.GetTimestamp(),
	}
	require.NoError(t, db.Create(&transfer).Error)
	transfer.Id = 0
	assert.Error(t, db.Create(&transfer).Error)
	transfer.Id = 0
	transfer.UserId = secondInviter.Id
	require.NoError(t, db.Create(&transfer).Error)
}

func TestTransferAffiliateSignupRewardsIsIdempotent(t *testing.T) {
	db := setupAffiliateReferralTestDB(t)
	user := createAffiliateReferralTestUser(t, db, "transfer-user", "transfer-user", common.UserStatusEnabled)
	require.NoError(t, db.Model(user).Updates(map[string]interface{}{
		"quota":     100,
		"aff_quota": 50,
	}).Error)

	first, err := TransferAffiliateSignupRewards(user.Id, 30, "request-1")
	require.NoError(t, err)
	assert.False(t, first.AlreadyCompleted)
	assert.Equal(t, int64(30), first.Transfer.TransferredQuota)
	assert.Equal(t, int64(50), first.Transfer.AffQuotaBefore)
	assert.Equal(t, int64(20), first.Transfer.AffQuotaAfter)
	assert.Equal(t, int64(100), first.Transfer.MainQuotaBefore)
	assert.Equal(t, int64(130), first.Transfer.MainQuotaAfter)

	replayed, err := TransferAffiliateSignupRewards(user.Id, 30, " request-1 ")
	require.NoError(t, err)
	assert.True(t, replayed.AlreadyCompleted)
	assert.Equal(t, first.Transfer.Id, replayed.Transfer.Id)

	var stored User
	require.NoError(t, db.First(&stored, user.Id).Error)
	assert.Equal(t, 20, stored.AffQuota)
	assert.Equal(t, 130, stored.Quota)
	var transferCount int64
	require.NoError(t, db.Model(&AffiliateSignupRewardTransfer{}).Count(&transferCount).Error)
	assert.EqualValues(t, 1, transferCount)
	var outboxEvents []AffiliateOutboxEvent
	require.NoError(t, db.Find(&outboxEvents).Error)
	require.Len(t, outboxEvents, 1)
	assert.Equal(t, AffiliateEventActionSignupRewardTransferred, outboxEvents[0].Action)
	assert.Equal(t, user.Id, outboxEvents[0].UserId)
	assert.Contains(t, outboxEvents[0].DedupKey, fmt.Sprintf(":%d", first.Transfer.Id))

	_, err = TransferAffiliateSignupRewards(user.Id, 20, "request-1")
	assert.ErrorIs(t, err, ErrAffiliateSignupRewardTransferConflict)
}

func TestTransferAffQuotaToQuotaUsesSignupRewardTransferLedger(t *testing.T) {
	db := setupAffiliateReferralTestDB(t)
	previousQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 100
	t.Cleanup(func() {
		common.QuotaPerUnit = previousQuotaPerUnit
	})
	user := createAffiliateReferralTestUser(t, db, "legacy-transfer-user", "legacy-transfer-user", common.UserStatusEnabled)
	require.NoError(t, db.Model(user).Updates(map[string]interface{}{
		"quota":     10,
		"aff_quota": 150,
	}).Error)

	require.NoError(t, user.TransferAffQuotaToQuota(100))
	var stored User
	require.NoError(t, db.First(&stored, user.Id).Error)
	assert.Equal(t, 50, stored.AffQuota)
	assert.Equal(t, 110, stored.Quota)
	var transfers []AffiliateSignupRewardTransfer
	require.NoError(t, db.Find(&transfers).Error)
	require.Len(t, transfers, 1)
	assert.Equal(t, int64(100), transfers[0].TransferredQuota)
	assert.True(t, strings.HasPrefix(transfers[0].IdempotencyKey, "legacy_"))
	var outboxCount int64
	require.NoError(t, db.Model(&AffiliateOutboxEvent{}).Count(&outboxCount).Error)
	assert.EqualValues(t, 1, outboxCount)
}

func TestTransferAffiliateSignupRewardsRejectsInvalidBalances(t *testing.T) {
	tests := []struct {
		name     string
		affQuota int
		quota    int
		transfer int64
		wantErr  error
	}{
		{
			name:     "insufficient affiliate quota",
			affQuota: 9,
			transfer: 10,
			wantErr:  ErrAffiliateSignupRewardInsufficient,
		},
		{
			name:     "main quota overflow",
			affQuota: 10,
			quota:    common.MaxQuota,
			transfer: 1,
			wantErr:  ErrAffiliateSignupRewardTransferOverflow,
		},
		{
			name:     "affiliate quota above storage limit",
			affQuota: common.MaxQuota + 1,
			transfer: 1,
			wantErr:  ErrAffiliateSignupRewardTransferOverflow,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := setupAffiliateReferralTestDB(t)
			user := createAffiliateReferralTestUser(t, db, "invalid-transfer-user", "invalid-transfer-user", common.UserStatusEnabled)
			require.NoError(t, db.Model(user).Updates(map[string]interface{}{
				"quota":     test.quota,
				"aff_quota": test.affQuota,
			}).Error)

			_, err := TransferAffiliateSignupRewards(user.Id, test.transfer, "invalid-balance")
			assert.ErrorIs(t, err, test.wantErr)
			var transferCount int64
			require.NoError(t, db.Model(&AffiliateSignupRewardTransfer{}).Count(&transferCount).Error)
			assert.Zero(t, transferCount)
			var stored User
			require.NoError(t, db.First(&stored, user.Id).Error)
			assert.Equal(t, test.affQuota, stored.AffQuota)
			assert.Equal(t, test.quota, stored.Quota)
		})
	}
}

func TestTransferAffiliateSignupRewardsRollsBackWhenOutboxWriteFails(t *testing.T) {
	db := setupAffiliateReferralTestDB(t)
	user := createAffiliateReferralTestUser(t, db, "transfer-outbox-failure", "transfer-outbox-failure", common.UserStatusEnabled)
	require.NoError(t, db.Model(user).Updates(map[string]interface{}{
		"quota":     100,
		"aff_quota": 50,
	}).Error)
	require.NoError(t, db.Migrator().DropTable(&AffiliateOutboxEvent{}))

	_, err := TransferAffiliateSignupRewards(user.Id, 30, "outbox-failure")
	require.Error(t, err)
	var stored User
	require.NoError(t, db.First(&stored, user.Id).Error)
	assert.Equal(t, 50, stored.AffQuota)
	assert.Equal(t, 100, stored.Quota)
	var transferCount int64
	require.NoError(t, db.Model(&AffiliateSignupRewardTransfer{}).Count(&transferCount).Error)
	assert.Zero(t, transferCount)
}
