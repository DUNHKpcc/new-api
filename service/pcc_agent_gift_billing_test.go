package service

import (
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestPccAgentBillingUsesGiftBeforeOrdinarySubscription(t *testing.T) {
	previousDB, previousLogDB := model.DB, model.LOG_DB
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
		&model.User{},
		&model.DesktopGrant{},
		&model.SubscriptionPlan{},
		&model.UserSubscription{},
		&model.SubscriptionPreConsumeRecord{},
	))
	model.DB, model.LOG_DB = db, db
	common.RedisEnabled = false
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	t.Cleanup(func() {
		model.DB, model.LOG_DB = previousDB, previousLogDB
		common.RedisEnabled = previousRedis
		common.SetDatabaseTypes(previousMainType, previousLogType)
		_ = sqlDB.Close()
	})

	user := model.User{
		Username: "pcc-billing-user",
		Password: "unused-password-hash",
		Status:   common.UserStatusEnabled,
		Group:    "default",
	}
	require.NoError(t, db.Create(&user).Error)
	plan := model.SubscriptionPlan{
		Title:         "Shared billing plan",
		DurationUnit:  model.SubscriptionDurationMonth,
		DurationValue: 1,
		Enabled:       true,
		TotalAmount:   100,
	}
	require.NoError(t, db.Create(&plan).Error)
	model.InvalidateSubscriptionPlanCache(plan.Id)
	now := time.Now().Unix()
	gift := model.UserSubscription{
		UserId:      user.Id,
		PlanId:      plan.Id,
		AmountTotal: 5,
		StartTime:   now,
		EndTime:     now + 3600,
		Status:      "active",
		Source:      model.UserSubscriptionSourcePccAgentGift,
	}
	ordinary := gift
	ordinary.Id = 0
	ordinary.AmountTotal = 100
	ordinary.Source = "order"
	require.NoError(t, db.Create(&gift).Error)
	require.NoError(t, db.Create(&ordinary).Error)
	pccTokenId := 55
	require.NoError(t, db.Create(&model.DesktopGrant{
		PublicId:    "pcc-billing-grant",
		UserId:      user.Id,
		ClientId:    DesktopClientID,
		TokenId:     &pccTokenId,
		Status:      model.DesktopGrantStatusActive,
		ActiveSlot:  common.GetPointer(1),
		ExpiredTime: now + 3600,
	}).Error)

	newRelayInfo := func(tokenId int, requestId string) *relaycommon.RelayInfo {
		return &relaycommon.RelayInfo{
			TokenId:         tokenId,
			UserId:          user.Id,
			OriginModelName: "gpt-test",
			RequestId:       requestId,
			IsPlayground:    true,
			UserSetting: dto.UserSetting{
				BillingPreference: "subscription_only",
			},
		}
	}
	context, _ := gin.CreateTestContext(nil)

	firstInfo := newRelayInfo(pccTokenId, "pcc-gift-first")
	_, apiErr := NewBillingSession(context, firstInfo, 3)
	require.Nil(t, apiErr)
	assert.Equal(t, gift.Id, firstInfo.SubscriptionId)

	fallbackInfo := newRelayInfo(pccTokenId, "pcc-gift-fallback")
	_, apiErr = NewBillingSession(context, fallbackInfo, 3)
	require.Nil(t, apiErr)
	assert.Equal(t, ordinary.Id, fallbackInfo.SubscriptionId)

	ordinaryInfo := newRelayInfo(99, "ordinary-key")
	_, apiErr = NewBillingSession(context, ordinaryInfo, 3)
	require.Nil(t, apiErr)
	assert.Equal(t, ordinary.Id, ordinaryInfo.SubscriptionId)

	require.NoError(t, db.First(&gift, gift.Id).Error)
	require.NoError(t, db.First(&ordinary, ordinary.Id).Error)
	assert.EqualValues(t, 3, gift.AmountUsed)
	assert.EqualValues(t, 6, ordinary.AmountUsed)
}
