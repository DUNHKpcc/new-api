package model

import (
	"errors"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	ExternalRevenueStatusActive = "active"
	ExternalRevenueStatusVoided = "voided"
)

var (
	ErrExternalRevenueNotFound        = errors.New("external revenue record not found")
	ErrExternalRevenueDuplicate       = errors.New("external revenue order already exists")
	ErrExternalRevenueVersionConflict = errors.New("external revenue record was changed")
	ErrExternalRevenueVoided          = errors.New("external revenue record is voided")
)

// ExternalRevenueRecord is reporting-only income entered by an administrator.
// It intentionally has no relationship to users, quota, top-ups, or settlements.
type ExternalRevenueRecord struct {
	Id              int64   `json:"id"`
	Source          string  `json:"source" gorm:"type:varchar(32);not null;index;uniqueIndex:idx_external_revenue_source_order,priority:1"`
	SourceLabel     string  `json:"source_label" gorm:"type:varchar(64);not null"`
	ExternalOrderNo *string `json:"external_order_no" gorm:"type:varchar(128);uniqueIndex:idx_external_revenue_source_order,priority:2"`
	AmountMinor     int64   `json:"amount_minor,string" gorm:"not null"`
	Currency        string  `json:"currency" gorm:"type:varchar(3);not null;index"`
	OccurredAt      int64   `json:"occurred_at" gorm:"not null;index"`
	Note            string  `json:"note" gorm:"type:varchar(500);not null"`
	Status          string  `json:"status" gorm:"type:varchar(16);not null;index"`
	Version         int64   `json:"version" gorm:"not null"`
	CreatedBy       int     `json:"created_by" gorm:"not null"`
	UpdatedBy       int     `json:"updated_by" gorm:"not null"`
	VoidedBy        int     `json:"voided_by" gorm:"not null"`
	CreateTime      int64   `json:"create_time" gorm:"not null"`
	UpdateTime      int64   `json:"update_time" gorm:"not null"`
	VoidTime        int64   `json:"void_time" gorm:"not null"`
}

type ExternalRevenueListFilter struct {
	Source    string
	Currency  string
	Status    string
	Keyword   string
	StartTime int64
	EndTime   int64
}

type ExternalRevenueTotal struct {
	Currency    string `json:"currency"`
	AmountMinor int64  `json:"amount_minor,string"`
	Count       int64  `json:"count"`
}

type ExternalRevenueSourceTotal struct {
	Source      string `json:"source"`
	SourceLabel string `json:"source_label"`
	Currency    string `json:"currency"`
	AmountMinor int64  `json:"amount_minor,string"`
	Count       int64  `json:"count"`
}

type ExternalRevenueTimelinePoint struct {
	Date        string `json:"date"`
	Source      string `json:"source"`
	SourceLabel string `json:"source_label"`
	Currency    string `json:"currency"`
	AmountMinor int64  `json:"amount_minor,string"`
	Count       int64  `json:"count"`
}

type ExternalRevenueSummary struct {
	Totals    []ExternalRevenueTotal         `json:"totals"`
	BySource  []ExternalRevenueSourceTotal   `json:"by_source"`
	Timeline  []ExternalRevenueTimelinePoint `json:"timeline"`
	StartTime int64                          `json:"start_time"`
	EndTime   int64                          `json:"end_time"`
	Timezone  int                            `json:"timezone_offset"`
}

func externalRevenueOrderExists(tx *gorm.DB, source string, orderNo *string, excludeId int64) (bool, error) {
	if orderNo == nil {
		return false, nil
	}
	query := tx.Model(&ExternalRevenueRecord{}).Where("source = ? AND external_order_no = ?", source, *orderNo)
	if excludeId > 0 {
		query = query.Where("id <> ?", excludeId)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func CreateExternalRevenue(record *ExternalRevenueRecord) error {
	exists, err := externalRevenueOrderExists(DB, record.Source, record.ExternalOrderNo, 0)
	if err != nil {
		return err
	}
	if exists {
		return ErrExternalRevenueDuplicate
	}
	return DB.Create(record).Error
}

func ListExternalRevenue(pageInfo *common.PageInfo, filter ExternalRevenueListFilter) ([]*ExternalRevenueRecord, int64, error) {
	query := DB.Model(&ExternalRevenueRecord{})
	if filter.Source != "" {
		query = query.Where("source = ?", filter.Source)
	}
	if filter.Currency != "" {
		query = query.Where("currency = ?", filter.Currency)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.StartTime > 0 {
		query = query.Where("occurred_at >= ?", filter.StartTime)
	}
	if filter.EndTime > 0 {
		query = query.Where("occurred_at < ?", filter.EndTime)
	}
	if filter.Keyword != "" {
		pattern, err := sanitizeLikePattern(filter.Keyword)
		if err != nil {
			return nil, 0, err
		}
		query = query.Where("external_order_no LIKE ? ESCAPE '!' OR note LIKE ? ESCAPE '!'", pattern, pattern)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var records []*ExternalRevenueRecord
	if err := query.Order("occurred_at desc, id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&records).Error; err != nil {
		return nil, 0, err
	}
	return records, total, nil
}

func UpdateExternalRevenue(record *ExternalRevenueRecord, expectedVersion int64) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		current := &ExternalRevenueRecord{}
		if err := lockForUpdate(tx).First(current, record.Id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrExternalRevenueNotFound
			}
			return err
		}
		if current.Status == ExternalRevenueStatusVoided {
			return ErrExternalRevenueVoided
		}
		if current.Version != expectedVersion {
			return ErrExternalRevenueVersionConflict
		}
		exists, err := externalRevenueOrderExists(tx, record.Source, record.ExternalOrderNo, record.Id)
		if err != nil {
			return err
		}
		if exists {
			return ErrExternalRevenueDuplicate
		}
		updates := map[string]interface{}{
			"source":            record.Source,
			"source_label":      record.SourceLabel,
			"external_order_no": record.ExternalOrderNo,
			"amount_minor":      record.AmountMinor,
			"currency":          record.Currency,
			"occurred_at":       record.OccurredAt,
			"note":              record.Note,
			"updated_by":        record.UpdatedBy,
			"update_time":       record.UpdateTime,
			"version":           expectedVersion + 1,
		}
		result := tx.Model(current).Where("version = ?", expectedVersion).Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrExternalRevenueVersionConflict
		}
		record.Status = current.Status
		record.CreatedBy = current.CreatedBy
		record.CreateTime = current.CreateTime
		record.VoidedBy = current.VoidedBy
		record.VoidTime = current.VoidTime
		record.Version = expectedVersion + 1
		return nil
	})
}

func VoidExternalRevenue(id int64, expectedVersion int64, adminId int, now int64) error {
	result := DB.Model(&ExternalRevenueRecord{}).
		Where("id = ? AND version = ? AND status = ?", id, expectedVersion, ExternalRevenueStatusActive).
		Updates(map[string]interface{}{
			"status":      ExternalRevenueStatusVoided,
			"voided_by":   adminId,
			"updated_by":  adminId,
			"void_time":   now,
			"update_time": now,
			"version":     expectedVersion + 1,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 1 {
		return nil
	}
	current := &ExternalRevenueRecord{}
	if err := DB.Select("id", "version", "status").First(current, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrExternalRevenueNotFound
		}
		return err
	}
	if current.Status == ExternalRevenueStatusVoided {
		return ErrExternalRevenueVoided
	}
	return ErrExternalRevenueVersionConflict
}

func GetExternalRevenueSummary(startTime int64, endTime int64, timezoneOffset int) (*ExternalRevenueSummary, error) {
	var records []*ExternalRevenueRecord
	err := DB.Where("status = ? AND occurred_at >= ? AND occurred_at < ?", ExternalRevenueStatusActive, startTime, endTime).
		Order("occurred_at asc, id asc").Find(&records).Error
	if err != nil {
		return nil, err
	}

	totalMap := map[string]*ExternalRevenueTotal{}
	sourceMap := map[string]*ExternalRevenueSourceTotal{}
	timelineMap := map[string]*ExternalRevenueTimelinePoint{}
	location := time.FixedZone("admin", timezoneOffset*60)
	for _, record := range records {
		totalKey := record.Currency
		sourceKey := strings.Join([]string{record.Source, record.SourceLabel, record.Currency}, "\x00")
		date := time.Unix(record.OccurredAt, 0).In(location).Format("2006-01-02")
		timelineKey := strings.Join([]string{date, record.Source, record.SourceLabel, record.Currency}, "\x00")

		if totalMap[totalKey] == nil {
			totalMap[totalKey] = &ExternalRevenueTotal{Currency: record.Currency}
		}
		if sourceMap[sourceKey] == nil {
			sourceMap[sourceKey] = &ExternalRevenueSourceTotal{Source: record.Source, SourceLabel: record.SourceLabel, Currency: record.Currency}
		}
		if timelineMap[timelineKey] == nil {
			timelineMap[timelineKey] = &ExternalRevenueTimelinePoint{Date: date, Source: record.Source, SourceLabel: record.SourceLabel, Currency: record.Currency}
		}
		if totalMap[totalKey].AmountMinor > math.MaxInt64-record.AmountMinor ||
			sourceMap[sourceKey].AmountMinor > math.MaxInt64-record.AmountMinor ||
			timelineMap[timelineKey].AmountMinor > math.MaxInt64-record.AmountMinor {
			return nil, errors.New("external revenue summary exceeds supported range")
		}
		totalMap[totalKey].AmountMinor += record.AmountMinor
		totalMap[totalKey].Count++
		sourceMap[sourceKey].AmountMinor += record.AmountMinor
		sourceMap[sourceKey].Count++
		timelineMap[timelineKey].AmountMinor += record.AmountMinor
		timelineMap[timelineKey].Count++
	}

	summary := &ExternalRevenueSummary{
		Totals:    make([]ExternalRevenueTotal, 0, len(totalMap)),
		BySource:  make([]ExternalRevenueSourceTotal, 0, len(sourceMap)),
		Timeline:  make([]ExternalRevenueTimelinePoint, 0, len(timelineMap)),
		StartTime: startTime,
		EndTime:   endTime,
		Timezone:  timezoneOffset,
	}
	for _, total := range totalMap {
		summary.Totals = append(summary.Totals, *total)
	}
	for _, total := range sourceMap {
		summary.BySource = append(summary.BySource, *total)
	}
	for _, point := range timelineMap {
		summary.Timeline = append(summary.Timeline, *point)
	}
	sort.Slice(summary.Totals, func(i, j int) bool { return summary.Totals[i].Currency < summary.Totals[j].Currency })
	sort.Slice(summary.BySource, func(i, j int) bool {
		if summary.BySource[i].Currency == summary.BySource[j].Currency {
			return summary.BySource[i].Source < summary.BySource[j].Source
		}
		return summary.BySource[i].Currency < summary.BySource[j].Currency
	})
	sort.Slice(summary.Timeline, func(i, j int) bool {
		if summary.Timeline[i].Date == summary.Timeline[j].Date {
			return summary.Timeline[i].Source < summary.Timeline[j].Source
		}
		return summary.Timeline[i].Date < summary.Timeline[j].Date
	})
	return summary, nil
}
