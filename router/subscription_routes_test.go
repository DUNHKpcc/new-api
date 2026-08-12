package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setSubscriptionPricingAccess(t *testing.T, raw string) {
	t.Helper()

	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = map[string]string{}
	}
	previous, hadPrevious := common.OptionMap["HeaderNavModules"]
	common.OptionMap["HeaderNavModules"] = raw
	common.OptionMapRWMutex.Unlock()

	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		defer common.OptionMapRWMutex.Unlock()
		if hadPrevious {
			common.OptionMap["HeaderNavModules"] = previous
			return
		}
		delete(common.OptionMap, "HeaderNavModules")
	})
}

func requestSubscriptionRoute(engine *gin.Engine, method string, path string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, nil)
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	return response
}

func TestSubscriptionPlansRouteMatchesPricingVisibility(t *testing.T) {
	gin.SetMode(gin.TestMode)
	paymentSetting := operation_setting.GetPaymentSetting()
	previousPaymentSetting := *paymentSetting
	paymentSetting.ComplianceConfirmed = false
	paymentSetting.ComplianceTermsVersion = ""
	t.Cleanup(func() {
		*paymentSetting = previousPaymentSetting
	})

	engine := gin.New()
	SetApiRouter(engine)

	t.Run("allows anonymous access when pricing is public", func(t *testing.T) {
		setSubscriptionPricingAccess(t, `{"pricing":{"enabled":true,"requireAuth":false}}`)

		response := requestSubscriptionRoute(engine, http.MethodGet, "/api/subscription/plans")

		assert.Equal(t, http.StatusOK, response.Code)
		assert.Contains(t, response.Body.String(), `"success":true`)
	})

	t.Run("requires authentication when pricing requires login", func(t *testing.T) {
		setSubscriptionPricingAccess(t, `{"pricing":{"enabled":true,"requireAuth":true}}`)

		response := requestSubscriptionRoute(engine, http.MethodGet, "/api/subscription/plans")

		require.Equal(t, http.StatusUnauthorized, response.Code)
		assert.Contains(t, response.Body.String(), "AUTH_UNAUTHORIZED")
	})

	t.Run("keeps every subscription payment route authenticated", func(t *testing.T) {
		setSubscriptionPricingAccess(t, `{"pricing":{"enabled":true,"requireAuth":false}}`)
		paymentPaths := []string{
			"/api/subscription/balance/pay",
			"/api/subscription/epay/pay",
			"/api/subscription/stripe/pay",
			"/api/subscription/creem/pay",
			"/api/subscription/waffo-pancake/pay",
		}

		for _, path := range paymentPaths {
			response := requestSubscriptionRoute(engine, http.MethodPost, path)

			assert.Equal(t, http.StatusUnauthorized, response.Code, path)
			assert.Contains(t, response.Body.String(), "AUTH_UNAUTHORIZED", path)
		}
	})
}
