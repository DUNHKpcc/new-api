package model

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type GuardianFeatures []string

func (f GuardianFeatures) Value() (driver.Value, error) {
	b, err := common.Marshal(f)
	return string(b), err
}
func (f *GuardianFeatures) Scan(value any) error {
	switch v := value.(type) {
	case string:
		return common.UnmarshalJsonStr(v, f)
	case []byte:
		return common.Unmarshal(v, f)
	case nil:
		*f = GuardianFeatures{}
		return nil
	default:
		return errors.New("invalid guardian features")
	}
}

type GuardianService struct {
	ID           int              `json:"id"`
	Title        string           `json:"title" gorm:"type:varchar(128);not null"`
	Subtitle     string           `json:"subtitle" gorm:"type:varchar(255)"`
	Description  string           `json:"description" gorm:"type:text"`
	Features     GuardianFeatures `json:"features" gorm:"type:text"`
	PriceCents   int64            `json:"price_cents" gorm:"not null"`
	DeliveryNote string           `json:"delivery_note" gorm:"type:varchar(128)"`
	Badge        string           `json:"badge" gorm:"type:varchar(32)"`
	Route        string           `json:"route" gorm:"type:varchar(16)"`
	Enabled      bool             `json:"enabled"`
	SortOrder    int              `json:"sort_order"`
	CreatedAt    int64            `json:"created_at"`
	UpdatedAt    int64            `json:"updated_at"`
}

func (s *GuardianService) Validate() error {
	for _, field := range []struct {
		value *string
		max   int
	}{
		{&s.Title, 128}, {&s.Subtitle, 255}, {&s.Description, 0}, {&s.DeliveryNote, 128}, {&s.Badge, 32}, {&s.Route, 16},
	} {
		*field.value = strings.TrimSpace(*field.value)
		if field.max > 0 && utf8.RuneCountInString(*field.value) > field.max {
			return errors.New("服务信息超过长度限制")
		}
	}
	if s.Title == "" || s.PriceCents < 1 || s.PriceCents > 10000000 || !slices.Contains([]string{"any", "local", "vm", "vps"}, s.Route) {
		return errors.New("服务名称、价格或适用路线无效")
	}
	features := GuardianFeatures{}
	for _, f := range s.Features {
		f = strings.TrimSpace(f)
		if utf8.RuneCountInString(f) > 120 {
			return errors.New("服务规格最多 120 字")
		}
		if f != "" {
			features = append(features, f)
		}
	}
	if len(features) > 12 {
		return errors.New("服务规格最多 12 条")
	}
	s.Features = features
	return nil
}

type GuardianOrderRequest struct {
	ServiceID     int    `json:"service_id"`
	PaymentMethod string `json:"payment_method"`
	Route         string `json:"route"`
	OS            string `json:"os"`
	ContactType   string `json:"contact_type"`
	Contact       string `json:"contact"`
	Note          string `json:"note"`
}

func (r *GuardianOrderRequest) Validate() error {
	r.Contact = strings.TrimSpace(r.Contact)
	if r.ServiceID <= 0 || !slices.Contains([]string{"local", "vm", "vps", "unsure"}, r.Route) || !slices.Contains([]string{"macos", "windows", "linux", "other"}, r.OS) || !slices.Contains([]string{"telegram", "qq", "wechat", "email"}, r.ContactType) {
		return errors.New("服务或环境参数无效")
	}
	if r.Contact == "" || utf8.RuneCountInString(r.Contact) > 128 || utf8.RuneCountInString(r.Note) > 1000 {
		return errors.New("联系方式需为 1–128 字，备注最多 1000 字")
	}
	return nil
}

type GuardianServiceOrder struct {
	ID              int    `json:"id"`
	TradeNo         string `json:"trade_no" gorm:"type:varchar(64);uniqueIndex"`
	UserID          int    `json:"-" gorm:"index"`
	ServiceID       int    `json:"service_id" gorm:"index"`
	ServiceTitle    string `json:"service_title" gorm:"type:varchar(128)"`
	AmountCents     int64  `json:"amount_cents"`
	PaymentMethod   string `json:"payment_method" gorm:"type:varchar(32)"`
	Status          string `json:"status" gorm:"type:varchar(24);index"`
	Route           string `json:"route" gorm:"type:varchar(16)"`
	OS              string `json:"os" gorm:"type:varchar(32)"`
	ContactType     string `json:"contact_type" gorm:"type:varchar(16)"`
	Contact         string `json:"contact" gorm:"type:varchar(128)"`
	Note            string `json:"note" gorm:"type:text"`
	AdminNote       string `json:"admin_note" gorm:"type:text"`
	ProviderPayload string `json:"-" gorm:"type:text"`
	CreatedAt       int64  `json:"created_at"`
	PaidAt          int64  `json:"paid_at"`
	DeliveredAt     int64  `json:"delivered_at"`
	UpdatedAt       int64  `json:"updated_at"`
}

type AdminGuardianOrder struct {
	GuardianServiceOrder
	AdminUserID int    `json:"user_id" gorm:"column:user_id"`
	Username    string `json:"username"`
}

func CreateGuardianOrder(userID int, req GuardianOrderRequest) (*GuardianServiceOrder, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	order := &GuardianServiceOrder{}
	err := DB.Transaction(func(tx *gorm.DB) error {
		// Serialize the pending-order limit for this user across application instances.
		var user User
		if err := lockForUpdate(tx).Select("id").First(&user, userID).Error; err != nil {
			return err
		}
		if err := expireGuardianOrders(tx, userID); err != nil {
			return err
		}
		var count int64
		if err := tx.Model(&GuardianServiceOrder{}).Where("user_id = ? AND status = ?", userID, "pending_payment").Count(&count).Error; err != nil {
			return err
		}
		if count >= 3 {
			return errors.New("待付款订单过多，请先完成或等待过期")
		}
		var service GuardianService
		if err := tx.First(&service, req.ServiceID).Error; err != nil {
			return errors.New("服务不存在")
		}
		if !service.Enabled || service.PriceCents < 1 {
			return errors.New("服务已下架或价格无效")
		}
		now := time.Now().Unix()
		*order = GuardianServiceOrder{TradeNo: fmt.Sprintf("SVCUSR%dNO%s%d", userID, common.GetRandomString(6), now), UserID: userID, ServiceID: service.ID, ServiceTitle: service.Title, AmountCents: service.PriceCents, PaymentMethod: req.PaymentMethod, Status: "pending_payment", Route: req.Route, OS: req.OS, ContactType: req.ContactType, Contact: req.Contact, Note: req.Note, CreatedAt: now, UpdatedAt: now}
		return tx.Create(order).Error
	})
	return order, err
}

// Pending orders expire after two hours; userID 0 expires them for every user.
func expireGuardianOrders(db *gorm.DB, userID int) error {
	now := time.Now().Unix()
	query := db.Model(&GuardianServiceOrder{}).Where("status = ? AND created_at <= ?", "pending_payment", now-7200)
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}
	return query.Updates(map[string]any{"status": "expired", "updated_at": now}).Error
}

// Caller holds LockOrder for both asynchronous notification and browser return.
// The cashier stays open after an order expires or is cancelled, so a verified
// late payment still completes the order instead of being acknowledged and lost.
func CompleteGuardianOrder(tradeNo, payload, paidMoney, paymentMethod string) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var order GuardianServiceOrder
		if err := lockForUpdate(tx).Where("trade_no = ?", tradeNo).First(&order).Error; err != nil {
			return err
		}
		if !slices.Contains([]string{"pending_payment", "expired", "cancelled"}, order.Status) {
			return nil
		}
		money, err := decimal.NewFromString(paidMoney)
		if err != nil || !money.Equal(decimal.NewFromInt(order.AmountCents).Div(decimal.NewFromInt(100))) {
			return errors.New("支付金额与订单金额不一致")
		}
		if order.Status != "pending_payment" {
			common.SysLog(fmt.Sprintf("Guardian order %s was paid after it became %s", tradeNo, order.Status))
		}
		now := time.Now().Unix()
		return tx.Model(&order).Updates(map[string]any{"status": "paid", "paid_at": now, "updated_at": now, "provider_payload": payload, "payment_method": paymentMethod}).Error
	})
}

// An unpaid order can reopen its cashier until it expires; the trade number and amount stay fixed.
func GetPayableGuardianOrder(userID, id int) (*GuardianServiceOrder, error) {
	if err := expireGuardianOrders(DB, userID); err != nil {
		return nil, err
	}
	order := &GuardianServiceOrder{}
	if err := DB.Where("id = ? AND user_id = ?", id, userID).First(order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("订单不存在")
		}
		return nil, err
	}
	if order.Status != "pending_payment" {
		return nil, errors.New("订单已过期或无需付款")
	}
	return order, nil
}

// A buyer may withdraw an unpaid order; a payment that still arrives completes it as usual.
func CancelGuardianOrder(userID, id int) (*GuardianServiceOrder, error) {
	order := &GuardianServiceOrder{}
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := lockForUpdate(tx).Where("id = ? AND user_id = ?", id, userID).First(order).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("订单不存在")
			}
			return err
		}
		if order.Status != "pending_payment" {
			return errors.New("只能取消待付款订单")
		}
		order.Status, order.UpdatedAt = "cancelled", time.Now().Unix()
		return tx.Model(order).Select("status", "updated_at").Updates(order).Error
	})
	return order, err
}

func UpdateGuardianOrder(id int, status, note string) (*GuardianServiceOrder, error) {
	if utf8.RuneCountInString(note) > 2000 {
		return nil, errors.New("交付备注最多 2000 字")
	}
	order := &GuardianServiceOrder{}
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := lockForUpdate(tx).First(order, id).Error; err != nil {
			return err
		}
		transitions := map[string][]string{"pending_payment": {"cancelled"}, "paid": {"scheduled", "in_progress", "refunded"}, "scheduled": {"in_progress", "refunded"}, "in_progress": {"delivered", "refunded"}, "delivered": {"refunded"}}
		if status != order.Status && !slices.Contains(transitions[order.Status], status) {
			return errors.New("不允许的订单状态变更")
		}
		now := time.Now().Unix()
		if status == "delivered" && order.Status != status {
			order.DeliveredAt = now
		}
		order.Status, order.AdminNote, order.UpdatedAt = status, note, now
		return tx.Model(order).Select("status", "admin_note", "updated_at", "delivered_at").Updates(order).Error
	})
	return order, err
}

func ListUserGuardianOrders(userID, page, pageSize int) ([]GuardianServiceOrder, int64, error) {
	if err := expireGuardianOrders(DB, userID); err != nil {
		return nil, 0, err
	}
	query := DB.Model(&GuardianServiceOrder{}).Where("user_id = ?", userID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	orders := []GuardianServiceOrder{}
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&orders).Error; err != nil {
		return nil, 0, err
	}
	return orders, total, nil
}

// Usernames include soft-deleted accounts so their paid orders stay traceable.
func ListAdminGuardianOrders(status, keyword string, page, pageSize int) ([]AdminGuardianOrder, int64, error) {
	if err := expireGuardianOrders(DB, 0); err != nil {
		return nil, 0, err
	}
	query := DB.Model(&GuardianServiceOrder{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if keyword != "" {
		pattern := "%" + keyword + "%"
		query = query.Where("trade_no LIKE ? OR contact LIKE ? OR user_id IN (?)", pattern, pattern, DB.Unscoped().Model(&User{}).Select("id").Where("username LIKE ?", pattern))
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	orders := []GuardianServiceOrder{}
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&orders).Error; err != nil {
		return nil, 0, err
	}
	ids := make([]int, 0, len(orders))
	for _, order := range orders {
		ids = append(ids, order.UserID)
	}
	users := []User{}
	if len(ids) > 0 {
		if err := DB.Unscoped().Select("id", "username").Where("id IN ?", ids).Find(&users).Error; err != nil {
			return nil, 0, err
		}
	}
	names := map[int]string{}
	for _, user := range users {
		names[user.Id] = user.Username
	}
	result := make([]AdminGuardianOrder, 0, len(orders))
	for _, order := range orders {
		result = append(result, AdminGuardianOrder{GuardianServiceOrder: order, AdminUserID: order.UserID, Username: names[order.UserID]})
	}
	return result, total, nil
}
