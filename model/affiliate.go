package model

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	AffiliateAccessInherit = "inherit"
	AffiliateAccessAllow   = "allow"
	AffiliateAccessDeny    = "deny"

	AffiliateStatusInactive  = "inactive"
	AffiliateStatusActive    = "active"
	AffiliateStatusSuspended = "suspended"

	AffiliateCommissionStatusPending     = "pending"
	AffiliateCommissionStatusAvailable   = "available"
	AffiliateCommissionStatusTransferred = "transferred"
	AffiliateCommissionStatusReversed    = "reversed"
)

var (
	ErrAffiliateConfigInvalid               = errors.New("affiliate configuration is invalid")
	ErrAffiliateProfileNotFound             = errors.New("affiliate profile was not found")
	ErrAffiliateAccessInvalid               = errors.New("affiliate access is invalid")
	ErrAffiliateAccessDenied                = errors.New("affiliate access is denied")
	ErrAffiliateProgramDisabled             = errors.New("affiliate commission program is disabled")
	ErrAffiliateQualificationNotMet         = errors.New("affiliate qualification threshold is not met")
	ErrAffiliateAlreadyActive               = errors.New("affiliate is already active")
	ErrAffiliateCommissionInvalid           = errors.New("affiliate commission settlement is invalid")
	ErrAffiliateCommissionEvidenceMismatch  = errors.New("affiliate commission evidence does not match")
	ErrAffiliateCommissionNotFound          = errors.New("affiliate commission was not found")
	ErrAffiliateCommissionNotReversible     = errors.New("affiliate commission cannot be reversed")
	ErrAffiliateCommissionBalanceOverflow   = errors.New("affiliate commission would overflow a quota balance")
	ErrAffiliateCommissionNothingToTransfer = errors.New("no affiliate commission is available to transfer")
	ErrAffiliateIdempotencyKeyRequired      = errors.New("affiliate idempotency key is required")
)

// AffiliateProfile stores mutable access and activation state separately from
// User so legacy full-row user updates cannot overwrite financial state.
type AffiliateProfile struct {
	UserId               int    `json:"user_id" gorm:"primaryKey;autoIncrement:false"`
	Access               string `json:"access" gorm:"type:varchar(16);not null"`
	Status               string `json:"status" gorm:"type:varchar(16);not null;index"`
	ActivatedAt          int64  `json:"activated_at" gorm:"not null"`
	QualifiedAmountMinor int64  `json:"qualified_amount_minor,string" gorm:"not null"`
	Currency             string `json:"currency" gorm:"type:varchar(3);not null"`
	CommissionDebtQuota  int64  `json:"commission_debt_quota,string" gorm:"not null"`
	CreatedAt            int64  `json:"created_at" gorm:"not null;autoCreateTime"`
	UpdatedAt            int64  `json:"updated_at" gorm:"not null;autoUpdateTime"`
}

func (AffiliateProfile) TableName() string { return "affiliate_profiles" }

// AffiliateCommission is immutable for all financial snapshot columns. Only
// lifecycle, transfer and reversal columns may change after creation.
type AffiliateCommission struct {
	Id                    int64  `json:"id" gorm:"primaryKey"`
	AgentUserId           int    `json:"agent_user_id" gorm:"not null;index"`
	ReferredUserId        int    `json:"referred_user_id" gorm:"not null;index"`
	TopUpId               int    `json:"topup_id" gorm:"not null;uniqueIndex"`
	ProviderTradeNo       string `json:"-" gorm:"type:varchar(255);not null"`
	PaidAmountMinor       int64  `json:"paid_amount_minor,string" gorm:"not null"`
	PaidCurrency          string `json:"paid_currency" gorm:"type:varchar(3);not null"`
	PurchasedQuota        int    `json:"purchased_quota" gorm:"not null"`
	UnitPriceSnapshot     string `json:"unit_price_snapshot" gorm:"type:varchar(64);not null"`
	QuotaPerUnitSnapshot  string `json:"quota_per_unit_snapshot" gorm:"type:varchar(64);not null"`
	CommissionRateBPS     int64  `json:"commission_rate_bps" gorm:"not null"`
	CommissionAmountMinor int64  `json:"commission_amount_minor,string" gorm:"not null"`
	GrossRewardQuota      int    `json:"gross_reward_quota" gorm:"not null"`
	DebtOffsetQuota       int    `json:"debt_offset_quota" gorm:"not null"`
	RewardQuota           int    `json:"reward_quota" gorm:"not null"`
	Status                string `json:"status" gorm:"type:varchar(16);not null;index"`
	AvailableAt           int64  `json:"available_at" gorm:"not null;index"`
	TransferId            *int64 `json:"transfer_id,omitempty" gorm:"index"`
	TransferredAt         int64  `json:"transferred_at" gorm:"not null"`
	ReversedAt            int64  `json:"reversed_at" gorm:"not null"`
	ReversedBy            int    `json:"reversed_by" gorm:"not null"`
	ReverseReason         string `json:"reverse_reason" gorm:"type:varchar(255);not null"`
	ConfigVersion         int64  `json:"config_version" gorm:"not null"`
	CreatedAt             int64  `json:"created_at" gorm:"not null;autoCreateTime;index"`
	ReferredUsername      string `json:"-" gorm:"column:referred_username;->;-:migration"`
}

func (AffiliateCommission) TableName() string { return "affiliate_commissions" }

type AffiliateCommissionTransfer struct {
	Id               int64  `json:"id" gorm:"primaryKey"`
	UserId           int    `json:"user_id" gorm:"not null;index;uniqueIndex:ux_aff_commission_transfer_key,priority:1"`
	IdempotencyKey   string `json:"idempotency_key" gorm:"type:varchar(64);not null;uniqueIndex:ux_aff_commission_transfer_key,priority:2"`
	TransferredQuota int    `json:"transferred_quota" gorm:"not null"`
	CommissionCount  int    `json:"commission_count" gorm:"not null"`
	MainQuotaBefore  int    `json:"main_quota_before" gorm:"not null"`
	MainQuotaAfter   int    `json:"main_quota_after" gorm:"not null"`
	CreatedAt        int64  `json:"created_at" gorm:"not null;autoCreateTime"`
}

func (AffiliateCommissionTransfer) TableName() string {
	return "affiliate_commission_transfers"
}

type AffiliateCommissionReversal struct {
	Id              int64  `json:"id" gorm:"primaryKey"`
	CommissionId    int64  `json:"commission_id" gorm:"not null;uniqueIndex"`
	TopUpId         int    `json:"topup_id" gorm:"not null;index"`
	AgentUserId     int    `json:"agent_user_id" gorm:"not null;index"`
	OperatorId      int    `json:"operator_id" gorm:"not null;index"`
	StatusBefore    string `json:"status_before" gorm:"type:varchar(16);not null"`
	DebtAddedQuota  int64  `json:"debt_added_quota,string" gorm:"not null"`
	DebtBeforeQuota int64  `json:"debt_before_quota,string" gorm:"not null"`
	DebtAfterQuota  int64  `json:"debt_after_quota,string" gorm:"not null"`
	Reason          string `json:"reason" gorm:"type:varchar(255);not null"`
	CreatedAt       int64  `json:"created_at" gorm:"not null;autoCreateTime"`
}

func (AffiliateCommissionReversal) TableName() string {
	return "affiliate_commission_reversals"
}

type AffiliateAccessChange struct {
	Id           int64  `json:"id" gorm:"primaryKey"`
	UserId       int    `json:"user_id" gorm:"not null;index"`
	OperatorId   int    `json:"operator_id" gorm:"not null;index"`
	AccessBefore string `json:"access_before" gorm:"type:varchar(16);not null"`
	AccessAfter  string `json:"access_after" gorm:"type:varchar(16);not null"`
	StatusBefore string `json:"status_before" gorm:"type:varchar(16);not null"`
	StatusAfter  string `json:"status_after" gorm:"type:varchar(16);not null"`
	Reason       string `json:"reason" gorm:"type:varchar(255);not null"`
	CreatedAt    int64  `json:"created_at" gorm:"not null;autoCreateTime"`
}

func (AffiliateAccessChange) TableName() string { return "affiliate_access_changes" }

type AffiliateConfigChange struct {
	Id            int64  `json:"id" gorm:"primaryKey"`
	OperatorId    int    `json:"operator_id" gorm:"not null;index"`
	VersionBefore int64  `json:"version_before" gorm:"not null"`
	VersionAfter  int64  `json:"version_after" gorm:"not null;uniqueIndex"`
	ValueBefore   string `json:"value_before" gorm:"type:text;not null"`
	ValueAfter    string `json:"value_after" gorm:"type:text;not null"`
	CreatedAt     int64  `json:"created_at" gorm:"not null;autoCreateTime"`
}

func (AffiliateConfigChange) TableName() string { return "affiliate_config_changes" }

// AffiliateCommissionConfigSnapshot is captured once for each activation or
// payment transaction. No transaction may observe a partially updated config.
type AffiliateCommissionConfigSnapshot struct {
	Enabled                     bool
	QualificationThresholdMinor int64
	Currency                    string
	CommissionRateBPS           int64
	CommissionWaitDays          int64
	ConfigVersion               int64
}

type AffiliateProgramConfigSnapshot struct {
	Setting                    operation_setting.AffiliateSetting
	PaymentComplianceConfirmed bool
	EpayCurrency               string
}

func normalizeAffiliateCommissionConfig(config AffiliateCommissionConfigSnapshot) AffiliateCommissionConfigSnapshot {
	config.Currency = strings.ToUpper(strings.TrimSpace(config.Currency))
	return config
}

func validateAffiliateCommissionConfig(config AffiliateCommissionConfigSnapshot) error {
	if config.QualificationThresholdMinor < 0 || len(config.Currency) != 3 ||
		config.CommissionRateBPS < 0 || config.CommissionRateBPS > 10000 ||
		config.CommissionWaitDays < 0 || config.CommissionWaitDays > 3650 ||
		config.ConfigVersion < 1 {
		return ErrAffiliateConfigInvalid
	}
	for _, char := range config.Currency {
		if char < 'A' || char > 'Z' {
			return ErrAffiliateConfigInvalid
		}
	}
	return nil
}

func affiliateProgramConfigWithTx(tx *gorm.DB) (AffiliateProgramConfigSnapshot, error) {
	values, err := optionValuesWithTx(
		tx,
		operation_setting.AffiliateSettingOptionKey,
		PaymentComplianceConfirmedOptionKey,
		PaymentComplianceTermsVersionOptionKey,
		EpayCurrencyOptionKey,
	)
	if err != nil {
		return AffiliateProgramConfigSnapshot{}, err
	}
	setting, err := affiliateSettingFromOptionValues(values)
	if err != nil {
		return AffiliateProgramConfigSnapshot{}, err
	}
	currency, err := epayCurrencyFromOptionValues(values)
	if err != nil {
		return AffiliateProgramConfigSnapshot{}, err
	}
	return AffiliateProgramConfigSnapshot{
		Setting:                    setting,
		PaymentComplianceConfirmed: paymentComplianceConfirmedFromOptionValues(values),
		EpayCurrency:               currency,
	}, nil
}

func GetAffiliateProgramConfigSnapshot() (AffiliateProgramConfigSnapshot, error) {
	return affiliateProgramConfigWithTx(DB)
}

func affiliateCommissionConfigWithTx(tx *gorm.DB) (AffiliateCommissionConfigSnapshot, error) {
	program, err := affiliateProgramConfigWithTx(tx)
	if err != nil {
		return AffiliateCommissionConfigSnapshot{}, err
	}
	config := AffiliateCommissionConfigSnapshot{
		Enabled:                     program.Setting.CommissionEnabled && program.PaymentComplianceConfirmed,
		QualificationThresholdMinor: program.Setting.QualificationThresholdMinor,
		Currency:                    program.EpayCurrency,
		CommissionRateBPS:           program.Setting.CommissionRateBPS,
		CommissionWaitDays:          program.Setting.CommissionWaitDays,
		ConfigVersion:               program.Setting.Version,
	}
	config = normalizeAffiliateCommissionConfig(config)
	if err := validateAffiliateCommissionConfig(config); err != nil {
		return AffiliateCommissionConfigSnapshot{}, err
	}
	return config, nil
}

func validAffiliateAccess(access string) bool {
	return access == AffiliateAccessInherit || access == AffiliateAccessAllow || access == AffiliateAccessDeny
}

func newAffiliateProfile(userId int) *AffiliateProfile {
	now := common.GetTimestamp()
	return &AffiliateProfile{
		UserId:    userId,
		Access:    AffiliateAccessInherit,
		Status:    AffiliateStatusInactive,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func getOrCreateAffiliateProfileWithTx(tx *gorm.DB, userId int) (*AffiliateProfile, error) {
	if userId <= 0 {
		return nil, ErrAffiliateProfileNotFound
	}
	profile := AffiliateProfile{}
	err := lockForUpdate(tx).Where("user_id = ?", userId).First(&profile).Error
	if err == nil {
		return &profile, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	var count int64
	if err := tx.Model(&User{}).Where("id = ?", userId).Count(&count).Error; err != nil {
		return nil, err
	}
	if count != 1 {
		return nil, ErrAffiliateProfileNotFound
	}
	profile = *newAffiliateProfile(userId)
	create := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&profile)
	if create.Error != nil {
		return nil, create.Error
	}
	if create.RowsAffected == 0 {
		if err := lockForUpdate(tx).Where("user_id = ?", userId).First(&profile).Error; err != nil {
			return nil, err
		}
	}
	return &profile, nil
}

func addInt64Checked(left int64, right int64) (int64, error) {
	if right > 0 && left > math.MaxInt64-right {
		return 0, ErrAffiliateCommissionBalanceOverflow
	}
	if right < 0 && left < math.MinInt64-right {
		return 0, ErrAffiliateCommissionBalanceOverflow
	}
	return left + right, nil
}

func affiliateCommissionValueMinor(paidAmountMinor int64, rateBPS int64) (int64, error) {
	if paidAmountMinor <= 0 || rateBPS < 0 || rateBPS > 10000 {
		return 0, ErrAffiliateCommissionInvalid
	}
	return decimal.NewFromInt(paidAmountMinor).
		Mul(decimal.NewFromInt(rateBPS)).
		Div(decimal.NewFromInt(10000)).
		Floor().IntPart(), nil
}

// purchasedQuota is the exact TopUp.QuotaAmount credited to the referred user;
// never derive it from cash, price, discounts, or the current quota settings.
func affiliateBalanceRewardQuota(purchasedQuota int, rateBPS int64) (int, error) {
	if purchasedQuota <= 0 || rateBPS < 0 || rateBPS > 10000 {
		return 0, ErrAffiliateCommissionInvalid
	}
	quotaDecimal := decimal.NewFromInt(int64(purchasedQuota)).
		Mul(decimal.NewFromInt(rateBPS)).
		Div(decimal.NewFromInt(10000)).
		Floor()
	quota, clamp := common.QuotaFromDecimalChecked(quotaDecimal)
	if clamp != nil || quota < 0 {
		return 0, ErrAffiliateCommissionBalanceOverflow
	}
	return quota, nil
}

func sumVerifiedEpayAmountWithTx(tx *gorm.DB, userId int, currency string) (int64, error) {
	var total int64
	err := verifiedEpayPayments(tx, currency).
		Select("COALESCE(SUM(top_ups.paid_amount_minor), 0)").
		Where("top_ups.user_id = ?", userId).
		Scan(&total).Error
	return total, err
}

func GetVerifiedEpayAmount(userId int, currency string) (int64, error) {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if userId <= 0 || len(currency) != 3 {
		return 0, ErrAffiliateConfigInvalid
	}
	return sumVerifiedEpayAmountWithTx(DB, userId, currency)
}

type AffiliateActivationResult struct {
	Profile             AffiliateProfile `json:"profile"`
	VerifiedAmountMinor int64            `json:"verified_amount_minor,string"`
	AlreadyActive       bool             `json:"already_active"`
}

func ActivateAffiliate(userId int) (AffiliateActivationResult, error) {
	result := AffiliateActivationResult{}
	err := DB.Transaction(func(tx *gorm.DB) error {
		config, err := affiliateCommissionConfigWithTx(tx)
		if err != nil {
			return err
		}
		if !config.Enabled {
			return ErrAffiliateProgramDisabled
		}
		profile, err := getOrCreateAffiliateProfileWithTx(tx, userId)
		if err != nil {
			return err
		}
		if profile.Access == AffiliateAccessDeny || profile.Status == AffiliateStatusSuspended {
			return ErrAffiliateAccessDenied
		}

		verifiedAmount, err := sumVerifiedEpayAmountWithTx(tx, userId, config.Currency)
		if err != nil {
			return err
		}
		result.VerifiedAmountMinor = verifiedAmount
		if profile.Status == AffiliateStatusActive && profile.ActivatedAt > 0 {
			result.Profile = *profile
			result.AlreadyActive = true
			return nil
		}
		if profile.Access != AffiliateAccessAllow && verifiedAmount < config.QualificationThresholdMinor {
			return ErrAffiliateQualificationNotMet
		}

		now := common.GetTimestamp()
		updates := map[string]interface{}{
			"status":                 AffiliateStatusActive,
			"activated_at":           now,
			"qualified_amount_minor": verifiedAmount,
			"currency":               config.Currency,
			"updated_at":             now,
		}
		if update := tx.Model(&AffiliateProfile{}).
			Where("user_id = ? AND status <> ? AND access <> ?", userId, AffiliateStatusActive, AffiliateAccessDeny).
			Updates(updates); update.Error != nil {
			return update.Error
		} else if update.RowsAffected != 1 {
			return ErrAffiliateAlreadyActive
		}
		profile.Status = AffiliateStatusActive
		profile.ActivatedAt = now
		profile.QualifiedAmountMinor = verifiedAmount
		profile.Currency = config.Currency
		profile.UpdatedAt = now
		result.Profile = *profile
		_, err = EnqueueAffiliateEventWithTx(
			tx,
			fmt.Sprintf("affiliate.activated:%d:%d", userId, now),
			userId,
			"affiliate.activated",
			LogTypeSystem,
			"Affiliate commission program activated",
			map[string]interface{}{
				"access":                        profile.Access,
				"activated_at":                  now,
				"qualification_currency":        config.Currency,
				"qualification_threshold_minor": fmt.Sprintf("%d", config.QualificationThresholdMinor),
				"verified_amount_minor":         fmt.Sprintf("%d", verifiedAmount),
			},
			now,
		)
		if err != nil {
			return err
		}
		return nil
	})
	return result, wrapAffiliateError("activate affiliate", err)
}

type AffiliateAccessChangeResult struct {
	Profile AffiliateProfile      `json:"profile"`
	Change  AffiliateAccessChange `json:"change"`
}

func SetAffiliateAccess(
	userId int,
	operatorId int,
	access string,
	reason string,
) (AffiliateAccessChangeResult, error) {
	access = strings.ToLower(strings.TrimSpace(access))
	reason = strings.TrimSpace(reason)
	if userId <= 0 || operatorId <= 0 || !validAffiliateAccess(access) || reason == "" || utf8.RuneCountInString(reason) > 255 {
		return AffiliateAccessChangeResult{}, ErrAffiliateAccessInvalid
	}

	result := AffiliateAccessChangeResult{}
	err := DB.Transaction(func(tx *gorm.DB) error {
		profile, err := getOrCreateAffiliateProfileWithTx(tx, userId)
		if err != nil {
			return err
		}
		beforeAccess := profile.Access
		beforeStatus := profile.Status
		afterStatus := AffiliateStatusInactive
		updates := map[string]interface{}{}
		activationReset := false
		qualificationRecalculated := false
		verifiedAmountAtChange := int64(0)
		qualificationThresholdAtChange := int64(0)
		qualificationCurrencyAtChange := ""
		if profile.ActivatedAt > 0 {
			afterStatus = AffiliateStatusActive
		}
		if access == AffiliateAccessDeny && profile.ActivatedAt > 0 {
			afterStatus = AffiliateStatusSuspended
		}
		if beforeAccess != AffiliateAccessInherit && access == AffiliateAccessInherit && profile.ActivatedAt > 0 {
			config, err := affiliateCommissionConfigWithTx(tx)
			if err != nil {
				return err
			}
			verifiedAmount, err := sumVerifiedEpayAmountWithTx(tx, userId, config.Currency)
			if err != nil {
				return err
			}
			qualificationRecalculated = true
			verifiedAmountAtChange = verifiedAmount
			qualificationThresholdAtChange = config.QualificationThresholdMinor
			qualificationCurrencyAtChange = config.Currency
			if verifiedAmount < config.QualificationThresholdMinor {
				afterStatus = AffiliateStatusInactive
				updates["activated_at"] = 0
				updates["qualified_amount_minor"] = 0
				updates["currency"] = ""
				activationReset = true
			}
		}

		now := common.GetTimestamp()
		updates["access"] = access
		updates["status"] = afterStatus
		updates["updated_at"] = now
		if err := tx.Model(&AffiliateProfile{}).
			Where("user_id = ?", userId).
			Updates(updates).Error; err != nil {
			return err
		}
		change := AffiliateAccessChange{
			UserId:       userId,
			OperatorId:   operatorId,
			AccessBefore: beforeAccess,
			AccessAfter:  access,
			StatusBefore: beforeStatus,
			StatusAfter:  afterStatus,
			Reason:       reason,
			CreatedAt:    now,
		}
		if err := tx.Create(&change).Error; err != nil {
			return err
		}
		eventPayload := map[string]interface{}{
			"access_after":   access,
			"access_before":  beforeAccess,
			"reason":         reason,
			"status_after":   afterStatus,
			"status_before":  beforeStatus,
			"target_user_id": userId,
		}
		if qualificationRecalculated {
			eventPayload["activated_at_before"] = profile.ActivatedAt
			eventPayload["activation_reset"] = activationReset
			eventPayload["qualification_currency"] = qualificationCurrencyAtChange
			eventPayload["qualification_recalculated"] = true
			eventPayload["qualification_threshold_minor"] = fmt.Sprintf("%d", qualificationThresholdAtChange)
			eventPayload["verified_amount_minor"] = fmt.Sprintf("%d", verifiedAmountAtChange)
		}
		if _, err := EnqueueAffiliateEventWithTx(
			tx,
			fmt.Sprintf("affiliate.access_changed:%d", change.Id),
			operatorId,
			"affiliate.access_changed",
			LogTypeManage,
			"Changed affiliate access",
			eventPayload,
			now,
		); err != nil {
			return err
		}
		profile.Access = access
		profile.Status = afterStatus
		if activationReset {
			profile.ActivatedAt = 0
			profile.QualifiedAmountMinor = 0
			profile.Currency = ""
		}
		profile.UpdatedAt = now
		result.Profile = *profile
		result.Change = change
		return nil
	})
	return result, wrapAffiliateError("set affiliate access", err)
}

type AffiliateCommissionSettlement struct {
	TopUp           *TopUp
	ProviderTradeNo string
	PaidAmountMinor int64
	PaidCurrency    string
	CompletedAt     int64
	Config          AffiliateCommissionConfigSnapshot
}

// CreateAffiliateCommissionForTopUpWithTx attaches at most one commission to
// a verified Epay order. The caller must use the same transaction that records
// the payment and credits the referred user.
func CreateAffiliateCommissionForTopUpWithTx(
	tx *gorm.DB,
	input AffiliateCommissionSettlement,
) (*AffiliateCommission, bool, error) {
	input.ProviderTradeNo = strings.TrimSpace(input.ProviderTradeNo)
	input.PaidCurrency = strings.ToUpper(strings.TrimSpace(input.PaidCurrency))
	input.Config = normalizeAffiliateCommissionConfig(input.Config)
	if tx == nil || input.TopUp == nil || input.TopUp.Id <= 0 || input.TopUp.UserId <= 0 ||
		input.TopUp.PaymentProvider != PaymentProviderEpay || input.ProviderTradeNo == "" ||
		input.PaidAmountMinor <= 0 || input.PaidAmountMinor != input.TopUp.ExpectedAmountMinor ||
		input.PaidCurrency != input.TopUp.PaidCurrency || input.CompletedAt <= 0 {
		return nil, false, ErrAffiliateCommissionInvalid
	}
	if err := validateAffiliateCommissionConfig(input.Config); err != nil {
		return nil, false, err
	}
	if !input.TopUp.CommissionEligible || !input.Config.Enabled || input.Config.CommissionRateBPS == 0 ||
		input.PaidCurrency != input.Config.Currency {
		return nil, false, nil
	}

	var existing AffiliateCommission
	err := lockForUpdate(tx).Where("top_up_id = ?", input.TopUp.Id).First(&existing).Error
	if err == nil {
		if existing.ProviderTradeNo != input.ProviderTradeNo ||
			existing.PaidAmountMinor != input.PaidAmountMinor ||
			existing.PaidCurrency != input.PaidCurrency ||
			existing.PurchasedQuota != input.TopUp.QuotaAmount ||
			existing.ReferredUserId != input.TopUp.UserId {
			return nil, false, ErrAffiliateCommissionEvidenceMismatch
		}
		return &existing, false, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}
	inviterId := 0
	var referral AffiliateReferral
	err = tx.Where("referred_user_id = ?", input.TopUp.UserId).First(&referral).Error
	if err == nil {
		inviterId = referral.InviterUserId
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}
	if inviterId == 0 {
		var referred User
		if err := tx.Select("id", "inviter_id").Where("id = ?", input.TopUp.UserId).First(&referred).Error; err != nil {
			return nil, false, err
		}
		inviterId = referred.InviterId
	}
	if inviterId <= 0 || inviterId == input.TopUp.UserId {
		return nil, false, nil
	}

	var inviter User
	if err := lockForUpdate(tx).
		Select("id", "status").
		Where("id = ?", inviterId).
		First(&inviter).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	if inviter.Status != common.UserStatusEnabled {
		return nil, false, nil
	}

	var profile AffiliateProfile
	if err := lockForUpdate(tx).Where("user_id = ?", inviterId).First(&profile).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	if profile.Access == AffiliateAccessDeny || profile.Status != AffiliateStatusActive ||
		profile.ActivatedAt <= 0 || input.CompletedAt < profile.ActivatedAt {
		return nil, false, nil
	}
	if profile.CommissionDebtQuota < 0 {
		return nil, false, ErrAffiliateCommissionBalanceOverflow
	}

	commissionAmountMinor, err := affiliateCommissionValueMinor(input.PaidAmountMinor, input.Config.CommissionRateBPS)
	if err != nil {
		return nil, false, err
	}
	grossRewardQuota, err := affiliateBalanceRewardQuota(
		input.TopUp.QuotaAmount,
		input.Config.CommissionRateBPS,
	)
	if err != nil {
		return nil, false, err
	}
	if commissionAmountMinor == 0 && grossRewardQuota == 0 {
		return nil, false, nil
	}

	debtOffset := 0
	if profile.CommissionDebtQuota > 0 {
		debtOffset = grossRewardQuota
		if int64(debtOffset) > profile.CommissionDebtQuota {
			debtOffset = int(profile.CommissionDebtQuota)
		}
	}
	rewardQuota := grossRewardQuota - debtOffset
	waitSeconds := input.Config.CommissionWaitDays * 24 * 60 * 60
	if input.CompletedAt > math.MaxInt64-waitSeconds {
		return nil, false, ErrAffiliateConfigInvalid
	}
	availableAt := input.CompletedAt + waitSeconds
	status := AffiliateCommissionStatusPending
	if input.Config.CommissionWaitDays == 0 {
		status = AffiliateCommissionStatusAvailable
	}

	commission := &AffiliateCommission{
		AgentUserId:           inviterId,
		ReferredUserId:        input.TopUp.UserId,
		TopUpId:               input.TopUp.Id,
		ProviderTradeNo:       input.ProviderTradeNo,
		PaidAmountMinor:       input.PaidAmountMinor,
		PaidCurrency:          input.PaidCurrency,
		PurchasedQuota:        input.TopUp.QuotaAmount,
		UnitPriceSnapshot:     input.TopUp.UnitPriceSnapshot,
		QuotaPerUnitSnapshot:  input.TopUp.QuotaPerUnitSnapshot,
		CommissionRateBPS:     input.Config.CommissionRateBPS,
		CommissionAmountMinor: commissionAmountMinor,
		GrossRewardQuota:      grossRewardQuota,
		DebtOffsetQuota:       debtOffset,
		RewardQuota:           rewardQuota,
		Status:                status,
		AvailableAt:           availableAt,
		ConfigVersion:         input.Config.ConfigVersion,
		CreatedAt:             input.CompletedAt,
	}
	if err := tx.Create(commission).Error; err != nil {
		return nil, false, err
	}
	if _, err := EnqueueAffiliateEventWithTx(
		tx,
		fmt.Sprintf("affiliate.commission_created:%d", commission.Id),
		inviterId,
		"affiliate.commission_created",
		LogTypeSystem,
		"Affiliate commission created",
		map[string]interface{}{
			"commission_amount_minor": fmt.Sprintf("%d", commissionAmountMinor),
			"commission_id":           commission.Id,
			"commission_rate_bps":     input.Config.CommissionRateBPS,
			"gross_reward_quota":      grossRewardQuota,
			"paid_amount_minor":       fmt.Sprintf("%d", input.PaidAmountMinor),
			"paid_currency":           input.PaidCurrency,
			"purchased_quota":         input.TopUp.QuotaAmount,
			"referred_user":           fmt.Sprintf("us***%02d", input.TopUp.UserId%100),
			"reward_quota":            rewardQuota,
			"status":                  status,
			"topup_id":                input.TopUp.Id,
		},
		input.CompletedAt,
	); err != nil {
		return nil, false, err
	}
	if status == AffiliateCommissionStatusAvailable {
		if _, err := EnqueueAffiliateEventWithTx(
			tx,
			fmt.Sprintf("affiliate.commission_available:%d", commission.Id),
			inviterId,
			"affiliate.commission_available",
			LogTypeSystem,
			"Affiliate commission became available",
			map[string]interface{}{
				"available_at":  commission.AvailableAt,
				"commission_id": commission.Id,
				"reward_quota":  commission.RewardQuota,
			},
			input.CompletedAt,
		); err != nil {
			return nil, false, err
		}
	}
	if debtOffset > 0 {
		debtAfter := profile.CommissionDebtQuota - int64(debtOffset)
		profileUpdate := tx.Model(&AffiliateProfile{}).
			Where("user_id = ? AND commission_debt_quota = ?", inviterId, profile.CommissionDebtQuota).
			Updates(map[string]interface{}{
				"commission_debt_quota": debtAfter,
				"updated_at":            common.GetTimestamp(),
			})
		if profileUpdate.Error != nil {
			return nil, false, profileUpdate.Error
		}
		if profileUpdate.RowsAffected != 1 {
			return nil, false, ErrAffiliateCommissionBalanceOverflow
		}
		if _, err := EnqueueAffiliateEventWithTx(
			tx,
			fmt.Sprintf("affiliate.commission_debt_offset:%d", commission.Id),
			inviterId,
			"affiliate.commission_debt_offset",
			LogTypeSystem,
			"Affiliate commission offset an outstanding commission debt",
			map[string]interface{}{
				"commission_id":     commission.Id,
				"debt_after_quota":  fmt.Sprintf("%d", debtAfter),
				"debt_before_quota": fmt.Sprintf("%d", profile.CommissionDebtQuota),
				"offset_quota":      debtOffset,
			},
			input.CompletedAt,
		); err != nil {
			return nil, false, err
		}
	}
	return commission, true, nil
}

func promoteAffiliateCommissionsWithTx(tx *gorm.DB, userId int, limit int, now int64) (int, error) {
	if now <= 0 {
		now = common.GetTimestamp()
	}
	query := lockForUpdate(tx).
		Where("status = ? AND available_at <= ?", AffiliateCommissionStatusPending, now).
		Order("id asc")
	if userId > 0 {
		query = query.Where("agent_user_id = ?", userId)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}
	commissions := make([]AffiliateCommission, 0)
	if err := query.Find(&commissions).Error; err != nil {
		return 0, err
	}
	promoted := 0
	for _, commission := range commissions {
		update := tx.Model(&AffiliateCommission{}).
			Where("id = ? AND status = ?", commission.Id, AffiliateCommissionStatusPending).
			Update("status", AffiliateCommissionStatusAvailable)
		if update.Error != nil {
			return 0, update.Error
		}
		if update.RowsAffected == 0 {
			continue
		}
		if _, err := EnqueueAffiliateEventWithTx(
			tx,
			fmt.Sprintf("affiliate.commission_available:%d", commission.Id),
			commission.AgentUserId,
			"affiliate.commission_available",
			LogTypeSystem,
			"Affiliate commission became available",
			map[string]interface{}{
				"available_at":  commission.AvailableAt,
				"commission_id": commission.Id,
				"reward_quota":  commission.RewardQuota,
			},
			now,
		); err != nil {
			return 0, err
		}
		promoted++
	}
	return promoted, nil
}

func PromoteDueAffiliateCommissions(limit int, now int64) (int, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	if now <= 0 {
		now = common.GetTimestamp()
	}
	promoted := 0
	err := DB.Transaction(func(tx *gorm.DB) error {
		var err error
		promoted, err = promoteAffiliateCommissionsWithTx(tx, 0, limit, now)
		return err
	})
	return promoted, wrapAffiliateError("promote affiliate commissions", err)
}

type AffiliateCommissionTransferResult struct {
	Transfer         AffiliateCommissionTransfer `json:"transfer"`
	AlreadyCompleted bool                        `json:"already_completed"`
}

func TransferAvailableAffiliateCommissions(
	userId int,
	idempotencyKey string,
	now int64,
) (AffiliateCommissionTransferResult, error) {
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if userId <= 0 || idempotencyKey == "" || len(idempotencyKey) > 64 {
		return AffiliateCommissionTransferResult{}, ErrAffiliateIdempotencyKeyRequired
	}
	if now <= 0 {
		now = common.GetTimestamp()
	}

	result := AffiliateCommissionTransferResult{}
	err := DB.Transaction(func(tx *gorm.DB) error {
		confirmed, err := paymentComplianceConfirmedWithTx(tx)
		if err != nil {
			return err
		}
		if !confirmed {
			return ErrPaymentComplianceRequired
		}
		var existing AffiliateCommissionTransfer
		err = lockForUpdate(tx).
			Where("user_id = ? AND idempotency_key = ?", userId, idempotencyKey).
			First(&existing).Error
		if err == nil {
			result.Transfer = existing
			result.AlreadyCompleted = true
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		var user User
		if err := lockForUpdate(tx).
			Select("id", "quota").
			Where("id = ?", userId).
			First(&user).Error; err != nil {
			return err
		}
		if user.Quota < 0 {
			return ErrAffiliateCommissionBalanceOverflow
		}
		var profile AffiliateProfile
		if err := lockForUpdate(tx).
			Select("user_id", "access", "status").
			Where("user_id = ?", userId).
			First(&profile).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrAffiliateAccessDenied
			}
			return err
		}
		if profile.Access == AffiliateAccessDeny || profile.Status != AffiliateStatusActive {
			return ErrAffiliateAccessDenied
		}
		if _, err := promoteAffiliateCommissionsWithTx(tx, userId, 0, now); err != nil {
			return err
		}

		commissions := make([]AffiliateCommission, 0)
		if err := lockForUpdate(tx).
			Where(
				"agent_user_id = ? AND status = ? AND reward_quota > 0",
				userId,
				AffiliateCommissionStatusAvailable,
			).
			Order("id asc").
			Find(&commissions).Error; err != nil {
			return err
		}
		if len(commissions) == 0 {
			return ErrAffiliateCommissionNothingToTransfer
		}

		var total int64
		ids := make([]int64, 0, len(commissions))
		for _, commission := range commissions {
			if commission.RewardQuota <= 0 {
				return ErrAffiliateCommissionInvalid
			}
			var err error
			total, err = addInt64Checked(total, int64(commission.RewardQuota))
			if err != nil {
				return err
			}
			ids = append(ids, commission.Id)
		}
		if total <= 0 || total > int64(common.MaxQuota-user.Quota) {
			return ErrAffiliateCommissionBalanceOverflow
		}

		transfer := AffiliateCommissionTransfer{
			UserId:           userId,
			IdempotencyKey:   idempotencyKey,
			TransferredQuota: int(total),
			CommissionCount:  len(commissions),
			MainQuotaBefore:  user.Quota,
			MainQuotaAfter:   user.Quota + int(total),
			CreatedAt:        now,
		}
		if err := tx.Create(&transfer).Error; err != nil {
			return err
		}
		commissionUpdate := tx.Model(&AffiliateCommission{}).
			Where("id IN ? AND status = ?", ids, AffiliateCommissionStatusAvailable).
			Updates(map[string]interface{}{
				"status":         AffiliateCommissionStatusTransferred,
				"transfer_id":    transfer.Id,
				"transferred_at": now,
			})
		if commissionUpdate.Error != nil {
			return commissionUpdate.Error
		}
		if commissionUpdate.RowsAffected != int64(len(ids)) {
			return ErrAffiliateCommissionInvalid
		}
		quotaUpdate := tx.Model(&User{}).
			Where("id = ? AND quota = ?", userId, user.Quota).
			Update("quota", transfer.MainQuotaAfter)
		if quotaUpdate.Error != nil {
			return quotaUpdate.Error
		}
		if quotaUpdate.RowsAffected != 1 {
			return ErrAffiliateCommissionBalanceOverflow
		}
		if _, err := EnqueueAffiliateEventWithTx(
			tx,
			fmt.Sprintf("affiliate.commission_transferred:%d", transfer.Id),
			userId,
			"affiliate.commission_transferred",
			LogTypeSystem,
			"Transferred affiliate commissions to the main balance",
			map[string]interface{}{
				"commission_count":  transfer.CommissionCount,
				"main_quota_after":  transfer.MainQuotaAfter,
				"main_quota_before": transfer.MainQuotaBefore,
				"transfer_id":       transfer.Id,
				"transferred_quota": transfer.TransferredQuota,
			},
			now,
		); err != nil {
			return err
		}
		result.Transfer = transfer
		return nil
	})
	if err != nil {
		return AffiliateCommissionTransferResult{}, wrapAffiliateError("transfer affiliate commissions", err)
	}
	if !result.AlreadyCompleted {
		if err := InvalidateUserCache(userId); err != nil {
			common.SysError(fmt.Sprintf(
				"failed to invalidate user cache after affiliate commission transfer: user_id=%d transfer_id=%d error=%v",
				userId,
				result.Transfer.Id,
				err,
			))
		}
	}
	return result, nil
}

type AffiliateCommissionReversalResult struct {
	Commission      AffiliateCommission         `json:"commission"`
	Reversal        AffiliateCommissionReversal `json:"reversal"`
	AlreadyReversed bool                        `json:"already_reversed"`
}

func ReverseAffiliateCommission(
	commissionId int64,
	operatorId int,
	reason string,
	now int64,
) (AffiliateCommissionReversalResult, error) {
	reason = strings.TrimSpace(reason)
	if commissionId <= 0 || operatorId <= 0 || reason == "" || utf8.RuneCountInString(reason) > 255 {
		return AffiliateCommissionReversalResult{}, ErrAffiliateCommissionNotReversible
	}
	if now <= 0 {
		now = common.GetTimestamp()
	}

	result := AffiliateCommissionReversalResult{}
	err := DB.Transaction(func(tx *gorm.DB) error {
		var reference AffiliateCommission
		if err := tx.Select("id", "top_up_id", "agent_user_id").Where("id = ?", commissionId).First(&reference).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrAffiliateCommissionNotFound
			}
			return err
		}
		var topUp TopUp
		if err := lockForUpdate(tx).Where("id = ?", reference.TopUpId).First(&topUp).Error; err != nil {
			return ErrAffiliateCommissionNotReversible
		}
		profile, err := getOrCreateAffiliateProfileWithTx(tx, reference.AgentUserId)
		if err != nil {
			return err
		}
		var commission AffiliateCommission
		if err := lockForUpdate(tx).Where("id = ? AND top_up_id = ?", commissionId, topUp.Id).First(&commission).Error; err != nil {
			return ErrAffiliateCommissionNotFound
		}
		if !affiliateCommissionMatchesTopUp(&commission, &topUp) {
			return ErrAffiliateCommissionNotReversible
		}
		if commission.Status == AffiliateCommissionStatusReversed {
			var reversal AffiliateCommissionReversal
			if err := tx.Where("commission_id = ?", commission.Id).First(&reversal).Error; err != nil {
				return err
			}
			result.Commission = commission
			result.Reversal = reversal
			result.AlreadyReversed = true
			return nil
		}
		reversal, err := reverseAffiliateCommissionWithTx(tx, &commission, profile, operatorId, reason, now)
		if err != nil {
			return err
		}
		result.Commission = commission
		result.Reversal = reversal
		return nil
	})
	return result, wrapAffiliateError("reverse affiliate commission", err)
}

func affiliateCommissionMatchesTopUp(commission *AffiliateCommission, topUp *TopUp) bool {
	return commission != nil && topUp != nil &&
		topUp.Id == commission.TopUpId &&
		topUp.UserId == commission.ReferredUserId &&
		topUp.PaymentProvider == PaymentProviderEpay &&
		topUp.Status == common.TopUpStatusSuccess &&
		topUp.CompletionSource == CompletionSourceWebhook &&
		topUp.ProviderTradeNo != nil && *topUp.ProviderTradeNo == commission.ProviderTradeNo &&
		topUp.PaidAmountMinor == commission.PaidAmountMinor &&
		topUp.PaidCurrency == commission.PaidCurrency &&
		topUp.QuotaAmount == commission.PurchasedQuota
}

func reverseAffiliateCommissionWithTx(
	tx *gorm.DB,
	commission *AffiliateCommission,
	profile *AffiliateProfile,
	operatorId int,
	reason string,
	now int64,
) (AffiliateCommissionReversal, error) {
	if tx == nil || commission == nil || profile == nil || profile.UserId != commission.AgentUserId ||
		operatorId <= 0 || reason == "" ||
		(commission.Status != AffiliateCommissionStatusPending &&
			commission.Status != AffiliateCommissionStatusAvailable &&
			commission.Status != AffiliateCommissionStatusTransferred) {
		return AffiliateCommissionReversal{}, ErrAffiliateCommissionNotReversible
	}
	if profile.CommissionDebtQuota < 0 {
		return AffiliateCommissionReversal{}, ErrAffiliateCommissionBalanceOverflow
	}
	debtAdded := int64(commission.DebtOffsetQuota)
	if commission.Status == AffiliateCommissionStatusTransferred {
		debtAdded = int64(commission.GrossRewardQuota)
	}
	debtAfter, err := addInt64Checked(profile.CommissionDebtQuota, debtAdded)
	if err != nil {
		return AffiliateCommissionReversal{}, err
	}
	if debtAdded > 0 {
		profileUpdate := tx.Model(&AffiliateProfile{}).
			Where("user_id = ? AND commission_debt_quota = ?", commission.AgentUserId, profile.CommissionDebtQuota).
			Updates(map[string]interface{}{
				"commission_debt_quota": debtAfter,
				"updated_at":            now,
			})
		if profileUpdate.Error != nil {
			return AffiliateCommissionReversal{}, profileUpdate.Error
		}
		if profileUpdate.RowsAffected != 1 {
			return AffiliateCommissionReversal{}, ErrAffiliateCommissionBalanceOverflow
		}
	}

	statusBefore := commission.Status
	commissionUpdate := tx.Model(&AffiliateCommission{}).
		Where("id = ? AND status = ?", commission.Id, statusBefore).
		Updates(map[string]interface{}{
			"status":         AffiliateCommissionStatusReversed,
			"reversed_at":    now,
			"reversed_by":    operatorId,
			"reverse_reason": reason,
		})
	if commissionUpdate.Error != nil {
		return AffiliateCommissionReversal{}, commissionUpdate.Error
	}
	if commissionUpdate.RowsAffected != 1 {
		return AffiliateCommissionReversal{}, ErrAffiliateCommissionNotReversible
	}
	reversal := AffiliateCommissionReversal{
		CommissionId:    commission.Id,
		TopUpId:         commission.TopUpId,
		AgentUserId:     commission.AgentUserId,
		OperatorId:      operatorId,
		StatusBefore:    statusBefore,
		DebtAddedQuota:  debtAdded,
		DebtBeforeQuota: profile.CommissionDebtQuota,
		DebtAfterQuota:  debtAfter,
		Reason:          reason,
		CreatedAt:       now,
	}
	if err := tx.Create(&reversal).Error; err != nil {
		return AffiliateCommissionReversal{}, err
	}
	if _, err := EnqueueAffiliateEventWithTx(
		tx,
		fmt.Sprintf("affiliate.commission_reversed:user:%d", commission.Id),
		commission.AgentUserId,
		"affiliate.commission_reversed",
		LogTypeSystem,
		"Affiliate commission reversed",
		map[string]interface{}{
			"commission_id":    commission.Id,
			"debt_added_quota": fmt.Sprintf("%d", debtAdded),
			"reason":           reason,
			"topup_id":         commission.TopUpId,
		},
		now,
	); err != nil {
		return AffiliateCommissionReversal{}, err
	}
	if _, err := EnqueueAffiliateEventWithTx(
		tx,
		fmt.Sprintf("affiliate.commission_reversed:admin:%d", commission.Id),
		operatorId,
		"affiliate.commission_reversed",
		LogTypeManage,
		"Reversed affiliate commission",
		map[string]interface{}{
			"agent_user_id":    commission.AgentUserId,
			"commission_id":    commission.Id,
			"debt_added_quota": fmt.Sprintf("%d", debtAdded),
			"reason":           reason,
			"topup_id":         commission.TopUpId,
		},
		now,
	); err != nil {
		return AffiliateCommissionReversal{}, err
	}
	commission.Status = AffiliateCommissionStatusReversed
	commission.ReversedAt = now
	commission.ReversedBy = operatorId
	commission.ReverseReason = reason
	return reversal, nil
}

type AffiliateCommissionSummary struct {
	PendingQuota     int64 `json:"pending_quota,string"`
	AvailableQuota   int64 `json:"available_quota,string"`
	TransferredQuota int64 `json:"transferred_quota,string"`
	LifetimeQuota    int64 `json:"lifetime_quota,string"`
	ReversedQuota    int64 `json:"reversed_quota,string"`
}

func sumAffiliateCommissionQuota(query *gorm.DB) (int64, error) {
	var total int64
	err := query.Select("COALESCE(SUM(reward_quota), 0)").Scan(&total).Error
	return total, err
}

func GetAffiliateCommissionSummary(userId int, now int64) (AffiliateCommissionSummary, error) {
	if userId <= 0 {
		return AffiliateCommissionSummary{}, ErrAffiliateProfileNotFound
	}
	if now <= 0 {
		now = common.GetTimestamp()
	}
	base := DB.Model(&AffiliateCommission{}).Where("agent_user_id = ?", userId)
	pending, err := sumAffiliateCommissionQuota(
		base.Session(&gorm.Session{}).
			Where("status = ? AND available_at > ?", AffiliateCommissionStatusPending, now),
	)
	if err != nil {
		return AffiliateCommissionSummary{}, err
	}
	available, err := sumAffiliateCommissionQuota(
		base.Session(&gorm.Session{}).
			Where(
				"reward_quota > 0 AND (status = ? OR (status = ? AND available_at <= ?))",
				AffiliateCommissionStatusAvailable,
				AffiliateCommissionStatusPending,
				now,
			),
	)
	if err != nil {
		return AffiliateCommissionSummary{}, err
	}
	transferred, err := sumAffiliateCommissionQuota(
		base.Session(&gorm.Session{}).Where("status = ?", AffiliateCommissionStatusTransferred),
	)
	if err != nil {
		return AffiliateCommissionSummary{}, err
	}
	reversed, err := sumAffiliateCommissionQuota(
		base.Session(&gorm.Session{}).Where("status = ?", AffiliateCommissionStatusReversed),
	)
	if err != nil {
		return AffiliateCommissionSummary{}, err
	}
	lifetime, err := addInt64Checked(pending, available)
	if err == nil {
		lifetime, err = addInt64Checked(lifetime, transferred)
	}
	if err != nil {
		return AffiliateCommissionSummary{}, err
	}
	return AffiliateCommissionSummary{
		PendingQuota:     pending,
		AvailableQuota:   available,
		TransferredQuota: transferred,
		LifetimeQuota:    lifetime,
		ReversedQuota:    reversed,
	}, nil
}

func GetAffiliateProfile(userId int) (AffiliateProfile, error) {
	if userId <= 0 {
		return AffiliateProfile{}, ErrAffiliateProfileNotFound
	}
	var profile AffiliateProfile
	err := DB.Where("user_id = ?", userId).First(&profile).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return *newAffiliateProfile(userId), nil
	}
	return profile, err
}

func CountAffiliateReferrals(userId int) (int64, error) {
	if userId <= 0 {
		return 0, ErrAffiliateProfileNotFound
	}
	var ledgerCount int64
	if err := DB.Model(&AffiliateReferral{}).Where("inviter_user_id = ?", userId).Count(&ledgerCount).Error; err != nil {
		return 0, err
	}
	var user User
	if err := DB.Select("id", "aff_count").Where("id = ?", userId).First(&user).Error; err != nil {
		return 0, err
	}
	if int64(user.AffCount) > ledgerCount {
		return int64(user.AffCount), nil
	}
	return ledgerCount, nil
}

func GetReferredVerifiedEpayAmount(userId int, currency string) (int64, error) {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if userId <= 0 || len(currency) != 3 {
		return 0, ErrAffiliateConfigInvalid
	}
	var total int64
	err := verifiedEpayPayments(DB, currency).
		Select("COALESCE(SUM(top_ups.paid_amount_minor), 0)").
		Joins("JOIN affiliate_referrals ON affiliate_referrals.referred_user_id = top_ups.user_id").
		Where("affiliate_referrals.inviter_user_id = ?", userId).
		Scan(&total).Error
	return total, err
}

type AffiliateCommissionPage struct {
	Items    []AffiliateCommission `json:"items"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"page_size"`
	Total    int64                 `json:"total"`
}

func paginateAffiliateCommissions(query *gorm.DB, page int, pageSize int) (AffiliateCommissionPage, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return AffiliateCommissionPage{}, err
	}
	items := make([]AffiliateCommission, 0, pageSize)
	if err := query.Order("affiliate_commissions.id desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&items).Error; err != nil {
		return AffiliateCommissionPage{}, err
	}
	return AffiliateCommissionPage{
		Items:    items,
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	}, nil
}

func ListAffiliateCommissions(userId int, page int, pageSize int, now int64) (AffiliateCommissionPage, error) {
	if userId <= 0 {
		return AffiliateCommissionPage{}, ErrAffiliateProfileNotFound
	}
	if now <= 0 {
		now = common.GetTimestamp()
	}
	result, err := paginateAffiliateCommissions(
		DB.Model(&AffiliateCommission{}).
			Select("affiliate_commissions.*, users.username AS referred_username").
			Joins("LEFT JOIN users ON users.id = affiliate_commissions.referred_user_id").
			Where("affiliate_commissions.agent_user_id = ?", userId),
		page,
		pageSize,
	)
	if err != nil {
		return AffiliateCommissionPage{}, err
	}
	for index := range result.Items {
		if result.Items[index].Status == AffiliateCommissionStatusPending && result.Items[index].AvailableAt <= now {
			result.Items[index].Status = AffiliateCommissionStatusAvailable
		}
	}
	return result, nil
}

type AffiliateAdminCommissionFilter struct {
	AgentUserId    int
	ReferredUserId int
	Status         string
	Page           int
	PageSize       int
}

type AffiliateUserAdminSummary struct {
	Access                  string `json:"access"`
	Status                  string `json:"status"`
	ActivatedAt             int64  `json:"activated_at"`
	Eligible                bool   `json:"eligible"`
	VerifiedAmountMinor     int64  `json:"verified_amount_minor,string"`
	LifetimeCommissionQuota int64  `json:"lifetime_commission_quota,string"`
	Currency                string `json:"currency"`
}

func AttachAffiliateUserAdminSummaries(
	users []*User,
	currency string,
	thresholdMinor int64,
	programEnabled bool,
) error {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if len(users) == 0 {
		return nil
	}
	if len(currency) != 3 || thresholdMinor < 0 {
		return ErrAffiliateConfigInvalid
	}
	userIds := make([]int, 0, len(users))
	usersById := make(map[int]*User, len(users))
	for _, user := range users {
		if user == nil || user.Id <= 0 {
			continue
		}
		user.AffiliateSummary = &AffiliateUserAdminSummary{
			Access:   AffiliateAccessInherit,
			Status:   AffiliateStatusInactive,
			Currency: currency,
		}
		userIds = append(userIds, user.Id)
		usersById[user.Id] = user
	}
	if len(userIds) == 0 {
		return nil
	}

	profiles := make([]AffiliateProfile, 0, len(userIds))
	if err := DB.Where("user_id IN ?", userIds).Find(&profiles).Error; err != nil {
		return err
	}
	for _, profile := range profiles {
		if user := usersById[profile.UserId]; user != nil {
			user.AffiliateSummary.Access = profile.Access
			user.AffiliateSummary.Status = profile.Status
			user.AffiliateSummary.ActivatedAt = profile.ActivatedAt
		}
	}

	type userAmount struct {
		UserId int
		Amount int64
	}
	verified := make([]userAmount, 0, len(userIds))
	if err := verifiedEpayPayments(DB, currency).
		Select("top_ups.user_id, SUM(top_ups.paid_amount_minor) AS amount").
		Where("top_ups.user_id IN ?", userIds).
		Group("top_ups.user_id").
		Scan(&verified).Error; err != nil {
		return err
	}
	for _, row := range verified {
		if user := usersById[row.UserId]; user != nil {
			user.AffiliateSummary.VerifiedAmountMinor = row.Amount
		}
	}

	commissions := make([]userAmount, 0, len(userIds))
	if err := DB.Model(&AffiliateCommission{}).
		Select("agent_user_id AS user_id, SUM(reward_quota) AS amount").
		Where("agent_user_id IN ? AND status <> ?", userIds, AffiliateCommissionStatusReversed).
		Group("agent_user_id").
		Scan(&commissions).Error; err != nil {
		return err
	}
	for _, row := range commissions {
		if user := usersById[row.UserId]; user != nil {
			user.AffiliateSummary.LifetimeCommissionQuota = row.Amount
		}
	}
	for _, user := range usersById {
		summary := user.AffiliateSummary
		summary.Eligible = programEnabled && summary.Access != AffiliateAccessDeny &&
			(summary.Status == AffiliateStatusActive || summary.Access == AffiliateAccessAllow || summary.VerifiedAmountMinor >= thresholdMinor)
	}
	return nil
}

func ListAffiliateCommissionsForAdmin(filter AffiliateAdminCommissionFilter) (AffiliateCommissionPage, error) {
	query := DB.Model(&AffiliateCommission{})
	if filter.AgentUserId > 0 {
		query = query.Where("agent_user_id = ?", filter.AgentUserId)
	}
	if filter.ReferredUserId > 0 {
		query = query.Where("referred_user_id = ?", filter.ReferredUserId)
	}
	if strings.TrimSpace(filter.Status) != "" {
		query = query.Where("status = ?", strings.TrimSpace(filter.Status))
	}
	return paginateAffiliateCommissions(query, filter.Page, filter.PageSize)
}

func wrapAffiliateError(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", operation, err)
}
