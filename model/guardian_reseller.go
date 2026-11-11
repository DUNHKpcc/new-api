package model

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

type GuardianReseller struct {
	Tier      int    `json:"tier"`
	UserID    int    `json:"user_id" gorm:"primaryKey;autoIncrement:false"`
	Status    string `json:"status" gorm:"type:varchar(16);not null"`
	Code      string `json:"code" gorm:"type:varchar(32);not null;uniqueIndex"`
	CreatedBy int    `json:"created_by"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

type GuardianResellerEvent struct {
	ID        int    `json:"id"`
	UserID    int    `json:"user_id" gorm:"index"`
	ActorID   int    `json:"actor_id"`
	Before    string `json:"before" gorm:"type:varchar(16)"`
	After     string `json:"after" gorm:"type:varchar(16)"`
	Reason    string `json:"reason" gorm:"type:varchar(500)"`
	CreatedAt int64  `json:"created_at"`
}

type GuardianResellerOperation struct {
	ID         int    `json:"id"`
	ActorID    int    `json:"actor_id"`
	ResellerID int    `json:"reseller_id" gorm:"index"`
	Action     string `json:"action" gorm:"type:varchar(32)"`
	Detail     string `json:"detail" gorm:"type:text"`
	CreatedAt  int64  `json:"created_at"`
}

func recordGuardianResellerOperation(tx *gorm.DB, actorID, resellerID int, action string, detail any) error {
	payload, err := common.Marshal(detail)
	if err != nil {
		return err
	}
	return tx.Create(&GuardianResellerOperation{ActorID: actorID, ResellerID: resellerID, Action: action, Detail: string(payload), CreatedAt: time.Now().Unix()}).Error
}

type GuardianResellerAccessRequest struct {
	Tier   int    `json:"tier"`
	Status string `json:"status"`
	Reason string `json:"reason"`
}

func (r *GuardianResellerAccessRequest) Validate() error {
	r.Reason = strings.TrimSpace(r.Reason)
	if r.Status != "active" && r.Status != "suspended" {
		return errors.New("代理状态无效")
	}
	if r.Reason == "" || utf8.RuneCountInString(r.Reason) > 500 {
		return errors.New("请填写 1 至 500 字的代理资格变更原因")
	}
	return nil
}

func GetGuardianReseller(userID int) (*GuardianReseller, error) {
	var reseller GuardianReseller
	err := DB.First(&reseller, "user_id = ?", userID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &reseller, err
}

func SetGuardianResellerAccess(userID, actorID int, request GuardianResellerAccessRequest) (*GuardianReseller, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	var reseller GuardianReseller
	err := DB.Transaction(func(tx *gorm.DB) error {
		// Lock the existing user so first-time grants serialize as well as updates.
		var user User
		if err := lockForUpdate(tx).Select("id").First(&user, userID).Error; err != nil {
			return err
		}
		err := tx.First(&reseller, "user_id = ?", userID).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		before := reseller.Status
		if before == request.Status {
			return nil
		}
		now := time.Now().Unix()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if request.Status != "active" {
				return errors.New("该用户尚未开通 Guardian 代理")
			}
			if request.Tier < 1 || request.Tier > 3 {
				return errors.New("请选择代理等级")
			}
			var count int64
			if err := tx.Model(&GuardianTierSupply{}).Where("tier = ?", request.Tier).Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				return errors.New("请先配置该等级供货价")
			}
			reseller = GuardianReseller{Tier: request.Tier, UserID: userID, Status: request.Status, Code: common.GetRandomString(24), CreatedBy: actorID, CreatedAt: now, UpdatedAt: now}
			if err := tx.Create(&reseller).Error; err != nil {
				return err
			}
		} else {
			reseller.Status, reseller.UpdatedAt = request.Status, now
			if err := tx.Model(&reseller).Select("status", "updated_at").Updates(&reseller).Error; err != nil {
				return err
			}
		}
		return tx.Create(&GuardianResellerEvent{UserID: userID, ActorID: actorID, Before: before, After: request.Status, Reason: request.Reason, CreatedAt: now}).Error
	})
	return &reseller, err
}

type GuardianResellerSummary struct {
	IndependentProducts int    `json:"independent_products"`
	UnavailableProducts int    `json:"unavailable_products"`
	Tier                int    `json:"tier"`
	UserID              int    `json:"user_id"`
	Username            string `json:"username"`
	DisplayName         string `json:"display_name"`
	Status              string `json:"status"`
}

func ListGuardianResellers() ([]GuardianResellerSummary, error) {
	items := []GuardianResellerSummary{}
	err := DB.Table("guardian_resellers AS reseller").
		Joins("JOIN users ON users.id = reseller.user_id").
		Where("users.deleted_at IS NULL").
		Select("reseller.user_id, reseller.status, reseller.tier, users.username, users.display_name").
		Order("reseller.user_id DESC").Scan(&items).Error
	if err != nil {
		return nil, err
	}
	setting, err := GetGuardianResellerSetting()
	if err != nil {
		return nil, err
	}
	for i := range items {
		offers, err := effectiveGuardianOffers(DB, items[i].UserID)
		if err != nil {
			return nil, err
		}
		for _, offer := range offers {
			if !offer.FollowTier {
				items[i].IndependentProducts++
			}
			if offer.Listed && (setting == nil || guardianOfferNeedsRetailUpdate(offer, setting.FeeBPS)) {
				items[i].UnavailableProducts++
			}
		}
	}
	return items, nil
}
