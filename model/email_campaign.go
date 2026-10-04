package model

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

const (
	EmailCategoryPromotion = "promotion"
	EmailCategoryPlatform  = "platform"
	EmailCampaignDraft     = "draft"
	EmailCampaignQueued    = "queued"
	EmailCampaignSending   = "sending"
	EmailCampaignCompleted = "completed"
	EmailCampaignCancelled = "cancelled"
	EmailDeliveryPending   = "pending"
	EmailDeliverySending   = "sending"
	EmailDeliveryRetry     = "retry"
	EmailDeliverySent      = "sent"
	EmailDeliveryFailed    = "failed"
	EmailDeliverySkipped   = "skipped"
	EmailDeliveryUncertain = "uncertain"
	EmailDeliveryCancelled = "cancelled"
)

var (
	ErrInvalidEmailCampaign   = errors.New("invalid email campaign")
	ErrEmailCampaignNotDraft  = errors.New("email campaign is no longer a draft")
	ErrEmailAudienceEmpty     = errors.New("no subscribed users with valid email addresses match this audience")
	ErrEmailDeliveryLeaseLost = errors.New("email delivery lease lost")
)

type EmailDeliveryCounts struct {
	Pending   int64 `json:"pending"`
	Sending   int64 `json:"sending"`
	Retry     int64 `json:"retry"`
	Sent      int64 `json:"sent"`
	Failed    int64 `json:"failed"`
	Skipped   int64 `json:"skipped"`
	Uncertain int64 `json:"uncertain"`
	Cancelled int64 `json:"cancelled"`
	Total     int64 `json:"total"`
}

type EmailCampaign struct {
	ID             int64               `json:"id" gorm:"primaryKey"`
	Subject        string              `json:"subject" gorm:"type:varchar(200);not null"`
	Body           string              `json:"body" gorm:"type:text;not null"`
	Category       string              `json:"category" gorm:"type:varchar(20);not null"`
	TargetGroup    string              `json:"group" gorm:"type:varchar(64);not null"`
	Status         string              `json:"status" gorm:"type:varchar(20);not null;index"`
	CreatedBy      int                 `json:"created_by"`
	CreatedAt      int64               `json:"created_at"`
	UpdatedAt      int64               `json:"updated_at"`
	DeliveryCounts EmailDeliveryCounts `json:"delivery_counts" gorm:"-"`
}

type EmailDelivery struct {
	ID            int64  `json:"id" gorm:"primaryKey"`
	CampaignID    int64  `json:"campaign_id" gorm:"uniqueIndex:idx_email_campaign_recipient,priority:1;index"`
	UserID        int    `json:"user_id" gorm:"index"`
	Email         string `json:"email" gorm:"type:varchar(254);not null"`
	EmailHash     string `json:"-" gorm:"type:varchar(64);not null;uniqueIndex:idx_email_campaign_recipient,priority:2"`
	Status        string `json:"status" gorm:"type:varchar(20);not null;index:idx_email_delivery_due,priority:1"`
	Attempts      int    `json:"attempts"`
	NextAttemptAt int64  `json:"next_attempt_at" gorm:"index:idx_email_delivery_due,priority:2"`
	LockedBy      string `json:"-" gorm:"type:varchar(64)"`
	LockedUntil   int64  `json:"-" gorm:"index"`
	MessageID     string `json:"message_id" gorm:"type:varchar(160)"`
	LastError     string `json:"last_error" gorm:"type:varchar(500)"`
	SentAt        int64  `json:"sent_at"`
	CreatedAt     int64  `json:"created_at"`
	UpdatedAt     int64  `json:"updated_at"`
}

type EmailSubscription struct {
	ID               int64  `json:"-" gorm:"primaryKey"`
	UserID           int    `json:"user_id" gorm:"uniqueIndex:idx_email_subscription_user_category,priority:1"`
	Category         string `json:"category" gorm:"type:varchar(20);not null;uniqueIndex:idx_email_subscription_user_category,priority:2"`
	Subscribed       bool   `json:"subscribed" gorm:"not null"`
	UnsubscribeToken string `json:"-" gorm:"type:varchar(64);not null;uniqueIndex"`
	CreatedAt        int64  `json:"created_at"`
	UpdatedAt        int64  `json:"updated_at"`
}

type EmailDispatchConfig struct {
	ID            int   `json:"-" gorm:"primaryKey;autoIncrement:false"`
	RatePerMinute int   `json:"rate_per_minute"`
	NextSendAt    int64 `json:"-"`
}

type EmailPreferences struct {
	Promotion bool `json:"promotion"`
	Platform  bool `json:"platform"`
}

type EmailUnsubscribeInfo struct {
	Category   string `json:"category"`
	Subscribed bool   `json:"subscribed"`
}

type EmailSendJob struct {
	Delivery         EmailDelivery
	Campaign         EmailCampaign
	UnsubscribeToken string `json:"-"`
}

func ValidateEmailCampaign(subject, body, category, group string) error {
	if strings.TrimSpace(subject) == "" || utf8.RuneCountInString(subject) > 200 || strings.ContainsAny(subject, "\r\n\x00") ||
		strings.TrimSpace(body) == "" || len(body) > 20000 || strings.ContainsRune(body, '\x00') ||
		(category != EmailCategoryPromotion && category != EmailCategoryPlatform) || utf8.RuneCountInString(group) > 64 {
		return ErrInvalidEmailCampaign
	}
	return nil
}

// NormalizeCampaignEmail accepts a single bare SMTP address, never a recipient list.
func NormalizeCampaignEmail(value string) (string, error) {
	value = strings.ToLower(strings.Trim(value, " "))
	if value == "" || len(value) > 254 || strings.ContainsAny(value, "\r\n\t\x00") {
		return "", errors.New("invalid email address")
	}
	address, err := mail.ParseAddress(value)
	if err != nil || address.Address != value || address.Name != "" {
		return "", errors.New("invalid email address")
	}
	return value, nil
}

func CreateEmailCampaign(campaign *EmailCampaign) error {
	if err := ValidateEmailCampaign(campaign.Subject, campaign.Body, campaign.Category, campaign.TargetGroup); err != nil {
		return err
	}
	campaign.ID = 0
	campaign.Status = EmailCampaignDraft
	return DB.Create(campaign).Error
}

func UpdateEmailCampaign(id int64, subject, body, category, group string) error {
	if err := ValidateEmailCampaign(subject, body, category, group); err != nil {
		return err
	}
	result := DB.Model(&EmailCampaign{}).Where("id = ? AND status = ?", id, EmailCampaignDraft).Updates(map[string]any{
		"subject": subject, "body": body, "category": category, "target_group": group, "updated_at": time.Now().Unix(),
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		var campaign EmailCampaign
		if err := DB.First(&campaign, id).Error; err != nil {
			return err
		}
		if campaign.Status != EmailCampaignDraft {
			return ErrEmailCampaignNotDraft
		}
	}
	return nil
}

func emailCampaignCounts(db *gorm.DB, campaigns []EmailCampaign) error {
	if len(campaigns) == 0 {
		return nil
	}
	ids := make([]int64, len(campaigns))
	byID := make(map[int64]*EmailDeliveryCounts, len(campaigns))
	for i := range campaigns {
		ids[i] = campaigns[i].ID
		byID[campaigns[i].ID] = &campaigns[i].DeliveryCounts
	}
	var rows []struct {
		CampaignID int64
		Status     string
		Count      int64
	}
	if err := db.Model(&EmailDelivery{}).Select("campaign_id, status, COUNT(*) AS count").Where("campaign_id IN ?", ids).Group("campaign_id, status").Scan(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		counts := byID[row.CampaignID]
		counts.Total += row.Count
		switch row.Status {
		case EmailDeliveryPending:
			counts.Pending = row.Count
		case EmailDeliverySending:
			counts.Sending = row.Count
		case EmailDeliveryRetry:
			counts.Retry = row.Count
		case EmailDeliverySent:
			counts.Sent = row.Count
		case EmailDeliveryFailed:
			counts.Failed = row.Count
		case EmailDeliverySkipped:
			counts.Skipped = row.Count
		case EmailDeliveryUncertain:
			counts.Uncertain = row.Count
		case EmailDeliveryCancelled:
			counts.Cancelled = row.Count
		}
	}
	return nil
}

func GetEmailCampaign(id int64) (*EmailCampaign, error) {
	var campaigns []EmailCampaign
	if err := DB.Where("id = ?", id).Find(&campaigns).Error; err != nil {
		return nil, err
	}
	if len(campaigns) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	if err := emailCampaignCounts(DB, campaigns); err != nil {
		return nil, err
	}
	return &campaigns[0], nil
}

func ListEmailCampaigns(offset, limit int) ([]EmailCampaign, int64, error) {
	var total int64
	if err := DB.Model(&EmailCampaign{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	campaigns := make([]EmailCampaign, 0)
	if err := DB.Order("id DESC").Offset(max(0, offset)).Limit(max(1, min(limit, 100))).Find(&campaigns).Error; err != nil {
		return nil, 0, err
	}
	return campaigns, total, emailCampaignCounts(DB, campaigns)
}

func ListEmailDeliveries(campaignID int64, offset, limit int, status string) ([]EmailDelivery, int64, error) {
	var total int64
	query := DB.Model(&EmailDelivery{}).Where("campaign_id = ?", campaignID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	deliveries := make([]EmailDelivery, 0)
	err := query.Order("id DESC").Offset(max(0, offset)).Limit(max(1, min(limit, 100))).Find(&deliveries).Error
	return deliveries, total, err
}

type emailCampaignRecipient struct {
	UserID int
	Email  string
}

// The database groups recipients so memory usage stays bounded for large audiences.
func emailCampaignAudience(db *gorm.DB, category, group string) *gorm.DB {
	query := db.Table("users").Select("MIN(users.id) AS user_id, LOWER(TRIM(users.email)) AS email").
		Where("users.deleted_at IS NULL AND users.status = ? AND users.email <> ?", common.UserStatusEnabled, "")
	subscription := db.Model(&EmailSubscription{}).Select("1").Where("email_subscriptions.user_id = users.id AND category = ?", category)
	if category == EmailCategoryPromotion {
		query = query.Where("EXISTS (?)", subscription.Where("subscribed = ?", true))
	} else {
		query = query.Where("NOT EXISTS (?)", subscription.Where("subscribed = ?", false))
	}
	if group != "" {
		query = query.Where(clause.Eq{Column: clause.Column{Table: "users", Name: "group"}, Value: group})
	}
	return query.Group("LOWER(TRIM(users.email))").Order("MIN(users.id)")
}

func PreviewEmailCampaignRecipients(category, group string) (int64, error) {
	if category != EmailCategoryPromotion && category != EmailCategoryPlatform {
		return 0, ErrInvalidEmailCampaign
	}
	var total int64
	for offset := 0; ; offset += 250 {
		var recipients []emailCampaignRecipient
		if err := emailCampaignAudience(DB, category, group).Offset(offset).Limit(250).Scan(&recipients).Error; err != nil {
			return 0, err
		}
		for _, recipient := range recipients {
			if _, err := NormalizeCampaignEmail(recipient.Email); err == nil {
				total++
			}
		}
		if len(recipients) < 250 {
			return total, nil
		}
	}
}

func QueueEmailCampaign(id int64) (*EmailCampaign, error) {
	err := DB.Transaction(func(tx *gorm.DB) error {
		// Take the write lock before reading so two SQLite callers cannot enqueue twice.
		result := tx.Model(&EmailCampaign{}).Where("id = ? AND status = ?", id, EmailCampaignDraft).
			Updates(map[string]any{"status": EmailCampaignQueued, "updated_at": time.Now().Unix()})
		if result.Error != nil {
			return result.Error
		}
		var campaign EmailCampaign
		if err := lockForUpdate(tx).First(&campaign, id).Error; err != nil {
			return err
		}
		if result.RowsAffected == 0 {
			return nil
		}
		var inserted int64
		for offset := 0; ; offset += 250 {
			var recipients []emailCampaignRecipient
			if err := emailCampaignAudience(tx, campaign.Category, campaign.TargetGroup).Offset(offset).Limit(250).Scan(&recipients).Error; err != nil {
				return err
			}
			deliveries := make([]EmailDelivery, 0, len(recipients))
			for _, recipient := range recipients {
				email, err := NormalizeCampaignEmail(recipient.Email)
				if err != nil {
					continue
				}
				digest := sha256.Sum256([]byte(email))
				deliveries = append(deliveries, EmailDelivery{CampaignID: id, UserID: recipient.UserID, Email: email, EmailHash: hex.EncodeToString(digest[:]), Status: EmailDeliveryPending})
			}
			if len(deliveries) > 0 {
				result := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "campaign_id"}, {Name: "email_hash"}}, DoNothing: true}).Create(&deliveries)
				if result.Error != nil {
					return result.Error
				}
				inserted += result.RowsAffected
			}
			if len(recipients) < 250 {
				break
			}
		}
		if inserted == 0 {
			return ErrEmailAudienceEmpty
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return GetEmailCampaign(id)
}

func CancelEmailCampaign(id int64) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&EmailCampaign{}).Where("id = ? AND status IN ?", id, []string{EmailCampaignDraft, EmailCampaignQueued, EmailCampaignSending}).Updates(map[string]any{"status": EmailCampaignCancelled, "updated_at": time.Now().Unix()})
		if result.Error != nil {
			return result.Error
		}
		var campaign EmailCampaign
		if err := tx.First(&campaign, id).Error; err != nil {
			return err
		}
		if campaign.Status != EmailCampaignCancelled {
			return nil
		}
		return tx.Model(&EmailDelivery{}).Where("campaign_id = ? AND status IN ?", id, []string{EmailDeliveryPending, EmailDeliveryRetry}).
			Updates(map[string]any{"status": EmailDeliveryCancelled, "updated_at": time.Now().Unix()}).Error
	})
}

func GetEmailPreferences(userID int) (*EmailPreferences, error) {
	preferences := &EmailPreferences{Platform: true}
	var subscriptions []EmailSubscription
	if err := DB.Where("user_id = ?", userID).Find(&subscriptions).Error; err != nil {
		return nil, err
	}
	for _, subscription := range subscriptions {
		switch subscription.Category {
		case EmailCategoryPromotion:
			preferences.Promotion = subscription.Subscribed
		case EmailCategoryPlatform:
			preferences.Platform = subscription.Subscribed
		}
	}
	return preferences, nil
}

func ensureEmailSubscription(tx *gorm.DB, userID int, category string) (*EmailSubscription, error) {
	// These queries contain a bearer unsubscribe token; keep it out of debug SQL too.
	tx = tx.Session(&gorm.Session{Logger: logger.Discard})
	var token [32]byte
	if _, err := rand.Read(token[:]); err != nil {
		return nil, err
	}
	subscription := EmailSubscription{UserID: userID, Category: category, Subscribed: category == EmailCategoryPlatform, UnsubscribeToken: hex.EncodeToString(token[:])}
	if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}, {Name: "category"}}, DoNothing: true}).Create(&subscription).Error; err != nil {
		return nil, sanitizeDBError(err)
	}
	// A current locking read observes a concurrent unsubscribe even under MySQL's
	// repeatable-read isolation, and preserves the existing token on conflict.
	subscription = EmailSubscription{}
	err := lockForUpdate(tx).Where("user_id = ? AND category = ?", userID, category).First(&subscription).Error
	return &subscription, sanitizeDBError(err)
}

func SetEmailPreferences(userID int, promotion, platform bool) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var user User
		if err := tx.Select("id").First(&user, userID).Error; err != nil {
			return err
		}
		for _, preference := range []struct {
			category   string
			subscribed bool
		}{{EmailCategoryPromotion, promotion}, {EmailCategoryPlatform, platform}} {
			subscription, err := ensureEmailSubscription(tx, userID, preference.category)
			if err != nil {
				return err
			}
			if err := tx.Model(subscription).Updates(map[string]any{"subscribed": preference.subscribed, "updated_at": time.Now().Unix()}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func GetEmailUnsubscribeInfo(token string) (*EmailUnsubscribeInfo, error) {
	if len(token) != 64 {
		return nil, gorm.ErrRecordNotFound
	}
	var subscription EmailSubscription
	if err := DB.Session(&gorm.Session{Logger: logger.Discard}).Where("unsubscribe_token = ?", token).First(&subscription).Error; err != nil {
		return nil, sanitizeDBError(err)
	}
	return &EmailUnsubscribeInfo{Category: subscription.Category, Subscribed: subscription.Subscribed}, nil
}

func UnsubscribeEmail(token string) error {
	if len(token) != 64 {
		return gorm.ErrRecordNotFound
	}
	result := DB.Session(&gorm.Session{Logger: logger.Discard}).Model(&EmailSubscription{}).Where("unsubscribe_token = ?", token).Updates(map[string]any{"subscribed": false, "updated_at": time.Now().Unix()})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		_, err := GetEmailUnsubscribeInfo(token)
		return err
	}
	return nil
}

func GetEmailDispatchConfig() (*EmailDispatchConfig, error) {
	config := EmailDispatchConfig{ID: 1, RatePerMinute: 20}
	if err := DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&config).Error; err != nil {
		return nil, err
	}
	if err := DB.First(&config, 1).Error; err != nil {
		return nil, err
	}
	return &config, nil
}

func UpdateEmailDispatchRate(rate int) error {
	if rate < 1 || rate > 120 {
		return errors.New("email rate must be between 1 and 120 per minute")
	}
	if _, err := GetEmailDispatchConfig(); err != nil {
		return err
	}
	return DB.Model(&EmailDispatchConfig{}).Where("id = ?", 1).Update("rate_per_minute", rate).Error
}

func lockEmailCampaign(tx *gorm.DB, id int64) (*EmailCampaign, error) {
	// Acquiring a write lock before any reads also works for SQLite transactions.
	if err := tx.Model(&EmailCampaign{}).Where("id = ?", id).Update("updated_at", time.Now().Unix()).Error; err != nil {
		return nil, err
	}
	var campaign EmailCampaign
	if err := lockForUpdate(tx).First(&campaign, id).Error; err != nil {
		return nil, err
	}
	return &campaign, nil
}

func completeEmailCampaign(tx *gorm.DB, id int64) error {
	var active EmailDelivery
	err := lockForUpdate(tx).Select("id").Where("campaign_id = ? AND status IN ?", id, []string{EmailDeliveryPending, EmailDeliverySending, EmailDeliveryRetry}).Take(&active).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return tx.Model(&EmailCampaign{}).Where("id = ? AND status IN ?", id, []string{EmailCampaignQueued, EmailCampaignSending}).
		Updates(map[string]any{"status": EmailCampaignCompleted, "updated_at": time.Now().Unix()}).Error
}

func HasPendingEmailDeliveries() bool {
	var delivery EmailDelivery
	err := DB.Select("id").Where("status IN ?", []string{EmailDeliveryPending, EmailDeliveryRetry, EmailDeliverySending}).First(&delivery).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false
	}
	if err != nil {
		common.SysError("failed to check email delivery queue: " + sanitizeDBError(err).Error())
		return false
	}
	return true
}

// campaignMessageIDDomain derives the Message-ID domain from the configured
// sender address. Message-IDs on unresolvable placeholder domains (e.g.
// .local) are treated as spam signals by Gmail, QQ Mail and NetEase, so the
// campaign queue must reuse the same domain the SMTP envelope already uses.
func campaignMessageIDDomain() string {
	from := common.SMTPFrom
	if from == "" {
		from = common.SMTPAccount
	}
	if at := strings.LastIndex(from, "@"); at >= 0 && at+1 < len(from) {
		return from[at+1:]
	}
	return "notifications.local"
}

// ClaimEmailDelivery persists the global pacing reservation in the same transaction
// as the lease. A stale sending lease is ambiguous and must never auto-retry.
// nowMS, leaseMS, NextAttemptAt and LockedUntil are milliseconds.
func ClaimEmailDelivery(workerID string, nowMS, leaseMS int64) (*EmailDelivery, error) {
	if workerID == "" || len(workerID) > 64 || leaseMS <= 0 || leaseMS > 3600000 {
		return nil, errors.New("invalid email worker lease")
	}
	if _, err := GetEmailDispatchConfig(); err != nil {
		return nil, err
	}
	var claimed *EmailDelivery
	err := DB.Transaction(func(tx *gorm.DB) error {
		// A conditional write serializes SQLite as well as row-locking databases.
		result := tx.Model(&EmailDispatchConfig{}).Where("id = ? AND next_send_at <= ?", 1, nowMS).Update("next_send_at", nowMS+1)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		var config EmailDispatchConfig
		if err := lockForUpdate(tx).First(&config, 1).Error; err != nil {
			return err
		}
		var stale []EmailDelivery
		if err := tx.Where("status = ? AND locked_until <= ?", EmailDeliverySending, nowMS).Limit(250).Find(&stale).Error; err != nil {
			return err
		}
		for _, delivery := range stale {
			if _, err := lockEmailCampaign(tx, delivery.CampaignID); err != nil {
				return err
			}
			if err := tx.Model(&EmailDelivery{}).Where("id = ? AND status = ? AND locked_until <= ?", delivery.ID, EmailDeliverySending, nowMS).
				Updates(map[string]any{"status": EmailDeliveryUncertain, "locked_by": "", "locked_until": 0, "last_error": "Worker lease expired; SMTP acceptance is unknown. Automatic retry is disabled.", "updated_at": nowMS / 1000}).Error; err != nil {
				return err
			}
			if err := completeEmailCampaign(tx, delivery.CampaignID); err != nil {
				return err
			}
		}
		var delivery EmailDelivery
		err := tx.Where("status IN ? AND next_attempt_at <= ?", []string{EmailDeliveryPending, EmailDeliveryRetry}, nowMS).Order("id").First(&delivery).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// The +1 reservation above acquires the lock, but no SMTP slot was used.
			// Keeping it in the future would make future retries trigger 1 ms polling.
			return tx.Model(&config).Update("next_send_at", nowMS).Error
		}
		if err != nil {
			return err
		}
		if _, err := lockEmailCampaign(tx, delivery.CampaignID); err != nil {
			return err
		}
		messageID := delivery.MessageID
		if messageID == "" {
			var random [16]byte
			if _, err := rand.Read(random[:]); err != nil {
				return err
			}
			messageID = fmt.Sprintf("<new-api-%d-%s@%s>", delivery.ID, hex.EncodeToString(random[:]), campaignMessageIDDomain())
		}
		result = tx.Model(&EmailDelivery{}).Where("id = ? AND status IN ?", delivery.ID, []string{EmailDeliveryPending, EmailDeliveryRetry}).Updates(map[string]any{
			"status": EmailDeliverySending, "attempts": gorm.Expr("attempts + 1"), "locked_by": workerID, "locked_until": nowMS + leaseMS, "message_id": messageID, "updated_at": nowMS / 1000,
		})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return tx.Model(&config).Update("next_send_at", nowMS).Error
		}
		rate := max(1, min(120, config.RatePerMinute))
		interval := (60000 + rate - 1) / rate
		if err := tx.Model(&config).Update("next_send_at", nowMS+int64(interval)).Error; err != nil {
			return err
		}
		if err := tx.Model(&EmailCampaign{}).Where("id = ? AND status = ?", delivery.CampaignID, EmailCampaignQueued).
			Updates(map[string]any{"status": EmailCampaignSending, "updated_at": nowMS / 1000}).Error; err != nil {
			return err
		}
		if err := tx.First(&delivery, delivery.ID).Error; err != nil {
			return err
		}
		claimed = &delivery
		return nil
	})
	return claimed, err
}

func PrepareEmailDelivery(id int64, workerID string) (*EmailSendJob, error) {
	var identity EmailDelivery
	if err := DB.Select("campaign_id").First(&identity, id).Error; err != nil {
		return nil, err
	}
	var job *EmailSendJob
	err := DB.Transaction(func(tx *gorm.DB) error {
		campaign, err := lockEmailCampaign(tx, identity.CampaignID)
		if err != nil {
			return err
		}
		var delivery EmailDelivery
		if err := lockForUpdate(tx).Where("id = ? AND status = ? AND locked_by = ?", id, EmailDeliverySending, workerID).First(&delivery).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrEmailDeliveryLeaseLost
			}
			return err
		}
		if delivery.LockedUntil <= time.Now().UnixMilli() {
			return ErrEmailDeliveryLeaseLost
		}
		status, reason := "", ""
		if campaign.Status == EmailCampaignCancelled {
			status, reason = EmailDeliveryCancelled, "Campaign cancelled"
		}
		var user User
		err = lockForUpdate(tx).Select("id", "email", "status").First(&user, delivery.UserID).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		email, emailErr := NormalizeCampaignEmail(user.Email)
		if status == "" && (errors.Is(err, gorm.ErrRecordNotFound) || user.Status != common.UserStatusEnabled || emailErr != nil || email != delivery.Email) {
			status, reason = EmailDeliverySkipped, "Recipient account or email address changed"
		}
		var subscription *EmailSubscription
		if status == "" {
			subscription, err = ensureEmailSubscription(tx, delivery.UserID, campaign.Category)
			if err != nil {
				return err
			}
			if !subscription.Subscribed {
				status, reason = EmailDeliverySkipped, "Recipient unsubscribed"
			}
		}
		if status != "" {
			result := tx.Model(&EmailDelivery{}).Where("id = ? AND status = ? AND locked_by = ?", id, EmailDeliverySending, workerID).
				Updates(map[string]any{"status": status, "last_error": reason, "locked_by": "", "locked_until": 0, "updated_at": time.Now().Unix()})
			if result.Error != nil {
				return result.Error
			}
			return completeEmailCampaign(tx, delivery.CampaignID)
		}
		job = &EmailSendJob{Delivery: delivery, Campaign: *campaign, UnsubscribeToken: subscription.UnsubscribeToken}
		return nil
	})
	return job, err
}

func FinishEmailDelivery(id int64, workerID, status, lastError string, nextAttemptMS int64) error {
	switch status {
	case EmailDeliverySent, EmailDeliveryFailed, EmailDeliveryRetry, EmailDeliveryUncertain:
	default:
		return errors.New("invalid email delivery result")
	}
	if utf8.RuneCountInString(lastError) > 500 {
		lastError = string([]rune(lastError)[:500])
	}
	var delivery EmailDelivery
	if err := DB.First(&delivery, id).Error; err != nil {
		return err
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		campaign, err := lockEmailCampaign(tx, delivery.CampaignID)
		if err != nil {
			return err
		}
		updates := map[string]any{"status": status, "last_error": lastError, "next_attempt_at": nextAttemptMS, "locked_by": "", "locked_until": 0, "updated_at": time.Now().Unix()}
		if status == EmailDeliverySent {
			updates["sent_at"] = time.Now().Unix()
		}
		if status == EmailDeliveryRetry {
			if campaign.Status == EmailCampaignCancelled {
				updates["status"] = EmailDeliveryCancelled
			}
		}
		result := tx.Model(&EmailDelivery{}).Where("id = ? AND status = ? AND locked_by = ?", id, EmailDeliverySending, workerID).Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrEmailDeliveryLeaseLost
		}
		return completeEmailCampaign(tx, delivery.CampaignID)
	})
}
