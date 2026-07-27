package oauth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWeChatProviderCompletesWebsiteOAuthFlow(t *testing.T) {
	originalAppID := common.WeChatAppId
	originalAppSecret := common.WeChatAppSecret
	common.WeChatAppId = "wx-app-id"
	common.WeChatAppSecret = "wx-app-secret"
	t.Cleanup(func() {
		common.WeChatAppId = originalAppID
		common.WeChatAppSecret = originalAppSecret
	})

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "wx-app-id", r.URL.Query().Get("appid"))
		assert.Equal(t, "wx-app-secret", r.URL.Query().Get("secret"))
		assert.Equal(t, "authorization-code", r.URL.Query().Get("code"))
		assert.Equal(t, "authorization_code", r.URL.Query().Get("grant_type"))
		_, err := w.Write([]byte(`{
			"access_token": "access-token",
			"expires_in": 7200,
			"refresh_token": "refresh-token",
			"openid": "website-openid",
			"scope": "snsapi_login",
			"unionid": "shared-unionid"
			}`))
		assert.NoError(t, err)
	})
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "access-token", r.URL.Query().Get("access_token"))
		assert.Equal(t, "website-openid", r.URL.Query().Get("openid"))
		assert.Equal(t, "zh_CN", r.URL.Query().Get("lang"))
		_, err := w.Write([]byte(`{
			"openid": "website-openid",
			"nickname": "WeChat User",
			"headimgurl": "https://example.com/avatar.png",
			"unionid": "shared-unionid"
			}`))
		assert.NoError(t, err)
	})

	provider := &WeChatProvider{
		httpClient:       server.Client(),
		tokenEndpoint:    server.URL + "/token",
		userInfoEndpoint: server.URL + "/userinfo",
	}
	token, err := provider.ExchangeToken(t.Context(), "authorization-code", nil)
	require.NoError(t, err)
	assert.Equal(t, "access-token", token.AccessToken)
	assert.Equal(t, "website-openid", token.OpenID)
	assert.Equal(t, "shared-unionid", token.UnionID)

	user, err := provider.GetUserInfo(t.Context(), token)
	require.NoError(t, err)
	assert.Equal(t, "shared-unionid", user.ProviderUserID)
	assert.Equal(t, "WeChat User", user.DisplayName)
	assert.Equal(t, "website-openid", user.Extra["openid"])
	assert.Equal(t, "shared-unionid", user.Extra["union_id"])
}

func TestWeChatProviderRejectsProviderTokenError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte(`{"errcode":40029,"errmsg":"invalid code"}`))
		assert.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	provider := &WeChatProvider{
		httpClient:    server.Client(),
		tokenEndpoint: server.URL,
	}
	token, err := provider.ExchangeToken(t.Context(), "invalid-code", nil)

	assert.Nil(t, token)
	var oauthError *OAuthError
	require.ErrorAs(t, err, &oauthError)
	assert.Equal(t, "oauth.token_failed", oauthError.MsgKey)
	assert.Contains(t, oauthError.RawError, "errcode=40029")
}

func TestWeChatProviderDoesNotExposeCredentialsInNetworkErrors(t *testing.T) {
	originalAppSecret := common.WeChatAppSecret
	common.WeChatAppSecret = "should-not-leak"
	t.Cleanup(func() {
		common.WeChatAppSecret = originalAppSecret
	})

	server := httptest.NewServer(http.NotFoundHandler())
	client := server.Client()
	endpoint := server.URL
	server.Close()

	provider := &WeChatProvider{
		httpClient:    client,
		tokenEndpoint: endpoint,
	}
	token, err := provider.ExchangeToken(t.Context(), "should-not-leak-either", nil)

	assert.Nil(t, token)
	var oauthError *OAuthError
	require.ErrorAs(t, err, &oauthError)
	assert.NotContains(t, oauthError.RawError, "should-not-leak")
	assert.NotContains(t, oauthError.RawError, endpoint)
}

func TestWeChatProviderRequiresCompleteConfiguration(t *testing.T) {
	originalEnabled := common.WeChatAuthEnabled
	originalAppID := common.WeChatAppId
	originalAppSecret := common.WeChatAppSecret
	t.Cleanup(func() {
		common.WeChatAuthEnabled = originalEnabled
		common.WeChatAppId = originalAppID
		common.WeChatAppSecret = originalAppSecret
	})

	provider := &WeChatProvider{}
	common.WeChatAuthEnabled = true
	common.WeChatAppId = "wx-app-id"
	common.WeChatAppSecret = ""
	assert.False(t, provider.IsEnabled())

	common.WeChatAppSecret = "wx-app-secret"
	assert.True(t, provider.IsEnabled())
}
