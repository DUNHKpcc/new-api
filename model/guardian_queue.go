package model

import (
	"errors"
	"slices"
	"time"

	"github.com/QuantumNous/new-api/setting/operation_setting"
	"gorm.io/gorm"
)

// Orders paid before queue ordering existed keep queue_seq 0 and sort by payment time.
var guardianWaitingStatuses = []string{"paid", "scheduled"}

const guardianWaitingOrder = "queue_seq, paid_at, id"

type GuardianQueueSlot struct {
	Ahead     int    `json:"ahead"`
	StartDate string `json:"start_date"`
}

// GuardianQueue is the delivery queue as buyers see it: placeholder slots and
// orders in progress come first, then waiting orders in their queue order.
type GuardianQueue struct {
	Setting      operation_setting.GuardianQueueSetting
	Paused       bool
	StartedToday int
	InProgress   []GuardianServiceOrder
	Waiting      []GuardianServiceOrder
	Slots        map[int]GuardianQueueSlot
}

func LoadGuardianQueue(setting operation_setting.GuardianQueueSetting, now time.Time) (*GuardianQueue, error) {
	queue := &GuardianQueue{Setting: setting, Paused: setting.PausedAt(now.Unix()), Slots: map[int]GuardianQueueSlot{}}
	if err := DB.Where("status = ?", "in_progress").Order("id").Find(&queue.InProgress).Error; err != nil {
		return nil, err
	}
	if err := DB.Where("status IN ?", guardianWaitingStatuses).Order(guardianWaitingOrder).Find(&queue.Waiting).Error; err != nil {
		return nil, err
	}
	year, month, day := now.Date()
	today := time.Date(year, month, day, 0, 0, 0, 0, now.Location())
	var started int64
	if err := DB.Model(&GuardianOrderEvent{}).Where("after_status = ? AND before_status <> ? AND created_at >= ?", "in_progress", "in_progress", today.Unix()).Distinct("order_id").Count(&started).Error; err != nil {
		return nil, err
	}
	queue.StartedToday = int(started)
	for i, order := range queue.Waiting {
		queue.Slots[order.ID] = GuardianQueueSlot{Ahead: setting.ExtraSlots + len(queue.InProgress) + i, StartDate: queue.startDate(setting.ExtraSlots+i, today)}
	}
	return queue, nil
}

// startDate places the given upcoming start (0-based, placeholders first) on the
// first day with capacity left. Without a daily limit only a pause yields a date.
func (q *GuardianQueue) startDate(upcoming int, today time.Time) string {
	capacity := q.Setting.DailyCapacity
	first, remaining := today, capacity-q.StartedToday
	if q.Paused {
		if q.Setting.ResumeAt == 0 {
			return ""
		}
		year, month, day := time.Unix(q.Setting.ResumeAt, 0).In(today.Location()).Date()
		if resumeDay := time.Date(year, month, day, 0, 0, 0, 0, today.Location()); resumeDay.After(today) {
			first, remaining = resumeDay, capacity
		}
	}
	if capacity <= 0 {
		if q.Paused {
			return first.Format(time.DateOnly)
		}
		return ""
	}
	remaining = max(0, remaining)
	if upcoming < remaining {
		return first.Format(time.DateOnly)
	}
	return first.AddDate(0, 0, 1+(upcoming-remaining)/capacity).Format(time.DateOnly)
}

// MoveGuardianQueueOrder moves a waiting order to a 1-based position among waiting
// orders; orders already in progress keep their place.
func MoveGuardianQueueOrder(id, position int) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var waiting []GuardianServiceOrder
		if err := lockForUpdate(tx).Where("status IN ?", guardianWaitingStatuses).Order(guardianWaitingOrder).Find(&waiting).Error; err != nil {
			return err
		}
		from := slices.IndexFunc(waiting, func(order GuardianServiceOrder) bool { return order.ID == id })
		if from < 0 {
			return errors.New("只能调整排队中的订单")
		}
		order := waiting[from]
		waiting = slices.Delete(waiting, from, from+1)
		waiting = slices.Insert(waiting, min(max(position, 1), len(waiting)+1)-1, order)
		for i, item := range waiting {
			if item.QueueSeq == int64(i+1) {
				continue
			}
			if err := tx.Model(&GuardianServiceOrder{}).Where("id = ?", item.ID).UpdateColumn("queue_seq", i+1).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
