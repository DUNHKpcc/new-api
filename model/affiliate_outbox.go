package model

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	AffiliateOutboxStatusPending    = "pending"
	AffiliateOutboxStatusProcessing = "processing"
	AffiliateOutboxStatusDelivered  = "delivered"
)

var ErrAffiliateOutboxConflict = errors.New("affiliate outbox event conflicts with an existing event")

// AffiliateOutboxEvent is written in the main database transaction that owns
// the business fact. It bridges user/admin logs without pretending LOG_DB can
// participate in the same transaction.
type AffiliateOutboxEvent struct {
	Id          int64  `json:"id" gorm:"primaryKey"`
	EventId     string `json:"event_id" gorm:"type:varchar(40);not null;uniqueIndex"`
	DedupKey    string `json:"dedup_key" gorm:"type:varchar(191);not null;uniqueIndex"`
	UserId      int    `json:"user_id" gorm:"not null;index"`
	Action      string `json:"action" gorm:"type:varchar(64);not null;index"`
	LogType     int    `json:"log_type" gorm:"not null"`
	Content     string `json:"content" gorm:"type:text;not null"`
	Payload     string `json:"payload" gorm:"type:text;not null"`
	Status      string `json:"status" gorm:"type:varchar(16);not null;index"`
	Attempts    int    `json:"attempts" gorm:"not null"`
	LockedBy    string `json:"locked_by" gorm:"type:varchar(64);not null;index"`
	LockedUntil int64  `json:"locked_until" gorm:"not null;index"`
	LastError   string `json:"last_error" gorm:"type:varchar(512);not null"`
	CreatedAt   int64  `json:"created_at" gorm:"not null;autoCreateTime;index"`
	DeliveredAt int64  `json:"delivered_at" gorm:"not null"`
}

func (AffiliateOutboxEvent) TableName() string { return "affiliate_outbox_events" }

func affiliateEventId(dedupKey string) string {
	digest := sha256.Sum256([]byte(dedupKey))
	return "aff_" + hex.EncodeToString(digest[:16])
}

func EnqueueAffiliateEventWithTx(
	tx *gorm.DB,
	dedupKey string,
	userId int,
	action string,
	logType int,
	content string,
	payload any,
	createdAt int64,
) (*AffiliateOutboxEvent, error) {
	dedupKey = strings.TrimSpace(dedupKey)
	action = strings.TrimSpace(action)
	content = strings.TrimSpace(content)
	if tx == nil || dedupKey == "" || len(dedupKey) > 191 || userId <= 0 ||
		action == "" || len(action) > 64 || content == "" {
		return nil, ErrAffiliateOutboxConflict
	}
	payloadJSON, err := common.Marshal(payload)
	if err != nil {
		return nil, err
	}
	if createdAt <= 0 {
		createdAt = common.GetTimestamp()
	}
	event := AffiliateOutboxEvent{
		EventId:   affiliateEventId(dedupKey),
		DedupKey:  dedupKey,
		UserId:    userId,
		Action:    action,
		LogType:   logType,
		Content:   content,
		Payload:   string(payloadJSON),
		Status:    AffiliateOutboxStatusPending,
		CreatedAt: createdAt,
	}
	create := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&event)
	if create.Error != nil {
		return nil, create.Error
	}
	if create.RowsAffected == 1 {
		return &event, nil
	}

	var existing AffiliateOutboxEvent
	if err := tx.Where("dedup_key = ?", dedupKey).First(&existing).Error; err != nil {
		return nil, err
	}
	if existing.EventId != event.EventId || existing.UserId != userId || existing.Action != action ||
		existing.LogType != logType || existing.Content != content || existing.Payload != event.Payload {
		return nil, ErrAffiliateOutboxConflict
	}
	return &existing, nil
}

func claimAffiliateOutboxEvent(workerId string, now int64) (*AffiliateOutboxEvent, error) {
	var claimed AffiliateOutboxEvent
	err := DB.Transaction(func(tx *gorm.DB) error {
		query := lockForUpdate(tx).
			Where(
				"status = ? OR (status = ? AND locked_until < ?)",
				AffiliateOutboxStatusPending,
				AffiliateOutboxStatusProcessing,
				now,
			).
			Order("id asc").
			First(&claimed)
		if query.Error != nil {
			return query.Error
		}
		update := tx.Model(&AffiliateOutboxEvent{}).
			Where("id = ? AND (status = ? OR (status = ? AND locked_until < ?))", claimed.Id, AffiliateOutboxStatusPending, AffiliateOutboxStatusProcessing, now).
			Updates(map[string]interface{}{
				"status":       AffiliateOutboxStatusProcessing,
				"locked_by":    workerId,
				"locked_until": now + 60,
				"attempts":     gorm.Expr("attempts + 1"),
			})
		if update.Error != nil {
			return update.Error
		}
		if update.RowsAffected != 1 {
			return ErrAffiliateOutboxConflict
		}
		claimed.Status = AffiliateOutboxStatusProcessing
		claimed.LockedBy = workerId
		claimed.LockedUntil = now + 60
		claimed.Attempts++
		return nil
	})
	return &claimed, err
}

func deliverAffiliateOutboxEvent(event *AffiliateOutboxEvent) error {
	if event == nil || event.Id <= 0 || event.EventId == "" || event.LockedBy == "" {
		return ErrAffiliateOutboxConflict
	}
	var existing Log
	err := LOG_DB.Select("id").Where("request_id = ?", event.EventId).First(&existing).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		logEntry := Log{
			UserId:    event.UserId,
			CreatedAt: event.CreatedAt,
			Type:      event.LogType,
			Content:   event.Content,
			RequestId: event.EventId,
			Other:     event.Payload,
		}
		if err := createLog(&logEntry); err != nil {
			return err
		}
	}
	result := DB.Model(&AffiliateOutboxEvent{}).
		Where("id = ? AND status = ? AND locked_by = ?", event.Id, AffiliateOutboxStatusProcessing, event.LockedBy).
		Updates(map[string]interface{}{
			"status":       AffiliateOutboxStatusDelivered,
			"locked_by":    "",
			"locked_until": 0,
			"last_error":   "",
			"delivered_at": common.GetTimestamp(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrAffiliateOutboxConflict
	}
	return nil
}

func releaseAffiliateOutboxEvent(event *AffiliateOutboxEvent, deliveryErr error) {
	if event == nil || event.Id <= 0 || event.LockedBy == "" {
		return
	}
	message := ""
	if deliveryErr != nil {
		message = deliveryErr.Error()
		if len(message) > 512 {
			message = message[:512]
		}
	}
	_ = DB.Model(&AffiliateOutboxEvent{}).
		Where("id = ? AND status = ? AND locked_by = ?", event.Id, AffiliateOutboxStatusProcessing, event.LockedBy).
		Updates(map[string]interface{}{
			"status":       AffiliateOutboxStatusProcessing,
			"locked_by":    "",
			"locked_until": common.GetTimestamp() + 60,
			"last_error":   message,
		}).Error
}

// DispatchAffiliateOutbox delivers a bounded batch. Claim leases keep multiple
// application instances from concurrently writing the same event, while the
// stable request_id lets a retry detect a log written immediately before a
// process crash.
func DispatchAffiliateOutbox(limit int) (int, int, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	workerKey, err := common.GenerateRandomCharsKey(24)
	if err != nil {
		return 0, 0, err
	}
	workerId := "affout_" + workerKey
	processed := 0
	failed := 0
	for processed+failed < limit {
		event, err := claimAffiliateOutboxEvent(workerId, common.GetTimestamp())
		if errors.Is(err, gorm.ErrRecordNotFound) {
			break
		}
		if err != nil {
			return processed, failed, err
		}
		if err := deliverAffiliateOutboxEvent(event); err != nil {
			failed++
			releaseAffiliateOutboxEvent(event, err)
			common.SysError(fmt.Sprintf("affiliate outbox delivery failed: event_id=%s action=%s error=%v", event.EventId, event.Action, err))
			continue
		}
		processed++
	}
	return processed, failed, nil
}

func HasPendingAffiliateWork() bool {
	now := common.GetTimestamp()
	var count int64
	if err := DB.Model(&AffiliateOutboxEvent{}).
		Where(
			"status = ? OR (status = ? AND locked_until < ?)",
			AffiliateOutboxStatusPending,
			AffiliateOutboxStatusProcessing,
			now,
		).
		Limit(1).
		Count(&count).Error; err != nil {
		common.SysError("failed to inspect affiliate outbox work: " + err.Error())
		return false
	}
	if count > 0 {
		return true
	}
	if err := DB.Model(&AffiliateCommission{}).
		Where("status = ? AND available_at <= ?", AffiliateCommissionStatusPending, now).
		Limit(1).
		Count(&count).Error; err != nil {
		common.SysError("failed to inspect due affiliate commissions: " + err.Error())
		return false
	}
	return count > 0
}
