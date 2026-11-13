package model

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"testing"

	"github.com/QuantumNous/new-api/common"
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

	require.NoError(t, db.AutoMigrate(&Option{}, &TopUp{}, &GuardianServiceOrder{}))
	platformSummary, err := GetPlatformRevenueSummary(1_725_120_000, 1_725_206_400, 480)
	require.NoError(t, err)
	assert.NotNil(t, platformSummary.Totals)
	assert.NotNil(t, platformSummary.ByPaymentMethod)
	assert.NotNil(t, platformSummary.Timeline)
	assert.Empty(t, platformSummary.Totals)
}

func TestPlatformRevenueSummaryIncludesOnlySuccessfulCompletedOrders(t *testing.T) {
	db := setupExternalRevenueTestDB(t)
	require.NoError(t, db.AutoMigrate(&Option{}, &TopUp{}, &GuardianServiceOrder{}))
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

func TestGuardianPlatformRevenue(t *testing.T) {
	for _, dialect := range []string{"sqlite", "mysql", "postgres"} {
		t.Run(dialect, func(t *testing.T) {
			var driver gorm.Dialector
			switch dialect {
			case "sqlite":
				driver = sqlite.Open(filepath.Join(t.TempDir(), "revenue.db"))
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
				driver = postgres.Open(dsn)
			}
			db, err := gorm.Open(driver, &gorm.Config{})
			require.NoError(t, err)
			oldType := common.MainDatabaseType()
			switch dialect {
			case "sqlite":
				common.SetMainDatabaseType(common.DatabaseTypeSQLite)
			case "mysql":
				common.SetMainDatabaseType(common.DatabaseTypeMySQL)
			case "postgres":
				common.SetMainDatabaseType(common.DatabaseTypePostgreSQL)
			}
			initCol()
			old := DB
			DB = db
			t.Cleanup(func() {
				DB = old
				common.SetMainDatabaseType(oldType)
				initCol()
				sqlDB, err := db.DB()
				if err == nil {
					_ = sqlDB.Close()
				}
			})
			require.NoError(t, db.AutoMigrate(&Option{}, &TopUp{}, &GuardianServiceOrder{}))
			db = db.Begin()
			require.NoError(t, db.Error)
			DB = db
			t.Cleanup(func() { _ = db.Rollback().Error })
			var version string
			query := "SELECT version()"
			if dialect == "sqlite" {
				query = "SELECT sqlite_version()"
			}
			require.NoError(t, db.Raw(query).Scan(&version).Error)
			t.Log(version)
			const start int64 = 1725148800
			const end int64 = start + 86400
			orders := []GuardianServiceOrder{}
			for i, status := range []string{"paid", "scheduled", "in_progress", "delivered", "refunded", "pending_payment", "cancelled", "expired"} {
				orders = append(orders, GuardianServiceOrder{TradeNo: fmt.Sprintf("revenue-%s", status), Status: status, AmountCents: int64(i+1) * 100, PaidAt: start, CreatedAt: start - 86400, RefundedAt: end + 86400, PaymentMethod: "alipay"})
			}
			orders = append(orders,
				GuardianServiceOrder{TradeNo: "unpaid-pending", Status: "pending_payment", AmountCents: 99999, PaidAt: 0},
				GuardianServiceOrder{TradeNo: "unpaid-cancelled", Status: "cancelled", AmountCents: 99999, PaidAt: 0},
				GuardianServiceOrder{TradeNo: "unpaid-refund", Status: "refunded", AmountCents: 99999, PaidAt: 0},
				GuardianServiceOrder{TradeNo: "before-range", Status: "paid", AmountCents: 99999, PaidAt: start - 1},
				GuardianServiceOrder{TradeNo: "end-exclusive", Status: "paid", AmountCents: 99999, PaidAt: end},
			)
			require.NoError(t, db.Create(&orders).Error)
			topups := []TopUp{
				{TradeNo: "revenue-cny", Status: "success", PaidCurrency: "CNY", PaidAmountMinor: 250, CompleteTime: start, PaymentMethod: "alipay"},
				{TradeNo: "revenue-usd", Status: "success", PaidCurrency: "USD", PaidAmountMinor: 900, CompleteTime: start, PaymentMethod: "stripe"},
			}
			require.NoError(t, db.Create(&topups).Error)
			summary, err := GetPlatformRevenueSummary(start, end, 480)
			require.NoError(t, err)
			assert.Equal(t, []PlatformRevenueTotal{{Currency: "CNY", AmountMinor: 3850, Count: 9, VerifiedCount: 9}, {Currency: "USD", AmountMinor: 900, Count: 1, VerifiedCount: 1}}, summary.Totals)
			assert.Contains(t, summary.ByPaymentMethod, PlatformRevenueMethodTotal{PaymentMethod: "alipay", Currency: "CNY", AmountMinor: 3850, Count: 9})
			assert.Contains(t, summary.Timeline, PlatformRevenueTimelinePoint{Date: "2024-09-01", PaymentMethod: "alipay", Currency: "CNY", AmountMinor: 3850, Count: 9})
			next, err := GetPlatformRevenueSummary(end+86400, end+172800, 480)
			require.NoError(t, err)
			assert.Empty(t, next.Totals, "refund date must not count the original payment twice")
			previousDay, err := GetPlatformRevenueSummary(start, end, -480)
			require.NoError(t, err)
			assert.Equal(t, summary.Totals, previousDay.Totals, "timezone changes date labels, not selected receipts")
			assert.Contains(t, previousDay.Timeline, PlatformRevenueTimelinePoint{Date: "2024-08-31", PaymentMethod: "alipay", Currency: "CNY", AmountMinor: 3850, Count: 9})
			adjacent, err := GetPlatformRevenueSummary(end, end+86400, 480)
			require.NoError(t, err)
			assert.Equal(t, []PlatformRevenueTotal{{Currency: "CNY", AmountMinor: 99999, Count: 1, VerifiedCount: 1}}, adjacent.Totals, "receipt at the end boundary belongs only to the next period")
			repeated, err := GetPlatformRevenueSummary(start, end, 480)
			require.NoError(t, err)
			assert.Equal(t, summary, repeated, "reloading historical revenue must preserve the original payment period")
			var after []GuardianServiceOrder
			require.NoError(t, db.Order("id").Find(&after).Error)
			assert.Equal(t, orders, after, "reporting must not modify order state or payment amounts")
		})
	}
}
