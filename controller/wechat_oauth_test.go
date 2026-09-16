package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/oauth"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGetStatusAdvertisesWeChatOnlyWithCompleteConfiguration(t *testing.T) {
	originalEnabled := common.WeChatAuthEnabled
	originalAppID := common.WeChatAppId
	originalAppSecret := common.WeChatAppSecret
	originalServerAddress := common.WeChatServerAddress
	originalServerToken := common.WeChatServerToken
	originalQRCode := common.WeChatAccountQRCodeImageURL
	originalRegistrationVerification := common.WeChatRegistrationVerificationEnabled
	originalOptionMap := common.OptionMap
	common.OptionMap = map[string]string{}
	common.WeChatAuthEnabled = true
	common.WeChatAppId = "wx-app-id"
	common.WeChatAppSecret = ""
	common.WeChatServerAddress = ""
	common.WeChatServerToken = ""
	common.WeChatAccountQRCodeImageURL = ""
	common.WeChatRegistrationVerificationEnabled = true
	t.Cleanup(func() {
		common.WeChatAuthEnabled = originalEnabled
		common.WeChatAppId = originalAppID
		common.WeChatAppSecret = originalAppSecret
		common.WeChatServerAddress = originalServerAddress
		common.WeChatServerToken = originalServerToken
		common.WeChatAccountQRCodeImageURL = originalQRCode
		common.WeChatRegistrationVerificationEnabled = originalRegistrationVerification
		common.OptionMap = originalOptionMap
	})

	getStatus := func() map[string]any {
		response := httptest.NewRecorder()
		context, _ := gin.CreateTestContext(response)
		context.Request = httptest.NewRequest(http.MethodGet, "/api/status", nil)
		GetStatus(context)

		var payload struct {
			Success bool           `json:"success"`
			Data    map[string]any `json:"data"`
		}
		require.NoError(t, common.Unmarshal(response.Body.Bytes(), &payload))
		require.True(t, payload.Success)
		return payload.Data
	}

	incomplete := getStatus()
	assert.Equal(t, false, incomplete["wechat_login"])
	assert.Equal(t, false, incomplete["wechat_direct_oauth"])
	assert.Equal(t, false, incomplete["wechat_server_bridge"])
	assert.Equal(t, "wx-app-id", incomplete["wechat_app_id"])
	assert.Equal(t, true, incomplete["wechat_registration_verification"])

	common.WeChatAppSecret = "wx-app-secret"
	complete := getStatus()
	assert.Equal(t, true, complete["wechat_login"])
	assert.Equal(t, true, complete["wechat_direct_oauth"])
	assert.Equal(t, false, complete["wechat_server_bridge"])

	common.WeChatAppId = ""
	common.WeChatAppSecret = ""
	common.WeChatServerAddress = "https://wechat.example.com/base"
	common.WeChatServerToken = "server-token"
	common.WeChatAccountQRCodeImageURL = "https://wechat.example.com/qr.png"
	bridge := getStatus()
	assert.Equal(t, true, bridge["wechat_login"])
	assert.Equal(t, false, bridge["wechat_direct_oauth"])
	assert.Equal(t, true, bridge["wechat_server_bridge"])
	assert.Equal(t, "https://wechat.example.com/qr.png", bridge["wechat_qrcode"])

	common.WeChatServerToken = ""
	incompleteBridge := getStatus()
	assert.Equal(t, false, incompleteBridge["wechat_login"])
	assert.Equal(t, false, incompleteBridge["wechat_server_bridge"])
}

func TestWeChatAuthDispatchesStateBearingCallbacksToUnifiedOAuth(t *testing.T) {
	previousDB := model.DB
	db, err := gorm.Open(sqlite.Open("file:wechat_dispatch?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.AuthFlow{}))
	model.DB = db
	t.Cleanup(func() {
		model.DB = previousDB
		sqlDB, dbErr := db.DB()
		if dbErr == nil {
			_ = sqlDB.Close()
		}
	})

	previousEnabled := common.WeChatAuthEnabled
	previousAppID := common.WeChatAppId
	previousAppSecret := common.WeChatAppSecret
	previousServerAddress := common.WeChatServerAddress
	previousServerToken := common.WeChatServerToken
	common.WeChatAuthEnabled = true
	common.WeChatAppId = "wx-app-id"
	common.WeChatAppSecret = "wx-app-secret"
	// Deliberately leave the bridge incomplete. A state-bearing callback must
	// still reach the direct OAuth state validator instead of the bridge.
	common.WeChatServerAddress = ""
	common.WeChatServerToken = ""
	t.Cleanup(func() {
		common.WeChatAuthEnabled = previousEnabled
		common.WeChatAppId = previousAppID
		common.WeChatAppSecret = previousAppSecret
		common.WeChatServerAddress = previousServerAddress
		common.WeChatServerToken = previousServerToken
	})

	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/oauth/wechat?state=unknown&code=code", nil)
	WeChatAuth(context)

	assert.Equal(t, http.StatusForbidden, response.Code)
	assert.Contains(t, response.Body.String(), "State parameter is empty or mismatched")
	assert.NotContains(t, response.Body.String(), "WECHAT_SERVER_BRIDGE_NOT_CONFIGURED")
}

func TestWeChatAuthRejectsStateLessRequestsWithoutCompleteBridge(t *testing.T) {
	previousEnabled := common.WeChatAuthEnabled
	previousAppID := common.WeChatAppId
	previousAppSecret := common.WeChatAppSecret
	previousServerAddress := common.WeChatServerAddress
	previousServerToken := common.WeChatServerToken
	common.WeChatAuthEnabled = true
	common.WeChatAppId = "wx-app-id"
	common.WeChatAppSecret = "wx-app-secret"
	common.WeChatServerAddress = "https://wechat.example.com"
	common.WeChatServerToken = ""
	t.Cleanup(func() {
		common.WeChatAuthEnabled = previousEnabled
		common.WeChatAppId = previousAppID
		common.WeChatAppSecret = previousAppSecret
		common.WeChatServerAddress = previousServerAddress
		common.WeChatServerToken = previousServerToken
	})

	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/oauth/wechat?code=code", nil)
	WeChatAuth(context)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.Contains(t, response.Body.String(), "WECHAT_SERVER_BRIDGE_NOT_CONFIGURED")
}

func TestWeChatBridgeUsesBasePathAndAuthorizationToken(t *testing.T) {
	previousAddress := common.WeChatServerAddress
	previousToken := common.WeChatServerToken
	common.WeChatServerToken = "bridge-token"
	var gotPath, gotCode, gotAuthorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotCode = r.URL.Query().Get("code")
		gotAuthorization = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"success":true,"data":"wechat-user"}`))
	}))
	common.WeChatServerAddress = server.URL + "/base/"
	t.Cleanup(func() {
		server.Close()
		common.WeChatServerAddress = previousAddress
		common.WeChatServerToken = previousToken
	})

	wechatID, err := getWeChatIdByCodeWithContext(t.Context(), "code with spaces")
	require.NoError(t, err)
	assert.Equal(t, "wechat-user", wechatID)
	assert.Equal(t, "/base/api/wechat/user", gotPath)
	assert.Equal(t, "code with spaces", gotCode)
	assert.Equal(t, "bridge-token", gotAuthorization)
}

func TestWeChatBridgeRejectsNonSuccessHTTPStatus(t *testing.T) {
	previousAddress := common.WeChatServerAddress
	previousToken := common.WeChatServerToken
	server := httptest.NewServer(http.NotFoundHandler())
	common.WeChatServerAddress = server.URL
	common.WeChatServerToken = "bridge-token"
	t.Cleanup(func() {
		server.Close()
		common.WeChatServerAddress = previousAddress
		common.WeChatServerToken = previousToken
	})

	wechatID, err := getWeChatIdByCodeWithContext(t.Context(), "code")
	assert.Empty(t, wechatID)
	require.Error(t, err)
	assert.EqualError(t, err, "微信登录服务请求失败")
}

func TestLoginMethodFromContextRecognizesUnifiedWeChatRoute(t *testing.T) {
	router := gin.New()
	var method string
	router.GET("/api/oauth/:provider", func(c *gin.Context) {
		method = loginMethodFromContext(c)
		c.Status(http.StatusNoContent)
	})

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/oauth/wechat", nil))

	assert.Equal(t, http.StatusNoContent, response.Code)
	assert.Equal(t, "wechat", method)
}

func TestWeChatOAuthUnionIDCreatesDurableIdentityClaim(t *testing.T) {
	previousDB := model.DB
	db, err := gorm.Open(sqlite.Open("file:wechat_union_claim?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.ExternalIdentityClaim{}))
	model.DB = db
	t.Cleanup(func() {
		model.DB = previousDB
		sqlDB, dbErr := db.DB()
		if dbErr == nil {
			_ = sqlDB.Close()
		}
	})

	provider := &oauth.WeChatProvider{}
	oauthUser := &oauth.OAuthUser{
		ProviderUserID: "wechat-union-id",
		Extra: map[string]any{
			"union_id": "wechat-union-id",
		},
	}
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return claimWeChatUnionIDWithTx(tx, provider, oauthUser, 7)
	}))
	subject, err := model.GetExternalIdentitySubjectByUserWithTx(
		db,
		model.ExternalIdentityProviderWeChatUnionID,
		7,
	)
	require.NoError(t, err)
	assert.Equal(t, "wechat-union-id", subject)
	subject, err = model.GetExternalIdentitySubjectByUserWithTx(
		db,
		model.ExternalIdentityProviderWeChat,
		7,
	)
	require.NoError(t, err)
	assert.Equal(t, "wechat-union-id", subject)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return claimWeChatUnionIDWithTx(tx, provider, &oauth.OAuthUser{
			ProviderUserID: "openid-fallback",
			Extra:          map[string]any{"union_id": ""},
		}, 8)
	}))
	_, err = model.GetExternalIdentitySubjectByUserWithTx(
		db,
		model.ExternalIdentityProviderWeChatUnionID,
		8,
	)
	assert.ErrorIs(t, err, model.ErrExternalIdentityNotClaimed)
	subject, err = model.GetExternalIdentitySubjectByUserWithTx(
		db,
		model.ExternalIdentityProviderWeChat,
		8,
	)
	require.NoError(t, err)
	assert.Equal(t, "openid-fallback", subject)
}
