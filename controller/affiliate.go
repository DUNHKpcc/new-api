package controller

import (
	"errors"
	"net/url"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
)

func attachAffiliateAdminSummaries(users []*model.User) error {
	program, err := model.GetAffiliateProgramConfigSnapshot()
	if err != nil {
		return err
	}
	return model.AttachAffiliateUserAdminSummaries(
		users,
		program.EpayCurrency,
		program.Setting.QualificationThresholdMinor,
		program.Setting.CommissionEnabled && program.PaymentComplianceConfirmed,
	)
}

type affiliateOverviewResponse struct {
	Invite        affiliateInviteOverview        `json:"invite"`
	SignupRewards affiliateSignupRewardsOverview `json:"signup_rewards"`
	Program       affiliateProgramOverview       `json:"program"`
	Commissions   affiliateCommissionsOverview   `json:"commissions"`
}

type affiliateInviteOverview struct {
	Code  string `json:"code"`
	Link  string `json:"link"`
	Count int64  `json:"count"`
}

type affiliateSignupRewardsOverview struct {
	Enabled            bool   `json:"enabled"`
	InviterRewardQuota string `json:"inviter_reward_quota"`
	InviteeRewardQuota string `json:"invitee_reward_quota"`
	AvailableQuota     string `json:"available_quota"`
	LifetimeQuota      string `json:"lifetime_quota"`
}

type affiliateProgramOverview struct {
	Enabled                     bool   `json:"enabled"`
	Access                      string `json:"access"`
	Status                      string `json:"status"`
	ActivatedAt                 int64  `json:"activated_at"`
	Eligible                    bool   `json:"eligible"`
	QualificationSource         string `json:"qualification_source"`
	VerifiedAmountMinor         string `json:"verified_amount_minor"`
	QualificationThresholdMinor string `json:"qualification_threshold_minor"`
	RemainingAmountMinor        string `json:"remaining_amount_minor"`
	Currency                    string `json:"currency"`
	CommissionRateBPS           int64  `json:"commission_rate_bps"`
	CommissionWaitDays          int64  `json:"commission_wait_days"`
	CommissionDebtQuota         string `json:"commission_debt_quota"`
	ConfigVersion               int64  `json:"config_version"`
}

type affiliateCommissionsOverview struct {
	ReferredPaidAmountMinor string `json:"referred_paid_amount_minor"`
	PendingQuota            string `json:"pending_quota"`
	AvailableQuota          string `json:"available_quota"`
	TransferredQuota        string `json:"transferred_quota"`
	LifetimeQuota           string `json:"lifetime_quota"`
	ReversedQuota           string `json:"reversed_quota"`
}

func GetAffiliateOverview(c *gin.Context) {
	userId := c.GetInt("id")
	user, err := model.GetUserById(userId, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	affCode, err := model.EnsureUserAffiliateCode(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	referralCount, err := model.CountAffiliateReferrals(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	profile, err := model.GetAffiliateProfile(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	program, err := model.GetAffiliateProgramConfigSnapshot()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	setting := program.Setting
	currency := program.EpayCurrency
	verifiedAmount, err := model.GetVerifiedEpayAmount(userId, currency)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	referredPaidAmount, err := model.GetReferredVerifiedEpayAmount(userId, currency)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	commissionSummary, err := model.GetAffiliateCommissionSummary(userId, common.GetTimestamp())
	if err != nil {
		common.ApiError(c, err)
		return
	}

	programEnabled := setting.CommissionEnabled && program.PaymentComplianceConfirmed
	eligible := false
	qualificationSource := "threshold"
	switch {
	case !programEnabled:
		qualificationSource = "program_disabled"
	case profile.Access == model.AffiliateAccessDeny:
		qualificationSource = "denied"
	case profile.Status == model.AffiliateStatusActive:
		eligible = true
		qualificationSource = "activated"
	case profile.Access == model.AffiliateAccessAllow:
		eligible = true
		qualificationSource = "admin_allow"
	case verifiedAmount >= setting.QualificationThresholdMinor:
		eligible = true
		qualificationSource = "verified_topup"
	}
	remainingAmount := setting.QualificationThresholdMinor - verifiedAmount
	if remainingAmount < 0 || eligible {
		remainingAmount = 0
	}
	invitePath := "/sign-up?aff=" + url.QueryEscape(affCode)
	inviteLink := strings.TrimRight(system_setting.ServerAddress, "/") + invitePath
	if strings.TrimSpace(system_setting.ServerAddress) == "" {
		inviteLink = invitePath
	}

	common.ApiSuccess(c, affiliateOverviewResponse{
		Invite: affiliateInviteOverview{
			Code:  affCode,
			Link:  inviteLink,
			Count: referralCount,
		},
		SignupRewards: affiliateSignupRewardsOverview{
			Enabled:            setting.RegistrationRewardEnabled && operation_setting.IsPaymentComplianceConfirmed(),
			InviterRewardQuota: strconv.FormatInt(setting.InviterRewardQuota, 10),
			InviteeRewardQuota: strconv.FormatInt(setting.InviteeRewardQuota, 10),
			AvailableQuota:     strconv.Itoa(user.AffQuota),
			LifetimeQuota:      strconv.Itoa(user.AffHistoryQuota),
		},
		Program: affiliateProgramOverview{
			Enabled:                     programEnabled,
			Access:                      profile.Access,
			Status:                      profile.Status,
			ActivatedAt:                 profile.ActivatedAt,
			Eligible:                    eligible,
			QualificationSource:         qualificationSource,
			VerifiedAmountMinor:         strconv.FormatInt(verifiedAmount, 10),
			QualificationThresholdMinor: strconv.FormatInt(setting.QualificationThresholdMinor, 10),
			RemainingAmountMinor:        strconv.FormatInt(remainingAmount, 10),
			Currency:                    currency,
			CommissionRateBPS:           setting.CommissionRateBPS,
			CommissionWaitDays:          setting.CommissionWaitDays,
			CommissionDebtQuota:         strconv.FormatInt(profile.CommissionDebtQuota, 10),
			ConfigVersion:               setting.Version,
		},
		Commissions: affiliateCommissionsOverview{
			ReferredPaidAmountMinor: strconv.FormatInt(referredPaidAmount, 10),
			PendingQuota:            strconv.FormatInt(commissionSummary.PendingQuota, 10),
			AvailableQuota:          strconv.FormatInt(commissionSummary.AvailableQuota, 10),
			TransferredQuota:        strconv.FormatInt(commissionSummary.TransferredQuota, 10),
			LifetimeQuota:           strconv.FormatInt(commissionSummary.LifetimeQuota, 10),
			ReversedQuota:           strconv.FormatInt(commissionSummary.ReversedQuota, 10),
		},
	})
}

func ActivateAffiliate(c *gin.Context) {
	result, err := model.ActivateAffiliate(c.GetInt("id"))
	if err != nil {
		switch {
		case errors.Is(err, model.ErrAffiliateProgramDisabled):
			common.ApiErrorMsg(c, "代理充值佣金当前未开启")
		case errors.Is(err, model.ErrAffiliateAccessDenied):
			common.ApiErrorMsg(c, "当前账号不可开启代理充值佣金")
		case errors.Is(err, model.ErrAffiliateQualificationNotMet):
			common.ApiErrorMsg(c, "已核验充值总额尚未达到开启门槛")
		default:
			common.ApiError(c, err)
		}
		return
	}
	common.ApiSuccess(c, result)
}

type affiliateTransferRequest struct {
	IdempotencyKey string `json:"idempotency_key"`
	Quota          string `json:"quota"`
}

func TransferAffiliateInviteRewards(c *gin.Context) {
	var request affiliateTransferRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	quota, err := strconv.ParseInt(strings.TrimSpace(request.Quota), 10, 64)
	if err != nil {
		common.ApiErrorMsg(c, "转入额度无效")
		return
	}
	result, err := model.TransferAffiliateSignupRewards(c.GetInt("id"), quota, request.IdempotencyKey)
	if err != nil {
		if writePaymentComplianceError(c, err) {
			return
		}
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func TransferAffiliateCommissions(c *gin.Context) {
	var request affiliateTransferRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	result, err := model.TransferAvailableAffiliateCommissions(c.GetInt("id"), request.IdempotencyKey, common.GetTimestamp())
	if err != nil {
		if writePaymentComplianceError(c, err) {
			return
		}
		switch {
		case errors.Is(err, model.ErrAffiliateAccessDenied):
			common.ApiErrorMsg(c, "当前账号不可转入代理佣金")
		case errors.Is(err, model.ErrAffiliateCommissionNothingToTransfer):
			common.ApiErrorMsg(c, "当前没有可转入的代理佣金")
		default:
			common.ApiError(c, err)
		}
		return
	}
	common.ApiSuccess(c, result)
}

type affiliateCommissionItemResponse struct {
	Id                    string `json:"id"`
	ReferredUser          string `json:"referred_user"`
	PaidAmountMinor       string `json:"paid_amount_minor"`
	PaidCurrency          string `json:"paid_currency"`
	PurchasedQuota        string `json:"purchased_quota"`
	CommissionRateBPS     int64  `json:"commission_rate_bps"`
	CommissionAmountMinor string `json:"commission_amount_minor"`
	RewardQuota           string `json:"reward_quota"`
	DebtOffsetQuota       string `json:"debt_offset_quota"`
	Status                string `json:"status"`
	AvailableAt           int64  `json:"available_at"`
	TransferredAt         int64  `json:"transferred_at"`
	ReversedAt            int64  `json:"reversed_at"`
	ReverseReason         string `json:"reverse_reason,omitempty"`
	CreatedAt             int64  `json:"created_at"`
}

func ListAffiliateCommissions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	result, err := model.ListAffiliateCommissions(c.GetInt("id"), page, pageSize, common.GetTimestamp())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	items := make([]affiliateCommissionItemResponse, 0, len(result.Items))
	for _, commission := range result.Items {
		reverseReason := ""
		if commission.Status == model.AffiliateCommissionStatusReversed {
			reverseReason = commission.ReverseReason
		}
		items = append(items, affiliateCommissionItemResponse{
			Id:                    strconv.FormatInt(commission.Id, 10),
			ReferredUser:          common.MaskUsername(commission.ReferredUsername),
			PaidAmountMinor:       strconv.FormatInt(commission.PaidAmountMinor, 10),
			PaidCurrency:          commission.PaidCurrency,
			PurchasedQuota:        strconv.Itoa(commission.PurchasedQuota),
			CommissionRateBPS:     commission.CommissionRateBPS,
			CommissionAmountMinor: strconv.FormatInt(commission.CommissionAmountMinor, 10),
			RewardQuota:           strconv.Itoa(commission.RewardQuota),
			DebtOffsetQuota:       strconv.Itoa(commission.DebtOffsetQuota),
			Status:                commission.Status,
			AvailableAt:           commission.AvailableAt,
			TransferredAt:         commission.TransferredAt,
			ReversedAt:            commission.ReversedAt,
			ReverseReason:         reverseReason,
			CreatedAt:             commission.CreatedAt,
		})
	}
	common.ApiSuccess(c, gin.H{
		"items":     items,
		"page":      result.Page,
		"page_size": result.PageSize,
		"total":     result.Total,
	})
}

type affiliateAccessRequest struct {
	Access string `json:"access"`
	Reason string `json:"reason"`
}

func UpdateAffiliateAccess(c *gin.Context) {
	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil || userId <= 0 {
		common.ApiErrorMsg(c, "用户 ID 无效")
		return
	}
	var request affiliateAccessRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	result, err := model.SetAffiliateAccess(userId, c.GetInt("id"), request.Access, request.Reason)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	markAuditLogged(c)
	common.ApiSuccess(c, result)
}

type adminAffiliateCommissionItemResponse struct {
	Id                    string  `json:"id"`
	AgentUserId           int     `json:"agent_user_id"`
	ReferredUserId        int     `json:"referred_user_id"`
	TopUpId               int     `json:"topup_id"`
	PaidAmountMinor       string  `json:"paid_amount_minor"`
	PaidCurrency          string  `json:"paid_currency"`
	PurchasedQuota        string  `json:"purchased_quota"`
	UnitPriceSnapshot     string  `json:"unit_price_snapshot"`
	QuotaPerUnitSnapshot  string  `json:"quota_per_unit_snapshot"`
	CommissionRateBPS     int64   `json:"commission_rate_bps"`
	CommissionAmountMinor string  `json:"commission_amount_minor"`
	GrossRewardQuota      string  `json:"gross_reward_quota"`
	DebtOffsetQuota       string  `json:"debt_offset_quota"`
	RewardQuota           string  `json:"reward_quota"`
	Status                string  `json:"status"`
	AvailableAt           int64   `json:"available_at"`
	TransferId            *string `json:"transfer_id,omitempty"`
	TransferredAt         int64   `json:"transferred_at"`
	ReversedAt            int64   `json:"reversed_at"`
	ReversedBy            int     `json:"reversed_by"`
	ReverseReason         string  `json:"reverse_reason"`
	ConfigVersion         int64   `json:"config_version"`
	CreatedAt             int64   `json:"created_at"`
}

type adminAffiliateCommissionPageResponse struct {
	Items    []adminAffiliateCommissionItemResponse `json:"items"`
	Page     int                                    `json:"page"`
	PageSize int                                    `json:"page_size"`
	Total    int64                                  `json:"total"`
}

func AdminListAffiliateCommissions(c *gin.Context) {
	agentUserId := 0
	if rawAgentUserId, exists := c.GetQuery("agent_user_id"); exists {
		parsed, err := strconv.Atoi(rawAgentUserId)
		if err != nil || parsed <= 0 {
			common.ApiErrorMsg(c, "代理用户 ID 无效")
			return
		}
		agentUserId = parsed
	}
	referredUserId := 0
	if rawReferredUserId, exists := c.GetQuery("referred_user_id"); exists {
		parsed, err := strconv.Atoi(rawReferredUserId)
		if err != nil || parsed <= 0 {
			common.ApiErrorMsg(c, "受邀用户 ID 无效")
			return
		}
		referredUserId = parsed
	}
	status := strings.TrimSpace(c.Query("status"))
	if status != "" &&
		status != model.AffiliateCommissionStatusPending &&
		status != model.AffiliateCommissionStatusAvailable &&
		status != model.AffiliateCommissionStatusTransferred &&
		status != model.AffiliateCommissionStatusReversed {
		common.ApiErrorMsg(c, "佣金状态无效")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	result, err := model.ListAffiliateCommissionsForAdmin(model.AffiliateAdminCommissionFilter{
		AgentUserId:    agentUserId,
		ReferredUserId: referredUserId,
		Status:         status,
		Page:           page,
		PageSize:       pageSize,
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	items := make([]adminAffiliateCommissionItemResponse, 0, len(result.Items))
	for _, commission := range result.Items {
		var transferId *string
		if commission.TransferId != nil {
			formatted := strconv.FormatInt(*commission.TransferId, 10)
			transferId = &formatted
		}
		items = append(items, adminAffiliateCommissionItemResponse{
			Id:                    strconv.FormatInt(commission.Id, 10),
			AgentUserId:           commission.AgentUserId,
			ReferredUserId:        commission.ReferredUserId,
			TopUpId:               commission.TopUpId,
			PaidAmountMinor:       strconv.FormatInt(commission.PaidAmountMinor, 10),
			PaidCurrency:          commission.PaidCurrency,
			PurchasedQuota:        strconv.Itoa(commission.PurchasedQuota),
			UnitPriceSnapshot:     commission.UnitPriceSnapshot,
			QuotaPerUnitSnapshot:  commission.QuotaPerUnitSnapshot,
			CommissionRateBPS:     commission.CommissionRateBPS,
			CommissionAmountMinor: strconv.FormatInt(commission.CommissionAmountMinor, 10),
			GrossRewardQuota:      strconv.Itoa(commission.GrossRewardQuota),
			DebtOffsetQuota:       strconv.Itoa(commission.DebtOffsetQuota),
			RewardQuota:           strconv.Itoa(commission.RewardQuota),
			Status:                commission.Status,
			AvailableAt:           commission.AvailableAt,
			TransferId:            transferId,
			TransferredAt:         commission.TransferredAt,
			ReversedAt:            commission.ReversedAt,
			ReversedBy:            commission.ReversedBy,
			ReverseReason:         commission.ReverseReason,
			ConfigVersion:         commission.ConfigVersion,
			CreatedAt:             commission.CreatedAt,
		})
	}
	common.ApiSuccess(c, adminAffiliateCommissionPageResponse{
		Items:    items,
		Page:     result.Page,
		PageSize: result.PageSize,
		Total:    result.Total,
	})
}

type affiliateReverseRequest struct {
	Reason string `json:"reason"`
}

func AdminReverseAffiliateCommission(c *gin.Context) {
	commissionId, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || commissionId <= 0 {
		common.ApiErrorMsg(c, "佣金 ID 无效")
		return
	}
	var request affiliateReverseRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	result, err := model.ReverseAffiliateCommission(commissionId, c.GetInt("id"), request.Reason, common.GetTimestamp())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	markAuditLogged(c)
	common.ApiSuccess(c, result)
}

type affiliateSettingPayload struct {
	RegistrationRewardEnabled   bool   `json:"registration_reward_enabled"`
	InviterRewardQuota          int64  `json:"inviter_reward_quota"`
	InviteeRewardQuota          int64  `json:"invitee_reward_quota"`
	CommissionEnabled           bool   `json:"commission_enabled"`
	QualificationThresholdMinor string `json:"qualification_threshold_minor"`
	CommissionRateBPS           int64  `json:"commission_rate_bps"`
	CommissionWaitDays          int64  `json:"commission_wait_days"`
	Version                     int64  `json:"version"`
}

func UpdateAffiliateSetting(c *gin.Context) {
	var request affiliateSettingPayload
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	thresholdMinor, err := strconv.ParseInt(strings.TrimSpace(request.QualificationThresholdMinor), 10, 64)
	if err != nil {
		common.ApiErrorMsg(c, "代理资格门槛无效")
		return
	}
	normalized, err := operation_setting.NormalizeAffiliateSetting(operation_setting.AffiliateSetting{
		RegistrationRewardEnabled:   request.RegistrationRewardEnabled,
		InviterRewardQuota:          request.InviterRewardQuota,
		InviteeRewardQuota:          request.InviteeRewardQuota,
		CommissionEnabled:           request.CommissionEnabled,
		QualificationThresholdMinor: thresholdMinor,
		CommissionRateBPS:           request.CommissionRateBPS,
		CommissionWaitDays:          request.CommissionWaitDays,
		Version:                     request.Version,
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	result, err := model.UpdateAffiliateSettingOption(request.Version, normalized, c.GetInt("id"))
	if err != nil {
		if writePaymentComplianceError(c, err) {
			return
		}
		if errors.Is(err, model.ErrOptionVersionConflict) {
			common.ApiErrorMsg(c, "代理配置已被其他管理员更新，请刷新后重试")
			return
		}
		common.ApiError(c, err)
		return
	}
	markAuditLogged(c)
	common.ApiSuccess(c, affiliateSettingPayload{
		RegistrationRewardEnabled:   result.Current.RegistrationRewardEnabled,
		InviterRewardQuota:          result.Current.InviterRewardQuota,
		InviteeRewardQuota:          result.Current.InviteeRewardQuota,
		CommissionEnabled:           result.Current.CommissionEnabled,
		QualificationThresholdMinor: strconv.FormatInt(result.Current.QualificationThresholdMinor, 10),
		CommissionRateBPS:           result.Current.CommissionRateBPS,
		CommissionWaitDays:          result.Current.CommissionWaitDays,
		Version:                     result.Current.Version,
	})
}
