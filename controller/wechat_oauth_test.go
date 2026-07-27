package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
