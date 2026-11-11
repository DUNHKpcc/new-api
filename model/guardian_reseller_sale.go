package model

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"
)

type GuardianResellerSale struct {
	UltraSupplyCents   int64  `json:"ultra_supply_cents"`
	RefundTermsVersion string `json:"refund_terms_version" gorm:"type:varchar(32)"`
	OrderID            int    `json:"order_id" gorm:"primaryKey;autoIncrement:false"`
	ResellerID         int    `json:"reseller_id" gorm:"index"`
	SupplyCents        int64  `json:"supply_cents"`
	MaxSupplyCents     int64  `json:"max_supply_cents"`
	FeeBPS             int64  `json:"fee_bps"`
	EstimatedFeeCents  int64  `json:"estimated_fee_cents"`
	ActualFeeCents     int64  `json:"actual_fee_cents"`
	FeeConfirmedAt     int64  `json:"fee_confirmed_at"`
	FeeReference       string `json:"-" gorm:"type:varchar(128)"`
	RefundState        string `json:"refund_state" gorm:"type:varchar(16)"`
	RefundCents        int64  `json:"refund_cents"`
	ReturnedFeeCents   int64  `json:"returned_fee_cents"`
	RefundReference    string `json:"-" gorm:"type:varchar(128)"`
}

type GuardianResellerOrderView struct {
	GuardianResellerSale
	TradeNo       string `json:"trade_no"`
	ServiceTitle  string `json:"service_title"`
	BillingPeriod string `json:"billing_period"`
	AmountCents   int64  `json:"amount_cents"`
	Status        string `json:"status"`
	CreatedAt     int64  `json:"created_at"`
	PaidAt        int64  `json:"paid_at"`
	DeliveredAt   int64  `json:"delivered_at"`
}

// Keep customer contact, notes, credentials and internal delivery notes out of this projection.
func guardianResellerOrders(db *gorm.DB) *gorm.DB {
	return db.Table("guardian_reseller_sales AS sale").Joins("JOIN guardian_service_orders AS orders ON orders.id = sale.order_id").Select("sale.*, orders.trade_no, orders.service_title, orders.billing_period, orders.amount_cents, orders.status, orders.created_at, orders.paid_at, orders.delivered_at")
}

func ListGuardianResellerOrders(resellerID, page, size int) ([]GuardianResellerOrderView, int64, error) {
	query := guardianResellerOrders(DB)
	if resellerID != 0 {
		query = query.Where("sale.reseller_id = ?", resellerID)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	items := []GuardianResellerOrderView{}
	err := query.Order("orders.id DESC").Limit(size).Offset((page - 1) * size).Scan(&items).Error
	return items, total, err
}

func GetGuardianResellerOrder(orderID int) (*GuardianResellerOrderView, error) {
	var order GuardianResellerOrderView
	err := guardianResellerOrders(DB).Where("sale.order_id = ?", orderID).Take(&order).Error
	return &order, err
}

func guardianResaleSnapshot(tx *gorm.DB, userID int, req GuardianOrderRequest, product GuardianService) (*GuardianResellerSale, int64, int64, error) {
	if req.PaymentMethod != "alipay" || !req.AcceptRefundFee {
		return nil, 0, 0, errors.New("代理订单仅支持支付宝，请确认退款扣除实际未退手续费的规则")
	}
	var reseller GuardianReseller
	if err := tx.First(&reseller, "code = ? AND status = ?", req.ResellerCode, "active").Error; err != nil {
		return nil, 0, 0, errors.New("代理销售入口不可用")
	}
	if reseller.UserID == userID {
		return nil, 0, 0, errors.New("代理不能购买自己销售的商品")
	}
	var offer GuardianResellerOffer
	if err := effectiveGuardianOffer(tx, reseller.UserID, product.ID, &offer); err != nil || !offer.Granted || !offer.Listed {
		return nil, 0, 0, errors.New("代理尚未开放该商品")
	}
	var setting GuardianResellerSetting
	if err := tx.First(&setting, 1).Error; err != nil {
		return nil, 0, 0, errors.New("管理员尚未配置支付费率")
	}
	retail, supply := offer.RetailCents, offer.SupplyCents
	switch req.BillingPeriod {
	case "quarterly":
		retail, supply = offer.QuarterRetailCents, offer.QuarterSupplyCents
	case "yearly":
		retail, supply = offer.AnnualRetailCents, offer.AnnualSupplyCents
	}
	if !guardianResalePriceAvailable(retail, supply, setting.FeeBPS) {
		return nil, 0, 0, errors.New("代理报价已失效，请联系平台")
	}
	var addon, addonSupply int64
	if req.AddClaudeMax {
		addon, addonSupply = offer.MaxRetailCents, offer.MaxSupplyCents
		if !guardianResalePriceAvailable(addon, addonSupply, setting.FeeBPS) {
			return nil, 0, 0, errors.New("代理尚未开放 MAX 加购报价")
		}
	}
	if req.AddClaudeUltra {
		addon, addonSupply = offer.UltraRetailCents, offer.UltraSupplyCents
		if !guardianResalePriceAvailable(addon, addonSupply, setting.FeeBPS) {
			return nil, 0, 0, errors.New("代理尚未开放 Ultra 加购报价")
		}
	}
	amount := retail + addon
	if req.ExpectedAmountCents == nil || *req.ExpectedAmountCents != amount {
		return nil, 0, 0, errors.New("报价已变更，请刷新后确认金额")
	}
	sale := &GuardianResellerSale{RefundTermsVersion: "deduct_channel_fee_v1", ResellerID: reseller.UserID, SupplyCents: supply + addonSupply, FeeBPS: setting.FeeBPS, EstimatedFeeCents: guardianResaleFeeEstimate(amount, setting.FeeBPS)}
	if req.AddClaudeUltra {
		sale.UltraSupplyCents = addonSupply
	} else {
		sale.MaxSupplyCents = addonSupply
	}
	return sale, amount, addon, nil
}

type GuardianFeeConfirmation struct {
	FeeCents  int64  `json:"fee_cents"`
	Reference string `json:"reference"`
}

func ConfirmGuardianResellerFee(orderID, actorID int, req GuardianFeeConfirmation) error {
	req.Reference = strings.TrimSpace(req.Reference)
	if req.FeeCents < 0 || req.FeeCents > 20000000 || req.Reference == "" || utf8.RuneCountInString(req.Reference) > 128 {
		return errors.New("请填写有效的实际手续费及账单凭证")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var order GuardianServiceOrder
		if err := lockForUpdate(tx).First(&order, orderID).Error; err != nil {
			return err
		}
		var sale GuardianResellerSale
		if err := lockForUpdate(tx).First(&sale, orderID).Error; err != nil {
			return err
		}
		if order.PaidAt == 0 || req.FeeCents > order.AmountCents || sale.RefundState != "" {
			return errors.New("该订单当前不能核对手续费")
		}
		if sale.FeeConfirmedAt != 0 {
			if sale.ActualFeeCents == req.FeeCents && sale.FeeReference == req.Reference {
				return nil
			}
			return errors.New("手续费已核对，不能覆盖已确认账目")
		}
		sale.ActualFeeCents, sale.FeeConfirmedAt, sale.FeeReference = req.FeeCents, time.Now().Unix(), req.Reference
		if err := tx.Model(&sale).Select("actual_fee_cents", "fee_confirmed_at", "fee_reference").Updates(&sale).Error; err != nil {
			return err
		}
		return recordGuardianResellerOperation(tx, actorID, sale.ResellerID, "fee_confirmed", map[string]any{"order_id": orderID, "fee_cents": req.FeeCents, "reference": req.Reference})
	})
}

type GuardianResellerBalance struct {
	HeldCents      int64 `json:"held_cents"`
	PendingCents   int64 `json:"pending_cents"`
	AvailableCents int64 `json:"available_cents"`
	ReservedCents  int64 `json:"reserved_cents"`
	PaidCents      int64 `json:"paid_cents"`
	DebtCents      int64 `json:"debt_cents"`
}

func GetGuardianResellerBalance(resellerID int) (GuardianResellerBalance, error) {
	return guardianResellerBalance(DB, resellerID, time.Now().Unix())
}

func guardianResellerBalance(tx *gorm.DB, resellerID int, now int64) (GuardianResellerBalance, error) {
	var balance GuardianResellerBalance
	orders := []GuardianResellerOrderView{}
	if err := guardianResellerOrders(tx).Where("sale.reseller_id = ? AND orders.paid_at > 0", resellerID).Scan(&orders).Error; err != nil {
		return balance, err
	}
	var earned int64
	for _, order := range orders {
		if order.RefundState == "completed" || order.Status == "refunded" {
			continue
		}
		fee := order.EstimatedFeeCents
		if order.FeeConfirmedAt > 0 {
			fee = order.ActualFeeCents
		}
		net := order.AmountCents - order.SupplyCents - fee
		if order.FeeConfirmedAt > 0 && order.DeliveredAt > 0 && now >= order.DeliveredAt+7*86400 {
			earned += net
			if order.RefundState == "processing" {
				balance.HeldCents += max(0, net)
			}
		} else {
			if order.RefundState != "processing" {
				balance.PendingCents += net
			}
		}
	}
	withdrawals := []GuardianResellerWithdrawal{}
	if err := tx.Where("reseller_id = ?", resellerID).Find(&withdrawals).Error; err != nil {
		return balance, err
	}
	for _, w := range withdrawals {
		switch w.Status {
		case "pending", "processing":
			balance.ReservedCents += w.AmountCents
		case "paid":
			balance.PaidCents += w.AmountCents
		}
	}
	net := earned - balance.PaidCents - balance.ReservedCents
	balance.AvailableCents, balance.DebtCents = max(0, net-balance.HeldCents), max(0, -net)
	return balance, nil
}

type GuardianResellerRefundRequest struct {
	Action           string `json:"action"`
	ReturnedFeeCents int64  `json:"returned_fee_cents"`
	Reference        string `json:"reference"`
	Reason           string `json:"reason"`
}

func UpdateGuardianResellerRefund(orderID, actorID int, req GuardianResellerRefundRequest) error {
	req.Reference, req.Reason = strings.TrimSpace(req.Reference), strings.TrimSpace(req.Reason)
	if (req.Action != "start" && req.Action != "complete" && req.Action != "cancel") || req.ReturnedFeeCents < 0 || req.ReturnedFeeCents > 20000000 || req.Reason == "" || utf8.RuneCountInString(req.Reason) > 500 || utf8.RuneCountInString(req.Reference) > 128 || (req.Action == "complete" && req.Reference == "") {
		return errors.New("退款操作、原因或凭证无效")
	}
	var owner GuardianResellerSale
	if err := DB.First(&owner, orderID).Error; err != nil {
		return err
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var reseller GuardianReseller
		if err := lockForUpdate(tx).First(&reseller, "user_id = ?", owner.ResellerID).Error; err != nil {
			return err
		}
		var order GuardianServiceOrder
		if err := lockForUpdate(tx).First(&order, orderID).Error; err != nil {
			return err
		}
		var sale GuardianResellerSale
		if err := lockForUpdate(tx).First(&sale, orderID).Error; err != nil {
			return err
		}
		if order.PaidAt == 0 || sale.FeeConfirmedAt == 0 || req.ReturnedFeeCents > sale.ActualFeeCents {
			return errors.New("请先核对已付款订单的实际手续费")
		}
		if sale.RefundState == "completed" {
			if req.Action == "complete" && req.Reference == sale.RefundReference && req.ReturnedFeeCents == sale.ReturnedFeeCents {
				return nil
			}
			return errors.New("该订单退款已完成")
		}
		var open int64
		if err := tx.Model(&GuardianResellerWithdrawal{}).Where("reseller_id = ? AND status IN ?", sale.ResellerID, []string{"pending", "processing"}).Count(&open).Error; err != nil {
			return err
		}
		if open > 0 {
			return errors.New("请先处理该代理未完成的提现申请，再操作退款")
		}
		switch req.Action {
		case "start":
			if sale.RefundState != "" {
				return errors.New("退款已在处理中")
			}
			sale.RefundState = "processing"
		case "cancel":
			if sale.RefundState != "processing" {
				return errors.New("没有处理中的退款")
			}
			sale.RefundState = ""
		case "complete":
			if sale.RefundState != "processing" {
				return errors.New("请先发起退款并核对实际退款结果")
			}
			sale.RefundState, sale.ReturnedFeeCents, sale.RefundReference = "completed", req.ReturnedFeeCents, req.Reference
			sale.RefundCents = order.AmountCents - sale.ActualFeeCents + req.ReturnedFeeCents
			before := order
			order.Status, order.RefundedAt, order.UpdatedAt = "refunded", time.Now().Unix(), time.Now().Unix()
			if err := tx.Model(&order).Select("status", "refunded_at", "updated_at").Updates(&order).Error; err != nil {
				return err
			}
			if err := recordGuardianOrderEvent(tx, before, order, actorID, "admin", "reseller_refund"); err != nil {
				return err
			}
		}
		if err := tx.Model(&sale).Select("refund_state", "refund_cents", "returned_fee_cents", "refund_reference").Updates(&sale).Error; err != nil {
			return err
		}
		return recordGuardianResellerOperation(tx, actorID, sale.ResellerID, "refund_"+req.Action, map[string]any{"order_id": orderID, "request": req, "refund_cents": sale.RefundCents})
	})
}
