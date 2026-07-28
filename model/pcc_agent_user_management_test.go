package model

import (
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func createPccAgentManagementUser(t *testing.T, username string) User {
	t.Helper()
	user := User{
		Username:    username,
		Password:    "password123",
		DisplayName: username,
		Email:       username + "@example.com",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		AffCode:     username,
	}
	require.NoError(t, DB.Create(&user).Error)
	return user
}

func createPccAgentManagementGrant(t *testing.T, userId int, status string, index int) {
	t.Helper()
	activeSlot := (*int)(nil)
	expiredTime := int64(0)
	if status == DesktopGrantStatusActive {
		activeSlot = common.GetPointer(1)
		expiredTime = common.GetTimestamp() + 3600
	}
	require.NoError(t, DB.Create(&DesktopGrant{
		PublicId:     fmt.Sprintf("management-grant-%d-%d", userId, index),
		UserId:       userId,
		ClientId:     "pcc-agent-desktop",
		DeviceIdHash: fmt.Sprintf("%064d", userId*10+index),
		DeviceName:   fmt.Sprintf("Device %d", index),
		Scopes:       "relay account.read usage.read",
		Status:       status,
		ActiveSlot:   activeSlot,
		ExpiredTime:  expiredTime,
		CreatedTime:  int64(1_700_000_000 + index),
	}).Error)
}

func TestSearchUsersPccAgentScopeAndSummary(t *testing.T) {
	truncateTables(t)

	pendingOnly := createPccAgentManagementUser(t, "pcc-pending")
	awaiting := createPccAgentManagementUser(t, "pcc-awaiting")
	active := createPccAgentManagementUser(t, "pcc-active")
	revoked := createPccAgentManagementUser(t, "pcc-revoked")
	expired := createPccAgentManagementUser(t, "pcc-expired")
	giftOnly := createPccAgentManagementUser(t, "pcc-gift-only")
	unrelated := createPccAgentManagementUser(t, "pcc-unrelated")

	createPccAgentManagementGrant(t, pendingOnly.Id, DesktopGrantStatusPending, 1)
	createPccAgentManagementGrant(t, awaiting.Id, DesktopGrantStatusAwaitingConfirmation, 1)
	createPccAgentManagementGrant(t, active.Id, DesktopGrantStatusActive, 1)
	createPccAgentManagementGrant(t, active.Id, DesktopGrantStatusActive, 2)
	createPccAgentManagementGrant(t, active.Id, DesktopGrantStatusRevoked, 3)
	createPccAgentManagementGrant(t, active.Id, DesktopGrantStatusActive, 4)
	require.NoError(t, DB.Model(&DesktopGrant{}).
		Where("public_id = ?", fmt.Sprintf("management-grant-%d-%d", active.Id, 4)).
		Update("expired_time", common.GetTimestamp()-1).Error)
	createPccAgentManagementGrant(t, active.Id, DesktopGrantStatusActive, 5)
	require.NoError(t, DB.Model(&DesktopGrant{}).
		Where("public_id = ?", fmt.Sprintf("management-grant-%d-%d", active.Id, 5)).
		Update("active_slot", nil).Error)
	createPccAgentManagementGrant(t, revoked.Id, DesktopGrantStatusRevoked, 1)
	createPccAgentManagementGrant(t, expired.Id, DesktopGrantStatusExpired, 1)

	plan := SubscriptionPlan{
		Title:              "PccAgent Daily Gift",
		DurationUnit:       SubscriptionDurationMonth,
		DurationValue:      1,
		Enabled:            true,
		MaxPurchasePerUser: 1,
		TotalAmount:        5_000,
	}
	require.NoError(t, DB.Create(&plan).Error)
	gift := UserSubscription{
		UserId:        active.Id,
		PlanId:        plan.Id,
		AmountTotal:   5_000,
		AmountUsed:    1_250,
		StartTime:     1_700_000_000,
		EndTime:       1_800_000_000,
		Status:        "active",
		Source:        UserSubscriptionSourcePccAgentGift,
		NextResetTime: 1_700_086_400,
	}
	require.NoError(t, DB.Create(&gift).Error)
	require.NoError(t, DB.Create(&UserSubscription{
		UserId:      giftOnly.Id,
		PlanId:      plan.Id,
		AmountTotal: 5_000,
		AmountUsed:  5_000,
		StartTime:   1_600_000_000,
		EndTime:     1_650_000_000,
		Status:      "cancelled",
		Source:      UserSubscriptionSourcePccAgentGift,
	}).Error)
	require.NoError(t, ClaimExternalIdentityWithTx(
		DB,
		ExternalIdentityProviderWeChatUnionID,
		"management-union-id-must-not-leak",
		active.Id,
	))

	users, total, err := SearchUsers(
		"pcc-",
		"",
		nil,
		nil,
		0,
		20,
		true,
		NewUserSortOptions("id", "asc"),
	)
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Equal(t, []int{awaiting.Id, active.Id, revoked.Id, expired.Id, giftOnly.Id}, collectUserIDs(users))
	assert.NotContains(t, collectUserIDs(users), pendingOnly.Id)
	assert.NotContains(t, collectUserIDs(users), unrelated.Id)

	activeSummary := users[1].PccAgentSummary
	require.NotNil(t, activeSummary)
	assert.EqualValues(t, 2, activeSummary.ActiveDeviceCount)
	assert.True(t, activeSummary.WeChatVerified)
	require.NotNil(t, activeSummary.Gift)
	assert.Equal(t, gift.Id, activeSummary.Gift.SubscriptionId)
	assert.Equal(t, plan.Id, activeSummary.Gift.PlanId)
	assert.Equal(t, plan.Title, activeSummary.Gift.PlanTitle)
	assert.EqualValues(t, 5_000, activeSummary.Gift.AmountTotal)
	assert.EqualValues(t, 1_250, activeSummary.Gift.AmountUsed)
	assert.EqualValues(t, 3_750, activeSummary.Gift.AmountRemaining)
	assert.EqualValues(t, 1_700_086_400, activeSummary.Gift.NextResetTime)
	assert.EqualValues(t, 1_800_000_000, activeSummary.Gift.EndTime)

	awaitingSummary := users[0].PccAgentSummary
	require.NotNil(t, awaitingSummary)
	assert.Zero(t, awaitingSummary.ActiveDeviceCount)
	assert.False(t, awaitingSummary.WeChatVerified)
	assert.Nil(t, awaitingSummary.Gift)

	giftOnlySummary := users[4].PccAgentSummary
	require.NotNil(t, giftOnlySummary)
	assert.Zero(t, giftOnlySummary.ActiveDeviceCount)
	require.NotNil(t, giftOnlySummary.Gift)
	assert.Equal(t, "cancelled", giftOnlySummary.Gift.Status)

	payload, err := common.Marshal(users)
	require.NoError(t, err)
	assert.NotContains(t, string(payload), "management-union-id-must-not-leak")
}

func TestSearchUsersPccAgentScopePreservesFiltersAndPagination(t *testing.T) {
	truncateTables(t)

	first := createPccAgentManagementUser(t, "pcc-filter-first")
	second := createPccAgentManagementUser(t, "pcc-filter-second")
	third := createPccAgentManagementUser(t, "pcc-filter-third")
	require.NoError(t, DB.Model(&second).Updates(map[string]interface{}{
		"status": common.UserStatusDisabled,
		"group":  "vip",
	}).Error)
	require.NoError(t, DB.Model(&third).Update("group", "vip").Error)
	createPccAgentManagementGrant(t, first.Id, DesktopGrantStatusActive, 1)
	createPccAgentManagementGrant(t, second.Id, DesktopGrantStatusActive, 1)
	createPccAgentManagementGrant(t, third.Id, DesktopGrantStatusActive, 1)

	status := common.UserStatusEnabled
	users, total, err := SearchUsers(
		"pcc-filter",
		"vip",
		nil,
		&status,
		0,
		1,
		true,
		NewUserSortOptions("id", "asc"),
	)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, users, 1)
	assert.Equal(t, third.Id, users[0].Id)
}

func TestSearchUsersDoesNotAttachPccAgentSummaryWithoutScope(t *testing.T) {
	truncateTables(t)

	user := createPccAgentManagementUser(t, "pcc-normal-search")
	createPccAgentManagementGrant(t, user.Id, DesktopGrantStatusActive, 1)

	users, total, err := SearchUsers(
		"pcc-normal-search",
		"",
		nil,
		nil,
		0,
		20,
		false,
		NewUserSortOptions("id", "asc"),
	)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, users, 1)
	assert.Nil(t, users[0].PccAgentSummary)
}

func TestSearchUsersPccAgentSummaryUsesBoundedQueries(t *testing.T) {
	truncateTables(t)

	for index := 0; index < 3; index++ {
		user := createPccAgentManagementUser(t, fmt.Sprintf("pcc-bounded-%d", index))
		createPccAgentManagementGrant(t, user.Id, DesktopGrantStatusActive, 1)
	}

	queryCount := 0
	const callbackName = "test:count_pcc_agent_management_queries"
	require.NoError(t, DB.Callback().Query().Before("gorm:query").Register(callbackName, func(*gorm.DB) {
		queryCount++
	}))
	t.Cleanup(func() {
		_ = DB.Callback().Query().Remove(callbackName)
	})

	oneUser, total, err := SearchUsers(
		"pcc-bounded",
		"",
		nil,
		nil,
		0,
		1,
		true,
		NewUserSortOptions("id", "asc"),
	)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	require.Len(t, oneUser, 1)
	singleUserQueryCount := queryCount

	queryCount = 0
	users, total, err := SearchUsers(
		"pcc-bounded",
		"",
		nil,
		nil,
		0,
		20,
		true,
		NewUserSortOptions("id", "asc"),
	)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	require.Len(t, users, 3)
	assert.Equal(t, singleUserQueryCount, queryCount)
}
