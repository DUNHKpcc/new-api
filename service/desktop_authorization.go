package service

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	DesktopClientID                = "pcc-agent-desktop"
	DesktopClientDisplayName       = "PCC Agent"
	DesktopAuthorizationScopes     = "relay account.read usage.read"
	DesktopAuthorizationRequestTTL = 10 * time.Minute
	DesktopAuthorizationCodeTTL    = 2 * time.Minute
	DesktopAccessTokenTTL          = 90 * 24 * time.Hour

	desktopRedirectNonceMinBytes = 16
	desktopStateMinBytes         = 32
	desktopDeviceNameMaxLength   = 128
	desktopPlatformMaxLength     = 64
	desktopAppVersionMaxLength   = 32
	desktopDeviceIDMaxLength     = 128
)

var (
	ErrDesktopInvalidRequest    = errors.New("desktop authorization request is invalid")
	ErrDesktopUnsupportedClient = errors.New("desktop client is not supported")
	ErrDesktopInvalidRedirect   = errors.New("desktop redirect URI is invalid")
	ErrDesktopInvalidPKCE       = errors.New("desktop PKCE proof is invalid")
	ErrDesktopRequestExpired    = errors.New("desktop authorization request has expired")
	ErrDesktopRequestConsumed   = errors.New("desktop authorization request has already been consumed")
	ErrDesktopDeviceLimit       = errors.New("desktop device limit reached")
	ErrDesktopTokenInvalid      = errors.New("desktop access token is invalid")
	ErrDesktopScopeDenied       = errors.New("desktop access token scope is denied")
	ErrDesktopBrowserSession    = errors.New("desktop authorization requires a browser session")
	ErrDesktopGroupUnavailable  = errors.New("desktop token group is unavailable to this user")

	desktopBase64URLPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	desktopVerifierPattern  = regexp.MustCompile(`^[A-Za-z0-9._~-]+$`)
)

type DesktopAuthorizationRequestInput struct {
	ClientID            string `json:"client_id"`
	RedirectURI         string `json:"redirect_uri"`
	State               string `json:"state"`
	CodeChallenge       string `json:"code_challenge"`
	CodeChallengeMethod string `json:"code_challenge_method"`
	DeviceID            string `json:"device_id"`
	DeviceName          string `json:"device_name"`
	Platform            string `json:"platform"`
	AppVersion          string `json:"app_version"`
}

type desktopAuthorizationRequestPayload struct {
	ClientID      string `json:"client_id"`
	RedirectURI   string `json:"redirect_uri"`
	State         string `json:"state"`
	CodeChallenge string `json:"code_challenge"`
	DeviceIDHash  string `json:"device_id_hash"`
	DeviceName    string `json:"device_name"`
	Platform      string `json:"platform"`
	AppVersion    string `json:"app_version"`
}

type desktopAuthorizationCodePayload struct {
	ClientID      string `json:"client_id"`
	RedirectURI   string `json:"redirect_uri"`
	CodeChallenge string `json:"code_challenge"`
	UserID        int    `json:"user_id"`
	SessionID     string `json:"session_id"`
	GrantPublicID string `json:"grant_public_id"`
	DeviceIDHash  string `json:"device_id_hash"`
}

type DesktopAuthorizationRequestResult struct {
	RequestToken     string `json:"request_token"`
	AuthorizationURL string `json:"authorization_url"`
	ExpiresIn        int    `json:"expires_in"`
}

type DesktopAuthorizationRequestView struct {
	ClientID      string   `json:"client_id"`
	ClientName    string   `json:"client_name"`
	DeviceName    string   `json:"device_name"`
	Platform      string   `json:"platform"`
	AppVersion    string   `json:"app_version"`
	Scopes        []string `json:"scopes"`
	AllowedModels []string `json:"allowed_models"`
	ExpiresAt     int64    `json:"expires_at"`
	TokenTTL      int64    `json:"token_ttl"`
}

type DesktopAuthorizationDecisionInput struct {
	RequestToken string
	Decision     string
	UserID       int
	SessionID    string
}

type DesktopAuthorizationDecisionResult struct {
	RedirectURI string `json:"redirect_uri"`
}

type DesktopTokenExchangeInput struct {
	GrantType    string `json:"grant_type"`
	ClientID     string `json:"client_id"`
	Code         string `json:"code"`
	RedirectURI  string `json:"redirect_uri"`
	CodeVerifier string `json:"code_verifier"`
	DeviceID     string `json:"device_id"`
}

type DesktopAccountSubscription struct {
	State     string `json:"state"`
	ExpiresAt int64  `json:"expires_at,omitempty"`
}

type DesktopAccount struct {
	DisplayName       string                     `json:"display_name"`
	Email             string                     `json:"email"`
	Status            int                        `json:"status"`
	Quota             int                        `json:"quota"`
	Subscription      DesktopAccountSubscription `json:"subscription"`
	SubscriptionState string                     `json:"subscription_state"`
	AllowedModels     []string                   `json:"allowed_models"`
	TokenExpiresAt    int64                      `json:"token_expires_at"`
	Scopes            []string                   `json:"scopes"`
	DeviceName        string                     `json:"device_name"`
	GrantPublicID     string                     `json:"grant_public_id"`
}

type DesktopTokenExchangeResult struct {
	TokenType string                    `json:"token_type"`
	Tokens    DesktopEngineAccessTokens `json:"tokens"`
	ExpiresIn int64                     `json:"expires_in"`
	Scope     string                    `json:"scope"`
	Account   DesktopAccount            `json:"account"`
}

type DesktopEngineAccessToken struct {
	AccessToken   string   `json:"access_token"`
	Group         string   `json:"group"`
	AllowedModels []string `json:"allowed_models"`
}

type DesktopEngineAccessTokens struct {
	Claude DesktopEngineAccessToken `json:"claude"`
	Codex  DesktopEngineAccessToken `json:"codex"`
}

type DesktopAccess struct {
	Token *model.Token
	Grant *model.DesktopGrant
	User  *model.UserBase
}

type DesktopUsageItem struct {
	CreatedAt        int64  `json:"created_at"`
	Type             int    `json:"type"`
	ModelName        string `json:"model_name"`
	Quota            int    `json:"quota"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	UseTime          int    `json:"use_time"`
	IsStream         bool   `json:"is_stream"`
	Group            string `json:"group"`
	RequestID        string `json:"request_id,omitempty"`
}

type DesktopUsagePage struct {
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
	Total    int64              `json:"total"`
	Items    []DesktopUsageItem `json:"items"`
}

func CreateDesktopAuthorizationRequest(input DesktopAuthorizationRequestInput, authorizationOrigin string) (*DesktopAuthorizationRequestResult, error) {
	payload, err := validateDesktopAuthorizationRequest(input)
	if err != nil {
		return nil, err
	}
	authorizationURL, err := buildDesktopAuthorizationURL(authorizationOrigin, "")
	if err != nil {
		return nil, err
	}
	payloadBytes, err := common.Marshal(payload)
	if err != nil {
		return nil, err
	}
	expiresAt := time.Now().Add(DesktopAuthorizationRequestTTL)
	requestToken, _, err := model.CreateAuthFlow(model.AuthFlowCreate{
		Purpose:   model.AuthFlowPurposeDesktopRequest,
		Provider:  DesktopClientID,
		Payload:   string(payloadBytes),
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return nil, err
	}
	authorizationURL, err = buildDesktopAuthorizationURL(authorizationOrigin, requestToken)
	if err != nil {
		return nil, err
	}
	return &DesktopAuthorizationRequestResult{
		RequestToken:     requestToken,
		AuthorizationURL: authorizationURL,
		ExpiresIn:        int(DesktopAuthorizationRequestTTL / time.Second),
	}, nil
}

func GetDesktopAuthorizationRequest(requestToken string, userID int) (*DesktopAuthorizationRequestView, error) {
	flow, err := model.GetAuthFlow(strings.TrimSpace(requestToken), model.AuthFlowMatch{
		Purpose:  model.AuthFlowPurposeDesktopRequest,
		Provider: DesktopClientID,
	})
	if err != nil {
		return nil, mapDesktopAuthFlowError(err)
	}
	payload, err := decodeDesktopRequestPayload(flow.Payload)
	if err != nil {
		return nil, err
	}
	allowedModels := []string{}
	if userID > 0 {
		claudePolicy, codexPolicy, policyErr := desktopTokenPolicies(userID, payload.DeviceName)
		if policyErr != nil {
			return nil, policyErr
		}
		allowedModels = mergeDesktopAllowedModels(claudePolicy, codexPolicy)
	}
	return &DesktopAuthorizationRequestView{
		ClientID:      payload.ClientID,
		ClientName:    DesktopClientDisplayName,
		DeviceName:    payload.DeviceName,
		Platform:      payload.Platform,
		AppVersion:    payload.AppVersion,
		Scopes:        strings.Fields(DesktopAuthorizationScopes),
		AllowedModels: allowedModels,
		ExpiresAt:     flow.ExpiresAt.Unix(),
		TokenTTL:      int64(DesktopAccessTokenTTL / time.Second),
	}, nil
}

func DecideDesktopAuthorization(input DesktopAuthorizationDecisionInput) (*DesktopAuthorizationDecisionResult, error) {
	if input.UserID <= 0 || strings.TrimSpace(input.SessionID) == "" {
		return nil, ErrDesktopBrowserSession
	}
	decision := strings.ToLower(strings.TrimSpace(input.Decision))
	if decision != "allow" && decision != "deny" {
		return nil, ErrDesktopInvalidRequest
	}

	var redirectURI string
	_, err := model.ConsumeAuthFlowWithAction(strings.TrimSpace(input.RequestToken), model.AuthFlowMatch{
		Purpose:  model.AuthFlowPurposeDesktopRequest,
		Provider: DesktopClientID,
	}, func(tx *gorm.DB, flow *model.AuthFlow) error {
		payload, decodeErr := decodeDesktopRequestPayload(flow.Payload)
		if decodeErr != nil {
			return decodeErr
		}
		if decision == "deny" {
			redirectURI, decodeErr = desktopLoopbackResultURL(payload.RedirectURI, url.Values{
				"error": {"access_denied"},
				"state": {payload.State},
			})
			return decodeErr
		}

		maxActiveDevices := common.GetEnvOrDefault("DESKTOP_GRANT_ACTIVE_LIMIT", 10)
		if maxActiveDevices <= 0 {
			maxActiveDevices = 10
		}
		var activeCount int64
		if countErr := tx.Model(&model.DesktopGrant{}).
			Where("user_id = ? AND status = ? AND active_slot = ? AND expired_time > ?",
				input.UserID, model.DesktopGrantStatusActive, 1, time.Now().Unix()).
			Count(&activeCount).Error; countErr != nil {
			return countErr
		}
		var sameDeviceCount int64
		if countErr := tx.Model(&model.DesktopGrant{}).
			Where("user_id = ? AND client_id = ? AND device_id_hash = ? AND status = ? AND active_slot = ? AND expired_time > ?",
				input.UserID, payload.ClientID, payload.DeviceIDHash, model.DesktopGrantStatusActive, 1, time.Now().Unix()).
			Count(&sameDeviceCount).Error; countErr != nil {
			return countErr
		}
		if activeCount >= int64(maxActiveDevices) && sameDeviceCount == 0 {
			return ErrDesktopDeviceLimit
		}

		grant := &model.DesktopGrant{
			PublicId:     uuid.NewString(),
			UserId:       input.UserID,
			ClientId:     payload.ClientID,
			DeviceIdHash: payload.DeviceIDHash,
			DeviceName:   payload.DeviceName,
			Platform:     payload.Platform,
			AppVersion:   payload.AppVersion,
			Scopes:       DesktopAuthorizationScopes,
		}
		if createErr := model.CreatePendingDesktopGrantWithTx(tx, grant); createErr != nil {
			return createErr
		}

		codePayloadBytes, marshalErr := common.Marshal(desktopAuthorizationCodePayload{
			ClientID:      payload.ClientID,
			RedirectURI:   payload.RedirectURI,
			CodeChallenge: payload.CodeChallenge,
			UserID:        input.UserID,
			SessionID:     input.SessionID,
			GrantPublicID: grant.PublicId,
			DeviceIDHash:  payload.DeviceIDHash,
		})
		if marshalErr != nil {
			return marshalErr
		}
		code, _, createErr := model.CreateAuthFlowWithTx(tx, model.AuthFlowCreate{
			Purpose:   model.AuthFlowPurposeDesktopCode,
			Provider:  DesktopClientID,
			UserId:    input.UserID,
			SessionId: input.SessionID,
			Payload:   string(codePayloadBytes),
			ExpiresAt: time.Now().Add(DesktopAuthorizationCodeTTL),
		})
		if createErr != nil {
			return createErr
		}
		redirectURI, createErr = desktopLoopbackResultURL(payload.RedirectURI, url.Values{
			"code":  {code},
			"state": {payload.State},
		})
		return createErr
	})
	if err != nil {
		return nil, mapDesktopAuthFlowError(err)
	}
	return &DesktopAuthorizationDecisionResult{RedirectURI: redirectURI}, nil
}

func ExchangeDesktopAuthorizationCode(input DesktopTokenExchangeInput) (*DesktopTokenExchangeResult, error) {
	if input.GrantType != "authorization_code" || input.ClientID != DesktopClientID ||
		strings.TrimSpace(input.Code) == "" || strings.TrimSpace(input.DeviceID) == "" {
		return nil, ErrDesktopInvalidRequest
	}
	if err := validateDesktopRedirectURI(input.RedirectURI); err != nil {
		return nil, err
	}
	if !validDesktopVerifier(input.CodeVerifier) {
		return nil, ErrDesktopInvalidPKCE
	}

	flow, err := model.GetAuthFlow(input.Code, model.AuthFlowMatch{
		Purpose:  model.AuthFlowPurposeDesktopCode,
		Provider: DesktopClientID,
	})
	if err != nil {
		return nil, mapDesktopAuthFlowError(err)
	}
	var payload desktopAuthorizationCodePayload
	if err := common.UnmarshalJsonStr(flow.Payload, &payload); err != nil {
		return nil, ErrDesktopInvalidRequest
	}
	if payload.ClientID != input.ClientID || payload.RedirectURI != input.RedirectURI ||
		payload.DeviceIDHash != desktopDeviceIDHash(input.DeviceID) {
		return nil, ErrDesktopInvalidRequest
	}
	challenge := sha256.Sum256([]byte(input.CodeVerifier))
	computedChallenge := base64.RawURLEncoding.EncodeToString(challenge[:])
	if subtle.ConstantTimeCompare([]byte(computedChallenge), []byte(payload.CodeChallenge)) != 1 {
		return nil, ErrDesktopInvalidPKCE
	}
	if _, err := ValidateSessionReference(payload.UserID, payload.SessionID); err != nil {
		return nil, ErrDesktopBrowserSession
	}

	pendingGrant, err := model.GetPendingDesktopGrant(payload.GrantPublicID, payload.UserID)
	if err != nil {
		return nil, err
	}
	claudePolicy, codexPolicy, err := desktopTokenPolicies(payload.UserID, pendingGrant.DeviceName)
	if err != nil {
		return nil, err
	}
	claudeKey, err := common.GenerateKey()
	if err != nil {
		return nil, err
	}
	codexKey, err := common.GenerateKey()
	if err != nil {
		return nil, err
	}
	expiresAt := time.Now().Add(DesktopAccessTokenTTL).Unix()
	claudePolicy.Key = claudeKey
	claudePolicy.ExpiredTime = expiresAt
	codexPolicy.Key = codexKey
	codexPolicy.ExpiredTime = expiresAt

	var activation *model.DesktopGrantActivationResult
	_, err = model.ConsumeAuthFlowWithAction(input.Code, model.AuthFlowMatch{
		Purpose:   model.AuthFlowPurposeDesktopCode,
		Provider:  DesktopClientID,
		UserId:    payload.UserID,
		SessionId: payload.SessionID,
	}, func(tx *gorm.DB, _ *model.AuthFlow) error {
		if sessionErr := model.ValidateActiveUserSessionWithTx(tx, payload.UserID, payload.SessionID); sessionErr != nil {
			return sessionErr
		}
		maxActiveDevices := common.GetEnvOrDefault("DESKTOP_GRANT_ACTIVE_LIMIT", 10)
		if maxActiveDevices <= 0 {
			maxActiveDevices = 10
		}
		var activeCount int64
		if countErr := tx.Model(&model.DesktopGrant{}).
			Where("user_id = ? AND status = ? AND active_slot = ? AND expired_time > ?",
				payload.UserID, model.DesktopGrantStatusActive, 1, time.Now().Unix()).
			Count(&activeCount).Error; countErr != nil {
			return countErr
		}
		var sameDeviceCount int64
		if countErr := tx.Model(&model.DesktopGrant{}).
			Where("user_id = ? AND client_id = ? AND device_id_hash = ? AND status = ? AND active_slot = ? AND expired_time > ?",
				payload.UserID, payload.ClientID, payload.DeviceIDHash, model.DesktopGrantStatusActive, 1, time.Now().Unix()).
			Count(&sameDeviceCount).Error; countErr != nil {
			return countErr
		}
		if activeCount >= int64(maxActiveDevices) && sameDeviceCount == 0 {
			return ErrDesktopDeviceLimit
		}
		var activateErr error
		activation, activateErr = model.ActivateDesktopGrantWithTokensTx(
			tx,
			payload.GrantPublicID,
			payload.UserID,
			claudePolicy,
			codexPolicy,
		)
		return activateErr
	})
	if err != nil {
		return nil, mapDesktopAuthFlowError(err)
	}
	if err := model.InvalidateTokenKeysCache(activation.RevokedTokenKeys); err != nil {
		common.SysLog("failed to invalidate reauthorized desktop token cache: " + err.Error())
	}

	account, err := BuildDesktopAccount(activation.ClaudeToken, activation.Grant, activation.CodexToken)
	if err != nil {
		return nil, err
	}
	return &DesktopTokenExchangeResult{
		TokenType: "Bearer",
		Tokens: DesktopEngineAccessTokens{
			Claude: DesktopEngineAccessToken{
				AccessToken:   "sk-" + activation.ClaudeToken.Key,
				Group:         activation.ClaudeToken.Group,
				AllowedModels: activation.ClaudeToken.GetModelLimits(),
			},
			Codex: DesktopEngineAccessToken{
				AccessToken:   "sk-" + activation.CodexToken.Key,
				Group:         activation.CodexToken.Group,
				AllowedModels: activation.CodexToken.GetModelLimits(),
			},
		},
		ExpiresIn: int64(DesktopAccessTokenTTL / time.Second),
		Scope:     DesktopAuthorizationScopes,
		Account:   *account,
	}, nil
}

func AuthenticateDesktopAccessToken(rawToken, requiredScope string) (*DesktopAccess, error) {
	key := normalizeDesktopAccessToken(rawToken)
	if key == "" {
		return nil, ErrDesktopTokenInvalid
	}
	token, err := model.GetTokenByKey(key, false)
	if err != nil || token == nil || token.Status != common.TokenStatusEnabled ||
		(token.ExpiredTime != -1 && token.ExpiredTime <= time.Now().Unix()) {
		return nil, ErrDesktopTokenInvalid
	}
	grant, err := model.GetActiveDesktopGrantByTokenId(token.Id)
	if err != nil || grant.UserId != token.UserId {
		return nil, ErrDesktopTokenInvalid
	}
	if requiredScope != "" && !desktopScopeAllowed(grant.Scopes, requiredScope) {
		return nil, ErrDesktopScopeDenied
	}
	user, err := model.GetUserCache(token.UserId)
	if err != nil || user.Status != common.UserStatusEnabled {
		return nil, ErrDesktopTokenInvalid
	}
	return &DesktopAccess{Token: token, Grant: grant, User: user}, nil
}

func RevokeDesktopAccessToken(rawToken string) error {
	key := normalizeDesktopAccessToken(rawToken)
	if key == "" {
		return nil
	}
	token, err := model.GetTokenByKey(key, true)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	_, keys, err := model.RevokeDesktopGrantByTokenId(token.Id, "client_revoked")
	if errors.Is(err, model.ErrDesktopGrantNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	return model.InvalidateTokenKeysCache(keys)
}

func RevokeUserDesktopGrant(userID int, publicID string) error {
	keys, err := model.RevokeDesktopGrant(userID, strings.TrimSpace(publicID), "user_revoked")
	if err != nil {
		return err
	}
	return model.InvalidateTokenKeysCache(keys)
}

func RevokeAllUserDesktopGrants(userID int, reason string) error {
	keys, err := model.RevokeAllDesktopGrants(userID, reason)
	if err != nil {
		return err
	}
	return model.InvalidateTokenKeysCache(keys)
}

func BuildDesktopAccount(token *model.Token, grant *model.DesktopGrant, additionalTokens ...*model.Token) (*DesktopAccount, error) {
	if token == nil || grant == nil || token.UserId <= 0 || grant.UserId != token.UserId {
		return nil, ErrDesktopTokenInvalid
	}
	user, err := model.GetUserById(token.UserId, false)
	if err != nil {
		return nil, err
	}
	displayName := strings.TrimSpace(user.DisplayName)
	if displayName == "" {
		displayName = user.Username
	}
	subscription := DesktopAccountSubscription{State: "none"}
	subscriptions, err := model.GetAllActiveUserSubscriptions(user.Id)
	if err != nil {
		return nil, err
	}
	if len(subscriptions) > 0 {
		subscription.State = "active"
		for _, summary := range subscriptions {
			if summary.Subscription != nil && summary.Subscription.EndTime > subscription.ExpiresAt {
				subscription.ExpiresAt = summary.Subscription.EndTime
			}
		}
	}
	modelNames := make(map[string]struct{})
	for _, allowedToken := range append([]*model.Token{token}, additionalTokens...) {
		if allowedToken == nil || allowedToken.UserId != token.UserId {
			continue
		}
		for _, modelName := range allowedToken.GetModelLimits() {
			modelNames[modelName] = struct{}{}
		}
	}
	allowedModels := make([]string, 0, len(modelNames))
	for modelName := range modelNames {
		allowedModels = append(allowedModels, modelName)
	}
	sort.Strings(allowedModels)
	return &DesktopAccount{
		DisplayName:       displayName,
		Email:             common.MaskEmail(user.Email),
		Status:            user.Status,
		Quota:             user.Quota,
		Subscription:      subscription,
		SubscriptionState: subscription.State,
		AllowedModels:     allowedModels,
		TokenExpiresAt:    token.ExpiredTime,
		Scopes:            strings.Fields(grant.Scopes),
		DeviceName:        grant.DeviceName,
		GrantPublicID:     grant.PublicId,
	}, nil
}

func GetDesktopUsage(userID, page, pageSize int, startTimestamp, endTimestamp int64) (*DesktopUsagePage, error) {
	if userID <= 0 {
		return nil, ErrDesktopTokenInvalid
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	logs, total, err := model.GetUserLogs(
		userID,
		model.LogTypeUnknown,
		startTimestamp,
		endTimestamp,
		"",
		"",
		(page-1)*pageSize,
		pageSize,
		"",
		"",
		"",
	)
	if err != nil {
		return nil, err
	}
	items := make([]DesktopUsageItem, 0, len(logs))
	for _, log := range logs {
		items = append(items, DesktopUsageItem{
			CreatedAt:        log.CreatedAt,
			Type:             log.Type,
			ModelName:        log.ModelName,
			Quota:            log.Quota,
			PromptTokens:     log.PromptTokens,
			CompletionTokens: log.CompletionTokens,
			UseTime:          log.UseTime,
			IsStream:         log.IsStream,
			Group:            log.Group,
			RequestID:        log.RequestId,
		})
	}
	return &DesktopUsagePage{
		Page:     page,
		PageSize: pageSize,
		Total:    total,
		Items:    items,
	}, nil
}

func validateDesktopAuthorizationRequest(input DesktopAuthorizationRequestInput) (*desktopAuthorizationRequestPayload, error) {
	if input.ClientID != DesktopClientID {
		return nil, ErrDesktopUnsupportedClient
	}
	if input.CodeChallengeMethod != "S256" || !validBase64URLBytes(input.CodeChallenge, sha256.Size) {
		return nil, ErrDesktopInvalidPKCE
	}
	if !validBase64URLMinBytes(input.State, desktopStateMinBytes) {
		return nil, ErrDesktopInvalidRequest
	}
	if err := validateDesktopRedirectURI(input.RedirectURI); err != nil {
		return nil, err
	}
	deviceID := strings.TrimSpace(input.DeviceID)
	deviceName := strings.TrimSpace(input.DeviceName)
	platform := strings.TrimSpace(input.Platform)
	appVersion := strings.TrimSpace(input.AppVersion)
	if deviceID == "" || len(deviceID) > desktopDeviceIDMaxLength ||
		deviceName == "" || len(deviceName) > desktopDeviceNameMaxLength ||
		len(platform) > desktopPlatformMaxLength || len(appVersion) > desktopAppVersionMaxLength {
		return nil, ErrDesktopInvalidRequest
	}
	return &desktopAuthorizationRequestPayload{
		ClientID:      input.ClientID,
		RedirectURI:   input.RedirectURI,
		State:         input.State,
		CodeChallenge: input.CodeChallenge,
		DeviceIDHash:  desktopDeviceIDHash(deviceID),
		DeviceName:    deviceName,
		Platform:      platform,
		AppVersion:    appVersion,
	}, nil
}

func validateDesktopRedirectURI(value string) error {
	if strings.TrimSpace(value) != value || value == "" {
		return ErrDesktopInvalidRedirect
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Scheme != "http" || parsed.Hostname() != "127.0.0.1" ||
		parsed.User != nil || parsed.Fragment != "" || parsed.RawFragment != "" ||
		parsed.RawQuery != "" || parsed.ForceQuery || parsed.Opaque != "" || parsed.RawPath != "" {
		return ErrDesktopInvalidRedirect
	}
	port := parsed.Port()
	if port == "" {
		return ErrDesktopInvalidRedirect
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1024 || portNumber > 65535 {
		return ErrDesktopInvalidRedirect
	}
	const callbackPrefix = "/oauth/callback/"
	if !strings.HasPrefix(parsed.Path, callbackPrefix) ||
		strings.Contains(strings.TrimPrefix(parsed.Path, callbackPrefix), "/") ||
		!validBase64URLMinBytes(strings.TrimPrefix(parsed.Path, callbackPrefix), desktopRedirectNonceMinBytes) {
		return ErrDesktopInvalidRedirect
	}
	return nil
}

func validDesktopVerifier(verifier string) bool {
	return len(verifier) >= 43 && len(verifier) <= 128 && desktopVerifierPattern.MatchString(verifier)
}

func validBase64URLBytes(value string, expected int) bool {
	if !desktopBase64URLPattern.MatchString(value) {
		return false
	}
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	return err == nil && len(decoded) == expected
}

func validBase64URLMinBytes(value string, minimum int) bool {
	if len(value) > 256 || !desktopBase64URLPattern.MatchString(value) {
		return false
	}
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	return err == nil && len(decoded) >= minimum
}

func desktopDeviceIDHash(deviceID string) string {
	return common.GenerateHMACWithKey([]byte("desktop-device-v1:"+common.SessionSecret), strings.TrimSpace(deviceID))
}

func decodeDesktopRequestPayload(raw string) (*desktopAuthorizationRequestPayload, error) {
	var payload desktopAuthorizationRequestPayload
	if err := common.UnmarshalJsonStr(raw, &payload); err != nil {
		return nil, ErrDesktopInvalidRequest
	}
	if payload.ClientID != DesktopClientID || payload.DeviceIDHash == "" || payload.DeviceName == "" {
		return nil, ErrDesktopInvalidRequest
	}
	if err := validateDesktopRedirectURI(payload.RedirectURI); err != nil {
		return nil, err
	}
	return &payload, nil
}

func desktopTokenPolicies(
	userID int,
	deviceName string,
) (model.DesktopGrantTokenPolicy, model.DesktopGrantTokenPolicy, error) {
	user, err := model.GetUserCache(userID)
	if err != nil {
		return model.DesktopGrantTokenPolicy{}, model.DesktopGrantTokenPolicy{}, err
	}
	settings := operation_setting.GetDesktopAgentSetting()
	configuredGroups := []string{settings.ClaudeGroup, settings.CodexGroup}
	engineNames := []string{"Claude", "Codex"}
	policies := make([]model.DesktopGrantTokenPolicy, 0, len(configuredGroups))
	for i, configuredGroup := range configuredGroups {
		group := strings.TrimSpace(configuredGroup)
		if group == "" {
			group = operation_setting.DesktopAgentAutoGroup
		}
		groups := []string{group}
		if group == operation_setting.DesktopAgentAutoGroup {
			group = user.Group
			groups = []string{user.Group}
			if _, ok := GetUserUsableGroups(user.Group)[operation_setting.DesktopAgentAutoGroup]; ok {
				autoGroups := GetUserAutoGroup(user.Group)
				if len(autoGroups) > 0 {
					group = operation_setting.DesktopAgentAutoGroup
					groups = autoGroups
				}
			}
		} else if !GroupInUserUsableGroups(user.Group, group) {
			return model.DesktopGrantTokenPolicy{}, model.DesktopGrantTokenPolicy{}, ErrDesktopGroupUnavailable
		}

		models := GetGroupsEnabledModels(groups)
		configuredAllowlist := splitNonEmpty(common.GetEnvOrDefaultString("PCC_DESKTOP_MODEL_ALLOWLIST", ""))
		if len(configuredAllowlist) > 0 {
			allowed := make(map[string]struct{}, len(configuredAllowlist))
			for _, modelName := range configuredAllowlist {
				allowed[modelName] = struct{}{}
			}
			filtered := models[:0]
			for _, modelName := range models {
				if _, ok := allowed[modelName]; ok {
					filtered = append(filtered, modelName)
				}
			}
			models = filtered
		}
		sort.Strings(models)
		name := DesktopClientDisplayName + " " + engineNames[i]
		if trimmedName := strings.TrimSpace(deviceName); trimmedName != "" {
			deviceNameRunes := []rune(trimmedName)
			if len(deviceNameRunes) > 32 {
				trimmedName = string(deviceNameRunes[:32])
			}
			name += " - " + trimmedName
		}
		policies = append(policies, model.DesktopGrantTokenPolicy{
			Name:           name,
			Group:          group,
			ModelLimits:    strings.Join(models, ","),
			Scopes:         DesktopAuthorizationScopes,
			UnlimitedQuota: true,
		})
	}
	return policies[0], policies[1], nil
}

func mergeDesktopAllowedModels(policies ...model.DesktopGrantTokenPolicy) []string {
	seen := make(map[string]struct{})
	models := make([]string, 0)
	for _, policy := range policies {
		for _, modelName := range splitNonEmpty(policy.ModelLimits) {
			if _, ok := seen[modelName]; ok {
				continue
			}
			seen[modelName] = struct{}{}
			models = append(models, modelName)
		}
	}
	sort.Strings(models)
	return models
}

func splitNonEmpty(value string) []string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ' '
	})
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func desktopScopeAllowed(scopes, required string) bool {
	for _, scope := range strings.Fields(scopes) {
		if scope == required {
			return true
		}
	}
	return false
}

func buildDesktopAuthorizationURL(authorizationOrigin, requestToken string) (string, error) {
	base, err := url.Parse(strings.TrimSpace(authorizationOrigin))
	if err != nil || (base.Scheme != "http" && base.Scheme != "https") || base.Host == "" ||
		base.User != nil || base.Fragment != "" {
		return "", ErrDesktopInvalidRequest
	}
	base.RawQuery = ""
	base.Path = strings.TrimRight(base.Path, "/") + "/desktop/authorize"
	query := url.Values{}
	if requestToken != "" {
		query.Set("request", requestToken)
	}
	base.RawQuery = query.Encode()
	return base.String(), nil
}

func desktopLoopbackResultURL(redirectURI string, values url.Values) (string, error) {
	if err := validateDesktopRedirectURI(redirectURI); err != nil {
		return "", err
	}
	parsed, err := url.Parse(redirectURI)
	if err != nil {
		return "", ErrDesktopInvalidRedirect
	}
	parsed.RawQuery = values.Encode()
	return parsed.String(), nil
}

func normalizeDesktopAccessToken(raw string) string {
	raw = strings.TrimSpace(raw)
	parts := strings.Fields(raw)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		raw = parts[1]
	} else if len(parts) != 1 {
		return ""
	}
	return strings.TrimPrefix(raw, "sk-")
}

func mapDesktopAuthFlowError(err error) error {
	switch {
	case errors.Is(err, model.ErrAuthFlowExpired):
		return ErrDesktopRequestExpired
	case errors.Is(err, model.ErrAuthFlowConsumed):
		return ErrDesktopRequestConsumed
	case errors.Is(err, model.ErrAuthFlowInvalid):
		return ErrDesktopInvalidRequest
	case errors.Is(err, model.ErrUserSessionInvalid), errors.Is(err, model.ErrUserSessionInactive):
		return ErrDesktopBrowserSession
	default:
		return err
	}
}
