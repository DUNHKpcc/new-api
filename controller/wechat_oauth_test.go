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
	originalOptionMap := common.OptionMap
	common.OptionMap = map[string]string{}
	common.WeChatAuthEnabled = true
	common.WeChatAppId = "wx-app-id"
	common.WeChatAppSecret = ""
	t.Cleanup(func() {
		common.WeChatAuthEnabled = originalEnabled
		common.WeChatAppId = originalAppID
		common.WeChatAppSecret = originalAppSecret
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
	assert.Equal(t, "wx-app-id", incomplete["wechat_app_id"])

	common.WeChatAppSecret = "wx-app-secret"
	complete := getStatus()
	assert.Equal(t, true, complete["wechat_login"])
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
}
