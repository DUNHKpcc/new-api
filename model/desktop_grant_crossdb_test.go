package model

import (
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestDesktopGrantLifecycleConfiguredDatabases(t *testing.T) {
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
			require.NoError(t, db.AutoMigrate(&User{}, &Token{}, &DesktopGrant{}))
			require.NoError(t, db.AutoMigrate(&User{}, &Token{}, &DesktopGrant{}),
				"desktop authorization migrations must be repeatable")

			previousDB := DB
			previousMainType := common.MainDatabaseType()
			previousRedis := common.RedisEnabled
			DB = db
			common.SetMainDatabaseType(test.databaseType)
			common.RedisEnabled = false
			t.Cleanup(func() {
				DB = previousDB
				common.SetMainDatabaseType(previousMainType)
				common.RedisEnabled = previousRedis
			})

			suffix := uuid.NewString()
			user := &User{
				Username:    "desktop-crossdb-" + suffix,
				Password:    "unused",
				Status:      common.UserStatusEnabled,
				Group:       "default",
				AuthVersion: 1,
			}
			require.NoError(t, db.Create(user).Error)
			t.Cleanup(func() {
				var tokenIDs []int
				_ = db.Model(&Token{}).Where("user_id = ?", user.Id).Pluck("id", &tokenIDs).Error
				_ = db.Unscoped().Where("user_id = ?", user.Id).Delete(&DesktopGrant{}).Error
				if len(tokenIDs) > 0 {
					_ = db.Unscoped().Where("id IN ?", tokenIDs).Delete(&Token{}).Error
				}
				_ = db.Unscoped().Where("id = ?", user.Id).Delete(&User{}).Error
			})

			expiresAt := time.Now().Add(time.Hour).Unix()
			policy := DesktopGrantTokenPolicy{
				Name:           "PCC Agent cross-db",
				Group:          "default",
				ModelLimits:    "gpt-contract",
				Scopes:         "relay account.read usage.read",
				ExpiredTime:    expiresAt,
				UnlimitedQuota: true,
			}
			pending := &DesktopGrant{
				PublicId:     uuid.NewString(),
				UserId:       user.Id,
				ClientId:     "pcc-agent-desktop",
				DeviceIdHash: common.GenerateHMAC("cross-db-device-" + suffix),
				DeviceName:   "Cross DB device",
				Scopes:       policy.Scopes,
			}
			require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
				return CreatePendingDesktopGrantWithTx(tx, pending)
			}))
			legacyClaude := policy
			legacyClaude.Key = "legacy-claude-" + suffix
			legacyCodex := policy
			legacyCodex.Key = "legacy-codex-" + suffix
			legacy, err := func() (*DesktopGrantActivationResult, error) {
				var activation *DesktopGrantActivationResult
				err := db.Transaction(func(tx *gorm.DB) error {
					var activateErr error
					activation, activateErr = ActivateDesktopGrantWithTokensTx(
						tx, pending.PublicId, user.Id, legacyClaude, legacyCodex,
					)
					return activateErr
				})
				return activation, err
			}()
			require.NoError(t, err)
			require.NotNil(t, legacy)
			_, err = GetActiveDesktopGrantByTokenId(legacy.ClaudeToken.Id)
			require.NoError(t, err)

			stagedGrant := &DesktopGrant{
				PublicId:     uuid.NewString(),
				UserId:       user.Id,
				ClientId:     "pcc-agent-desktop",
				DeviceIdHash: pending.DeviceIdHash,
				DeviceName:   pending.DeviceName,
				Scopes:       policy.Scopes,
			}
			require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
				return CreatePendingDesktopGrantWithTx(tx, stagedGrant)
			}))
			confirmationToken := "confirmation-" + suffix
			confirmationHash := common.GenerateHMAC(confirmationToken)
			stagedClaude := policy
			stagedClaude.Key = "staged-claude-" + suffix
			stagedCodex := policy
			stagedCodex.Key = "staged-codex-" + suffix
			require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
				_, stageErr := StageDesktopGrantWithTokensTx(
					tx,
					stagedGrant.PublicId,
					user.Id,
					stagedClaude,
					stagedCodex,
					confirmationHash,
					time.Now().Add(time.Minute).Unix(),
				)
				return stageErr
			}))
			_, err = GetActiveDesktopGrantByTokenId(legacy.ClaudeToken.Id)
			require.NoError(t, err, "staging must not revoke the current device credential")

			confirmed, err := ConfirmDesktopGrant(confirmationHash, 1, 0)
			require.NoError(t, err)
			require.NotNil(t, confirmed)
			assert.Equal(t, DesktopGrantStatusActive, confirmed.Grant.Status)
			assert.Len(t, confirmed.RevokedGrants, 1)
			_, err = GetActiveDesktopGrantByTokenId(legacy.ClaudeToken.Id)
			assert.ErrorIs(t, err, ErrDesktopGrantInactive)
			_, err = GetActiveDesktopGrantByTokenId(confirmed.ClaudeToken.Id)
			require.NoError(t, err)

			keys, err := RevokeDesktopGrant(user.Id, confirmed.Grant.PublicId, "cross_db_test")
			require.NoError(t, err)
			assert.Len(t, keys, 2)
			_, err = GetActiveDesktopGrantByTokenId(confirmed.ClaudeToken.Id)
			assert.ErrorIs(t, err, ErrDesktopGrantInactive)

			confirmationHashes := make([]string, 0, 2)
			for index := 0; index < 2; index++ {
				deviceSuffix := []string{"a", "b"}[index]
				concurrentGrant := &DesktopGrant{
					PublicId:     uuid.NewString(),
					UserId:       user.Id,
					ClientId:     "pcc-agent-desktop",
					DeviceIdHash: common.GenerateHMAC("concurrent-device-" + suffix + deviceSuffix),
					DeviceName:   "Concurrent device",
					Scopes:       policy.Scopes,
				}
				require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
					return CreatePendingDesktopGrantWithTx(tx, concurrentGrant)
				}))
				hash := common.GenerateHMAC("concurrent-confirmation-" + suffix + deviceSuffix)
				claude := policy
				claude.Key = "concurrent-claude-" + suffix + deviceSuffix
				codex := policy
				codex.Key = "concurrent-codex-" + suffix + deviceSuffix
				require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
					_, stageErr := StageDesktopGrantWithTokensTx(
						tx,
						concurrentGrant.PublicId,
						user.Id,
						claude,
						codex,
						hash,
						time.Now().Add(time.Minute).Unix(),
					)
					return stageErr
				}))
				confirmationHashes = append(confirmationHashes, hash)
			}

			startConfirm := make(chan struct{})
			confirmationErrors := make(chan error, len(confirmationHashes))
			var confirmations sync.WaitGroup
			for _, hash := range confirmationHashes {
				confirmations.Add(1)
				go func(confirmationHash string) {
					defer confirmations.Done()
					<-startConfirm
					_, confirmErr := ConfirmDesktopGrant(confirmationHash, 1, 0)
					confirmationErrors <- confirmErr
				}(hash)
			}
			close(startConfirm)
			confirmations.Wait()
			close(confirmationErrors)
			successes := 0
			deviceLimitFailures := 0
			for confirmationErr := range confirmationErrors {
				if confirmationErr == nil {
					successes++
				} else if errors.Is(confirmationErr, ErrDesktopGrantDeviceLimit) {
					deviceLimitFailures++
				} else {
					require.NoError(t, confirmationErr)
				}
			}
			assert.Equal(t, 1, successes)
			assert.Equal(t, 1, deviceLimitFailures)
		})
	}
}
