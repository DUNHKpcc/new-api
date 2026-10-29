package model

import (
	"errors"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"
)

// Order history stays in the primary transaction, unlike best-effort request audit logs.
type GuardianOrderState struct {
	Status      string `json:"status" gorm:"type:varchar(24)"`
	AdminNote   string `json:"admin_note" gorm:"type:text"`
	ContactType string `json:"contact_type" gorm:"type:varchar(16)"`
	Contact     string `json:"contact" gorm:"type:varchar(128)"`
	Note        string `json:"note" gorm:"type:text"`
}
type GuardianOrderEvent struct {
	ID        int                `json:"id"`
	OrderID   int                `json:"order_id" gorm:"index:idx_guardian_history,priority:1"`
	ActorID   int                `json:"actor_id"`
	ActorType string             `json:"actor_type" gorm:"type:varchar(16)"`
	Action    string             `json:"action" gorm:"type:varchar(32)"`
	Before    GuardianOrderState `json:"before" gorm:"embedded;embeddedPrefix:before_"`
	After     GuardianOrderState `json:"after" gorm:"embedded;embeddedPrefix:after_"`
	CreatedAt int64              `json:"created_at" gorm:"index:idx_guardian_history,priority:2"`
}
type GuardianPaymentAnomaly struct {
	ID            int    `json:"id"`
	OrderID       int    `json:"order_id" gorm:"index"`
	DedupKey      string `json:"-" gorm:"type:varchar(64);uniqueIndex"`
	ExpectedCents int64  `json:"expected_cents"`
	PaidMoney     string `json:"paid_money" gorm:"type:varchar(128)"`
	PaymentMethod string `json:"payment_method" gorm:"type:varchar(32)"`
	Occurrences   int    `json:"occurrences"`
	FirstSeenAt   int64  `json:"first_seen_at"`
	LastSeenAt    int64  `json:"last_seen_at"`
}

func guardianOrderState(order GuardianServiceOrder) GuardianOrderState {
	return GuardianOrderState{Status: order.Status, AdminNote: order.AdminNote, ContactType: order.ContactType, Contact: order.Contact, Note: order.Note}
}
func recordGuardianOrderEvent(tx *gorm.DB, before, after GuardianServiceOrder, actorID int, actorType, action string) error {
	previous, next := guardianOrderState(before), guardianOrderState(after)
	if previous == next {
		return nil
	}
	return tx.Create(&GuardianOrderEvent{OrderID: after.ID, ActorID: actorID, ActorType: actorType, Action: action, Before: previous, After: next, CreatedAt: time.Now().Unix()}).Error
}

type GuardianContactUpdate struct {
	ContactType string `json:"contact_type"`
	Contact     string `json:"contact"`
	Note        string `json:"note"`
}

func (r *GuardianContactUpdate) Validate() error {
	r.Contact = strings.TrimSpace(r.Contact)
	if !slices.Contains([]string{"telegram", "qq", "wechat", "email"}, r.ContactType) {
		return errors.New("联系方式类型无效")
	}
	if r.Contact == "" || utf8.RuneCountInString(r.Contact) > 128 || utf8.RuneCountInString(r.Note) > 1000 {
		return errors.New("联系方式需为 1–128 字，备注最多 1000 字")
	}
	return nil
}
func UpdateGuardianContact(userID, id int, req GuardianContactUpdate) (*GuardianServiceOrder, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if err := expireGuardianOrders(DB, userID); err != nil {
		return nil, err
	}
	order := &GuardianServiceOrder{}
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := lockForUpdate(tx).Where("id = ? AND user_id = ?", id, userID).First(order).Error; err != nil {
			return errors.New("订单不存在")
		}
		if !slices.Contains([]string{"pending_payment", "paid", "scheduled", "in_progress"}, order.Status) {
			return errors.New("订单已关闭，不能修改联系方式或备注")
		}
		before := *order
		order.ContactType, order.Contact, order.Note, order.UpdatedAt = req.ContactType, req.Contact, req.Note, time.Now().Unix()
		if err := tx.Model(order).Select("contact_type", "contact", "note", "updated_at").Updates(order).Error; err != nil {
			return err
		}
		return recordGuardianOrderEvent(tx, before, *order, userID, "user", "contact_updated")
	})
	return order, err
}

func ExpireGuardianPaymentInitiation(id int) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var order GuardianServiceOrder
		if err := lockForUpdate(tx).First(&order, id).Error; err != nil {
			return err
		}
		if order.Status != "pending_payment" {
			return nil
		}
		before := order
		order.Status = "expired"
		order.UpdatedAt = time.Now().Unix()
		if err := tx.Model(&order).Select("status", "updated_at").Updates(&order).Error; err != nil {
			return err
		}
		return recordGuardianOrderEvent(tx, before, order, 0, "system", "payment_failed")
	})
}
