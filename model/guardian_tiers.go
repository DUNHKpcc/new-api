package model

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GuardianTierSupply struct {
	Tier                  int `json:"tier" gorm:"primaryKey;autoIncrement:false"`
	ServiceID             int `json:"service_id" gorm:"primaryKey;autoIncrement:false"`
	GuardianSupplyRequest `gorm:"embedded"`
	UpdatedAt             int64 `json:"updated_at"`
}

type GuardianTierChange struct {
	Tier         int                  `json:"tier"`
	Prices       []GuardianTierSupply `json:"prices"`
	ResellerIDs  []int                `json:"reseller_ids"`
	ServiceIDs   []int                `json:"service_ids"`
	Restore      bool                 `json:"restore"`
	PreviewToken string               `json:"preview_token"`
}

type GuardianOfferChange struct {
	ResellerID int                   `json:"reseller_id"`
	ServiceID  int                   `json:"service_id"`
	Before     GuardianResellerOffer `json:"before"`
	After      GuardianResellerOffer `json:"after"`
}

type GuardianTierPreview struct {
	OfferChanges []GuardianOfferChange `json:"offer_changes"`
	ResellerIDs  []int                 `json:"reseller_ids"`
	Token        string                `json:"token"`
	Resellers    int                   `json:"resellers"`
	Independent  int                   `json:"independent"`
	Unavailable  int                   `json:"unavailable"`
	Before       []GuardianTierSupply  `json:"before"`
	After        []GuardianTierSupply  `json:"after"`
}

func ListGuardianTierSupplies() ([]GuardianTierSupply, error) {
	items := []GuardianTierSupply{}
	err := DB.Order("tier, service_id").Find(&items).Error
	return items, err
}

func effectiveGuardianOffers(tx *gorm.DB, resellerID int) ([]GuardianResellerOffer, error) {
	var reseller GuardianReseller
	if err := tx.First(&reseller, "user_id = ?", resellerID).Error; err != nil {
		return nil, err
	}
	items := []GuardianResellerOffer{}
	if err := tx.Where("reseller_id = ?", resellerID).Order("service_id").Find(&items).Error; err != nil {
		return nil, err
	}
	prices := []GuardianTierSupply{}
	if err := tx.Where("tier = ?", reseller.Tier).Order("service_id").Find(&prices).Error; err != nil {
		return nil, err
	}
	byService := map[int]GuardianTierSupply{}
	for _, p := range prices {
		byService[p.ServiceID] = p
	}
	for i := range items {
		if items[i].FollowTier {
			applyGuardianTierSupply(&items[i], byService[items[i].ServiceID])
		}
		delete(byService, items[i].ServiceID)
	}
	for _, p := range byService {
		offer := GuardianResellerOffer{ResellerID: resellerID, ServiceID: p.ServiceID, FollowTier: true}
		applyGuardianTierSupply(&offer, p)
		items = append(items, offer)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ServiceID < items[j].ServiceID })
	return items, nil
}

func applyGuardianTierSupply(offer *GuardianResellerOffer, price GuardianTierSupply) {
	offer.Granted = price.Granted
	offer.SupplyCents, offer.QuarterSupplyCents, offer.AnnualSupplyCents = price.SupplyCents, price.QuarterSupplyCents, price.AnnualSupplyCents
	offer.MaxSupplyCents, offer.UltraSupplyCents = price.MaxSupplyCents, price.UltraSupplyCents
}

func effectiveGuardianOffer(tx *gorm.DB, resellerID, serviceID int, offer *GuardianResellerOffer) error {
	items, err := effectiveGuardianOffers(tx, resellerID)
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.ServiceID == serviceID {
			*offer = item
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}

// Preview and save use the same transaction; the token detects a changed price
// configuration between the administrator's review and submission.
func ChangeGuardianTier(actorID int, req GuardianTierChange) (*GuardianTierPreview, error) {
	if err := validateGuardianTierChange(req); err != nil {
		return nil, err
	}
	result := &GuardianTierPreview{}
	err := DB.Transaction(func(tx *gorm.DB) error {
		prices := []GuardianTierSupply{}
		if err := lockForUpdate(tx).Order("tier, service_id").Find(&prices).Error; err != nil {
			return err
		}
		resellers := []GuardianReseller{}
		query := lockForUpdate(tx).Order("user_id")
		if len(req.Prices) > 0 {
			query = query.Where("tier = ?", req.Tier)
		} else {
			query = query.Where("user_id IN ?", req.ResellerIDs)
		}
		if err := query.Find(&resellers).Error; err != nil {
			return err
		}
		if len(req.Prices) == 0 && len(resellers) != len(req.ResellerIDs) {
			return errors.New("代理列表已变更，请刷新")
		}
		offers := []GuardianResellerOffer{}
		ids := []int{}
		for _, r := range resellers {
			ids = append(ids, r.UserID)
		}
		if len(ids) > 0 {
			if err := tx.Where("reseller_id IN ?", ids).Order("reseller_id, service_id").Find(&offers).Error; err != nil {
				return err
			}
		}
		payload, err := common.Marshal(struct {
			Prices    []GuardianTierSupply
			Resellers []GuardianReseller
			Offers    []GuardianResellerOffer
		}{prices, resellers, offers})
		if err != nil {
			return err
		}
		// Bind the preview to both the current data and the requested edit.
		edit := req
		edit.PreviewToken = ""
		editBytes, err := common.Marshal(edit)
		if err != nil {
			return err
		}
		result.Token = fmt.Sprintf("%x", sha256.Sum256(append(payload, editBytes...)))
		if req.PreviewToken != "" && req.PreviewToken != result.Token {
			return errors.New("配置已变化，请重新预览")
		}
		previousOffers := map[int]map[int]GuardianResellerOffer{}
		if len(req.Prices) == 0 {
			for _, r := range resellers {
				current, err := effectiveGuardianOffers(tx, r.UserID)
				if err != nil {
					return err
				}
				previousOffers[r.UserID] = map[int]GuardianResellerOffer{}
				for _, o := range current {
					previousOffers[r.UserID][o.ServiceID] = o
				}
			}
		}
		result.ResellerIDs = ids
		result.Resellers = len(resellers)
		for _, o := range offers {
			if !o.FollowTier {
				result.Independent++
			}
		}
		for _, p := range prices {
			if p.Tier == req.Tier {
				result.Before = append(result.Before, p)
			}
		}
		result.After = req.Prices
		if len(req.Prices) > 0 {
			for _, p := range req.Prices {
				var product GuardianService
				if err := tx.First(&product, p.ServiceID).Error; err != nil {
					return err
				}
				if product.Route != "mac_mini" && (p.QuarterSupplyCents != 0 || p.AnnualSupplyCents != 0 || p.MaxSupplyCents != 0 || p.UltraSupplyCents != 0) {
					return errors.New("部署服务不支持周期和加购价")
				}
				p.Tier = req.Tier
				p.UpdatedAt = time.Now().UnixNano()
				if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "tier"}, {Name: "service_id"}}, DoUpdates: clause.AssignmentColumns([]string{"granted", "supply_cents", "quarter_supply_cents", "annual_supply_cents", "max_supply_cents", "ultra_supply_cents", "updated_at"})}).Create(&p).Error; err != nil {
					return err
				}
			}
		} else {
			var configured int64
			if err := tx.Model(&GuardianTierSupply{}).Where("tier = ?", req.Tier).Count(&configured).Error; err != nil {
				return err
			}
			if configured == 0 {
				return errors.New("请先配置该等级供货价")
			}
			if !req.Restore {
				if err := tx.Model(&GuardianReseller{}).Where("user_id IN ?", ids).Updates(map[string]any{"tier": req.Tier, "updated_at": time.Now().Unix()}).Error; err != nil {
					return err
				}
			} else {
				for _, r := range resellers {
					if r.Tier != req.Tier {
						return errors.New("恢复跟随等级时请选择同一等级的代理")
					}
				}
				if err := tx.Model(&GuardianResellerOffer{}).Where("reseller_id IN ? AND service_id IN ?", ids, req.ServiceIDs).Update("follow_tier", true).Error; err != nil {
					return err
				}
			}
		}
		var setting GuardianResellerSetting
		if err := tx.First(&setting, 1).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		for _, r := range resellers {
			effective, err := effectiveGuardianOffers(tx, r.UserID)
			if err != nil {
				return err
			}
			for _, o := range effective {
				if len(req.Prices) == 0 {
					before := previousOffers[r.UserID][o.ServiceID]
					if before.FollowTier != o.FollowTier || before.Granted != o.Granted || before.SupplyCents != o.SupplyCents || before.QuarterSupplyCents != o.QuarterSupplyCents || before.AnnualSupplyCents != o.AnnualSupplyCents || before.MaxSupplyCents != o.MaxSupplyCents || before.UltraSupplyCents != o.UltraSupplyCents {
						result.OfferChanges = append(result.OfferChanges, GuardianOfferChange{ResellerID: r.UserID, ServiceID: o.ServiceID, Before: before, After: o})
					}
				}
				if guardianOfferNeedsRetailUpdate(o, setting.FeeBPS) {
					result.Unavailable++
				}
			}
		}
		if req.PreviewToken == "" {
			return errGuardianTierPreview
		}
		return recordGuardianResellerOperation(tx, actorID, 0, "tier_configuration", map[string]any{"before": prices, "resellers_before": resellers, "offers_before": offers, "change": edit})
	})
	if errors.Is(err, errGuardianTierPreview) {
		err = nil
	}
	return result, err
}

var errGuardianTierPreview = errors.New("guardian tier preview rollback")

func validateGuardianTierChange(req GuardianTierChange) error {
	if req.Tier < 1 || req.Tier > 3 {
		return errors.New("请选择一级、二级或三级代理")
	}
	if len(req.Prices) > 0 && (len(req.ResellerIDs) > 0 || req.Restore) {
		return errors.New("不能同时修改等级价格和代理等级")
	}
	if len(req.Prices) == 0 && len(req.ResellerIDs) == 0 {
		return errors.New("请选择代理或填写供货价")
	}
	if req.Restore && len(req.ServiceIDs) == 0 {
		return errors.New("请选择恢复继承的商品")
	}
	seen := map[int]bool{}
	for _, id := range req.ResellerIDs {
		if id <= 0 || seen[id] {
			return errors.New("代理列表无效")
		}
		seen[id] = true
	}
	seen = map[int]bool{}
	for _, p := range req.Prices {
		if p.ServiceID <= 0 || seen[p.ServiceID] {
			return errors.New("商品列表无效")
		}
		seen[p.ServiceID] = true
		for _, amount := range []int64{p.SupplyCents, p.QuarterSupplyCents, p.AnnualSupplyCents, p.MaxSupplyCents, p.UltraSupplyCents} {
			if amount < 0 || amount > 10000000 {
				return errors.New("供货金额须在 0 至 100000 元之间")
			}
		}
		if p.Granted && p.SupplyCents == 0 {
			return errors.New("请设置单次或月付供货价")
		}
	}
	return nil
}

func guardianOfferNeedsRetailUpdate(offer GuardianResellerOffer, feeBPS int64) bool {
	if !offer.Listed {
		return false
	}
	if !offer.Granted || !guardianResalePriceAvailable(offer.RetailCents, offer.SupplyCents, feeBPS) {
		return true
	}
	for _, price := range [][2]int64{{offer.QuarterRetailCents, offer.QuarterSupplyCents}, {offer.AnnualRetailCents, offer.AnnualSupplyCents}, {offer.MaxRetailCents, offer.MaxSupplyCents}, {offer.UltraRetailCents, offer.UltraSupplyCents}} {
		if price[0] > 0 && !guardianResalePriceAvailable(price[0], price[1], feeBPS) {
			return true
		}
	}
	return false
}
