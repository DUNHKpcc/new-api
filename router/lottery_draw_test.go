package router

import (
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestNativeLottery(t *testing.T) {
	for _, dialect := range []string{"sqlite", "mysql", "postgres"} {
		t.Run(dialect, func(t *testing.T) {
			var driver gorm.Dialector
			dbType := common.DatabaseTypeSQLite
			switch dialect {
			case "sqlite":
				driver = sqlite.Open(filepath.Join(t.TempDir(), "lottery.db"))
			case "mysql":
				if os.Getenv("TEST_MYSQL_DSN") == "" {
					t.Skip("TEST_MYSQL_DSN not configured")
				}
				driver = mysql.Open(os.Getenv("TEST_MYSQL_DSN"))
				dbType = common.DatabaseTypeMySQL
			case "postgres":
				if os.Getenv("TEST_POSTGRES_DSN") == "" {
					t.Skip("TEST_POSTGRES_DSN not configured")
				}
				driver = postgres.Open(os.Getenv("TEST_POSTGRES_DSN"))
				dbType = common.DatabaseTypePostgreSQL
			}
			db, err := gorm.Open(driver, &gorm.Config{})
			require.NoError(t, err)
			oldDB, oldLog, oldType := model.DB, model.LOG_DB, common.MainDatabaseType()
			oldRedis, oldSecret := common.RedisEnabled, common.SessionSecret
			model.DB, model.LOG_DB = db, db
			common.SetMainDatabaseType(dbType)
			common.RedisEnabled = false
			common.SessionSecret = "lottery-test-secret"
			sqlDB, err := db.DB()
			require.NoError(t, err)
			sqlDB.SetMaxOpenConns(1)
			if dialect != "sqlite" {
				sqlDB.SetMaxOpenConns(4)
			}
			t.Cleanup(func() {
				model.DB, model.LOG_DB = oldDB, oldLog
				common.SetMainDatabaseType(oldType)
				common.RedisEnabled = oldRedis
				common.SessionSecret = oldSecret
				_ = sqlDB.Close()
			})
			// The dedicated databases are disposable. Do not use an application database DSN.
			require.NoError(t, db.Migrator().DropTable(&model.LotteryEntry{}, &model.Lottery{}, &model.UserSession{}, &model.User{}, &model.TopUp{}))
			require.NoError(t, db.AutoMigrate(&model.User{}, &model.UserSession{}, &model.TopUp{}))
			users := []model.User{}
			for i := range 4 {
				users = append(users, model.User{Username: fmt.Sprintf("lottery-%d", i), AffCode: fmt.Sprintf("lottery-%d", i), Password: "unused", Email: fmt.Sprintf("lottery%d@example.com", i), Status: common.UserStatusEnabled, Role: common.RoleCommonUser, Group: "default", AuthVersion: 1})
			}
			users[0].Role = common.RoleAdminUser
			require.NoError(t, db.Create(&users).Error)
			// Existing users survive two schema passes; new tables and indexes are idempotent.
			for range 2 {
				require.NoError(t, db.AutoMigrate(&model.Lottery{}, &model.LotteryEntry{}))
			}
			var userCount int64
			require.NoError(t, db.Model(&model.User{}).Count(&userCount).Error)
			assert.EqualValues(t, 4, userCount)
			now := time.Now().Unix()
			lottery := model.Lottery{Title: "Test prize", Content: "Free entry", Prize: "One gift", WinnerCount: 1, DrawMode: "full", TargetCount: 2, EmailNotify: true}
			require.NoError(t, model.SaveLottery(&lottery))
			require.NoError(t, model.SaveLottery(&lottery))
			require.Error(t, model.JoinLottery(lottery.ID, users[1].Id))
			require.NoError(t, model.TransitionLottery(lottery.ID, "publish"))
			require.Error(t, model.SaveLottery(&lottery))
			require.NoError(t, model.JoinLottery(lottery.ID, users[1].Id))
			require.Error(t, model.JoinLottery(lottery.ID, users[1].Id))
			require.NoError(t, model.DrawLottery(lottery.ID, now))
			var before model.Lottery
			require.NoError(t, db.First(&before, lottery.ID).Error)
			assert.Equal(t, "active", before.Status)
			// Two entrants compete for one slot; the target cannot be exceeded.
			var wg sync.WaitGroup
			joinErrors := make(chan error, 2)
			for _, u := range users[2:] {
				wg.Go(func() { joinErrors <- model.JoinLottery(lottery.ID, u.Id) })
			}
			wg.Wait()
			close(joinErrors)
			joined := 0
			for err := range joinErrors {
				if err == nil {
					joined++
				}
			}
			assert.Equal(t, 1, joined)
			for range 2 {
				wg.Go(func() { assert.NoError(t, model.DrawLottery(lottery.ID, now)) })
			}
			wg.Wait()
			var entries []model.LotteryEntry
			require.NoError(t, db.Where("lottery_id = ?", lottery.ID).Find(&entries).Error)
			require.Len(t, entries, 2)
			winners := []model.LotteryEntry{}
			for _, e := range entries {
				if e.Winner {
					winners = append(winners, e)
				}
			}
			require.Len(t, winners, 1)
			require.Error(t, db.Create(&model.LotteryEntry{LotteryID: lottery.ID, UserID: entries[0].UserID}).Error)
			require.NoError(t, db.AutoMigrate(&model.Lottery{}, &model.LotteryEntry{}))
			var kept model.LotteryEntry
			require.NoError(t, db.First(&kept, winners[0].ID).Error)
			assert.True(t, kept.Winner)
			require.Error(t, model.JoinLottery(lottery.ID, users[0].Id))

			// Scheduled / either draws and zero-entry completion.
			for _, mode := range []string{"time", "either"} {
				l := model.Lottery{Title: mode, Content: "scheduled", Prize: "gift", WinnerCount: 2, DrawMode: mode, TargetCount: 5, DrawAt: now + 3600}
				require.NoError(t, model.SaveLottery(&l))
				require.NoError(t, model.TransitionLottery(l.ID, "publish"))
				if mode == "either" {
					require.NoError(t, model.JoinLottery(l.ID, users[1].Id))
				}
				require.NoError(t, model.DrawLottery(l.ID, now+3600))
				require.NoError(t, db.First(&l, l.ID).Error)
				assert.Equal(t, "drawn", l.Status)
				var count int64
				require.NoError(t, db.Model(&model.LotteryEntry{}).Where("lottery_id = ? AND winner = ?", l.ID, true).Count(&count).Error)
				if mode == "either" {
					assert.EqualValues(t, 1, count)
				} else {
					assert.Zero(t, count)
				}
			}
			gated := model.Lottery{Title: "Gated", Content: "rules", Prize: "gift", WinnerCount: 1, DrawMode: "full", TargetCount: 10, RequireRecharge: true, MinInvites: 1, RequiredGroup: "vip"}
			require.NoError(t, model.SaveLottery(&gated))
			require.NoError(t, model.TransitionLottery(gated.ID, "publish"))
			require.ErrorContains(t, model.JoinLottery(gated.ID, users[1].Id), "group")
			require.NoError(t, db.Model(&users[1]).Update("group", "vip").Error)
			require.ErrorContains(t, model.JoinLottery(gated.ID, users[1].Id), "recharge")
			require.NoError(t, db.Create(&model.TopUp{UserId: users[1].Id, TradeNo: "lottery-test", Status: common.TopUpStatusSuccess}).Error)
			require.ErrorContains(t, model.JoinLottery(gated.ID, users[1].Id), "invited")
			require.NoError(t, db.Model(&users[2]).Update("inviter_id", users[1].Id).Error)
			require.NoError(t, model.JoinLottery(gated.ID, users[1].Id))
			require.NoError(t, model.TransitionLottery(gated.ID, "cancel"))
			require.NoError(t, model.DrawLottery(gated.ID, now+5000))
			require.NoError(t, db.First(&gated, gated.ID).Error)
			assert.Equal(t, "cancelled", gated.Status)

			timed := model.Lottery{Title: "Timed closure", Content: "rules", Prize: "gift", WinnerCount: 1, DrawMode: "time", DrawAt: now + 3600}
			require.NoError(t, model.SaveLottery(&timed))
			require.NoError(t, model.TransitionLottery(timed.ID, "publish"))
			require.NoError(t, db.Model(&timed).Update("draw_at", now-1).Error)
			require.ErrorContains(t, model.JoinLottery(timed.ID, users[1].Id), "closed")
			require.NoError(t, model.ProcessDueLotteries(now))
			require.NoError(t, db.First(&timed, timed.ID).Error)
			assert.Equal(t, "drawn", timed.Status)
			// Mail failure persists for retries, successful delivery is not repeated.
			common.OptionMapRWMutex.Lock()
			oldFrom := common.SMTPFrom
			common.SMTPFrom = "lottery@example.com"
			common.OptionMapRWMutex.Unlock()
			defer func() { common.OptionMapRWMutex.Lock(); common.SMTPFrom = oldFrom; common.OptionMapRWMutex.Unlock() }()
			calls := 0
			send := func(_ context.Context, m common.CampaignEmailMessage) error {
				calls++
				assert.Contains(t, m.TextBody, "One gift")
				assert.Contains(t, m.MessageID, "@example.com>")
				if calls == 1 {
					return errors.New("SMTP unavailable")
				}
				return nil
			}
			require.NoError(t, service.DispatchLotteryEmails(context.Background(), send))
			require.NoError(t, db.First(&kept, kept.ID).Error)
			assert.Equal(t, "pending", kept.EmailStatus)
			require.NoError(t, db.Model(&kept).Update("email_next_at", 0).Error)
			require.NoError(t, service.DispatchLotteryEmails(context.Background(), send))
			require.NoError(t, service.DispatchLotteryEmails(context.Background(), send))
			assert.Equal(t, 2, calls)
			require.NoError(t, db.First(&kept, kept.ID).Error)
			assert.Equal(t, "sent", kept.EmailStatus)

			require.NoError(t, db.Model(&kept).Updates(map[string]any{"email_status": "pending", "email_next_at": 0}).Error)
			uncertainSend := func(context.Context, common.CampaignEmailMessage) error {
				return &common.CampaignEmailDeliveryError{Uncertain: true}
			}
			require.NoError(t, service.DispatchLotteryEmails(context.Background(), uncertainSend))
			require.NoError(t, db.First(&kept, kept.ID).Error)
			assert.Equal(t, "uncertain", kept.EmailStatus)
			require.NoError(t, service.DispatchLotteryEmails(context.Background(), send))
			assert.Equal(t, 2, calls)
			require.NoError(t, db.Model(&kept).Updates(map[string]any{"email_status": "sending", "email_next_at": now - 1}).Error)
			require.NoError(t, service.DispatchLotteryEmails(context.Background(), send))
			assert.Equal(t, 2, calls)
			require.NoError(t, db.First(&kept, kept.ID).Error)
			assert.Equal(t, "uncertain", kept.EmailStatus)
			// Real router/session boundaries and public privacy.
			gin.SetMode(gin.TestMode)
			engine := gin.New()
			SetApiRouter(engine)
			tokens := []string{}
			for _, u := range users[:2] {
				bundle, err := service.CreateLoginSession(u.Id, "password", "127.0.0.1", "lottery-test")
				require.NoError(t, err)
				tokens = append(tokens, bundle.AccessToken)
			}
			request := func(method, path, token, body string) *httptest.ResponseRecorder {
				req := httptest.NewRequest(method, path, strings.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				if token != "" {
					req.Header.Set("Authorization", "Bearer "+token)
				}
				w := httptest.NewRecorder()
				engine.ServeHTTP(w, req)
				return w
			}
			for _, route := range []struct{ method, path string }{{"GET", "/api/lotteries-admin"}, {"POST", "/api/lotteries-admin"}, {"POST", fmt.Sprintf("/api/lotteries-admin/%d/cancel", lottery.ID)}, {"GET", fmt.Sprintf("/api/lotteries-admin/%d/winners", lottery.ID)}, {"PUT", fmt.Sprintf("/api/lotteries-admin/%d/winners/%d", lottery.ID, kept.ID)}} {
				assert.Equal(t, 401, request(route.method, route.path, "", "{}").Code)
				assert.Equal(t, 403, request(route.method, route.path, tokens[1], "{}").Code)
			}
			assert.Equal(t, 200, request("GET", "/api/lotteries-admin", tokens[0], "").Code)
			oldSecure := common.SessionCookieSecure
			common.SessionCookieSecure = true
			assert.Equal(t, 403, request("POST", "/api/lotteries-admin", tokens[0], "{}").Code)
			common.SessionCookieSecure = oldSecure
			assert.Equal(t, 401, request("GET", fmt.Sprintf("/api/lotteries/%d/self", lottery.ID), "invalid-token", "").Code)
			require.NoError(t, db.AutoMigrate(&model.UserAccessToken{}))
			suffix, err := common.GenerateRandomCharsKey(43)
			require.NoError(t, err)
			pat := model.AccessTokenPrefix + suffix
			grant := &model.UserAccessToken{Name: "lottery test", TokenHash: model.AccessTokenFingerprint(pat), TokenHint: model.AccessTokenHint(pat)}
			require.NoError(t, grant.SetScopes([]string{"profile:read"}))
			require.NoError(t, model.CreateUserAccessToken(users[0].Id, grant, service.AccessTokenMaxPerUser))
			assert.Equal(t, 403, request("GET", "/api/lotteries-admin", pat, "").Code)

			public := request("GET", "/api/lottery-items", "", "")
			assert.Equal(t, 200, public.Code)
			assert.NotContains(t, public.Body.String(), "example.com")
			assert.NotContains(t, public.Body.String(), "user_id")
			assert.NotContains(t, public.Body.String(), "Gated")
			assert.Equal(t, 401, request("POST", fmt.Sprintf("/api/lotteries/%d/join", lottery.ID), "", "{}").Code)
			own := request("GET", fmt.Sprintf("/api/lotteries/%d/self", lottery.ID), tokens[1], "")
			assert.Equal(t, 200, own.Code)
			assert.Contains(t, own.Body.String(), fmt.Sprintf(`"user_id":%d`, users[1].Id))
		})
	}
}
