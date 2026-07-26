package service

import (
	"errors"
	"net/http"
	"os"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDesktopErrorContractMapping(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		code   DesktopErrorCode
	}{
		{name: "invalid request", err: ErrDesktopInvalidRequest, status: http.StatusBadRequest, code: DesktopErrorInvalidRequest},
		{name: "invalid token", err: ErrDesktopTokenInvalid, status: http.StatusUnauthorized, code: DesktopErrorTokenInvalid},
		{name: "scope denied", err: ErrDesktopScopeDenied, status: http.StatusForbidden, code: DesktopErrorScopeDenied},
		{name: "request expired", err: ErrDesktopRequestExpired, status: http.StatusGone, code: DesktopErrorRequestExpired},
		{name: "request consumed", err: ErrDesktopRequestConsumed, status: http.StatusConflict, code: DesktopErrorRequestConsumed},
		{name: "device limit", err: ErrDesktopDeviceLimit, status: http.StatusConflict, code: DesktopErrorDeviceLimit},
		{name: "confirmation invalid", err: ErrDesktopConfirmationInvalid, status: http.StatusBadRequest, code: DesktopErrorConfirmationInvalid},
		{name: "confirmation expired", err: ErrDesktopConfirmationExpired, status: http.StatusGone, code: DesktopErrorConfirmationExpired},
		{name: "not found", err: model.ErrDesktopGrantNotFound, status: http.StatusNotFound, code: DesktopErrorNotFound},
		{name: "wrapped", err: errors.Join(errors.New("context"), ErrDesktopTokenInvalid), status: http.StatusUnauthorized, code: DesktopErrorTokenInvalid},
		{name: "internal", err: errors.New("database unavailable"), status: http.StatusInternalServerError, code: DesktopErrorInternal},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			status, response := DesktopErrorFor(test.err)
			assert.Equal(t, test.status, status)
			assert.Equal(t, test.code, response.Error)
			assert.Equal(t, DesktopContractVersion, response.ContractVersion)
			assert.NotEmpty(t, response.ErrorDescription)
		})
	}
}

func TestDesktopContractSchemaMatchesServerConstants(t *testing.T) {
	data, err := os.ReadFile("../docs/contracts/desktop-auth-v2.schema.json")
	require.NoError(t, err)
	var schema struct {
		ContractVersion int                `json:"contract_version"`
		ErrorCodes      []DesktopErrorCode `json:"error_codes"`
	}
	require.NoError(t, common.Unmarshal(data, &schema))

	serverCodes := []DesktopErrorCode{
		DesktopErrorInternal,
		DesktopErrorUnsupportedClient,
		DesktopErrorInvalidRedirect,
		DesktopErrorInvalidPKCE,
		DesktopErrorInvalidRequest,
		DesktopErrorBrowserSession,
		DesktopErrorRequestExpired,
		DesktopErrorRequestConsumed,
		DesktopErrorDeviceLimit,
		DesktopErrorGroupUnavailable,
		DesktopErrorTokenInvalid,
		DesktopErrorScopeDenied,
		DesktopErrorConfirmationInvalid,
		DesktopErrorConfirmationExpired,
		DesktopErrorNotFound,
		DesktopErrorGrantNotFound,
	}
	assert.Equal(t, DesktopContractVersion, schema.ContractVersion)
	assert.ElementsMatch(t, serverCodes, schema.ErrorCodes)
	assert.Len(t, schema.ErrorCodes, len(serverCodes), "schema error codes must remain unique")
}
