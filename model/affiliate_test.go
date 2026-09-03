package model

import (
	"errors"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func useAffiliateSettingForTopUpTest(t *testing.T, waitDays int64) {
	t.Helper()
	previousAffiliate := operation_setting.GetAffiliateSettingSnapshot()
	previousPayment := *operation_setting.GetPaymentSetting()
	setting := operation_setting.DefaultAffiliateSetting()
	setting.CommissionEnabled = true
	setting.QualificationThresholdMinor = 10_000
	setting.CommissionRateBPS = 5_000
	setting.CommissionWaitDays = waitDays
	require.NoError(t, operation_setting.SetAffiliateSetting(setting))
	encoded, err := operation_setting.MarshalAffiliateSetting(setting)
	require.NoError(t, err)
	require.NoError(t, UpdateOption(operation_setting.AffiliateSettingOptionKey, encoded))
	operation_setting.GetPaymentSetting().ComplianceConfirmed = true
	operation_setting.GetPaymentSetting().ComplianceTermsVersion = operation_setting.CurrentComplianceTermsVersion
	setAffiliatePaymentComplianceOptions(t, true)
	t.Cleanup(func() {
		require.NoError(t, operation_setting.SetAffiliateSetting(previousAffiliate))
		*operation_setting.GetPaymentSetting() = previousPayment
	})
}

func createAffiliateUsersAndProfile(t *testing.T, activatedAt int64) (User, User) {
	t.Helper()
	agent := User{Username: "affiliate-agent", AffCode: "agent-code", Status: common.UserStatusEnabled, Quota: 10}
	referred := User{Username: "affiliate-referred", AffCode: "referred-code", Status: common.UserStatusEnabled, Quota: 20}
	require.NoError(t, DB.Create(&agent).Error)
	referred.InviterId = agent.Id
	require.NoError(t, DB.Create(&referred).Error)
	require.NoError(t, DB.Create(&AffiliateReferral{
		InviterUserId:   agent.Id,
		ReferredUserId:  referred.Id,
		AffCodeSnapshot: agent.AffCode,
		BoundAt:         activatedAt - 10,
	}).Error)
	require.NoError(t, DB.Create(&AffiliateProfile{
		UserId:      agent.Id,
		Access:      AffiliateAccessInherit,
		Status:      AffiliateStatusActive,
		ActivatedAt: activatedAt,
		Currency:    "CNY",
		CreatedAt:   activatedAt,
		UpdatedAt:   activatedAt,
	}).Error)
	return agent, referred
}

func createAffiliateEpayOrder(t *testing.T, referred User, tradeNo string, expectedMinor int64) TopUp {
	t.Helper()
	order := TopUp{
		UserId:                  referred.Id,
		Amount:                  expectedMinor / 100,
		Money:                   float64(expectedMinor) / 100,
		TradeNo:                 tradeNo,
		PaymentMethod:           "alipay",
		PaymentProvider:         PaymentProviderEpay,
		ExpectedAmountMinor:     expectedMinor,
		PaidCurrency:            "CNY",
		QuotaAmount:             1_000,
		UnitPriceSnapshot:       "1",
		QuotaPerUnitSnapshot:    "100",
		TopUpGroupRatioSnapshot: "1",
		AmountDiscountSnapshot:  "1",
		CommissionEligible:      true,
		CreateTime:              100,
		Status:                  common.TopUpStatusPending,
	}
	require.NoError(t, DB.Create(&order).Error)
	return order
}

func TestActivateAffiliateUsesOnlyVerifiedWebhookPaymentsAndIsIdempotent(t *testing.T) {
	truncateTables(t)
	useAffiliateSettingForTopUpTest(t, 7)
	user := User{Username: "affiliate-activate", AffCode: "activate-code", Status: common.UserStatusEnabled}
	require.NoError(t, DB.Create(&user).Error)
	webhookTrade := "verified-activation-payment"
	adminTrade := "admin-activation-payment"
	topUps := []TopUp{
		{
			UserId: user.Id, TradeNo: "activation-webhook", PaymentProvider: PaymentProviderEpay,
			Status: common.TopUpStatusSuccess, CompletionSource: CompletionSourceWebhook,
			PaidAmountMinor: 10_000, PaidCurrency: "CNY", ProviderTradeNo: &webhookTrade,
		},
		{
			UserId: user.Id, TradeNo: "activation-admin", PaymentProvider: PaymentProviderEpay,
			Status: common.TopUpStatusSuccess, CompletionSource: CompletionSourceAdmin,
			PaidAmountMinor: 99_000, PaidCurrency: "CNY", ProviderTradeNo: &adminTrade,
		},
	}
	require.NoError(t, DB.Create(&topUps).Error)

	first, err := ActivateAffiliate(user.Id)
	require.NoError(t, err)
	assert.False(t, first.AlreadyActive)
	assert.Equal(t, int64(10_000), first.VerifiedAmountMinor)
	assert.Equal(t, AffiliateStatusActive, first.Profile.Status)

	setting := operation_setting.GetAffiliateSettingSnapshot()
	setting.QualificationThresholdMinor = 1_000_000
	encoded, err := operation_setting.MarshalAffiliateSetting(setting)
	require.NoError(t, err)
	require.NoError(t, UpdateOption(operation_setting.AffiliateSettingOptionKey, encoded))
	second, err := ActivateAffiliate(user.Id)
	require.NoError(t, err)
	assert.True(t, second.AlreadyActive)
	assert.Equal(t, first.Profile.ActivatedAt, second.Profile.ActivatedAt)

	var eventCount int64
	require.NoError(t, DB.Model(&AffiliateOutboxEvent{}).
		Where("action = ?", "affiliate.activated").Count(&eventCount).Error)
	assert.Equal(t, int64(1), eventCount)
}

func TestActivateAffiliateAllowBypassesThresholdAndDenyBlocks(t *testing.T) {
	truncateTables(t)
	useAffiliateSettingForTopUpTest(t, 7)
	root := User{Username: "affiliate-root", AffCode: "affiliate-root", Status: common.UserStatusEnabled}
	allowed := User{Username: "affiliate-allowed", AffCode: "affiliate-allowed", Status: common.UserStatusEnabled}
	denied := User{Username: "affiliate-denied", AffCode: "affiliate-denied", Status: common.UserStatusEnabled}
	require.NoError(t, DB.Create(&root).Error)
	require.NoError(t, DB.Create(&allowed).Error)
	require.NoError(t, DB.Create(&denied).Error)
	_, err := SetAffiliateAccess(allowed.Id, root.Id, AffiliateAccessAllow, "approved")
	require.NoError(t, err)
	_, err = SetAffiliateAccess(denied.Id, root.Id, AffiliateAccessDeny, "blocked")
	require.NoError(t, err)

	result, err := ActivateAffiliate(allowed.Id)
	require.NoError(t, err)
	assert.Equal(t, AffiliateStatusActive, result.Profile.Status)
	_, err = ActivateAffiliate(denied.Id)
	assert.ErrorIs(t, err, ErrAffiliateAccessDenied)
}

func TestSetAffiliateAccessInheritReappliesCurrentQualificationThreshold(t *testing.T) {
	truncateTables(t)
	useAffiliateSettingForTopUpTest(t, 7)
	root := User{Username: "affiliate-access-root", AffCode: "affiliate-access-root", Status: common.UserStatusEnabled}
	belowThreshold := User{Username: "affiliate-access-below", AffCode: "affiliate-access-below", Status: common.UserStatusEnabled}
	qualified := User{Username: "affiliate-access-qualified", AffCode: "affiliate-access-qualified", Status: common.UserStatusEnabled}
	alreadyInherited := User{Username: "affiliate-access-inherited", AffCode: "affiliate-access-inherited", Status: common.UserStatusEnabled}
	suspended := User{Username: "affiliate-access-suspended", AffCode: "affiliate-access-suspended", Status: common.UserStatusEnabled}
	require.NoError(t, DB.Create(&root).Error)
	require.NoError(t, DB.Create(&belowThreshold).Error)
	require.NoError(t, DB.Create(&qualified).Error)
	require.NoError(t, DB.Create(&alreadyInherited).Error)
	require.NoError(t, DB.Create(&suspended).Error)
	require.NoError(t, DB.Create(&[]AffiliateProfile{
		{
			UserId: belowThreshold.Id, Access: AffiliateAccessAllow, Status: AffiliateStatusActive,
			ActivatedAt: 100, QualifiedAmountMinor: 0, Currency: "CNY", CreatedAt: 100, UpdatedAt: 100,
		},
		{
			UserId: qualified.Id, Access: AffiliateAccessAllow, Status: AffiliateStatusActive,
			ActivatedAt: 200, QualifiedAmountMinor: 10_000, Currency: "CNY", CreatedAt: 200, UpdatedAt: 200,
		},
		{
			UserId: alreadyInherited.Id, Access: AffiliateAccessInherit, Status: AffiliateStatusActive,
			ActivatedAt: 300, QualifiedAmountMinor: 1_000, Currency: "CNY", CreatedAt: 300, UpdatedAt: 300,
		},
		{
			UserId: suspended.Id, Access: AffiliateAccessDeny, Status: AffiliateStatusSuspended,
			ActivatedAt: 400, QualifiedAmountMinor: 0, Currency: "CNY", CreatedAt: 400, UpdatedAt: 400,
		},
	}).Error)
	providerTradeNo := "affiliate-access-qualified-provider"
	require.NoError(t, DB.Create(&TopUp{
		UserId: qualified.Id, TradeNo: "affiliate-access-qualified-order",
		PaymentProvider: PaymentProviderEpay, Status: common.TopUpStatusSuccess,
		CompletionSource: CompletionSourceWebhook, PaidAmountMinor: 10_000,
		PaidCurrency: "CNY", ProviderTradeNo: &providerTradeNo,
	}).Error)

	locked, err := SetAffiliateAccess(
		belowThreshold.Id,
		root.Id,
		AffiliateAccessInherit,
		"return to system threshold",
	)
	require.NoError(t, err)
	assert.Equal(t, AffiliateStatusInactive, locked.Profile.Status)
	assert.Zero(t, locked.Profile.ActivatedAt)
	assert.Zero(t, locked.Profile.QualifiedAmountMinor)
	assert.Empty(t, locked.Profile.Currency)
	var accessEvent AffiliateOutboxEvent
	require.NoError(t, DB.Where("action = ?", "affiliate.access_changed").First(&accessEvent).Error)
	assert.Contains(t, accessEvent.Payload, `"qualification_recalculated":true`)
	assert.Contains(t, accessEvent.Payload, `"activation_reset":true`)
	assert.Contains(t, accessEvent.Payload, `"qualification_threshold_minor":"10000"`)
	assert.Contains(t, accessEvent.Payload, `"verified_amount_minor":"0"`)
	_, err = ActivateAffiliate(belowThreshold.Id)
	assert.ErrorIs(t, err, ErrAffiliateQualificationNotMet)

	keptActive, err := SetAffiliateAccess(
		qualified.Id,
		root.Id,
		AffiliateAccessInherit,
		"use system threshold",
	)
	require.NoError(t, err)
	assert.Equal(t, AffiliateStatusActive, keptActive.Profile.Status)
	assert.Equal(t, int64(200), keptActive.Profile.ActivatedAt)

	unchanged, err := SetAffiliateAccess(
		alreadyInherited.Id,
		root.Id,
		AffiliateAccessInherit,
		"keep existing system access",
	)
	require.NoError(t, err)
	assert.Equal(t, AffiliateStatusActive, unchanged.Profile.Status)
	assert.Equal(t, int64(300), unchanged.Profile.ActivatedAt)

	restoredToThreshold, err := SetAffiliateAccess(
		suspended.Id,
		root.Id,
		AffiliateAccessInherit,
		"restore system threshold",
	)
	require.NoError(t, err)
	assert.Equal(t, AffiliateStatusInactive, restoredToThreshold.Profile.Status)
	assert.Zero(t, restoredToThreshold.Profile.ActivatedAt)
}

func TestCompleteEpayTopUpCreatesOneCommissionFromImmutableSnapshots(t *testing.T) {
	truncateTables(t)
	useAffiliateSettingForTopUpTest(t, 7)
	agent, referred := createAffiliateUsersAndProfile(t, 100)
	order := createAffiliateEpayOrder(t, referred, "affiliate-epay-one", 5_000)
	input := EpaySettlement{
		TradeNo: order.TradeNo, ProviderTradeNo: "epay-provider-one", PaidAmountMinor: 5_000,
		PaymentMethod: "alipay", CompletedAt: 200,
	}

	first, err := CompleteEpayTopUp(input)
	require.NoError(t, err)
	assert.True(t, first.CommissionCreated)
	assert.NotZero(t, first.CommissionID)
	second, err := CompleteEpayTopUp(input)
	require.NoError(t, err)
	assert.True(t, second.AlreadyCompleted)

	var commission AffiliateCommission
	require.NoError(t, DB.Where("top_up_id = ?", order.Id).First(&commission).Error)
	assert.Equal(t, agent.Id, commission.AgentUserId)
	assert.Equal(t, int64(2_500), commission.CommissionAmountMinor)
	assert.Equal(t, order.QuotaAmount, commission.PurchasedQuota)
	assert.Equal(t, 500, commission.GrossRewardQuota)
	assert.Equal(t, 500, commission.RewardQuota)
	assert.Equal(t, AffiliateCommissionStatusPending, commission.Status)
	assert.Equal(t, int64(200+7*24*60*60), commission.AvailableAt)

	var count int64
	require.NoError(t, DB.Model(&AffiliateCommission{}).Where("top_up_id = ?", order.Id).Count(&count).Error)
	assert.Equal(t, int64(1), count)
	var reloadedReferred User
	require.NoError(t, DB.First(&reloadedReferred, referred.Id).Error)
	assert.Equal(t, referred.Quota+order.QuotaAmount, reloadedReferred.Quota)
}

func TestAffiliateCommissionCalculationsUseIndependentImmutableBases(t *testing.T) {
	cashMinor, err := affiliateCommissionValueMinor(800, 1_000)
	require.NoError(t, err)
	assert.Equal(t, int64(80), cashMinor)

	balanceQuota, err := affiliateBalanceRewardQuota(12_500_003, 1_000)
	require.NoError(t, err)
	assert.Equal(t, 1_250_000, balanceQuota)
}

func TestCompleteEpayTopUpSnapshotsCashAndPurchasedQuotaCommissionValues(t *testing.T) {
	truncateTables(t)
	useAffiliateSettingForTopUpTest(t, 0)
	setting := operation_setting.GetAffiliateSettingSnapshot()
	setting.CommissionRateBPS = 1_000
	encoded, err := operation_setting.MarshalAffiliateSetting(setting)
	require.NoError(t, err)
	require.NoError(t, UpdateOption(operation_setting.AffiliateSettingOptionKey, encoded))

	_, referred := createAffiliateUsersAndProfile(t, 100)
	order := TopUp{
		UserId:                  referred.Id,
		Amount:                  25,
		Money:                   8,
		TradeNo:                 "affiliate-paid-amount-basis",
		PaymentMethod:           "alipay",
		PaymentProvider:         PaymentProviderEpay,
		ExpectedAmountMinor:     800,
		PaidCurrency:            "CNY",
		QuotaAmount:             12_500_000,
		UnitPriceSnapshot:       "0.32",
		QuotaPerUnitSnapshot:    "500000",
		TopUpGroupRatioSnapshot: "1",
		AmountDiscountSnapshot:  "1",
		CommissionEligible:      true,
		CreateTime:              100,
		Status:                  common.TopUpStatusPending,
	}
	require.NoError(t, DB.Create(&order).Error)

	result, err := CompleteEpayTopUp(EpaySettlement{
		TradeNo: order.TradeNo, ProviderTradeNo: "epay-paid-amount-basis", PaidAmountMinor: 800,
		PaymentMethod: "alipay", CompletedAt: 200,
	})
	require.NoError(t, err)
	assert.True(t, result.CommissionCreated)
	assert.Equal(t, order.QuotaAmount, result.QuotaAdded)

	var commission AffiliateCommission
	require.NoError(t, DB.Where("top_up_id = ?", order.Id).First(&commission).Error)
	assert.Equal(t, result.QuotaAdded, commission.PurchasedQuota)
	assert.Equal(t, int64(80), commission.CommissionAmountMinor)
	assert.Equal(t, 1_250_000, commission.GrossRewardQuota)
	assert.Equal(t, 1_250_000, commission.RewardQuota)

	var reloadedReferred User
	require.NoError(t, DB.First(&reloadedReferred, referred.Id).Error)
	assert.Equal(t, referred.Quota+commission.PurchasedQuota, reloadedReferred.Quota)
}

func TestCompleteEpayTopUpUsesDatabaseAffiliateConfigSnapshot(t *testing.T) {
	truncateTables(t)
	useAffiliateSettingForTopUpTest(t, 0)
	runtimeSetting := operation_setting.GetAffiliateSettingSnapshot()
	runtimeSetting.CommissionRateBPS = 1_000
	require.NoError(t, operation_setting.SetAffiliateSetting(runtimeSetting))
	databaseSetting := runtimeSetting
	databaseSetting.CommissionRateBPS = 5_000
	databaseSetting.Version = 7
	encoded, err := operation_setting.MarshalAffiliateSetting(databaseSetting)
	require.NoError(t, err)
	require.NoError(t, DB.Model(&Option{}).
		Where(commonKeyCol+" = ?", operation_setting.AffiliateSettingOptionKey).
		Update("value", encoded).Error)

	_, referred := createAffiliateUsersAndProfile(t, 100)
	order := createAffiliateEpayOrder(t, referred, "affiliate-db-config", 5_000)
	result, err := CompleteEpayTopUp(EpaySettlement{
		TradeNo: order.TradeNo, ProviderTradeNo: "affiliate-db-config-provider", PaidAmountMinor: 5_000,
		PaymentMethod: "alipay", CompletedAt: 200,
	})
	require.NoError(t, err)
	var commission AffiliateCommission
	require.NoError(t, DB.First(&commission, result.CommissionID).Error)
	assert.Equal(t, int64(5_000), commission.CommissionRateBPS)
	assert.Equal(t, int64(7), commission.ConfigVersion)
}

func TestAffiliateCommissionTransferReversalAndDebtOffset(t *testing.T) {
	truncateTables(t)
	useAffiliateSettingForTopUpTest(t, 0)
	agent, referred := createAffiliateUsersAndProfile(t, 100)
	order := createAffiliateEpayOrder(t, referred, "affiliate-transfer-one", 5_000)
	settlement, err := CompleteEpayTopUp(EpaySettlement{
		TradeNo: order.TradeNo, ProviderTradeNo: "epay-transfer-one", PaidAmountMinor: 5_000,
		PaymentMethod: "alipay", CompletedAt: 200,
	})
	require.NoError(t, err)
	var availableEventCount int64
	require.NoError(t, DB.Model(&AffiliateOutboxEvent{}).
		Where("action = ?", "affiliate.commission_available").
		Count(&availableEventCount).Error)
	assert.Equal(t, int64(1), availableEventCount)

	transfer, err := TransferAvailableAffiliateCommissions(agent.Id, "transfer-key", 300)
	require.NoError(t, err)
	assert.False(t, transfer.AlreadyCompleted)
	assert.Equal(t, 500, transfer.Transfer.TransferredQuota)
	replay, err := TransferAvailableAffiliateCommissions(agent.Id, "transfer-key", 301)
	require.NoError(t, err)
	assert.True(t, replay.AlreadyCompleted)
	assert.Equal(t, transfer.Transfer.Id, replay.Transfer.Id)

	reversal, err := ReverseAffiliateCommission(settlement.CommissionID, agent.Id, "payment refunded", 400)
	require.NoError(t, err)
	assert.Equal(t, int64(500), reversal.Reversal.DebtAddedQuota)
	reversalReplay, err := ReverseAffiliateCommission(settlement.CommissionID, agent.Id, "payment refunded", 401)
	require.NoError(t, err)
	assert.True(t, reversalReplay.AlreadyReversed)
	secondOrder := createAffiliateEpayOrder(t, referred, "affiliate-transfer-two", 2_000)
	secondSettlement, err := CompleteEpayTopUp(EpaySettlement{
		TradeNo: secondOrder.TradeNo, ProviderTradeNo: "epay-transfer-two", PaidAmountMinor: 2_000,
		PaymentMethod: "alipay", CompletedAt: 500,
	})
	require.NoError(t, err)
	var secondCommission AffiliateCommission
	require.NoError(t, DB.First(&secondCommission, secondSettlement.CommissionID).Error)
	assert.Equal(t, 500, secondCommission.GrossRewardQuota)
	assert.Equal(t, 500, secondCommission.DebtOffsetQuota)
	assert.Zero(t, secondCommission.RewardQuota)

	var profile AffiliateProfile
	require.NoError(t, DB.First(&profile, agent.Id).Error)
	assert.Zero(t, profile.CommissionDebtQuota)
	var reloadedAgent User
	require.NoError(t, DB.First(&reloadedAgent, agent.Id).Error)
	assert.Equal(t, agent.Quota+500, reloadedAgent.Quota)

	_, err = ReverseAffiliateCommission(secondCommission.Id, agent.Id, "second payment refunded", 600)
	require.NoError(t, err)
	require.NoError(t, DB.First(&profile, agent.Id).Error)
	assert.Equal(t, int64(500), profile.CommissionDebtQuota)
	verified, err := GetVerifiedEpayAmount(referred.Id, "CNY")
	require.NoError(t, err)
	assert.Equal(t, int64(7_000), verified)
}

func TestTransferAffiliateCommissionOverflowRollsBack(t *testing.T) {
	truncateTables(t)
	setAffiliatePaymentComplianceOptions(t, true)
	user := User{Username: "affiliate-overflow", AffCode: "affiliate-overflow", Status: common.UserStatusEnabled, Quota: common.MaxQuota}
	require.NoError(t, DB.Create(&user).Error)
	require.NoError(t, DB.Create(&AffiliateProfile{
		UserId: user.Id, Access: AffiliateAccessInherit, Status: AffiliateStatusActive,
		ActivatedAt: 1, Currency: "CNY", CreatedAt: 1, UpdatedAt: 1,
	}).Error)
	commission := AffiliateCommission{
		AgentUserId: user.Id, ReferredUserId: user.Id + 1, TopUpId: 123456,
		ProviderTradeNo: "overflow-provider", PaidAmountMinor: 100, PaidCurrency: "CNY",
		UnitPriceSnapshot: "1", QuotaPerUnitSnapshot: "1", CommissionRateBPS: 10_000,
		CommissionAmountMinor: 100, GrossRewardQuota: 1, RewardQuota: 1,
		Status: AffiliateCommissionStatusAvailable, AvailableAt: 100, ConfigVersion: 1, CreatedAt: 100,
	}
	require.NoError(t, DB.Create(&commission).Error)

	_, err := TransferAvailableAffiliateCommissions(user.Id, "overflow-key", 200)
	assert.ErrorIs(t, err, ErrAffiliateCommissionBalanceOverflow)
	require.NoError(t, DB.First(&commission, commission.Id).Error)
	assert.Equal(t, AffiliateCommissionStatusAvailable, commission.Status)
	var transferCount int64
	require.NoError(t, DB.Model(&AffiliateCommissionTransfer{}).Count(&transferCount).Error)
	assert.Zero(t, transferCount)
}

func TestTransferAffiliateCommissionRejectsSuspendedAgent(t *testing.T) {
	truncateTables(t)
	setAffiliatePaymentComplianceOptions(t, true)
	user := User{Username: "affiliate-suspended", AffCode: "affiliate-suspended", Status: common.UserStatusEnabled, Quota: 100}
	require.NoError(t, DB.Create(&user).Error)
	require.NoError(t, DB.Create(&AffiliateProfile{
		UserId: user.Id, Access: AffiliateAccessDeny, Status: AffiliateStatusSuspended,
		ActivatedAt: 1, Currency: "CNY", CreatedAt: 1, UpdatedAt: 1,
	}).Error)
	commission := AffiliateCommission{
		AgentUserId: user.Id, ReferredUserId: user.Id + 1, TopUpId: 654321,
		ProviderTradeNo: "suspended-provider", PaidAmountMinor: 100, PaidCurrency: "CNY",
		UnitPriceSnapshot: "1", QuotaPerUnitSnapshot: "1", CommissionRateBPS: 10_000,
		CommissionAmountMinor: 100, GrossRewardQuota: 10, RewardQuota: 10,
		Status: AffiliateCommissionStatusAvailable, AvailableAt: 1, ConfigVersion: 1, CreatedAt: 1,
	}
	require.NoError(t, DB.Create(&commission).Error)

	_, err := TransferAvailableAffiliateCommissions(user.Id, "suspended-key", 2)
	assert.ErrorIs(t, err, ErrAffiliateAccessDenied)
	require.NoError(t, DB.First(&commission, commission.Id).Error)
	assert.Equal(t, AffiliateCommissionStatusAvailable, commission.Status)
	var stored User
	require.NoError(t, DB.First(&stored, user.Id).Error)
	assert.Equal(t, 100, stored.Quota)
}

func TestAffiliateCommissionRequiresActiveAgentAtPaymentTime(t *testing.T) {
	truncateTables(t)
	config := AffiliateCommissionConfigSnapshot{
		Enabled: true, QualificationThresholdMinor: 10_000, Currency: "CNY",
		CommissionRateBPS: 5_000, CommissionWaitDays: 0, ConfigVersion: 1,
	}
	agent := User{Username: "inactive-agent", AffCode: "inactive-agent", Status: common.UserStatusEnabled}
	referred := User{Username: "inactive-referred", AffCode: "inactive-referred", Status: common.UserStatusEnabled}
	require.NoError(t, DB.Create(&agent).Error)
	referred.InviterId = agent.Id
	require.NoError(t, DB.Create(&referred).Error)
	require.NoError(t, DB.Create(&AffiliateProfile{
		UserId: agent.Id, Access: AffiliateAccessInherit, Status: AffiliateStatusActive,
		ActivatedAt: 300, Currency: "CNY", CreatedAt: 300, UpdatedAt: 300,
	}).Error)
	order := createAffiliateEpayOrder(t, referred, "inactive-agent-order", 1_000)

	err := DB.Transaction(func(tx *gorm.DB) error {
		commission, created, err := CreateAffiliateCommissionForTopUpWithTx(tx, AffiliateCommissionSettlement{
			TopUp: &order, ProviderTradeNo: "inactive-provider", PaidAmountMinor: 1_000,
			PaidCurrency: "CNY", CompletedAt: 200, Config: config,
		})
		require.NoError(t, err)
		assert.Nil(t, commission)
		assert.False(t, created)
		return nil
	})
	require.NoError(t, err)
	var count int64
	require.NoError(t, DB.Model(&AffiliateCommission{}).Count(&count).Error)
	assert.Zero(t, count)
}

func TestAffiliateTransfersUsePersistedPaymentCompliance(t *testing.T) {
	truncateTables(t)
	setAffiliatePaymentComplianceOptions(t, false)
	previousPaymentSetting := *operation_setting.GetPaymentSetting()
	operation_setting.GetPaymentSetting().ComplianceConfirmed = true
	operation_setting.GetPaymentSetting().ComplianceTermsVersion = operation_setting.CurrentComplianceTermsVersion
	t.Cleanup(func() { *operation_setting.GetPaymentSetting() = previousPaymentSetting })

	user := User{Username: "compliance-transfer-user", AffCode: "compliance-transfer", Quota: 100, AffQuota: 50}
	require.NoError(t, DB.Create(&user).Error)
	require.NoError(t, DB.Create(&AffiliateCommission{
		AgentUserId: user.Id, ReferredUserId: user.Id + 1, TopUpId: 1,
		ProviderTradeNo: "compliance-transfer-provider", PaidAmountMinor: 100,
		PaidCurrency: "CNY", UnitPriceSnapshot: "1", QuotaPerUnitSnapshot: "1",
		CommissionRateBPS: 100, CommissionAmountMinor: 1, GrossRewardQuota: 10,
		RewardQuota: 10, Status: AffiliateCommissionStatusAvailable, AvailableAt: 1,
		ConfigVersion: 1, CreatedAt: 1,
	}).Error)

	_, err := TransferAffiliateSignupRewards(user.Id, 10, "compliance-signup")
	assert.ErrorIs(t, err, ErrPaymentComplianceRequired)
	_, err = TransferAvailableAffiliateCommissions(user.Id, "compliance-commission", 2)
	assert.ErrorIs(t, err, ErrPaymentComplianceRequired)

	var stored User
	require.NoError(t, DB.First(&stored, user.Id).Error)
	assert.Equal(t, 100, stored.Quota)
	assert.Equal(t, 50, stored.AffQuota)
}

func TestEpayTopUpQuotaOverflowDoesNotCompleteOrder(t *testing.T) {
	truncateTables(t)
	user := User{Username: "epay-overflow-user", AffCode: "epay-overflow", Status: common.UserStatusEnabled, Quota: common.MaxQuota}
	require.NoError(t, DB.Create(&user).Error)
	order := createPendingEpayTopUp(t, "epay-overflow-order", user.Id)

	_, err := CompleteEpayTopUp(EpaySettlement{
		TradeNo: order.TradeNo, ProviderTradeNo: "epay-overflow-provider", PaidAmountMinor: order.ExpectedAmountMinor,
		PaymentMethod: "alipay", CompletedAt: 200,
	})
	assert.True(t, errors.Is(err, ErrTopUpQuotaLimitExceeded))
	var reloaded TopUp
	require.NoError(t, DB.First(&reloaded, order.Id).Error)
	assert.Equal(t, common.TopUpStatusPending, reloaded.Status)
}
