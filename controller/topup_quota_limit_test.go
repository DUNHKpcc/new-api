package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func useTopUpQuotaControllerTestDB(t *testing.T) {
	t.Helper()
	oldDB := model.DB
	oldMainDatabaseType := common.MainDatabaseType()
	oldRedisEnabled := common.RedisEnabled
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}))
	model.DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	common.RedisEnabled = false
	t.Cleanup(func() {
		model.DB = oldDB
		common.SetMainDatabaseType(oldMainDatabaseType)
		common.RedisEnabled = oldRedisEnabled
		sqlDB, dbErr := db.DB()
		if dbErr == nil {
			require.NoError(t, sqlDB.Close())
		}
	})
}

func useTopUpQuotaControllerSettings(t *testing.T) {
	t.Helper()
	oldQuotaPerUnit := common.QuotaPerUnit
	oldDisplayType := operation_setting.GetGeneralSetting().QuotaDisplayType
	oldPrice := operation_setting.Price
	oldMinTopUp := operation_setting.MinTopUp
	oldPaymentSetting := *operation_setting.GetPaymentSetting()
	common.QuotaPerUnit = 500000
	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeUSD
	operation_setting.Price = 1
	operation_setting.MinTopUp = 1
	operation_setting.GetPaymentSetting().AmountDiscount = map[int]float64{}
	t.Cleanup(func() {
		common.QuotaPerUnit = oldQuotaPerUnit
		operation_setting.GetGeneralSetting().QuotaDisplayType = oldDisplayType
		operation_setting.Price = oldPrice
		operation_setting.MinTopUp = oldMinTopUp
		*operation_setting.GetPaymentSetting() = oldPaymentSetting
	})
}

func performEpayAmountRequest(t *testing.T, userID int, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("id", userID)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/user/amount", strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	RequestAmount(ctx)
	return recorder
}

func TestTopUpQuotaValidation(t *testing.T) {
	useTopUpQuotaControllerSettings(t)

	testCases := []struct {
		name        string
		displayType string
		amount      int64
		wantQuota   int
		wantErr     bool
	}{
		{
			name:        "currency amount below limit",
			displayType: operation_setting.QuotaDisplayTypeUSD,
			amount:      4294,
			wantQuota:   2_147_000_000,
		},
		{
			name:        "currency amount above limit",
			displayType: operation_setting.QuotaDisplayTypeUSD,
			amount:      4295,
			wantErr:     true,
		},
		{
			name:        "token amount preserves settlement truncation",
			displayType: operation_setting.QuotaDisplayTypeTokens,
			amount:      common.MaxQuota,
			wantQuota:   2_147_000_000,
		},
		{
			name:        "token amount above settlement limit",
			displayType: operation_setting.QuotaDisplayTypeTokens,
			amount:      2_147_500_000,
			wantErr:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			operation_setting.GetGeneralSetting().QuotaDisplayType = tc.displayType
			quota, err := getTopUpQuota(tc.amount)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantQuota, quota)
		})
	}
}

func TestRequestAmountRejectsTopUpThatWouldExceedCapacity(t *testing.T) {
	useTopUpQuotaControllerTestDB(t)
	useTopUpQuotaControllerSettings(t)
	require.NoError(t, model.DB.Create(&model.User{
		Id:       42,
		Username: "topup-capacity",
		Quota:    common.MaxQuota - 100_000,
		Status:   common.UserStatusEnabled,
	}).Error)

	recorder := performEpayAmountRequest(t, 42, `{"amount":1}`)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `{"message":"error","data":"top-up quota limit exceeded"}`, recorder.Body.String())
}

func TestRequestAmountAllowsNegativeBalanceToRepayDebt(t *testing.T) {
	useTopUpQuotaControllerTestDB(t)
	useTopUpQuotaControllerSettings(t)
	require.NoError(t, model.DB.Create(&model.User{
		Id:       43,
		Username: "topup-debt",
		Quota:    -600_000,
		Status:   common.UserStatusEnabled,
	}).Error)

	recorder := performEpayAmountRequest(t, 43, `{"amount":1}`)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `{"message":"success","data":"1.00"}`, recorder.Body.String())
}

func TestValidateCreditedQuotaRejectsSaturation(t *testing.T) {
	_, err := validateCreditedQuota(decimal.NewFromInt(common.MaxQuota - 1))
	require.NoError(t, err)
	_, err = validateCreditedQuota(decimal.Zero)
	require.EqualError(t, err, "充值额度必须大于 0")
	_, err = validateCreditedQuota(decimal.NewFromInt(common.MaxQuota))
	require.EqualError(t, err, "充值额度超出系统可表示范围")
}

func TestStripeCreditedQuotaIncludesGroupRatio(t *testing.T) {
	oldQuotaPerUnit := common.QuotaPerUnit
	oldTopUpGroupRatio := common.TopupGroupRatio2JSONString()
	common.QuotaPerUnit = 500000
	require.NoError(t, common.UpdateTopupGroupRatioByJSONString(`{"vip":2}`))
	t.Cleanup(func() {
		common.QuotaPerUnit = oldQuotaPerUnit
		require.NoError(t, common.UpdateTopupGroupRatioByJSONString(oldTopUpGroupRatio))
	})

	_, err := validateCreditedQuota(getStripeCreditedQuota(2147, "vip"))
	require.NoError(t, err)
	_, err = validateCreditedQuota(getStripeCreditedQuota(2148, "vip"))
	require.Error(t, err)
}
