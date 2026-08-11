package router

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestAffiliateFinancialRoutesEnforceAuthenticationAndRootRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousDB := model.DB
	previousRedisEnabled := common.RedisEnabled
	db, err := gorm.Open(sqlite.Open("file:affiliate-route-auth?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	common.RedisEnabled = false
	t.Cleanup(func() {
		model.DB = previousDB
		common.RedisEnabled = previousRedisEnabled
	})
	require.NoError(t, db.AutoMigrate(&model.User{}))

	accessToken := "ordinary-user-access-token-000001"
	user := model.User{
		Username: "affiliate-route-user", Status: common.UserStatusEnabled,
		Role: common.RoleCommonUser, AccessToken: &accessToken,
	}
	require.NoError(t, db.Create(&user).Error)

	engine := gin.New()
	SetApiRouter(engine)
	rootOnlyRoutes := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/api/admin/affiliate/commissions", ""},
		{http.MethodPost, "/api/admin/affiliate/commissions/1/reverse", `{}`},
		{http.MethodPut, "/api/user/2/affiliate-access", `{}`},
		{http.MethodPut, "/api/option/affiliate", `{}`},
	}
	for _, route := range rootOnlyRoutes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			request := httptest.NewRequest(route.method, route.path, bytes.NewBufferString(route.body))
			request.Header.Set("Authorization", "Bearer "+accessToken)
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, request)
			assert.Equal(t, http.StatusForbidden, response.Code)
			assert.Contains(t, response.Body.String(), "AUTH_INSUFFICIENT_PRIVILEGE")
		})
	}

	for _, route := range []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/admin/affiliate/commissions"},
		{http.MethodPost, "/api/admin/affiliate/commissions/1/reverse"},
		{http.MethodPost, "/api/user/affiliate/commissions/transfer"},
	} {
		request := httptest.NewRequest(route.method, route.path, bytes.NewBufferString(`{}`))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		engine.ServeHTTP(response, request)
		assert.Equal(t, http.StatusUnauthorized, response.Code)
		assert.Contains(t, response.Body.String(), "AUTH_UNAUTHORIZED")
	}
}
