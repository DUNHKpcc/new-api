package model

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestEmailCampaignDatabaseMatrix(t *testing.T) {
	for _, dialect := range []string{"sqlite", "mysql", "postgres"} {
		t.Run(dialect, func(t *testing.T) {
			var driver gorm.Dialector
			switch dialect {
			case "sqlite":
				driver = sqlite.Open(filepath.Join(t.TempDir(), "campaign.db") + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
			case "mysql":
				dsn := os.Getenv("TEST_MYSQL_DSN")
				if dsn == "" {
					t.Skip("TEST_MYSQL_DSN not configured")
				}
				driver = mysql.Open(dsn)
			case "postgres":
				dsn := os.Getenv("TEST_POSTGRES_DSN")
				if dsn == "" {
					t.Skip("TEST_POSTGRES_DSN not configured")
				}
				driver = postgres.New(postgres.Config{DSN: dsn, PreferSimpleProtocol: true})
			}
			recorder := &migrationSQLRecorder{}
			db, err := gorm.Open(driver, &gorm.Config{Logger: recorder})
			require.NoError(t, err)
			sqlDB, err := db.DB()
			require.NoError(t, err)
			sqlDB.SetMaxOpenConns(8)
			previousDB, previousType := DB, common.MainDatabaseType()
			DB = db
			common.SetMainDatabaseType(common.DatabaseType(dialect))
			initCol()
			t.Cleanup(func() { DB = previousDB; common.SetMainDatabaseType(previousType); initCol(); _ = sqlDB.Close() })
			require.NoError(t, db.AutoMigrate(&User{}))
			models := []any{&EmailCampaign{}, &EmailDelivery{}, &EmailSubscription{}, &EmailDispatchConfig{}}
			require.NoError(t, db.AutoMigrate(models...))
			recorder.reset()
			require.NoError(t, db.AutoMigrate(models...))
			assert.Empty(t, recorder.schemaMutations(), "repeated migration must not change schema")
			group := "mail-" + uuid.NewString()[:8]
			users := []User{
				{Username: group + "-a", Email: "A@example.com", Group: group, Status: common.UserStatusEnabled},
				{Username: group + "-dup", Email: " a@example.com ", Group: group, Status: common.UserStatusEnabled},
				{Username: group + "-b", Email: "b@example.com", Group: group, Status: common.UserStatusEnabled},
				{Username: group + "-bad", Email: "not an email", Group: group, Status: common.UserStatusEnabled},
				{Username: group + "-off", Email: "off@example.com", Group: group, Status: common.UserStatusDisabled},
				{Username: group + "-gone", Email: "gone@example.com", Group: group, Status: common.UserStatusEnabled},
			}
			for i := range users {
				users[i].AffCode = uuid.NewString()[:12]
			}
			require.NoError(t, db.Create(&users).Error)
			require.NoError(t, db.Delete(&users[5]).Error)
			userIDs := make([]int, len(users))
			for i := range users {
				userIDs[i] = users[i].Id
			}
			t.Cleanup(func() {
				var ids []int64
				require.NoError(t, db.Model(&EmailCampaign{}).Where("target_group = ?", group).Pluck("id", &ids).Error)
				if len(ids) > 0 {
					require.NoError(t, db.Where("campaign_id IN ?", ids).Delete(&EmailDelivery{}).Error)
					require.NoError(t, db.Where("id IN ?", ids).Delete(&EmailCampaign{}).Error)
				}
				require.NoError(t, db.Where("user_id IN ?", userIDs).Delete(&EmailSubscription{}).Error)
				require.NoError(t, db.Unscoped().Where("id IN ?", userIDs).Delete(&User{}).Error)
			})
			preferences, err := GetEmailPreferences(users[0].Id)
			require.NoError(t, err)
			assert.Equal(t, &EmailPreferences{Platform: true}, preferences)
			count, err := PreviewEmailCampaignRecipients(EmailCategoryPlatform, group)
			require.NoError(t, err)
			assert.EqualValues(t, 2, count, "dedupe normalized email; exclude disabled, deleted and invalid recipients")
			campaign := EmailCampaign{Subject: "Promotion", Body: "Offer", Category: EmailCategoryPromotion, TargetGroup: group, CreatedBy: users[0].Id}
			require.NoError(t, CreateEmailCampaign(&campaign))
			_, err = QueueEmailCampaign(campaign.ID)
			require.ErrorIs(t, err, ErrEmailAudienceEmpty)
			saved, err := GetEmailCampaign(campaign.ID)
			require.NoError(t, err)
			assert.Equal(t, EmailCampaignDraft, saved.Status)
			require.NoError(t, SetEmailPreferences(users[0].Id, true, true))
			queued, err := QueueEmailCampaign(campaign.ID)
			require.NoError(t, err)
			assert.EqualValues(t, 1, queued.DeliveryCounts.Pending)
			queued, err = QueueEmailCampaign(campaign.ID)
			require.NoError(t, err)
			assert.EqualValues(t, 1, queued.DeliveryCounts.Total)
			assert.ErrorIs(t, UpdateEmailCampaign(campaign.ID, "Changed", "Changed", EmailCategoryPlatform, group), ErrEmailCampaignNotDraft)
			config, err := GetEmailDispatchConfig()
			require.NoError(t, err)
			original := *config
			t.Cleanup(func() {
				require.NoError(t, db.Model(&EmailDispatchConfig{}).Where("id = ?", 1).Updates(map[string]any{"rate_per_minute": original.RatePerMinute, "next_send_at": original.NextSendAt}).Error)
			})
			require.NoError(t, UpdateEmailDispatchRate(20))
			require.NoError(t, db.Model(&EmailDispatchConfig{}).Where("id = ?", 1).Update("next_send_at", 0).Error)
			now := time.Now().UnixMilli()
			delivery, err := ClaimEmailDelivery("worker-a", now, 60000)
			require.NoError(t, err)
			require.NotNil(t, delivery)
			assert.Equal(t, campaign.ID, delivery.CampaignID)
			assert.Equal(t, "a@example.com", delivery.Email)
			assert.Equal(t, 1, delivery.Attempts)
			assert.NotEmpty(t, delivery.MessageID)
			blocked, err := ClaimEmailDelivery("worker-b", now, 60000)
			require.NoError(t, err)
			assert.Nil(t, blocked, "pacing survives a separate worker")
			job, err := PrepareEmailDelivery(delivery.ID, "worker-a")
			require.NoError(t, err)
			require.NotNil(t, job)
			unsubscribeToken := job.UnsubscribeToken
			info, err := GetEmailUnsubscribeInfo(job.UnsubscribeToken)
			require.NoError(t, err)
			assert.Equal(t, &EmailUnsubscribeInfo{Category: EmailCategoryPromotion, Subscribed: true}, info)
			encoded, err := common.Marshal(job)
			require.NoError(t, err)
			assert.NotContains(t, string(encoded), job.UnsubscribeToken)
			assert.ErrorIs(t, FinishEmailDelivery(delivery.ID, "worker-b", EmailDeliverySent, "", 0), ErrEmailDeliveryLeaseLost)
			require.NoError(t, FinishEmailDelivery(delivery.ID, "worker-a", EmailDeliverySent, "", 0))
			saved, err = GetEmailCampaign(campaign.ID)
			require.NoError(t, err)
			assert.Equal(t, EmailCampaignCompleted, saved.Status)
			assert.EqualValues(t, 1, saved.DeliveryCounts.Sent)
			require.NoError(t, UnsubscribeEmail(job.UnsubscribeToken))
			require.NoError(t, UnsubscribeEmail(job.UnsubscribeToken))
			preferences, err = GetEmailPreferences(users[0].Id)
			require.NoError(t, err)
			assert.Equal(t, &EmailPreferences{Platform: true}, preferences, "promotion unsubscribe leaves platform notices enabled")

			platform := EmailCampaign{Subject: "Maintenance", Body: "Scheduled maintenance", Category: EmailCategoryPlatform, TargetGroup: group}
			require.NoError(t, CreateEmailCampaign(&platform))
			_, err = QueueEmailCampaign(platform.ID)
			require.NoError(t, err)
			delivery, err = ClaimEmailDelivery("worker-a", now+3000, 60000)
			require.NoError(t, err)
			require.NotNil(t, delivery)
			require.NoError(t, SetEmailPreferences(delivery.UserID, false, false))
			job, err = PrepareEmailDelivery(delivery.ID, "worker-a")
			require.NoError(t, err)
			assert.Nil(t, job, "unsubscribe after enqueue suppresses delivery")
			delivery, err = ClaimEmailDelivery("crashed-worker", now+6000, 1000)
			require.NoError(t, err)
			require.NotNil(t, delivery)
			blocked, err = ClaimEmailDelivery("restarted-worker", now+10000, 60000)
			require.NoError(t, err)
			assert.Nil(t, blocked, "expired SMTP attempt must never automatically resend")
			saved, err = GetEmailCampaign(platform.ID)
			require.NoError(t, err)
			assert.Equal(t, EmailCampaignCompleted, saved.Status)
			assert.EqualValues(t, 1, saved.DeliveryCounts.Skipped)
			assert.EqualValues(t, 1, saved.DeliveryCounts.Uncertain)
			uncertain, total, err := ListEmailDeliveries(platform.ID, 0, 20, EmailDeliveryUncertain)
			require.NoError(t, err)
			assert.EqualValues(t, 1, total)
			require.Len(t, uncertain, 1)
			assert.Equal(t, delivery.ID, uncertain[0].ID)
			assert.ErrorIs(t, FinishEmailDelivery(delivery.ID, "crashed-worker", EmailDeliverySent, "", 0), ErrEmailDeliveryLeaseLost)

			require.NoError(t, SetEmailPreferences(users[0].Id, false, true))
			cancelled := EmailCampaign{Subject: "Cancel", Body: "Do not send", Category: EmailCategoryPlatform, TargetGroup: group}
			require.NoError(t, CreateEmailCampaign(&cancelled))
			_, err = QueueEmailCampaign(cancelled.ID)
			require.NoError(t, err)
			// Two independent workers race for one persisted rate reservation.
			var workers sync.WaitGroup
			start := make(chan struct{})
			type claimResult struct {
				delivery *EmailDelivery
				err      error
			}
			results := make(chan claimResult, 2)
			for _, worker := range []string{"worker-a", "worker-b"} {
				workers.Go(func() {
					<-start
					d, err := ClaimEmailDelivery(worker, now+13000, 60000)
					results <- claimResult{d, err}
				})
			}
			close(start)
			workers.Wait()
			close(results)
			var claims []*EmailDelivery
			for result := range results {
				require.NoError(t, result.err)
				if result.delivery != nil {
					claims = append(claims, result.delivery)
				}
			}
			require.Len(t, claims, 1)
			require.NoError(t, CancelEmailCampaign(cancelled.ID))
			require.NoError(t, FinishEmailDelivery(claims[0].ID, claims[0].LockedBy, EmailDeliveryRetry, "Temporary rejection", now+20000))
			saved, err = GetEmailCampaign(cancelled.ID)
			require.NoError(t, err)
			assert.Equal(t, EmailCampaignCancelled, saved.Status)
			assert.EqualValues(t, 2, saved.DeliveryCounts.Cancelled)
			assert.False(t, HasPendingEmailDeliveries())
			assert.Error(t, UpdateEmailDispatchRate(0))
			assert.Error(t, UpdateEmailDispatchRate(121))

			// Explicitly rejected SMTP attempts may retry, retaining one logical Message-ID.
			require.NoError(t, SetEmailPreferences(users[0].Id, true, true))
			retrying := EmailCampaign{Subject: "Retry", Body: "Offer", Category: EmailCategoryPromotion, TargetGroup: group}
			require.NoError(t, CreateEmailCampaign(&retrying))
			_, err = QueueEmailCampaign(retrying.ID)
			require.NoError(t, err)
			delivery, err = ClaimEmailDelivery("retry-worker", now+16000, 60000)
			require.NoError(t, err)
			require.NotNil(t, delivery)
			messageID := delivery.MessageID
			require.NoError(t, FinishEmailDelivery(delivery.ID, "retry-worker", EmailDeliveryRetry, "Temporary SMTP rejection", now+22000))
			blocked, err = ClaimEmailDelivery("retry-worker", now+19000, 60000)
			require.NoError(t, err)
			assert.Nil(t, blocked, "next retry time is persisted")
			idleConfig, err := GetEmailDispatchConfig()
			require.NoError(t, err)
			assert.LessOrEqual(t, idleConfig.NextSendAt, now+19000, "a future retry must not leave a 1 ms pacing reservation")
			delivery, err = ClaimEmailDelivery("retry-worker", now+22000, 60000)
			require.NoError(t, err)
			require.NotNil(t, delivery)
			assert.Equal(t, 2, delivery.Attempts)
			assert.Equal(t, messageID, delivery.MessageID)
			require.NoError(t, FinishEmailDelivery(delivery.ID, "retry-worker", EmailDeliverySent, "", 0))

			// Concurrent completion must not leave a campaign stuck in sending.
			concurrent := EmailCampaign{Subject: "Concurrent", Body: "Notice", Category: EmailCategoryPlatform, TargetGroup: group}
			require.NoError(t, CreateEmailCampaign(&concurrent))
			_, err = QueueEmailCampaign(concurrent.ID)
			require.NoError(t, err)
			first, err := ClaimEmailDelivery("finisher-a", now+25000, 60000)
			require.NoError(t, err)
			require.NotNil(t, first)
			second, err := ClaimEmailDelivery("finisher-b", now+28000, 60000)
			require.NoError(t, err)
			require.NotNil(t, second)
			finishStart := make(chan struct{})
			finishResults := make(chan error, 2)
			for _, claimed := range []*EmailDelivery{first, second} {
				workers.Go(func() {
					<-finishStart
					finishResults <- FinishEmailDelivery(claimed.ID, claimed.LockedBy, EmailDeliverySent, "", 0)
				})
			}
			close(finishStart)
			workers.Wait()
			close(finishResults)
			for finishErr := range finishResults {
				require.NoError(t, finishErr)
			}
			saved, err = GetEmailCampaign(concurrent.ID)
			require.NoError(t, err)
			assert.Equal(t, EmailCampaignCompleted, saved.Status)
			assert.EqualValues(t, 2, saved.DeliveryCounts.Sent)
			changed := EmailCampaign{Subject: "Changed recipients", Body: "Notice", Category: EmailCategoryPlatform, TargetGroup: group}
			require.NoError(t, CreateEmailCampaign(&changed))
			_, err = QueueEmailCampaign(changed.ID)
			require.NoError(t, err)
			require.NoError(t, db.Model(&users[0]).Update("email", "changed@example.com").Error)
			require.NoError(t, db.Model(&users[2]).Update("status", common.UserStatusDisabled).Error)
			for _, timestamp := range []int64{now + 31000, now + 34000} {
				delivery, err = ClaimEmailDelivery("recheck-worker", timestamp, 60000)
				require.NoError(t, err)
				require.NotNil(t, delivery)
				job, err = PrepareEmailDelivery(delivery.ID, "recheck-worker")
				require.NoError(t, err)
				assert.Nil(t, job, "changed email and disabled account must suppress queued mail")
			}
			saved, err = GetEmailCampaign(changed.ID)
			require.NoError(t, err)
			assert.Equal(t, EmailCampaignCompleted, saved.Status)
			assert.EqualValues(t, 2, saved.DeliveryCounts.Skipped)
			// Repeated migration preserves queued history and stable unsubscribe tokens.
			require.NoError(t, db.AutoMigrate(models...))
			recorder.reset()
			require.NoError(t, db.AutoMigrate(models...))
			assert.Empty(t, recorder.schemaMutations(), "repeated migration must not change schema")
			saved, err = GetEmailCampaign(campaign.ID)
			require.NoError(t, err)
			assert.EqualValues(t, 1, saved.DeliveryCounts.Sent)
			var subscription EmailSubscription
			require.NoError(t, db.Where("user_id = ? AND category = ?", users[0].Id, EmailCategoryPromotion).First(&subscription).Error)
			assert.Equal(t, unsubscribeToken, subscription.UnsubscribeToken, "unsubscribe link stays valid across preferences and migrations")
			dedup := EmailSubscription{UserID: users[0].Id, Category: EmailCategoryPromotion, UnsubscribeToken: strings.Repeat("f", 64)}
			assert.Error(t, db.Create(&dedup).Error, "category preference uniqueness survives migration")
		})
	}
}
