package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupPccAgentUserSearchTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	previousDB := model.DB
	previousMainDatabaseType := common.MainDatabaseType()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.DesktopGrant{},
		&model.ExternalIdentityClaim{},
		&model.SubscriptionPlan{},
		&model.UserSubscription{},
	))
	model.DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	t.Cleanup(func() {
		model.DB = previousDB
		common.SetMainDatabaseType(previousMainDatabaseType)
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func performUserSearchRequest(t *testing.T, target string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, target, nil)
	SearchUsers(context)
	return recorder
}

func TestSearchUsersPccAgentParameterScopesAndAddsSummary(t *testing.T) {
	db := setupPccAgentUserSearchTestDB(t)
	pccUser := model.User{
		Username: "controller-pcc-user", Password: "password123",
		Status: common.UserStatusEnabled, Role: common.RoleCommonUser, Group: "default", AffCode: "controller-pcc",
	}
	ordinaryUser := model.User{
		Username: "controller-ordinary-user", Password: "password123",
		Status: common.UserStatusEnabled, Role: common.RoleCommonUser, Group: "default", AffCode: "controller-ordinary",
	}
	require.NoError(t, db.Create(&pccUser).Error)
	require.NoError(t, db.Create(&ordinaryUser).Error)
	activeSlot := 1
	require.NoError(t, db.Create(&model.DesktopGrant{
		PublicId:     "controller-pcc-grant",
		UserId:       pccUser.Id,
		ClientId:     "pcc-agent-desktop",
		DeviceIdHash: strings.Repeat("a", 64),
		DeviceName:   "Controller Test Device",
		Scopes:       "relay account.read usage.read",
		Status:       model.DesktopGrantStatusActive,
		ActiveSlot:   &activeSlot,
		ExpiredTime:  common.GetTimestamp() + 3600,
		CreatedTime:  1_700_000_000,
	}).Error)

	response := performUserSearchRequest(
		t,
		"/api/user/search?keyword=controller-&pcc_agent=true&p=1&page_size=20&sort_by=id&sort_order=asc",
	)
	require.Equal(t, http.StatusOK, response.Code)
	var payload struct {
		Success bool `json:"success"`
		Data    struct {
			Items []model.User `json:"items"`
			Total int          `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &payload))
	assert.True(t, payload.Success)
	assert.Equal(t, 1, payload.Data.Total)
	require.Len(t, payload.Data.Items, 1)
	assert.Equal(t, pccUser.Id, payload.Data.Items[0].Id)
	require.NotNil(t, payload.Data.Items[0].PccAgentSummary)
	assert.EqualValues(t, 1, payload.Data.Items[0].PccAgentSummary.ActiveDeviceCount)

	ordinaryResponse := performUserSearchRequest(
		t,
		"/api/user/search?keyword=controller-&p=1&page_size=20&sort_by=id&sort_order=asc",
	)
	require.Equal(t, http.StatusOK, ordinaryResponse.Code)
	assert.NotContains(t, ordinaryResponse.Body.String(), "pcc_agent_summary")
}
