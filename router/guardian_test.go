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
			for range 2 {
				require.NoError(t, db.AutoMigrate(&model.GuardianService{}, &model.GuardianServiceOrder{}, &model.GuardianOrderEvent{}, &model.GuardianPaymentAnomaly{}))
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
			for _, route := range []struct{ method, path string }{{"POST", "/api/guardian/orders"}, {"GET", "/api/guardian/orders/self"}, {"GET", "/api/guardian/admin/services"}, {"POST", "/api/guardian/admin/services"}, {"PUT", "/api/guardian/admin/services/1"}, {"GET", "/api/guardian/admin/orders"}, {"PUT", "/api/guardian/admin/orders/1"}} {
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
				require.NoError(t, db.AutoMigrate(&model.GuardianService{}, &model.GuardianServiceOrder{}, &model.GuardianOrderEvent{}, &model.GuardianPaymentAnomaly{}))
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
				require.NoError(t, db.AutoMigrate(&model.GuardianService{}, &model.GuardianServiceOrder{}, &model.GuardianOrderEvent{}, &model.GuardianPaymentAnomaly{}))
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
		})
	}
}
