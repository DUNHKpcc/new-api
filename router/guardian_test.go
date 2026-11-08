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
			require.NoError(t, db.AutoMigrate(&model.User{}, &model.UserSession{}, &model.UserAccessToken{}, &model.Option{}, &model.AuditLog{}))
			suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
			users := []model.User{{Username: "buyer-" + suffix, Password: "unused", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, AuthVersion: 1, AffCode: "a" + suffix}, {Username: "admin-" + suffix, Password: "unused", Role: common.RoleAdminUser, Status: common.UserStatusEnabled, AuthVersion: 1, AffCode: "b" + suffix}, {Username: "other-" + suffix, Password: "unused", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, AuthVersion: 1, AffCode: "c" + suffix}}
			require.NoError(t, db.Create(&users).Error)
			var legacy guardianServiceBeforeChip
			if os.Getenv("TEST_GUARDIAN_UPGRADE") == "1" {
				require.NoError(t, db.AutoMigrate(&guardianServiceBeforeChip{}, &guardianOrderBeforePeriod{}, &guardianAccountStatusBeforeLinks{}))
				require.NoError(t, db.Create(&guardianAccountStatusBeforeLinks{UserID: users[2].Id, Banned: true, UpdatedAt: 123, UpdatedBy: users[1].Id}).Error)
				legacy = guardianServiceBeforeChip{BaseSales: 5, Title: "Historical service", PriceCents: 19900, Features: model.GuardianFeatures{"Existing feature"}, Route: "any"}
				require.NoError(t, db.Create(&legacy).Error)
				require.NoError(t, db.Create(&guardianOrderBeforePeriod{AmountCents: 19900, TradeNo: "historical-" + suffix, ServiceID: legacy.ID, Status: "delivered"}).Error)
			}
			for range 2 {
				require.NoError(t, db.AutoMigrate(&model.GuardianAccountStatus{}, &model.GuardianService{}, &model.GuardianServiceOrder{}, &model.GuardianOrderEvent{}, &model.GuardianPaymentAnomaly{}))
			}
			if legacy.ID != 0 {
				var migratedStatus model.GuardianAccountStatus
				require.NoError(t, db.First(&migratedStatus, "user_id = ?", users[2].Id).Error)
				assert.True(t, migratedStatus.Banned)
				assert.Empty(t, migratedStatus.HistoricalCategory)
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
				var historicalOrder model.GuardianServiceOrder
				require.NoError(t, db.Where("trade_no = ?", "historical-"+suffix).First(&historicalOrder).Error)
				assert.Empty(t, historicalOrder.BillingPeriod)
				assert.Zero(t, historicalOrder.ClaudeMaxPriceCents)
				assert.EqualValues(t, 19900, historicalOrder.AmountCents)
				services := []model.GuardianService{migrated}
				require.NoError(t, model.PopulateGuardianSales(services))
				assert.EqualValues(t, 6, *services[0].SoldCount)
			}
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
			for _, route := range []struct{ method, path string }{{"POST", "/api/guardian/orders"}, {"GET", "/api/guardian/orders/self"}, {"GET", "/api/guardian/admin/services"}, {"POST", "/api/guardian/admin/services"}, {"PUT", "/api/guardian/admin/services/1"}, {"GET", "/api/guardian/admin/orders"}, {"PUT", "/api/guardian/admin/orders/1"}, {"GET", "/api/guardian/admin/queue"}, {"PUT", "/api/guardian/admin/queue/settings"}, {"POST", "/api/guardian/admin/queue/1/move"}} {
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
				{"GET", "/api/guardian/admin/historical-accounts"},
				{"PUT", "/api/guardian/admin/historical-accounts"},
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

			t.Run("mac mini uses existing service and order storage", func(t *testing.T) {
				macPayload := strings.Replace(payload, `"route":"vps"`, `"route":"mac_mini","chip":"m4","region":"kr_seoul","hardware_config":"16_256","quarter_price_cents":150000,"annual_price_cents":500000,"claude_max_price_cents":100000`, 1)
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
					{`"claude_max_price_cents":100000`, `"claude_max_price_cents":10000001`},
				} {
					assert.Equal(t, false, decode(request("PUT", servicePath, tokens[1], strings.Replace(updated, invalid.from, invalid.to, 1)))["success"])
				}
				for _, tc := range []struct {
					period string
					amount int64
					addon  int64
				}{
					{"monthly", 59901, 0}, {"quarterly", 150000, 0}, {"yearly", 500000, 0},
					{"monthly", 159901, 100000}, {"quarterly", 250000, 100000}, {"yearly", 600000, 100000},
				} {
					body := fmt.Sprintf(`{"service_id":%d,"billing_period":%q,"add_claude_max":%t,"claude_max_price_cents":1,"amount_cents":1,"payment_method":"alipay","route":"mac_mini","os":"macos","contact_type":"email","contact":"buyer@example.com"}`, macID, tc.period, tc.addon > 0)
					result := decode(request("POST", "/api/guardian/orders", tokens[0], body))
					require.Equal(t, true, result["success"], result)
					var saved model.GuardianServiceOrder
					require.NoError(t, db.Where("trade_no = ?", result["data"].(map[string]any)["trade_no"]).First(&saved).Error)
					assert.Equal(t, tc.period, saved.BillingPeriod)
					assert.Equal(t, tc.amount, saved.AmountCents)
					assert.Equal(t, tc.addon, saved.ClaudeMaxPriceCents)
					assert.Equal(t, fmt.Sprintf("%.2f", float64(tc.amount)/100), result["data"].(map[string]any)["pay_params"].(map[string]any)["money"])
					require.NoError(t, db.Model(&storedService).Updates(map[string]any{"annual_price_cents": 400000, "claude_max_price_cents": 50000}).Error)
					require.NoError(t, db.First(&saved, saved.ID).Error)
					assert.Equal(t, tc.amount, saved.AmountCents)
					assert.Equal(t, tc.addon, saved.ClaudeMaxPriceCents)
					require.NoError(t, db.Model(&storedService).Updates(map[string]any{"annual_price_cents": 500000, "claude_max_price_cents": 100000}).Error)
					if tc.addon > 0 {
						require.Error(t, model.CompleteGuardianOrder(saved.TradeNo, "{}", fmt.Sprintf("%.2f", float64(tc.amount-tc.addon)/100), "alipay"))
						require.NoError(t, model.CompleteGuardianOrder(saved.TradeNo, "{}", fmt.Sprintf("%.2f", float64(tc.amount)/100), "alipay"))
						require.NoError(t, db.First(&saved, saved.ID).Error)
						assert.Equal(t, "paid", saved.Status)
					}
					require.NoError(t, db.Where("order_id = ?", saved.ID).Delete(&model.GuardianOrderEvent{}).Error)
					require.NoError(t, db.Delete(&saved).Error)
				}
				baseRequest := model.GuardianOrderRequest{ServiceID: macID, PaymentMethod: "alipay", Route: "mac_mini", OS: "macos", ContactType: "email", Contact: "buyer@example.com"}
				for _, mismatch := range []struct {
					serviceID int
					route     string
				}{
					{macID, "vps"}, {id, "mac_mini"},
				} {
					body := fmt.Sprintf(`{"service_id":%d,"payment_method":"alipay","route":%q,"os":"macos","contact_type":"email","contact":"buyer@example.com"}`, mismatch.serviceID, mismatch.route)
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

				order, err := model.CreateGuardianOrder(users[0].Id, model.GuardianOrderRequest{
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

				var historicalService model.GuardianService
				require.NoError(t, db.First(&historicalService, id).Error)
				originalHistoryBase := historicalService.BaseSales
				require.NoError(t, db.Model(&historicalService).Update("base_sales", originalHistoryBase+20).Error)
				historyPath := "/api/guardian/admin/historical-accounts"
				assert.Equal(t, http.StatusUnauthorized, request("GET", historyPath, "", "").Code)
				assert.Equal(t, http.StatusForbidden, request("GET", historyPath, tokens[0], "").Code)
				defaults, err := model.GetGuardianHistoricalAccounts()
				require.NoError(t, err)
				assert.Equal(t, model.GuardianHistoricalAccounts{}, defaults)
				baseline := snapshot()
				bodyHistory := `{"total_users":20,"banned_only":1,"kyc_only":1,"phone_only":1,"banned_kyc":1,"banned_phone":1,"kyc_phone":1,"all_three":1}`
				assert.Equal(t, http.StatusUnauthorized, request("PUT", historyPath, "", bodyHistory).Code)
				assert.Equal(t, http.StatusForbidden, request("PUT", historyPath, tokens[0], bodyHistory).Code)
				require.Equal(t, true, decode(request("PUT", historyPath, tokens[1], bodyHistory))["success"])
				historicalStats := snapshot()
				assert.Equal(t, baseline.PurchasedUsers+20, historicalStats.PurchasedUsers)
				assert.Equal(t, baseline.RecordedUsers+20, historicalStats.RecordedUsers)
				assert.Equal(t, int64(20), historicalStats.HistoricalUsers)
				assert.Equal(t, baseline.HealthyUsers+13, historicalStats.HealthyUsers)
				assert.Equal(t, baseline.AllIssuesUsers+1, historicalStats.AllIssuesUsers)
				assert.Equal(t, baseline.BannedUsers+4, historicalStats.BannedUsers)
				assert.Equal(t, baseline.PhoneRequiredUsers+4, historicalStats.PhoneRequiredUsers)
				assert.Equal(t, baseline.KYCRequiredUsers+4, historicalStats.KYCRequiredUsers)
				storedHistory, err := model.GetGuardianHistoricalAccounts()
				require.NoError(t, err)
				assert.Equal(t, int64(20), storedHistory.TotalUsers)
				for _, invalid := range []string{`{"total_users":-1}`, `{"total_users":1,"banned_only":1,"kyc_only":1}`, `{"total_users":1.5}`, `{"total_users":1000000001}`, `{"total_users":1000000000}`} {
					assert.Equal(t, false, decode(request("PUT", historyPath, tokens[1], invalid))["success"])
				}
				assert.Error(t, model.UpdateOption("GuardianHistoricalAccounts", `{"total_users":0,"all_three":1}`))
				assert.Equal(t, historicalStats, snapshot())
				require.Equal(t, true, decode(request("PUT", historyPath, tokens[1], `{"total_users":0,"banned_only":0,"kyc_only":0,"phone_only":0,"banned_kyc":0,"banned_phone":0,"kyc_phone":0,"all_three":0}`))["success"])
				assert.Equal(t, baseline, snapshot())
				before := snapshot()
				var service model.GuardianService
				require.NoError(t, db.First(&service, id).Error)
				originalBase := service.BaseSales
				require.NoError(t, db.Model(&service).Update("base_sales", originalBase+7).Error)
				withBase := snapshot()
				assert.Equal(t, before.HistoricalUsers, withBase.HistoricalUsers)
				assert.Equal(t, before.PurchasedUsers, withBase.PurchasedUsers)
				assert.Equal(t, before.HealthyUsers, withBase.HealthyUsers)
				require.NoError(t, db.Model(&service).Update("base_sales", originalBase).Error)

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

				historyAll := `{"total_users":8,"banned_only":1,"kyc_only":1,"phone_only":1,"banned_kyc":1,"banned_phone":1,"kyc_phone":1,"all_three":1}`
				require.Equal(t, true, decode(request("PUT", historyPath, tokens[1], historyAll))["success"])
				beforeLink := snapshot()
				linkPath := fmt.Sprintf("%s/%d", path, buyers[1].Id)
				for _, tc := range []struct {
					category                string
					banned, phone, kyc, all int64
				}{
					{"healthy", 0, 0, 0, 0}, {"banned_only", 1, 0, 0, 0}, {"kyc_only", 0, 0, 1, 0}, {"phone_only", 0, 1, 0, 0},
					{"banned_kyc", 1, 0, 1, 0}, {"banned_phone", 1, 1, 0, 0}, {"kyc_phone", 0, 1, 1, 0}, {"all_three", 1, 1, 1, 1},
				} {
					linkedBody := fmt.Sprintf(`{"banned":false,"phone_required":false,"kyc_required":false,"historical_category":%q}`, tc.category)
					for range 2 {
						require.Equal(t, true, decode(request("PUT", linkPath, tokens[1], linkedBody))["success"])
					}
					linked := snapshot()
					assert.Equal(t, beforeLink.PurchasedUsers-1, linked.PurchasedUsers)
					assert.Equal(t, beforeLink.RecordedUsers, linked.RecordedUsers)
					assert.Equal(t, beforeLink.BannedUsers-tc.banned, linked.BannedUsers)
					assert.Equal(t, beforeLink.PhoneRequiredUsers-tc.phone, linked.PhoneRequiredUsers)
					assert.Equal(t, beforeLink.KYCRequiredUsers-tc.kyc, linked.KYCRequiredUsers)
					assert.Equal(t, beforeLink.AllIssuesUsers-tc.all, linked.AllIssuesUsers)
				}

				linkedSnapshot := snapshot()
				repeatOrder := model.GuardianServiceOrder{TradeNo: "repeat-" + suffix, ServiceID: id, UserID: buyers[1].Id, Status: "paid"}
				require.NoError(t, db.Create(&repeatOrder).Error)
				assert.Equal(t, linkedSnapshot, snapshot())
				require.NoError(t, db.Model(&orders[0]).Update("status", "paid").Error)
				assert.Equal(t, false, decode(request("PUT", updatePath, tokens[1], `{"banned":false,"phone_required":false,"kyc_required":false,"historical_category":"all_three"}`))["success"])
				require.NoError(t, db.Model(&orders[0]).Update("status", "refunded").Error)
				assert.Equal(t, linkedSnapshot, snapshot())
				assert.Equal(t, false, decode(request("PUT", historyPath, tokens[1], `{}`))["success"])
				assert.Equal(t, false, decode(request("PUT", fmt.Sprintf("/api/guardian/admin/services/%d", id), tokens[1], payload))["success"])
				var linkedStored model.GuardianAccountStatus
				for range 2 {
					require.NoError(t, db.AutoMigrate(&model.GuardianAccountStatus{}))
				}
				require.NoError(t, db.First(&linkedStored, "user_id = ?", buyers[1].Id).Error)
				assert.Equal(t, "all_three", linkedStored.HistoricalCategory)
				assert.Equal(t, false, decode(request("PUT", linkPath, tokens[1], `{"banned":false,"phone_required":false,"kyc_required":false,"historical_category":"invalid"}`))["success"])
				require.NoError(t, db.Model(&model.GuardianServiceOrder{}).Where("user_id = ?", buyers[1].Id).Update("status", "refunded").Error)
				assert.Equal(t, linkedSnapshot.PurchasedUsers, snapshot().PurchasedUsers)
				linkedList, _, err := model.ListGuardianAccountStatuses(1, 50, buyers[1].Username)
				require.NoError(t, err)
				require.Len(t, linkedList, 1)
				assert.Equal(t, "all_three", linkedList[0].HistoricalCategory)
				require.Equal(t, true, decode(request("PUT", linkPath, tokens[1], cleared))["success"])
				require.NoError(t, db.Model(&orders[2]).Update("status", "in_progress").Error)
				assert.Equal(t, beforeLink.PurchasedUsers, snapshot().PurchasedUsers)
				require.Equal(t, true, decode(request("PUT", historyPath, tokens[1], `{}`))["success"])
				require.NoError(t, db.Model(&historicalService).Update("base_sales", originalHistoryBase).Error)
			})
			orderBody := fmt.Sprintf(`{"service_id":%d,"payment_method":"alipay","route":"vps","os":"linux","contact_type":"email","contact":" reader@example.test ","note":"部署"}`, id)
			payment.ComplianceConfirmed = false
			assert.Equal(t, false, decode(request("GET", "/api/guardian/services", "", ""))["data"].(map[string]any)["purchase_enabled"])
			assert.Equal(t, false, decode(request("POST", "/api/guardian/orders", tokens[0], orderBody))["success"])
			payment.ComplianceConfirmed = true
			for _, invalid := range []string{strings.Replace(orderBody, "alipay", "stripe", 1), strings.Replace(orderBody, "alipay", "waffo_pancake", 1), strings.Replace(orderBody, "alipay", "missing", 1), strings.Replace(orderBody, `"os":"linux"`, `"os":"unknown"`, 1), strings.Replace(orderBody, "reader@example.test", " ", 1)} {
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
				require.NoError(t, db.AutoMigrate(&model.GuardianAccountStatus{}, &model.GuardianService{}, &model.GuardianServiceOrder{}, &model.GuardianOrderEvent{}, &model.GuardianPaymentAnomaly{}))
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
				require.NoError(t, db.AutoMigrate(&model.GuardianAccountStatus{}, &model.GuardianService{}, &model.GuardianServiceOrder{}, &model.GuardianOrderEvent{}, &model.GuardianPaymentAnomaly{}))
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
		})
	}
}
