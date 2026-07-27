package service

import (
	"errors"
	"net/http"

	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
)

const (
	DesktopContractVersion       = 2
	DesktopContractVersionHeader = "X-PccAgent-Contract-Version"
)

type DesktopErrorCode string

const (
	DesktopErrorInternal            DesktopErrorCode = "DESKTOP_INTERNAL_ERROR"
	DesktopErrorUnsupportedClient   DesktopErrorCode = "DESKTOP_UNSUPPORTED_CLIENT"
	DesktopErrorInvalidRedirect     DesktopErrorCode = "DESKTOP_INVALID_REDIRECT_URI"
	DesktopErrorInvalidPKCE         DesktopErrorCode = "DESKTOP_INVALID_PKCE"
	DesktopErrorInvalidRequest      DesktopErrorCode = "DESKTOP_INVALID_REQUEST"
	DesktopErrorBrowserSession      DesktopErrorCode = "DESKTOP_BROWSER_SESSION_REQUIRED"
	DesktopErrorRequestExpired      DesktopErrorCode = "DESKTOP_REQUEST_EXPIRED"
	DesktopErrorRequestConsumed     DesktopErrorCode = "DESKTOP_REQUEST_CONSUMED"
	DesktopErrorDeviceLimit         DesktopErrorCode = "DESKTOP_DEVICE_LIMIT"
	DesktopErrorGroupUnavailable    DesktopErrorCode = "DESKTOP_TOKEN_GROUP_UNAVAILABLE"
	DesktopErrorWeChatRequired      DesktopErrorCode = "DESKTOP_WECHAT_VERIFICATION_REQUIRED"
	DesktopErrorGiftAlreadyClaimed  DesktopErrorCode = "DESKTOP_GIFT_ALREADY_CLAIMED"
	DesktopErrorTokenInvalid        DesktopErrorCode = "DESKTOP_TOKEN_INVALID"
	DesktopErrorScopeDenied         DesktopErrorCode = "DESKTOP_SCOPE_DENIED"
	DesktopErrorConfirmationInvalid DesktopErrorCode = "DESKTOP_CONFIRMATION_INVALID"
	DesktopErrorConfirmationExpired DesktopErrorCode = "DESKTOP_CONFIRMATION_EXPIRED"
	DesktopErrorNotFound            DesktopErrorCode = "DESKTOP_NOT_FOUND"
	DesktopErrorGrantNotFound       DesktopErrorCode = "DESKTOP_GRANT_NOT_FOUND"
)

type DesktopErrorResponse struct {
	ContractVersion  int              `json:"contract_version"`
	Error            DesktopErrorCode `json:"error"`
	ErrorDescription string           `json:"error_description"`
}

func DesktopErrorFor(err error) (int, DesktopErrorResponse) {
	status := http.StatusInternalServerError
	code := DesktopErrorInternal
	switch {
	case errors.Is(err, ErrDesktopUnsupportedClient):
		status, code = http.StatusBadRequest, DesktopErrorUnsupportedClient
	case errors.Is(err, ErrDesktopInvalidRedirect):
		status, code = http.StatusBadRequest, DesktopErrorInvalidRedirect
	case errors.Is(err, ErrDesktopInvalidPKCE):
		status, code = http.StatusBadRequest, DesktopErrorInvalidPKCE
	case errors.Is(err, ErrDesktopInvalidRequest):
		status, code = http.StatusBadRequest, DesktopErrorInvalidRequest
	case errors.Is(err, ErrDesktopBrowserSession):
		status, code = http.StatusForbidden, DesktopErrorBrowserSession
	case errors.Is(err, ErrDesktopRequestExpired):
		status, code = http.StatusGone, DesktopErrorRequestExpired
	case errors.Is(err, ErrDesktopRequestConsumed):
		status, code = http.StatusConflict, DesktopErrorRequestConsumed
	case errors.Is(err, ErrDesktopDeviceLimit):
		status, code = http.StatusConflict, DesktopErrorDeviceLimit
	case errors.Is(err, ErrDesktopGroupUnavailable):
		status, code = http.StatusConflict, DesktopErrorGroupUnavailable
	case errors.Is(err, ErrDesktopWeChatRequired):
		status, code = http.StatusForbidden, DesktopErrorWeChatRequired
	case errors.Is(err, ErrDesktopGiftAlreadyClaimed):
		status, code = http.StatusConflict, DesktopErrorGiftAlreadyClaimed
	case errors.Is(err, ErrDesktopTokenInvalid):
		status, code = http.StatusUnauthorized, DesktopErrorTokenInvalid
	case errors.Is(err, ErrDesktopScopeDenied):
		status, code = http.StatusForbidden, DesktopErrorScopeDenied
	case errors.Is(err, ErrDesktopConfirmationInvalid):
		status, code = http.StatusBadRequest, DesktopErrorConfirmationInvalid
	case errors.Is(err, ErrDesktopConfirmationExpired):
		status, code = http.StatusGone, DesktopErrorConfirmationExpired
	case errors.Is(err, model.ErrDesktopGrantNotFound), errors.Is(err, gorm.ErrRecordNotFound):
		status, code = http.StatusNotFound, DesktopErrorNotFound
	}
	return status, DesktopErrorResponse{
		ContractVersion:  DesktopContractVersion,
		Error:            code,
		ErrorDescription: http.StatusText(status),
	}
}
