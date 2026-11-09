package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GuardianResellerSetting struct {
	ID        int   `json:"-" gorm:"primaryKey;autoIncrement:false"`
	FeeBPS    int64 `json:"fee_bps"`
	UpdatedBy int   `json:"updated_by"`
	UpdatedAt int64 `json:"updated_at"`
}

func GetGuardianResellerSetting() (*GuardianResellerSetting, error) {
	var setting GuardianResellerSetting
	err := DB.First(&setting, 1).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &setting, err
}

func SaveGuardianResellerSetting(actorID int, feeBPS int64) error {
	if feeBPS < 0 || feeBPS > 1000 {
		return errors.New("支付费率须在 0 至 10% 之间")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		setting := GuardianResellerSetting{ID: 1, FeeBPS: feeBPS, UpdatedBy: actorID, UpdatedAt: time.Now().Unix()}
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "id"}}, DoUpdates: clause.AssignmentColumns([]string{"fee_bps", "updated_by", "updated_at"})}).Create(&setting).Error; err != nil {
			return err
		}
		return recordGuardianResellerOperation(tx, actorID, 0, "fee_setting", setting)
	})
}

type GuardianResellerOffer struct {
	UltraSupplyCents   int64 `json:"ultra_supply_cents"`
	UltraRetailCents   int64 `json:"ultra_retail_cents"`
	ID                 int   `json:"id"`
	ResellerID         int   `json:"reseller_id" gorm:"uniqueIndex:ux_guardian_offer,priority:1"`
	ServiceID          int   `json:"service_id" gorm:"uniqueIndex:ux_guardian_offer,priority:2"`
	Granted            bool  `json:"granted"`
	Listed             bool  `json:"listed"`
	SupplyCents        int64 `json:"supply_cents"`
	QuarterSupplyCents int64 `json:"quarter_supply_cents"`
	AnnualSupplyCents  int64 `json:"annual_supply_cents"`
	MaxSupplyCents     int64 `json:"max_supply_cents"`
	RetailCents        int64 `json:"retail_cents"`
	QuarterRetailCents int64 `json:"quarter_retail_cents"`
	AnnualRetailCents  int64 `json:"annual_retail_cents"`
	MaxRetailCents     int64 `json:"max_retail_cents"`
	UpdatedAt          int64 `json:"updated_at"`
}

type GuardianSupplyRequest struct {
	UltraSupplyCents   int64 `json:"ultra_supply_cents"`
	Granted            bool  `json:"granted"`
	SupplyCents        int64 `json:"supply_cents"`
	QuarterSupplyCents int64 `json:"quarter_supply_cents"`
	AnnualSupplyCents  int64 `json:"annual_supply_cents"`
	MaxSupplyCents     int64 `json:"max_supply_cents"`
}

type GuardianRetailRequest struct {
	UltraRetailCents   int64 `json:"ultra_retail_cents"`
	Listed             bool  `json:"listed"`
	RetailCents        int64 `json:"retail_cents"`
	QuarterRetailCents int64 `json:"quarter_retail_cents"`
	AnnualRetailCents  int64 `json:"annual_retail_cents"`
	MaxRetailCents     int64 `json:"max_retail_cents"`
}

func SaveGuardianSupply(resellerID, serviceID, actorID int, req GuardianSupplyRequest) error {
	for _, amount := range []int64{req.SupplyCents, req.QuarterSupplyCents, req.AnnualSupplyCents, req.MaxSupplyCents, req.UltraSupplyCents} {
		if amount < 0 || amount > 10000000 {
			return errors.New("供货金额须在 0 至 100000 元之间")
		}
	}
	if req.Granted && req.SupplyCents == 0 {
		return errors.New("请设置单次或月付供货价")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var reseller GuardianReseller
		if err := lockForUpdate(tx).First(&reseller, "user_id = ?", resellerID).Error; err != nil {
			return err
		}
		var product GuardianService
		if err := tx.First(&product, serviceID).Error; err != nil {
			return err
		}
		if product.Route != "mac_mini" && (req.QuarterSupplyCents != 0 || req.AnnualSupplyCents != 0 || req.MaxSupplyCents != 0 || req.UltraSupplyCents != 0) {
			return errors.New("部署服务不支持周期及订阅加购供货价")
		}
		offer := GuardianResellerOffer{UltraSupplyCents: req.UltraSupplyCents, ResellerID: resellerID, ServiceID: serviceID, Granted: req.Granted, SupplyCents: req.SupplyCents, QuarterSupplyCents: req.QuarterSupplyCents, AnnualSupplyCents: req.AnnualSupplyCents, MaxSupplyCents: req.MaxSupplyCents, UpdatedAt: time.Now().Unix()}
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "reseller_id"}, {Name: "service_id"}}, DoUpdates: clause.AssignmentColumns([]string{"granted", "supply_cents", "quarter_supply_cents", "annual_supply_cents", "max_supply_cents", "ultra_supply_cents", "updated_at"})}).Create(&offer).Error; err != nil {
			return err
		}
		return recordGuardianResellerOperation(tx, actorID, resellerID, "supply_updated", map[string]any{"service_id": serviceID, "request": req})
	})
}

func SaveGuardianRetail(resellerID, serviceID int, req GuardianRetailRequest) error {
	for _, amount := range []int64{req.RetailCents, req.QuarterRetailCents, req.AnnualRetailCents, req.MaxRetailCents, req.UltraRetailCents} {
		if amount < 0 || amount > 10000000 {
			return errors.New("销售金额须在 0 至 100000 元之间")
		}
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var reseller GuardianReseller
		if err := lockForUpdate(tx).First(&reseller, "user_id = ? AND status = ?", resellerID, "active").Error; err != nil {
			return errors.New("Guardian 代理未开通或已暂停销售")
		}
		var offer GuardianResellerOffer
		if err := tx.First(&offer, "reseller_id = ? AND service_id = ? AND granted = ?", resellerID, serviceID, true).Error; err != nil {
			return errors.New("未获准销售该商品")
		}
		var setting GuardianResellerSetting
		if err := tx.First(&setting, 1).Error; err != nil {
			return errors.New("管理员尚未配置支付费率")
		}
		if req.Listed && req.RetailCents == 0 {
			return errors.New("请设置单次或月付售价")
		}
		for _, price := range [][2]int64{{req.RetailCents, offer.SupplyCents}, {req.QuarterRetailCents, offer.QuarterSupplyCents}, {req.AnnualRetailCents, offer.AnnualSupplyCents}, {req.MaxRetailCents, offer.MaxSupplyCents}, {req.UltraRetailCents, offer.UltraSupplyCents}} {
			if req.Listed && price[0] > 0 && !guardianResalePriceAvailable(price[0], price[1], setting.FeeBPS) {
				return errors.New("售价须覆盖供货价和预计支付手续费，且该规格必须已授权")
			}
		}
		offer.RetailCents, offer.QuarterRetailCents, offer.AnnualRetailCents, offer.MaxRetailCents = req.RetailCents, req.QuarterRetailCents, req.AnnualRetailCents, req.MaxRetailCents
		offer.UltraRetailCents = req.UltraRetailCents
		offer.Listed, offer.UpdatedAt = req.Listed, time.Now().Unix()
		if err := tx.Model(&offer).Select("retail_cents", "quarter_retail_cents", "annual_retail_cents", "max_retail_cents", "ultra_retail_cents", "listed", "updated_at").Updates(&offer).Error; err != nil {
			return err
		}
		return recordGuardianResellerOperation(tx, resellerID, resellerID, "retail_updated", map[string]any{"service_id": serviceID, "request": req})
	})
}

// Round the estimate up to a fen; actual channel fees must be reconciled before payout.
func guardianResaleFeeEstimate(amount, bps int64) int64 {
	return (amount*bps + 9999) / 10000
}

func guardianResalePriceAvailable(retail, supply, bps int64) bool {
	return supply > 0 && retail > 0 && retail-supply >= guardianResaleFeeEstimate(retail, bps)
}

func ListGuardianResellerOffers(resellerID int) ([]GuardianResellerOffer, error) {
	items := []GuardianResellerOffer{}
	err := DB.Where("reseller_id = ?", resellerID).Order("service_id").Find(&items).Error
	return items, err
}

func GuardianResellerCatalog(code string) ([]GuardianService, error) {
	var reseller GuardianReseller
	if err := DB.First(&reseller, "code = ? AND status = ?", code, "active").Error; err != nil {
		return nil, errors.New("代理销售入口不可用")
	}
	var setting GuardianResellerSetting
	if err := DB.First(&setting, 1).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []GuardianService{}, nil
		}
		return nil, err
	}
	offers, err := ListGuardianResellerOffers(reseller.UserID)
	if err != nil {
		return nil, err
	}
	ids := []int{}
	byService := map[int]GuardianResellerOffer{}
	for _, offer := range offers {
		if offer.Granted && offer.Listed && guardianResalePriceAvailable(offer.RetailCents, offer.SupplyCents, setting.FeeBPS) {
			ids = append(ids, offer.ServiceID)
			byService[offer.ServiceID] = offer
		}
	}
	products := []GuardianService{}
	if len(ids) == 0 {
		return products, nil
	}
	if err := DB.Where("id IN ? AND enabled = ?", ids, true).Order("sort_order, id").Find(&products).Error; err != nil {
		return nil, err
	}
	for i := range products {
		p := &products[i]
		o := byService[p.ID]
		p.PriceCents = o.RetailCents
		for _, option := range []struct {
			target         *int64
			retail, supply int64
		}{{&p.QuarterPriceCents, o.QuarterRetailCents, o.QuarterSupplyCents}, {&p.AnnualPriceCents, o.AnnualRetailCents, o.AnnualSupplyCents}, {&p.ClaudeMaxPriceCents, o.MaxRetailCents, o.MaxSupplyCents}, {&p.ClaudeUltraPriceCents, o.UltraRetailCents, o.UltraSupplyCents}} {
			if *option.target > 0 && guardianResalePriceAvailable(option.retail, option.supply, setting.FeeBPS) {
				*option.target = option.retail
			} else {
				*option.target = 0
			}
		}
	}
	return products, nil
}
