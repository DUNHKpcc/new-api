package controller

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type desktopHTTPFixture struct {
	router   *gin.Engine
	db       *gorm.DB
	user     *model.User
	session  *model.UserSession
	request  service.DesktopAuthorizationRequestInput
	verifier string
}

func setupDesktopHTTPWorkflow(t *testing.T) desktopHTTPFixture {
	t.Helper()

	gin.SetMode(gin.TestMode)
	previousDB, previousLogDB := model.DB, model.LOG_DB
	previousRedis := common.RedisEnabled
	previousSecret := common.SessionSecret
	previousMainType := common.MainDatabaseType()
	previousLogType := common.LogDatabaseType()
	previousDesktopAgentSetting := *operation_setting.GetDesktopAgentSetting()

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
		&model.UserSession{},
		&model.AuthFlow{},
		&model.Token{},
		&model.DesktopGrant{},
		&model.Ability{},
		&model.SubscriptionPlan{},
		&model.UserSubscription{},
		&model.Log{},
	))

	model.DB = db
	model.LOG_DB = db
	common.RedisEnabled = false
	common.SessionSecret = "desktop-http-regression-secret"
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	*operation_setting.GetDesktopAgentSetting() = operation_setting.DesktopAgentSetting{
		ClaudeGroup: operation_setting.DesktopAgentAutoGroup,
		CodexGroup:  operation_setting.DesktopAgentAutoGroup,
	}
	t.Setenv("DESKTOP_AUTHORIZATION_ORIGIN", "https://api.dpccgaming.xyz")

	t.Cleanup(func() {
		model.DB = previousDB
		model.LOG_DB = previousLogDB
		common.RedisEnabled = previousRedis
		common.SessionSecret = previousSecret
		common.SetDatabaseTypes(previousMainType, previousLogType)
		*operation_setting.GetDesktopAgentSetting() = previousDesktopAgentSetting
		_ = sqlDB.Close()
	})

	user := &model.User{
		Username:    "desktop-http-user",
		DisplayName: "Desktop HTTP User",
		Password:    "unused-password-hash",
		Email:       "desktop-http@example.com",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		Quota:       123456,
		AuthVersion: 1,
	}
	require.NoError(t, db.Create(user).Error)
	session := &model.UserSession{
		SID:             "desktop-http-browser-session",
		UserID:          user.Id,
		Version:         1,
		UserAuthVersion: user.AuthVersion,
		Status:          model.UserSessionStatusActive,
		RefreshHash:     "desktop-http-refresh-hash",
		LoginMethod:     "password",
		LastActiveAt:    time.Now().Unix(),
		ExpiresAt:       time.Now().Add(time.Hour).Unix(),
	}
	require.NoError(t, db.Create(session).Error)
	require.NoError(t, db.Create(&model.Ability{
		Group:     "default",
		Model:     "gpt-desktop",
		ChannelId: 1,
		Enabled:   true,
	}).Error)

	browserSession := func(c *gin.Context) {
		c.Set("id", user.Id)
		c.Set("session_id", session.SID)
		c.Set("auth_version", user.AuthVersion)
		c.Set("session_version", session.Version)
		c.Next()
	}
	router := gin.New()
	oauth := router.Group("/api/desktop/oauth")
	oauth.Use(middleware.DesktopAuthorizationSecurityHeaders())
	oauth.POST("/authorization-requests", CreateDesktopAuthorizationRequest)
	oauth.GET("/authorization-requests/:request_token", browserSession, GetDesktopAuthorizationRequest)
	oauth.POST("/authorize", browserSession, DecideDesktopAuthorization)
	oauth.POST("/token", ExchangeDesktopAuthorizationCode)
	oauth.POST("/confirm", ConfirmDesktopAuthorization)
	oauth.POST("/revoke", RevokeDesktopAuthorization)

	desktop := router.Group("/api/desktop")
	desktop.Use(middleware.DesktopAuthorizationSecurityHeaders())
	desktop.GET("/account", middleware.DesktopTokenAuth("account.read"), GetDesktopAccount)
	desktop.GET("/subscriptions", middleware.DesktopTokenAuth("account.read"), GetDesktopSubscriptions)
	desktop.GET("/usage", middleware.DesktopTokenAuth("usage.read"), GetDesktopUsage)
	desktop.GET("/usage/summary", middleware.DesktopTokenAuth("usage.read"), GetDesktopUsageSummary)

	router.GET("/api/user/desktop-grants", browserSession, ListDesktopGrants)
	router.DELETE("/api/user/desktop-grants/:public_id/history", browserSession, DeleteRevokedDesktopGrant)
	relayIdentity := func(c *gin.Context) {
		modelLimit, _ := c.Get("token_model_limit")
		c.JSON(http.StatusOK, gin.H{
			"token_name":  c.GetString("token_name"),
			"model_limit": modelLimit,
		})
	}
	router.POST("/v1/messages", middleware.TokenAuth(), relayIdentity)
	router.POST("/v1/responses", middleware.TokenAuth(), relayIdentity)

	verifier := strings.Repeat("a", 64)
	challenge := sha256.Sum256([]byte(verifier))
	return desktopHTTPFixture{
		router:  router,
		db:      db,
		user:    user,
		session: session,
		request: service.DesktopAuthorizationRequestInput{
			ClientID:            service.DesktopClientID,
			RedirectURI:         "http://127.0.0.1:49152/oauth/callback/" + base64.RawURLEncoding.EncodeToString([]byte("0123456789abcdef")),
			State:               base64.RawURLEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")),
			CodeChallenge:       base64.RawURLEncoding.EncodeToString(challenge[:]),
			CodeChallengeMethod: "S256",
			DeviceID:            "8656d280-558d-4e62-8607-3f2b2c53d4bd",
			DeviceName:          "Regression MacBook",
			Platform:            "darwin-arm64",
			AppVersion:          "2.1.6",
		},
		verifier: verifier,
	}
}

func performDesktopJSON(
	t *testing.T,
	router http.Handler,
	method string,
	target string,
	body any,
	authorization string,
) *httptest.ResponseRecorder {
	t.Helper()

	payload := []byte(nil)
	if body != nil {
		var err error
		payload, err = common.Marshal(body)
		require.NoError(t, err)
	}
	request := httptest.NewRequest(method, target, bytes.NewReader(payload))
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func decodeDesktopHTTPResponse[T any](t *testing.T, response *httptest.ResponseRecorder) T {
	t.Helper()

	var decoded T
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &decoded))
	return decoded
}

func TestDesktopAuthorizationHTTPWorkflowRegression(t *testing.T) {
	fixture := setupDesktopHTTPWorkflow(t)

	startResponse := performDesktopJSON(
		t,
		fixture.router,
		http.MethodPost,
		"/api/desktop/oauth/authorization-requests",
		fixture.request,
		"",
	)
	require.Equal(t, http.StatusCreated, startResponse.Code, startResponse.Body.String())
	assert.Equal(t, "no-store", startResponse.Header().Get("Cache-Control"))
	started := decodeDesktopHTTPResponse[service.DesktopAuthorizationRequestResult](t, startResponse)
	authorizationURL, err := url.Parse(started.AuthorizationURL)
	require.NoError(t, err)
	assert.Equal(t, "https://api.dpccgaming.xyz", authorizationURL.Scheme+"://"+authorizationURL.Host)
	assert.Equal(t, "/desktop/authorize", authorizationURL.Path)
	assert.Equal(t, started.RequestToken, authorizationURL.Query().Get("request"))
	assert.Equal(t, int(service.DesktopAuthorizationRequestTTL/time.Second), started.ExpiresIn)

	viewResponse := performDesktopJSON(
		t,
		fixture.router,
		http.MethodGet,
		"/api/desktop/oauth/authorization-requests/"+started.RequestToken,
		nil,
		"",
	)
	require.Equal(t, http.StatusOK, viewResponse.Code, viewResponse.Body.String())
	view := decodeDesktopHTTPResponse[service.DesktopAuthorizationRequestView](t, viewResponse)
	assert.Equal(t, service.DesktopClientDisplayName, view.ClientName)
	assert.Equal(t, fixture.request.DeviceName, view.DeviceName)
	assert.Equal(t, []string{"gpt-desktop"}, view.AllowedModels)
	assert.Equal(t, int64(service.DesktopAccessTokenTTL/time.Second), view.TokenTTL)

	decisionResponse := performDesktopJSON(
		t,
		fixture.router,
		http.MethodPost,
		"/api/desktop/oauth/authorize",
		map[string]string{
			"request_token": started.RequestToken,
			"decision":      "allow",
		},
		"",
	)
	require.Equal(t, http.StatusOK, decisionResponse.Code, decisionResponse.Body.String())
	decision := decodeDesktopHTTPResponse[service.DesktopAuthorizationDecisionResult](t, decisionResponse)
	callback, err := url.Parse(decision.RedirectURI)
	require.NoError(t, err)
	assert.Equal(t, fixture.request.State, callback.Query().Get("state"))
	code := callback.Query().Get("code")
	require.NotEmpty(t, code)

	exchangeResponse := performDesktopJSON(
		t,
		fixture.router,
		http.MethodPost,
		"/api/desktop/oauth/token",
		service.DesktopTokenExchangeInput{
			GrantType:    "authorization_code",
			ClientID:     service.DesktopClientID,
			Code:         code,
			RedirectURI:  fixture.request.RedirectURI,
			CodeVerifier: fixture.verifier,
			DeviceID:     fixture.request.DeviceID,
		},
		"",
	)
	require.Equal(t, http.StatusOK, exchangeResponse.Code, exchangeResponse.Body.String())
	exchanged := decodeDesktopHTTPResponse[service.DesktopTokenExchangeResult](t, exchangeResponse)
	assert.Equal(t, "Bearer", exchanged.TokenType)
	assert.Equal(t, int64(90*24*60*60), exchanged.ExpiresIn)
	assert.Equal(t, service.DesktopAuthorizationScopes, exchanged.Scope)
	assert.Equal(t, "default", exchanged.Tokens.Claude.Group)
	assert.Equal(t, "default", exchanged.Tokens.Codex.Group)
	assert.Equal(t, []string{"gpt-desktop"}, exchanged.Tokens.Claude.AllowedModels)
	assert.Equal(t, []string{"gpt-desktop"}, exchanged.Tokens.Codex.AllowedModels)
	assert.NotEqual(t, exchanged.Tokens.Claude.AccessToken, exchanged.Tokens.Codex.AccessToken)
	assert.Equal(t, fixture.request.DeviceName, exchanged.Account.DeviceName)

	var tokens []model.Token
	require.NoError(t, fixture.db.Where("user_id = ?", fixture.user.Id).Order("id").Find(&tokens).Error)
	require.Len(t, tokens, 2)
	assert.Equal(t, "PCC Agent Claude - Regression MacBook", tokens[0].Name)
	assert.Equal(t, "PCC Agent Codex - Regression MacBook", tokens[1].Name)
	for _, token := range tokens {
		assert.True(t, token.ModelLimitsEnabled)
		assert.True(t, token.UnlimitedQuota)
		assert.WithinDuration(
			t,
			time.Now().Add(service.DesktopAccessTokenTTL),
			time.Unix(token.ExpiredTime, 0),
			2*time.Second,
		)
	}

	claudeRelayRequest := httptest.NewRequest(
		http.MethodPost,
		"/v1/messages",
		bytes.NewBufferString(`{"model":"gpt-desktop","messages":[]}`),
	)
	claudeRelayRequest.Header.Set("Content-Type", "application/json")
	claudeRelayRequest.Header.Set("x-api-key", exchanged.Tokens.Claude.AccessToken)
	claudeRelayResponse := httptest.NewRecorder()
	fixture.router.ServeHTTP(claudeRelayResponse, claudeRelayRequest)
	require.Equal(t, http.StatusOK, claudeRelayResponse.Code, claudeRelayResponse.Body.String())
	var claudeRelayIdentity struct {
		TokenName  string          `json:"token_name"`
		ModelLimit map[string]bool `json:"model_limit"`
	}
	require.NoError(t, common.Unmarshal(claudeRelayResponse.Body.Bytes(), &claudeRelayIdentity))
	assert.Equal(t, "PCC Agent Claude - Regression MacBook", claudeRelayIdentity.TokenName)
	assert.Equal(t, map[string]bool{"gpt-desktop": true}, claudeRelayIdentity.ModelLimit)

	codexRelayResponse := performDesktopJSON(
		t,
		fixture.router,
		http.MethodPost,
		"/v1/responses",
		map[string]any{"model": "gpt-desktop", "input": "regression"},
		"Bearer "+exchanged.Tokens.Codex.AccessToken,
	)
	require.Equal(t, http.StatusOK, codexRelayResponse.Code, codexRelayResponse.Body.String())
	var codexRelayIdentity struct {
		TokenName  string          `json:"token_name"`
		ModelLimit map[string]bool `json:"model_limit"`
	}
	require.NoError(t, common.Unmarshal(codexRelayResponse.Body.Bytes(), &codexRelayIdentity))
	assert.Equal(t, "PCC Agent Codex - Regression MacBook", codexRelayIdentity.TokenName)
	assert.Equal(t, map[string]bool{"gpt-desktop": true}, codexRelayIdentity.ModelLimit)

	require.NoError(t, fixture.db.Create(&model.Log{
		UserId:           fixture.user.Id,
		CreatedAt:        time.Now().Unix(),
		Type:             model.LogTypeConsume,
		ModelName:        "gpt-desktop",
		Quota:            50,
		PromptTokens:     12,
		CompletionTokens: 8,
		UseTime:          3,
		TokenId:          tokens[0].Id,
		Group:            "default",
		RequestId:        "desktop-regression-request",
	}).Error)

	accountResponse := performDesktopJSON(
		t,
		fixture.router,
		http.MethodGet,
		"/api/desktop/account",
		nil,
		"Bearer "+exchanged.Tokens.Claude.AccessToken,
	)
	require.Equal(t, http.StatusOK, accountResponse.Code, accountResponse.Body.String())
	account := decodeDesktopHTTPResponse[service.DesktopAccount](t, accountResponse)
	assert.Equal(t, "Desktop HTTP User", account.DisplayName)
	assert.Equal(t, fixture.request.DeviceName, account.DeviceName)
	assert.Equal(t, []string{"gpt-desktop"}, account.AllowedModels)
	assert.Equal(t, fixture.user.Quota, account.Quota)

	plan := &model.SubscriptionPlan{
		Title:         "DPCC Pro",
		PriceAmount:   20,
		Currency:      "USD",
		DurationUnit:  model.SubscriptionDurationMonth,
		DurationValue: 1,
		TotalAmount:   2_000_000,
	}
	require.NoError(t, fixture.db.Create(plan).Error)
	now := time.Now()
	require.NoError(t, fixture.db.Create(&model.UserSubscription{
		UserId:        fixture.user.Id,
		PlanId:        plan.Id,
		AmountTotal:   2_000_000,
		AmountUsed:    500_000,
		StartTime:     now.Add(-time.Hour).Unix(),
		EndTime:       now.Add(30 * 24 * time.Hour).Unix(),
		Status:        "active",
		NextResetTime: now.Add(24 * time.Hour).Unix(),
	}).Error)
	require.NoError(t, fixture.db.Create(&model.UserSubscription{
		UserId:      fixture.user.Id,
		PlanId:      plan.Id,
		AmountTotal: 1_000_000,
		AmountUsed:  100_000,
		StartTime:   now.Add(-60 * 24 * time.Hour).Unix(),
		EndTime:     now.Add(-30 * 24 * time.Hour).Unix(),
		Status:      "expired",
	}).Error)

	subscriptionsResponse := performDesktopJSON(
		t,
		fixture.router,
		http.MethodGet,
		"/api/desktop/subscriptions",
		nil,
		"Bearer "+exchanged.Tokens.Claude.AccessToken,
	)
	require.Equal(t, http.StatusOK, subscriptionsResponse.Code, subscriptionsResponse.Body.String())
	subscriptions := decodeDesktopHTTPResponse[service.DesktopSubscriptions](t, subscriptionsResponse)
	assert.Equal(t, service.DesktopContractVersion, subscriptions.ContractVersion)
	require.Len(t, subscriptions.Subscriptions, 1)
	assert.Equal(t, "DPCC Pro", subscriptions.Subscriptions[0].PlanTitle)
	assert.EqualValues(t, 2_000_000, subscriptions.Subscriptions[0].AmountTotal)
	assert.EqualValues(t, 500_000, subscriptions.Subscriptions[0].AmountUsed)
	assert.EqualValues(t, 1_500_000, subscriptions.Subscriptions[0].AmountRemaining)
	assert.False(t, subscriptions.Subscriptions[0].Unlimited)

	usageResponse := performDesktopJSON(
		t,
		fixture.router,
		http.MethodGet,
		"/api/desktop/usage?page=1&page_size=100",
		nil,
		"Bearer "+exchanged.Tokens.Codex.AccessToken,
	)
	require.Equal(t, http.StatusOK, usageResponse.Code, usageResponse.Body.String())
	usage := decodeDesktopHTTPResponse[service.DesktopUsagePage](t, usageResponse)
	assert.Equal(t, 1, usage.Page)
	assert.Equal(t, 100, usage.PageSize)
	assert.EqualValues(t, 1, usage.Total)
	require.Len(t, usage.Items, 1)
	assert.Equal(t, model.LogTypeConsume, usage.Items[0].Type)
	assert.Equal(t, 12, usage.Items[0].PromptTokens)
	assert.Equal(t, 8, usage.Items[0].CompletionTokens)

	summaryResponse := performDesktopJSON(
		t,
		fixture.router,
		http.MethodGet,
		"/api/desktop/usage/summary",
		nil,
		"Bearer "+exchanged.Tokens.Codex.AccessToken,
	)
	require.Equal(t, http.StatusOK, summaryResponse.Code, summaryResponse.Body.String())
	summary := decodeDesktopHTTPResponse[service.DesktopUsageSummary](t, summaryResponse)
	assert.Equal(t, service.DesktopContractVersion, summary.ContractVersion)
	assert.Equal(t, int64(1), summary.Totals.RequestCount)
	assert.Equal(t, int64(20), summary.Totals.PromptTokens+summary.Totals.CompletionTokens)
	assert.False(t, summary.Truncated)

	grantsResponse := performDesktopJSON(
		t,
		fixture.router,
		http.MethodGet,
		"/api/user/desktop-grants",
		nil,
		"",
	)
	require.Equal(t, http.StatusOK, grantsResponse.Code, grantsResponse.Body.String())
	var grantsEnvelope struct {
		Success bool                 `json:"success"`
		Data    []model.DesktopGrant `json:"data"`
	}
	require.NoError(t, common.Unmarshal(grantsResponse.Body.Bytes(), &grantsEnvelope))
	require.True(t, grantsEnvelope.Success)
	require.Len(t, grantsEnvelope.Data, 1)
	assert.Equal(t, model.DesktopGrantStatusActive, grantsEnvelope.Data[0].Status)

	deleteActiveResponse := performDesktopJSON(
		t,
		fixture.router,
		http.MethodDelete,
		"/api/user/desktop-grants/"+grantsEnvelope.Data[0].PublicId+"/history",
		nil,
		"",
	)
	assert.Equal(t, http.StatusConflict, deleteActiveResponse.Code)

	revokeResponse := performDesktopJSON(
		t,
		fixture.router,
		http.MethodPost,
		"/api/desktop/oauth/revoke",
		map[string]any{},
		"Bearer "+exchanged.Tokens.Claude.AccessToken,
	)
	require.Equal(t, http.StatusNoContent, revokeResponse.Code, revokeResponse.Body.String())

	revokedAccountResponse := performDesktopJSON(
		t,
		fixture.router,
		http.MethodGet,
		"/api/desktop/account",
		nil,
		"Bearer "+exchanged.Tokens.Codex.AccessToken,
	)
	assert.Equal(t, http.StatusUnauthorized, revokedAccountResponse.Code)

	var revokedTokens []model.Token
	require.NoError(t, fixture.db.Unscoped().Where("user_id = ?", fixture.user.Id).Order("id").Find(&revokedTokens).Error)
	require.Len(t, revokedTokens, 2)
	assert.Equal(t, common.TokenStatusDisabled, revokedTokens[0].Status)
	assert.Equal(t, common.TokenStatusDisabled, revokedTokens[1].Status)

	deleteRevokedResponse := performDesktopJSON(
		t,
		fixture.router,
		http.MethodDelete,
		"/api/user/desktop-grants/"+grantsEnvelope.Data[0].PublicId+"/history",
		nil,
		"",
	)
	require.Equal(t, http.StatusOK, deleteRevokedResponse.Code, deleteRevokedResponse.Body.String())

	deletedGrantsResponse := performDesktopJSON(
		t,
		fixture.router,
		http.MethodGet,
		"/api/user/desktop-grants",
		nil,
		"",
	)
	require.Equal(t, http.StatusOK, deletedGrantsResponse.Code, deletedGrantsResponse.Body.String())
	var deletedGrantsEnvelope struct {
		Success bool                 `json:"success"`
		Data    []model.DesktopGrant `json:"data"`
	}
	require.NoError(t, common.Unmarshal(deletedGrantsResponse.Body.Bytes(), &deletedGrantsEnvelope))
	require.True(t, deletedGrantsEnvelope.Success)
	assert.Empty(t, deletedGrantsEnvelope.Data)

	var remainingTokens int64
	require.NoError(t, fixture.db.Model(&model.Token{}).
		Where("user_id = ?", fixture.user.Id).
		Count(&remainingTokens).Error)
	assert.Zero(t, remainingTokens)
}

func TestDesktopAuthorizationHTTPProtocolV2Confirmation(t *testing.T) {
	fixture := setupDesktopHTTPWorkflow(t)
	startResponse := performDesktopJSON(
		t,
		fixture.router,
		http.MethodPost,
		"/api/desktop/oauth/authorization-requests",
		fixture.request,
		"",
	)
	require.Equal(t, http.StatusCreated, startResponse.Code, startResponse.Body.String())
	started := decodeDesktopHTTPResponse[service.DesktopAuthorizationRequestResult](t, startResponse)

	decisionResponse := performDesktopJSON(
		t,
		fixture.router,
		http.MethodPost,
		"/api/desktop/oauth/authorize",
		map[string]string{
			"request_token": started.RequestToken,
			"decision":      "allow",
		},
		"",
	)
	require.Equal(t, http.StatusOK, decisionResponse.Code, decisionResponse.Body.String())
	decision := decodeDesktopHTTPResponse[service.DesktopAuthorizationDecisionResult](t, decisionResponse)
	callback, err := url.Parse(decision.RedirectURI)
	require.NoError(t, err)
	protocolVersion := service.DesktopContractVersion
	exchangeResponse := performDesktopJSON(
		t,
		fixture.router,
		http.MethodPost,
		"/api/desktop/oauth/token",
		service.DesktopTokenExchangeInput{
			GrantType:       "authorization_code",
			ClientID:        service.DesktopClientID,
			Code:            callback.Query().Get("code"),
			RedirectURI:     fixture.request.RedirectURI,
			CodeVerifier:    fixture.verifier,
			DeviceID:        fixture.request.DeviceID,
			ProtocolVersion: &protocolVersion,
		},
		"",
	)
	require.Equal(t, http.StatusOK, exchangeResponse.Code, exchangeResponse.Body.String())
	exchanged := decodeDesktopHTTPResponse[service.DesktopTokenExchangeResult](t, exchangeResponse)
	assert.True(t, exchanged.ConfirmationRequired)
	require.NotEmpty(t, exchanged.ConfirmationToken)

	pendingGrantsResponse := performDesktopJSON(
		t,
		fixture.router,
		http.MethodGet,
		"/api/user/desktop-grants",
		nil,
		"",
	)
	require.Equal(t, http.StatusOK, pendingGrantsResponse.Code, pendingGrantsResponse.Body.String())
	var pendingGrantsEnvelope struct {
		Success bool                 `json:"success"`
		Data    []model.DesktopGrant `json:"data"`
	}
	require.NoError(t, common.Unmarshal(pendingGrantsResponse.Body.Bytes(), &pendingGrantsEnvelope))
	require.True(t, pendingGrantsEnvelope.Success)
	assert.Empty(t, pendingGrantsEnvelope.Data,
		"unconfirmed grants must remain hidden from the legacy authorized-device list")

	beforeConfirmation := performDesktopJSON(
		t,
		fixture.router,
		http.MethodGet,
		"/api/desktop/account",
		nil,
		"Bearer "+exchanged.Tokens.Claude.AccessToken,
	)
	assert.Equal(t, http.StatusUnauthorized, beforeConfirmation.Code)

	for range 2 {
		confirmationResponse := performDesktopJSON(
			t,
			fixture.router,
			http.MethodPost,
			"/api/desktop/oauth/confirm",
			map[string]string{"confirmation_token": exchanged.ConfirmationToken},
			"",
		)
		require.Equal(t, http.StatusNoContent, confirmationResponse.Code, confirmationResponse.Body.String())
	}

	afterConfirmation := performDesktopJSON(
		t,
		fixture.router,
		http.MethodGet,
		"/api/desktop/account",
		nil,
		"Bearer "+exchanged.Tokens.Claude.AccessToken,
	)
	assert.Equal(t, http.StatusOK, afterConfirmation.Code, afterConfirmation.Body.String())
}
