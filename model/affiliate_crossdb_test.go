package model

import (
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestAffiliateLifecycleConfiguredDatabases(t *testing.T) {
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
			sqlDB.SetMaxOpenConns(8)
			sqlDB.SetMaxIdleConns(8)
			t.Cleanup(func() { _ = sqlDB.Close() })

			models := []interface{}{
				&User{},
				&Option{},
				&TopUp{},
				&AffiliateReferral{},
				&AffiliateSignupReward{},
				&AffiliateSignupRewardTransfer{},
				&AffiliateProfile{},
				&AffiliateCommission{},
				&AffiliateCommissionTransfer{},
				&AffiliateCommissionReversal{},
				&AffiliateAccessChange{},
				&AffiliateConfigChange{},
				&AffiliateOutboxEvent{},
			}
			require.NoError(t, db.AutoMigrate(models...))
			require.NoError(t, db.AutoMigrate(models...), "affiliate migrations must be repeatable")
			if test.databaseType == common.DatabaseTypeMySQL {
				require.NoError(t, checkMySQLChineseSupport(db), "MySQL test schema must match the production charset requirement")
			}

			previousDB := DB
			previousMainType := common.MainDatabaseType()
			previousRedis := common.RedisEnabled
			DB = db
			common.SetMainDatabaseType(test.databaseType)
			initCol()
			common.RedisEnabled = false
			t.Cleanup(func() {
				DB = previousDB
				common.SetMainDatabaseType(previousMainType)
				initCol()
				common.RedisEnabled = previousRedis
			})
			useAffiliateSettingForTopUpTest(t, 0)
			affiliateSetting := operation_setting.GetAffiliateSettingSnapshot()
			affiliateSetting.RegistrationRewardEnabled = true
			affiliateSetting.InviterRewardQuota = 3
			affiliateSetting.InviteeRewardQuota = 2
			affiliateSettingJSON, err := operation_setting.MarshalAffiliateSetting(affiliateSetting)
			require.NoError(t, err)
			require.NoError(t, UpdateOption(operation_setting.AffiliateSettingOptionKey, affiliateSettingJSON))

			suffix := strings.ReplaceAll(uuid.NewString(), "-", "")
			agent := User{
				Username: "affiliate-crossdb-agent-" + suffix,
				Password: "unused",
				Status:   common.UserStatusEnabled,
				Group:    "default",
				AffCode:  "ac" + suffix[:12],
				AffCount: 1,
				Quota:    100,
			}
			require.NoError(t, db.Create(&agent).Error)
			referred := User{
				Username:  "affiliate-crossdb-referred-" + suffix,
				Password:  "unused",
				Status:    common.UserStatusEnabled,
				Group:     "default",
				AffCode:   "rc" + suffix[:12],
				InviterId: agent.Id,
				Quota:     50,
			}
			require.NoError(t, db.Create(&referred).Error)
			now := common.GetTimestamp()
			require.NoError(t, db.Create(&AffiliateReferral{
				InviterUserId:   agent.Id,
				ReferredUserId:  referred.Id,
				AffCodeSnapshot: agent.AffCode,
				BoundAt:         now - 10,
			}).Error)
			require.NoError(t, db.Create(&AffiliateProfile{
				UserId:      agent.Id,
				Access:      AffiliateAccessInherit,
				Status:      AffiliateStatusActive,
				ActivatedAt: now - 5,
				Currency:    "CNY",
				CreatedAt:   now - 5,
				UpdatedAt:   now - 5,
			}).Error)

			const registrationCount = 8
			type registrationResult struct {
				index int
				err   error
			}
			registrationTransactionsReady := make(chan struct{}, registrationCount)
			startRegistrationWrites := make(chan struct{})
			registrationResults := make(chan registrationResult, registrationCount)
			var registrations sync.WaitGroup
			for index := 0; index < registrationCount; index++ {
				registrations.Add(1)
				go func(registrationIndex int) {
					defer registrations.Done()
					newUser := User{
						Username: "affiliate-crossdb-registration-" + suffix + "-" + strconv.Itoa(registrationIndex),
						Password: "unused",
						Status:   common.UserStatusEnabled,
						Group:    "default",
					}
					registrationErr := db.Transaction(func(tx *gorm.DB) error {
						registrationTransactionsReady <- struct{}{}
						<-startRegistrationWrites
						return newUser.InsertWithTx(tx, agent.Id)
					})
					registrationResults <- registrationResult{index: registrationIndex, err: registrationErr}
				}(index)
			}
			registrationStartTimer := time.NewTimer(15 * time.Second)
			for index := 0; index < registrationCount; index++ {
				select {
				case <-registrationTransactionsReady:
				case <-registrationStartTimer.C:
					close(startRegistrationWrites)
					t.Fatalf("only %d of %d registration transactions acquired independent database connections", index, registrationCount)
				}
			}
			if !registrationStartTimer.Stop() {
				select {
				case <-registrationStartTimer.C:
				default:
				}
			}
			close(startRegistrationWrites)
			registrations.Wait()
			close(registrationResults)
			for registration := range registrationResults {
				require.NoErrorf(
					t,
					registration.err,
					"%s concurrent registration %d failed; all %d registrations must commit",
					test.name,
					registration.index,
					registrationCount,
				)
			}
			var reloadedRegistrationAgent User
			require.NoError(t, db.First(&reloadedRegistrationAgent, agent.Id).Error)
			assert.Equal(t, 1+registrationCount, reloadedRegistrationAgent.AffCount)
			assert.Equal(t, registrationCount*3, reloadedRegistrationAgent.AffQuota)
			assert.Equal(t, registrationCount*3, reloadedRegistrationAgent.AffHistoryQuota)

			registeredUsers := make([]User, 0, registrationCount)
			require.NoError(t, db.Where("username LIKE ?", "affiliate-crossdb-registration-"+suffix+"-%").
				Order("id asc").Find(&registeredUsers).Error)
			require.Len(t, registeredUsers, registrationCount)
			registeredUserIds := make([]int, 0, registrationCount)
			affiliateCodes := make(map[string]struct{}, registrationCount)
			for _, registeredUser := range registeredUsers {
				registeredUserIds = append(registeredUserIds, registeredUser.Id)
				assert.Equal(t, agent.Id, registeredUser.InviterId)
				assert.Equal(t, 2, registeredUser.Quota)
				assert.NotEmpty(t, registeredUser.AffCode)
				affiliateCodes[registeredUser.AffCode] = struct{}{}
			}
			assert.Len(t, affiliateCodes, registrationCount)

			var referralCount int64
			require.NoError(t, db.Model(&AffiliateReferral{}).
				Where("inviter_user_id = ?", agent.Id).
				Count(&referralCount).Error)
			assert.Equal(t, int64(1+registrationCount), referralCount)
			var referralEvent AffiliateOutboxEvent
			require.NoError(t, db.Where("user_id = ? AND action = ?", agent.Id, AffiliateEventActionReferralBound).
				Order("id asc").First(&referralEvent).Error)
			assert.Equal(t, "邀请关系已建立", referralEvent.Content)

			type rewardAggregate struct {
				Count int64
				Quota int64
			}
			var inviterRewards rewardAggregate
			require.NoError(t, db.Model(&AffiliateSignupReward{}).
				Select("COUNT(*) AS count, COALESCE(SUM(reward_quota), 0) AS quota").
				Where("beneficiary_user_id = ? AND beneficiary_role = ?", agent.Id, AffiliateSignupRewardRoleInviter).
				Scan(&inviterRewards).Error)
			assert.Equal(t, int64(registrationCount), inviterRewards.Count)
			assert.Equal(t, int64(registrationCount*3), inviterRewards.Quota)
			var inviteeRewards rewardAggregate
			require.NoError(t, db.Model(&AffiliateSignupReward{}).
				Select("COUNT(*) AS count, COALESCE(SUM(reward_quota), 0) AS quota").
				Where("beneficiary_user_id IN ? AND beneficiary_role = ?", registeredUserIds, AffiliateSignupRewardRoleInvitee).
				Scan(&inviteeRewards).Error)
			assert.Equal(t, int64(registrationCount), inviteeRewards.Count)
			assert.Equal(t, int64(registrationCount*2), inviteeRewards.Quota)

			order := TopUp{
				UserId:                  referred.Id,
				Amount:                  50,
				Money:                   50,
				TradeNo:                 "affiliate-crossdb-order-" + suffix,
				PaymentMethod:           "alipay",
				PaymentProvider:         PaymentProviderEpay,
				ExpectedAmountMinor:     5_000,
				PaidCurrency:            "CNY",
				QuotaAmount:             1_000,
				UnitPriceSnapshot:       "1",
				QuotaPerUnitSnapshot:    "100",
				TopUpGroupRatioSnapshot: "1",
				AmountDiscountSnapshot:  "1",
				CommissionEligible:      true,
				CreateTime:              now - 1,
				Status:                  common.TopUpStatusPending,
			}
			require.NoError(t, db.Create(&order).Error)

			t.Cleanup(func() {
				var userIds []int
				_ = db.Model(&User{}).Where("username LIKE ?", "%"+suffix+"%").Pluck("id", &userIds).Error
				_ = db.Unscoped().Where("commission_id IN (?)", db.Model(&AffiliateCommission{}).
					Select("id").Where("agent_user_id = ?", agent.Id)).Delete(&AffiliateCommissionReversal{}).Error
				_ = db.Unscoped().Where("user_id = ?", agent.Id).Delete(&AffiliateCommissionTransfer{}).Error
				_ = db.Unscoped().Where("agent_user_id = ?", agent.Id).Delete(&AffiliateCommission{}).Error
				_ = db.Unscoped().Where("beneficiary_user_id IN ?", userIds).Delete(&AffiliateSignupReward{}).Error
				_ = db.Unscoped().Where("user_id IN ?", userIds).Delete(&AffiliateOutboxEvent{}).Error
				_ = db.Unscoped().Where("id = ?", order.Id).Delete(&TopUp{}).Error
				_ = db.Unscoped().Where("inviter_user_id = ? OR referred_user_id IN ?", agent.Id, userIds).Delete(&AffiliateReferral{}).Error
				_ = db.Unscoped().Where("user_id = ?", agent.Id).Delete(&AffiliateProfile{}).Error
				_ = db.Unscoped().Where("id IN ?", userIds).Delete(&User{}).Error
				_ = db.Where(commonKeyCol+" IN ?", []string{
					operation_setting.AffiliateSettingOptionKey,
					PaymentComplianceConfirmedOptionKey,
					PaymentComplianceTermsVersionOptionKey,
				}).Delete(&Option{}).Error
			})

			settlement := EpaySettlement{
				TradeNo:         order.TradeNo,
				ProviderTradeNo: "affiliate-crossdb-provider-" + suffix,
				PaidAmountMinor: 5_000,
				PaymentMethod:   "alipay",
				CompletedAt:     now,
			}
			start := make(chan struct{})
			results := make(chan EpaySettlementResult, 2)
			errors := make(chan error, 2)
			var callbacks sync.WaitGroup
			for index := 0; index < 2; index++ {
				callbacks.Add(1)
				go func() {
					defer callbacks.Done()
					<-start
					result, completeErr := CompleteEpayTopUp(settlement)
					results <- result
					errors <- completeErr
				}()
			}
			close(start)
			callbacks.Wait()
			close(results)
			close(errors)
			for completeErr := range errors {
				require.NoError(t, completeErr)
			}
			completed := 0
			replayed := 0
			for result := range results {
				if result.AlreadyCompleted {
					replayed++
				} else {
					completed++
				}
			}
			assert.Equal(t, 1, completed)
			assert.Equal(t, 1, replayed)

			var reloadedReferred User
			require.NoError(t, db.First(&reloadedReferred, referred.Id).Error)
			assert.Equal(t, referred.Quota+order.QuotaAmount, reloadedReferred.Quota)
			var commission AffiliateCommission
			require.NoError(t, db.Where("top_up_id = ?", order.Id).First(&commission).Error)
			assert.Equal(t, order.QuotaAmount, commission.PurchasedQuota)
			assert.Equal(t, 500, commission.RewardQuota)
			assert.Equal(t, AffiliateCommissionStatusAvailable, commission.Status)

			raceStart := make(chan struct{})
			transferResults := make(chan AffiliateCommissionTransferResult, 1)
			transferErrors := make(chan error, 1)
			reversalResults := make(chan AffiliateCommissionReversalResult, 1)
			reversalErrors := make(chan error, 1)
			raceReady := make(chan struct{}, 2)
			var race sync.WaitGroup
			race.Add(2)
			go func() {
				defer race.Done()
				raceReady <- struct{}{}
				<-raceStart
				transfer, transferErr := TransferAvailableAffiliateCommissions(agent.Id, "crossdb-"+suffix, now+1)
				transferResults <- transfer
				transferErrors <- transferErr
			}()
			go func() {
				defer race.Done()
				raceReady <- struct{}{}
				<-raceStart
				reversal, reversalErr := ReverseAffiliateCommission(
					commission.Id,
					agent.Id,
					"cross-database commission reversal",
					now+2,
				)
				reversalResults <- reversal
				reversalErrors <- reversalErr
			}()
			<-raceReady
			<-raceReady
			close(raceStart)
			race.Wait()

			transfer := <-transferResults
			transferErr := <-transferErrors
			reversal := <-reversalResults
			reversalErr := <-reversalErrors
			require.NoError(t, reversalErr, "commission reversal must commit regardless of whether transfer wins the row-lock race")
			assert.False(t, reversal.AlreadyReversed)
			assert.Equal(t, AffiliateCommissionStatusReversed, reversal.Commission.Status)

			transferSucceeded := transferErr == nil
			if transferSucceeded {
				assert.False(t, transfer.AlreadyCompleted)
				assert.Equal(t, 500, transfer.Transfer.TransferredQuota)
				assert.Equal(t, int64(500), reversal.Reversal.DebtAddedQuota)
			} else {
				assert.ErrorIs(
					t,
					transferErr,
					ErrAffiliateCommissionNothingToTransfer,
					"the only valid transfer failure is that the reversal won the commission row lock first",
				)
				assert.Zero(t, reversal.Reversal.DebtAddedQuota)
			}

			var reloadedAgent User
			require.NoError(t, db.First(&reloadedAgent, agent.Id).Error)
			var profile AffiliateProfile
			require.NoError(t, db.First(&profile, agent.Id).Error)
			if transferSucceeded {
				assert.Equal(t, agent.Quota+500, reloadedAgent.Quota)
				assert.Equal(t, int64(500), profile.CommissionDebtQuota)
			} else {
				assert.Equal(t, agent.Quota, reloadedAgent.Quota)
				assert.Zero(t, profile.CommissionDebtQuota)
			}
			var reloadedReferredAfterReversal User
			require.NoError(t, db.First(&reloadedReferredAfterReversal, referred.Id).Error)
			assert.Equal(t, referred.Quota+order.QuotaAmount, reloadedReferredAfterReversal.Quota)
			var reloadedOrder TopUp
			require.NoError(t, db.First(&reloadedOrder, order.Id).Error)
			assert.Equal(t, common.TopUpStatusSuccess, reloadedOrder.Status)
			var reloadedCommission AffiliateCommission
			require.NoError(t, db.First(&reloadedCommission, commission.Id).Error)
			assert.Equal(t, AffiliateCommissionStatusReversed, reloadedCommission.Status)

			var transferCount int64
			require.NoError(t, db.Model(&AffiliateCommissionTransfer{}).
				Where("user_id = ?", agent.Id).Count(&transferCount).Error)
			if transferSucceeded {
				assert.Equal(t, int64(1), transferCount)
			} else {
				assert.Zero(t, transferCount)
			}
			var reversalCount int64
			require.NoError(t, db.Model(&AffiliateCommissionReversal{}).
				Where("commission_id = ?", commission.Id).Count(&reversalCount).Error)
			assert.Equal(t, int64(1), reversalCount)

			totals, err := GetVerifiedTopUpPaymentTotals(referred.Id, PaymentProviderEpay)
			require.NoError(t, err)
			require.Len(t, totals, 1)
			assert.Equal(t, order.ExpectedAmountMinor, totals[0].AmountMinor)
			reversalReplay, replayErr := ReverseAffiliateCommission(
				commission.Id,
				agent.Id,
				"cross-database commission reversal",
				now+3,
			)
			require.NoError(t, replayErr)
			assert.True(t, reversalReplay.AlreadyReversed)
			assert.Equal(t, reversal.Reversal.Id, reversalReplay.Reversal.Id)
		})
	}
}
