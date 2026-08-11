package model

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestAffiliateOutboxRecoversDeliveryWithoutDuplicateLog(t *testing.T) {
	truncateTables(t)
	user := User{Username: "affiliate-outbox", AffCode: "affiliate-outbox"}
	require.NoError(t, DB.Create(&user).Error)
	var event *AffiliateOutboxEvent
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		var err error
		event, err = EnqueueAffiliateEventWithTx(
			tx,
			"affiliate.test:1",
			user.Id,
			"affiliate.test",
			LogTypeSystem,
			"Affiliate test event",
			map[string]interface{}{"value": "1"},
			100,
		)
		return err
	}))
	require.NotNil(t, event)

	previousLogDB := LOG_DB
	missingLogDB, err := gorm.Open(sqlite.Open("file:affiliate-outbox-missing-log?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	LOG_DB = missingLogDB
	processed, failed, err := DispatchAffiliateOutbox(1)
	require.NoError(t, err)
	assert.Zero(t, processed)
	assert.Equal(t, 1, failed)
	LOG_DB = previousLogDB
	t.Cleanup(func() { LOG_DB = previousLogDB })

	var retry AffiliateOutboxEvent
	require.NoError(t, DB.First(&retry, event.Id).Error)
	assert.Equal(t, AffiliateOutboxStatusProcessing, retry.Status)
	assert.NotEmpty(t, retry.LastError)
	require.NoError(t, DB.Model(&AffiliateOutboxEvent{}).
		Where("id = ?", event.Id).
		Updates(map[string]interface{}{"locked_until": 0}).Error)

	processed, failed, err = DispatchAffiliateOutbox(10)
	require.NoError(t, err)
	assert.Equal(t, 1, processed)
	assert.Zero(t, failed)
	processed, failed, err = DispatchAffiliateOutbox(10)
	require.NoError(t, err)
	assert.Zero(t, processed)
	assert.Zero(t, failed)

	var logCount int64
	require.NoError(t, DB.Model(&Log{}).Where("request_id = ?", event.EventId).Count(&logCount).Error)
	assert.Equal(t, int64(1), logCount)
	require.NoError(t, DB.First(&retry, event.Id).Error)
	assert.Equal(t, AffiliateOutboxStatusDelivered, retry.Status)
}

func TestAffiliateOutboxDedupRejectsPayloadMismatch(t *testing.T) {
	truncateTables(t)
	user := User{Username: "affiliate-outbox-dedup", AffCode: "affiliate-outbox-dedup"}
	require.NoError(t, DB.Create(&user).Error)
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		_, err := EnqueueAffiliateEventWithTx(
			tx, "affiliate.test:dedup", user.Id, "affiliate.test", LogTypeSystem,
			"Affiliate test event", map[string]interface{}{"value": "1"}, 100,
		)
		return err
	}))
	err := DB.Transaction(func(tx *gorm.DB) error {
		_, err := EnqueueAffiliateEventWithTx(
			tx, "affiliate.test:dedup", user.Id, "affiliate.test", LogTypeSystem,
			"Affiliate test event", map[string]interface{}{"value": "2"}, 100,
		)
		return err
	})
	assert.ErrorIs(t, err, ErrAffiliateOutboxConflict)
	var count int64
	require.NoError(t, DB.Model(&AffiliateOutboxEvent{}).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}
