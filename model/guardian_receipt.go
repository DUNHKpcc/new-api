package model

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

type GuardianReceipt struct {
	IPBillingPeriod    string `json:"ip_billing_period" gorm:"type:varchar(16)"`
	IPOptionName       string `json:"ip_option_name" gorm:"type:varchar(128)"`
	IPOptionPriceCents int64  `json:"ip_option_price_cents"`
	OrderID            int    `json:"order_id" gorm:"primaryKey;autoIncrement:false"`
	Number             string `json:"number" gorm:"type:varchar(96);uniqueIndex"`
	TradeNo            string `json:"trade_no" gorm:"type:varchar(64)"`
	ServiceTitle       string `json:"service_title" gorm:"type:varchar(128)"`
	BillingPeriod      string `json:"billing_period" gorm:"type:varchar(16)"`
	AmountCents        int64  `json:"amount_cents"`
	MaxCents           int64  `json:"max_cents"`
	ProCents           int64  `json:"pro_cents"`
	UltraCents         int64  `json:"ultra_cents"`
	PaymentMethod      string `json:"payment_method" gorm:"type:varchar(32)"`
	PaidAt             int64  `json:"paid_at"`
	CreatedAt          int64  `json:"created_at"`
	Email              string `json:"-" gorm:"type:varchar(254)"`
	EmailStatus        string `json:"email_status" gorm:"type:varchar(24)"`
	Attempts           int    `json:"-"`
	NextAttemptAt      int64  `json:"-"`
	RequestedAt        int64  `json:"-"`
}

type GuardianReceiptView struct {
	GuardianReceipt
	Recipient   string `json:"recipient"`
	RefundState string `json:"refund_state"`
	RefundCents *int64 `json:"refund_cents"`
	RefundedAt  int64  `json:"refunded_at"`
}

func GetGuardianReceipt(orderID, userID int, admin bool) (*GuardianReceiptView, error) {
	var view GuardianReceiptView
	err := DB.Transaction(func(tx *gorm.DB) error {
		var order GuardianServiceOrder
		query := lockForUpdate(tx).Where("id = ?", orderID)
		if !admin {
			query = query.Where("user_id = ?", userID)
		}
		if err := query.First(&order).Error; err != nil {
			return errors.New("订单不存在或无权查看")
		}
		if order.PaidAt == 0 {
			return errors.New("订单尚未确认付款")
		}
		receipt, err := ensureGuardianReceipt(tx, order)
		if err != nil {
			return err
		}
		view.GuardianReceipt = *receipt
		var user User
		if err := tx.Select("email").First(&user, order.UserID).Error; err != nil {
			return err
		}
		view.Recipient = guardianMaskedEmail(user.Email)
		view.RefundedAt = order.RefundedAt
		if order.Status == "refunded" {
			view.RefundState = "refunded"
		}
		var sale GuardianResellerSale
		err = tx.First(&sale, "order_id = ?", order.ID).Error
		if err == nil {
			view.RefundState = sale.RefundState
			if sale.RefundState == "completed" {
				view.RefundCents = &sale.RefundCents
			}
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return nil
	})
	return &view, err
}

func ensureGuardianReceipt(tx *gorm.DB, order GuardianServiceOrder) (*GuardianReceipt, error) {
	var receipt GuardianReceipt
	err := tx.First(&receipt, "order_id = ?", order.ID).Error
	if err == nil {
		return &receipt, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	var user User
	if err := tx.Select("email").First(&user, order.UserID).Error; err != nil {
		return nil, err
	}
	now := time.Now().Unix()
	receipt = GuardianReceipt{
		IPBillingPeriod:    order.IPBillingPeriod,
		IPOptionName:       order.IPOptionName,
		IPOptionPriceCents: order.IPOptionPriceCents,
		OrderID:            order.ID,
		Number:             fmt.Sprintf("GRC-%s", order.TradeNo),
		TradeNo:            order.TradeNo,
		ServiceTitle:       order.ServiceTitle,
		BillingPeriod:      order.BillingPeriod,
		AmountCents:        order.AmountCents,
		MaxCents:           order.ClaudeMaxPriceCents,
		UltraCents:         order.ClaudeUltraPriceCents,
		ProCents:           order.ClaudeProPriceCents,
		PaymentMethod:      order.PaymentMethod,
		PaidAt:             order.PaidAt,
		CreatedAt:          now,
		EmailStatus:        "not_requested",
	}
	if order.ReceiptNotification {
		receipt.Email = user.Email
		receipt.RequestedAt = now
		receipt.EmailStatus = "pending"
		if receipt.Email == "" {
			receipt.EmailStatus = "no_email"
		}
	}
	if err := tx.Create(&receipt).Error; err != nil {
		return nil, err
	}
	return &receipt, nil
}

func RequestGuardianReceiptEmail(orderID, userID int, admin bool) error {
	if _, err := GetGuardianReceipt(orderID, userID, admin); err != nil {
		return err
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var receipt GuardianReceipt
		if err := lockForUpdate(tx).First(&receipt, "order_id = ?", orderID).Error; err != nil {
			return err
		}
		now := time.Now().Unix()
		if receipt.EmailStatus == "pending" || receipt.EmailStatus == "sending" || now-receipt.RequestedAt < 60 {
			return errors.New("邮件正在处理或发送过于频繁，请稍后再试")
		}
		var order GuardianServiceOrder
		if err := tx.First(&order, orderID).Error; err != nil {
			return err
		}
		var user User
		if err := tx.Select("email").First(&user, order.UserID).Error; err != nil {
			return err
		}
		if user.Email == "" {
			return errors.New("请先在账户设置中绑定邮箱")
		}
		return tx.Model(&receipt).Updates(map[string]any{"email": user.Email, "email_status": "pending", "attempts": 0, "next_attempt_at": 0, "requested_at": now}).Error
	})
}

// New orders opt in at creation. No payment callback writes receipt or SMTP data,
// and orders predating this feature are never included in automatic mail.
func PrepareGuardianReceipts() error {
	ids := []int{}
	if err := DB.Model(&GuardianServiceOrder{}).Where("receipt_notification = ? AND paid_at > 0", true).Where("id NOT IN (?)", DB.Model(&GuardianReceipt{}).Select("order_id")).Order("id").Limit(100).Pluck("id", &ids).Error; err != nil {
		return err
	}
	for _, id := range ids {
		if _, err := GetGuardianReceipt(id, 0, true); err != nil {
			return err
		}
	}
	return nil
}

func ClaimGuardianReceipt(now int64) (*GuardianReceipt, error) {
	var receipt GuardianReceipt
	err := DB.Transaction(func(tx *gorm.DB) error {
		// An interrupted send has an unknown SMTP result; leave it for explicit resend.
		if err := tx.Model(&GuardianReceipt{}).Where("email_status = ? AND next_attempt_at < ?", "sending", now).Update("email_status", "uncertain").Error; err != nil {
			return err
		}
		err := lockForUpdate(tx).Where("email_status = ? AND next_attempt_at <= ?", "pending", now).Order("order_id").First(&receipt).Error
		if err != nil {
			return err
		}
		result := tx.Model(&GuardianReceipt{}).Where("order_id = ? AND email_status = ?", receipt.OrderID, "pending").Updates(map[string]any{"email_status": "sending", "attempts": receipt.Attempts + 1, "next_attempt_at": now + 300})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return gorm.ErrRecordNotFound
		}
		receipt.Attempts++
		return nil
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &receipt, err
}

func FinishGuardianReceiptEmail(receipt GuardianReceipt, status string) error {
	next := int64(0)
	if status == "pending" {
		delays := []int64{60, 300, 1800}
		if receipt.Attempts >= 4 {
			status = "failed"
		} else {
			next = time.Now().Unix() + delays[receipt.Attempts-1]
		}
	}
	return DB.Model(&GuardianReceipt{}).Where("order_id = ? AND email_status = ? AND attempts = ?", receipt.OrderID, "sending", receipt.Attempts).Updates(map[string]any{"email_status": status, "next_attempt_at": next}).Error
}

func guardianMaskedEmail(email string) string {
	local, domain, ok := strings.Cut(email, "@")
	if !ok || local == "" {
		return ""
	}
	return string([]rune(local)[:1]) + "***@" + domain
}

func HasGuardianReceiptWork() bool {
	var count int64
	now := time.Now().Unix()
	if err := DB.Model(&GuardianReceipt{}).Where("email_status IN ? AND next_attempt_at <= ?", []string{"pending", "sending"}, now).Limit(1).Count(&count).Error; err != nil {
		return false
	}
	if count > 0 {
		return true
	}
	if err := DB.Model(&GuardianServiceOrder{}).Where("receipt_notification = ? AND paid_at > 0", true).Where("id NOT IN (?)", DB.Model(&GuardianReceipt{}).Select("order_id")).Limit(1).Count(&count).Error; err != nil {
		return false
	}
	return count > 0
}
