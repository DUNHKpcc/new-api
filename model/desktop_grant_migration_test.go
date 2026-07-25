package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type desktopGrantBeforeCodexToken struct {
	Id           int64  `gorm:"primaryKey"`
	PublicId     string `gorm:"type:varchar(64);not null;uniqueIndex"`
	UserId       int    `gorm:"not null;index;uniqueIndex:idx_desktop_grants_active_device,priority:1"`
	ClientId     string `gorm:"type:varchar(64);not null;uniqueIndex:idx_desktop_grants_active_device,priority:2"`
	DeviceIdHash string `gorm:"type:char(64);not null;index;uniqueIndex:idx_desktop_grants_active_device,priority:3"`
	DeviceName   string `gorm:"type:varchar(128);not null"`
	Platform     string `gorm:"type:varchar(64)"`
	AppVersion   string `gorm:"type:varchar(32)"`
	TokenId      *int   `gorm:"uniqueIndex"`
	Scopes       string `gorm:"type:varchar(255);not null"`
	Status       string `gorm:"type:varchar(16);not null;index"`
	ActiveSlot   *int   `gorm:"uniqueIndex:idx_desktop_grants_active_device,priority:4"`
	CreatedTime  int64  `gorm:"type:bigint;not null;index"`
	LastUsedTime int64  `gorm:"type:bigint;not null;default:0"`
	ExpiredTime  int64  `gorm:"type:bigint;not null;default:0;index"`
	RevokedTime  int64  `gorm:"type:bigint;not null;default:0"`
	RevokeReason string `gorm:"type:varchar(64)"`
}

func (desktopGrantBeforeCodexToken) TableName() string {
	return "desktop_grants"
}

func TestDesktopGrantCodexTokenMigrationSQLite(t *testing.T) {
	previousDB := DB
	previousType := common.MainDatabaseType()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	t.Cleanup(func() {
		DB = previousDB
		common.SetMainDatabaseType(previousType)
	})

	require.NoError(t, db.AutoMigrate(&desktopGrantBeforeCodexToken{}))
	require.NoError(t, db.Create(&desktopGrantBeforeCodexToken{
		PublicId:     "legacy-grant",
		UserId:       1,
		ClientId:     "pcc-agent-desktop",
		DeviceIdHash: "legacy-device",
		DeviceName:   "Legacy device",
		Scopes:       "relay",
		Status:       DesktopGrantStatusRevoked,
		CreatedTime:  1,
	}).Error)

	require.NoError(t, ensureDesktopGrantCodexTokenColumnSQLite())
	require.NoError(t, ensureDesktopGrantCodexTokenColumnSQLite())
	require.NoError(t, db.AutoMigrate(&DesktopGrant{}))
	assert.True(t, db.Migrator().HasColumn(&DesktopGrant{}, "codex_token_id"))

	var legacy DesktopGrant
	require.NoError(t, db.Where("public_id = ?", "legacy-grant").First(&legacy).Error)
	assert.Nil(t, legacy.CodexTokenId)

	tokenID := 22
	require.NoError(t, db.Create(&DesktopGrant{
		PublicId:     "new-grant",
		UserId:       2,
		ClientId:     "pcc-agent-desktop",
		DeviceIdHash: "new-device",
		DeviceName:   "New device",
		CodexTokenId: &tokenID,
		Scopes:       "relay",
		Status:       DesktopGrantStatusRevoked,
		CreatedTime:  2,
	}).Error)
	err = db.Create(&DesktopGrant{
		PublicId:     "duplicate-codex-token",
		UserId:       3,
		ClientId:     "pcc-agent-desktop",
		DeviceIdHash: "another-device",
		DeviceName:   "Another device",
		CodexTokenId: &tokenID,
		Scopes:       "relay",
		Status:       DesktopGrantStatusRevoked,
		CreatedTime:  3,
	}).Error
	assert.Error(t, err)
}
