/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
package model

import (
	"errors"
	"math"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	RevenueCostCategoryServer         = "server"
	RevenueCostCategoryUpstream       = "upstream"
	RevenueCostCategoryAccount        = "account"
	RevenueCostCategoryOther          = "other"
	maxRevenueCostAmountMinor   int64 = 1_000_000_000_000_000
)

var (
	ErrRevenueCostNotFound        = errors.New("revenue cost record not found")
	ErrRevenueCostVersionConflict = errors.New("revenue cost record was changed")
	ErrRevenueCostAmountInvalid   = errors.New("revenue cost amount is outside the supported range")
)

// RevenueCostRecord is an administrator-maintained reporting cost. It is
// intentionally isolated from platform billing, balances, quotas, and
// settlement paths.
type RevenueCostRecord struct {
	Id          int64  `json:"id"`
	Month       string `json:"month" gorm:"type:varchar(7);not null;index:idx_revenue_cost_month_currency,priority:1"`
	Category    string `json:"category" gorm:"type:varchar(32);not null;index"`
	Description string `json:"description" gorm:"type:varchar(500);not null"`
	AmountMinor int64  `json:"amount_minor,string" gorm:"type:bigint;not null"`
	Currency    string `json:"currency" gorm:"type:varchar(3);not null;index:idx_revenue_cost_month_currency,priority:2"`
	Version     int64  `json:"version" gorm:"type:bigint;not null"`
	CreatedBy   int    `json:"created_by" gorm:"not null"`
	UpdatedBy   int    `json:"updated_by" gorm:"not null"`
	CreateTime  int64  `json:"create_time" gorm:"type:bigint;not null"`
	UpdateTime  int64  `json:"update_time" gorm:"type:bigint;not null"`
}

func (RevenueCostRecord) TableName() string {
	return "revenue_cost_records"
}

type RevenueCostListFilter struct {
	Month    string
	Category string
	Currency string
}

func IsRevenueCostCategory(category string) bool {
	switch category {
	case RevenueCostCategoryServer, RevenueCostCategoryUpstream, RevenueCostCategoryAccount, RevenueCostCategoryOther:
		return true
	default:
		return false
	}
}

func validateRevenueCostAmount(amountMinor int64) error {
	if amountMinor < 0 || amountMinor > maxRevenueCostAmountMinor {
		return ErrRevenueCostAmountInvalid
	}
	return nil
}

func revenueCostQuery(tx *gorm.DB, filter RevenueCostListFilter) *gorm.DB {
	query := tx.Model(&RevenueCostRecord{})
	if filter.Month != "" {
		query = query.Where("month = ?", filter.Month)
	}
	if filter.Category != "" {
		query = query.Where("category = ?", filter.Category)
	}
	if filter.Currency != "" {
		query = query.Where("currency = ?", filter.Currency)
	}
	return query
}

func CreateRevenueCost(record *RevenueCostRecord) error {
	if record == nil {
		return ErrRevenueCostAmountInvalid
	}
	if err := validateRevenueCostAmount(record.AmountMinor); err != nil {
		return err
	}
	return DB.Create(record).Error
}

// ListRevenueCosts returns the requested page and the total amount for the
// same filter. Amounts are summed in Go so overflow is detected consistently
// across SQLite, MySQL, and PostgreSQL.
func ListRevenueCosts(pageInfo *common.PageInfo, filter RevenueCostListFilter) ([]*RevenueCostRecord, int64, int64, error) {
	query := revenueCostQuery(DB, filter)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, 0, err
	}

	var amounts []int64
	if err := revenueCostQuery(DB, filter).Select("amount_minor").Find(&amounts).Error; err != nil {
		return nil, 0, 0, err
	}
	var totalAmountMinor int64
	for _, amount := range amounts {
		if err := validateRevenueCostAmount(amount); err != nil {
			return nil, 0, 0, err
		}
		if totalAmountMinor > math.MaxInt64-amount {
			return nil, 0, 0, errors.New("revenue cost total exceeds supported range")
		}
		totalAmountMinor += amount
	}

	pageQuery := revenueCostQuery(DB, filter).Order("month desc, id desc")
	if pageInfo != nil {
		pageQuery = pageQuery.Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx())
	}
	var records []*RevenueCostRecord
	if err := pageQuery.Find(&records).Error; err != nil {
		return nil, 0, 0, err
	}
	if records == nil {
		records = make([]*RevenueCostRecord, 0)
	}
	return records, total, totalAmountMinor, nil
}

func UpdateRevenueCost(record *RevenueCostRecord, expectedVersion int64) error {
	if record == nil {
		return ErrRevenueCostNotFound
	}
	if err := validateRevenueCostAmount(record.AmountMinor); err != nil {
		return err
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		current := &RevenueCostRecord{}
		if err := lockForUpdate(tx).First(current, record.Id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrRevenueCostNotFound
			}
			return err
		}
		if current.Version != expectedVersion {
			return ErrRevenueCostVersionConflict
		}
		updates := map[string]interface{}{
			"month":        record.Month,
			"category":     record.Category,
			"description":  record.Description,
			"amount_minor": record.AmountMinor,
			"currency":     record.Currency,
			"updated_by":   record.UpdatedBy,
			"update_time":  record.UpdateTime,
			"version":      expectedVersion + 1,
		}
		result := tx.Model(current).Where("version = ?", expectedVersion).Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrRevenueCostVersionConflict
		}
		record.CreatedBy = current.CreatedBy
		record.CreateTime = current.CreateTime
		record.Version = expectedVersion + 1
		return nil
	})
}
