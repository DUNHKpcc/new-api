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
)

type desktopAuthorizationDecisionRequest struct {
	RequestToken string `json:"request_token"`
	Decision     string `json:"decision"`
}

type desktopAuthorizationConfirmationRequest struct {
	ConfirmationToken string `json:"confirmation_token"`
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

func ConfirmDesktopAuthorization(c *gin.Context) {
	var request desktopAuthorizationConfirmationRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		writeDesktopAuthorizationError(c, service.ErrDesktopConfirmationInvalid)
		return
	}
	if err := service.ConfirmDesktopAuthorization(request.ConfirmationToken); err != nil {
		writeDesktopAuthorizationError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func RevokeDesktopAuthorization(c *gin.Context) {
	if err := service.RevokeDesktopAccessToken(c.GetHeader("Authorization")); err != nil {
		writeDesktopAuthorizationError(c, err)
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

func GetDesktopSubscriptions(c *gin.Context) {
	access, ok := middleware.GetDesktopAccess(c)
	if !ok {
		writeDesktopAuthorizationError(c, service.ErrDesktopTokenInvalid)
		return
	}
	subscriptions, err := service.GetDesktopSubscriptions(access.User.Id)
	if err != nil {
		writeDesktopAuthorizationError(c, err)
		return
	}
	c.JSON(http.StatusOK, subscriptions)
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
	var usage *service.DesktopUsagePage
	var err error
	if c.Query("pagination") == "cursor" {
		usage, err = service.GetDesktopUsageByCursor(
			access.User.Id,
			c.Query("cursor"),
			pageSize,
			startTimestamp,
			endTimestamp,
		)
	} else {
		usage, err = service.GetDesktopUsage(access.User.Id, page, pageSize, startTimestamp, endTimestamp)
	}
	if err != nil {
		writeDesktopAuthorizationError(c, err)
		return
	}
	c.JSON(http.StatusOK, usage)
}

func GetDesktopUsageSummary(c *gin.Context) {
	access, ok := middleware.GetDesktopAccess(c)
	if !ok {
		writeDesktopAuthorizationError(c, service.ErrDesktopTokenInvalid)
		return
	}
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	usage, err := service.GetDesktopUsageSummary(access.User.Id, startTimestamp, endTimestamp)
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
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"code":    "DESKTOP_GRANT_NOT_FOUND",
			"message": http.StatusText(http.StatusNotFound),
		})
		return
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func DeleteRevokedDesktopGrant(c *gin.Context) {
	if _, ok := middleware.GetSessionAuthIdentity(c); !ok {
		writeDesktopAuthorizationError(c, service.ErrDesktopBrowserSession)
		return
	}
	publicID := strings.TrimSpace(c.Param("public_id"))
	if publicID == "" {
		writeDesktopAuthorizationError(c, service.ErrDesktopInvalidRequest)
		return
	}
	err := service.DeleteRevokedUserDesktopGrant(c.GetInt("id"), publicID)
	if errors.Is(err, model.ErrDesktopGrantNotFound) {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"code":    "DESKTOP_GRANT_NOT_FOUND",
			"message": http.StatusText(http.StatusNotFound),
		})
		return
	}
	if errors.Is(err, model.ErrDesktopGrantNotRevoked) {
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"code":    "DESKTOP_GRANT_NOT_REVOKED",
			"message": http.StatusText(http.StatusConflict),
		})
		return
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func writeDesktopAuthorizationError(c *gin.Context, err error) {
	status, response := service.DesktopErrorFor(err)
	if status == http.StatusInternalServerError {
		common.SysLog("desktop authorization error: " + err.Error())
	}
	c.Header(service.DesktopContractVersionHeader, strconv.Itoa(service.DesktopContractVersion))
	c.JSON(status, response)
}
