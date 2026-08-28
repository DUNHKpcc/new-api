package model

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupRevenueCostTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&RevenueCostRecord{}))

	previousDB := DB
	DB = db
	t.Cleanup(func() {
		DB = previousDB
		sqlDB, sqlErr := db.DB()
		if sqlErr == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func revenueCostFixture(month, category, currency string, amount, createdAt int64) *RevenueCostRecord {
	return &RevenueCostRecord{
		Month:       month,
		Category:    category,
		Description: "fixture",
		AmountMinor: amount,
		Currency:    currency,
		Version:     1,
		CreatedBy:   1,
		UpdatedBy:   1,
		CreateTime:  createdAt,
		UpdateTime:  createdAt,
	}
}

func TestRevenueCostListFiltersAndSumsPersistedRecords(t *testing.T) {
	setupRevenueCostTestDB(t)
	server := revenueCostFixture("2026-08", RevenueCostCategoryServer, "CNY", 1200, 10)
	upstream := revenueCostFixture("2026-08", RevenueCostCategoryUpstream, "CNY", 3450, 11)
	otherMonth := revenueCostFixture("2026-07", RevenueCostCategoryOther, "CNY", 9999, 12)
	otherCurrency := revenueCostFixture("2026-08", RevenueCostCategoryOther, "USD", 500, 13)
	for _, record := range []*RevenueCostRecord{server, upstream, otherMonth, otherCurrency} {
		require.NoError(t, CreateRevenueCost(record))
	}

	page := &common.PageInfo{Page: 1, PageSize: 20}
	records, total, amount, err := ListRevenueCosts(page, RevenueCostListFilter{Month: "2026-08", Currency: "CNY"})
	require.NoError(t, err)
	assert.Len(t, records, 2)
	assert.Equal(t, int64(2), total)
	assert.Equal(t, int64(4650), amount)

	records, total, amount, err = ListRevenueCosts(page, RevenueCostListFilter{Month: "2026-08", Category: RevenueCostCategoryUpstream, Currency: "CNY"})
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, upstream.Id, records[0].Id)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, int64(3450), amount)
}

func TestRevenueCostOptimisticUpdateRejectsStaleVersion(t *testing.T) {
	setupRevenueCostTestDB(t)
	record := revenueCostFixture("2026-08", RevenueCostCategoryServer, "CNY", 1200, 10)
	require.NoError(t, CreateRevenueCost(record))

	record.AmountMinor = 1800
	record.Category = RevenueCostCategoryAccount
	record.UpdatedBy = 2
	record.UpdateTime = 20
	require.NoError(t, UpdateRevenueCost(record, 1))
	assert.Equal(t, int64(2), record.Version)
	assert.Equal(t, 1, record.CreatedBy)

	record.AmountMinor = 2400
	err := UpdateRevenueCost(record, 1)
	assert.ErrorIs(t, err, ErrRevenueCostVersionConflict)
}

func TestRevenueCostRejectsNegativeAmounts(t *testing.T) {
	setupRevenueCostTestDB(t)
	err := CreateRevenueCost(revenueCostFixture("2026-08", RevenueCostCategoryOther, "CNY", -1, 10))
	assert.ErrorIs(t, err, ErrRevenueCostAmountInvalid)
}
