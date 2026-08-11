package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func useAffiliateControllerSetting(t *testing.T, mutate func(*operation_setting.AffiliateSetting)) operation_setting.AffiliateSetting {
	t.Helper()
	previous := operation_setting.GetAffiliateSettingSnapshot()
	setting := operation_setting.DefaultAffiliateSetting()
	mutate(&setting)
	require.NoError(t, operation_setting.SetAffiliateSetting(setting))
	t.Cleanup(func() { require.NoError(t, operation_setting.SetAffiliateSetting(previous)) })
	return setting
}

func TestGetAffiliateOverviewReturnsStringAmountsAndLedgerCounts(t *testing.T) {
	fixture := setupEpayNotifyFixture(t)
	setting := useAffiliateControllerSetting(t, func(setting *operation_setting.AffiliateSetting) {
		setting.RegistrationRewardEnabled = true
		setting.InviterRewardQuota = 500
		setting.InviteeRewardQuota = 100
		setting.CommissionEnabled = true
		setting.QualificationThresholdMinor = 10_000
		setting.CommissionRateBPS = 2_500
	})
	encodedSetting, err := operation_setting.MarshalAffiliateSetting(setting)
	require.NoError(t, err)
	require.NoError(t, fixture.db.Create(&[]model.Option{
		{Key: operation_setting.AffiliateSettingOptionKey, Value: encodedSetting},
		{Key: model.EpayCurrencyOptionKey, Value: "CNY"},
	}).Error)
	require.NoError(t, fixture.db.Model(&model.User{}).
		Where("id = ?", fixture.user.Id).
		Updates(map[string]interface{}{"aff_quota": 500, "aff_history": 900}).Error)
	referred := model.User{
		Username:  "affiliate-overview-referred",
		AffCode:   "overview-referred",
		Status:    common.UserStatusEnabled,
		InviterId: fixture.user.Id,
	}
	require.NoError(t, fixture.db.Create(&referred).Error)
	require.NoError(t, fixture.db.Create(&model.AffiliateReferral{
		InviterUserId: fixture.user.Id, ReferredUserId: referred.Id,
		AffCodeSnapshot: fixture.user.AffCode, BoundAt: 100,
	}).Error)
	ownProviderTrade := "overview-own-provider"
	referredProviderTrade := "overview-referred-provider"
	require.NoError(t, fixture.db.Create(&[]model.TopUp{
		{
			UserId: fixture.user.Id, TradeNo: "overview-own-topup", PaymentProvider: model.PaymentProviderEpay,
			Status: common.TopUpStatusSuccess, CompletionSource: model.CompletionSourceWebhook,
			PaidAmountMinor: 12_000, PaidCurrency: "CNY", ProviderTradeNo: &ownProviderTrade,
		},
		{
			UserId: referred.Id, TradeNo: "overview-referred-topup", PaymentProvider: model.PaymentProviderEpay,
			Status: common.TopUpStatusSuccess, CompletionSource: model.CompletionSourceWebhook,
			PaidAmountMinor: 5_000, PaidCurrency: "CNY", ProviderTradeNo: &referredProviderTrade,
		},
	}).Error)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Set("id", fixture.user.Id)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/user/affiliate/overview", nil)
	GetAffiliateOverview(context)

	var response struct {
		Success bool `json:"success"`
		Data    struct {
			Invite struct {
				Link  string `json:"link"`
				Count int64  `json:"count"`
			} `json:"invite"`
			SignupRewards struct {
				InviterRewardQuota string `json:"inviter_reward_quota"`
				AvailableQuota     string `json:"available_quota"`
				LifetimeQuota      string `json:"lifetime_quota"`
			} `json:"signup_rewards"`
			Program struct {
				Eligible             bool   `json:"eligible"`
				VerifiedAmountMinor  string `json:"verified_amount_minor"`
				RemainingAmountMinor string `json:"remaining_amount_minor"`
				Currency             string `json:"currency"`
			} `json:"program"`
			Commissions struct {
				ReferredPaidAmountMinor string `json:"referred_paid_amount_minor"`
			} `json:"commissions"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.True(t, response.Success)
	assert.Equal(t, int64(1), response.Data.Invite.Count)
	assert.Contains(t, response.Data.Invite.Link, "/sign-up?aff=")
	assert.Equal(t, "500", response.Data.SignupRewards.InviterRewardQuota)
	assert.Equal(t, "500", response.Data.SignupRewards.AvailableQuota)
	assert.Equal(t, "900", response.Data.SignupRewards.LifetimeQuota)
	assert.True(t, response.Data.Program.Eligible)
	assert.Equal(t, "12000", response.Data.Program.VerifiedAmountMinor)
	assert.Equal(t, "0", response.Data.Program.RemainingAmountMinor)
	assert.Equal(t, "CNY", response.Data.Program.Currency)
	assert.Equal(t, "5000", response.Data.Commissions.ReferredPaidAmountMinor)
}

func TestListAffiliateCommissionsReturnsReversalReasonWithoutPrivateFields(t *testing.T) {
	fixture := setupEpayNotifyFixture(t)
	referred := model.User{Username: "affiliate-private-referred", AffCode: "private-referred"}
	require.NoError(t, fixture.db.Create(&referred).Error)
	const commissionId = int64(9_007_199_254_740_993)
	commission := model.AffiliateCommission{
		Id: commissionId, AgentUserId: fixture.user.Id, ReferredUserId: referred.Id, TopUpId: fixture.topUp.Id,
		ProviderTradeNo: "private-provider-trade-number", PaidAmountMinor: 5000, PaidCurrency: "CNY",
		PurchasedQuota: 5_000_000, UnitPriceSnapshot: "7.3", QuotaPerUnitSnapshot: "500000", CommissionRateBPS: 5000,
		CommissionAmountMinor: 2500, GrossRewardQuota: 1_712_328, RewardQuota: 1_712_328,
		Status: model.AffiliateCommissionStatusReversed, AvailableAt: 9999999999,
		ReversedAt: 200, ReversedBy: 77, ReverseReason: "payment disputed", ConfigVersion: 1, CreatedAt: 100,
	}
	require.NoError(t, fixture.db.Create(&commission).Error)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Set("id", fixture.user.Id)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/user/affiliate/commissions", nil)
	ListAffiliateCommissions(context)
	body := recorder.Body.String()
	assert.Contains(t, body, `"id":"9007199254740993"`)
	assert.Contains(t, body, `"purchased_quota":"5000000"`)
	assert.Contains(t, body, `"reverse_reason":"payment disputed"`)
	assert.Contains(t, body, `"referred_user":"af***ed"`)
	assert.NotContains(t, body, referred.Username)
	assert.NotContains(t, body, "agent_user_id")
	assert.NotContains(t, body, "referred_user_id")
	assert.NotContains(t, body, "provider_trade_no")
	assert.NotContains(t, body, "private-provider-trade-number")
	assert.NotContains(t, body, `"topup_id"`)
	assert.NotContains(t, body, `"reversed_by"`)
}

func TestAdminListAffiliateCommissionsRejectsMalformedFilters(t *testing.T) {
	setupEpayNotifyFixture(t)

	tests := []string{
		"/api/admin/affiliate/commissions?agent_user_id=not-a-user",
		"/api/admin/affiliate/commissions?referred_user_id=-1",
		"/api/admin/affiliate/commissions?status=unknown",
	}
	for _, target := range tests {
		t.Run(target, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(recorder)
			context.Request = httptest.NewRequest(http.MethodGet, target, nil)
			AdminListAffiliateCommissions(context)

			assert.Contains(t, recorder.Body.String(), `"success":false`)
			assert.NotContains(t, recorder.Body.String(), `"items"`)
		})
	}
}

func TestAdminListAffiliateCommissionsUsesStringFinancialFieldsAndHidesProviderTrade(t *testing.T) {
	fixture := setupEpayNotifyFixture(t)
	const commissionId = int64(9_007_199_254_740_993)
	transferId := int64(9_007_199_254_740_995)
	commission := model.AffiliateCommission{
		Id: commissionId, AgentUserId: fixture.user.Id, ReferredUserId: 32, TopUpId: fixture.topUp.Id,
		ProviderTradeNo: "admin-private-provider-trade", PaidAmountMinor: 9_007_199_254_740_994, PaidCurrency: "CNY",
		PurchasedQuota: 2_000_000_002, UnitPriceSnapshot: "7.3", QuotaPerUnitSnapshot: "500000", CommissionRateBPS: 2500,
		CommissionAmountMinor: 2_251_799_813_685_248, GrossRewardQuota: 2_000_000_001,
		DebtOffsetQuota: 123, RewardQuota: 1_999_999_878, Status: model.AffiliateCommissionStatusReversed,
		AvailableAt: 101, TransferId: &transferId, TransferredAt: 102, ReversedAt: 103,
		ReversedBy: 42, ReverseReason: "chargeback evidence", ConfigVersion: 7, CreatedAt: 100,
	}
	require.NoError(t, fixture.db.Create(&commission).Error)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/admin/affiliate/commissions", nil)
	AdminListAffiliateCommissions(context)

	var response struct {
		Success bool                                 `json:"success"`
		Data    adminAffiliateCommissionPageResponse `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Len(t, response.Data.Items, 1)
	item := response.Data.Items[0]
	assert.Equal(t, "9007199254740993", item.Id)
	assert.Equal(t, "9007199254740994", item.PaidAmountMinor)
	assert.Equal(t, "2000000002", item.PurchasedQuota)
	assert.Equal(t, "2251799813685248", item.CommissionAmountMinor)
	assert.Equal(t, "2000000001", item.GrossRewardQuota)
	assert.Equal(t, "123", item.DebtOffsetQuota)
	assert.Equal(t, "1999999878", item.RewardQuota)
	require.NotNil(t, item.TransferId)
	assert.Equal(t, "9007199254740995", *item.TransferId)
	assert.Equal(t, int64(103), item.ReversedAt)
	assert.Equal(t, 42, item.ReversedBy)
	assert.Equal(t, "chargeback evidence", item.ReverseReason)
	assert.NotContains(t, recorder.Body.String(), "provider_trade_no")
	assert.NotContains(t, recorder.Body.String(), "admin-private-provider-trade")
}

func TestUpdateAffiliateSettingUsesVersionCASAndSingleAuditFact(t *testing.T) {
	fixture := setupEpayNotifyFixture(t)
	current := useAffiliateControllerSetting(t, func(_ *operation_setting.AffiliateSetting) {})
	encoded, err := operation_setting.MarshalAffiliateSetting(current)
	require.NoError(t, err)
	require.NoError(t, fixture.db.Create(&model.Option{
		Key: operation_setting.AffiliateSettingOptionKey, Value: encoded,
	}).Error)
	next := current
	next.CommissionEnabled = true
	next.CommissionRateBPS = 3000
	next.QualificationThresholdMinor = 9_007_199_254_740_993
	payload, err := common.Marshal(affiliateSettingPayload{
		RegistrationRewardEnabled:   next.RegistrationRewardEnabled,
		InviterRewardQuota:          next.InviterRewardQuota,
		InviteeRewardQuota:          next.InviteeRewardQuota,
		CommissionEnabled:           next.CommissionEnabled,
		QualificationThresholdMinor: strconv.FormatInt(next.QualificationThresholdMinor, 10),
		CommissionRateBPS:           next.CommissionRateBPS,
		CommissionWaitDays:          next.CommissionWaitDays,
		Version:                     next.Version,
	})
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Set("id", fixture.user.Id)
	context.Request = httptest.NewRequest(http.MethodPut, "/api/option/affiliate", bytes.NewReader(payload))
	UpdateAffiliateSetting(context)
	assert.Contains(t, recorder.Body.String(), `"success":true`)
	assert.Contains(t, recorder.Body.String(), `"version":2`)
	assert.Contains(t, recorder.Body.String(), `"qualification_threshold_minor":"9007199254740993"`)

	staleRecorder := httptest.NewRecorder()
	staleContext, _ := gin.CreateTestContext(staleRecorder)
	staleContext.Set("id", fixture.user.Id)
	staleContext.Request = httptest.NewRequest(http.MethodPut, "/api/option/affiliate", bytes.NewReader(payload))
	UpdateAffiliateSetting(staleContext)
	assert.Contains(t, staleRecorder.Body.String(), "请刷新后重试")
	var changes int64
	require.NoError(t, fixture.db.Model(&model.AffiliateConfigChange{}).Count(&changes).Error)
	assert.Equal(t, int64(1), changes)
}

func TestUpdateAffiliateAccessPersistsSuspensionAndReason(t *testing.T) {
	fixture := setupEpayNotifyFixture(t)
	target := model.User{Username: "affiliate-access-target", AffCode: "affiliate-access-target"}
	require.NoError(t, fixture.db.Create(&target).Error)
	require.NoError(t, fixture.db.Create(&model.AffiliateProfile{
		UserId: target.Id, Access: model.AffiliateAccessInherit, Status: model.AffiliateStatusActive,
		ActivatedAt: 100, Currency: "CNY", CreatedAt: 100, UpdatedAt: 100,
	}).Error)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Set("id", fixture.user.Id)
	context.Params = gin.Params{{Key: "id", Value: strconv.Itoa(target.Id)}}
	context.Request = httptest.NewRequest(
		http.MethodPut,
		"/api/user/"+strconv.Itoa(target.Id)+"/affiliate-access",
		strings.NewReader(`{"access":"deny","reason":"policy review"}`),
	)
	UpdateAffiliateAccess(context)
	assert.Contains(t, recorder.Body.String(), `"success":true`)
	var profile model.AffiliateProfile
	require.NoError(t, fixture.db.First(&profile, target.Id).Error)
	assert.Equal(t, model.AffiliateAccessDeny, profile.Access)
	assert.Equal(t, model.AffiliateStatusSuspended, profile.Status)
	var change model.AffiliateAccessChange
	require.NoError(t, fixture.db.First(&change).Error)
	assert.Equal(t, "policy review", change.Reason)
}
