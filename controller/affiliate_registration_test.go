package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/oauth"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupAffiliateRegistrationControllerTest(t *testing.T) *gorm.DB {
	t.Helper()
	previousDB := model.DB
	previousDatabaseType := common.MainDatabaseType()
	previousRegisterEnabled := common.RegisterEnabled
	previousPasswordRegisterEnabled := common.PasswordRegisterEnabled
	previousEmailVerificationEnabled := common.EmailVerificationEnabled
	previousWeChatVerificationEnabled := common.WeChatRegistrationVerificationEnabled
	previousWeChatAuthEnabled := common.WeChatAuthEnabled
	previousWeChatAppID := common.WeChatAppId
	previousWeChatAppSecret := common.WeChatAppSecret
	previousGenerateDefaultToken := constant.GenerateDefaultToken
	previousQuotaForNewUser := common.QuotaForNewUser
	previousQuotaForInviter := common.QuotaForInviter
	previousQuotaForInvitee := common.QuotaForInvitee
	previousRedisEnabled := common.RedisEnabled
	previousAffiliateSetting := operation_setting.GetAffiliateSettingSnapshot()
	previousPaymentSetting := *operation_setting.GetPaymentSetting()

	dsn := fmt.Sprintf("file:affiliate-registration-controller-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.Option{},
		&model.AuthFlow{},
		&model.ExternalIdentityClaim{},
		&model.UserOAuthBinding{},
		&model.AffiliateReferral{},
		&model.AffiliateSignupReward{},
		&model.AffiliateSignupRewardTransfer{},
		&model.AffiliateOutboxEvent{},
	))
	model.DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	common.RegisterEnabled = true
	common.PasswordRegisterEnabled = true
	common.EmailVerificationEnabled = false
	common.WeChatRegistrationVerificationEnabled = false
	common.WeChatAuthEnabled = false
	common.WeChatAppId = ""
	common.WeChatAppSecret = ""
	constant.GenerateDefaultToken = false
	common.QuotaForNewUser = 0
	common.QuotaForInviter = 101
	common.QuotaForInvitee = 103
	common.RedisEnabled = false
	affiliateSetting := previousAffiliateSetting
	affiliateSetting.RegistrationRewardEnabled = true
	affiliateSetting.InviterRewardQuota = 17
	affiliateSetting.InviteeRewardQuota = 9
	affiliateSetting.Version = 21
	require.NoError(t, operation_setting.SetAffiliateSetting(affiliateSetting))
	encodedAffiliateSetting, err := operation_setting.MarshalAffiliateSetting(affiliateSetting)
	require.NoError(t, err)
	operation_setting.GetPaymentSetting().ComplianceConfirmed = true
	operation_setting.GetPaymentSetting().ComplianceTermsVersion = operation_setting.CurrentComplianceTermsVersion
	require.NoError(t, db.Create(&[]model.Option{
		{Key: operation_setting.AffiliateSettingOptionKey, Value: encodedAffiliateSetting},
		{Key: model.PaymentComplianceConfirmedOptionKey, Value: "true"},
		{Key: model.PaymentComplianceTermsVersionOptionKey, Value: operation_setting.CurrentComplianceTermsVersion},
	}).Error)

	t.Cleanup(func() {
		model.DB = previousDB
		common.SetMainDatabaseType(previousDatabaseType)
		common.RegisterEnabled = previousRegisterEnabled
		common.PasswordRegisterEnabled = previousPasswordRegisterEnabled
		common.EmailVerificationEnabled = previousEmailVerificationEnabled
		common.WeChatRegistrationVerificationEnabled = previousWeChatVerificationEnabled
		common.WeChatAuthEnabled = previousWeChatAuthEnabled
		common.WeChatAppId = previousWeChatAppID
		common.WeChatAppSecret = previousWeChatAppSecret
		constant.GenerateDefaultToken = previousGenerateDefaultToken
		common.QuotaForNewUser = previousQuotaForNewUser
		common.QuotaForInviter = previousQuotaForInviter
		common.QuotaForInvitee = previousQuotaForInvitee
		common.RedisEnabled = previousRedisEnabled
		require.NoError(t, operation_setting.SetAffiliateSetting(previousAffiliateSetting))
		*operation_setting.GetPaymentSetting() = previousPaymentSetting
		sqlDB, dbErr := db.DB()
		if dbErr == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func createAffiliateRegistrationInviter(t *testing.T, db *gorm.DB, username, affCode string) *model.User {
	t.Helper()
	inviter := &model.User{
		Username:    username,
		Password:    "password123",
		DisplayName: username,
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		AffCode:     affCode,
	}
	require.NoError(t, db.Create(inviter).Error)
	return inviter
}

func sendAffiliateRegistrationRequest(t *testing.T, body string) map[string]any {
	t.Helper()
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/user/register", strings.NewReader(body))
	context.Request.Header.Set("Content-Type", "application/json")
	Register(context)
	var response map[string]any
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	return response
}

func assertAffiliateRegistrationBalances(t *testing.T, db *gorm.DB, inviterID, referredID int) {
	t.Helper()
	var inviter model.User
	require.NoError(t, db.First(&inviter, inviterID).Error)
	assert.Equal(t, 1, inviter.AffCount)
	assert.Equal(t, 17, inviter.AffQuota)
	assert.Equal(t, 17, inviter.AffHistoryQuota)
	var referred model.User
	require.NoError(t, db.First(&referred, referredID).Error)
	assert.Equal(t, inviterID, referred.InviterId)
	assert.Equal(t, 9, referred.Quota)
	var referralCount int64
	require.NoError(t, db.Model(&model.AffiliateReferral{}).Count(&referralCount).Error)
	assert.EqualValues(t, 1, referralCount)
	var rewardCount int64
	require.NoError(t, db.Model(&model.AffiliateSignupReward{}).Count(&rewardCount).Error)
	assert.EqualValues(t, 2, rewardCount)
}

func TestPasswordRegistrationCreatesAffiliateLedger(t *testing.T) {
	db := setupAffiliateRegistrationControllerTest(t)
	inviter := createAffiliateRegistrationInviter(t, db, "password-inviter", "old4")

	response := sendAffiliateRegistrationRequest(t, `{
		"username":"password-referred",
		"password":"password123",
		"aff_code":"old4"
	}`)
	assert.Equal(t, true, response["success"])
	var referred model.User
	require.NoError(t, db.Where("username = ?", "password-referred").First(&referred).Error)
	assertAffiliateRegistrationBalances(t, db, inviter.Id, referred.Id)
}

func TestPasswordRegistrationRejectsInvalidAffiliateCode(t *testing.T) {
	db := setupAffiliateRegistrationControllerTest(t)

	response := sendAffiliateRegistrationRequest(t, `{
		"username":"invalid-referred",
		"password":"password123",
		"aff_code":"missing-code"
	}`)
	assert.Equal(t, false, response["success"])
	assert.Equal(t, model.ErrAffiliateInviterNotFound.Error(), response["message"])
	var userCount int64
	require.NoError(t, db.Model(&model.User{}).Where("username = ?", "invalid-referred").Count(&userCount).Error)
	assert.Zero(t, userCount)
}

func TestWeChatVerifiedRegistrationCreatesAffiliateLedgerInProofTransaction(t *testing.T) {
	db := setupAffiliateRegistrationControllerTest(t)
	common.WeChatRegistrationVerificationEnabled = true
	common.WeChatAuthEnabled = true
	common.WeChatAppId = "wx-affiliate-test"
	common.WeChatAppSecret = "wx-affiliate-secret"
	inviter := createAffiliateRegistrationInviter(t, db, "wechat-inviter", "wechat-old")
	proofToken, _, err := createWeChatRegistrationVerification(&oauth.OAuthUser{
		ProviderUserID: "wechat-affiliate-openid",
		Extra:          map[string]any{"union_id": "wechat-affiliate-unionid"},
	})
	require.NoError(t, err)

	response := sendAffiliateRegistrationRequest(t, fmt.Sprintf(`{
		"username":"wechat-referred",
		"password":"password123",
		"aff_code":"wechat-old",
		"wechat_verification_token":%q
	}`, proofToken))
	assert.Equal(t, true, response["success"])
	var referred model.User
	require.NoError(t, db.Where("username = ?", "wechat-referred").First(&referred).Error)
	assert.Equal(t, "wechat-affiliate-openid", referred.WeChatId)
	assertAffiliateRegistrationBalances(t, db, inviter.Id, referred.Id)
}

func TestOAuthRegistrationsPersistAffiliateLedger(t *testing.T) {
	tests := []struct {
		name     string
		provider oauth.Provider
	}{
		{
			name:     "built-in provider",
			provider: &authFlowTestOAuthProvider{},
		},
		{
			name: "generic provider",
			provider: oauth.NewGenericOAuthProvider(&model.CustomOAuthProvider{
				Id:      7,
				Name:    "Generic Test",
				Slug:    "generic-test",
				Enabled: true,
			}),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := setupAffiliateRegistrationControllerTest(t)
			inviter := createAffiliateRegistrationInviter(t, db, "oauth-inviter", "oauth-old")
			referred, err := findOrCreateOAuthUser(nil, test.provider, &oauth.OAuthUser{
				ProviderUserID: "oauth-affiliate-external",
				Username:       "oauth-affiliate-referred",
				DisplayName:    "OAuth Affiliate Referred",
			}, inviter.AffCode)
			require.NoError(t, err)
			require.NotNil(t, referred)
			assertAffiliateRegistrationBalances(t, db, inviter.Id, referred.Id)
		})
	}
}
