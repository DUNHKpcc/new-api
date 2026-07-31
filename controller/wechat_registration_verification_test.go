package controller

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	appi18n "github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/oauth"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupWeChatRegistrationVerificationTest(t *testing.T) *gorm.DB {
	t.Helper()
	require.NoError(t, appi18n.Init())
	previousDB := model.DB
	previousType := common.MainDatabaseType()
	previousSecret := common.SessionSecret
	previousRedisEnabled := common.RedisEnabled
	db, err := gorm.Open(sqlite.Open("file:wechat_registration_verification?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.AuthFlow{},
		&model.ExternalIdentityClaim{},
	))
	model.DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	common.SessionSecret = "wechat-registration-verification-test-secret"
	common.RedisEnabled = false
	t.Cleanup(func() {
		model.DB = previousDB
		common.SetMainDatabaseType(previousType)
		common.SessionSecret = previousSecret
		common.RedisEnabled = previousRedisEnabled
		sqlDB, dbErr := db.DB()
		if dbErr == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func TestWeChatRegistrationProofIsConsumedWithUserAndIdentity(t *testing.T) {
	db := setupWeChatRegistrationVerificationTest(t)
	proofToken, _, err := createWeChatRegistrationVerification(&oauth.OAuthUser{
		ProviderUserID: "wechat-registration-union",
		Extra: map[string]any{
			"union_id": "wechat-registration-union",
		},
	})
	require.NoError(t, err)

	user := &model.User{
		Username:    "wechat-proof-user",
		Password:    "password123",
		DisplayName: "wechat-proof-user",
		Role:        common.RoleCommonUser,
	}
	require.NoError(t, insertUserWithWeChatVerification(user, proofToken, 0))

	var stored model.User
	require.NoError(t, db.First(&stored, user.Id).Error)
	assert.Equal(t, "wechat-registration-union", stored.WeChatId)
	assert.True(t, common.ValidatePasswordAndHash("password123", stored.Password))

	subject, err := model.GetExternalIdentitySubjectByUserWithTx(
		db,
		model.ExternalIdentityProviderWeChat,
		user.Id,
	)
	require.NoError(t, err)
	assert.Equal(t, "wechat-registration-union", subject)
	subject, err = model.GetExternalIdentitySubjectByUserWithTx(
		db,
		model.ExternalIdentityProviderWeChatUnionID,
		user.Id,
	)
	require.NoError(t, err)
	assert.Equal(t, "wechat-registration-union", subject)

	replayUser := &model.User{
		Username:    "wechat-proof-replay",
		Password:    "password123",
		DisplayName: "wechat-proof-replay",
		Role:        common.RoleCommonUser,
	}
	err = insertUserWithWeChatVerification(replayUser, proofToken, 0)
	assert.ErrorIs(t, err, model.ErrAuthFlowConsumed)
}

func TestRegistrationRequiresEmailCodeAndWeChatProofTogether(t *testing.T) {
	setupWeChatRegistrationVerificationTest(t)
	previousRegisterEnabled := common.RegisterEnabled
	previousPasswordRegisterEnabled := common.PasswordRegisterEnabled
	previousEmailVerificationEnabled := common.EmailVerificationEnabled
	previousWeChatVerificationEnabled := common.WeChatRegistrationVerificationEnabled
	previousWeChatAuthEnabled := common.WeChatAuthEnabled
	previousAppID := common.WeChatAppId
	previousAppSecret := common.WeChatAppSecret
	previousGenerateDefaultToken := constant.GenerateDefaultToken
	common.RegisterEnabled = true
	common.PasswordRegisterEnabled = true
	common.EmailVerificationEnabled = true
	common.WeChatRegistrationVerificationEnabled = true
	common.WeChatAuthEnabled = true
	common.WeChatAppId = "wx-test-app"
	common.WeChatAppSecret = "wx-test-secret"
	constant.GenerateDefaultToken = false
	t.Cleanup(func() {
		common.RegisterEnabled = previousRegisterEnabled
		common.PasswordRegisterEnabled = previousPasswordRegisterEnabled
		common.EmailVerificationEnabled = previousEmailVerificationEnabled
		common.WeChatRegistrationVerificationEnabled = previousWeChatVerificationEnabled
		common.WeChatAuthEnabled = previousWeChatAuthEnabled
		common.WeChatAppId = previousAppID
		common.WeChatAppSecret = previousAppSecret
		constant.GenerateDefaultToken = previousGenerateDefaultToken
		common.DeleteKey("both@example.com", common.EmailVerificationPurpose)
	})
	common.RegisterVerificationCodeWithKey(
		"both@example.com",
		"123456",
		common.EmailVerificationPurpose,
	)

	sendRegistration := func(body string) map[string]any {
		recorder := httptest.NewRecorder()
		context, _ := gin.CreateTestContext(recorder)
		context.Request = httptest.NewRequest(http.MethodPost, "/api/user/register", strings.NewReader(body))
		context.Request.Header.Set("Content-Type", "application/json")
		Register(context)
		var response map[string]any
		require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
		return response
	}

	missingProof := sendRegistration(`{
		"username":"both-required",
		"password":"password123",
		"email":"both@example.com",
		"verification_code":"123456"
	}`)
	assert.Equal(t, false, missingProof["success"])
	assert.Equal(t, weChatRegistrationErrorRequired, missingProof["code"])

	proofToken, _, err := createWeChatRegistrationVerification(&oauth.OAuthUser{
		ProviderUserID: "both-required-wechat",
		Extra:          map[string]any{"union_id": "both-required-wechat"},
	})
	require.NoError(t, err)
	invalidEmailCode := sendRegistration(`{
		"username":"both-required",
		"password":"password123",
		"email":"both@example.com",
		"verification_code":"000000",
		"wechat_verification_token":"` + proofToken + `"
	}`)
	assert.Equal(t, false, invalidEmailCode["success"])
	var userCount int64
	require.NoError(t, model.DB.Model(&model.User{}).Count(&userCount).Error)
	assert.Zero(t, userCount)
	_, err = model.GetAuthFlow(proofToken, model.AuthFlowMatch{
		Purpose:  model.AuthFlowPurposeWeChatRegistration,
		Provider: "wechat",
		Intent:   model.AuthFlowIntentRegister,
	})
	require.NoError(t, err)

	success := sendRegistration(`{
		"username":"both-required",
		"password":"password123",
		"email":"both@example.com",
		"verification_code":"123456",
		"wechat_verification_token":"` + proofToken + `"
	}`)
	assert.Equal(t, true, success["success"])
	var registeredUser model.User
	require.NoError(t, model.DB.Where("username = ?", "both-required").First(&registeredUser).Error)
	assert.Equal(t, "both@example.com", registeredUser.Email)
	assert.Equal(t, "both-required-wechat", registeredUser.WeChatId)
}

func TestGenerateOAuthCodeCreatesWeChatRegistrationFlowOnlyWhenAvailable(t *testing.T) {
	setupWeChatRegistrationVerificationTest(t)
	previousRegisterEnabled := common.RegisterEnabled
	previousPasswordRegisterEnabled := common.PasswordRegisterEnabled
	previousVerificationEnabled := common.WeChatRegistrationVerificationEnabled
	previousAuthEnabled := common.WeChatAuthEnabled
	previousAppID := common.WeChatAppId
	previousAppSecret := common.WeChatAppSecret
	common.RegisterEnabled = true
	common.PasswordRegisterEnabled = true
	common.WeChatRegistrationVerificationEnabled = true
	common.WeChatAuthEnabled = true
	common.WeChatAppId = "wx-test-app"
	common.WeChatAppSecret = "wx-test-secret"
	t.Cleanup(func() {
		common.RegisterEnabled = previousRegisterEnabled
		common.PasswordRegisterEnabled = previousPasswordRegisterEnabled
		common.WeChatRegistrationVerificationEnabled = previousVerificationEnabled
		common.WeChatAuthEnabled = previousAuthEnabled
		common.WeChatAppId = previousAppID
		common.WeChatAppSecret = previousAppSecret
	})

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/oauth/state",
		strings.NewReader(`{"provider":"wechat","intent":"register"}`),
	)
	context.Request.Header.Set("Content-Type", "application/json")
	GenerateOAuthCode(context)

	var response struct {
		Success bool   `json:"success"`
		Code    string `json:"code"`
		Data    struct {
			FlowToken string `json:"flow_token"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	_, err := model.GetAuthFlow(response.Data.FlowToken, model.AuthFlowMatch{
		Purpose:  model.AuthFlowPurposeOAuth,
		Provider: "wechat",
		Intent:   model.AuthFlowIntentRegister,
	})
	require.NoError(t, err)

	common.WeChatAuthEnabled = false
	recorder = httptest.NewRecorder()
	context, _ = gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/oauth/state",
		strings.NewReader(`{"provider":"wechat","intent":"register"}`),
	)
	context.Request.Header.Set("Content-Type", "application/json")
	GenerateOAuthCode(context)
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.False(t, response.Success)
	assert.Equal(t, weChatRegistrationErrorUnavailable, response.Code)
}

func TestOtherOAuthProvidersCannotCreateAccountsWhenWeChatVerificationIsRequired(t *testing.T) {
	setupWeChatRegistrationVerificationTest(t)
	previousEnabled := common.WeChatRegistrationVerificationEnabled
	previousRegisterEnabled := common.RegisterEnabled
	common.WeChatRegistrationVerificationEnabled = true
	common.RegisterEnabled = true
	t.Cleanup(func() {
		common.WeChatRegistrationVerificationEnabled = previousEnabled
		common.RegisterEnabled = previousRegisterEnabled
	})

	_, err := findOrCreateOAuthUser(
		&gin.Context{},
		&authFlowTestOAuthProvider{},
		&oauth.OAuthUser{ProviderUserID: "other-oauth-new-user"},
		"",
	)
	var requiredError *OAuthWeChatVerificationRequiredError
	assert.True(t, errors.As(err, &requiredError))
}

func TestWeChatRegistrationVerificationOptionGuardsDependencies(t *testing.T) {
	previousVerificationEnabled := common.WeChatRegistrationVerificationEnabled
	previousAuthEnabled := common.WeChatAuthEnabled
	previousPasswordRegisterEnabled := common.PasswordRegisterEnabled
	previousAppID := common.WeChatAppId
	previousAppSecret := common.WeChatAppSecret
	t.Cleanup(func() {
		common.WeChatRegistrationVerificationEnabled = previousVerificationEnabled
		common.WeChatAuthEnabled = previousAuthEnabled
		common.PasswordRegisterEnabled = previousPasswordRegisterEnabled
		common.WeChatAppId = previousAppID
		common.WeChatAppSecret = previousAppSecret
	})

	updateOption := func(body string) map[string]any {
		recorder := httptest.NewRecorder()
		context, _ := gin.CreateTestContext(recorder)
		context.Request = httptest.NewRequest(http.MethodPut, "/api/option/", strings.NewReader(body))
		context.Request.Header.Set("Content-Type", "application/json")
		UpdateOption(context)
		var response map[string]any
		require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
		return response
	}

	common.WeChatRegistrationVerificationEnabled = false
	common.WeChatAuthEnabled = false
	common.PasswordRegisterEnabled = true
	common.WeChatAppId = "wx-test-app"
	common.WeChatAppSecret = "wx-test-secret"
	response := updateOption(`{"key":"WeChatRegistrationVerificationEnabled","value":true}`)
	assert.Equal(t, false, response["success"])
	assert.Contains(t, response["message"], "微信 OAuth")

	common.WeChatRegistrationVerificationEnabled = true
	common.WeChatAuthEnabled = true
	response = updateOption(`{"key":"WeChatAuthEnabled","value":false}`)
	assert.Equal(t, false, response["success"])
	assert.Contains(t, response["message"], "先关闭")

	response = updateOption(`{"key":"PasswordRegisterEnabled","value":false}`)
	assert.Equal(t, false, response["success"])
	assert.Contains(t, response["message"], "先关闭")

	response = updateOption(`{"key":"WeChatAppId","value":""}`)
	assert.Equal(t, false, response["success"])
	assert.Contains(t, response["message"], "不能为空")
}
