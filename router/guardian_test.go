package router

import (
	"bytes"
	"fmt"
	"github.com/QuantumNous/new-api/middleware"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Calcium-Ion/go-epay/epay"
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Released rc.48 service schema, retained as an upgrade fixture.
type guardianServiceBeforeChip struct {
	BaseSales    int64
	ID           int
	Title        string                 `gorm:"type:varchar(128);not null"`
	Subtitle     string                 `gorm:"type:varchar(255)"`
	Description  string                 `gorm:"type:text"`
	Features     model.GuardianFeatures `gorm:"type:text"`
	PriceCents   int64                  `gorm:"not null"`
	DeliveryNote string                 `gorm:"type:varchar(128)"`
	Badge        string                 `gorm:"type:varchar(32)"`
	Route        string                 `gorm:"type:varchar(16)"`
	Enabled      bool
	SortOrder    int
	CreatedAt    int64
	UpdatedAt    int64
}

func (guardianServiceBeforeChip) TableName() string { return "guardian_services" }

type guardianOrderBeforePeriod struct {
	RefundedAt        int64  `json:"refunded_at"`
	HasPaymentAnomaly bool   `json:"-" gorm:"index"`
	ID                int    `json:"id"`
	TradeNo           string `json:"trade_no" gorm:"type:varchar(64);uniqueIndex"`
	UserID            int    `json:"-" gorm:"index"`
	ServiceID         int    `json:"service_id" gorm:"index"`
	ServiceTitle      string `json:"service_title" gorm:"type:varchar(128)"`
	AmountCents       int64  `json:"amount_cents"`
	PaymentMethod     string `json:"payment_method" gorm:"type:varchar(32)"`
	Status            string `json:"status" gorm:"type:varchar(24);index"`
	Route             string `json:"route" gorm:"type:varchar(16)"`
	OS                string `json:"os" gorm:"type:varchar(32)"`
	ContactType       string `json:"contact_type" gorm:"type:varchar(16)"`
	Contact           string `json:"contact" gorm:"type:varchar(128)"`
	Note              string `json:"note" gorm:"type:text"`
	AdminNote         string `json:"admin_note" gorm:"type:text"`
	ProviderPayload   string `json:"-" gorm:"type:text"`
	CreatedAt         int64  `json:"created_at"`
	PaidAt            int64  `json:"paid_at"`
	QueueSeq          int64  `json:"-" gorm:"index"`
	DeliveredAt       int64  `json:"delivered_at"`
	UpdatedAt         int64  `json:"updated_at"`
}

func (guardianOrderBeforePeriod) TableName() string { return "guardian_service_orders" }

type guardianAccountStatusBeforeLinks struct {
	UserID        int `gorm:"primaryKey;autoIncrement:false"`
	Banned        bool
	PhoneRequired bool
	KYCRequired   bool
	UpdatedAt     int64
	UpdatedBy     int
}

func (guardianAccountStatusBeforeLinks) TableName() string { return "guardian_account_statuses" }

// Released rc.54 account status schema, retained only for upgrade verification.
type guardianRC54AccountStatus struct {
	model.GuardianAccountStatus
	HistoricalCategory string `gorm:"type:varchar(32)"`
}

func (guardianRC54AccountStatus) TableName() string { return "guardian_account_statuses" }

// rc.51 order schema immediately before reseller support.
type guardianOrderBeforeReseller struct {
	Legacy              guardianOrderBeforePeriod `gorm:"embedded"`
	BillingPeriod       string                    `gorm:"type:varchar(16)"`
	ClaudeMaxPriceCents int64
}

func (guardianOrderBeforeReseller) TableName() string { return "guardian_service_orders" }

type guardianRC52ServiceOrder struct {
	ClaudeUltraPriceCents     int64  `json:"claude_ultra_price_cents"`
	RenewalPreviousResellerID int    `json:"-"`
	CustomerServiceID         int    `json:"customer_service_id" gorm:"index"`
	ResellerCode              string `json:"reseller_code,omitempty" gorm:"type:varchar(32)"`
	ClaudeMaxPriceCents       int64  `json:"claude_max_price_cents"`
	BillingPeriod             string `json:"billing_period" gorm:"type:varchar(16)"`
	RefundedAt                int64  `json:"refunded_at"`
	HasPaymentAnomaly         bool   `json:"-" gorm:"index"`
	ID                        int    `json:"id"`
	TradeNo                   string `json:"trade_no" gorm:"type:varchar(64);uniqueIndex"`
	UserID                    int    `json:"-" gorm:"index"`
	ServiceID                 int    `json:"service_id" gorm:"index"`
	ServiceTitle              string `json:"service_title" gorm:"type:varchar(128)"`
	AmountCents               int64  `json:"amount_cents"`
	PaymentMethod             string `json:"payment_method" gorm:"type:varchar(32)"`
	Status                    string `json:"status" gorm:"type:varchar(24);index"`
	Route                     string `json:"route" gorm:"type:varchar(16)"`
	OS                        string `json:"os" gorm:"type:varchar(32)"`
	ContactType               string `json:"contact_type" gorm:"type:varchar(16)"`
	Contact                   string `json:"contact" gorm:"type:varchar(128)"`
	Note                      string `json:"note" gorm:"type:text"`
	AdminNote                 string `json:"admin_note" gorm:"type:text"`
	ProviderPayload           string `json:"-" gorm:"type:text"`
	CreatedAt                 int64  `json:"created_at"`
	PaidAt                    int64  `json:"paid_at"`
	QueueSeq                  int64  `json:"-" gorm:"index"`
	DeliveredAt               int64  `json:"delivered_at"`
	UpdatedAt                 int64  `json:"updated_at"`
}

func (guardianRC52ServiceOrder) TableName() string { return "guardian_service_orders" }

type guardianRC52Reseller struct {
	UserID    int    `json:"user_id" gorm:"primaryKey;autoIncrement:false"`
	Status    string `json:"status" gorm:"type:varchar(16);not null"`
	Code      string `json:"code" gorm:"type:varchar(32);not null;uniqueIndex"`
	CreatedBy int    `json:"created_by"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

func (guardianRC52Reseller) TableName() string { return "guardian_resellers" }

type guardianRC52ResellerOffer struct {
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

func (guardianRC52ResellerOffer) TableName() string { return "guardian_reseller_offers" }

// rc.53 schemas preserve the released data shape for upgrade verification.
type guardianRC53Service struct {
	InvoiceEnabled        bool                   `json:"invoice_enabled"`
	ClaudeUltraPriceCents int64                  `json:"claude_ultra_price_cents"`
	ClaudeMaxPriceCents   int64                  `json:"claude_max_price_cents"`
	QuarterPriceCents     int64                  `json:"quarter_price_cents"`
	AnnualPriceCents      int64                  `json:"annual_price_cents"`
	Region                string                 `json:"region" gorm:"type:varchar(16)"`
	HardwareConfig        string                 `json:"hardware_config" gorm:"type:varchar(16)"`
	Chip                  string                 `json:"chip" gorm:"type:varchar(8)"`
	BaseSales             int64                  `json:"base_sales"`
	ActualSales           *int64                 `json:"actual_sales,omitempty" gorm:"-"`
	SoldCount             *int64                 `json:"sold_count,omitempty" gorm:"-"`
	ID                    int                    `json:"id"`
	Title                 string                 `json:"title" gorm:"type:varchar(128);not null"`
	Subtitle              string                 `json:"subtitle" gorm:"type:varchar(255)"`
	Description           string                 `json:"description" gorm:"type:text"`
	Features              model.GuardianFeatures `json:"features" gorm:"type:text"`
	PriceCents            int64                  `json:"price_cents" gorm:"not null"`
	DeliveryNote          string                 `json:"delivery_note" gorm:"type:varchar(128)"`
	Badge                 string                 `json:"badge" gorm:"type:varchar(32)"`
	Route                 string                 `json:"route" gorm:"type:varchar(16)"`
	Enabled               bool                   `json:"enabled"`
	SortOrder             int                    `json:"sort_order"`
	CreatedAt             int64                  `json:"created_at"`
	UpdatedAt             int64                  `json:"updated_at"`
}

func (guardianRC53Service) TableName() string { return "guardian_services" }

type guardianRC53Order struct {
	InvoiceInformation        string `json:"invoice_information" gorm:"type:text"`
	DiscoveryChannel          string `json:"discovery_channel" gorm:"type:varchar(16)"`
	ReceiptNotification       bool   `json:"-"`
	ClaudeUltraPriceCents     int64  `json:"claude_ultra_price_cents"`
	RenewalPreviousResellerID int    `json:"-"`
	CustomerServiceID         int    `json:"customer_service_id" gorm:"index"`
	ResellerCode              string `json:"reseller_code,omitempty" gorm:"type:varchar(32)"`
	ClaudeMaxPriceCents       int64  `json:"claude_max_price_cents"`
	BillingPeriod             string `json:"billing_period" gorm:"type:varchar(16)"`
	RefundedAt                int64  `json:"refunded_at"`
	HasPaymentAnomaly         bool   `json:"-" gorm:"index"`
	ID                        int    `json:"id"`
	TradeNo                   string `json:"trade_no" gorm:"type:varchar(64);uniqueIndex"`
	UserID                    int    `json:"-" gorm:"index"`
	ServiceID                 int    `json:"service_id" gorm:"index"`
	ServiceTitle              string `json:"service_title" gorm:"type:varchar(128)"`
	AmountCents               int64  `json:"amount_cents"`
	PaymentMethod             string `json:"payment_method" gorm:"type:varchar(32)"`
	Status                    string `json:"status" gorm:"type:varchar(24);index"`
	Route                     string `json:"route" gorm:"type:varchar(16)"`
	OS                        string `json:"os" gorm:"type:varchar(32)"`
	ContactType               string `json:"contact_type" gorm:"type:varchar(16)"`
	Contact                   string `json:"contact" gorm:"type:varchar(128)"`
	Note                      string `json:"note" gorm:"type:text"`
	AdminNote                 string `json:"admin_note" gorm:"type:text"`
	ProviderPayload           string `json:"-" gorm:"type:text"`
	CreatedAt                 int64  `json:"created_at"`
	PaidAt                    int64  `json:"paid_at"`
	QueueSeq                  int64  `json:"-" gorm:"index"`
	DeliveredAt               int64  `json:"delivered_at"`
	UpdatedAt                 int64  `json:"updated_at"`
}

func (guardianRC53Order) TableName() string { return "guardian_service_orders" }

type guardianRC53Receipt struct {
	OrderID       int    `json:"order_id" gorm:"primaryKey;autoIncrement:false"`
	Number        string `json:"number" gorm:"type:varchar(96);uniqueIndex"`
	TradeNo       string `json:"trade_no" gorm:"type:varchar(64)"`
	ServiceTitle  string `json:"service_title" gorm:"type:varchar(128)"`
	BillingPeriod string `json:"billing_period" gorm:"type:varchar(16)"`
	AmountCents   int64  `json:"amount_cents"`
	MaxCents      int64  `json:"max_cents"`
	UltraCents    int64  `json:"ultra_cents"`
	PaymentMethod string `json:"payment_method" gorm:"type:varchar(32)"`
	PaidAt        int64  `json:"paid_at"`
	CreatedAt     int64  `json:"created_at"`
	Email         string `json:"-" gorm:"type:varchar(254)"`
	EmailStatus   string `json:"email_status" gorm:"type:varchar(24)"`
	Attempts      int    `json:"-"`
	NextAttemptAt int64  `json:"-"`
	RequestedAt   int64  `json:"-"`
}

func (guardianRC53Receipt) TableName() string { return "guardian_receipts" }

type guardianRC53Supply struct {
	UltraSupplyCents   int64 `json:"ultra_supply_cents"`
	Granted            bool  `json:"granted"`
	SupplyCents        int64 `json:"supply_cents"`
	QuarterSupplyCents int64 `json:"quarter_supply_cents"`
	AnnualSupplyCents  int64 `json:"annual_supply_cents"`
	MaxSupplyCents     int64 `json:"max_supply_cents"`
}
type guardianRC53Offer struct {
	FollowTier         bool  `json:"follow_tier"`
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

func (guardianRC53Offer) TableName() string { return "guardian_reseller_offers" }

type guardianRC53Sale struct {
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

func (guardianRC53Sale) TableName() string { return "guardian_reseller_sales" }

type guardianRC53Tier struct {
	Tier      int                `json:"tier" gorm:"primaryKey;autoIncrement:false"`
	ServiceID int                `json:"service_id" gorm:"primaryKey;autoIncrement:false"`
	Supply    guardianRC53Supply `gorm:"embedded"`
	UpdatedAt int64              `json:"updated_at"`
}

func (guardianRC53Tier) TableName() string { return "guardian_tier_supplies" }

type guardianRC54Service struct {
	Legacy                guardianRC53Service    `gorm:"embedded"`
	Tags                  model.GuardianFeatures `gorm:"type:text"`
	OutOfStock            bool
	StockNote             string `gorm:"type:varchar(255)"`
	ClaudeProOutOfStock   bool
	ClaudeMaxOutOfStock   bool
	ClaudeUltraOutOfStock bool
	ClaudeProPriceCents   int64
}

func (guardianRC54Service) TableName() string { return "guardian_services" }

type guardianRC54Receipt struct {
	Legacy   guardianRC53Receipt `gorm:"embedded"`
	ProCents int64
}

func (guardianRC54Receipt) TableName() string { return "guardian_receipts" }

type guardianRC54Order struct {
	Legacy              guardianRC53Order `gorm:"embedded"`
	ClaudeProPriceCents int64
}

func (guardianRC54Order) TableName() string { return "guardian_service_orders" }

func TestGuardianLifecycle(t *testing.T) {
	for _, dialect := range []string{"sqlite", "mysql", "postgres"} {
		t.Run(dialect, func(t *testing.T) {
			var driver gorm.Dialector
			dbType := common.DatabaseTypeSQLite
			switch dialect {
			case "sqlite":
				dsn := os.Getenv("TEST_GUARDIAN_SQLITE")
				if dsn == "" {
					dsn = filepath.Join(t.TempDir(), "guardian.db")
				}
				driver = sqlite.Open(dsn)
			case "mysql":
				dsn := os.Getenv("TEST_MYSQL_DSN")
				if dsn == "" {
					t.Skip("TEST_MYSQL_DSN not configured")
				}
				driver = mysql.Open(dsn)
				dbType = common.DatabaseTypeMySQL
			case "postgres":
				dsn := os.Getenv("TEST_POSTGRES_DSN")
				if dsn == "" {
					t.Skip("TEST_POSTGRES_DSN not configured")
				}
				driver = postgres.Open(dsn)
				dbType = common.DatabaseTypePostgreSQL
			}
			db, err := gorm.Open(driver, &gorm.Config{})
			require.NoError(t, err)
			var version string
			if dialect == "sqlite" {
				require.NoError(t, db.Raw("SELECT sqlite_version()").Scan(&version).Error)
			} else {
				require.NoError(t, db.Raw("SELECT version()").Scan(&version).Error)
			}
			t.Logf("Database version: %s", version)
			sqlDB, err := db.DB()
			require.NoError(t, err)
			if dialect == "sqlite" {
				sqlDB.SetMaxOpenConns(1)
			}
			oldDB, oldLog, oldType := model.DB, model.LOG_DB, common.MainDatabaseType()
			oldRedis, oldSecret, oldLimit := common.RedisEnabled, common.SessionSecret, common.CriticalRateLimitEnable
			oldAddress := system_setting.ServerAddress
			oldPayAddress, oldPID, oldKey, oldMethods := operation_setting.PayAddress, operation_setting.EpayId, operation_setting.EpayKey, operation_setting.PayMethods
			oldPayment := *operation_setting.GetPaymentSetting()
			model.DB, model.LOG_DB = db, db
			common.SetMainDatabaseType(dbType)
			common.RedisEnabled = false
			common.SessionSecret = "guardian-test-session-secret"
			common.CriticalRateLimitEnable = false
			system_setting.ServerAddress = "https://guardian.example.test"
			operation_setting.PayAddress = "https://payment.example.test"
			operation_setting.EpayId = "123"
			operation_setting.EpayKey = "guardian-fixture-key"
			operation_setting.PayMethods = []map[string]string{{"type": "alipay", "name": "支付宝"}, {"type": "stripe", "name": "Stripe"}, {"type": "waffo_pancake", "name": "Waffo"}}
			payment := operation_setting.GetPaymentSetting()
			payment.ComplianceConfirmed = true
			payment.ComplianceTermsVersion = operation_setting.CurrentComplianceTermsVersion
			t.Cleanup(func() {
				model.DB, model.LOG_DB = oldDB, oldLog
				common.SetMainDatabaseType(oldType)
				common.RedisEnabled, common.SessionSecret, common.CriticalRateLimitEnable = oldRedis, oldSecret, oldLimit
				system_setting.ServerAddress = oldAddress
				operation_setting.PayAddress, operation_setting.EpayId, operation_setting.EpayKey, operation_setting.PayMethods = oldPayAddress, oldPID, oldKey, oldMethods
				*payment = oldPayment
				_ = sqlDB.Close()
			})
			require.NoError(t, db.AutoMigrate(&model.User{}, &model.UserSession{}, &model.UserAccessToken{}, &model.Option{}, &model.AuditLog{}, &model.AuthFlow{}, &model.TwoFA{}, &model.PasskeyCredential{}))
			suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
			users := []model.User{{Username: "buyer-" + suffix, Password: "unused", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, AuthVersion: 1, AffCode: "a" + suffix}, {Username: "admin-" + suffix, Password: "unused", Role: common.RoleAdminUser, Status: common.UserStatusEnabled, AuthVersion: 1, AffCode: "b" + suffix}, {Username: "other-" + suffix, Password: "unused", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, AuthVersion: 1, AffCode: "c" + suffix}}
			require.NoError(t, db.Create(&users).Error)
			if os.Getenv("TEST_GUARDIAN_LINKS_UPGRADE") == "1" {
				require.NoError(t, db.AutoMigrate(&guardianRC54AccountStatus{}))
				require.NoError(t, db.Create(&guardianRC54AccountStatus{GuardianAccountStatus: model.GuardianAccountStatus{UserID: users[2].Id, Banned: true, UpdatedAt: 123, UpdatedBy: users[1].Id}, HistoricalCategory: "banned_only"}).Error)
			}
			var legacy guardianServiceBeforeChip
			if os.Getenv("TEST_GUARDIAN_UPGRADE") == "1" {
				require.NoError(t, db.AutoMigrate(&guardianServiceBeforeChip{}, &guardianOrderBeforePeriod{}, &guardianAccountStatusBeforeLinks{}))
				require.NoError(t, db.Create(&guardianAccountStatusBeforeLinks{UserID: users[2].Id, Banned: true, UpdatedAt: 123, UpdatedBy: users[1].Id}).Error)
				legacy = guardianServiceBeforeChip{BaseSales: 5, Title: "Historical service", PriceCents: 19900, Features: model.GuardianFeatures{"Existing feature"}, Route: "any"}
				require.NoError(t, db.Create(&legacy).Error)
				require.NoError(t, db.Create(&guardianOrderBeforePeriod{AmountCents: 19900, TradeNo: "historical-" + suffix, ServiceID: legacy.ID, Status: "delivered"}).Error)
			}
			if os.Getenv("TEST_GUARDIAN_UPGRADE") == "1" {
				require.NoError(t, db.AutoMigrate(&guardianOrderBeforeReseller{}))
				require.NoError(t, db.Create(&guardianOrderBeforeReseller{Legacy: guardianOrderBeforePeriod{TradeNo: "rc51-" + suffix, AmountCents: 180000, Status: "delivered"}, BillingPeriod: "quarterly", ClaudeMaxPriceCents: 30000}).Error)
			}
			if os.Getenv("TEST_GUARDIAN_UPGRADE") == "1" {
				require.NoError(t, db.AutoMigrate(&guardianRC52ServiceOrder{}, &guardianRC52Reseller{}, &guardianRC52ResellerOffer{}))
				require.NoError(t, db.Create(&guardianRC52Reseller{UserID: users[1].Id, Status: "active", Code: "rc52-" + suffix}).Error)
				require.NoError(t, db.Create(&guardianRC52ResellerOffer{ResellerID: users[1].Id, ServiceID: legacy.ID, Granted: true, Listed: true, SupplyCents: 10000, RetailCents: 19900}).Error)
				require.NoError(t, db.Create(&guardianRC52ServiceOrder{TradeNo: "rc52-order-" + suffix, UserID: users[1].Id, ServiceID: legacy.ID, ServiceTitle: "rc52 original", AmountCents: 19900, PaidAt: 123, Status: "delivered"}).Error)
			}
			var rc53Product guardianRC53Service
			var rc53Order guardianRC53Order
			if os.Getenv("TEST_GUARDIAN_RC53_UPGRADE") == "1" {
				require.NoError(t, db.AutoMigrate(&guardianRC53Service{}, &guardianRC53Order{}, &guardianRC53Receipt{}, &guardianRC53Offer{}, &guardianRC53Sale{}, &guardianRC53Tier{}))
				rc53Product = guardianRC53Service{Title: "rc53 Mac", Route: "mac_mini", Enabled: true, PriceCents: 50000, ClaudeMaxPriceCents: 90000, ClaudeUltraPriceCents: 200000, InvoiceEnabled: true, Features: model.GuardianFeatures{"preserve"}}
				require.NoError(t, db.Create(&rc53Product).Error)
				rc53Order = guardianRC53Order{TradeNo: "rc53-" + suffix, UserID: users[0].Id, ServiceID: rc53Product.ID, AmountCents: 250000, ClaudeUltraPriceCents: 200000, InvoiceInformation: "original invoice", DiscoveryChannel: "github", PaidAt: 123, Status: "delivered"}
				require.NoError(t, db.Create(&rc53Order).Error)
				require.NoError(t, db.Create(&guardianRC53Receipt{OrderID: rc53Order.ID, Number: "rc53-receipt-" + suffix, AmountCents: 250000, UltraCents: 200000}).Error)
				require.NoError(t, db.Create(&guardianRC53Offer{ResellerID: users[2].Id, ServiceID: rc53Product.ID, SupplyCents: 40000, UltraSupplyCents: 190000, UltraRetailCents: 200000, Granted: true, Listed: true, FollowTier: true}).Error)
				require.NoError(t, db.Create(&guardianRC53Sale{OrderID: rc53Order.ID, ResellerID: users[2].Id, SupplyCents: 230000, UltraSupplyCents: 190000}).Error)
				require.NoError(t, db.Create(&guardianRC53Tier{Tier: 3, ServiceID: rc53Product.ID, Supply: guardianRC53Supply{SupplyCents: 40000, UltraSupplyCents: 190000, Granted: true}}).Error)
			}
			var rc54Order guardianRC54Order
			if os.Getenv("TEST_GUARDIAN_TERMS_UPGRADE") == "1" {
				require.NoError(t, db.AutoMigrate(&guardianRC54Order{}, &guardianRC54Service{}, &guardianRC54Receipt{}))
				rc54Order = guardianRC54Order{Legacy: guardianRC53Order{TradeNo: "terms-upgrade-" + suffix, UserID: users[0].Id, ServiceID: legacy.ID, AmountCents: 30000, PaidAt: 123, Status: "delivered", DiscoveryChannel: "github"}, ClaudeProPriceCents: 10000}
				require.NoError(t, db.Create(&rc54Order).Error)
				require.NoError(t, db.Create(&guardianRC54Receipt{Legacy: guardianRC53Receipt{OrderID: rc54Order.Legacy.ID, Number: "rc54-" + suffix, AmountCents: 30000, PaidAt: 123}, ProCents: 10000}).Error)
			}
			for range 2 {
				require.NoError(t, db.AutoMigrate(&model.GuardianAccountStatus{}, &model.GuardianService{}, &model.GuardianServiceOrder{}, &model.GuardianOrderEvent{}, &model.GuardianPaymentAnomaly{}, &model.GuardianReseller{}, &model.GuardianResellerEvent{}, &model.GuardianResellerOperation{}, &model.GuardianResellerSetting{}, &model.GuardianResellerOffer{}, &model.GuardianTierSupply{}, &model.GuardianReceipt{}, &model.GuardianResellerSale{}, &model.GuardianResellerWithdrawal{}, &model.GuardianCustomerService{}))
			}
			if rc54Order.Legacy.ID != 0 {
				var migrated model.GuardianServiceOrder
				require.NoError(t, db.First(&migrated, rc54Order.Legacy.ID).Error)
				assert.EqualValues(t, 30000, migrated.AmountCents)
				assert.EqualValues(t, 10000, migrated.ClaudeProPriceCents)
				assert.EqualValues(t, 123, migrated.PaidAt)
				assert.Equal(t, "delivered", migrated.Status)
				assert.Empty(t, migrated.IPOptionName)
				assert.Zero(t, migrated.IPOptionPriceCents)
				var receipt model.GuardianReceipt
				require.NoError(t, db.First(&receipt, "order_id = ?", migrated.ID).Error)
				assert.EqualValues(t, 30000, receipt.AmountCents)
				assert.EqualValues(t, 10000, receipt.ProCents)
				assert.EqualValues(t, 123, receipt.PaidAt)
				assert.Empty(t, receipt.IPOptionName)
				require.NoError(t, db.Delete(&receipt).Error)
				assert.Empty(t, migrated.AfterSalesTerms)
				assert.Zero(t, migrated.AfterSalesAcceptedAt)
				require.NoError(t, db.Delete(&migrated).Error)
			}
			if rc53Product.ID != 0 {
				// Reopen after migration to discard PostgreSQL prepared SELECT * plans.
				require.NoError(t, sqlDB.Close())
				db, err = gorm.Open(driver, &gorm.Config{})
				require.NoError(t, err)
				sqlDB, err = db.DB()
				require.NoError(t, err)
				model.DB, model.LOG_DB = db, db
				var product model.GuardianService
				require.NoError(t, db.First(&product, rc53Product.ID).Error)
				assert.True(t, product.InvoiceEnabled)
				assert.Empty(t, product.Tags)
				assert.False(t, product.OutOfStock)
				assert.False(t, product.ClaudeProOutOfStock)
				assert.False(t, product.ClaudeMaxOutOfStock)
				assert.False(t, product.ClaudeUltraOutOfStock)
				assert.Zero(t, product.ClaudeProPriceCents)
				assert.Equal(t, rc53Product.ClaudeUltraPriceCents, product.ClaudeUltraPriceCents)
				var order model.GuardianServiceOrder
				require.NoError(t, db.First(&order, rc53Order.ID).Error)
				assert.Equal(t, rc53Order.AmountCents, order.AmountCents)
				assert.Equal(t, rc53Order.InvoiceInformation, order.InvoiceInformation)
				assert.Equal(t, rc53Order.DiscoveryChannel, order.DiscoveryChannel)
				assert.Zero(t, order.ClaudeProPriceCents)
				var receipt model.GuardianReceipt
				require.NoError(t, db.First(&receipt, order.ID).Error)
				assert.EqualValues(t, 200000, receipt.UltraCents)
				assert.Zero(t, receipt.ProCents)
				var offer model.GuardianResellerOffer
				require.NoError(t, db.Where("service_id = ?", product.ID).First(&offer).Error)
				assert.True(t, offer.FollowTier)
				assert.EqualValues(t, 190000, offer.UltraSupplyCents)
				assert.Zero(t, offer.ProSupplyCents)
				var sale model.GuardianResellerSale
				require.NoError(t, db.First(&sale, order.ID).Error)
				assert.EqualValues(t, 230000, sale.SupplyCents)
				assert.Zero(t, sale.ProSupplyCents)
				var tier model.GuardianTierSupply
				require.NoError(t, db.Where("service_id = ?", product.ID).First(&tier).Error)
				assert.EqualValues(t, 190000, tier.UltraSupplyCents)
				assert.Zero(t, tier.ProSupplyCents)
				require.NoError(t, db.Delete(&receipt).Error)
				require.NoError(t, db.Delete(&sale).Error)
				require.NoError(t, db.Delete(&offer).Error)
				require.NoError(t, db.Delete(&tier).Error)
				require.NoError(t, db.Delete(&order).Error)
				require.NoError(t, db.Delete(&product).Error)
			}
			if legacy.ID != 0 {
				var oldReseller model.GuardianReseller
				require.NoError(t, db.First(&oldReseller, "user_id = ?", users[1].Id).Error)
				assert.Zero(t, oldReseller.Tier)
				oldOffers, err := model.ListGuardianResellerOffers(users[1].Id)
				require.NoError(t, err)
				require.Len(t, oldOffers, 1)
				assert.False(t, oldOffers[0].FollowTier)
				assert.EqualValues(t, 10000, oldOffers[0].SupplyCents)
				assert.EqualValues(t, 19900, oldOffers[0].RetailCents)
				var oldOrder model.GuardianServiceOrder
				require.NoError(t, db.Where("trade_no = ?", "rc52-order-"+suffix).First(&oldOrder).Error)
				assert.Empty(t, oldOrder.DiscoveryChannel)
				assert.False(t, oldOrder.ReceiptNotification)
				assert.EqualValues(t, 19900, oldOrder.AmountCents)
				assert.EqualValues(t, 123, oldOrder.PaidAt)
				require.NoError(t, db.Delete(&oldOrder).Error)
				require.NoError(t, db.Delete(&oldReseller).Error)
				require.NoError(t, db.Delete(&oldOffers).Error)

				var released model.GuardianServiceOrder
				require.NoError(t, db.Where("trade_no = ?", "rc51-"+suffix).First(&released).Error)
				assert.EqualValues(t, 180000, released.AmountCents)
				assert.Equal(t, "quarterly", released.BillingPeriod)
				assert.EqualValues(t, 30000, released.ClaudeMaxPriceCents)
				assert.Zero(t, released.ClaudeUltraPriceCents)
				assert.Empty(t, released.ResellerCode)
				assert.Zero(t, released.CustomerServiceID)

				var migratedStatus model.GuardianAccountStatus
				require.NoError(t, db.First(&migratedStatus, "user_id = ?", users[2].Id).Error)
				assert.True(t, migratedStatus.Banned)
				assert.EqualValues(t, 123, migratedStatus.UpdatedAt)
				var migrated model.GuardianService
				require.NoError(t, db.First(&migrated, legacy.ID).Error)
				assert.Equal(t, legacy.Title, migrated.Title)
				assert.Equal(t, legacy.Features, migrated.Features)
				assert.Equal(t, legacy.PriceCents, migrated.PriceCents)
				assert.Equal(t, legacy.BaseSales, migrated.BaseSales)
				assert.Empty(t, migrated.Chip)
				assert.Empty(t, migrated.Region)
				assert.Empty(t, migrated.HardwareConfig)
				assert.Zero(t, migrated.QuarterPriceCents)
				assert.Zero(t, migrated.AnnualPriceCents)
				assert.Zero(t, migrated.ClaudeMaxPriceCents)
				assert.Zero(t, migrated.ClaudeUltraPriceCents)
				var historicalOrder model.GuardianServiceOrder
				require.NoError(t, db.Where("trade_no = ?", "historical-"+suffix).First(&historicalOrder).Error)
				assert.Empty(t, historicalOrder.AfterSalesTerms)
				assert.Zero(t, historicalOrder.AfterSalesAcceptedAt)
				assert.Empty(t, historicalOrder.BillingPeriod)
				assert.Zero(t, historicalOrder.ClaudeMaxPriceCents)
				assert.Zero(t, historicalOrder.ClaudeUltraPriceCents)
				assert.EqualValues(t, 19900, historicalOrder.AmountCents)
				services := []model.GuardianService{migrated}
				require.NoError(t, model.PopulateGuardianSales(services))
				assert.EqualValues(t, 6, *services[0].SoldCount)
			}
			require.NoError(t, model.SaveGuardianAfterSalesTerms(model.GuardianAfterSalesTerms{Deployment: "test terms", Mac: "test terms"}))
			t.Run("after-sales consent", func(t *testing.T) {
				product := model.GuardianService{Title: "Terms test", Enabled: true, Route: "vps", PriceCents: 100}
				require.NoError(t, db.Create(&product).Error)
				t.Cleanup(func() {
					db.Delete(&product)
					require.NoError(t, model.SaveGuardianAfterSalesTerms(model.GuardianAfterSalesTerms{Deployment: "test terms", Mac: "test terms"}))
				})
				req := model.GuardianOrderRequest{ServiceID: product.ID, DiscoveryChannel: "github", Route: "vps", OS: "linux", ContactType: "email", Contact: "test@example.com", PaymentMethod: "alipay"}
				_, err := model.CreateGuardianOrder(users[0].Id, req)
				require.EqualError(t, err, "请阅读并同意售后条款")
				req.AcceptAfterSalesTerms = true
				req.AfterSalesTerms = "old terms"
				_, err = model.CreateGuardianOrder(users[0].Id, req)
				require.EqualError(t, err, "售后条款已更新，请重新打开下单窗口阅读并同意")
				req.AfterSalesTerms = "test terms"
				order, err := model.CreateGuardianOrder(users[0].Id, req)
				require.NoError(t, err)
				t.Cleanup(func() { db.Where("order_id = ?", order.ID).Delete(&model.GuardianOrderEvent{}); db.Delete(order) })
				assert.Equal(t, "test terms", order.AfterSalesTerms)
				assert.Equal(t, order.CreatedAt, order.AfterSalesAcceptedAt)
				require.NoError(t, model.SaveGuardianAfterSalesTerms(model.GuardianAfterSalesTerms{Deployment: "new deployment terms", Mac: "mac terms"}))
				require.NoError(t, db.First(order, order.ID).Error)
				assert.Equal(t, "test terms", order.AfterSalesTerms)
				require.NoError(t, db.Model(&product).Update("route", "mac_mini").Error)
				req.Route = "mac_mini"
				req.AfterSalesTerms = "new deployment terms"
				_, err = model.CreateGuardianOrder(users[0].Id, req)
				require.EqualError(t, err, "售后条款已更新，请重新打开下单窗口阅读并同意")
				require.NoError(t, model.SaveGuardianAfterSalesTerms(model.GuardianAfterSalesTerms{}))
				_, err = model.CreateGuardianOrder(users[0].Id, req)
				require.EqualError(t, err, "该分类尚未配置售后条款，暂不可下单")
			})
			t.Run("environment IP add-ons", func(t *testing.T) {
				product := model.GuardianService{Title: "IP test", Route: "vps", Enabled: true, IPEnabled: true, PriceCents: 19900, IPOptions: model.GuardianIPOptions{{ID: "residential", Name: "Static residential IP", Description: "One month", PriceCents: 5000, QuarterPriceCents: 14000, AnnualPriceCents: 50000, Enabled: true}, {ID: "disabled", Name: "Unavailable", PriceCents: 6000, QuarterPriceCents: 17000, AnnualPriceCents: 60000}}}
				require.NoError(t, product.Validate())
				require.NoError(t, db.Create(&product).Error)
				t.Cleanup(func() { db.Delete(&product) })
				total := int64(24900)
				req := model.GuardianOrderRequest{AcceptAfterSalesTerms: true, AfterSalesTerms: "test terms", DiscoveryChannel: "github", ServiceID: product.ID, Route: "vps", OS: "linux", ContactType: "email", Contact: "ip@example.test", PaymentMethod: "alipay", IPOptionID: "residential", IPBillingPeriod: "monthly", ExpectedAmountCents: &total}
				for _, id := range []string{"", "missing", "disabled"} {
					bad := req
					bad.IPOptionID = id
					_, err := model.CreateGuardianOrder(users[0].Id, bad)
					require.ErrorContains(t, err, "IP")
				}
				for _, period := range []string{"", "weekly"} {
					bad := req
					bad.IPBillingPeriod = period
					_, err := model.CreateGuardianOrder(users[0].Id, bad)
					require.ErrorContains(t, err, "IP")
				}
				for _, tc := range []struct {
					period string
					price  int64
				}{{"quarterly", 14000}, {"yearly", 50000}} {
					periodReq := req
					periodReq.IPBillingPeriod = tc.period
					amount := int64(19900) + tc.price
					periodReq.ExpectedAmountCents = &amount
					periodOrder, err := model.CreateGuardianOrder(users[0].Id, periodReq)
					require.NoError(t, err)
					assert.Equal(t, amount, periodOrder.AmountCents)
					assert.Equal(t, tc.price, periodOrder.IPOptionPriceCents)
					assert.Equal(t, tc.period, periodOrder.IPBillingPeriod)
					require.NoError(t, db.Where("order_id = ?", periodOrder.ID).Delete(&model.GuardianOrderEvent{}).Error)
					require.NoError(t, db.Delete(periodOrder).Error)
				}
				require.NoError(t, db.Model(&product).Update("ip_enabled", false).Error)
				_, err := model.CreateGuardianOrder(users[0].Id, req)
				require.ErrorContains(t, err, "未开启")
				require.NoError(t, db.Model(&product).Update("ip_enabled", true).Error)
				bad := req
				wrong := int64(19900)
				bad.ExpectedAmountCents = &wrong
				_, err = model.CreateGuardianOrder(users[0].Id, bad)
				require.ErrorContains(t, err, "报价")
				order, err := model.CreateGuardianOrder(users[0].Id, req)
				require.NoError(t, err)
				t.Cleanup(func() {
					db.Where("order_id = ?", order.ID).Delete(&model.GuardianOrderEvent{})
					db.Where("order_id = ?", order.ID).Delete(&model.GuardianPaymentAnomaly{})
					db.Where("order_id = ?", order.ID).Delete(&model.GuardianReceipt{})
					db.Delete(order)
				})
				assert.EqualValues(t, 24900, order.AmountCents)
				assert.EqualValues(t, 5000, order.IPOptionPriceCents)
				assert.Equal(t, "Static residential IP", order.IPOptionName)
				assert.Zero(t, order.ClaudeMaxPriceCents)
				require.NoError(t, db.Model(&product).Update("ip_options", model.GuardianIPOptions{}).Error)
				require.NoError(t, db.First(order, order.ID).Error)
				assert.EqualValues(t, 5000, order.IPOptionPriceCents)
				assert.Equal(t, "One month", order.IPOptionDescription)
				require.Error(t, model.CompleteGuardianOrder(order.TradeNo, "fixture", "199.00", "alipay"))
				require.NoError(t, model.CompleteGuardianOrder(order.TradeNo, "fixture", "249.00", "alipay"))
				receipt, err := model.GetGuardianReceipt(order.ID, users[0].Id, false)
				require.NoError(t, err)
				assert.EqualValues(t, 24900, receipt.AmountCents)
				assert.EqualValues(t, 5000, receipt.IPOptionPriceCents)
				assert.Equal(t, "monthly", receipt.IPBillingPeriod)
				require.NoError(t, db.Model(&product).Update("route", "mac_mini").Error)
				req.Route = "mac_mini"
				_, err = model.CreateGuardianOrder(users[0].Id, req)
				require.ErrorContains(t, err, "Mac")
				for _, price := range []int64{-1, 0, 10000001} {
					options := model.GuardianIPOptions{{ID: "a", Name: "IP", PriceCents: price}}
					require.Error(t, options.Validate())
				}
				require.Error(t, model.GuardianIPOptions{{ID: "a", Name: "IP", PriceCents: 1}, {ID: "a", Name: "IP", PriceCents: 1}}.Validate())
			})
			var preserved model.User
			require.NoError(t, db.First(&preserved, users[0].Id).Error)
			assert.Equal(t, users[0].Username, preserved.Username)
			tokens := make([]string, len(users))
			for i := range users {
				bundle, err := service.CreateLoginSession(users[i].Id, "password", "127.0.0.1", "guardian-test")
				require.NoError(t, err)
				tokens[i] = bundle.AccessToken
			}
			require.NoError(t, model.EnsureLegacyAccessTokenRetireAt(time.Now().Unix()))
			gin.SetMode(gin.TestMode)
			var accessLog bytes.Buffer
			previousWriter := gin.DefaultWriter
			gin.DefaultWriter = &accessLog
			t.Cleanup(func() { gin.DefaultWriter = previousWriter })
			engine := gin.New()
			middleware.SetUpLogger(engine)
			SetApiRouter(engine)
			request := func(method, path, token, body string) *httptest.ResponseRecorder {
				req := httptest.NewRequest(method, path, strings.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				req.RemoteAddr = "198.51.100.41:1234"
				if token != "" {
					req.Header.Set("Authorization", "Bearer "+token)
				}
				response := httptest.NewRecorder()
				engine.ServeHTTP(response, req)
				return response
			}
			decode := func(response *httptest.ResponseRecorder) map[string]any {
				t.Helper()
				var result map[string]any
				require.NoError(t, common.Unmarshal(response.Body.Bytes(), &result), response.Body.String())
				return result
			}
			require.Equal(t, true, decode(request("PUT", "/api/guardian/admin/after-sales-terms", tokens[1], `{"deployment":"test terms","mac":"test terms"}`))["success"])
			termsResponse := decode(request("GET", "/api/guardian/admin/after-sales-terms", tokens[1], ""))
			require.Equal(t, map[string]any{"deployment": "test terms", "mac": "test terms"}, termsResponse["data"])
			for _, route := range []struct{ method, path string }{{"GET", "/api/guardian/admin/after-sales-terms"}, {"PUT", "/api/guardian/admin/after-sales-terms"}, {"POST", "/api/guardian/orders"}, {"GET", "/api/guardian/orders/self"}, {"GET", "/api/guardian/admin/services"}, {"POST", "/api/guardian/admin/services"}, {"PUT", "/api/guardian/admin/services/1"}, {"GET", "/api/guardian/admin/orders"}, {"PUT", "/api/guardian/admin/orders/1"}, {"GET", "/api/guardian/admin/queue"}, {"PUT", "/api/guardian/admin/queue/settings"}, {"POST", "/api/guardian/admin/queue/1/move"}} {
				assert.Equal(t, http.StatusUnauthorized, request(route.method, route.path, "", "{}").Code)
				if strings.Contains(route.path, "/admin/") {
					assert.Equal(t, http.StatusForbidden, request(route.method, route.path, tokens[0], "{}").Code)
				}
			}
			tokenSuffix, err := common.GenerateRandomCharsKey(43)
			require.NoError(t, err)
			pat := model.AccessTokenPrefix + tokenSuffix
			grant := &model.UserAccessToken{Name: "guardian test", TokenHash: model.AccessTokenFingerprint(pat), TokenHint: model.AccessTokenHint(pat)}
			require.NoError(t, grant.SetScopes([]string{"billing:read", "billing:write"}))
			require.NoError(t, model.CreateUserAccessToken(users[1].Id, grant, service.AccessTokenMaxPerUser))
			assert.Equal(t, http.StatusForbidden, request("GET", "/api/guardian/admin/orders", pat, "").Code)
			for _, route := range []struct{ method, path string }{
				{"GET", "/api/guardian/admin/account-statuses"},
				{"PUT", "/api/guardian/admin/account-statuses/1"},
			} {
				assert.Equal(t, http.StatusUnauthorized, request(route.method, route.path, "", "{}").Code)
				assert.Equal(t, http.StatusForbidden, request(route.method, route.path, tokens[0], "{}").Code)
				assert.Equal(t, http.StatusForbidden, request(route.method, route.path, pat, "{}").Code)
			}
			assert.Equal(t, http.StatusUnauthorized, request("GET", "/api/guardian/orders/self", "invalid-token", "").Code)
			payload := `{"title":" VPS 部署 ","subtitle":"环境","description":"说明","features":[" Linux "," "],"price_cents":59901,"delivery_note":"三天","badge":"","route":"vps","enabled":true,"sort_order":10}`
			created := decode(request("POST", "/api/guardian/admin/services", tokens[1], payload))
			require.Equal(t, true, created["success"])
			serviceData := created["data"].(map[string]any)
			id := int(serviceData["id"].(float64))
			assert.Equal(t, "VPS 部署", serviceData["title"])
			assert.Equal(t, []any{"Linux"}, serviceData["features"])
			catalog := decode(request("GET", "/api/guardian/services", "", ""))["data"].(map[string]any)
			assert.Equal(t, true, catalog["purchase_enabled"])
			assert.Len(t, catalog["pay_methods"], 1)

			t.Run("admin IP options persist updates and removal", func(t *testing.T) {
				t.Cleanup(func() {
					require.Equal(t, true, decode(request("PUT", fmt.Sprintf("/api/guardian/admin/services/%d", id), tokens[1], payload))["success"])
				})
				path := fmt.Sprintf("/api/guardian/admin/services/%d", id)
				for _, options := range []string{
					`[{"id":"residential","name":"Residential IP","description":"One time","price_cents":5000,"quarter_price_cents":14000,"annual_price_cents":50000,"enabled":true}]`,
					`[{"id":"residential","name":"Updated IP","description":"Updated","price_cents":6000,"quarter_price_cents":17000,"annual_price_cents":60000,"enabled":false}]`,
					`[]`,
				} {
					body := strings.TrimSuffix(payload, "}") + `,"ip_enabled":true,"ip_options":` + options + "}"
					require.Equal(t, true, decode(request("PUT", path, tokens[1], body))["success"])
					var stored model.GuardianService
					require.NoError(t, db.First(&stored, id).Error)
					var expected model.GuardianIPOptions
					require.NoError(t, common.UnmarshalJsonStr(options, &expected))
					assert.Equal(t, expected, stored.IPOptions)
					assert.True(t, stored.IPEnabled)
					assert.EqualValues(t, 59901, stored.PriceCents)
				}
			})

			t.Run("optional display tags persist and clear without affecting price", func(t *testing.T) {
				withTags := strings.Replace(payload, `"badge":""`, `"badge":"","tags":[" 静态 IP ","Claude Pro 订阅","静态 IP",""]`, 1)
				created := decode(request("POST", "/api/guardian/admin/services", tokens[1], withTags))
				require.Equal(t, true, created["success"])
				data := created["data"].(map[string]any)
				tagID := int(data["id"].(float64))
				defer db.Delete(&model.GuardianService{}, tagID)
				assert.Equal(t, []any{"静态 IP", "Claude Pro 订阅"}, data["tags"])
				var stored model.GuardianService
				require.NoError(t, db.First(&stored, tagID).Error)
				assert.Equal(t, model.GuardianFeatures{"静态 IP", "Claude Pro 订阅"}, stored.Tags)
				assert.EqualValues(t, 59901, stored.PriceCents)
				path := fmt.Sprintf("/api/guardian/admin/services/%d", tagID)
				for _, tags := range []string{`["` + strings.Repeat("字", 25) + `"]`, `["1","2","3","4","5","6","7","8","9"]`} {
					invalid := strings.Replace(payload, `"badge":""`, `"badge":"","tags":`+tags, 1)
					assert.Equal(t, false, decode(request("PUT", path, tokens[1], invalid))["success"])
				}
				cleared := strings.Replace(payload, `"badge":""`, `"badge":"","tags":[]`, 1)
				assert.Equal(t, true, decode(request("PUT", path, tokens[1], cleared))["success"])
				require.NoError(t, db.First(&stored, tagID).Error)
				assert.Empty(t, stored.Tags)
				assert.EqualValues(t, 59901, stored.PriceCents)
			})

			t.Run("mac mini uses existing service and order storage", func(t *testing.T) {
				macPayload := strings.Replace(payload, `"route":"vps"`, `"route":"mac_mini","chip":"m4","region":"kr_seoul","hardware_config":"16_256","quarter_price_cents":150000,"annual_price_cents":500000,"claude_max_price_cents":100000,"claude_ultra_price_cents":250000`, 1)
				result := decode(request("POST", "/api/guardian/admin/services", tokens[1], macPayload))
				require.Equal(t, true, result["success"])
				macID := int(result["data"].(map[string]any)["id"].(float64))
				defer db.Delete(&model.GuardianService{}, macID)
				assert.Equal(t, "m4", result["data"].(map[string]any)["chip"])
				servicePath := fmt.Sprintf("/api/guardian/admin/services/%d", macID)
				updated := strings.Replace(macPayload, `"chip":"m4"`, `"chip":"m6"`, 1)
				require.Equal(t, true, decode(request("PUT", servicePath, tokens[1], updated))["success"])
				updated = strings.Replace(updated, `"region":"kr_seoul"`, `"region":"au_sydney"`, 1)
				require.Equal(t, true, decode(request("PUT", servicePath, tokens[1], updated))["success"])
				var storedService model.GuardianService
				require.NoError(t, db.First(&storedService, macID).Error)
				assert.Equal(t, "m6", storedService.Chip)
				assert.Equal(t, "au_sydney", storedService.Region)
				assert.Equal(t, "16_256", storedService.HardwareConfig)
				assert.EqualValues(t, 150000, storedService.QuarterPriceCents)
				assert.EqualValues(t, 500000, storedService.AnnualPriceCents)
				assert.EqualValues(t, 100000, storedService.ClaudeMaxPriceCents)
				assert.EqualValues(t, 250000, storedService.ClaudeUltraPriceCents)
				public := decode(request("GET", "/api/guardian/services", "", ""))["data"].(map[string]any)["services"].([]any)
				var publicChip any
				for _, item := range public {
					entry := item.(map[string]any)
					if int(entry["id"].(float64)) == macID {
						publicChip = entry["chip"]
						assert.Equal(t, "au_sydney", entry["region"])
						assert.Equal(t, "16_256", entry["hardware_config"])
						assert.Equal(t, float64(500000), entry["annual_price_cents"])
						assert.Equal(t, float64(100000), entry["claude_max_price_cents"])
						assert.Equal(t, float64(250000), entry["claude_ultra_price_cents"])
					}
				}
				assert.Equal(t, "m6", publicChip)
				invalid := strings.Replace(updated, `"chip":"m6"`, `"chip":"m5"`, 1)
				assert.Equal(t, false, decode(request("PUT", servicePath, tokens[1], invalid))["success"])
				require.NoError(t, db.First(&storedService, macID).Error)
				assert.Equal(t, "m6", storedService.Chip)
				cleared := strings.Replace(updated, `"chip":"m6"`, `"chip":""`, 1)
				require.Equal(t, true, decode(request("PUT", servicePath, tokens[1], cleared))["success"])
				require.NoError(t, db.First(&storedService, macID).Error)
				assert.Empty(t, storedService.Chip)

				for _, invalid := range []struct{ from, to string }{
					{`"region":"au_sydney"`, `"region":"unknown"`},
					{`"hardware_config":"16_256"`, `"hardware_config":"64_1024"`},
					{`"quarter_price_cents":150000`, `"quarter_price_cents":-1`},
					{`"annual_price_cents":500000`, `"annual_price_cents":10000001`},
					{`"claude_max_price_cents":100000`, `"claude_max_price_cents":-1`},
					{`"claude_ultra_price_cents":250000`, `"claude_ultra_price_cents":-1`},
					{`"claude_ultra_price_cents":250000`, `"claude_ultra_price_cents":10000001`},
					{`"claude_max_price_cents":100000`, `"claude_max_price_cents":10000001`},
				} {
					assert.Equal(t, false, decode(request("PUT", servicePath, tokens[1], strings.Replace(updated, invalid.from, invalid.to, 1)))["success"])
				}
				for _, tc := range []struct {
					period string
					amount int64
					addon  int64
					ultra  int64
				}{
					{"monthly", 59901, 0, 0}, {"quarterly", 150000, 0, 0}, {"yearly", 500000, 0, 0},
					{"monthly", 159901, 100000, 0}, {"quarterly", 250000, 100000, 0}, {"yearly", 600000, 100000, 0},
					{"monthly", 309901, 0, 250000}, {"quarterly", 400000, 0, 250000}, {"yearly", 750000, 0, 250000},
				} {
					body := fmt.Sprintf(`{"accept_after_sales_terms":true,"after_sales_terms":"test terms","discovery_channel":"github","service_id":%d,"billing_period":%q,"add_claude_max":%t,"add_claude_ultra":%t,"claude_ultra_price_cents":1,"claude_max_price_cents":1,"amount_cents":1,"payment_method":"alipay","route":"mac_mini","os":"macos","contact_type":"email","contact":"buyer@example.com"}`, macID, tc.period, tc.addon > 0, tc.ultra > 0)
					result := decode(request("POST", "/api/guardian/orders", tokens[0], body))
					require.Equal(t, true, result["success"], result)
					var saved model.GuardianServiceOrder
					require.NoError(t, db.Where("trade_no = ?", result["data"].(map[string]any)["trade_no"]).First(&saved).Error)
					assert.Equal(t, tc.period, saved.BillingPeriod)
					assert.Equal(t, tc.amount, saved.AmountCents)
					assert.Equal(t, tc.addon, saved.ClaudeMaxPriceCents)
					assert.Equal(t, tc.ultra, saved.ClaudeUltraPriceCents)
					assert.Equal(t, fmt.Sprintf("%.2f", float64(tc.amount)/100), result["data"].(map[string]any)["pay_params"].(map[string]any)["money"])
					require.NoError(t, db.Model(&storedService).Updates(map[string]any{"annual_price_cents": 400000, "claude_max_price_cents": 50000, "claude_ultra_price_cents": 200000}).Error)
					require.NoError(t, db.First(&saved, saved.ID).Error)
					assert.Equal(t, tc.amount, saved.AmountCents)
					assert.Equal(t, tc.addon, saved.ClaudeMaxPriceCents)
					assert.Equal(t, tc.ultra, saved.ClaudeUltraPriceCents)
					require.NoError(t, db.Model(&storedService).Updates(map[string]any{"annual_price_cents": 500000, "claude_max_price_cents": 100000, "claude_ultra_price_cents": 250000}).Error)
					if tc.addon+tc.ultra > 0 {
						require.Error(t, model.CompleteGuardianOrder(saved.TradeNo, "{}", fmt.Sprintf("%.2f", float64(tc.amount-tc.addon-tc.ultra)/100), "alipay"))
						require.NoError(t, model.CompleteGuardianOrder(saved.TradeNo, "{}", fmt.Sprintf("%.2f", float64(tc.amount)/100), "alipay"))
						require.NoError(t, db.First(&saved, saved.ID).Error)
						assert.Equal(t, "paid", saved.Status)
					}
					require.NoError(t, db.Where("order_id = ?", saved.ID).Delete(&model.GuardianOrderEvent{}).Error)
					require.NoError(t, db.Delete(&saved).Error)
				}
				baseRequest := model.GuardianOrderRequest{AcceptAfterSalesTerms: true, AfterSalesTerms: "test terms", DiscoveryChannel: "github", ServiceID: macID, PaymentMethod: "alipay", Route: "mac_mini", OS: "macos", ContactType: "email", Contact: "buyer@example.com"}
				for _, mismatch := range []struct {
					serviceID int
					route     string
				}{
					{macID, "vps"}, {id, "mac_mini"},
				} {
					body := fmt.Sprintf(`{"accept_after_sales_terms":true,"after_sales_terms":"test terms","discovery_channel":"github","service_id":%d,"payment_method":"alipay","route":%q,"os":"macos","contact_type":"email","contact":"buyer@example.com"}`, mismatch.serviceID, mismatch.route)
					assert.Equal(t, false, decode(request("POST", "/api/guardian/orders", tokens[0], body))["success"])
				}
				for _, period := range []string{"weekly", "yearly"} {
					baseRequest.BillingPeriod = period
					require.NoError(t, db.Model(&storedService).Update("annual_price_cents", 0).Error)
					_, err := model.CreateGuardianOrder(users[0].Id, baseRequest)
					require.Error(t, err)
				}
				baseRequest.ServiceID = id
				_, err := model.CreateGuardianOrder(users[0].Id, baseRequest)
				require.Error(t, err)

				baseRequest.BillingPeriod = ""
				baseRequest.AddClaudeMax = true
				_, err = model.CreateGuardianOrder(users[0].Id, baseRequest)
				require.Error(t, err)
				baseRequest.ServiceID = macID
				disabledAddon := strings.Replace(updated, `"claude_max_price_cents":100000`, `"claude_max_price_cents":0`, 1)
				require.Equal(t, true, decode(request("PUT", servicePath, tokens[1], disabledAddon))["success"])
				_, err = model.CreateGuardianOrder(users[0].Id, baseRequest)
				require.Error(t, err)

				ultraRequest := model.GuardianOrderRequest{AcceptAfterSalesTerms: true, AfterSalesTerms: "test terms", DiscoveryChannel: "github", ServiceID: macID, PaymentMethod: "alipay", Route: "mac_mini", OS: "macos", ContactType: "email", Contact: "buyer@example.com", AddClaudeUltra: true, AddClaudeMax: true}
				_, err = model.CreateGuardianOrder(users[0].Id, ultraRequest)
				require.ErrorContains(t, err, "只能选择一项")
				ultraRequest.AddClaudeMax = false
				require.NoError(t, db.Model(&storedService).Update("claude_ultra_price_cents", 0).Error)
				_, err = model.CreateGuardianOrder(users[0].Id, ultraRequest)
				require.ErrorContains(t, err, "尚未开放")
				ultraRequest.ServiceID, ultraRequest.Route = id, "vps"
				_, err = model.CreateGuardianOrder(users[0].Id, ultraRequest)
				require.Error(t, err)

				order, err := model.CreateGuardianOrder(users[0].Id, model.GuardianOrderRequest{AcceptAfterSalesTerms: true, AfterSalesTerms: "test terms", DiscoveryChannel: "github",
					ServiceID: macID, PaymentMethod: "alipay", Route: "mac_mini", OS: "macos", ContactType: "email", Contact: "buyer@example.com",
				})
				require.NoError(t, err)
				defer db.Delete(order)
				defer db.Where("order_id = ?", order.ID).Delete(&model.GuardianOrderEvent{})
				var stored model.GuardianServiceOrder
				require.NoError(t, db.First(&stored, order.ID).Error)
				assert.Equal(t, "monthly", stored.BillingPeriod)
				assert.EqualValues(t, 59901, stored.AmountCents)
				assert.Equal(t, "mac_mini", stored.Route)
				assert.Equal(t, "macos", stored.OS)
				assert.Equal(t, "pending_payment", stored.Status)
			})
			t.Run("sales", func(t *testing.T) {
				servicePath := fmt.Sprintf("/api/guardian/admin/services/%d", id)
				for _, invalid := range []string{"-1", "1.5", "1000000001"} {
					body := strings.Replace(payload, `"sort_order":10`, `"sort_order":10,"base_sales":`+invalid, 1)
					assert.Equal(t, false, decode(request("PUT", servicePath, tokens[1], body))["success"])
				}
				rows := []model.GuardianServiceOrder{}
				for i, status := range []string{"pending_payment", "paid", "scheduled", "in_progress", "delivered", "expired", "cancelled", "refunded"} {
					rows = append(rows, model.GuardianServiceOrder{TradeNo: fmt.Sprintf("sales-%s-%d", suffix, i), ServiceID: id, UserID: users[0].Id, Status: status})
				}
				rows = append(rows, model.GuardianServiceOrder{TradeNo: "sales-other-" + suffix, ServiceID: id + 100000, Status: "paid"})
				require.NoError(t, db.Create(&rows).Error)
				defer func() {
					for _, row := range rows {
						require.NoError(t, db.Delete(&row).Error)
					}
				}()
				body := strings.Replace(payload, `"sort_order":10`, `"sort_order":10,"base_sales":5,"sold_count":999,"actual_sales":999`, 1)
				result := decode(request("PUT", servicePath, tokens[1], body))
				require.Equal(t, true, result["success"])
				data := result["data"].(map[string]any)
				assert.Equal(t, float64(4), data["actual_sales"])
				assert.Equal(t, float64(9), data["sold_count"])
				var public struct {
					Data struct{ Services []model.GuardianService }
				}
				require.NoError(t, common.Unmarshal(request("GET", "/api/guardian/services", "", "").Body.Bytes(), &public))
				found := false
				for _, item := range public.Data.Services {
					if item.ID == id {
						found = true
						assert.EqualValues(t, 9, *item.SoldCount)
					}
				}
				require.True(t, found)
				require.NoError(t, db.Model(&rows[1]).Update("status", "delivered").Error)
				services := []model.GuardianService{{ID: id, BaseSales: 5}}
				require.NoError(t, model.PopulateGuardianSales(services))
				assert.EqualValues(t, 9, *services[0].SoldCount)
				require.NoError(t, db.Model(&rows[1]).Update("status", "refunded").Error)
				require.NoError(t, model.PopulateGuardianSales(services))
				assert.EqualValues(t, 8, *services[0].SoldCount)
				reset := decode(request("PUT", servicePath, tokens[1], payload))
				require.Equal(t, true, reset["success"])
				assert.Equal(t, float64(3), reset["data"].(map[string]any)["sold_count"])
				var stored model.GuardianService
				require.NoError(t, db.First(&stored, id).Error)
				assert.Zero(t, stored.BaseSales)
			})
			t.Run("account_status", func(t *testing.T) {
				path := "/api/guardian/admin/account-statuses"
				assert.Equal(t, http.StatusUnauthorized, request("GET", path, "", "").Code)
				assert.Equal(t, http.StatusForbidden, request("GET", path, tokens[0], "").Code)
				snapshot := func() model.GuardianAccountStatistics {
					t.Helper()
					response := request("GET", "/api/guardian/account-statistics", "", "")
					require.Equal(t, http.StatusOK, response.Code)
					var result struct {
						Success bool
						Data    model.GuardianAccountStatistics
					}
					require.NoError(t, common.Unmarshal(response.Body.Bytes(), &result))
					require.True(t, result.Success)
					assert.NotContains(t, response.Body.String(), "username")
					assert.NotContains(t, response.Body.String(), "user_id")
					return result.Data
				}

				baseline := snapshot()
				require.NoError(t, db.Create(&model.Option{Key: "GuardianHistoricalAccounts", Value: `{"total_users":20,"banned_only":3}`}).Error)
				assert.Equal(t, baseline, snapshot(), "retired historical settings must not affect real purchaser statistics")
				t.Cleanup(func() { db.Where(&model.Option{Key: "GuardianHistoricalAccounts"}).Delete(&model.Option{}) })
				before := snapshot()
				var service model.GuardianService
				require.NoError(t, db.First(&service, id).Error)
				originalBase := service.BaseSales
				for _, baseSales := range []int64{1, 2, 0} {
					require.NoError(t, db.Model(&service).Update("base_sales", baseSales).Error)
					expected := before
					expected.PurchasedUsers += baseSales - originalBase
					expected.RecordedUsers += baseSales - originalBase
					expected.HealthyUsers += baseSales - originalBase
					assert.Equal(t, expected, snapshot())
					assert.Equal(t, expected, snapshot(), "reading statistics must not accumulate default accounts")
				}
				require.NoError(t, db.Model(&service).Update("base_sales", originalBase).Error)
				assert.Equal(t, before, snapshot())

				buyers := []model.User{
					{Username: "status-a-" + suffix, AffCode: "sa" + suffix},
					{Username: "status-b-" + suffix, AffCode: "sb" + suffix},
				}
				require.NoError(t, db.Create(&buyers).Error)
				orders := []model.GuardianServiceOrder{
					{TradeNo: "status-a-" + suffix, ServiceID: id, UserID: buyers[0].Id, Status: "paid"},
					{TradeNo: "status-a2-" + suffix, ServiceID: id, UserID: buyers[0].Id, Status: "delivered"},
					{TradeNo: "status-b-" + suffix, ServiceID: id, UserID: buyers[1].Id, Status: "in_progress"},
				}
				require.NoError(t, db.Create(&orders).Error)
				after := snapshot()
				assert.Equal(t, before.PurchasedUsers+2, after.PurchasedUsers)
				assert.Equal(t, before.RecordedUsers, after.RecordedUsers)
				response := decode(request("GET", path+"?keyword=status-&page_size=50", tokens[1], ""))
				require.Equal(t, true, response["success"])
				found := false
				for _, item := range response["data"].(map[string]any)["items"].([]any) {
					row := item.(map[string]any)
					if row["user_id"] == float64(buyers[0].Id) {
						found = true
						assert.Equal(t, float64(0), row["updated_at"])
					}
				}
				require.True(t, found)
				updatePath := fmt.Sprintf("%s/%d", path, buyers[0].Id)
				body := `{"banned":true,"phone_required":true,"kyc_required":true}`
				assert.Equal(t, http.StatusUnauthorized, request("PUT", updatePath, "", body).Code)
				assert.Equal(t, http.StatusForbidden, request("PUT", updatePath, tokens[0], body).Code)
				for _, invalid := range []string{`{}`, `{"banned":true}`, `{"banned":"yes","phone_required":false,"kyc_required":false}`} {
					assert.Equal(t, false, decode(request("PUT", updatePath, tokens[1], invalid))["success"])
				}
				assert.Equal(t, false, decode(request("PUT", fmt.Sprintf("%s/%d", path, users[1].Id), tokens[1], body))["success"])
				for range 2 {
					require.Equal(t, true, decode(request("PUT", updatePath, tokens[1], body))["success"])
				}
				after = snapshot()
				assert.Equal(t, before.RecordedUsers+1, after.RecordedUsers)
				assert.Equal(t, before.AllIssuesUsers+1, after.AllIssuesUsers)
				assert.Equal(t, before.HealthyUsers, after.HealthyUsers)
				assert.Equal(t, before.BannedUsers+1, after.BannedUsers)
				assert.Equal(t, before.PhoneRequiredUsers+1, after.PhoneRequiredUsers)
				assert.Equal(t, before.KYCRequiredUsers+1, after.KYCRequiredUsers)
				require.Positive(t, after.UpdatedAt)
				for range 2 {
					require.NoError(t, db.AutoMigrate(&model.GuardianAccountStatus{}))
				}
				var stored model.GuardianAccountStatus
				require.NoError(t, db.First(&stored, "user_id = ?", buyers[0].Id).Error)
				assert.True(t, stored.Banned)
				assert.True(t, stored.PhoneRequired)
				assert.True(t, stored.KYCRequired)
				assert.Equal(t, users[1].Id, stored.UpdatedBy)

				for _, partial := range []string{
					`{"banned":true,"phone_required":false,"kyc_required":false}`,
					`{"banned":true,"phone_required":true,"kyc_required":false}`,
				} {
					require.Equal(t, true, decode(request("PUT", updatePath, tokens[1], partial))["success"])
					partialStats := snapshot()
					assert.Equal(t, before.AllIssuesUsers, partialStats.AllIssuesUsers)
					assert.Equal(t, before.HealthyUsers, partialStats.HealthyUsers)
					assert.Equal(t, before.RecordedUsers+1, partialStats.RecordedUsers)
				}
				cleared := `{"banned":false,"phone_required":false,"kyc_required":false}`
				require.Equal(t, true, decode(request("PUT", updatePath, tokens[1], cleared))["success"])
				after = snapshot()
				assert.Equal(t, before.RecordedUsers+1, after.RecordedUsers)
				assert.Equal(t, before.HealthyUsers+1, after.HealthyUsers)
				assert.Equal(t, before.AllIssuesUsers, after.AllIssuesUsers)
				assert.Equal(t, before.BannedUsers, after.BannedUsers)
				assert.Equal(t, before.PhoneRequiredUsers, after.PhoneRequiredUsers)
				assert.Equal(t, before.KYCRequiredUsers, after.KYCRequiredUsers)
				require.NoError(t, db.Model(&model.GuardianServiceOrder{}).Where("user_id = ?", buyers[0].Id).Update("status", "refunded").Error)
				after = snapshot()
				assert.Equal(t, before.PurchasedUsers+1, after.PurchasedUsers)
				assert.Equal(t, before.RecordedUsers, after.RecordedUsers)
				assert.Equal(t, false, decode(request("PUT", updatePath, tokens[1], body))["success"])

				beforeStatus := snapshot()
				buyerPath := fmt.Sprintf("%s/%d", path, buyers[1].Id)
				require.Equal(t, true, decode(request("PUT", buyerPath, tokens[1], cleared))["success"])
				assert.Equal(t, beforeStatus.PurchasedUsers, snapshot().PurchasedUsers)
				assert.Equal(t, beforeStatus.HealthyUsers+1, snapshot().HealthyUsers)
				repeatSnapshot := snapshot()
				repeatOrder := model.GuardianServiceOrder{TradeNo: "repeat-" + suffix, ServiceID: id, UserID: buyers[1].Id, Status: "paid"}
				require.NoError(t, db.Create(&repeatOrder).Error)
				assert.Equal(t, repeatSnapshot, snapshot())
				require.NoError(t, db.Model(&model.GuardianServiceOrder{}).Where("user_id = ?", buyers[1].Id).Update("status", "refunded").Error)
				assert.Equal(t, repeatSnapshot.PurchasedUsers-1, snapshot().PurchasedUsers)
				remaining, _, err := model.ListGuardianAccountStatuses(1, 50, buyers[1].Username)
				require.NoError(t, err)
				assert.Empty(t, remaining)
				assert.Equal(t, false, decode(request("PUT", buyerPath, tokens[1], cleared))["success"])
				if os.Getenv("TEST_GUARDIAN_LINKS_UPGRADE") == "1" {
					var stored model.GuardianAccountStatus
					require.NoError(t, db.First(&stored, "user_id = ?", users[2].Id).Error)
					assert.True(t, stored.Banned)
					assert.EqualValues(t, 123, stored.UpdatedAt)
					remaining, _, err = model.ListGuardianAccountStatuses(1, 50, users[2].Username)
					require.NoError(t, err)
					assert.Empty(t, remaining)
					assert.Equal(t, false, decode(request("PUT", fmt.Sprintf("%s/%d", path, users[2].Id), tokens[1], cleared))["success"])
					beforePurchase := snapshot()
					legacyPurchase := model.GuardianServiceOrder{TradeNo: "legacy-status-" + suffix, ServiceID: id, UserID: users[2].Id, Status: "paid"}
					require.NoError(t, db.Create(&legacyPurchase).Error)
					assert.Equal(t, beforePurchase.PurchasedUsers+1, snapshot().PurchasedUsers)
					assert.Equal(t, beforePurchase.BannedUsers+1, snapshot().BannedUsers)
					require.NoError(t, db.Delete(&legacyPurchase).Error)
				}
			})
			orderBody := fmt.Sprintf(`{"accept_after_sales_terms":true,"after_sales_terms":"test terms","discovery_channel":"github","service_id":%d,"payment_method":"alipay","route":"vps","os":"linux","contact_type":"email","contact":" reader@example.test ","note":"部署"}`, id)
			payment.ComplianceConfirmed = false
			assert.Equal(t, false, decode(request("GET", "/api/guardian/services", "", ""))["data"].(map[string]any)["purchase_enabled"])
			assert.Equal(t, false, decode(request("POST", "/api/guardian/orders", tokens[0], orderBody))["success"])
			payment.ComplianceConfirmed = true
			for _, invalid := range []string{strings.Replace(orderBody, `"discovery_channel":"github",`, "", 1), strings.Replace(orderBody, `"discovery_channel":"github"`, `"discovery_channel":"unknown"`, 1), strings.Replace(orderBody, "alipay", "stripe", 1), strings.Replace(orderBody, "alipay", "waffo_pancake", 1), strings.Replace(orderBody, "alipay", "missing", 1), strings.Replace(orderBody, `"os":"linux"`, `"os":"unknown"`, 1), strings.Replace(orderBody, "reader@example.test", " ", 1)} {
				assert.Equal(t, false, decode(request("POST", "/api/guardian/orders", tokens[0], invalid))["success"])
			}
			require.NoError(t, db.Model(&model.GuardianService{}).Where("id = ?", id).Update("enabled", false).Error)
			assert.Equal(t, false, decode(request("POST", "/api/guardian/orders", tokens[0], orderBody))["success"])
			require.NoError(t, db.Model(&model.GuardianService{}).Where("id = ?", id).Updates(map[string]any{"enabled": true, "price_cents": 0}).Error)
			assert.Equal(t, false, decode(request("POST", "/api/guardian/orders", tokens[0], orderBody))["success"])
			require.NoError(t, db.Model(&model.GuardianService{}).Where("id = ?", id).Update("price_cents", 59901).Error)
			checkout := decode(request("POST", "/api/guardian/orders", tokens[0], orderBody))
			require.Equal(t, true, checkout["success"])
			pay := checkout["data"].(map[string]any)
			trade := pay["trade_no"].(string)
			assert.Equal(t, "https://payment.example.test/submit.php", pay["pay_url"])
			params := pay["pay_params"].(map[string]any)
			assert.Equal(t, "599.01", params["money"])
			assert.Equal(t, "SVC:VPS 部署", params["name"])
			assert.Equal(t, "https://guardian.example.test/api/guardian/epay/notify", params["notify_url"])
			for range 2 {
				require.Equal(t, true, decode(request("POST", "/api/guardian/orders", tokens[0], orderBody))["success"])
			}
			assert.Equal(t, false, decode(request("POST", "/api/guardian/orders", tokens[0], orderBody))["success"])
			var order model.GuardianServiceOrder
			require.NoError(t, db.Where("trade_no = ?", trade).First(&order).Error)

			contactPath := fmt.Sprintf("/api/guardian/orders/%d/contact", order.ID)
			historyPath := fmt.Sprintf("/api/guardian/admin/orders/%d/history", order.ID)
			contactBody := `{"contact_type":"wechat","contact":"new-contact","note":"new note"}`
			assert.Equal(t, http.StatusUnauthorized, request("PUT", contactPath, "", contactBody).Code)
			assert.Equal(t, http.StatusForbidden, request("PUT", contactPath, pat, contactBody).Code)
			assert.Equal(t, false, decode(request("PUT", contactPath, tokens[2], contactBody))["success"])
			assert.Equal(t, false, decode(request("PUT", contactPath, tokens[0], `{"contact_type":"email","contact":""}`))["success"])
			assert.Equal(t, true, decode(request("PUT", contactPath, tokens[0], contactBody))["success"])
			assert.Equal(t, http.StatusForbidden, request("GET", historyPath, tokens[0], "").Code)
			assert.Equal(t, http.StatusForbidden, request("GET", historyPath, pat, "").Code)
			history := decode(request("GET", historyPath, tokens[1], ""))["data"].(map[string]any)
			change := history["items"].([]any)[0].(map[string]any)
			assert.Equal(t, float64(users[0].Id), change["actor_id"])
			assert.Equal(t, "reader@example.test", change["before"].(map[string]any)["contact"])
			assert.Equal(t, "new-contact", change["after"].(map[string]any)["contact"])
			// A history write failure must roll back the contact update as well.
			require.NoError(t, db.Callback().Create().Before("gorm:create").Register("guardian_history_failure", func(tx *gorm.DB) {
				if tx.Statement.Table == "guardian_order_events" {
					tx.AddError(fmt.Errorf("history fixture failure"))
				}
			}))
			_, updateErr := model.UpdateGuardianContact(users[0].Id, order.ID, model.GuardianContactUpdate{ContactType: "email", Contact: "must-rollback", Note: "must-rollback"})
			require.NoError(t, db.Callback().Create().Remove("guardian_history_failure"))
			require.Error(t, updateErr)
			require.NoError(t, db.First(&order, order.ID).Error)
			assert.Equal(t, "new-contact", order.Contact)
			assert.Equal(t, "new note", order.Note)
			// Restore the buyer contact so existing keyword searches still apply.
			assert.Equal(t, true, decode(request("PUT", contactPath, tokens[0], `{"contact_type":"email","contact":"reader@example.test","note":"部署"}`))["success"])
			// The buyer reopens the cashier for the same trade number and amount; other users cannot touch the order.
			payPath, cancelPath := fmt.Sprintf("/api/guardian/orders/%d/pay", order.ID), fmt.Sprintf("/api/guardian/orders/%d/cancel", order.ID)
			resumed := decode(request("POST", payPath, tokens[0], ""))
			require.Equal(t, true, resumed["success"])
			assert.Equal(t, trade, resumed["data"].(map[string]any)["trade_no"])
			assert.Equal(t, "599.01", resumed["data"].(map[string]any)["pay_params"].(map[string]any)["money"])
			assert.Equal(t, false, decode(request("POST", payPath, tokens[2], ""))["success"])
			assert.Equal(t, false, decode(request("POST", cancelPath, tokens[2], ""))["success"])
			// Cancelling frees a pending slot and closes the cashier for that order.
			var withdrawn model.GuardianServiceOrder
			require.NoError(t, db.Where("user_id = ? AND trade_no <> ?", users[0].Id, trade).First(&withdrawn).Error)
			cancelled := decode(request("POST", fmt.Sprintf("/api/guardian/orders/%d/cancel", withdrawn.ID), tokens[0], ""))
			require.Equal(t, true, cancelled["success"])
			assert.Equal(t, "cancelled", cancelled["data"].(map[string]any)["status"])
			assert.Equal(t, false, decode(request("POST", fmt.Sprintf("/api/guardian/orders/%d/pay", withdrawn.ID), tokens[0], ""))["success"])
			assert.Equal(t, false, decode(request("POST", fmt.Sprintf("/api/guardian/orders/%d/cancel", withdrawn.ID), tokens[0], ""))["success"])
			assert.Equal(t, true, decode(request("POST", "/api/guardian/orders", tokens[0], orderBody))["success"])
			// Signed notification transport, amount verification, and replay all use actual handlers.
			callback := func(method, path, money, status, tradeNo string, tamper bool) *httptest.ResponseRecorder {
				params := epay.GenerateParams(map[string]string{"pid": "123", "type": "alipay", "out_trade_no": tradeNo, "trade_no": "provider-1", "money": money, "trade_status": status, "sign_type": "MD5"}, operation_setting.EpayKey)
				if tamper {
					params["money"] = "0.01"
				}
				values := url.Values{}
				for k, v := range params {
					values.Set(k, v)
				}
				body := ""
				if method == "GET" {
					path += "?" + values.Encode()
				} else {
					body = values.Encode()
				}
				req := httptest.NewRequest(method, path, strings.NewReader(body))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				response := httptest.NewRecorder()
				engine.ServeHTTP(response, req)
				return response
			}
			assert.Equal(t, "fail", callback("GET", "/api/guardian/epay/notify", "599.01", epay.StatusTradeSuccess, trade, true).Body.String())
			var anomalies int64
			require.NoError(t, db.Model(&model.GuardianPaymentAnomaly{}).Where("order_id = ?", order.ID).Count(&anomalies).Error)
			assert.Zero(t, anomalies)
			assert.Equal(t, "fail", callback("POST", "/api/guardian/epay/notify", "599.02", epay.StatusTradeSuccess, trade, false).Body.String())
			assert.Equal(t, "fail", callback("POST", "/api/guardian/epay/notify", "599.02", epay.StatusTradeSuccess, trade, false).Body.String())
			paymentHistory := decode(request("GET", historyPath+"?kind=payments", tokens[1], ""))["data"].(map[string]any)
			require.Equal(t, float64(1), paymentHistory["total"])
			anomaly := paymentHistory["items"].([]any)[0].(map[string]any)
			assert.Equal(t, float64(2), anomaly["occurrences"])
			assert.Equal(t, float64(59901), anomaly["expected_cents"])
			assert.Equal(t, "599.02", anomaly["paid_money"])
			assert.Positive(t, anomaly["first_seen_at"])
			filtered := decode(request("GET", "/api/guardian/admin/orders?payment_anomaly=true&keyword="+trade, tokens[1], ""))["data"].(map[string]any)
			assert.Equal(t, float64(1), filtered["total"])
			assert.Equal(t, true, filtered["items"].([]any)[0].(map[string]any)["payment_anomaly"])
			assert.Equal(t, "fail", callback("GET", "/api/guardian/epay/notify", "599.01", epay.StatusTradeSuccess, "missing", false).Body.String())
			require.NoError(t, db.First(&order, order.ID).Error)
			assert.Equal(t, "pending_payment", order.Status)
			assert.Contains(t, callback("GET", "/api/guardian/epay/return", "599.01", "WAIT_BUYER_PAY", trade, false).Header().Get("Location"), "pay=pending")
			assert.Contains(t, callback("GET", "/api/guardian/epay/return", "599.01", epay.StatusTradeSuccess, trade, true).Header().Get("Location"), "pay=fail")
			assert.Equal(t, "success", callback("POST", "/api/guardian/epay/notify", "599.01", epay.StatusTradeSuccess, trade, false).Body.String())
			require.NoError(t, db.First(&order, order.ID).Error)
			paidAt := order.PaidAt
			require.Positive(t, paidAt)
			assert.Contains(t, callback("POST", "/api/guardian/epay/return", "599.01", epay.StatusTradeSuccess, trade, false).Header().Get("Location"), "pay=success")
			require.NoError(t, db.First(&order, order.ID).Error)
			assert.Equal(t, paidAt, order.PaidAt)
			assert.Equal(t, false, decode(request("POST", payPath, tokens[0], ""))["success"])
			assert.Equal(t, false, decode(request("POST", cancelPath, tokens[0], ""))["success"])
			own := request("GET", "/api/guardian/orders/self?page_size=100", tokens[0], "")
			assert.NotContains(t, own.Body.String(), "provider_payload")
			assert.NotContains(t, own.Body.String(), "user_id")
			assert.NotContains(t, own.Body.String(), "username")
			assert.Equal(t, float64(0), decode(request("GET", "/api/guardian/orders/self", tokens[2], ""))["data"].(map[string]any)["total"])

			t.Run("my_order_filters", func(t *testing.T) {
				for _, tc := range []struct {
					query string
					total float64
				}{
					{"order_number=" + url.QueryEscape(trade), 1},
					{fmt.Sprintf("order_number=%s&created_after=%d&created_before=%d", trade, order.CreatedAt, order.CreatedAt+1), 1},
					{fmt.Sprintf("order_number=%s&created_before=%d", trade, order.CreatedAt), 0},
					{fmt.Sprintf("order_number=%s&created_after=%d", trade, order.CreatedAt+1), 0},
					{"order_number=missing-order", 0},
				} {
					result := decode(request("GET", "/api/guardian/orders/self?"+tc.query, tokens[0], ""))
					require.Equal(t, true, result["success"])
					assert.Equal(t, tc.total, result["data"].(map[string]any)["total"], tc.query)
				}
				result := decode(request("GET", "/api/guardian/orders/self?order_number="+url.QueryEscape(trade), tokens[2], ""))
				assert.Equal(t, float64(0), result["data"].(map[string]any)["total"])
				for _, query := range []string{"created_after=-1", "created_before=invalid", "created_after=20&created_before=10"} {
					assert.Equal(t, false, decode(request("GET", "/api/guardian/orders/self?"+query, tokens[0], ""))["success"])
				}
			})
			for _, keyword := range []string{trade, users[0].Username, "reader@example.test"} {
				listed := decode(request("GET", "/api/guardian/admin/orders?keyword="+url.QueryEscape(keyword), tokens[1], ""))
				require.Equal(t, true, listed["success"])
				assert.Greater(t, listed["data"].(map[string]any)["total"].(float64), float64(0))
			}
			assert.Equal(t, false, decode(request("PUT", fmt.Sprintf("/api/guardian/admin/orders/%d", order.ID), tokens[1], `{"status":"delivered","admin_note":"skip"}`))["success"])
			for _, status := range []string{"paid", "scheduled", "in_progress", "delivered", "refunded"} {
				changed := decode(request("PUT", fmt.Sprintf("/api/guardian/admin/orders/%d", order.ID), tokens[1], fmt.Sprintf(`{"status":%q,"admin_note":"进度说明"}`, status)))
				require.Equal(t, true, changed["success"])
				data := changed["data"].(map[string]any)
				assert.Equal(t, users[0].Username, data["username"])
				assert.Equal(t, status, data["status"])
			}
			require.NoError(t, db.First(&order, order.ID).Error)
			require.Positive(t, order.RefundedAt)
			refundAt := order.RefundedAt
			assert.Equal(t, false, decode(request("PUT", contactPath, tokens[0], contactBody))["success"])
			refundUpdated, err := model.UpdateGuardianOrder(order.ID, users[1].Id, "refunded", "refund note")
			require.NoError(t, err)
			assert.Equal(t, refundAt, refundUpdated.RefundedAt)
			history = decode(request("GET", historyPath, tokens[1], ""))["data"].(map[string]any)
			change = history["items"].([]any)[0].(map[string]any)
			assert.Equal(t, float64(users[1].Id), change["actor_id"])
			assert.Equal(t, "进度说明", change["before"].(map[string]any)["admin_note"])
			assert.Equal(t, "refund note", change["after"].(map[string]any)["admin_note"])
			for range 2 {
				require.NoError(t, db.AutoMigrate(&model.GuardianAccountStatus{}, &model.GuardianService{}, &model.GuardianServiceOrder{}, &model.GuardianOrderEvent{}, &model.GuardianPaymentAnomaly{}, &model.GuardianReseller{}, &model.GuardianResellerEvent{}, &model.GuardianResellerOperation{}, &model.GuardianResellerSetting{}, &model.GuardianResellerOffer{}, &model.GuardianTierSupply{}, &model.GuardianReceipt{}, &model.GuardianResellerSale{}, &model.GuardianResellerWithdrawal{}, &model.GuardianCustomerService{}))
			}
			var persistedAnomaly model.GuardianPaymentAnomaly
			require.NoError(t, db.Where("order_id = ?", order.ID).First(&persistedAnomaly).Error)
			assert.Equal(t, 2, persistedAnomaly.Occurrences)
			var persistedEvent model.GuardianOrderEvent
			require.NoError(t, db.Where("order_id = ?", order.ID).Order("id DESC").First(&persistedEvent).Error)
			assert.Equal(t, "refund note", persistedEvent.After.AdminNote)
			for _, state := range []string{"pending_payment", "paid", "scheduled", "in_progress", "delivered", "refunded", "expired", "cancelled"} {
				require.NoError(t, db.Model(&order).Update("status", state).Error)
				result := decode(request("PUT", contactPath, tokens[0], contactBody))
				expected := state == "pending_payment" || state == "paid" || state == "scheduled" || state == "in_progress"
				assert.Equal(t, expected, result["success"], state)
			}
			allStatuses := []string{"pending_payment", "paid", "scheduled", "in_progress", "delivered", "expired", "cancelled", "refunded"}
			allowed := map[string][]string{"pending_payment": {"cancelled"}, "paid": {"scheduled", "in_progress", "refunded"}, "scheduled": {"in_progress", "refunded"}, "in_progress": {"delivered", "refunded"}, "delivered": {"refunded"}}
			for _, from := range allStatuses {
				for _, to := range allStatuses {
					require.NoError(t, db.Model(&order).Update("status", from).Error)
					_, err := model.UpdateGuardianOrder(order.ID, users[1].Id, to, "状态验证")
					ok := from == to
					for _, next := range allowed[from] {
						ok = ok || next == to
					}
					if ok {
						assert.NoError(t, err, from+" -> "+to)
					} else {
						assert.Error(t, err, from+" -> "+to)
					}
				}
			}
			require.NoError(t, db.Model(&model.GuardianServiceOrder{}).Where("user_id = ?", users[0].Id).Updates(map[string]any{"status": "pending_payment", "created_at": time.Now().Unix() - 7201}).Error)
			listed := decode(request("GET", "/api/guardian/orders/self", tokens[0], ""))["data"].(map[string]any)
			for _, item := range listed["items"].([]any) {
				assert.Equal(t, "expired", item.(map[string]any)["status"])
			}
			assert.Equal(t, false, decode(request("POST", payPath, tokens[0], ""))["success"])
			// The cashier outlives expiry and cancellation; a verified late payment must still complete the order.
			var late model.GuardianServiceOrder
			require.NoError(t, db.Where("user_id = ? AND trade_no <> ?", users[0].Id, trade).First(&late).Error)
			require.NoError(t, db.Model(&late).Update("status", "cancelled").Error)
			assert.Equal(t, "success", callback("POST", "/api/guardian/epay/notify", "599.01", epay.StatusTradeSuccess, trade, false).Body.String())
			assert.Contains(t, callback("GET", "/api/guardian/epay/return", "599.01", epay.StatusTradeSuccess, late.TradeNo, false).Header().Get("Location"), "pay=success")
			for _, tradeNo := range []string{trade, late.TradeNo} {
				var paid model.GuardianServiceOrder
				require.NoError(t, db.Where("trade_no = ?", tradeNo).First(&paid).Error)
				assert.Equal(t, "paid", paid.Status, tradeNo)
				assert.Positive(t, paid.PaidAt)
			}
			// Two concurrent requests competing for the third slot cannot both succeed.
			require.Equal(t, true, decode(request("POST", "/api/guardian/orders", tokens[0], orderBody))["success"])
			require.Equal(t, true, decode(request("POST", "/api/guardian/orders", tokens[0], orderBody))["success"])
			var req model.GuardianOrderRequest
			require.NoError(t, common.UnmarshalJsonStr(orderBody, &req))
			errs := make(chan error, 2)
			var wg sync.WaitGroup
			for range 2 {
				wg.Go(func() { _, err := model.CreateGuardianOrder(users[0].Id, req); errs <- err })
			}
			wg.Wait()
			close(errs)
			successes := 0
			for err := range errs {
				if err == nil {
					successes++
				}
			}
			assert.Equal(t, 1, successes)
			var count int64
			require.NoError(t, db.Model(&model.GuardianServiceOrder{}).Where("user_id = ? AND status = ?", users[0].Id, "pending_payment").Count(&count).Error)
			assert.Equal(t, int64(3), count)
			assert.NotContains(t, accessLog.String(), "sign=")
			assert.NotContains(t, accessLog.String(), operation_setting.EpayKey)
			// Repeated migrations preserve rows, zero-valued fields, JSON arrays and unique order numbers.
			for range 2 {
				require.NoError(t, db.AutoMigrate(&model.GuardianAccountStatus{}, &model.GuardianService{}, &model.GuardianServiceOrder{}, &model.GuardianOrderEvent{}, &model.GuardianPaymentAnomaly{}, &model.GuardianReseller{}, &model.GuardianResellerEvent{}, &model.GuardianResellerOperation{}, &model.GuardianResellerSetting{}, &model.GuardianResellerOffer{}, &model.GuardianTierSupply{}, &model.GuardianReceipt{}, &model.GuardianResellerSale{}, &model.GuardianResellerWithdrawal{}, &model.GuardianCustomerService{}))
			}
			require.NoError(t, db.First(&order, order.ID).Error)
			replacement := strings.Replace(payload, `"enabled":true,`, ``, 1)
			replacement = strings.Replace(replacement, `"sort_order":10`, `"sort_order":0`, 1)
			updated := decode(request("PUT", fmt.Sprintf("/api/guardian/admin/services/%d", id), tokens[1], replacement))
			require.Equal(t, true, updated["success"])
			var saved model.GuardianService
			require.NoError(t, db.First(&saved, id).Error)
			assert.False(t, saved.Enabled)
			assert.Zero(t, saved.SortOrder)
			assert.Equal(t, model.GuardianFeatures{"Linux"}, saved.Features)

			// Delivery queue: payment order, placeholders, daily capacity, manual moves and pauses.
			t.Cleanup(func() {
				_ = model.UpdateOptionsBulk(map[string]string{"guardian_queue.daily_capacity": "0", "guardian_queue.extra_slots": "0", "guardian_queue.paused": "false", "guardian_queue.resume_at": "0"})
			})
			require.NoError(t, db.Model(&model.GuardianServiceOrder{}).Where("1 = 1").Update("status", "delivered").Error)
			require.NoError(t, db.Where("1 = 1").Delete(&model.GuardianOrderEvent{}).Error)
			require.NoError(t, db.Model(&model.GuardianService{}).Where("id = ?", id).Update("enabled", true).Error)
			queued := map[string]model.GuardianServiceOrder{}
			for _, buyer := range []struct {
				name  string
				token string
			}{{"first", tokens[0]}, {"other", tokens[2]}, {"second", tokens[0]}} {
				created := decode(request("POST", "/api/guardian/orders", buyer.token, orderBody))
				require.Equal(t, true, created["success"], created["message"])
				tradeNo := created["data"].(map[string]any)["trade_no"].(string)
				require.Equal(t, "success", callback("POST", "/api/guardian/epay/notify", "599.01", epay.StatusTradeSuccess, tradeNo, false).Body.String())
				var paid model.GuardianServiceOrder
				require.NoError(t, db.Where("trade_no = ?", tradeNo).First(&paid).Error)
				queued[buyer.name] = paid
			}
			setQueue := func(body string) map[string]any {
				return decode(request("PUT", "/api/guardian/admin/queue/settings", tokens[1], body))
			}
			myQueue := func(token string) (map[int]map[string]any, map[string]any) {
				data := decode(request("GET", "/api/guardian/orders/self", token, ""))["data"].(map[string]any)
				byID := map[int]map[string]any{}
				for _, item := range data["items"].([]any) {
					row := item.(map[string]any)
					byID[int(row["id"].(float64))] = row
				}
				return byID, data["queue"].(map[string]any)
			}
			ahead := func(token string, order model.GuardianServiceOrder) any {
				rows, _ := myQueue(token)
				return rows[order.ID]["queue_ahead"]
			}
			today, tomorrow := time.Now().Format(time.DateOnly), time.Now().AddDate(0, 0, 1).Format(time.DateOnly)
			assert.Equal(t, float64(0), ahead(tokens[0], queued["first"]))
			assert.Equal(t, float64(1), ahead(tokens[2], queued["other"]))
			assert.Equal(t, float64(2), ahead(tokens[0], queued["second"]))
			rows, _ := myQueue(tokens[2])
			assert.Len(t, rows, 1, "buyers only see their own orders")

			require.Equal(t, true, setQueue(`{"daily_capacity":2,"extra_slots":1,"paused":false,"resume_at":0}`)["success"])
			rows, queueState := myQueue(tokens[0])
			assert.Equal(t, false, queueState["paused"])
			assert.Equal(t, float64(1), rows[queued["first"].ID]["queue_ahead"])
			assert.Equal(t, today, rows[queued["first"].ID]["queue_start_date"])
			assert.Equal(t, float64(3), rows[queued["second"].ID]["queue_ahead"])
			assert.Equal(t, tomorrow, rows[queued["second"].ID]["queue_start_date"])

			move := func(order model.GuardianServiceOrder, position int) map[string]any {
				return decode(request("POST", fmt.Sprintf("/api/guardian/admin/queue/%d/move", order.ID), tokens[1], fmt.Sprintf(`{"position":%d}`, position)))
			}
			require.Equal(t, true, move(queued["second"], 1)["success"])
			assert.Equal(t, float64(1), ahead(tokens[0], queued["second"]))
			assert.Equal(t, float64(2), ahead(tokens[0], queued["first"]))
			assert.Equal(t, float64(3), ahead(tokens[2], queued["other"]))
			adminQueue := decode(request("GET", "/api/guardian/admin/queue", tokens[1], ""))["data"].(map[string]any)
			assert.Len(t, adminQueue["waiting"], 3)
			assert.Equal(t, float64(queued["second"].ID), adminQueue["waiting"].([]any)[0].(map[string]any)["id"])

			// Starting an order counts against today's capacity and moves it out of the waiting list.
			require.Equal(t, true, decode(request("PUT", fmt.Sprintf("/api/guardian/admin/orders/%d", queued["other"].ID), tokens[1], `{"status":"in_progress","admin_note":""}`))["success"])
			assert.Equal(t, false, move(queued["other"], 1)["success"])
			rows, _ = myQueue(tokens[2])
			assert.Nil(t, rows[queued["other"].ID]["queue_ahead"])
			rows, _ = myQueue(tokens[0])
			assert.Equal(t, float64(2), rows[queued["second"].ID]["queue_ahead"])
			assert.Equal(t, tomorrow, rows[queued["second"].ID]["queue_start_date"])

			resumeAt := time.Now().AddDate(0, 0, 3)
			assert.Equal(t, false, setQueue(fmt.Sprintf(`{"daily_capacity":2,"extra_slots":1,"paused":true,"resume_at":%d}`, time.Now().Unix()-60))["success"])
			require.Equal(t, true, setQueue(fmt.Sprintf(`{"daily_capacity":2,"extra_slots":1,"paused":true,"resume_at":%d}`, resumeAt.Unix()))["success"])
			rows, queueState = myQueue(tokens[0])
			assert.Equal(t, true, queueState["paused"])
			assert.Equal(t, resumeAt.Format(time.DateOnly), rows[queued["second"].ID]["queue_start_date"])
			require.Equal(t, true, setQueue(`{"daily_capacity":2,"extra_slots":1,"paused":true,"resume_at":0}`)["success"])
			rows, _ = myQueue(tokens[0])
			assert.Equal(t, "", rows[queued["second"].ID]["queue_start_date"])
			assert.Equal(t, false, setQueue(`{"daily_capacity":-1,"extra_slots":0,"paused":false,"resume_at":0}`)["success"])

			t.Run("reseller money permissions renewals and full refund", func(t *testing.T) {
				adminID, resellerID, buyerID := users[1].Id, users[2].Id, users[0].Id
				require.NoError(t, db.Create(&model.GuardianTierSupply{Tier: 1, ServiceID: id, GuardianSupplyRequest: model.GuardianSupplyRequest{Granted: true, SupplyCents: 10000}}).Error)
				accessPath := fmt.Sprintf("/api/guardian/admin/resellers/%d", resellerID)
				assert.Equal(t, http.StatusForbidden, request("PUT", accessPath, tokens[2], `{"status":"active","reason":"unauthorized"}`).Code)
				assert.Equal(t, http.StatusForbidden, request("PUT", accessPath, pat, `{"status":"active","reason":"token"}`).Code)
				require.Equal(t, true, decode(request("PUT", accessPath, tokens[1], `{"tier":1,"status":"active","reason":"Approved reseller"}`))["success"])
				resellerList := request("GET", "/api/guardian/admin/resellers", tokens[1], "")
				resellerListData := decode(resellerList)
				require.Equal(t, true, resellerListData["success"])
				resellers := resellerListData["data"].([]any)
				require.Len(t, resellers, 1)
				assert.Equal(t, users[2].Username, resellers[0].(map[string]any)["username"])
				assert.EqualValues(t, resellerID, resellers[0].(map[string]any)["user_id"])
				assert.NotContains(t, resellerList.Body.String(), "password")
				assert.Equal(t, http.StatusForbidden, request("GET", "/api/guardian/admin/resellers", tokens[0], "").Code)
				assert.Equal(t, http.StatusForbidden, request("GET", "/api/guardian/admin/resellers", pat, "").Code)
				reseller, err := model.GetGuardianReseller(resellerID)
				require.NoError(t, err)
				require.NotNil(t, reseller)
				require.NoError(t, model.SaveGuardianResellerSetting(adminID, 60))
				product := model.GuardianService{Title: "Reseller Mac", Route: "mac_mini", Enabled: true, PriceCents: 30000, QuarterPriceCents: 90000, AnnualPriceCents: 300000, ClaudeMaxPriceCents: 10000, ClaudeUltraPriceCents: 250000}
				require.NoError(t, db.Create(&product).Error)
				require.NoError(t, model.SaveGuardianSupply(resellerID, product.ID, adminID, model.GuardianSupplyRequest{Granted: true, SupplyCents: 20000, QuarterSupplyCents: 60000, AnnualSupplyCents: 240000, MaxSupplyCents: 7000, UltraSupplyCents: 200000}))
				require.Error(t, model.SaveGuardianRetail(resellerID, product.ID, model.GuardianRetailRequest{Listed: true, RetailCents: 20000}))
				require.NoError(t, model.SaveGuardianRetail(resellerID, product.ID, model.GuardianRetailRequest{Listed: true, RetailCents: 26000, QuarterRetailCents: 78000, AnnualRetailCents: 280000, MaxRetailCents: 8000, UltraRetailCents: 250000}))
				catalogResponse := request("GET", "/api/guardian/resellers/"+reseller.Code, "", "")
				require.Equal(t, true, decode(catalogResponse)["success"])
				assert.NotContains(t, catalogResponse.Body.String(), "supply_cents")
				assert.NotContains(t, catalogResponse.Body.String(), "fee_bps")
				expected := int64(26000)
				req := model.GuardianOrderRequest{AcceptAfterSalesTerms: true, AfterSalesTerms: "test terms", DiscoveryChannel: "github", ServiceID: product.ID, Route: "mac_mini", BillingPeriod: "monthly", PaymentMethod: "alipay", OS: "macos", ContactType: "telegram", Contact: "@buyer-private", Note: "private-note", ResellerCode: reseller.Code, ExpectedAmountCents: &expected, AcceptRefundFee: true}
				_, err = model.CreateGuardianOrder(resellerID, req)
				require.ErrorContains(t, err, "自己")
				t.Run("IP uses platform price without reseller markup", func(t *testing.T) {
					require.NoError(t, db.Model(&product).Updates(map[string]any{"route": "vps", "ip_enabled": true, "ip_options": model.GuardianIPOptions{{ID: "ip", Name: "Static IP", PriceCents: 5000, QuarterPriceCents: 14000, AnnualPriceCents: 50000, Enabled: true}}}).Error)
					t.Cleanup(func() {
						require.NoError(t, db.Model(&product).Updates(map[string]any{"route": "mac_mini", "ip_enabled": false, "ip_options": model.GuardianIPOptions{}}).Error)
					})
					ipReq := req
					amount := int64(76000)
					ipReq.Route = "vps"
					ipReq.BillingPeriod = ""
					ipReq.IPOptionID = "ip"
					ipReq.IPBillingPeriod = "yearly"
					ipReq.ExpectedAmountCents = &amount
					ipOrder, err := model.CreateGuardianOrder(buyerID, ipReq)
					require.NoError(t, err)
					var sale model.GuardianResellerSale
					require.NoError(t, db.First(&sale, ipOrder.ID).Error)
					assert.EqualValues(t, 76000, ipOrder.AmountCents)
					assert.EqualValues(t, 70000, sale.SupplyCents)
					assert.EqualValues(t, 50000, ipOrder.IPOptionPriceCents)
					assert.EqualValues(t, 156, sale.EstimatedFeeCents)
					before, err := model.GetGuardianResellerBalance(resellerID)
					require.NoError(t, err)
					require.NoError(t, db.Model(ipOrder).Updates(map[string]any{"paid_at": time.Now().Unix(), "status": "paid"}).Error)
					pending, err := model.GetGuardianResellerBalance(resellerID)
					require.NoError(t, err)
					assert.EqualValues(t, 5844, pending.PendingCents-before.PendingCents)
					fee := model.GuardianFeeConfirmation{FeeCents: 760, Reference: "ip-channel-statement"}
					require.NoError(t, model.ConfirmGuardianResellerFee(ipOrder.ID, users[1].Id, fee))
					require.NoError(t, model.ConfirmGuardianResellerFee(ipOrder.ID, users[1].Id, fee))
					view, err := model.GetGuardianResellerOrder(ipOrder.ID)
					require.NoError(t, err)
					assert.EqualValues(t, 760, view.ActualFeeCents)
					assert.EqualValues(t, 260, view.SettlementFeeCents)
					confirmed, err := model.GetGuardianResellerBalance(resellerID)
					require.NoError(t, err)
					assert.EqualValues(t, 5740, confirmed.PendingCents-before.PendingCents)

					require.NoError(t, db.Delete(&sale).Error)
					require.NoError(t, db.Where("order_id = ?", ipOrder.ID).Delete(&model.GuardianOrderEvent{}).Error)
					require.NoError(t, db.Delete(ipOrder).Error)
				})
				for _, variant := range []struct {
					period         string
					addon          bool
					amount, supply int64
				}{
					{"quarterly", false, 78000, 60000}, {"yearly", true, 288000, 247000},
				} {
					variantReq := req
					variantReq.BillingPeriod, variantReq.AddClaudeMax, variantReq.ExpectedAmountCents = variant.period, variant.addon, &variant.amount
					variantOrder, err := model.CreateGuardianOrder(buyerID, variantReq)
					require.NoError(t, err)
					var snapshot model.GuardianResellerSale
					require.NoError(t, db.First(&snapshot, variantOrder.ID).Error)
					assert.Equal(t, variant.amount, variantOrder.AmountCents)
					assert.Equal(t, variant.supply, snapshot.SupplyCents)
					assert.Equal(t, "deduct_channel_fee_v1", snapshot.RefundTermsVersion)
					_, err = model.CancelGuardianOrder(buyerID, variantOrder.ID)
					require.NoError(t, err)
				}

				ultraReq := req
				ultraTotal := int64(276000)
				ultraReq.AddClaudeUltra, ultraReq.ExpectedAmountCents = true, &ultraTotal
				ultraOrder, err := model.CreateGuardianOrder(buyerID, ultraReq)
				require.NoError(t, err)
				var ultraSale model.GuardianResellerSale
				require.NoError(t, db.First(&ultraSale, ultraOrder.ID).Error)
				assert.EqualValues(t, 250000, ultraOrder.ClaudeUltraPriceCents)
				assert.Zero(t, ultraOrder.ClaudeMaxPriceCents)
				assert.EqualValues(t, 220000, ultraSale.SupplyCents)
				assert.EqualValues(t, 200000, ultraSale.UltraSupplyCents)
				assert.Zero(t, ultraSale.MaxSupplyCents)
				assert.EqualValues(t, 1656, ultraSale.EstimatedFeeCents)
				require.Error(t, model.SaveGuardianRetail(resellerID, product.ID, model.GuardianRetailRequest{Listed: true, RetailCents: 26000, UltraRetailCents: 200000}))
				require.NoError(t, model.SaveGuardianRetail(resellerID, product.ID, model.GuardianRetailRequest{Listed: true, RetailCents: 26000}))
				_, err = model.CreateGuardianOrder(buyerID, ultraReq)
				require.ErrorContains(t, err, "Ultra")
				require.NoError(t, db.First(&ultraSale, ultraOrder.ID).Error)
				assert.EqualValues(t, 220000, ultraSale.SupplyCents)
				_, err = model.CancelGuardianOrder(buyerID, ultraOrder.ID)
				require.NoError(t, err)
				wrong := req
				wrong.AcceptRefundFee = false
				_, err = model.CreateGuardianOrder(buyerID, wrong)
				require.Error(t, err)
				wrongPrice := int64(1)
				wrong = req
				wrong.ExpectedAmountCents = &wrongPrice
				_, err = model.CreateGuardianOrder(buyerID, wrong)
				require.ErrorContains(t, err, "报价")
				order, err := model.CreateGuardianOrder(buyerID, req)
				require.NoError(t, err)
				require.NoError(t, model.SaveGuardianRetail(resellerID, product.ID, model.GuardianRetailRequest{Listed: true, RetailCents: 27000}))
				require.NoError(t, model.CompleteGuardianOrder(order.TradeNo, "fixture", "260.00", "alipay"))
				require.NoError(t, model.CompleteGuardianOrder(order.TradeNo, "fixture-replay", "260.00", "alipay"))
				var saleCount int64
				require.NoError(t, db.Model(&model.GuardianResellerSale{}).Where("order_id = ?", order.ID).Count(&saleCount).Error)
				assert.EqualValues(t, 1, saleCount)
				financePath := fmt.Sprintf("/api/guardian/admin/orders/%d/reseller", order.ID)
				finance := decode(request("GET", financePath, tokens[1], ""))
				require.Equal(t, true, finance["success"])
				financeData := finance["data"].(map[string]any)
				assert.EqualValues(t, order.ID, financeData["order_id"])
				assert.EqualValues(t, resellerID, financeData["reseller_id"])
				assert.EqualValues(t, 20000, financeData["supply_cents"])
				assert.EqualValues(t, 26000, financeData["amount_cents"])
				assert.Equal(t, http.StatusForbidden, request("GET", financePath, tokens[2], "").Code)
				assert.Equal(t, http.StatusForbidden, request("GET", financePath, pat, "").Code)
				assert.Equal(t, false, decode(request("GET", "/api/guardian/admin/orders/999999/reseller", tokens[1], ""))["success"])
				view := request("GET", "/api/guardian/reseller/orders", tokens[2], "")
				require.Equal(t, true, decode(view)["success"])
				for _, secret := range []string{"buyer-private", "private-note", "admin_note", "contact_type", "contact", "user_id"} {
					assert.NotContains(t, view.Body.String(), secret)
				}
				assert.Equal(t, http.StatusForbidden, request("GET", "/api/guardian/reseller/orders", tokens[0], "").Code)
				_, err = model.UpdateGuardianOrder(order.ID, adminID, "in_progress", "Internal note")
				require.NoError(t, err)
				delivered, err := model.UpdateGuardianOrder(order.ID, adminID, "delivered", "Internal note")
				require.NoError(t, err)
				require.Positive(t, delivered.CustomerServiceID)
				balance, err := model.GetGuardianResellerBalance(resellerID)
				require.NoError(t, err)
				assert.EqualValues(t, 5844, balance.PendingCents)
				assert.Zero(t, balance.AvailableCents)
				missingFee := decode(request("PUT", fmt.Sprintf("/api/guardian/admin/reseller-orders/%d/fee", order.ID), tokens[1], `{"reference":"missing-fee"}`))
				assert.Equal(t, false, missingFee["success"])
				require.NoError(t, model.ConfirmGuardianResellerFee(order.ID, adminID, model.GuardianFeeConfirmation{FeeCents: 156, Reference: "channel-bill"}))
				require.NoError(t, db.Model(order).Update("delivered_at", time.Now().Unix()-8*86400).Error)
				balance, err = model.GetGuardianResellerBalance(resellerID)
				require.NoError(t, err)
				assert.EqualValues(t, 5844, balance.AvailableCents)
				quote, err := model.GetGuardianRenewalQuote(buyerID, delivered.CustomerServiceID)
				require.NoError(t, err)
				assert.EqualValues(t, 27000, quote.Service.PriceCents)
				assert.Equal(t, reseller.Code, quote.ResellerCode)
				_, err = model.GetGuardianRenewalQuote(resellerID, delivered.CustomerServiceID)
				require.Error(t, err)
				renewal := req
				renewal.CustomerServiceID = delivered.CustomerServiceID
				renewal.ResellerCode = "forged-other-code"
				renewPrice := int64(27000)
				renewal.ExpectedAmountCents = &renewPrice
				renewed, err := model.CreateGuardianOrder(buyerID, renewal)
				require.NoError(t, err)
				assert.Equal(t, reseller.Code, renewed.ResellerCode)
				_, err = model.CancelGuardianOrder(buyerID, renewed.ID)
				require.NoError(t, err)

				withdrawal := model.GuardianWithdrawalRequest{AmountCents: 5000, Account: "test-recipient", PayeeName: "Test payee", RequestKey: "reseller-withdrawal-" + suffix}
				require.NoError(t, withdrawal.Validate())
				body, err := common.Marshal(withdrawal)
				require.NoError(t, err)
				assert.NotEqual(t, true, decode(request("POST", "/api/guardian/reseller/withdrawals", tokens[2], string(body)))["success"])
				identity, err := service.ParseAccessToken(tokens[2])
				require.NoError(t, err)
				operation := service.VerificationOperation{Scope: service.VerificationScopeGuardianWithdrawal, Context: body}
				binding, err := service.BindVerificationOperation(operation)
				require.NoError(t, err)
				proof, _, err := service.IssueSecurityProof(identity, service.VerificationMethodPassword, binding)
				require.NoError(t, err)
				sendWithdrawal := func(payload string, proofToken string) *httptest.ResponseRecorder {
					req := httptest.NewRequest("POST", "/api/guardian/reseller/withdrawals", strings.NewReader(payload))
					req.Header.Set("Content-Type", "application/json")
					req.Header.Set("Authorization", "Bearer "+tokens[2])
					req.Header.Set("X-Security-Proof", proofToken)
					response := httptest.NewRecorder()
					engine.ServeHTTP(response, req)
					return response
				}
				changed := withdrawal
				changed.Account = "different-recipient"
				changedBody, err := common.Marshal(changed)
				require.NoError(t, err)
				assert.NotEqual(t, true, decode(sendWithdrawal(string(changedBody), proof))["success"])
				accepted := decode(sendWithdrawal(string(body), proof))
				require.Equal(t, true, accepted["success"], accepted)
				assert.NotEqual(t, true, decode(sendWithdrawal(string(body), proof))["success"], "one-use proof cannot be replayed")
				expiredProof, _, err := service.IssueSecurityProof(identity, service.VerificationMethodPassword, binding)
				require.NoError(t, err)
				require.NoError(t, db.Model(&model.AuthFlow{}).Where("user_id = ? AND purpose = ?", resellerID, model.AuthFlowPurposeSecurityProof).Update("expires_at", time.Now().Add(-time.Second)).Error)
				assert.NotEqual(t, true, decode(sendWithdrawal(string(body), expiredProof))["success"])
				passwordProof, _, err := service.IssueSecurityProof(identity, service.VerificationMethodPassword, binding)
				require.NoError(t, err)
				factor := model.TwoFA{UserId: resellerID, IsEnabled: true}
				require.NoError(t, db.Create(&factor).Error)
				assert.NotEqual(t, true, decode(sendWithdrawal(string(body), passwordProof))["success"], "enrolled MFA must not fall back to password")
				require.NoError(t, db.Delete(&factor).Error)
				wID := int(accepted["data"].(map[string]any)["id"].(float64))
				duplicate, err := model.CreateGuardianWithdrawal(resellerID, withdrawal)
				require.NoError(t, err)
				assert.Equal(t, wID, duplicate.ID)
				overdraw := withdrawal
				overdraw.RequestKey += "-2"
				_, err = model.CreateGuardianWithdrawal(resellerID, overdraw)
				require.Error(t, err)
				require.Error(t, model.UpdateGuardianWithdrawal(wID, buyerID, false, model.GuardianWithdrawalUpdate{Status: "cancelled", Note: "wrong owner"}))
				refund := model.GuardianResellerRefundRequest{Action: "start", Reason: "Customer refund"}
				require.Error(t, model.UpdateGuardianResellerRefund(order.ID, adminID, refund), "reserve must be resolved first")
				require.NoError(t, model.UpdateGuardianWithdrawal(wID, adminID, true, model.GuardianWithdrawalUpdate{Status: "processing", FeeCents: 50, Note: "Verified recipient"}))
				require.Error(t, model.UpdateGuardianWithdrawal(wID, resellerID, false, model.GuardianWithdrawalUpdate{Status: "cancelled", Note: "too late"}))
				require.NoError(t, model.UpdateGuardianWithdrawal(wID, adminID, true, model.GuardianWithdrawalUpdate{Status: "paid", Reference: "transfer-" + suffix, Note: "Sent 49.50"}))
				require.Error(t, model.UpdateGuardianWithdrawal(wID, adminID, true, model.GuardianWithdrawalUpdate{Status: "paid", Reference: "duplicate-" + suffix, Note: "Repeat"}))
				balance, err = model.GetGuardianResellerBalance(resellerID)
				require.NoError(t, err)
				assert.EqualValues(t, 844, balance.AvailableCents)
				assert.EqualValues(t, 5000, balance.PaidCents)
				_, err = model.UpdateGuardianOrder(order.ID, adminID, "refunded", "bypass")
				require.Error(t, err)
				require.NoError(t, model.UpdateGuardianResellerRefund(order.ID, adminID, refund))
				held, err := model.GetGuardianResellerBalance(resellerID)
				require.NoError(t, err)
				assert.EqualValues(t, 5844, held.HeldCents)
				assert.Zero(t, held.AvailableCents)
				assert.Zero(t, held.DebtCents)
				cancelledRefund := refund
				cancelledRefund.Action = "cancel"
				require.NoError(t, model.UpdateGuardianResellerRefund(order.ID, adminID, cancelledRefund))
				restored, err := model.GetGuardianResellerBalance(resellerID)
				require.NoError(t, err)
				assert.EqualValues(t, 844, restored.AvailableCents)
				require.NoError(t, model.UpdateGuardianResellerRefund(order.ID, adminID, refund))
				refund.Action, refund.Reference = "complete", "refund-"+suffix
				require.NoError(t, model.UpdateGuardianResellerRefund(order.ID, adminID, refund))
				require.NoError(t, model.UpdateGuardianResellerRefund(order.ID, adminID, refund))
				var sale model.GuardianResellerSale
				require.NoError(t, db.First(&sale, order.ID).Error)
				assert.EqualValues(t, 25844, sale.RefundCents)
				balance, err = model.GetGuardianResellerBalance(resellerID)
				require.NoError(t, err)
				assert.Zero(t, balance.AvailableCents)
				assert.EqualValues(t, 5000, balance.DebtCents)
				_, err = model.SetGuardianResellerAccess(resellerID, adminID, model.GuardianResellerAccessRequest{Status: "suspended", Reason: "Pause"})
				require.NoError(t, err)
				require.Equal(t, true, decode(request("GET", "/api/guardian/reseller/orders", tokens[2], ""))["success"])
				require.Error(t, model.SaveGuardianRetail(resellerID, product.ID, model.GuardianRetailRequest{Listed: true, RetailCents: 28000}))
				quote, err = model.GetGuardianRenewalQuote(buyerID, delivered.CustomerServiceID)
				require.NoError(t, err)
				assert.True(t, quote.DirectConfirmationRequired)
				directPrice := int64(30000)
				renewal.ExpectedAmountCents = &directPrice
				_, err = model.CreateGuardianOrder(buyerID, renewal)
				require.ErrorContains(t, err, "明确确认")
				renewal.AcceptDirectRenewal = true
				direct, err := model.CreateGuardianOrder(buyerID, renewal)
				require.NoError(t, err)
				assert.Empty(t, direct.ResellerCode)
				require.NoError(t, model.CompleteGuardianOrder(direct.TradeNo, "fixture", "300.00", "alipay"))
				quote, err = model.GetGuardianRenewalQuote(buyerID, delivered.CustomerServiceID)
				require.NoError(t, err)
				assert.Empty(t, quote.ResellerCode)
				assert.False(t, quote.DirectConfirmationRequired)
				for range 2 {
					require.NoError(t, db.AutoMigrate(&model.GuardianReseller{}, &model.GuardianResellerEvent{}, &model.GuardianResellerOperation{}, &model.GuardianResellerSetting{}, &model.GuardianResellerOffer{}, &model.GuardianTierSupply{}, &model.GuardianReceipt{}, &model.GuardianResellerSale{}, &model.GuardianResellerWithdrawal{}, &model.GuardianCustomerService{}))
				}
				var preservedSale model.GuardianResellerSale
				require.NoError(t, db.First(&preservedSale, order.ID).Error)
				assert.Equal(t, sale, preservedSale)
				duplicateOffer := model.GuardianResellerOffer{ResellerID: resellerID, ServiceID: product.ID}
				assert.Error(t, db.Create(&duplicateOffer).Error)
				var original model.GuardianServiceOrder
				require.NoError(t, db.First(&original, order.ID).Error)
				assert.Equal(t, reseller.Code, original.ResellerCode)
				assert.Equal(t, http.StatusNotFound, request("PUT", fmt.Sprintf("/api/guardian/admin/customer-services/%d/reseller", delivered.CustomerServiceID), tokens[1], `{"reseller_id":0,"reason":"Removed feature"}`).Code)

			})
			t.Run("stock availability and Pro snapshots", func(t *testing.T) {
				buyer := model.User{Username: "stock-" + suffix, Password: "unused", AffCode: "stk" + suffix}
				require.NoError(t, db.Create(&buyer).Error)
				product := model.GuardianService{Title: "Stock Mac", Route: "mac_mini", Enabled: true, PriceCents: 30000, ClaudeProPriceCents: 12000, ClaudeMaxPriceCents: 90000, ClaudeUltraPriceCents: 200000}
				require.NoError(t, product.Validate())
				require.NoError(t, db.Create(&product).Error)
				req := model.GuardianOrderRequest{AcceptAfterSalesTerms: true, AfterSalesTerms: "test terms", ServiceID: product.ID, DiscoveryChannel: "github", Route: "mac_mini", OS: "macos", ContactType: "email", Contact: "stock@example.test", PaymentMethod: "alipay", AddClaudePro: true}
				for _, pair := range []model.GuardianOrderRequest{{AddClaudePro: true, AddClaudeMax: true}, {AddClaudePro: true, AddClaudeUltra: true}, {AddClaudeMax: true, AddClaudeUltra: true}} {
					invalid := req
					invalid.AddClaudePro, invalid.AddClaudeMax, invalid.AddClaudeUltra = pair.AddClaudePro, pair.AddClaudeMax, pair.AddClaudeUltra
					_, err := model.CreateGuardianOrder(buyer.Id, invalid)
					require.ErrorContains(t, err, "只能选择一项")
				}
				for _, stock := range []string{"out_of_stock", "claude_pro_out_of_stock"} {
					require.NoError(t, db.Model(&product).Update(stock, true).Error)
					_, err := model.CreateGuardianOrder(buyer.Id, req)
					require.ErrorContains(t, err, "缺货")
					require.NoError(t, db.Model(&product).Update(stock, false).Error)
				}
				require.NoError(t, db.Model(&product).Updates(map[string]any{"claude_max_out_of_stock": true, "claude_ultra_out_of_stock": true}).Error)
				for _, tier := range []string{"max", "ultra"} {
					unavailable := req
					unavailable.AddClaudePro = false
					unavailable.AddClaudeMax, unavailable.AddClaudeUltra = tier == "max", tier == "ultra"
					_, err := model.CreateGuardianOrder(buyer.Id, unavailable)
					require.ErrorContains(t, err, "缺货")
				}
				order, err := model.CreateGuardianOrder(buyer.Id, req)
				require.NoError(t, err)
				assert.EqualValues(t, 42000, order.AmountCents)
				assert.EqualValues(t, 12000, order.ClaudeProPriceCents)
				assert.Zero(t, order.ClaudeMaxPriceCents)
				assert.Zero(t, order.ClaudeUltraPriceCents)
				require.NoError(t, db.Model(&product).Update("out_of_stock", true).Error)
				require.NoError(t, model.CompleteGuardianOrder(order.TradeNo, "fixture", "420.00", "alipay"))
				receipt, err := model.GetGuardianReceipt(order.ID, buyer.Id, false)
				require.NoError(t, err)
				assert.EqualValues(t, 12000, receipt.ProCents)
				renewal := model.GuardianCustomerService{UserID: buyer.Id, ServiceID: product.ID, InitialOrderID: order.ID, Title: product.Title}
				require.NoError(t, db.Create(&renewal).Error)
				req.AddClaudePro = false
				req.CustomerServiceID, req.ExpectedAmountCents = renewal.ID, &product.PriceCents
				renewed, err := model.CreateGuardianOrder(buyer.Id, req)
				require.NoError(t, err)
				assert.EqualValues(t, 30000, renewed.AmountCents)
				req.CustomerServiceID = 0
				_, err = model.CreateGuardianOrder(buyer.Id, req)
				require.ErrorContains(t, err, "缺货")
				resellerUser := model.User{Username: "stock-reseller-" + suffix, Password: "unused", AffCode: "str" + suffix}
				require.NoError(t, db.Create(&resellerUser).Error)
				reseller := model.GuardianReseller{UserID: resellerUser.Id, Code: "stock-" + suffix, Status: "active", Tier: 2}
				require.NoError(t, db.Create(&reseller).Error)
				require.NoError(t, model.SaveGuardianSupply(reseller.UserID, product.ID, users[1].Id, model.GuardianSupplyRequest{Granted: true, SupplyCents: 20000, ProSupplyCents: 8000}))
				require.NoError(t, model.SaveGuardianRetail(reseller.UserID, product.ID, model.GuardianRetailRequest{Listed: true, RetailCents: 28000, ProRetailCents: 11000}))
				products, err := model.GuardianResellerCatalog(reseller.Code)
				require.NoError(t, err)
				require.Len(t, products, 1)
				assert.True(t, products[0].OutOfStock)
				assert.True(t, products[0].ClaudeMaxOutOfStock)
				assert.EqualValues(t, 11000, products[0].ClaudeProPriceCents)
				req.AddClaudePro, req.ResellerCode, req.AcceptRefundFee = true, reseller.Code, true
				total := int64(39000)
				req.ExpectedAmountCents = &total
				_, err = model.CreateGuardianOrder(buyer.Id, req)
				require.ErrorContains(t, err, "缺货")
				require.NoError(t, db.Model(&product).Update("out_of_stock", false).Error)
				sold, err := model.CreateGuardianOrder(buyer.Id, req)
				require.NoError(t, err)
				sale, err := model.GetGuardianResellerOrder(sold.ID)
				require.NoError(t, err)
				assert.EqualValues(t, 28000, sale.SupplyCents)
				assert.EqualValues(t, 8000, sale.ProSupplyCents)
				assert.EqualValues(t, 11000, sold.ClaudeProPriceCents)
				change := model.GuardianTierChange{Tier: 2, Prices: []model.GuardianTierSupply{{ServiceID: product.ID, GuardianSupplyRequest: model.GuardianSupplyRequest{Granted: true, SupplyCents: 19000, ProSupplyCents: 7000}}}}
				preview, err := model.ChangeGuardianTier(users[1].Id, change)
				require.NoError(t, err)
				change.PreviewToken = preview.Token
				_, err = model.ChangeGuardianTier(users[1].Id, change)
				require.NoError(t, err)
				restore := model.GuardianTierChange{Tier: 2, ResellerIDs: []int{reseller.UserID}, ServiceIDs: []int{product.ID}, Restore: true}
				preview, err = model.ChangeGuardianTier(users[1].Id, restore)
				require.NoError(t, err)
				restore.PreviewToken = preview.Token
				_, err = model.ChangeGuardianTier(users[1].Id, restore)
				require.NoError(t, err)
				offers, err := model.ListGuardianResellerOffers(reseller.UserID)
				require.NoError(t, err)
				require.Len(t, offers, 1)
				assert.EqualValues(t, 7000, offers[0].ProSupplyCents)
				require.NoError(t, db.Delete(&reseller).Error)
			})
			t.Run("invoice availability and freeform information", func(t *testing.T) {
				buyer := model.User{Username: "invoice-" + suffix, Password: "unused", AffCode: "inv" + suffix}
				require.NoError(t, db.Create(&buyer).Error)
				product := model.GuardianService{Title: "Invoice product", Route: "vps", Enabled: true, PriceCents: 10000}
				require.NoError(t, db.Create(&product).Error)
				req := model.GuardianOrderRequest{AcceptAfterSalesTerms: true, AfterSalesTerms: "test terms", ServiceID: product.ID, DiscoveryChannel: "github", Route: "vps", OS: "linux", ContactType: "email", Contact: "invoice@example.test", PaymentMethod: "alipay", InvoiceInformation: "  公司名称\n税号及其他信息  "}
				_, err := model.CreateGuardianOrder(buyer.Id, req)
				require.ErrorContains(t, err, "不支持开票")
				require.NoError(t, db.Model(&product).Update("invoice_enabled", true).Error)
				order, err := model.CreateGuardianOrder(buyer.Id, req)
				require.NoError(t, err)
				assert.Equal(t, "公司名称\n税号及其他信息", order.InvoiceInformation)
				require.NoError(t, db.Model(&product).Update("invoice_enabled", false).Error)
				for range 2 {
					require.NoError(t, db.AutoMigrate(&model.GuardianService{}, &model.GuardianServiceOrder{}))
				}
				var stored model.GuardianServiceOrder
				require.NoError(t, db.First(&stored, order.ID).Error)
				assert.Equal(t, order.InvoiceInformation, stored.InvoiceInformation)
				req.InvoiceInformation = ""
				_, err = model.CreateGuardianOrder(buyer.Id, req)
				require.NoError(t, err)
				req.InvoiceInformation = strings.Repeat("字", 2001)
				_, err = model.CreateGuardianOrder(buyer.Id, req)
				require.ErrorContains(t, err, "2000")
			})
			t.Run("payment receipts remain separate from payment and enforce ownership", func(t *testing.T) {
				buyerID := users[0].Id
				require.NoError(t, db.Model(&model.User{}).Where("id = ?", buyerID).Update("email", "buyer@example.test").Error)
				receiptOrder := model.GuardianServiceOrder{TradeNo: "receipt-" + suffix, UserID: buyerID, ServiceID: id, ServiceTitle: "Original product", AmountCents: 12345, Status: "pending_payment", PaymentMethod: "alipay", CreatedAt: time.Now().Unix(), ReceiptNotification: true, DiscoveryChannel: "github"}
				require.NoError(t, db.Create(&receiptOrder).Error)
				_, err := model.GetGuardianReceipt(receiptOrder.ID, buyerID, false)
				require.Error(t, err)
				require.NoError(t, model.CompleteGuardianOrder(receiptOrder.TradeNo, "fixture", "123.45", "alipay"))
				require.NoError(t, model.CompleteGuardianOrder(receiptOrder.TradeNo, "fixture", "123.45", "alipay"))
				view, err := model.GetGuardianReceipt(receiptOrder.ID, buyerID, false)
				require.NoError(t, err)
				assert.EqualValues(t, 12345, view.AmountCents)
				assert.Equal(t, "b***@example.test", view.Recipient)
				assert.Equal(t, "pending", view.EmailStatus)
				contextResponse := decode(request("GET", "/api/guardian/checkout-context", tokens[0], ""))
				assert.Equal(t, "github", contextResponse["data"].(map[string]any)["discovery_channel"])
				filtered, _, err := model.ListAdminGuardianOrders("", "", 1, 50, false, "github")
				require.NoError(t, err)
				for _, row := range filtered {
					assert.Equal(t, "github", row.DiscoveryChannel)
				}
				require.NoError(t, db.Model(&model.GuardianService{}).Where("id = ?", id).Update("title", "Changed catalog title").Error)
				again, err := model.GetGuardianReceipt(receiptOrder.ID, buyerID, false)
				require.NoError(t, err)
				assert.Equal(t, view.Number, again.Number)
				assert.Equal(t, "Original product", again.ServiceTitle)
				endpoint := fmt.Sprintf("/api/guardian/orders/%d/receipt", receiptOrder.ID)
				assert.Equal(t, false, decode(request("GET", endpoint, tokens[2], ""))["success"])
				assert.Equal(t, http.StatusForbidden, request("GET", endpoint, pat, "").Code)
				response := request("GET", endpoint, tokens[0], "")
				assert.Equal(t, true, decode(response)["success"])
				assert.NotContains(t, response.Body.String(), "buyer@example.test")
				assert.Equal(t, true, decode(request("GET", fmt.Sprintf("/api/guardian/admin/orders/%d/receipt", receiptOrder.ID), tokens[1], ""))["success"])
				require.Error(t, model.RequestGuardianReceiptEmail(receiptOrder.ID, buyerID, false))
				require.NoError(t, db.Model(&model.GuardianReceipt{}).Where("order_id = ?", receiptOrder.ID).Updates(map[string]any{"email_status": "failed", "requested_at": time.Now().Unix() - 61}).Error)
				require.NoError(t, model.RequestGuardianReceiptEmail(receiptOrder.ID, buyerID, false))
				claimed, err := model.ClaimGuardianReceipt(time.Now().Unix())
				require.NoError(t, err)
				require.NotNil(t, claimed)
				message, err := service.BuildGuardianReceiptMessage(*claimed)
				require.NoError(t, err)
				assert.Contains(t, message.HTMLBody, "/guardian/receipt?order=")
				assert.NotContains(t, message.HTMLBody, "supply")
				require.NoError(t, model.FinishGuardianReceiptEmail(*claimed, "pending"))
				var retry model.GuardianReceipt
				require.NoError(t, db.First(&retry, "order_id = ?", claimed.OrderID).Error)
				assert.Equal(t, "pending", retry.EmailStatus)
				assert.Greater(t, retry.NextAttemptAt, time.Now().Unix())
				for attempt := 2; attempt <= 4; attempt++ {
					require.NoError(t, db.Model(&model.GuardianReceipt{}).Where("order_id = ?", claimed.OrderID).Update("next_attempt_at", 0).Error)
					claimed, err = model.ClaimGuardianReceipt(time.Now().Unix())
					require.NoError(t, err)
					require.NotNil(t, claimed)
					require.NoError(t, model.FinishGuardianReceiptEmail(*claimed, "pending"))
				}
				require.NoError(t, db.First(&retry, "order_id = ?", claimed.OrderID).Error)
				assert.Equal(t, "failed", retry.EmailStatus)
				require.NoError(t, db.Model(&model.User{}).Where("id = ?", buyerID).Update("email", "").Error)
				require.ErrorContains(t, model.RequestGuardianReceiptEmail(receiptOrder.ID, buyerID, false), "稍后")
				require.NoError(t, db.Model(&model.GuardianReceipt{}).Where("order_id = ?", receiptOrder.ID).Update("requested_at", time.Now().Unix()-61).Error)
				require.ErrorContains(t, model.RequestGuardianReceiptEmail(receiptOrder.ID, buyerID, false), "绑定邮箱")
				legacy := receiptOrder
				legacy.ID = 0
				legacy.TradeNo = "legacy-receipt-" + suffix
				legacy.PaidAt = time.Now().Unix() - 86400
				legacy.Status = "paid"
				legacy.ReceiptNotification = false
				require.NoError(t, db.Create(&legacy).Error)
				require.NoError(t, model.PrepareGuardianReceipts())
				var count int64
				require.NoError(t, db.Model(&model.GuardianReceipt{}).Where("order_id = ?", legacy.ID).Count(&count).Error)
				assert.Zero(t, count)
				legacyView, err := model.GetGuardianReceipt(legacy.ID, buyerID, false)
				require.NoError(t, err)
				assert.Equal(t, "not_requested", legacyView.EmailStatus)
				require.NoError(t, db.Model(&model.GuardianServiceOrder{}).Where("id = ?", receiptOrder.ID).Updates(map[string]any{"status": "refunded", "refunded_at": time.Now().Unix()}).Error)
				refunded, err := model.GetGuardianReceipt(receiptOrder.ID, buyerID, false)
				require.NoError(t, err)
				assert.EqualValues(t, 12345, refunded.AmountCents)
				assert.Equal(t, "refunded", refunded.RefundState)
				assert.Nil(t, refunded.RefundCents)
			})

			t.Run("tier supply preview inheritance overrides and historical snapshots", func(t *testing.T) {
				resellerID, adminID := users[2].Id, users[1].Id
				_, err := model.SetGuardianResellerAccess(resellerID, adminID, model.GuardianResellerAccessRequest{Status: "active", Reason: "tier test"})
				require.NoError(t, err)
				product := model.GuardianService{Title: "Tier product", Route: "vps", PriceCents: 20000, Enabled: true}
				require.NoError(t, db.Create(&product).Error)
				change := model.GuardianTierChange{Tier: 1, Prices: []model.GuardianTierSupply{{ServiceID: product.ID, GuardianSupplyRequest: model.GuardianSupplyRequest{Granted: true, SupplyCents: 10000}}}}
				preview, err := model.ChangeGuardianTier(adminID, change)
				require.NoError(t, err)
				var count int64
				require.NoError(t, db.Model(&model.GuardianTierSupply{}).Where("service_id = ?", product.ID).Count(&count).Error)
				assert.Zero(t, count)
				change.PreviewToken = preview.Token
				_, err = model.ChangeGuardianTier(adminID, change)
				require.NoError(t, err)
				require.NoError(t, model.SaveGuardianRetail(resellerID, product.ID, model.GuardianRetailRequest{Listed: true, RetailCents: 15000}))
				reseller, err := model.GetGuardianReseller(resellerID)
				require.NoError(t, err)
				price := int64(15000)
				order, err := model.CreateGuardianOrder(users[0].Id, model.GuardianOrderRequest{AcceptAfterSalesTerms: true, AfterSalesTerms: "test terms", DiscoveryChannel: "qq", ServiceID: product.ID, Route: "vps", OS: "linux", PaymentMethod: "alipay", ContactType: "qq", Contact: "123456", ResellerCode: reseller.Code, ExpectedAmountCents: &price, AcceptRefundFee: true})
				require.NoError(t, err)
				change.PreviewToken = ""
				change.Prices[0].SupplyCents = 16000
				preview, err = model.ChangeGuardianTier(adminID, change)
				require.NoError(t, err)
				assert.Greater(t, preview.Unavailable, 0)
				change.PreviewToken = preview.Token
				_, err = model.ChangeGuardianTier(adminID, change)
				require.NoError(t, err)
				_, err = model.ChangeGuardianTier(adminID, change)
				require.ErrorContains(t, err, "重新预览")
				catalog, err := model.GuardianResellerCatalog(reseller.Code)
				require.NoError(t, err)
				for _, p := range catalog {
					assert.NotEqual(t, product.ID, p.ID)
				}
				var sale model.GuardianResellerSale
				require.NoError(t, db.First(&sale, order.ID).Error)
				assert.EqualValues(t, 10000, sale.SupplyCents)
				require.NoError(t, model.CompleteGuardianOrder(order.TradeNo, "fixture", "150.00", "alipay"))
				require.NoError(t, model.SaveGuardianSupply(resellerID, product.ID, adminID, model.GuardianSupplyRequest{Granted: true, SupplyCents: 9000}))
				offers, err := model.ListGuardianResellerOffers(resellerID)
				require.NoError(t, err)
				for _, o := range offers {
					if o.ServiceID == product.ID {
						assert.False(t, o.FollowTier)
						assert.EqualValues(t, 9000, o.SupplyCents)
					}
				}
				restore := model.GuardianTierChange{Tier: 1, ResellerIDs: []int{resellerID}, ServiceIDs: []int{product.ID}, Restore: true}
				preview, err = model.ChangeGuardianTier(adminID, restore)
				require.NoError(t, err)
				restore.PreviewToken = preview.Token
				_, err = model.ChangeGuardianTier(adminID, restore)
				require.NoError(t, err)
				offers, err = model.ListGuardianResellerOffers(resellerID)
				require.NoError(t, err)
				for _, o := range offers {
					if o.ServiceID == product.ID {
						assert.True(t, o.FollowTier)
						assert.EqualValues(t, 16000, o.SupplyCents)
					}
				}
				for range 2 {
					require.NoError(t, db.AutoMigrate(&model.GuardianReseller{}, &model.GuardianTierSupply{}, &model.GuardianResellerOffer{}, &model.GuardianReceipt{}))
				}
				require.NoError(t, db.First(&sale, order.ID).Error)
				assert.EqualValues(t, 10000, sale.SupplyCents)
				assert.Error(t, db.Create(&model.GuardianTierSupply{Tier: 1, ServiceID: product.ID}).Error)
				_, err = model.GetGuardianReceipt(order.ID, users[0].Id, false)
				require.NoError(t, err)
				assert.Error(t, db.Create(&model.GuardianReceipt{OrderID: order.ID, Number: "duplicate"}).Error)
			})

		})
	}
}
