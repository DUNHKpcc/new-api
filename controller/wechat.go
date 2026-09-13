package controller

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type wechatLoginResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    string `json:"data"`
}

var errWeChatServerBridgeUnavailable = errors.New("微信登录服务未完整配置")

func getWeChatIdByCode(code string) (string, error) {
	return getWeChatIdByCodeWithContext(context.Background(), code)
}

func getWeChatIdByCodeWithContext(ctx context.Context, code string) (string, error) {
	if code == "" {
		return "", errors.New("无效的参数")
	}
	if !common.WeChatServerBridgeConfigured() {
		return "", errWeChatServerBridgeUnavailable
	}
	baseURL := strings.TrimSpace(common.WeChatServerAddress)
	endpoint, err := url.JoinPath(baseURL, "api/wechat/user")
	if err != nil {
		return "", errWeChatServerBridgeUnavailable
	}
	endpointURL, err := url.Parse(endpoint)
	if err != nil {
		return "", errWeChatServerBridgeUnavailable
	}
	query := endpointURL.Query()
	query.Set("code", code)
	endpointURL.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpointURL.String(), nil)
	if err != nil {
		return "", errWeChatServerBridgeUnavailable
	}
	req.Header.Set("Authorization", strings.TrimSpace(common.WeChatServerToken))
	req.Header.Set("Accept", "application/json")
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	httpResponse, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer httpResponse.Body.Close()
	if httpResponse.StatusCode < http.StatusOK || httpResponse.StatusCode >= http.StatusMultipleChoices {
		return "", errors.New("微信登录服务请求失败")
	}
	var res wechatLoginResponse
	err = common.DecodeJson(httpResponse.Body, &res)
	if err != nil {
		return "", err
	}
	if !res.Success {
		return "", errors.New(res.Message)
	}
	if res.Data == "" {
		return "", errors.New("验证码错误或已过期")
	}
	return res.Data, nil
}

// WeChatAuth dispatches the two supported WeChat contracts without allowing a
// code from one contract to be interpreted by the other. A state-bearing
// callback is the Open Platform OAuth flow; a state-less request is the
// administrator-configured verification bridge used by existing deployments.
func WeChatAuth(c *gin.Context) {
	if strings.TrimSpace(c.Query("state")) != "" {
		if c.Param("provider") == "" {
			c.Params = append(c.Params, gin.Param{Key: "provider", Value: "wechat"})
		}
		HandleOAuth(c)
		return
	}

	if !common.WeChatAuthEnabled {
		c.JSON(http.StatusOK, gin.H{
			"message": "管理员未开启通过微信登录以及注册",
			"success": false,
		})
		return
	}
	if !common.WeChatServerBridgeConfigured() {
		c.JSON(http.StatusOK, gin.H{
			"message": "微信登录服务未完整配置，请联系管理员",
			"success": false,
			"code":    "WECHAT_SERVER_BRIDGE_NOT_CONFIGURED",
		})
		return
	}
	code := c.Query("code")
	wechatId, err := getWeChatIdByCodeWithContext(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"message": err.Error(),
			"success": false,
		})
		return
	}
	user := model.User{
		WeChatId: wechatId,
	}
	if model.IsWeChatIdAlreadyTaken(wechatId) {
		err := user.FillUserByWeChatId()
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
		if user.Id == 0 {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "用户已注销",
			})
			return
		}
	} else {
		if common.RegisterEnabled {
			user.Username = "wechat_" + strconv.Itoa(model.GetMaxUserId()+1)
			user.DisplayName = "WeChat User"
			user.Role = common.RoleCommonUser
			user.Status = common.UserStatusEnabled

			if err := user.Insert(0); err != nil {
				c.JSON(http.StatusOK, gin.H{
					"success": false,
					"message": err.Error(),
				})
				return
			}
		} else {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "管理员关闭了新用户注册",
			})
			return
		}
	}

	if user.Status != common.UserStatusEnabled {
		c.JSON(http.StatusOK, gin.H{
			"message": "用户已被封禁",
			"success": false,
		})
		return
	}
	setupLogin(&user, c)
}

type wechatBindRequest struct {
	Code string `json:"code"`
}

func WeChatBind(c *gin.Context) {
	identity, ok := middleware.GetSessionAuthIdentity(c)
	if !ok {
		writeSecurityOperationError(c, service.ErrAuthTokenInvalid)
		return
	}
	succeeded, notificationFailed := false, false
	defer func() {
		recordUserSecurityAudit(c, identity.UserID, "user.binding_bind", map[string]any{"provider": "wechat", "success": succeeded, "notification_failed": notificationFailed})
	}()
	if !common.WeChatAuthEnabled {
		c.JSON(http.StatusOK, gin.H{
			"message": "管理员未开启通过微信登录以及注册",
			"success": false,
		})
		return
	}
	if !common.WeChatServerBridgeConfigured() {
		c.JSON(http.StatusOK, gin.H{
			"message": "微信登录服务未完整配置，请联系管理员",
			"success": false,
			"code":    "WECHAT_SERVER_BRIDGE_NOT_CONFIGURED",
		})
		return
	}
	var req wechatBindRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无效的请求",
		})
		return
	}
	code := strings.TrimSpace(req.Code)
	context, err := common.Marshal(service.AccountBindingContext{Provider: "wechat", Code: code})
	if err != nil {
		writeSecurityOperationError(c, err)
		return
	}
	if middleware.RequireSecurityProof(c, service.VerificationOperation{Scope: service.VerificationScopeAccountBind, Context: context}) == nil {
		return
	}
	wechatId, err := getWeChatIdByCodeWithContext(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"message": err.Error(),
			"success": false,
		})
		return
	}
	if model.IsWeChatIdAlreadyTaken(wechatId) {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "该微信账号已被绑定",
		})
		return
	}
	// 只更新绑定列，避免完整用户快照覆盖并发的封禁、降权或分组变更。
	if err := model.DB.Transaction(func(tx *gorm.DB) error {
		return model.UpdateUserBindColumnForSessionWithTx(tx, identity, "wechat_id", wechatId)
	}); err != nil {
		writeSecurityOperationError(c, err)
		return
	}
	succeeded = true
	user, err := model.GetUserById(identity.UserID, false)
	if err != nil {
		writeSecurityOperationError(c, err)
		return
	}
	notificationFailed = service.NotifyAccountSecurityChange(user.Email, "WeChat account linked") != nil
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    gin.H{"notification_warning": notificationFailed},
	})
	return
}
