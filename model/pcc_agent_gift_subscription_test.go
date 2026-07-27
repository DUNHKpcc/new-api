package model

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupPccAgentGiftSubscriptionTest(t *testing.T) *gorm.DB {
	t.Helper()

	previousDB, previousLogDB := DB, LOG_DB
	previousRedis := common.RedisEnabled
	previousMainType, previousLogType := common.MainDatabaseType(), common.LogDatabaseType()
	db, err := gorm.Open(
		sqlite.Open("file:"+strings.ReplaceAll(t.Name(), "/", "_")+"?mode=memory&cache=shared"),
		&gorm.Config{},
	)
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(
		&User{},
		&DesktopGrant{},
		&ExternalIdentityClaim{},
		&SubscriptionPlan{},
		&UserSubscription{},
		&SubscriptionPreConsumeRecord{},
	))

	DB, LOG_DB = db, db
	common.RedisEnabled = false
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	t.Cleanup(func() {
		DB, LOG_DB = previousDB, previousLogDB
		common.RedisEnabled = previousRedis
		common.SetDatabaseTypes(previousMainType, previousLogType)
		_ = sqlDB.Close()
	})
	return db
}

func createPccAgentGiftTestUser(t *testing.T, db *gorm.DB, username string) User {
	t.Helper()
	user := User{
		Username:    username,
		Password:    "unused-password-hash",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		AffCode:     username,
		AuthVersion: 1,
	}
	require.NoError(t, db.Create(&user).Error)
	return user
}

func createPccAgentGiftTestPlan(t *testing.T, db *gorm.DB, title string) SubscriptionPlan {
	t.Helper()
	plan := SubscriptionPlan{
		Title:              title,
		DurationUnit:       SubscriptionDurationMonth,
		DurationValue:      1,
		Enabled:            true,
		MaxPurchasePerUser: 1,
		TotalAmount:        500,
		UpgradeGroup:       "pro",
		DowngradeGroup:     "default",
	}
	require.NoError(t, db.Create(&plan).Error)
	InvalidateSubscriptionPlanCache(plan.Id)
	return plan
}

func TestPccAgentGiftRequiresWeChatAndCannotBeClaimedAgain(t *testing.T) {
	db := setupPccAgentGiftSubscriptionTest(t)
	first := createPccAgentGiftTestUser(t, db, "pcc-gift-first")
	second := createPccAgentGiftTestUser(t, db, "pcc-gift-second")
	plan := createPccAgentGiftTestPlan(t, db, "Pcc gift")

	err := db.Transaction(func(tx *gorm.DB) error {
		_, err := EnsurePccAgentGiftSubscriptionWithTx(tx, first.Id, plan.Id)
		return err
	})
	assert.ErrorIs(t, err, ErrExternalIdentityNotClaimed)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return ClaimExternalIdentityWithTx(
			tx,
			ExternalIdentityProviderWeChatUnionID,
			"wechat-union-once",
			first.Id,
		)
	}))
	var gift *UserSubscription
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var err error
		gift, err = EnsurePccAgentGiftSubscriptionWithTx(tx, first.Id, plan.Id)
		return err
	}))
	require.NotNil(t, gift)
	assert.Equal(t, UserSubscriptionSourcePccAgentGift, gift.Source)
	assert.Empty(t, gift.UpgradeGroup)
	assert.Empty(t, gift.DowngradeGroup)

	var storedFirst User
	require.NoError(t, db.First(&storedFirst, first.Id).Error)
	assert.Equal(t, "default", storedFirst.Group)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return ReleaseExternalIdentityWithTx(tx, ExternalIdentityProviderWeChatUnionID, first.Id)
	}))
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return ClaimExternalIdentityWithTx(
			tx,
			ExternalIdentityProviderWeChatUnionID,
			"wechat-union-once",
			second.Id,
		)
	}))
	err = db.Transaction(func(tx *gorm.DB) error {
		_, err := EnsurePccAgentGiftSubscriptionWithTx(tx, second.Id, plan.Id)
		return err
	})
	assert.ErrorIs(t, err, ErrExternalIdentityAlreadyClaimed)
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return ClaimExternalIdentityWithTx(
			tx,
			ExternalIdentityProviderWeChatUnionID,
			"wechat-union-new",
			first.Id,
		)
	}))

	require.NoError(t, db.Model(gift).Updates(map[string]interface{}{
		"status":   "cancelled",
		"end_time": time.Now().Unix(),
	}).Error)
	var repeated *UserSubscription
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var err error
		repeated, err = EnsurePccAgentGiftSubscriptionWithTx(tx, first.Id, plan.Id)
		return err
	}))
	assert.Equal(t, gift.Id, repeated.Id)

	var giftCount int64
	require.NoError(t, db.Model(&UserSubscription{}).
		Where("source = ?", UserSubscriptionSourcePccAgentGift).
		Count(&giftCount).Error)
	assert.EqualValues(t, 1, giftCount)

	require.NoError(t, db.Where("id = ?", gift.Id).Delete(&UserSubscription{}).Error)
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		_, err := EnsurePccAgentGiftSubscriptionWithTx(tx, first.Id, plan.Id)
		return err
	}))
	require.NoError(t, db.Model(&UserSubscription{}).
		Where("source = ?", UserSubscriptionSourcePccAgentGift).
		Count(&giftCount).Error)
	assert.Zero(t, giftCount)
}

func TestPccAgentGiftDoesNotConsumePurchaseLimitOrAllowHardDelete(t *testing.T) {
	db := setupPccAgentGiftSubscriptionTest(t)
	user := createPccAgentGiftTestUser(t, db, "pcc-gift-purchase-limit")
	plan := createPccAgentGiftTestPlan(t, db, "Shared plan")
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return ClaimExternalIdentityWithTx(
			tx,
			ExternalIdentityProviderWeChatUnionID,
			"wechat-purchase-limit",
			user.Id,
		)
	}))

	var gift *UserSubscription
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var err error
		gift, err = EnsurePccAgentGiftSubscriptionWithTx(tx, user.Id, plan.Id)
		return err
	}))
	count, err := CountUserSubscriptionsByPlan(user.Id, plan.Id)
	require.NoError(t, err)
	assert.Zero(t, count)

	var ordinary *UserSubscription
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var err error
		ordinary, err = CreateUserSubscriptionFromPlanTx(tx, user.Id, &plan, "admin")
		return err
	}))
	require.NotNil(t, ordinary)
	_, err = CreateUserSubscriptionFromPlanTx(db, user.Id, &plan, "admin")
	assert.EqualError(t, err, "已达到该套餐购买上限")

	_, err = AdminInvalidateUserSubscription(gift.Id)
	require.NoError(t, err)
	_, err = AdminDeleteUserSubscription(gift.Id)
	assert.ErrorIs(t, err, ErrPccAgentGiftDeleteForbidden)
}

func TestPccAgentGiftUsageIsIsolatedFromOrdinaryKeys(t *testing.T) {
	db := setupPccAgentGiftSubscriptionTest(t)
	user := createPccAgentGiftTestUser(t, db, "pcc-gift-source-isolation")
	plan := createPccAgentGiftTestPlan(t, db, "Isolated plan")
	now := time.Now().Unix()
	gift := UserSubscription{
		UserId:      user.Id,
		PlanId:      plan.Id,
		AmountTotal: 100,
		StartTime:   now,
		EndTime:     now + 3600,
		Status:      "active",
		Source:      UserSubscriptionSourcePccAgentGift,
	}
	ordinary := gift
	ordinary.Id = 0
	ordinary.Source = "order"
	require.NoError(t, db.Create(&gift).Error)
	require.NoError(t, db.Create(&ordinary).Error)

	standardResult, err := PreConsumeUserSubscription(
		fmt.Sprintf("standard-%d", user.Id),
		user.Id,
		"gpt-test",
		0,
		10,
	)
	require.NoError(t, err)
	assert.Equal(t, ordinary.Id, standardResult.UserSubscriptionId)

	giftResult, err := PreConsumeUserSubscriptionForScope(
		fmt.Sprintf("gift-%d", user.Id),
		user.Id,
		"gpt-test",
		0,
		10,
		SubscriptionUsageScopePccAgentGift,
	)
	require.NoError(t, err)
	assert.Equal(t, gift.Id, giftResult.UserSubscriptionId)

	_, err = PreConsumeUserSubscription(
		fmt.Sprintf("gift-%d", user.Id),
		user.Id,
		"gpt-test",
		0,
		10,
	)
	assert.ErrorIs(t, err, ErrSubscriptionSourceForbidden)

	retriedFallback, err := PreConsumeUserSubscriptionForScope(
		fmt.Sprintf("standard-%d", user.Id),
		user.Id,
		"gpt-test",
		0,
		10,
		SubscriptionUsageScopePccAgentGift,
	)
	require.NoError(t, err)
	assert.Equal(t, ordinary.Id, retriedFallback.UserSubscriptionId)
}

func TestPccAgentGiftPreConsumeFailsClosedOnQueryError(t *testing.T) {
	db := setupPccAgentGiftSubscriptionTest(t)
	user := createPccAgentGiftTestUser(t, db, "pcc-gift-query-error")
	forcedErr := errors.New("forced subscription query failure")
	const callbackName = "test:pcc-gift-subscription-query-failure"
	callbackRegistered := true
	require.NoError(t, db.Callback().Query().Before("gorm:query").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == "user_subscriptions" {
			tx.AddError(forcedErr)
		}
	}))
	t.Cleanup(func() {
		if callbackRegistered {
			_ = db.Callback().Query().Remove(callbackName)
		}
	})

	result, err := PreConsumeUserSubscriptionForScope(
		"pcc-gift-query-error",
		user.Id,
		"gpt-test",
		0,
		10,
		SubscriptionUsageScopePccAgentGift,
	)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, forcedErr)
	assert.NotErrorIs(t, err, ErrNoActiveSubscription)

	require.NoError(t, db.Callback().Query().Remove(callbackName))
	callbackRegistered = false
}

func TestPccAgentGiftPermanentClaimSurvivesAuthenticationCleanup(t *testing.T) {
	db := setupPccAgentGiftSubscriptionTest(t)
	user := createPccAgentGiftTestUser(t, db, "pcc-gift-claim-cleanup")
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		if err := ClaimExternalIdentityWithTx(tx, ExternalIdentityProviderWeChatUnionID, "wechat-cleanup", user.Id); err != nil {
			return err
		}
		return ClaimExternalIdentityWithTx(tx, ExternalIdentityProviderPccAgentGiftWeChat, "wechat-cleanup", user.Id)
	}))

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return releaseAllExternalIdentitiesWithTx(tx, user.Id)
	}))
	_, err := GetExternalIdentitySubjectByUserWithTx(db, ExternalIdentityProviderWeChatUnionID, user.Id)
	assert.True(t, errors.Is(err, ErrExternalIdentityNotClaimed))
	subject, err := GetExternalIdentitySubjectByUserWithTx(db, ExternalIdentityProviderPccAgentGiftWeChat, user.Id)
	require.NoError(t, err)
	assert.Equal(t, "wechat-cleanup", subject)
}
