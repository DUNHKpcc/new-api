package service

import (
	"crypto/sha256"
	"encoding/base64"
	"net/url"
	"strconv"
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
		&model.ExternalIdentityClaim{},
		&model.SubscriptionPlan{},
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
	assert.Equal(t, fixture.user.UsedQuota, result.Account.UsedQuota)
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

func TestDesktopAuthorizationProtocolV2ActivatesOnlyAfterConfirmation(t *testing.T) {
	fixture := setupDesktopAuthorizationTest(t)
	code, _ := authorizeDesktopFixture(t, fixture)
	protocolVersion := DesktopContractVersion

	result, err := ExchangeDesktopAuthorizationCode(DesktopTokenExchangeInput{
		GrantType:       "authorization_code",
		ClientID:        DesktopClientID,
		Code:            code,
		RedirectURI:     fixture.request.RedirectURI,
		CodeVerifier:    fixture.verifier,
		DeviceID:        fixture.request.DeviceID,
		ProtocolVersion: &protocolVersion,
	})
	require.NoError(t, err)
	assert.Equal(t, DesktopContractVersion, result.ContractVersion)
	assert.True(t, result.ConfirmationRequired)
	assert.NotEmpty(t, result.ConfirmationToken)
	assert.Equal(t, int64(DesktopConfirmationTTL/time.Second), result.ConfirmationExpiresIn)

	var storedTokens []model.Token
	require.NoError(t, model.DB.Where("user_id = ?", fixture.user.Id).Order("id").Find(&storedTokens).Error)
	require.Len(t, storedTokens, 2)
	assert.Equal(t, common.TokenStatusDisabled, storedTokens[0].Status)
	assert.Equal(t, common.TokenStatusDisabled, storedTokens[1].Status)

	_, err = AuthenticateDesktopAccessToken(result.Tokens.Claude.AccessToken, "account.read")
	assert.ErrorIs(t, err, ErrDesktopTokenInvalid)

	require.NoError(t, ConfirmDesktopAuthorization(result.ConfirmationToken))
	require.NoError(t, ConfirmDesktopAuthorization(result.ConfirmationToken), "confirmation must be idempotent")
	require.NoError(t, model.DB.Model(&model.DesktopGrant{}).
		Where("public_id = ?", result.Account.GrantPublicID).
		Update("confirmation_expires_at", time.Now().Add(-time.Minute).Unix()).Error)
	require.NoError(t, ConfirmDesktopAuthorization(result.ConfirmationToken),
		"an already active grant remains confirmable after the staging deadline")

	access, err := AuthenticateDesktopAccessToken(result.Tokens.Claude.AccessToken, "account.read")
	require.NoError(t, err)
	assert.Equal(t, model.DesktopGrantStatusActive, access.Grant.Status)
	assert.NotZero(t, access.Grant.ConfirmedTime)

	require.NoError(t, model.DB.Where("user_id = ?", fixture.user.Id).Order("id").Find(&storedTokens).Error)
	assert.Equal(t, common.TokenStatusEnabled, storedTokens[0].Status)
	assert.Equal(t, common.TokenStatusEnabled, storedTokens[1].Status)
}

func TestDesktopAuthorizationGiftRequiresWeChatAndIsGrantedOnConfirmation(t *testing.T) {
	fixture := setupDesktopAuthorizationTest(t)
	plan := model.SubscriptionPlan{
		Title:              "PccAgent gift",
		DurationUnit:       model.SubscriptionDurationMonth,
		DurationValue:      1,
		Enabled:            true,
		MaxPurchasePerUser: 1,
		TotalAmount:        500,
		UpgradeGroup:       "pro",
	}
	require.NoError(t, model.DB.Create(&plan).Error)
	model.InvalidateSubscriptionPlanCache(plan.Id)
	operation_setting.GetDesktopAgentSetting().GiftPlanId = plan.Id

	started, err := CreateDesktopAuthorizationRequest(fixture.request, "https://dpcc.example")
	require.NoError(t, err)
	view, err := GetDesktopAuthorizationRequest(started.RequestToken, fixture.user.Id)
	require.NoError(t, err)
	assert.True(t, view.WeChatVerificationRequired)
	assert.False(t, view.WeChatVerified)

	_, err = DecideDesktopAuthorization(DesktopAuthorizationDecisionInput{
		RequestToken: started.RequestToken,
		Decision:     "allow",
		UserID:       fixture.user.Id,
		SessionID:    fixture.session.SID,
	})
	assert.ErrorIs(t, err, ErrDesktopWeChatRequired)

	require.NoError(t, model.DB.Transaction(func(tx *gorm.DB) error {
		return model.ClaimExternalIdentityWithTx(
			tx,
			model.ExternalIdentityProviderWeChatUnionID,
			"desktop-wechat-union",
			fixture.user.Id,
		)
	}))
	view, err = GetDesktopAuthorizationRequest(started.RequestToken, fixture.user.Id)
	require.NoError(t, err)
	assert.True(t, view.WeChatVerified)

	decision, err := DecideDesktopAuthorization(DesktopAuthorizationDecisionInput{
		RequestToken: started.RequestToken,
		Decision:     "allow",
		UserID:       fixture.user.Id,
		SessionID:    fixture.session.SID,
	})
	require.NoError(t, err)
	callback, err := url.Parse(decision.RedirectURI)
	require.NoError(t, err)
	code := callback.Query().Get("code")
	require.NotEmpty(t, code)

	protocolVersion := DesktopContractVersion
	exchanged, err := ExchangeDesktopAuthorizationCode(DesktopTokenExchangeInput{
		GrantType:       "authorization_code",
		ClientID:        DesktopClientID,
		Code:            code,
		RedirectURI:     fixture.request.RedirectURI,
		CodeVerifier:    fixture.verifier,
		DeviceID:        fixture.request.DeviceID,
		ProtocolVersion: &protocolVersion,
	})
	require.NoError(t, err)

	var giftCount int64
	require.NoError(t, model.DB.Model(&model.UserSubscription{}).
		Where("user_id = ? AND source = ?", fixture.user.Id, model.UserSubscriptionSourcePccAgentGift).
		Count(&giftCount).Error)
	assert.Zero(t, giftCount, "staged credentials must not grant the benefit before confirmation")

	require.NoError(t, ConfirmDesktopAuthorization(exchanged.ConfirmationToken))
	require.NoError(t, ConfirmDesktopAuthorization(exchanged.ConfirmationToken))
	require.NoError(t, model.DB.Model(&model.UserSubscription{}).
		Where("user_id = ? AND source = ?", fixture.user.Id, model.UserSubscriptionSourcePccAgentGift).
		Count(&giftCount).Error)
	assert.EqualValues(t, 1, giftCount)

	var storedUser model.User
	require.NoError(t, model.DB.First(&storedUser, fixture.user.Id).Error)
	assert.Equal(t, "default", storedUser.Group)
	permanentSubject, err := model.GetExternalIdentitySubjectByUserWithTx(
		model.DB,
		model.ExternalIdentityProviderPccAgentGiftWeChat,
		fixture.user.Id,
	)
	require.NoError(t, err)
	assert.Equal(t, "desktop-wechat-union", permanentSubject)
}

func TestDesktopAuthorizationProtocolV2KeepsPreviousDeviceCredentialUntilConfirmation(t *testing.T) {
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
	protocolVersion := DesktopContractVersion
	second, err := ExchangeDesktopAuthorizationCode(DesktopTokenExchangeInput{
		GrantType:       "authorization_code",
		ClientID:        DesktopClientID,
		Code:            secondCode,
		RedirectURI:     fixture.request.RedirectURI,
		CodeVerifier:    fixture.verifier,
		DeviceID:        fixture.request.DeviceID,
		ProtocolVersion: &protocolVersion,
	})
	require.NoError(t, err)

	_, err = AuthenticateDesktopAccessToken(first.Tokens.Claude.AccessToken, "account.read")
	require.NoError(t, err)
	_, err = AuthenticateDesktopAccessToken(second.Tokens.Claude.AccessToken, "account.read")
	assert.ErrorIs(t, err, ErrDesktopTokenInvalid)

	require.NoError(t, ConfirmDesktopAuthorization(second.ConfirmationToken))
	_, err = AuthenticateDesktopAccessToken(first.Tokens.Claude.AccessToken, "account.read")
	assert.ErrorIs(t, err, ErrDesktopTokenInvalid)
	_, err = AuthenticateDesktopAccessToken(first.Tokens.Codex.AccessToken, "account.read")
	assert.ErrorIs(t, err, ErrDesktopTokenInvalid)
	_, err = AuthenticateDesktopAccessToken(second.Tokens.Claude.AccessToken, "account.read")
	require.NoError(t, err)
	_, err = AuthenticateDesktopAccessToken(second.Tokens.Codex.AccessToken, "account.read")
	require.NoError(t, err)
}

func TestDesktopAuthorizationProtocolV2RechecksDeviceLimitAtConfirmation(t *testing.T) {
	t.Setenv("DESKTOP_GRANT_ACTIVE_LIMIT", "1")
	firstFixture := setupDesktopAuthorizationTest(t)
	secondFixture := firstFixture
	secondFixture.request.DeviceID = "a63a0ac7-06d3-49f0-99cb-0105017d6b48"
	secondFixture.request.DeviceName = "Second device"

	firstCode, _ := authorizeDesktopFixture(t, firstFixture)
	secondCode, _ := authorizeDesktopFixture(t, secondFixture)
	protocolVersion := DesktopContractVersion
	first, err := ExchangeDesktopAuthorizationCode(DesktopTokenExchangeInput{
		GrantType:       "authorization_code",
		ClientID:        DesktopClientID,
		Code:            firstCode,
		RedirectURI:     firstFixture.request.RedirectURI,
		CodeVerifier:    firstFixture.verifier,
		DeviceID:        firstFixture.request.DeviceID,
		ProtocolVersion: &protocolVersion,
	})
	require.NoError(t, err)
	second, err := ExchangeDesktopAuthorizationCode(DesktopTokenExchangeInput{
		GrantType:       "authorization_code",
		ClientID:        DesktopClientID,
		Code:            secondCode,
		RedirectURI:     secondFixture.request.RedirectURI,
		CodeVerifier:    secondFixture.verifier,
		DeviceID:        secondFixture.request.DeviceID,
		ProtocolVersion: &protocolVersion,
	})
	require.NoError(t, err)

	require.NoError(t, ConfirmDesktopAuthorization(first.ConfirmationToken))
	assert.ErrorIs(t, ConfirmDesktopAuthorization(second.ConfirmationToken), ErrDesktopDeviceLimit)
	_, err = AuthenticateDesktopAccessToken(second.Tokens.Claude.AccessToken, "account.read")
	assert.ErrorIs(t, err, ErrDesktopTokenInvalid)
}

func TestDesktopGrantStateAlsoGuardsGenericRelayAuthentication(t *testing.T) {
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

	require.NoError(t, model.DB.Model(&model.DesktopGrant{}).
		Where("user_id = ?", fixture.user.Id).
		Updates(map[string]any{
			"status":      model.DesktopGrantStatusRevoked,
			"active_slot": nil,
		}).Error)

	_, err = model.ValidateUserToken(strings.TrimPrefix(result.Tokens.Claude.AccessToken, "sk-"))
	assert.ErrorIs(t, err, model.ErrTokenInvalid)
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

func TestDesktopAuthorizationAccountReadFailureDoesNotConsumeCode(t *testing.T) {
	fixture := setupDesktopAuthorizationTest(t)
	code, _ := authorizeDesktopFixture(t, fixture)
	input := DesktopTokenExchangeInput{
		GrantType:    "authorization_code",
		ClientID:     DesktopClientID,
		Code:         code,
		RedirectURI:  fixture.request.RedirectURI,
		CodeVerifier: fixture.verifier,
		DeviceID:     fixture.request.DeviceID,
	}

	require.NoError(t, model.DB.Migrator().DropTable(&model.UserSubscription{}))
	_, err := ExchangeDesktopAuthorizationCode(input)
	require.Error(t, err)

	require.NoError(t, model.DB.AutoMigrate(&model.UserSubscription{}))
	_, err = ExchangeDesktopAuthorizationCode(input)
	require.NoError(t, err)
}

func TestDesktopUsagePreservesLegacyLogTypesAndExplicitCacheCounts(t *testing.T) {
	fixture := setupDesktopAuthorizationTest(t)
	now := time.Now().Unix()
	require.NoError(t, model.DB.Create(&model.Log{
		UserId:           fixture.user.Id,
		CreatedAt:        now,
		Type:             model.LogTypeTopup,
		PromptTokens:     100,
		CompletionTokens: 200,
		Other:            `{"cache_tokens":300}`,
	}).Error)
	require.NoError(t, model.DB.Create(&model.Log{
		UserId:           fixture.user.Id,
		CreatedAt:        now,
		Type:             model.LogTypeConsume,
		ModelName:        "legacy-model",
		Quota:            13,
		PromptTokens:     2,
		CompletionTokens: 3,
		Other:            `{"cache_tokens":5,"cache_creation_tokens":7,"cache_write_tokens":11}`,
		RequestId:        "legacy-request",
		IsStream:         true,
	}).Error)
	require.NoError(t, model.DB.Create(&model.Log{
		UserId:              fixture.user.Id,
		CreatedAt:           now - 86400,
		Type:                model.LogTypeConsume,
		ModelName:           "stable-model",
		Quota:               17,
		PromptTokens:        7,
		CompletionTokens:    9,
		CacheTokens:         19,
		CacheCreationTokens: 23,
		Other:               `{"cache_tokens":99,"cache_write_tokens":101}`,
		RequestId:           "stable-request",
	}).Error)

	result, err := GetDesktopUsage(fixture.user.Id, 1, 100, 0, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(3), result.Total)
	require.Len(t, result.Items, 3)
	itemsByRequest := make(map[string]DesktopUsageItem, len(result.Items))
	logTypes := make(map[int]int)
	for _, item := range result.Items {
		logTypes[item.Type]++
		itemsByRequest[item.RequestID] = item
	}
	assert.Equal(t, 2, logTypes[model.LogTypeConsume])
	assert.Equal(t, 1, logTypes[model.LogTypeTopup],
		"the original page API must retain its pre-v2 all-log behavior")
	assert.Equal(t, 5, itemsByRequest["legacy-request"].CacheTokens)
	assert.Equal(t, 11, itemsByRequest["legacy-request"].CacheCreationTokens)
	assert.Equal(t, 19, itemsByRequest["stable-request"].CacheTokens)
	assert.Equal(t, 23, itemsByRequest["stable-request"].CacheCreationTokens)

	summary, err := GetDesktopUsageSummary(fixture.user.Id, now-86400, now)
	require.NoError(t, err)
	assert.Equal(t, int64(2), summary.Totals.RequestCount)
	assert.Equal(t, int64(30), summary.Totals.Quota)
	assert.Equal(t, int64(9), summary.Totals.PromptTokens)
	assert.Equal(t, int64(12), summary.Totals.CompletionTokens)
	assert.Equal(t, int64(24), summary.Totals.CacheTokens)
	assert.Equal(t, int64(34), summary.Totals.CacheCreationTokens)
	assert.Equal(t, int64(1), summary.Totals.StreamCount)
	assert.Len(t, summary.ByDay, 2)
	assert.Len(t, summary.ByModel, 2)
	assert.Equal(t, int64(0), summary.LongestTaskSeconds)
	assert.False(t, summary.ActivityTruncated)
	assert.False(t, summary.Truncated)

	firstPage, err := GetDesktopUsageByCursor(fixture.user.Id, "", 1, 0, 0)
	require.NoError(t, err)
	require.Len(t, firstPage.Items, 1)
	assert.True(t, firstPage.HasMore)
	require.NotEmpty(t, firstPage.NextCursor)
	secondPage, err := GetDesktopUsageByCursor(fixture.user.Id, firstPage.NextCursor, 1, 0, 0)
	require.NoError(t, err)
	require.Len(t, secondPage.Items, 1)
	assert.False(t, secondPage.HasMore)
	assert.NotEqual(t, firstPage.Items[0].RequestID, secondPage.Items[0].RequestID)
}

func TestDesktopUsageSummaryComputesLongestTaskFromBoundedActivity(t *testing.T) {
	fixture := setupDesktopAuthorizationTest(t)
	start := time.Now().Unix() - 10000
	for index, createdAt := range []int64{
		start,
		start + 600,
		start + 1200,
		start + 4000,
		start + 4300,
	} {
		require.NoError(t, model.LOG_DB.Create(&model.Log{
			UserId:    fixture.user.Id,
			CreatedAt: createdAt,
			Type:      model.LogTypeConsume,
			RequestId: "longest-task-" + strconv.Itoa(index),
		}).Error)
	}

	summary, err := GetDesktopUsageSummary(fixture.user.Id, start, start+5000)
	require.NoError(t, err)
	assert.Equal(t, int64(1200), summary.LongestTaskSeconds)
	assert.False(t, summary.ActivityTruncated)
	assert.False(t, summary.Truncated)
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
	billingToken, err := model.GetTokenById(desktopAccess.Token.Id)
	require.NoError(t, err)
	assert.True(t, billingToken.PccAgent, "billing lookups must preserve desktop token cache metadata")
	assert.Equal(t, "claude", billingToken.PccAgentEngine)
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
