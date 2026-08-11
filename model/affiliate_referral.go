package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"gorm.io/gorm"
)

const (
	AffiliateCodeLength             = 8
	affiliateCodeGenerationAttempts = 5

	AffiliateSignupRewardRoleInviter = "inviter"
	AffiliateSignupRewardRoleInvitee = "invitee"

	AffiliateSignupRewardStatusGranted = "granted"

	AffiliateEventActionReferralBound           = "affiliate.referral_bound"
	AffiliateEventActionSignupRewardGranted     = "affiliate.signup_reward_granted"
	AffiliateEventActionSignupRewardTransferred = "affiliate.signup_reward_transferred"
)

var (
	ErrAffiliateReferralInvalid         = errors.New("affiliate referral is invalid")
	ErrAffiliateReferredUserNotFound    = errors.New("affiliate referred user was not found")
	ErrAffiliateInviterNotFound         = errors.New("affiliate inviter was not found")
	ErrAffiliateInviterDisabled         = errors.New("affiliate inviter is disabled")
	ErrAffiliateSelfReferral            = errors.New("affiliate self-referral is not allowed")
	ErrAffiliateReferralConflict        = errors.New("affiliate referral already belongs to another inviter")
	ErrAffiliateRewardConfigInvalid     = errors.New("affiliate signup reward configuration is invalid")
	ErrAffiliateRewardBalanceOverflow   = errors.New("affiliate signup reward would overflow a quota balance")
	ErrAffiliateCodeGenerationExhausted = errors.New("could not generate a unique affiliate code")

	ErrAffiliateSignupRewardTransferInvalid      = errors.New("affiliate signup reward transfer is invalid")
	ErrAffiliateSignupRewardTransferUserNotFound = errors.New("affiliate signup reward transfer user was not found")
	ErrAffiliateSignupRewardInsufficient         = errors.New("affiliate signup reward balance is insufficient")
	ErrAffiliateSignupRewardTransferOverflow     = errors.New("affiliate signup reward transfer would overflow a quota balance")
	ErrAffiliateSignupRewardTransferConflict     = errors.New("affiliate signup reward transfer idempotency key conflicts with another amount")
)

// AffiliateReferral is the immutable ownership record for a direct referral.
// ReferredUserId is unique so a user cannot be attached to multiple inviters.
type AffiliateReferral struct {
	Id              int64  `json:"id" gorm:"primaryKey"`
	InviterUserId   int    `json:"inviter_user_id" gorm:"not null;index"`
	ReferredUserId  int    `json:"referred_user_id" gorm:"not null;uniqueIndex"`
	AffCodeSnapshot string `json:"aff_code_snapshot" gorm:"type:varchar(32);not null"`
	BoundAt         int64  `json:"bound_at" gorm:"not null;autoCreateTime"`
}

func (AffiliateReferral) TableName() string {
	return "affiliate_referrals"
}

// AffiliateSignupReward records one beneficiary side of a signup reward.
// The composite unique index makes inviter and invitee grants independently
// idempotent for a referral.
type AffiliateSignupReward struct {
	Id                int64  `json:"id" gorm:"primaryKey"`
	ReferralId        int64  `json:"referral_id" gorm:"not null;uniqueIndex:ux_aff_signup_reward_role,priority:1"`
	BeneficiaryUserId int    `json:"beneficiary_user_id" gorm:"not null;index"`
	BeneficiaryRole   string `json:"beneficiary_role" gorm:"type:varchar(16);not null;uniqueIndex:ux_aff_signup_reward_role,priority:2"`
	RewardQuota       int64  `json:"reward_quota" gorm:"not null"`
	Status            string `json:"status" gorm:"type:varchar(16);not null"`
	ConfigVersion     int64  `json:"config_version" gorm:"not null"`
	CreatedAt         int64  `json:"created_at" gorm:"not null;autoCreateTime"`
}

func (AffiliateSignupReward) TableName() string {
	return "affiliate_signup_rewards"
}

// AffiliateSignupRewardTransfer is the immutable receipt for moving signup
// rewards into a user's main quota. Transfer behavior is implemented separately.
type AffiliateSignupRewardTransfer struct {
	Id               int64  `json:"id" gorm:"primaryKey"`
	UserId           int    `json:"user_id" gorm:"not null;index;uniqueIndex:ux_aff_reward_transfer_key,priority:1"`
	IdempotencyKey   string `json:"idempotency_key" gorm:"type:varchar(64);not null;uniqueIndex:ux_aff_reward_transfer_key,priority:2"`
	TransferredQuota int64  `json:"transferred_quota" gorm:"not null"`
	AffQuotaBefore   int64  `json:"aff_quota_before" gorm:"not null"`
	AffQuotaAfter    int64  `json:"aff_quota_after" gorm:"not null"`
	MainQuotaBefore  int64  `json:"main_quota_before" gorm:"not null"`
	MainQuotaAfter   int64  `json:"main_quota_after" gorm:"not null"`
	CreatedAt        int64  `json:"created_at" gorm:"not null;autoCreateTime"`
}

func (AffiliateSignupRewardTransfer) TableName() string {
	return "affiliate_signup_reward_transfers"
}

// AffiliateSignupRewardConfigSnapshot is passed by value so one registration
// can only observe one immutable configuration version.
type AffiliateSignupRewardConfigSnapshot struct {
	Enabled            bool
	InviterRewardQuota int64
	InviteeRewardQuota int64
	ConfigVersion      int64
}

type AffiliateSignupRewardTransferResult struct {
	Transfer         AffiliateSignupRewardTransfer `json:"transfer"`
	AlreadyCompleted bool                          `json:"already_completed"`
}

// ResolveAffiliateInviterIdByCode resolves both legacy short codes and newly
// generated codes, while rejecting missing or disabled inviters explicitly.
func ResolveAffiliateInviterIdByCode(affCode string) (int, error) {
	affCode = strings.TrimSpace(affCode)
	if affCode == "" {
		return 0, nil
	}

	var inviter User
	if err := DB.Select("id", "status").Where("aff_code = ?", affCode).First(&inviter).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, ErrAffiliateInviterNotFound
		}
		return 0, err
	}
	if inviter.Status != common.UserStatusEnabled {
		return 0, ErrAffiliateInviterDisabled
	}
	return inviter.Id, nil
}

func createUserWithUniqueAffiliateCode(tx *gorm.DB, user *User, generate func() string) error {
	if tx == nil || user == nil || generate == nil {
		return ErrAffiliateReferralInvalid
	}
	for attempt := 0; attempt < affiliateCodeGenerationAttempts; attempt++ {
		candidate := strings.TrimSpace(generate())
		if candidate == "" || len(candidate) > 32 {
			continue
		}
		user.AffCode = candidate
		savepoint := fmt.Sprintf("affiliate_code_create_%d", attempt)
		if err := tx.SavePoint(savepoint).Error; err != nil {
			return err
		}
		if err := tx.Create(user).Error; err == nil {
			return nil
		} else {
			createErr := err
			if err := tx.RollbackTo(savepoint).Error; err != nil {
				return err
			}
			user.Id = 0
			exists, err := affiliateCodeExistsWithTx(tx, candidate)
			if err != nil {
				return createErr
			}
			if !exists {
				return createErr
			}
		}
	}
	return ErrAffiliateCodeGenerationExhausted
}

func affiliateCodeExistsWithTx(tx *gorm.DB, affCode string) (bool, error) {
	var count int64
	err := tx.Model(&User{}).Where("aff_code = ?", affCode).Count(&count).Error
	return count > 0, err
}

// EnsureUserAffiliateCode returns an existing code unchanged, including legacy
// four-character codes, or atomically assigns a collision-resistant new code.
func EnsureUserAffiliateCode(userId int) (string, error) {
	return ensureUserAffiliateCodeWithGenerator(userId, func() string {
		return common.GetRandomString(AffiliateCodeLength)
	})
}

func ensureUserAffiliateCodeWithGenerator(userId int, generate func() string) (string, error) {
	if userId <= 0 || generate == nil {
		return "", ErrAffiliateReferralInvalid
	}

	var affCode string
	err := DB.Transaction(func(tx *gorm.DB) error {
		var user User
		if err := lockForUpdate(tx).Select("id", "aff_code").Where("id = ?", userId).First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrAffiliateReferredUserNotFound
			}
			return err
		}
		if user.AffCode != "" {
			affCode = user.AffCode
			return nil
		}

		for attempt := 0; attempt < affiliateCodeGenerationAttempts; attempt++ {
			candidate := strings.TrimSpace(generate())
			if candidate == "" || len(candidate) > 32 {
				continue
			}
			savepoint := fmt.Sprintf("affiliate_code_update_%d", attempt)
			if err := tx.SavePoint(savepoint).Error; err != nil {
				return err
			}
			result := tx.Model(&User{}).
				Where("id = ? AND aff_code = ?", userId, "").
				Update("aff_code", candidate)
			if result.Error == nil && result.RowsAffected == 1 {
				affCode = candidate
				return nil
			}
			if result.Error == nil {
				if err := tx.Select("aff_code").Where("id = ?", userId).First(&user).Error; err != nil {
					return err
				}
				if user.AffCode != "" {
					affCode = user.AffCode
					return nil
				}
				continue
			}

			updateErr := result.Error
			if err := tx.RollbackTo(savepoint).Error; err != nil {
				return err
			}
			exists, err := affiliateCodeExistsWithTx(tx, candidate)
			if err != nil {
				return updateErr
			}
			if !exists {
				return updateErr
			}
		}
		return ErrAffiliateCodeGenerationExhausted
	})
	return affCode, err
}

// TransferAffiliateSignupRewards atomically moves signup-reward quota into the
// main balance. Reusing the same key and amount returns the original receipt.
func TransferAffiliateSignupRewards(
	userId int,
	quota int64,
	idempotencyKey string,
) (AffiliateSignupRewardTransferResult, error) {
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if userId <= 0 || quota <= 0 || quota > int64(common.MaxQuota) ||
		idempotencyKey == "" || len(idempotencyKey) > 64 {
		return AffiliateSignupRewardTransferResult{}, ErrAffiliateSignupRewardTransferInvalid
	}

	var result AffiliateSignupRewardTransferResult
	err := DB.Transaction(func(tx *gorm.DB) error {
		confirmed, err := paymentComplianceConfirmedWithTx(tx)
		if err != nil {
			return err
		}
		if !confirmed {
			return ErrPaymentComplianceRequired
		}
		var user User
		if err := lockForUpdate(tx).
			Select("id", "quota", "aff_quota").
			Where("id = ?", userId).
			First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrAffiliateSignupRewardTransferUserNotFound
			}
			return err
		}

		var existing AffiliateSignupRewardTransfer
		err = tx.Where("user_id = ? AND idempotency_key = ?", userId, idempotencyKey).
			First(&existing).Error
		if err == nil {
			if existing.TransferredQuota != quota {
				return ErrAffiliateSignupRewardTransferConflict
			}
			result = AffiliateSignupRewardTransferResult{Transfer: existing, AlreadyCompleted: true}
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if int64(user.AffQuota) > int64(common.MaxQuota) || int64(user.Quota) > int64(common.MaxQuota) {
			return ErrAffiliateSignupRewardTransferOverflow
		}
		if int64(user.AffQuota) < quota {
			return ErrAffiliateSignupRewardInsufficient
		}
		if int64(user.Quota) > int64(common.MaxQuota)-quota {
			return ErrAffiliateSignupRewardTransferOverflow
		}

		transfer := AffiliateSignupRewardTransfer{
			UserId:           userId,
			IdempotencyKey:   idempotencyKey,
			TransferredQuota: quota,
			AffQuotaBefore:   int64(user.AffQuota),
			AffQuotaAfter:    int64(user.AffQuota) - quota,
			MainQuotaBefore:  int64(user.Quota),
			MainQuotaAfter:   int64(user.Quota) + quota,
			CreatedAt:        common.GetTimestamp(),
		}
		if err := tx.Create(&transfer).Error; err != nil {
			return err
		}
		update := tx.Model(&User{}).
			Where(
				"id = ? AND aff_quota >= ? AND quota <= ?",
				userId,
				quota,
				int64(common.MaxQuota)-quota,
			).
			Updates(map[string]interface{}{
				"aff_quota": gorm.Expr("aff_quota - ?", quota),
				"quota":     gorm.Expr("quota + ?", quota),
			})
		if update.Error != nil {
			return update.Error
		}
		if update.RowsAffected != 1 {
			return ErrAffiliateSignupRewardTransferConflict
		}
		if _, err := EnqueueAffiliateEventWithTx(
			tx,
			fmt.Sprintf("affiliate:signup_reward_transfer:%d", transfer.Id),
			userId,
			AffiliateEventActionSignupRewardTransferred,
			LogTypeSystem,
			fmt.Sprintf("注册邀请奖励转入主余额 %s", logger.LogQuota(int(transfer.TransferredQuota))),
			struct {
				TransferId       int64 `json:"transfer_id"`
				UserId           int   `json:"user_id"`
				TransferredQuota int64 `json:"transferred_quota"`
				AffQuotaBefore   int64 `json:"aff_quota_before"`
				AffQuotaAfter    int64 `json:"aff_quota_after"`
				MainQuotaBefore  int64 `json:"main_quota_before"`
				MainQuotaAfter   int64 `json:"main_quota_after"`
			}{
				TransferId:       transfer.Id,
				UserId:           transfer.UserId,
				TransferredQuota: transfer.TransferredQuota,
				AffQuotaBefore:   transfer.AffQuotaBefore,
				AffQuotaAfter:    transfer.AffQuotaAfter,
				MainQuotaBefore:  transfer.MainQuotaBefore,
				MainQuotaAfter:   transfer.MainQuotaAfter,
			},
			transfer.CreatedAt,
		); err != nil {
			return err
		}
		result = AffiliateSignupRewardTransferResult{Transfer: transfer}
		return nil
	})
	if err != nil {
		return AffiliateSignupRewardTransferResult{}, err
	}
	if err := invalidateUserCache(userId); err != nil {
		common.SysError(fmt.Sprintf(
			"failed to invalidate user cache after affiliate signup reward transfer: user_id=%d transfer_id=%d error=%v",
			userId,
			result.Transfer.Id,
			err,
		))
	}
	return result, nil
}

func enqueueAffiliateSignupRewardGrantedWithTx(tx *gorm.DB, reward *AffiliateSignupReward) error {
	if reward == nil || reward.Id <= 0 {
		return ErrAffiliateReferralInvalid
	}
	_, err := EnqueueAffiliateEventWithTx(
		tx,
		fmt.Sprintf("affiliate:signup_reward:%d", reward.Id),
		reward.BeneficiaryUserId,
		AffiliateEventActionSignupRewardGranted,
		LogTypeSystem,
		fmt.Sprintf("注册邀请奖励到账 %s", logger.LogQuota(int(reward.RewardQuota))),
		struct {
			RewardId          int64  `json:"reward_id"`
			ReferralId        int64  `json:"referral_id"`
			BeneficiaryUserId int    `json:"beneficiary_user_id"`
			BeneficiaryRole   string `json:"beneficiary_role"`
			RewardQuota       int64  `json:"reward_quota"`
			ConfigVersion     int64  `json:"config_version"`
		}{
			RewardId:          reward.Id,
			ReferralId:        reward.ReferralId,
			BeneficiaryUserId: reward.BeneficiaryUserId,
			BeneficiaryRole:   reward.BeneficiaryRole,
			RewardQuota:       reward.RewardQuota,
			ConfigVersion:     reward.ConfigVersion,
		},
		reward.CreatedAt,
	)
	return err
}

// ApplyAffiliateReferralWithTx establishes a referral after referredUser has
// been created by tx. The caller owns the transaction: every relationship,
// reward, and compatibility-counter update commits or rolls back together.
func ApplyAffiliateReferralWithTx(
	tx *gorm.DB,
	referredUser *User,
	inviterUserId int,
	config AffiliateSignupRewardConfigSnapshot,
) (*AffiliateReferral, error) {
	if tx == nil || referredUser == nil || referredUser.Id <= 0 || inviterUserId <= 0 {
		return nil, ErrAffiliateReferralInvalid
	}
	if referredUser.Id == inviterUserId {
		return nil, ErrAffiliateSelfReferral
	}
	if config.InviterRewardQuota < 0 || config.InviterRewardQuota > int64(common.MaxQuota) ||
		config.InviteeRewardQuota < 0 || config.InviteeRewardQuota > int64(common.MaxQuota) ||
		config.ConfigVersion < 1 {
		return nil, ErrAffiliateRewardConfigInvalid
	}

	var storedReferred User
	if err := lockForUpdate(tx).
		Select("id", "inviter_id", "quota").
		Where("id = ?", referredUser.Id).
		First(&storedReferred).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAffiliateReferredUserNotFound
		}
		return nil, err
	}

	// Lock the shared inviter before probing the referral unique index. On MySQL,
	// probing a missing referral FOR UPDATE takes a next-key gap lock; taking that
	// gap lock first lets concurrent registrations deadlock on the inviter row.
	var inviter User
	inviterErr := lockForUpdate(tx).
		Select("id", "status", "aff_code", "aff_quota", "aff_history").
		Where("id = ?", inviterUserId).
		First(&inviter).Error

	var existing AffiliateReferral
	err := lockForUpdate(tx).
		Where("referred_user_id = ?", referredUser.Id).
		First(&existing).Error
	if err == nil {
		if existing.InviterUserId != inviterUserId ||
			(storedReferred.InviterId != 0 && storedReferred.InviterId != inviterUserId) {
			return nil, ErrAffiliateReferralConflict
		}
		if storedReferred.InviterId == 0 {
			if err := tx.Model(&User{}).
				Where("id = ?", referredUser.Id).
				Update("inviter_id", inviterUserId).Error; err != nil {
				return nil, err
			}
		}
		referredUser.InviterId = inviterUserId
		return &existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if storedReferred.InviterId != 0 && storedReferred.InviterId != inviterUserId {
		return nil, ErrAffiliateReferralConflict
	}

	if inviterErr != nil {
		if errors.Is(inviterErr, gorm.ErrRecordNotFound) {
			return nil, ErrAffiliateInviterNotFound
		}
		return nil, inviterErr
	}
	if inviter.Status != common.UserStatusEnabled {
		return nil, ErrAffiliateInviterDisabled
	}
	if config.Enabled && config.InviterRewardQuota > 0 &&
		(int64(inviter.AffQuota) > int64(common.MaxQuota)-config.InviterRewardQuota ||
			int64(inviter.AffHistoryQuota) > int64(common.MaxQuota)-config.InviterRewardQuota) {
		return nil, ErrAffiliateRewardBalanceOverflow
	}
	if config.Enabled && config.InviteeRewardQuota > 0 &&
		int64(storedReferred.Quota) > int64(common.MaxQuota)-config.InviteeRewardQuota {
		return nil, ErrAffiliateRewardBalanceOverflow
	}

	referral := &AffiliateReferral{
		InviterUserId:   inviterUserId,
		ReferredUserId:  referredUser.Id,
		AffCodeSnapshot: inviter.AffCode,
		BoundAt:         common.GetTimestamp(),
	}
	if err := tx.Create(referral).Error; err != nil {
		return nil, err
	}
	if _, err := EnqueueAffiliateEventWithTx(
		tx,
		fmt.Sprintf("affiliate:referral:%d", referral.Id),
		inviterUserId,
		AffiliateEventActionReferralBound,
		LogTypeSystem,
		"邀请关系已建立",
		struct {
			ReferralId   int64  `json:"referral_id"`
			ReferredUser string `json:"referred_user"`
		}{
			ReferralId:   referral.Id,
			ReferredUser: fmt.Sprintf("us***%02d", referral.ReferredUserId%100),
		},
		referral.BoundAt,
	); err != nil {
		return nil, err
	}
	if err := tx.Model(&User{}).
		Where("id = ?", referredUser.Id).
		Update("inviter_id", inviterUserId).Error; err != nil {
		return nil, err
	}

	inviterUpdates := map[string]interface{}{
		"aff_count": gorm.Expr("aff_count + ?", 1),
	}
	if config.Enabled && config.InviterRewardQuota > 0 {
		inviterReward := &AffiliateSignupReward{
			ReferralId:        referral.Id,
			BeneficiaryUserId: inviterUserId,
			BeneficiaryRole:   AffiliateSignupRewardRoleInviter,
			RewardQuota:       config.InviterRewardQuota,
			Status:            AffiliateSignupRewardStatusGranted,
			ConfigVersion:     config.ConfigVersion,
			CreatedAt:         common.GetTimestamp(),
		}
		if err := tx.Create(inviterReward).Error; err != nil {
			return nil, err
		}
		if err := enqueueAffiliateSignupRewardGrantedWithTx(tx, inviterReward); err != nil {
			return nil, err
		}
		inviterUpdates["aff_quota"] = gorm.Expr("aff_quota + ?", config.InviterRewardQuota)
		inviterUpdates["aff_history"] = gorm.Expr("aff_history + ?", config.InviterRewardQuota)
	}
	if result := tx.Model(&User{}).
		Where("id = ? AND status = ?", inviterUserId, common.UserStatusEnabled).
		Updates(inviterUpdates); result.Error != nil {
		return nil, result.Error
	} else if result.RowsAffected != 1 {
		return nil, ErrAffiliateInviterDisabled
	}

	if config.Enabled && config.InviteeRewardQuota > 0 {
		inviteeReward := &AffiliateSignupReward{
			ReferralId:        referral.Id,
			BeneficiaryUserId: referredUser.Id,
			BeneficiaryRole:   AffiliateSignupRewardRoleInvitee,
			RewardQuota:       config.InviteeRewardQuota,
			Status:            AffiliateSignupRewardStatusGranted,
			ConfigVersion:     config.ConfigVersion,
			CreatedAt:         common.GetTimestamp(),
		}
		if err := tx.Create(inviteeReward).Error; err != nil {
			return nil, err
		}
		if err := enqueueAffiliateSignupRewardGrantedWithTx(tx, inviteeReward); err != nil {
			return nil, err
		}
		result := tx.Model(&User{}).
			Where("id = ?", referredUser.Id).
			Update("quota", gorm.Expr("quota + ?", config.InviteeRewardQuota))
		if result.Error != nil {
			return nil, result.Error
		}
		if result.RowsAffected != 1 {
			return nil, ErrAffiliateReferredUserNotFound
		}
		referredUser.Quota += int(config.InviteeRewardQuota)
	}
	referredUser.InviterId = inviterUserId
	return referral, nil
}
