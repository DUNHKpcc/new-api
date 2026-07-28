package model

import (
	"os"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestPccAgentUserManagementConfiguredDatabases(t *testing.T) {
	tests := []struct {
		name         string
		env          string
		databaseType common.DatabaseType
		dialector    func(string) gorm.Dialector
	}{
		{
			name:         "mysql",
			env:          "TEST_MYSQL_DSN",
			databaseType: common.DatabaseTypeMySQL,
			dialector:    func(dsn string) gorm.Dialector { return mysql.Open(dsn) },
		},
		{
			name:         "postgres",
			env:          "TEST_POSTGRES_DSN",
			databaseType: common.DatabaseTypePostgreSQL,
			dialector: func(dsn string) gorm.Dialector {
				return postgres.New(postgres.Config{DSN: dsn, PreferSimpleProtocol: true})
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dsn := strings.TrimSpace(os.Getenv(test.env))
			if dsn == "" {
				t.Skip(test.env + " is not configured")
			}
			db, err := gorm.Open(test.dialector(dsn), &gorm.Config{})
			require.NoError(t, err)
			sqlDB, err := db.DB()
			require.NoError(t, err)
			t.Cleanup(func() { _ = sqlDB.Close() })
			require.NoError(t, db.AutoMigrate(
				&User{},
				&DesktopGrant{},
				&ExternalIdentityClaim{},
				&SubscriptionPlan{},
				&UserSubscription{},
			))

			previousDB := DB
			previousMainType := common.MainDatabaseType()
			DB = db
			common.SetMainDatabaseType(test.databaseType)
			t.Cleanup(func() {
				DB = previousDB
				common.SetMainDatabaseType(previousMainType)
			})

			suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:10]
			user := User{
				Username:    "pcc-mgmt-" + suffix,
				Password:    "unused",
				DisplayName: "PccAgent management cross-db",
				Status:      common.UserStatusEnabled,
				Role:        common.RoleCommonUser,
				Group:       "default",
				AffCode:     "pcc-" + suffix,
			}
			require.NoError(t, db.Create(&user).Error)
			plan := SubscriptionPlan{
				Title:              "PccAgent management " + suffix,
				DurationUnit:       SubscriptionDurationMonth,
				DurationValue:      1,
				Enabled:            true,
				MaxPurchasePerUser: 1,
				TotalAmount:        5_000,
			}
			require.NoError(t, db.Create(&plan).Error)
			activeSlot := 1
			require.NoError(t, db.Create(&DesktopGrant{
				PublicId:     uuid.NewString(),
				UserId:       user.Id,
				ClientId:     "pcc-agent-desktop",
				DeviceIdHash: common.GenerateHMAC("pcc-management-" + suffix),
				DeviceName:   "Cross DB device",
				Scopes:       "relay account.read usage.read",
				Status:       DesktopGrantStatusActive,
				ActiveSlot:   &activeSlot,
				ExpiredTime:  common.GetTimestamp() + 3600,
				CreatedTime:  1_700_000_000,
			}).Error)
			require.NoError(t, db.Create(&UserSubscription{
				UserId:      user.Id,
				PlanId:      plan.Id,
				AmountTotal: 5_000,
				AmountUsed:  1_000,
				StartTime:   1_700_000_000,
				EndTime:     1_800_000_000,
				Status:      "active",
				Source:      UserSubscriptionSourcePccAgentGift,
			}).Error)
			require.NoError(t, ClaimExternalIdentityWithTx(
				db,
				ExternalIdentityProviderWeChatUnionID,
				"cross-db-union-"+suffix,
				user.Id,
			))
			t.Cleanup(func() {
				_ = db.Where("user_id = ?", user.Id).Delete(&UserSubscription{}).Error
				_ = db.Where("user_id = ?", user.Id).Delete(&DesktopGrant{}).Error
				_ = db.Where("user_id = ?", user.Id).Delete(&ExternalIdentityClaim{}).Error
				_ = db.Delete(&plan).Error
				_ = db.Unscoped().Delete(&user).Error
			})

			users, total, err := SearchUsers(
				user.Username,
				"",
				nil,
				nil,
				0,
				20,
				true,
				NewUserSortOptions("id", "asc"),
			)
			require.NoError(t, err)
			assert.Equal(t, int64(1), total)
			require.Len(t, users, 1)
			require.NotNil(t, users[0].PccAgentSummary)
			assert.EqualValues(t, 1, users[0].PccAgentSummary.ActiveDeviceCount)
			assert.True(t, users[0].PccAgentSummary.WeChatVerified)
			require.NotNil(t, users[0].PccAgentSummary.Gift)
			assert.EqualValues(t, 4_000, users[0].PccAgentSummary.Gift.AmountRemaining)
		})
	}
}
