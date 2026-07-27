package oauth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

const (
	weChatTokenEndpoint    = "https://api.weixin.qq.com/sns/oauth2/access_token"
	weChatUserInfoEndpoint = "https://api.weixin.qq.com/sns/userinfo"
)

type WeChatProvider struct {
	httpClient       *http.Client
	tokenEndpoint    string
	userInfoEndpoint string
}

type weChatTokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	OpenID       string `json:"openid"`
	Scope        string `json:"scope"`
	UnionID      string `json:"unionid"`
	ErrCode      int    `json:"errcode"`
	ErrMsg       string `json:"errmsg"`
}

type weChatUser struct {
	OpenID     string   `json:"openid"`
	Nickname   string   `json:"nickname"`
	HeadImgURL string   `json:"headimgurl"`
	Privilege  []string `json:"privilege"`
	UnionID    string   `json:"unionid"`
	ErrCode    int      `json:"errcode"`
	ErrMsg     string   `json:"errmsg"`
}

func weChatRequestError(err error) string {
	var urlError *url.Error
	if errors.As(err, &urlError) {
		return urlError.Err.Error()
	}
	return err.Error()
}

func init() {
	Register("wechat", &WeChatProvider{
		httpClient:       &http.Client{Timeout: 5 * time.Second},
		tokenEndpoint:    weChatTokenEndpoint,
		userInfoEndpoint: weChatUserInfoEndpoint,
	})
}

func (p *WeChatProvider) GetName() string {
	return "WeChat"
}

func (p *WeChatProvider) IsEnabled() bool {
	return common.WeChatAuthEnabled && common.WeChatAppId != "" && common.WeChatAppSecret != ""
}

func (p *WeChatProvider) ExchangeToken(ctx context.Context, code string, _ *gin.Context) (*OAuthToken, error) {
	if code == "" {
		return nil, NewOAuthError(i18n.MsgOAuthInvalidCode, nil)
	}

	endpoint := p.tokenEndpoint
	if endpoint == "" {
		endpoint = weChatTokenEndpoint
	}
	tokenURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	query := tokenURL.Query()
	query.Set("appid", common.WeChatAppId)
	query.Set("secret", common.WeChatAppSecret)
	query.Set("code", code)
	query.Set("grant_type", "authorization_code")
	tokenURL.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenURL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	client := p.httpClient
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	res, err := client.Do(req)
	if err != nil {
		rawError := weChatRequestError(err)
		logger.LogError(ctx, fmt.Sprintf("[OAuth-WeChat] ExchangeToken error: %s", rawError))
		return nil, NewOAuthErrorWithRaw(i18n.MsgOAuthConnectFailed, map[string]any{"Provider": "WeChat"}, rawError)
	}
	defer res.Body.Close()

	var tokenResponse weChatTokenResponse
	if err := common.DecodeJson(res.Body, &tokenResponse); err != nil {
		logger.LogError(ctx, fmt.Sprintf("[OAuth-WeChat] ExchangeToken decode error: %s", err.Error()))
		return nil, err
	}
	if res.StatusCode != http.StatusOK || tokenResponse.ErrCode != 0 {
		rawError := fmt.Sprintf("status=%d errcode=%d errmsg=%s", res.StatusCode, tokenResponse.ErrCode, tokenResponse.ErrMsg)
		logger.LogError(ctx, "[OAuth-WeChat] ExchangeToken failed: "+rawError)
		return nil, NewOAuthErrorWithRaw(i18n.MsgOAuthTokenFailed, map[string]any{"Provider": "WeChat"}, rawError)
	}
	if tokenResponse.AccessToken == "" || tokenResponse.OpenID == "" {
		logger.LogError(ctx, "[OAuth-WeChat] ExchangeToken failed: empty access token or openid")
		return nil, NewOAuthError(i18n.MsgOAuthTokenFailed, map[string]any{"Provider": "WeChat"})
	}

	return &OAuthToken{
		AccessToken:  tokenResponse.AccessToken,
		RefreshToken: tokenResponse.RefreshToken,
		ExpiresIn:    tokenResponse.ExpiresIn,
		Scope:        tokenResponse.Scope,
		OpenID:       tokenResponse.OpenID,
		UnionID:      tokenResponse.UnionID,
	}, nil
}

func (p *WeChatProvider) GetUserInfo(ctx context.Context, token *OAuthToken) (*OAuthUser, error) {
	if token == nil || token.AccessToken == "" || token.OpenID == "" {
		return nil, NewOAuthError(i18n.MsgOAuthGetUserErr, nil)
	}

	endpoint := p.userInfoEndpoint
	if endpoint == "" {
		endpoint = weChatUserInfoEndpoint
	}
	userInfoURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	query := userInfoURL.Query()
	query.Set("access_token", token.AccessToken)
	query.Set("openid", token.OpenID)
	query.Set("lang", "zh_CN")
	userInfoURL.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, userInfoURL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	client := p.httpClient
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	res, err := client.Do(req)
	if err != nil {
		rawError := weChatRequestError(err)
		logger.LogError(ctx, fmt.Sprintf("[OAuth-WeChat] GetUserInfo error: %s", rawError))
		return nil, NewOAuthErrorWithRaw(i18n.MsgOAuthConnectFailed, map[string]any{"Provider": "WeChat"}, rawError)
	}
	defer res.Body.Close()

	var weChatUser weChatUser
	if err := common.DecodeJson(res.Body, &weChatUser); err != nil {
		logger.LogError(ctx, fmt.Sprintf("[OAuth-WeChat] GetUserInfo decode error: %s", err.Error()))
		return nil, err
	}
	if res.StatusCode != http.StatusOK || weChatUser.ErrCode != 0 {
		rawError := fmt.Sprintf("status=%d errcode=%d errmsg=%s", res.StatusCode, weChatUser.ErrCode, weChatUser.ErrMsg)
		logger.LogError(ctx, "[OAuth-WeChat] GetUserInfo failed: "+rawError)
		return nil, NewOAuthErrorWithRaw(i18n.MsgOAuthGetUserErr, map[string]any{"Provider": "WeChat"}, rawError)
	}
	if weChatUser.OpenID == "" || weChatUser.OpenID != token.OpenID {
		logger.LogError(ctx, "[OAuth-WeChat] GetUserInfo failed: empty or mismatched openid")
		return nil, NewOAuthError(i18n.MsgOAuthUserInfoEmpty, map[string]any{"Provider": "WeChat"})
	}

	unionID := weChatUser.UnionID
	if unionID == "" {
		unionID = token.UnionID
	}
	// UnionID keeps identities stable across website and mini-program apps.
	providerUserID := unionID
	if providerUserID == "" {
		providerUserID = weChatUser.OpenID
	}

	return &OAuthUser{
		ProviderUserID: providerUserID,
		DisplayName:    weChatUser.Nickname,
		Extra: map[string]any{
			"openid":     weChatUser.OpenID,
			"union_id":   unionID,
			"avatar_url": weChatUser.HeadImgURL,
		},
	}, nil
}

func (p *WeChatProvider) IsUserIDTaken(providerUserID string) bool {
	return model.IsWeChatIdAlreadyTaken(providerUserID)
}

func (p *WeChatProvider) FillUserByProviderID(user *model.User, providerUserID string) error {
	user.WeChatId = providerUserID
	return user.FillUserByWeChatId()
}

func (p *WeChatProvider) SetProviderUserID(user *model.User, providerUserID string) {
	user.WeChatId = providerUserID
}

func (p *WeChatProvider) GetProviderPrefix() string {
	return "wechat_"
}
