package controller

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Calcium-Ion/go-epay/epay"
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type epayNotifyFixture struct {
	db     *gorm.DB
	user   model.User
	topUp  model.TopUp
	secret string
}

func setupEpayNotifyFixture(t *testing.T) epayNotifyFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)
	previousDB, previousLogDB := model.DB, model.LOG_DB
	previousRedis := common.RedisEnabled
	previousMainType := common.MainDatabaseType()
	previousLogType := common.LogDatabaseType()
	previousPayAddress := operation_setting.PayAddress
	previousEpayID := operation_setting.EpayId
	previousEpayKey := operation_setting.EpayKey
	previousPayMethods := operation_setting.PayMethods
	previousCustomCallbackAddress := operation_setting.CustomCallbackAddress
	previousPrice := operation_setting.Price
	previousQuotaPerUnit := common.QuotaPerUnit
	previousServerAddress := system_setting.ServerAddress
	previousGeneralSetting := *operation_setting.GetGeneralSetting()
	paymentSetting := operation_setting.GetPaymentSetting()
	previousPaymentSetting := *paymentSetting

	db, err := gorm.Open(sqlite.Open("file:"+strings.ReplaceAll(t.Name(), "/", "_")+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.Option{},
		&model.TopUp{},
		&model.Log{},
		&model.AffiliateReferral{},
		&model.AffiliateSignupReward{},
		&model.AffiliateSignupRewardTransfer{},
		&model.AffiliateProfile{},
		&model.AffiliateCommission{},
		&model.AffiliateCommissionTransfer{},
		&model.AffiliateCommissionReversal{},
		&model.AffiliateAccessChange{},
		&model.AffiliateConfigChange{},
		&model.AffiliateOutboxEvent{},
	))
	model.DB = db
	model.LOG_DB = db
	common.RedisEnabled = false
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	operation_setting.PayAddress = "https://pay.example.com"
	operation_setting.EpayId = "epay-test-partner"
	operation_setting.EpayKey = "epay-test-secret"
	operation_setting.PayMethods = []map[string]string{{"type": "alipay"}}
	operation_setting.CustomCallbackAddress = ""
	operation_setting.Price = 7.3
	common.QuotaPerUnit = 500_000
	system_setting.ServerAddress = "https://console.example.com"
	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeUSD
	paymentSetting.EpayCurrency = "CNY"
	paymentSetting.AmountDiscount = map[int]float64{}
	paymentSetting.ComplianceConfirmed = true
	paymentSetting.ComplianceTermsVersion = operation_setting.CurrentComplianceTermsVersion
	require.NoError(t, model.UpdateOptionsBulk(map[string]string{
		model.PaymentComplianceConfirmedOptionKey:    "true",
		model.PaymentComplianceTermsVersionOptionKey: operation_setting.CurrentComplianceTermsVersion,
	}))

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() {
		model.DB = previousDB
		model.LOG_DB = previousLogDB
		common.RedisEnabled = previousRedis
		common.SetDatabaseTypes(previousMainType, previousLogType)
		operation_setting.PayAddress = previousPayAddress
		operation_setting.EpayId = previousEpayID
		operation_setting.EpayKey = previousEpayKey
		operation_setting.PayMethods = previousPayMethods
		operation_setting.CustomCallbackAddress = previousCustomCallbackAddress
		operation_setting.Price = previousPrice
		common.QuotaPerUnit = previousQuotaPerUnit
		system_setting.ServerAddress = previousServerAddress
		*operation_setting.GetGeneralSetting() = previousGeneralSetting
		*paymentSetting = previousPaymentSetting
		_ = sqlDB.Close()
	})

	user := model.User{Username: "epay-notify-user", AffCode: "epay-notify-user", Group: "default", Quota: 100}
	require.NoError(t, db.Create(&user).Error)
	topUp := model.TopUp{
		UserId:                  user.Id,
		Amount:                  10,
		Money:                   73,
		TradeNo:                 "epay-notify-order",
		PaymentMethod:           "alipay",
		PaymentProvider:         model.PaymentProviderEpay,
		ExpectedAmountMinor:     7300,
		PaidCurrency:            "CNY",
		QuotaAmount:             5_000_000,
		UnitPriceSnapshot:       "7.3",
		QuotaPerUnitSnapshot:    "500000",
		TopUpGroupRatioSnapshot: "1",
		AmountDiscountSnapshot:  "1",
		CommissionEligible:      true,
		Status:                  common.TopUpStatusPending,
	}
	require.NoError(t, db.Create(&topUp).Error)
	return epayNotifyFixture{db: db, user: user, topUp: topUp, secret: operation_setting.EpayKey}
}

func performEpayNotify(t *testing.T, fixture epayNotifyFixture, params map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	params = epay.GenerateParams(params, fixture.secret)
	form := url.Values{}
	for key, value := range params {
		form.Set(key, value)
	}
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/user/epay/notify", strings.NewReader(form.Encode()))
	context.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	EpayNotify(context)
	return recorder
}

func validEpayNotifyParams(fixture epayNotifyFixture) map[string]string {
	return map[string]string{
		"pid":          operation_setting.EpayId,
		"type":         "alipay",
		"out_trade_no": fixture.topUp.TradeNo,
		"trade_no":     "epay-provider-order",
		"money":        "73.00",
		"trade_status": epay.StatusTradeSuccess,
	}
}

func TestEpayNotifyFailsClosedForInvalidPaymentEvidence(t *testing.T) {
	testCases := []struct {
		name   string
		mutate func(map[string]string)
	}{
		{name: "zero amount", mutate: func(params map[string]string) { params["money"] = "0" }},
		{name: "unparseable amount", mutate: func(params map[string]string) { params["money"] = "invalid" }},
		{name: "amount mismatch", mutate: func(params map[string]string) { params["money"] = "72.99" }},
		{name: "payment method mismatch", mutate: func(params map[string]string) { params["type"] = "wxpay" }},
		{name: "merchant mismatch", mutate: func(params map[string]string) { params["pid"] = "other-merchant" }},
		{name: "missing provider trade number", mutate: func(params map[string]string) { delete(params, "trade_no") }},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := setupEpayNotifyFixture(t)
			params := validEpayNotifyParams(fixture)
			testCase.mutate(params)
			response := performEpayNotify(t, fixture, params)
			assert.Equal(t, "fail", response.Body.String())

			var topUp model.TopUp
			require.NoError(t, fixture.db.First(&topUp, fixture.topUp.Id).Error)
			assert.Equal(t, common.TopUpStatusPending, topUp.Status)
			assert.Zero(t, topUp.PaidAmountMinor)
			assert.Nil(t, topUp.ProviderTradeNo)
			var user model.User
			require.NoError(t, fixture.db.First(&user, fixture.user.Id).Error)
			assert.Equal(t, 100, user.Quota)
		})
	}
}

func TestEpayNotifyCommitsBeforeSuccessAndReplayIsIdempotent(t *testing.T) {
	fixture := setupEpayNotifyFixture(t)
	params := validEpayNotifyParams(fixture)

	first := performEpayNotify(t, fixture, params)
	assert.Equal(t, "success", first.Body.String())
	second := performEpayNotify(t, fixture, params)
	assert.Equal(t, "success", second.Body.String())
	var topupEventCount int64
	require.NoError(t, fixture.db.Model(&model.AffiliateOutboxEvent{}).
		Where("action = ?", "payment.topup_completed").
		Count(&topupEventCount).Error)
	assert.Equal(t, int64(1), topupEventCount)
	delivered, failed, err := model.DispatchAffiliateOutbox(20)
	require.NoError(t, err)
	assert.Zero(t, failed)
	assert.Positive(t, delivered)

	var topUp model.TopUp
	require.NoError(t, fixture.db.First(&topUp, fixture.topUp.Id).Error)
	assert.Equal(t, common.TopUpStatusSuccess, topUp.Status)
	assert.Positive(t, topUp.CompleteTime)
	assert.Equal(t, int64(7300), topUp.PaidAmountMinor)
	require.NotNil(t, topUp.ProviderTradeNo)
	assert.Equal(t, "epay-provider-order", *topUp.ProviderTradeNo)
	var user model.User
	require.NoError(t, fixture.db.First(&user, fixture.user.Id).Error)
	assert.Equal(t, 100+fixture.topUp.QuotaAmount, user.Quota)
	var topupLogCount int64
	require.NoError(t, fixture.db.Model(&model.Log{}).
		Where("user_id = ? AND type = ?", fixture.user.Id, model.LogTypeTopup).
		Count(&topupLogCount).Error)
	assert.Equal(t, int64(1), topupLogCount)
}

func TestEpayNotifyReturnsFailWhenAtomicSettlementRollsBack(t *testing.T) {
	fixture := setupEpayNotifyFixture(t)
	callbackName := "test:fail_epay_notify_quota_update"
	require.NoError(t, fixture.db.Callback().Update().Before("gorm:update").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement.Table == "users" {
			tx.AddError(errors.New("injected quota update failure"))
		}
	}))
	t.Cleanup(func() {
		require.NoError(t, fixture.db.Callback().Update().Remove(callbackName))
	})

	response := performEpayNotify(t, fixture, validEpayNotifyParams(fixture))
	assert.Equal(t, "fail", response.Body.String())
	var topUp model.TopUp
	require.NoError(t, fixture.db.First(&topUp, fixture.topUp.Id).Error)
	assert.Equal(t, common.TopUpStatusPending, topUp.Status)
	assert.Nil(t, topUp.ProviderTradeNo)
	var user model.User
	require.NoError(t, fixture.db.First(&user, fixture.user.Id).Error)
	assert.Equal(t, 100, user.Quota)
}

func TestEpayNotifyAcknowledgesVerifiedNonSuccessWithoutSettlement(t *testing.T) {
	fixture := setupEpayNotifyFixture(t)
	params := validEpayNotifyParams(fixture)
	params["trade_status"] = "WAIT_BUYER_PAY"

	response := performEpayNotify(t, fixture, params)
	assert.Equal(t, "success", response.Body.String())
	var topUp model.TopUp
	require.NoError(t, fixture.db.First(&topUp, fixture.topUp.Id).Error)
	assert.Equal(t, common.TopUpStatusPending, topUp.Status)
}

func TestEpayOrderToVerifiedSummaryFlowUsesImmutableSnapshots(t *testing.T) {
	fixture := setupEpayNotifyFixture(t)
	require.NoError(t, fixture.db.Delete(&fixture.topUp).Error)
	previousAffiliateSetting := operation_setting.GetAffiliateSettingSnapshot()
	affiliateSetting := operation_setting.DefaultAffiliateSetting()
	affiliateSetting.CommissionEnabled = true
	affiliateSetting.CommissionRateBPS = 5000
	affiliateSetting.CommissionWaitDays = 0
	require.NoError(t, operation_setting.SetAffiliateSetting(affiliateSetting))
	affiliateSettingJSON, err := operation_setting.MarshalAffiliateSetting(affiliateSetting)
	require.NoError(t, err)
	require.NoError(t, fixture.db.Create(&[]model.Option{
		{Key: operation_setting.AffiliateSettingOptionKey, Value: affiliateSettingJSON},
		{Key: model.EpayCurrencyOptionKey, Value: "CNY"},
	}).Error)
	t.Cleanup(func() {
		require.NoError(t, operation_setting.SetAffiliateSetting(previousAffiliateSetting))
	})
	agent := model.User{
		Username: "epay-snapshot-agent",
		AffCode:  "snapshot-agent",
		Role:     common.RoleRootUser,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, fixture.db.Create(&agent).Error)
	require.NoError(t, fixture.db.Model(&model.User{}).
		Where("id = ?", fixture.user.Id).
		Update("inviter_id", agent.Id).Error)
	require.NoError(t, fixture.db.Create(&model.AffiliateReferral{
		InviterUserId:   agent.Id,
		ReferredUserId:  fixture.user.Id,
		AffCodeSnapshot: agent.AffCode,
		BoundAt:         1,
	}).Error)
	require.NoError(t, fixture.db.Create(&model.AffiliateProfile{
		UserId:      agent.Id,
		Access:      model.AffiliateAccessInherit,
		Status:      model.AffiliateStatusActive,
		ActivatedAt: 1,
		Currency:    "CNY",
		CreatedAt:   1,
		UpdatedAt:   1,
	}).Error)
	operation_setting.GetPaymentSetting().EpayCurrency = "USD"

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Set("id", fixture.user.Id)
	context.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/user/pay",
		bytes.NewBufferString(`{"amount":10,"payment_method":"alipay"}`),
	)
	context.Request.Header.Set("Content-Type", "application/json")
	RequestEpay(context)
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"message":"success"`)

	var topUp model.TopUp
	require.NoError(t, fixture.db.Where("user_id = ?", fixture.user.Id).First(&topUp).Error)
	assert.Equal(t, int64(7300), topUp.ExpectedAmountMinor)
	assert.Equal(t, "CNY", topUp.PaidCurrency)
	assert.Equal(t, "7.3", topUp.UnitPriceSnapshot)
	assert.Equal(t, 5_000_000, topUp.QuotaAmount)
	assert.True(t, topUp.CommissionEligible)

	operation_setting.Price = 99
	common.QuotaPerUnit = 123
	operation_setting.GetPaymentSetting().EpayCurrency = "USD"
	paidMoney := decimal.NewFromInt(topUp.ExpectedAmountMinor).
		Div(decimal.NewFromInt(epayMinorUnitsPerMajor)).
		StringFixed(2)
	params := map[string]string{
		"pid":          operation_setting.EpayId,
		"type":         "alipay",
		"out_trade_no": topUp.TradeNo,
		"trade_no":     "epay-provider-snapshot-flow",
		"money":        paidMoney,
		"trade_status": epay.StatusTradeSuccess,
	}
	first := performEpayNotify(t, fixture, params)
	assert.Equal(t, "success", first.Body.String())
	second := performEpayNotify(t, fixture, params)
	assert.Equal(t, "success", second.Body.String())

	var user model.User
	require.NoError(t, fixture.db.First(&user, fixture.user.Id).Error)
	assert.Equal(t, 100+topUp.QuotaAmount, user.Quota)
	var completed model.TopUp
	require.NoError(t, fixture.db.First(&completed, topUp.Id).Error)
	assert.Equal(t, "CNY", completed.PaidCurrency)
	assert.Equal(t, "7.3", completed.UnitPriceSnapshot)
	totals, err := model.GetVerifiedTopUpPaymentTotals(fixture.user.Id, model.PaymentProviderEpay)
	require.NoError(t, err)
	require.Len(t, totals, 1)
	assert.Equal(t, "CNY", totals[0].Currency)
	assert.Equal(t, int64(7300), totals[0].AmountMinor)

	var commission model.AffiliateCommission
	require.NoError(t, fixture.db.Where("top_up_id = ?", topUp.Id).First(&commission).Error)
	assert.Equal(t, int64(3650), commission.CommissionAmountMinor)
	assert.Equal(t, topUp.QuotaAmount, commission.PurchasedQuota)
	assert.Equal(t, 2_500_000, commission.RewardQuota)
	assert.Equal(t, model.AffiliateCommissionStatusAvailable, commission.Status)

	transferRecorder := httptest.NewRecorder()
	transferContext, _ := gin.CreateTestContext(transferRecorder)
	transferContext.Set("id", agent.Id)
	transferContext.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/user/affiliate/commissions/transfer",
		bytes.NewBufferString(`{"idempotency_key":"snapshot-flow-transfer"}`),
	)
	TransferAffiliateCommissions(transferContext)
	assert.Contains(t, transferRecorder.Body.String(), `"success":true`)
	replayRecorder := httptest.NewRecorder()
	replayContext, _ := gin.CreateTestContext(replayRecorder)
	replayContext.Set("id", agent.Id)
	replayContext.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/user/affiliate/commissions/transfer",
		bytes.NewBufferString(`{"idempotency_key":"snapshot-flow-transfer"}`),
	)
	TransferAffiliateCommissions(replayContext)
	assert.Contains(t, replayRecorder.Body.String(), `"already_completed":true`)
	var reloadedAgent model.User
	require.NoError(t, fixture.db.First(&reloadedAgent, agent.Id).Error)
	assert.Equal(t, 2_500_000, reloadedAgent.Quota)

	reverseBody := `{"reason":"verified commission reversal"}`
	reverseRecorder := httptest.NewRecorder()
	reverseContext, _ := gin.CreateTestContext(reverseRecorder)
	reverseContext.Set("id", agent.Id)
	reverseContext.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", commission.Id)}}
	reverseContext.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/admin/affiliate/commissions/1/reverse",
		bytes.NewBufferString(reverseBody),
	)
	AdminReverseAffiliateCommission(reverseContext)
	assert.Contains(t, reverseRecorder.Body.String(), `"success":true`)
	var profile model.AffiliateProfile
	require.NoError(t, fixture.db.First(&profile, agent.Id).Error)
	assert.Equal(t, int64(2_500_000), profile.CommissionDebtQuota)
	require.NoError(t, fixture.db.First(&reloadedAgent, agent.Id).Error)
	assert.Equal(t, 2_500_000, reloadedAgent.Quota)
	require.NoError(t, fixture.db.First(&user, fixture.user.Id).Error)
	assert.Equal(t, 100+topUp.QuotaAmount, user.Quota)
	require.NoError(t, fixture.db.First(&completed, topUp.Id).Error)
	assert.Equal(t, common.TopUpStatusSuccess, completed.Status)
	postReversalTotals, err := model.GetVerifiedTopUpPaymentTotals(fixture.user.Id, model.PaymentProviderEpay)
	require.NoError(t, err)
	require.Len(t, postReversalTotals, 1)
	assert.Equal(t, int64(7_300), postReversalTotals[0].AmountMinor)

	replayReverseRecorder := httptest.NewRecorder()
	replayReverseContext, _ := gin.CreateTestContext(replayReverseRecorder)
	replayReverseContext.Set("id", agent.Id)
	replayReverseContext.Params = reverseContext.Params
	replayReverseContext.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/admin/affiliate/commissions/1/reverse",
		bytes.NewBufferString(reverseBody),
	)
	AdminReverseAffiliateCommission(replayReverseContext)
	assert.Contains(t, replayReverseRecorder.Body.String(), `"already_reversed":true`)
	require.NoError(t, fixture.db.First(&profile, agent.Id).Error)
	assert.Equal(t, int64(2_500_000), profile.CommissionDebtQuota)
	var reversalCount int64
	require.NoError(t, fixture.db.Model(&model.AffiliateCommissionReversal{}).Count(&reversalCount).Error)
	assert.Equal(t, int64(1), reversalCount)
}
