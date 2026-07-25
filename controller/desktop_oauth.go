package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type desktopAuthorizationDecisionRequest struct {
	RequestToken string `json:"request_token"`
	Decision     string `json:"decision"`
}

func CreateDesktopAuthorizationRequest(c *gin.Context) {
	var input service.DesktopAuthorizationRequestInput
	if err := common.DecodeJson(c.Request.Body, &input); err != nil {
		writeDesktopAuthorizationError(c, service.ErrDesktopInvalidRequest)
		return
	}
	authorizationOrigin := common.GetEnvOrDefaultString(
		"DESKTOP_AUTHORIZATION_ORIGIN",
		system_setting.ServerAddress,
	)
	result, err := service.CreateDesktopAuthorizationRequest(input, authorizationOrigin)
	if err != nil {
		writeDesktopAuthorizationError(c, err)
		return
	}
	c.JSON(http.StatusCreated, result)
}

func GetDesktopAuthorizationRequest(c *gin.Context) {
	identity, ok := middleware.GetSessionAuthIdentity(c)
	if !ok {
		writeDesktopAuthorizationError(c, service.ErrDesktopBrowserSession)
		return
	}
	result, err := service.GetDesktopAuthorizationRequest(c.Param("request_token"), identity.UserID)
	if err != nil {
		writeDesktopAuthorizationError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func DecideDesktopAuthorization(c *gin.Context) {
	identity, ok := middleware.GetSessionAuthIdentity(c)
	if !ok {
		writeDesktopAuthorizationError(c, service.ErrDesktopBrowserSession)
		return
	}
	var request desktopAuthorizationDecisionRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		writeDesktopAuthorizationError(c, service.ErrDesktopInvalidRequest)
		return
	}
	result, err := service.DecideDesktopAuthorization(service.DesktopAuthorizationDecisionInput{
		RequestToken: request.RequestToken,
		Decision:     request.Decision,
		UserID:       identity.UserID,
		SessionID:    identity.SessionID,
	})
	if err != nil {
		writeDesktopAuthorizationError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func ExchangeDesktopAuthorizationCode(c *gin.Context) {
	var input service.DesktopTokenExchangeInput
	if err := common.DecodeJson(c.Request.Body, &input); err != nil {
		writeDesktopAuthorizationError(c, service.ErrDesktopInvalidRequest)
		return
	}
	result, err := service.ExchangeDesktopAuthorizationCode(input)
	if err != nil {
		writeDesktopAuthorizationError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func RevokeDesktopAuthorization(c *gin.Context) {
	if err := service.RevokeDesktopAccessToken(c.GetHeader("Authorization")); err != nil {
		common.SysLog("desktop token revocation failed: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":             "DESKTOP_INTERNAL_ERROR",
			"error_description": http.StatusText(http.StatusInternalServerError),
		})
		return
	}
	c.Status(http.StatusNoContent)
}

func GetDesktopAccount(c *gin.Context) {
	access, ok := middleware.GetDesktopAccess(c)
	if !ok {
		writeDesktopAuthorizationError(c, service.ErrDesktopTokenInvalid)
		return
	}
	account, err := service.BuildDesktopAccount(access.Token, access.Grant)
	if err != nil {
		writeDesktopAuthorizationError(c, err)
		return
	}
	c.JSON(http.StatusOK, account)
}

func GetDesktopUsage(c *gin.Context) {
	access, ok := middleware.GetDesktopAccess(c)
	if !ok {
		writeDesktopAuthorizationError(c, service.ErrDesktopTokenInvalid)
		return
	}
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	usage, err := service.GetDesktopUsage(access.User.Id, page, pageSize, startTimestamp, endTimestamp)
	if err != nil {
		writeDesktopAuthorizationError(c, err)
		return
	}
	c.JSON(http.StatusOK, usage)
}

func ListDesktopGrants(c *gin.Context) {
	if _, ok := middleware.GetSessionAuthIdentity(c); !ok {
		writeDesktopAuthorizationError(c, service.ErrDesktopBrowserSession)
		return
	}
	grants, err := model.ListDesktopGrants(c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, grants)
}

func DeleteDesktopGrant(c *gin.Context) {
	if _, ok := middleware.GetSessionAuthIdentity(c); !ok {
		writeDesktopAuthorizationError(c, service.ErrDesktopBrowserSession)
		return
	}
	publicID := strings.TrimSpace(c.Param("public_id"))
	if publicID == "" {
		writeDesktopAuthorizationError(c, service.ErrDesktopInvalidRequest)
		return
	}
	err := service.RevokeUserDesktopGrant(c.GetInt("id"), publicID)
	if errors.Is(err, model.ErrDesktopGrantNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "code": "DESKTOP_GRANT_NOT_FOUND", "message": http.StatusText(http.StatusNotFound)})
		return
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func writeDesktopAuthorizationError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	code := "DESKTOP_INTERNAL_ERROR"
	switch {
	case errors.Is(err, service.ErrDesktopUnsupportedClient):
		status, code = http.StatusBadRequest, "DESKTOP_UNSUPPORTED_CLIENT"
	case errors.Is(err, service.ErrDesktopInvalidRedirect):
		status, code = http.StatusBadRequest, "DESKTOP_INVALID_REDIRECT_URI"
	case errors.Is(err, service.ErrDesktopInvalidPKCE):
		status, code = http.StatusBadRequest, "DESKTOP_INVALID_PKCE"
	case errors.Is(err, service.ErrDesktopInvalidRequest):
		status, code = http.StatusBadRequest, "DESKTOP_INVALID_REQUEST"
	case errors.Is(err, service.ErrDesktopBrowserSession):
		status, code = http.StatusForbidden, "DESKTOP_BROWSER_SESSION_REQUIRED"
	case errors.Is(err, service.ErrDesktopRequestExpired):
		status, code = http.StatusGone, "DESKTOP_REQUEST_EXPIRED"
	case errors.Is(err, service.ErrDesktopRequestConsumed):
		status, code = http.StatusConflict, "DESKTOP_REQUEST_CONSUMED"
	case errors.Is(err, service.ErrDesktopDeviceLimit):
		status, code = http.StatusConflict, "DESKTOP_DEVICE_LIMIT"
	case errors.Is(err, service.ErrDesktopGroupUnavailable):
		status, code = http.StatusConflict, "DESKTOP_TOKEN_GROUP_UNAVAILABLE"
	case errors.Is(err, service.ErrDesktopTokenInvalid):
		status, code = http.StatusUnauthorized, "DESKTOP_TOKEN_INVALID"
	case errors.Is(err, service.ErrDesktopScopeDenied):
		status, code = http.StatusForbidden, "DESKTOP_SCOPE_DENIED"
	case errors.Is(err, model.ErrDesktopGrantNotFound), errors.Is(err, gorm.ErrRecordNotFound):
		status, code = http.StatusNotFound, "DESKTOP_NOT_FOUND"
	}
	if status == http.StatusInternalServerError {
		common.SysLog("desktop authorization error: " + err.Error())
	}
	c.JSON(status, gin.H{
		"error":             code,
		"error_description": http.StatusText(status),
	})
}
