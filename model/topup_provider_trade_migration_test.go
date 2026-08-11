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

type legacyTopUpProviderTrade struct {
	ID              int    `gorm:"column:id;primaryKey"`
	TradeNo         string `gorm:"column:trade_no"`
	PaymentProvider string `gorm:"column:payment_provider"`
	ProviderTradeNo string `gorm:"column:provider_trade_no"`
	Status          string `gorm:"column:status"`
}

func (legacyTopUpProviderTrade) TableName() string { return "top_ups" }

func useTopUpMigrationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	previousDB := DB
	db, err := gorm.Open(
		sqlite.Open("file:"+strings.ReplaceAll(t.Name(), "/", "_")+"?mode=memory&cache=shared"),
		&gorm.Config{},
	)
	require.NoError(t, err)
	DB = db
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() {
		DB = previousDB
		_ = sqlDB.Close()
	})
	require.NoError(t, db.AutoMigrate(&legacyTopUpProviderTrade{}))
	return db
}

func TestPrepareTopUpProviderTradeUniqueIndexNormalizesLegacyEmptyValues(t *testing.T) {
	db := useTopUpMigrationTestDB(t)
	require.NoError(t, db.Create(&[]legacyTopUpProviderTrade{
		{TradeNo: "legacy-empty-1", PaymentProvider: PaymentProviderEpay},
		{TradeNo: "legacy-empty-2", PaymentProvider: PaymentProviderEpay},
	}).Error)

	require.NoError(t, prepareTopUpProviderTradeUniqueIndex())
	require.NoError(t, db.AutoMigrate(&TopUp{}))
	var nullCount int64
	require.NoError(t, db.Model(&TopUp{}).Where("provider_trade_no IS NULL").Count(&nullCount).Error)
	assert.Equal(t, int64(2), nullCount)
	assert.True(t, db.Migrator().HasIndex(&TopUp{}, "idx_top_ups_provider_trade"))

	providerTradeNo := "provider-unique"
	require.NoError(t, db.Create(&TopUp{TradeNo: "new-1", PaymentProvider: PaymentProviderEpay, ProviderTradeNo: &providerTradeNo}).Error)
	err := db.Create(&TopUp{TradeNo: "new-2", PaymentProvider: PaymentProviderEpay, ProviderTradeNo: &providerTradeNo}).Error
	assert.Error(t, err)
}

func TestPrepareTopUpProviderTradeUniqueIndexRejectsLegacyDuplicates(t *testing.T) {
	db := useTopUpMigrationTestDB(t)
	require.NoError(t, db.Create(&[]legacyTopUpProviderTrade{
		{TradeNo: "legacy-duplicate-1", PaymentProvider: PaymentProviderEpay, ProviderTradeNo: "provider-duplicate"},
		{TradeNo: "legacy-duplicate-2", PaymentProvider: PaymentProviderEpay, ProviderTradeNo: "provider-duplicate"},
	}).Error)

	err := prepareTopUpProviderTradeUniqueIndex()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires manual audit")
}

func TestLegacyPendingEpayPreflightRunsBeforeSchemaMutation(t *testing.T) {
	db := useTopUpMigrationTestDB(t)
	require.NoError(t, db.Create(&legacyTopUpProviderTrade{
		TradeNo: "legacy-pending", PaymentProvider: PaymentProviderEpay,
		Status: common.TopUpStatusPending,
	}).Error)

	err := ensureNoLegacyPendingEpayOrders()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be drained before upgrade")
	assert.False(t, db.Migrator().HasColumn(&TopUp{}, "expected_amount_minor"))
}
