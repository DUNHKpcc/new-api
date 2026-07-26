package model

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
)

const (
	desktopGrantCacheSchema    = 1
	desktopGrantActiveCacheTTL = 5 * time.Second
	desktopGrantDenyCacheTTL   = time.Minute
)

var errDesktopGrantCacheObservationStale = errors.New("desktop grant cache observation is stale")

type desktopGrantCacheEntry struct {
	TokenID       int
	GrantID       int64
	PublicID      string
	UserID        int
	ClientID      string
	DeviceIDHash  string
	DeviceName    string
	Platform      string
	AppVersion    string
	ClaudeTokenID int
	CodexTokenID  int
	Scopes        string
	Status        string
	Version       int64
	CreatedTime   int64
	LastUsedTime  int64
	ExpiredTime   int64
	RevokedTime   int64
	RevokeReason  string
	ConfirmedTime int64
	CacheSchema   int
}

func desktopGrantCacheKey(tokenID int) string {
	digest := common.GenerateHMACWithKey(
		[]byte("desktop-grant-cache-v1:"+common.SessionSecret),
		fmt.Sprintf("%d", tokenID),
	)
	return "auth:desktop-grant:" + digest
}

func desktopGrantCacheDeadline() time.Time {
	return time.Now().Add(desktopGrantActiveCacheTTL)
}

func desktopGrantCacheEntryFor(grant *DesktopGrant, tokenID int) *desktopGrantCacheEntry {
	if grant == nil {
		return nil
	}
	entry := &desktopGrantCacheEntry{
		TokenID:       tokenID,
		GrantID:       grant.Id,
		PublicID:      grant.PublicId,
		UserID:        grant.UserId,
		ClientID:      grant.ClientId,
		DeviceIDHash:  grant.DeviceIdHash,
		DeviceName:    grant.DeviceName,
		Platform:      grant.Platform,
		AppVersion:    grant.AppVersion,
		Scopes:        grant.Scopes,
		Status:        grant.Status,
		Version:       grant.Version,
		CreatedTime:   grant.CreatedTime,
		LastUsedTime:  grant.LastUsedTime,
		ExpiredTime:   grant.ExpiredTime,
		RevokedTime:   grant.RevokedTime,
		RevokeReason:  grant.RevokeReason,
		ConfirmedTime: grant.ConfirmedTime,
		CacheSchema:   desktopGrantCacheSchema,
	}
	if grant.TokenId != nil {
		entry.ClaudeTokenID = *grant.TokenId
	}
	if grant.CodexTokenId != nil {
		entry.CodexTokenID = *grant.CodexTokenId
	}
	return entry
}

func (entry *desktopGrantCacheEntry) grant() *DesktopGrant {
	activeSlot := 1
	grant := &DesktopGrant{
		Id:            entry.GrantID,
		PublicId:      entry.PublicID,
		UserId:        entry.UserID,
		ClientId:      entry.ClientID,
		DeviceIdHash:  entry.DeviceIDHash,
		DeviceName:    entry.DeviceName,
		Platform:      entry.Platform,
		AppVersion:    entry.AppVersion,
		Scopes:        entry.Scopes,
		Status:        entry.Status,
		Version:       entry.Version,
		CreatedTime:   entry.CreatedTime,
		LastUsedTime:  entry.LastUsedTime,
		ExpiredTime:   entry.ExpiredTime,
		RevokedTime:   entry.RevokedTime,
		RevokeReason:  entry.RevokeReason,
		ConfirmedTime: entry.ConfirmedTime,
	}
	if entry.ClaudeTokenID > 0 {
		grant.TokenId = &entry.ClaudeTokenID
	}
	if entry.CodexTokenID > 0 {
		grant.CodexTokenId = &entry.CodexTokenID
	}
	if entry.Status == DesktopGrantStatusActive {
		grant.ActiveSlot = &activeSlot
	}
	return grant
}

func getDesktopGrantCache(tokenID int) (*desktopGrantCacheEntry, error) {
	var entry desktopGrantCacheEntry
	if err := common.RedisHGetObj(desktopGrantCacheKey(tokenID), &entry); err != nil {
		return nil, err
	}
	if entry.CacheSchema != desktopGrantCacheSchema || entry.TokenID != tokenID ||
		entry.GrantID <= 0 || entry.UserID <= 0 || entry.Version <= 0 {
		return nil, fmt.Errorf("desktop grant cache schema is stale")
	}
	if entry.Status != DesktopGrantStatusActive || entry.ExpiredTime <= time.Now().Unix() {
		return nil, ErrDesktopGrantInactive
	}
	return &entry, nil
}

func writeDesktopGrantCache(entry *desktopGrantCacheEntry, cacheDeadline time.Time) error {
	if entry == nil || !common.RedisEnabled || common.RDB == nil {
		return nil
	}
	now := time.Now()
	var expiration int64
	if entry.Status == DesktopGrantStatusActive {
		if cacheDeadline.IsZero() {
			return ErrDesktopGrantInvalid
		}
		cacheExpiresAt := cacheDeadline
		grantExpiresAt := time.Unix(entry.ExpiredTime, 0)
		if grantExpiresAt.Before(cacheExpiresAt) {
			cacheExpiresAt = grantExpiresAt
		}
		if !cacheExpiresAt.After(now) {
			return ErrDesktopGrantInactive
		}
		expiration = cacheExpiresAt.UnixMilli()
	} else {
		ttl := desktopGrantDenyCacheTTL
		if entry.ExpiredTime > 0 {
			remaining := time.Until(time.Unix(entry.ExpiredTime, 0))
			if remaining > 0 && remaining < ttl {
				ttl = remaining
			}
		}
		if ttl < time.Millisecond {
			ttl = time.Second
		}
		expiration = ttl.Milliseconds()
	}
	entry.CacheSchema = desktopGrantCacheSchema
	const script = `
local current_status = redis.call('HGET', KEYS[1], 'Status')
local current_version = tonumber(redis.call('HGET', KEYS[1], 'Version') or '0')
if ARGV[10] == 'active' and current_status and current_status ~= 'active'
  and current_version >= tonumber(ARGV[11]) then
  return 0
end
if current_version > tonumber(ARGV[11]) then
  return 0
end
redis.call('HSET', KEYS[1],
  'TokenID', ARGV[1], 'GrantID', ARGV[2], 'PublicID', ARGV[3],
  'UserID', ARGV[4], 'ClientID', ARGV[5], 'DeviceIDHash', ARGV[6],
  'DeviceName', ARGV[7], 'Platform', ARGV[8], 'AppVersion', ARGV[9],
  'Status', ARGV[10], 'Version', ARGV[11], 'ClaudeTokenID', ARGV[12],
  'CodexTokenID', ARGV[13], 'Scopes', ARGV[14], 'CreatedTime', ARGV[15],
  'LastUsedTime', ARGV[16], 'ExpiredTime', ARGV[17], 'RevokedTime', ARGV[18],
  'RevokeReason', ARGV[19], 'ConfirmedTime', ARGV[20], 'CacheSchema', ARGV[21])
if ARGV[10] == 'active' then
  redis.call('PEXPIREAT', KEYS[1], ARGV[22])
else
  redis.call('PEXPIRE', KEYS[1], ARGV[22])
end
return 1`
	result, err := common.RDB.Eval(
		context.Background(),
		script,
		[]string{desktopGrantCacheKey(entry.TokenID)},
		entry.TokenID,
		entry.GrantID,
		entry.PublicID,
		entry.UserID,
		entry.ClientID,
		entry.DeviceIDHash,
		entry.DeviceName,
		entry.Platform,
		entry.AppVersion,
		entry.Status,
		entry.Version,
		entry.ClaudeTokenID,
		entry.CodexTokenID,
		entry.Scopes,
		entry.CreatedTime,
		entry.LastUsedTime,
		entry.ExpiredTime,
		entry.RevokedTime,
		entry.RevokeReason,
		entry.ConfirmedTime,
		entry.CacheSchema,
		expiration,
	).Int()
	if err != nil {
		return err
	}
	if result == 0 {
		return ErrDesktopGrantInactive
	}
	if entry.Status == DesktopGrantStatusActive && !time.Now().Before(cacheDeadline) {
		return errDesktopGrantCacheObservationStale
	}
	return nil
}

func confirmDesktopGrantActiveSnapshot(grant *DesktopGrant, tokenID int) error {
	if grant == nil || grant.Id <= 0 || grant.Version <= 0 || tokenID <= 0 {
		return ErrDesktopGrantInvalid
	}
	var count int64
	err := DB.Model(&DesktopGrant{}).
		Where(
			"id = ? AND version = ? AND status = ? AND active_slot = ? AND expired_time > ? AND (token_id = ? OR codex_token_id = ?)",
			grant.Id,
			grant.Version,
			DesktopGrantStatusActive,
			1,
			time.Now().Unix(),
			tokenID,
			tokenID,
		).
		Count(&count).Error
	if err != nil {
		return err
	}
	if count != 1 {
		return ErrDesktopGrantInactive
	}
	return nil
}

func publishDesktopGrantState(grant *DesktopGrant) error {
	if grant == nil || !common.RedisEnabled {
		return nil
	}
	for _, tokenID := range desktopGrantTokenIds(grant) {
		deadline := time.Time{}
		if grant.Status == DesktopGrantStatusActive {
			deadline = desktopGrantCacheDeadline()
		}
		if err := writeDesktopGrantCache(desktopGrantCacheEntryFor(grant, tokenID), deadline); err != nil {
			return err
		}
	}
	return nil
}

// PublishDesktopGrantActivation publishes deny states before the new active
// state so stale credentials stop at every node before token cache deletion.
func PublishDesktopGrantActivation(activation *DesktopGrantActivationResult) error {
	if activation == nil {
		return ErrDesktopGrantInvalid
	}
	for i := range activation.RevokedGrants {
		if err := publishDesktopGrantState(&activation.RevokedGrants[i]); err != nil {
			return err
		}
	}
	return publishDesktopGrantState(activation.Grant)
}
