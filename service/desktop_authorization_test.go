package service

import (
	"crypto/sha256"
	"encoding/base64"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type desktopAuthorizationFixture struct {
	user       *model.User
	session    *model.UserSession
	request    DesktopAuthorizationRequestInput
	verifier   string
	startQuota int
}

func setupDesktopAuthorizationTest(t *testing.T) desktopAuthorizationFixture {
	t.Helper()
	previousDB, previousLogDB := model.DB, model.LOG_DB
	previousRedis := common.RedisEnabled
	previousSecret := common.SessionSecret
	previousMainType := common.MainDatabaseType()
	previousLogType := common.LogDatabaseType()
	previousDesktopAgentSetting := *operation_setting.GetDesktopAgentSetting()
	*operation_setting.GetDesktopAgentSetting() = operation_setting.DesktopAgentSetting{
		ClaudeGroup: operation_setting.DesktopAgentAutoGroup,
		CodexGroup:  operation_setting.DesktopAgentAutoGroup,
	}

	db, err := gorm.Open(sqlite.Open("file:"+strings.ReplaceAll(t.Name(), "/", "_")+"?mode=memory&cache=shared"), &gorm.Config{})
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
		&model.UserSubscription{},
		&model.Log{},
	))
	model.DB = db
	model.LOG_DB = db
	common.RedisEnabled = false
	common.SessionSecret = "desktop-authorization-test-secret"
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)

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
		Username:    "desktop-user",
		DisplayName: "Desktop User",
		Password:    "unused-password-hash",
		Email:       "desktop@example.com",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		Quota:       123456,
		AuthVersion: 1,
	}
	require.NoError(t, db.Create(user).Error)
	session := &model.UserSession{
		SID:             "desktop-browser-session",
		UserID:          user.Id,
		Version:         1,
		UserAuthVersion: user.AuthVersion,
		Status:          model.UserSessionStatusActive,
		RefreshHash:     "desktop-refresh-hash",
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

	verifier := strings.Repeat("a", 64)
	challenge := sha256.Sum256([]byte(verifier))
	request := DesktopAuthorizationRequestInput{
		ClientID:            DesktopClientID,
		RedirectURI:         "http://127.0.0.1:49152/oauth/callback/" + base64.RawURLEncoding.EncodeToString([]byte("0123456789abcdef")),
		State:               base64.RawURLEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")),
		CodeChallenge:       base64.RawURLEncoding.EncodeToString(challenge[:]),
		CodeChallengeMethod: "S256",
		DeviceID:            "8656d280-558d-4e62-8607-3f2b2c53d4bd",
		DeviceName:          "Alice MacBook",
		Platform:            "darwin-arm64",
		AppVersion:          "1.2.3",
	}
	return desktopAuthorizationFixture{
		user:       user,
		session:    session,
		request:    request,
		verifier:   verifier,
		startQuota: user.Quota,
	}
}

func authorizeDesktopFixture(t *testing.T, fixture desktopAuthorizationFixture) (string, string) {
	t.Helper()
	started, err := CreateDesktopAuthorizationRequest(fixture.request, "https://dpcc.example")
	require.NoError(t, err)
	assert.Contains(t, started.AuthorizationURL, "/desktop/authorize?request=")

	view, err := GetDesktopAuthorizationRequest(started.RequestToken, fixture.user.Id)
	require.NoError(t, err)
	assert.Equal(t, fixture.request.DeviceName, view.DeviceName)
	assert.Equal(t, []string{"gpt-desktop"}, view.AllowedModels)

	decision, err := DecideDesktopAuthorization(DesktopAuthorizationDecisionInput{
		RequestToken: started.RequestToken,
		Decision:     "allow",
		UserID:       fixture.user.Id,
		SessionID:    fixture.session.SID,
	})
	require.NoError(t, err)
	callback, err := url.Parse(decision.RedirectURI)
	require.NoError(t, err)
	assert.Equal(t, fixture.request.State, callback.Query().Get("state"))
	require.NotEmpty(t, callback.Query().Get("code"))
	return callback.Query().Get("code"), started.RequestToken
}

func TestDesktopRedirectValidationAcceptsOnlyStrictLoopbackCallbacks(t *testing.T) {
	validNonce := base64.RawURLEncoding.EncodeToString([]byte("0123456789abcdef"))
	valid := "http://127.0.0.1:49152/oauth/callback/" + validNonce
	require.NoError(t, validateDesktopRedirectURI(valid))

	invalid := []string{
		"https://127.0.0.1:49152/oauth/callback/" + validNonce,
		"http://localhost:49152/oauth/callback/" + validNonce,
		"http://127.0.0.2:49152/oauth/callback/" + validNonce,
		"http://user@127.0.0.1:49152/oauth/callback/" + validNonce,
		"http://127.0.0.1:80/oauth/callback/" + validNonce,
		"http://127.0.0.1:49152/oauth/callback/short",
		"http://127.0.0.1:49152/oauth/callback/" + validNonce + "?code=injected",
		"http://127.0.0.1:49152/oauth/callback/" + validNonce + "#fragment",
		"http://127.0.0.1:49152/oauth/callback%2F" + validNonce,
	}
	for _, value := range invalid {
		t.Run(value, func(t *testing.T) {
			assert.ErrorIs(t, validateDesktopRedirectURI(value), ErrDesktopInvalidRedirect)
		})
	}
}

func TestDesktopAuthorizationCreatesTwoRestrictedTokensOnceAndRevokesBoth(t *testing.T) {
	fixture := setupDesktopAuthorizationTest(t)
	code, requestToken := authorizeDesktopFixture(t, fixture)

	result, err := ExchangeDesktopAuthorizationCode(DesktopTokenExchangeInput{
		GrantType:    "authorization_code",
		ClientID:     DesktopClientID,
		Code:         code,
		RedirectURI:  fixture.request.RedirectURI,
		CodeVerifier: fixture.verifier,
		DeviceID:     fixture.request.DeviceID,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(90*24*60*60), result.ExpiresIn)
	assert.True(t, strings.HasPrefix(result.Tokens.Claude.AccessToken, "sk-"))
	assert.True(t, strings.HasPrefix(result.Tokens.Codex.AccessToken, "sk-"))
	assert.NotEqual(t, result.Tokens.Claude.AccessToken, result.Tokens.Codex.AccessToken)
	assert.Equal(t, DesktopAuthorizationScopes, result.Scope)
	assert.Equal(t, []string{"gpt-desktop"}, result.Account.AllowedModels)
	assert.Equal(t, fixture.startQuota, result.Account.Quota)
	assert.Equal(t, "Alice MacBook", result.Account.DeviceName)

	var storedUser model.User
	require.NoError(t, model.DB.First(&storedUser, fixture.user.Id).Error)
	assert.Equal(t, fixture.startQuota, storedUser.Quota, "authorization must not credit or charge the user")

	var storedTokens []model.Token
	require.NoError(t, model.DB.Where("user_id = ?", fixture.user.Id).Order("id").Find(&storedTokens).Error)
	require.Len(t, storedTokens, 2)
	for _, storedToken := range storedTokens {
		assert.True(t, storedToken.ModelLimitsEnabled)
		assert.Equal(t, "gpt-desktop", storedToken.ModelLimits)
		assert.True(t, storedToken.UnlimitedQuota)
		assert.Equal(t, "default", storedToken.Group)
		assert.WithinDuration(t, time.Now().Add(DesktopAccessTokenTTL), time.Unix(storedToken.ExpiredTime, 0), 2*time.Second)
	}
	lastUsedTime := time.Now().Add(time.Minute).Unix()
	require.NoError(t, model.DB.Model(&storedTokens[1]).Update("accessed_time", lastUsedTime).Error)
	grants, err := model.ListDesktopGrants(fixture.user.Id)
	require.NoError(t, err)
	require.Len(t, grants, 1)
	assert.Equal(t, lastUsedTime, grants[0].LastUsedTime)

	access, err := AuthenticateDesktopAccessToken(result.Tokens.Codex.AccessToken, "account.read")
	require.NoError(t, err)
	assert.Equal(t, model.DesktopGrantStatusActive, access.Grant.Status)
	assert.Equal(t, fixture.user.Id, access.User.Id)

	_, err = ExchangeDesktopAuthorizationCode(DesktopTokenExchangeInput{
		GrantType:    "authorization_code",
		ClientID:     DesktopClientID,
		Code:         code,
		RedirectURI:  fixture.request.RedirectURI,
		CodeVerifier: fixture.verifier,
		DeviceID:     fixture.request.DeviceID,
	})
	assert.ErrorIs(t, err, ErrDesktopRequestConsumed)
	_, err = GetDesktopAuthorizationRequest(requestToken, fixture.user.Id)
	assert.ErrorIs(t, err, ErrDesktopRequestConsumed)

	require.NoError(t, RevokeDesktopAccessToken(result.Tokens.Claude.AccessToken))
	require.NoError(t, RevokeDesktopAccessToken(result.Tokens.Claude.AccessToken), "revocation must be idempotent")
	_, err = AuthenticateDesktopAccessToken(result.Tokens.Claude.AccessToken, "account.read")
	assert.ErrorIs(t, err, ErrDesktopTokenInvalid)
	_, err = AuthenticateDesktopAccessToken(result.Tokens.Codex.AccessToken, "account.read")
	assert.ErrorIs(t, err, ErrDesktopTokenInvalid)
}

func TestDesktopAuthorizationWrongVerifierDoesNotConsumeCode(t *testing.T) {
	fixture := setupDesktopAuthorizationTest(t)
	code, _ := authorizeDesktopFixture(t, fixture)

	input := DesktopTokenExchangeInput{
		GrantType:    "authorization_code",
		ClientID:     DesktopClientID,
		Code:         code,
		RedirectURI:  fixture.request.RedirectURI,
		CodeVerifier: strings.Repeat("b", 64),
		DeviceID:     fixture.request.DeviceID,
	}
	_, err := ExchangeDesktopAuthorizationCode(input)
	assert.ErrorIs(t, err, ErrDesktopInvalidPKCE)

	input.CodeVerifier = fixture.verifier
	_, err = ExchangeDesktopAuthorizationCode(input)
	require.NoError(t, err)
}

func TestDesktopAuthorizationRequiresTheOriginalBrowserSessionAtExchange(t *testing.T) {
	fixture := setupDesktopAuthorizationTest(t)
	code, _ := authorizeDesktopFixture(t, fixture)
	revoked, err := model.RevokeUserSession(fixture.user.Id, fixture.session.SID, "test_revoked")
	require.NoError(t, err)
	require.True(t, revoked)

	_, err = ExchangeDesktopAuthorizationCode(DesktopTokenExchangeInput{
		GrantType:    "authorization_code",
		ClientID:     DesktopClientID,
		Code:         code,
		RedirectURI:  fixture.request.RedirectURI,
		CodeVerifier: fixture.verifier,
		DeviceID:     fixture.request.DeviceID,
	})
	assert.ErrorIs(t, err, ErrDesktopBrowserSession)

	var tokenCount int64
	require.NoError(t, model.DB.Model(&model.Token{}).Count(&tokenCount).Error)
	assert.Zero(t, tokenCount)
}

func TestDesktopReauthorizationReplacesTheSameDeviceToken(t *testing.T) {
	fixture := setupDesktopAuthorizationTest(t)
	firstCode, _ := authorizeDesktopFixture(t, fixture)
	first, err := ExchangeDesktopAuthorizationCode(DesktopTokenExchangeInput{
		GrantType:    "authorization_code",
		ClientID:     DesktopClientID,
		Code:         firstCode,
		RedirectURI:  fixture.request.RedirectURI,
		CodeVerifier: fixture.verifier,
		DeviceID:     fixture.request.DeviceID,
	})
	require.NoError(t, err)

	secondCode, _ := authorizeDesktopFixture(t, fixture)
	second, err := ExchangeDesktopAuthorizationCode(DesktopTokenExchangeInput{
		GrantType:    "authorization_code",
		ClientID:     DesktopClientID,
		Code:         secondCode,
		RedirectURI:  fixture.request.RedirectURI,
		CodeVerifier: fixture.verifier,
		DeviceID:     fixture.request.DeviceID,
	})
	require.NoError(t, err)
	assert.NotEqual(t, first.Tokens.Claude.AccessToken, second.Tokens.Claude.AccessToken)
	assert.NotEqual(t, first.Tokens.Codex.AccessToken, second.Tokens.Codex.AccessToken)

	_, err = AuthenticateDesktopAccessToken(first.Tokens.Claude.AccessToken, "account.read")
	assert.ErrorIs(t, err, ErrDesktopTokenInvalid)
	_, err = AuthenticateDesktopAccessToken(first.Tokens.Codex.AccessToken, "account.read")
	assert.ErrorIs(t, err, ErrDesktopTokenInvalid)
	_, err = AuthenticateDesktopAccessToken(second.Tokens.Claude.AccessToken, "account.read")
	require.NoError(t, err)
	_, err = AuthenticateDesktopAccessToken(second.Tokens.Codex.AccessToken, "account.read")
	require.NoError(t, err)

	grants, err := model.ListDesktopGrants(fixture.user.Id)
	require.NoError(t, err)
	require.Len(t, grants, 2)
	assert.Equal(t, model.DesktopGrantStatusActive, grants[0].Status)
	assert.Equal(t, model.DesktopGrantStatusRevoked, grants[1].Status)
	assert.Equal(t, "reauthorized", grants[1].RevokeReason)
}

func TestDesktopDeviceLimitIsRecheckedWhenAuthorizationCodeIsExchanged(t *testing.T) {
	t.Setenv("DESKTOP_GRANT_ACTIVE_LIMIT", "1")
	firstFixture := setupDesktopAuthorizationTest(t)
	secondFixture := firstFixture
	secondFixture.request.DeviceID = "a63a0ac7-06d3-49f0-99cb-0105017d6b48"
	secondFixture.request.DeviceName = "Second device"

	firstCode, _ := authorizeDesktopFixture(t, firstFixture)
	secondCode, _ := authorizeDesktopFixture(t, secondFixture)
	_, err := ExchangeDesktopAuthorizationCode(DesktopTokenExchangeInput{
		GrantType:    "authorization_code",
		ClientID:     DesktopClientID,
		Code:         firstCode,
		RedirectURI:  firstFixture.request.RedirectURI,
		CodeVerifier: firstFixture.verifier,
		DeviceID:     firstFixture.request.DeviceID,
	})
	require.NoError(t, err)
	_, err = ExchangeDesktopAuthorizationCode(DesktopTokenExchangeInput{
		GrantType:    "authorization_code",
		ClientID:     DesktopClientID,
		Code:         secondCode,
		RedirectURI:  secondFixture.request.RedirectURI,
		CodeVerifier: secondFixture.verifier,
		DeviceID:     secondFixture.request.DeviceID,
	})
	assert.ErrorIs(t, err, ErrDesktopDeviceLimit)

	var tokenCount int64
	require.NoError(t, model.DB.Model(&model.Token{}).Count(&tokenCount).Error)
	assert.EqualValues(t, 2, tokenCount)
}

func TestDesktopAuthorizationDenialCreatesNoGrantOrToken(t *testing.T) {
	fixture := setupDesktopAuthorizationTest(t)
	started, err := CreateDesktopAuthorizationRequest(fixture.request, "https://dpcc.example")
	require.NoError(t, err)
	decision, err := DecideDesktopAuthorization(DesktopAuthorizationDecisionInput{
		RequestToken: started.RequestToken,
		Decision:     "deny",
		UserID:       fixture.user.Id,
		SessionID:    fixture.session.SID,
	})
	require.NoError(t, err)
	callback, err := url.Parse(decision.RedirectURI)
	require.NoError(t, err)
	assert.Equal(t, "access_denied", callback.Query().Get("error"))
	assert.Equal(t, fixture.request.State, callback.Query().Get("state"))

	var grantCount, tokenCount int64
	require.NoError(t, model.DB.Model(&model.DesktopGrant{}).Count(&grantCount).Error)
	require.NoError(t, model.DB.Model(&model.Token{}).Count(&tokenCount).Error)
	assert.Zero(t, grantCount)
	assert.Zero(t, tokenCount)
}

func TestDesktopTokensAreReadOnlyInGenericTokenManagement(t *testing.T) {
	fixture := setupDesktopAuthorizationTest(t)
	code, _ := authorizeDesktopFixture(t, fixture)
	result, err := ExchangeDesktopAuthorizationCode(DesktopTokenExchangeInput{
		GrantType:    "authorization_code",
		ClientID:     DesktopClientID,
		Code:         code,
		RedirectURI:  fixture.request.RedirectURI,
		CodeVerifier: fixture.verifier,
		DeviceID:     fixture.request.DeviceID,
	})
	require.NoError(t, err)

	regular := &model.Token{
		UserId:      fixture.user.Id,
		Key:         "regular-token-key",
		Status:      common.TokenStatusEnabled,
		Name:        "regular",
		ExpiredTime: -1,
	}
	require.NoError(t, model.DB.Create(regular).Error)
	tokens, err := model.GetAllUserTokens(fixture.user.Id, 0, 20)
	require.NoError(t, err)
	require.Len(t, tokens, 3)
	engines := make(map[string]bool)
	for _, token := range tokens {
		if token.PccAgent {
			engines[token.PccAgentEngine] = true
		}
	}
	assert.Equal(t, map[string]bool{"claude": true, "codex": true}, engines)
	manualCount, err := model.CountUserTokens(fixture.user.Id)
	require.NoError(t, err)
	assert.EqualValues(t, 1, manualCount)
	allCount, err := model.CountAllUserTokens(fixture.user.Id)
	require.NoError(t, err)
	assert.EqualValues(t, 3, allCount)

	desktopAccess, err := AuthenticateDesktopAccessToken(result.Tokens.Claude.AccessToken, "account.read")
	require.NoError(t, err)
	managedToken, err := model.GetTokenByIds(desktopAccess.Token.Id, fixture.user.Id)
	require.NoError(t, err)
	assert.True(t, managedToken.PccAgent)
	assert.Equal(t, "claude", managedToken.PccAgentEngine)
	err = model.DeleteTokenById(desktopAccess.Token.Id, fixture.user.Id)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	_, err = AuthenticateDesktopAccessToken(result.Tokens.Claude.AccessToken, "account.read")
	require.NoError(t, err)
}

func TestDesktopAuthorizationUsesAdminConfiguredEngineGroupsOnlyForGeneratedTokens(t *testing.T) {
	fixture := setupDesktopAuthorizationTest(t)
	*operation_setting.GetDesktopAgentSetting() = operation_setting.DesktopAgentSetting{
		ClaudeGroup: "default",
		CodexGroup:  "vip",
	}
	require.NoError(t, model.DB.Create(&model.Ability{
		Group:     "vip",
		Model:     "gpt-codex",
		ChannelId: 2,
		Enabled:   true,
	}).Error)

	started, err := CreateDesktopAuthorizationRequest(fixture.request, "https://dpcc.example")
	require.NoError(t, err)
	view, err := GetDesktopAuthorizationRequest(started.RequestToken, fixture.user.Id)
	require.NoError(t, err)
	assert.Equal(t, []string{"gpt-codex", "gpt-desktop"}, view.AllowedModels)

	decision, err := DecideDesktopAuthorization(DesktopAuthorizationDecisionInput{
		RequestToken: started.RequestToken,
		Decision:     "allow",
		UserID:       fixture.user.Id,
		SessionID:    fixture.session.SID,
	})
	require.NoError(t, err)
	callback, err := url.Parse(decision.RedirectURI)
	require.NoError(t, err)
	result, err := ExchangeDesktopAuthorizationCode(DesktopTokenExchangeInput{
		GrantType:    "authorization_code",
		ClientID:     DesktopClientID,
		Code:         callback.Query().Get("code"),
		RedirectURI:  fixture.request.RedirectURI,
		CodeVerifier: fixture.verifier,
		DeviceID:     fixture.request.DeviceID,
	})
	require.NoError(t, err)

	assert.Equal(t, "default", result.Tokens.Claude.Group)
	assert.Equal(t, []string{"gpt-desktop"}, result.Tokens.Claude.AllowedModels)
	assert.Equal(t, "vip", result.Tokens.Codex.Group)
	assert.Equal(t, []string{"gpt-codex"}, result.Tokens.Codex.AllowedModels)
	assert.Equal(t, []string{"gpt-codex", "gpt-desktop"}, result.Account.AllowedModels)

	var generated []model.Token
	require.NoError(t, model.DB.Where("user_id = ?", fixture.user.Id).Order("id").Find(&generated).Error)
	require.Len(t, generated, 2)
	assert.Equal(t, "PCC Agent Claude - Alice MacBook", generated[0].Name)
	assert.Equal(t, "default", generated[0].Group)
	assert.Equal(t, "PCC Agent Codex - Alice MacBook", generated[1].Name)
	assert.Equal(t, "vip", generated[1].Group)
}

func TestDesktopAuthorizationRejectsConfiguredGroupUnavailableToUser(t *testing.T) {
	fixture := setupDesktopAuthorizationTest(t)
	*operation_setting.GetDesktopAgentSetting() = operation_setting.DesktopAgentSetting{
		ClaudeGroup: "admin-only",
		CodexGroup:  "default",
	}
	started, err := CreateDesktopAuthorizationRequest(fixture.request, "https://dpcc.example")
	require.NoError(t, err)

	_, err = GetDesktopAuthorizationRequest(started.RequestToken, fixture.user.Id)
	assert.ErrorIs(t, err, ErrDesktopGroupUnavailable)

	var tokenCount int64
	require.NoError(t, model.DB.Model(&model.Token{}).Count(&tokenCount).Error)
	assert.Zero(t, tokenCount)
}

func TestDesktopAuthorizationKeepsLegacySingleTokenGrantRevocable(t *testing.T) {
	fixture := setupDesktopAuthorizationTest(t)
	token := &model.Token{
		UserId:             fixture.user.Id,
		Key:                "legacy-desktop-token",
		Status:             common.TokenStatusEnabled,
		Name:               "PCC Agent - Legacy",
		CreatedTime:        time.Now().Unix(),
		AccessedTime:       time.Now().Unix(),
		ExpiredTime:        time.Now().Add(time.Hour).Unix(),
		UnlimitedQuota:     true,
		ModelLimitsEnabled: true,
		ModelLimits:        "gpt-desktop",
		Group:              "default",
	}
	require.NoError(t, model.DB.Create(token).Error)
	activeSlot := 1
	grant := &model.DesktopGrant{
		PublicId:     "legacy-desktop-grant",
		UserId:       fixture.user.Id,
		ClientId:     DesktopClientID,
		DeviceIdHash: "legacy-device-hash",
		DeviceName:   "Legacy device",
		TokenId:      &token.Id,
		Scopes:       DesktopAuthorizationScopes,
		Status:       model.DesktopGrantStatusActive,
		ActiveSlot:   &activeSlot,
		CreatedTime:  time.Now().Unix(),
		LastUsedTime: time.Now().Unix(),
		ExpiredTime:  token.ExpiredTime,
	}
	require.NoError(t, model.DB.Create(grant).Error)

	_, err := AuthenticateDesktopAccessToken("sk-"+token.Key, "account.read")
	require.NoError(t, err)
	require.NoError(t, RevokeDesktopAccessToken("sk-"+token.Key))
	_, err = AuthenticateDesktopAccessToken("sk-"+token.Key, "account.read")
	assert.ErrorIs(t, err, ErrDesktopTokenInvalid)
}
