package model

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const affiliateReferralBackfillMarker = "migration.affiliate_referrals_v1"

// backfillAffiliateReferrals turns legacy users.inviter_id values into the
// immutable V1 relationship ledger. It does not synthesize historical reward
// rows or commissions.
func backfillAffiliateReferrals() error {
	var marker Option
	err := DB.Where(commonKeyCol+" = ?", affiliateReferralBackfillMarker).First(&marker).Error
	if err == nil && marker.Value == "done" {
		return nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("load affiliate referral migration marker: %w", err)
	}

	type legacyReferralUser struct {
		Id        int
		InviterId int
		CreatedAt int64
	}
	lastId := 0
	for {
		users := make([]legacyReferralUser, 0, 500)
		if err := DB.Unscoped().Model(&User{}).
			Select("id", "inviter_id", "created_at").
			Where("id > ? AND inviter_id > 0", lastId).
			Order("id asc").
			Limit(500).
			Scan(&users).Error; err != nil {
			return fmt.Errorf("scan legacy affiliate referrals: %w", err)
		}
		if len(users) == 0 {
			break
		}

		inviterIds := make([]int, 0, len(users))
		seenInviters := make(map[int]struct{}, len(users))
		for _, user := range users {
			lastId = user.Id
			if user.InviterId == user.Id {
				continue
			}
			if _, ok := seenInviters[user.InviterId]; !ok {
				seenInviters[user.InviterId] = struct{}{}
				inviterIds = append(inviterIds, user.InviterId)
			}
		}
		inviters := make([]User, 0, len(inviterIds))
		if len(inviterIds) > 0 {
			if err := DB.Unscoped().Select("id", "aff_code").Where("id IN ?", inviterIds).Find(&inviters).Error; err != nil {
				return fmt.Errorf("load legacy affiliate inviters: %w", err)
			}
		}
		codes := make(map[int]string, len(inviters))
		for _, inviter := range inviters {
			codes[inviter.Id] = inviter.AffCode
		}

		referrals := make([]AffiliateReferral, 0, len(users))
		for _, user := range users {
			code, ok := codes[user.InviterId]
			if !ok || user.InviterId == user.Id {
				continue
			}
			boundAt := user.CreatedAt
			if boundAt <= 0 {
				boundAt = common.GetTimestamp()
			}
			referrals = append(referrals, AffiliateReferral{
				InviterUserId:   user.InviterId,
				ReferredUserId:  user.Id,
				AffCodeSnapshot: code,
				BoundAt:         boundAt,
			})
		}
		if len(referrals) > 0 {
			if err := DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&referrals).Error; err != nil {
				return fmt.Errorf("backfill affiliate referrals: %w", err)
			}
		}
	}

	type inviterCount struct {
		InviterUserId int
		Count         int64
	}
	counts := make([]inviterCount, 0)
	if err := DB.Model(&AffiliateReferral{}).
		Select("inviter_user_id, COUNT(*) AS count").
		Group("inviter_user_id").
		Scan(&counts).Error; err != nil {
		return fmt.Errorf("count affiliate referrals: %w", err)
	}
	for _, row := range counts {
		if row.Count < 0 || row.Count > int64(common.MaxQuota) {
			return fmt.Errorf("affiliate referral count overflows user counter: user_id=%d count=%d", row.InviterUserId, row.Count)
		}
		if err := DB.Unscoped().Model(&User{}).
			Where("id = ? AND aff_count < ?", row.InviterUserId, row.Count).
			UpdateColumn("aff_count", int(row.Count)).Error; err != nil {
			return fmt.Errorf("reconcile affiliate referral count: user_id=%d: %w", row.InviterUserId, err)
		}
	}

	marker = Option{Key: affiliateReferralBackfillMarker, Value: "done"}
	if err := DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value"}),
	}).Create(&marker).Error; err != nil {
		return fmt.Errorf("persist affiliate referral migration marker: %w", err)
	}
	return nil
}
