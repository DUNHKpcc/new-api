package model

import (
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupExternalRevenueTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&ExternalRevenueRecord{}))

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

func externalRevenueFixture(orderNo *string, amount int64, occurredAt int64) *ExternalRevenueRecord {
	return &ExternalRevenueRecord{
		Source:          "xianyu",
		ExternalOrderNo: orderNo,
		AmountMinor:     amount,
		Currency:        "CNY",
		OccurredAt:      occurredAt,
		Status:          ExternalRevenueStatusActive,
		Version:         1,
		CreatedBy:       1,
		UpdatedBy:       1,
		CreateTime:      occurredAt,
		UpdateTime:      occurredAt,
	}
}

func TestExternalRevenueOrderIdentityIsScopedBySource(t *testing.T) {
	setupExternalRevenueTestDB(t)
	orderNo := "third-party-1001"
	require.NoError(t, CreateExternalRevenue(externalRevenueFixture(&orderNo, 1200, 1_725_148_800)))

	err := CreateExternalRevenue(externalRevenueFixture(&orderNo, 1800, 1_725_148_801))
	assert.ErrorIs(t, err, ErrExternalRevenueDuplicate)

	otherSource := externalRevenueFixture(&orderNo, 1800, 1_725_148_801)
	otherSource.Source = "wechat"
	assert.NoError(t, CreateExternalRevenue(otherSource))
	assert.NoError(t, CreateExternalRevenue(externalRevenueFixture(nil, 500, 1_725_148_802)))
	assert.NoError(t, CreateExternalRevenue(externalRevenueFixture(nil, 600, 1_725_148_803)))
}

func TestExternalRevenueOptimisticUpdateRejectsStaleVersion(t *testing.T) {
	setupExternalRevenueTestDB(t)
	record := externalRevenueFixture(nil, 1200, 1_725_148_800)
	require.NoError(t, CreateExternalRevenue(record))

	record.AmountMinor = 1500
	record.CreatedBy = 2
	record.UpdatedBy = 2
	record.UpdateTime++
	require.NoError(t, UpdateExternalRevenue(record, 1))
	assert.Equal(t, int64(2), record.Version)
	assert.Equal(t, 1, record.CreatedBy)

	record.AmountMinor = 2000
	err := UpdateExternalRevenue(record, 1)
	assert.ErrorIs(t, err, ErrExternalRevenueVersionConflict)
}

func TestExternalRevenueSummaryExcludesVoidedRecords(t *testing.T) {
	setupExternalRevenueTestDB(t)
	active := externalRevenueFixture(nil, 1200, 1_725_148_800)
	voided := externalRevenueFixture(nil, 800, 1_725_152_400)
	require.NoError(t, CreateExternalRevenue(active))
	require.NoError(t, CreateExternalRevenue(voided))
	require.NoError(t, VoidExternalRevenue(voided.Id, voided.Version, 7, 1_725_152_500))

	summary, err := GetExternalRevenueSummary(1_725_120_000, 1_725_206_400, 480)
	require.NoError(t, err)
	require.Len(t, summary.Totals, 1)
	assert.Equal(t, int64(1200), summary.Totals[0].AmountMinor)
	assert.Equal(t, int64(1), summary.Totals[0].Count)
	require.Len(t, summary.Timeline, 1)
	assert.Equal(t, "2024-09-01", summary.Timeline[0].Date)
}

func TestRevenueEmptySummariesUseEmptyArrays(t *testing.T) {
	db := setupExternalRevenueTestDB(t)

	externalSummary, err := GetExternalRevenueSummary(1_725_120_000, 1_725_206_400, 480)
	require.NoError(t, err)
	assert.NotNil(t, externalSummary.Totals)
	assert.NotNil(t, externalSummary.BySource)
	assert.NotNil(t, externalSummary.Timeline)
	assert.Empty(t, externalSummary.Totals)

	require.NoError(t, db.AutoMigrate(&Option{}, &TopUp{}))
	platformSummary, err := GetPlatformRevenueSummary(1_725_120_000, 1_725_206_400, 480)
	require.NoError(t, err)
	assert.NotNil(t, platformSummary.Totals)
	assert.NotNil(t, platformSummary.ByPaymentMethod)
	assert.NotNil(t, platformSummary.Timeline)
	assert.Empty(t, platformSummary.Totals)
}

func TestPlatformRevenueSummaryIncludesOnlySuccessfulCompletedOrders(t *testing.T) {
	db := setupExternalRevenueTestDB(t)
	require.NoError(t, db.AutoMigrate(&Option{}, &TopUp{}))
	require.NoError(t, db.Create(&TopUp{
		Money:           99,
		PaidAmountMinor: 1200,
		PaidCurrency:    "CNY",
		PaymentMethod:   "wxpay",
		TradeNo:         "platform-verified",
		CompleteTime:    1_725_148_800,
		Status:          "success",
	}).Error)
	require.NoError(t, db.Create(&TopUp{
		Money:         2.5,
		PaymentMethod: "alipay",
		TradeNo:       "platform-recorded",
		CompleteTime:  1_725_152_400,
		Status:        "success",
	}).Error)
	require.NoError(t, db.Create(&TopUp{
		Money:         500,
		PaymentMethod: "alipay",
		TradeNo:       "platform-pending",
		CompleteTime:  1_725_152_400,
		Status:        "pending",
	}).Error)

	summary, err := GetPlatformRevenueSummary(1_725_120_000, 1_725_206_400, 480)
	require.NoError(t, err)
	require.Len(t, summary.Totals, 1)
	assert.Equal(t, "CNY", summary.Totals[0].Currency)
	assert.Equal(t, int64(1450), summary.Totals[0].AmountMinor)
	assert.Equal(t, int64(2), summary.Totals[0].Count)
	assert.Equal(t, int64(1), summary.Totals[0].VerifiedCount)
	assert.Len(t, summary.ByPaymentMethod, 2)
}
