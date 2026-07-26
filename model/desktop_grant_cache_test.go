package model

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDesktopGrantCacheRejectsDelayedActiveWriteAfterRevocation(t *testing.T) {
	server := useUserCacheMiniRedis(t)
	oldSecret := common.SessionSecret
	common.SessionSecret = "desktop-grant-cache-test-secret"
	t.Cleanup(func() {
		common.SessionSecret = oldSecret
	})

	tokenID := 4101
	activeSlot := 1
	active := &DesktopGrant{
		Id:            51,
		PublicId:      "desktop-cache-active",
		UserId:        71,
		ClientId:      "pcc-agent-desktop",
		DeviceIdHash:  "device-hash",
		DeviceName:    "Cache device",
		TokenId:       &tokenID,
		Scopes:        "relay account.read usage.read",
		Status:        DesktopGrantStatusActive,
		ActiveSlot:    &activeSlot,
		Version:       1,
		CreatedTime:   time.Now().Unix(),
		LastUsedTime:  time.Now().Unix(),
		ExpiredTime:   time.Now().Add(time.Hour).Unix(),
		ConfirmedTime: time.Now().Unix(),
	}
	require.NoError(t, writeDesktopGrantCache(
		desktopGrantCacheEntryFor(active, tokenID),
		desktopGrantCacheDeadline(),
	))
	cached, err := getDesktopGrantCache(tokenID)
	require.NoError(t, err)
	assert.Equal(t, active.PublicId, cached.PublicID)
	assert.NotEqual(t, "auth:desktop-grant:4101", desktopGrantCacheKey(tokenID))

	revoked := *active
	revoked.Status = DesktopGrantStatusRevoked
	revoked.ActiveSlot = nil
	revoked.Version = 2
	revoked.RevokedTime = time.Now().Unix()
	revoked.RevokeReason = "test_revoked"
	require.NoError(t, writeDesktopGrantCache(
		desktopGrantCacheEntryFor(&revoked, tokenID),
		time.Time{},
	))

	err = writeDesktopGrantCache(
		desktopGrantCacheEntryFor(active, tokenID),
		desktopGrantCacheDeadline(),
	)
	assert.ErrorIs(t, err, ErrDesktopGrantInactive)
	_, err = getDesktopGrantCache(tokenID)
	assert.ErrorIs(t, err, ErrDesktopGrantInactive)

	version, err := common.RDB.HGet(t.Context(), desktopGrantCacheKey(tokenID), "Version").Result()
	require.NoError(t, err)
	assert.Equal(t, "2", version)
	assert.True(t, server.Exists(desktopGrantCacheKey(tokenID)))
}

func TestDesktopGrantCacheAllowsConfirmedVersionToReplaceAwaitingDeny(t *testing.T) {
	useUserCacheMiniRedis(t)
	oldSecret := common.SessionSecret
	common.SessionSecret = "desktop-grant-cache-confirmation-test-secret"
	t.Cleanup(func() {
		common.SessionSecret = oldSecret
	})

	tokenID := 4201
	awaiting := &DesktopGrant{
		Id:           52,
		PublicId:     "desktop-cache-awaiting",
		UserId:       72,
		ClientId:     "pcc-agent-desktop",
		DeviceIdHash: "device-hash",
		DeviceName:   "Confirmation device",
		TokenId:      &tokenID,
		Scopes:       "relay account.read usage.read",
		Status:       DesktopGrantStatusAwaitingConfirmation,
		Version:      1,
		CreatedTime:  time.Now().Unix(),
		ExpiredTime:  time.Now().Add(time.Hour).Unix(),
	}
	require.NoError(t, writeDesktopGrantCache(
		desktopGrantCacheEntryFor(awaiting, tokenID),
		time.Time{},
	))

	activeSlot := 1
	confirmed := *awaiting
	confirmed.Status = DesktopGrantStatusActive
	confirmed.ActiveSlot = &activeSlot
	confirmed.Version = 2
	confirmed.ConfirmedTime = time.Now().Unix()
	require.NoError(t, writeDesktopGrantCache(
		desktopGrantCacheEntryFor(&confirmed, tokenID),
		desktopGrantCacheDeadline(),
	))

	cached, err := getDesktopGrantCache(tokenID)
	require.NoError(t, err)
	assert.EqualValues(t, 2, cached.Version)
	assert.Equal(t, DesktopGrantStatusActive, cached.Status)
}
