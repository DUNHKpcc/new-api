package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackfillAffiliateReferralsPreservesLegacyRelationshipsWithoutRewards(t *testing.T) {
	truncateTables(t)
	inviter := User{
		Username: "legacy-affiliate-inviter",
		AffCode:  "old-code",
		Status:   common.UserStatusEnabled,
		AffCount: 99,
	}
	require.NoError(t, DB.Create(&inviter).Error)
	referred := []User{
		{Username: "legacy-affiliate-one", AffCode: "legacy-one", Status: common.UserStatusEnabled, InviterId: inviter.Id, CreatedAt: 100},
		{Username: "legacy-affiliate-two", AffCode: "legacy-two", Status: common.UserStatusEnabled, InviterId: inviter.Id, CreatedAt: 200},
		{Username: "legacy-affiliate-orphan", AffCode: "legacy-orphan", Status: common.UserStatusEnabled, InviterId: 999999, CreatedAt: 300},
	}
	require.NoError(t, DB.Create(&referred).Error)
	require.NoError(t, DB.Delete(&referred[1]).Error)

	require.NoError(t, backfillAffiliateReferrals())
	require.NoError(t, backfillAffiliateReferrals())

	var referrals []AffiliateReferral
	require.NoError(t, DB.Order("referred_user_id asc").Find(&referrals).Error)
	require.Len(t, referrals, 2)
	assert.Equal(t, "old-code", referrals[0].AffCodeSnapshot)
	assert.Equal(t, int64(100), referrals[0].BoundAt)
	assert.Equal(t, int64(200), referrals[1].BoundAt)

	var reloadedInviter User
	require.NoError(t, DB.First(&reloadedInviter, inviter.Id).Error)
	assert.Equal(t, 99, reloadedInviter.AffCount)
	count, err := CountAffiliateReferrals(inviter.Id)
	require.NoError(t, err)
	assert.Equal(t, int64(99), count)

	var rewardCount int64
	require.NoError(t, DB.Model(&AffiliateSignupReward{}).Count(&rewardCount).Error)
	assert.Zero(t, rewardCount)
	var eventCount int64
	require.NoError(t, DB.Model(&AffiliateOutboxEvent{}).Count(&eventCount).Error)
	assert.Zero(t, eventCount)

	var marker Option
	require.NoError(t, DB.Where(commonKeyCol+" = ?", affiliateReferralBackfillMarker).First(&marker).Error)
	assert.Equal(t, "done", marker.Value)
}
