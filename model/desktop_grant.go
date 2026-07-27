package model

import (
	"errors"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	DesktopGrantStatusPending              = "pending"
	DesktopGrantStatusAwaitingConfirmation = "awaiting_confirmation"
	DesktopGrantStatusActive               = "active"
	DesktopGrantStatusRevoked              = "revoked"
	DesktopGrantStatusExpired              = "expired"
)

var (
	ErrDesktopGrantInvalid             = errors.New("desktop grant is invalid")
	ErrDesktopGrantNotFound            = errors.New("desktop grant was not found")
	ErrDesktopGrantInactive            = errors.New("desktop grant is inactive")
	ErrDesktopGrantNotRevoked          = errors.New("desktop grant is not revoked")
	ErrDesktopGrantDeviceLimit         = errors.New("desktop grant device limit reached")
	ErrDesktopGrantConfirmationInvalid = errors.New("desktop grant confirmation is invalid")
	ErrDesktopGrantConfirmationExpired = errors.New("desktop grant confirmation has expired")
)

// DesktopGrant records a user's explicit authorization of a public desktop
// client. DeviceIDHash is an HMAC; the raw installation identifier is never
// persisted.
type DesktopGrant struct {
	Id                    int64  `json:"-" gorm:"primaryKey"`
	PublicId              string `json:"public_id" gorm:"type:varchar(64);not null;uniqueIndex"`
	UserId                int    `json:"-" gorm:"not null;index;uniqueIndex:idx_desktop_grants_active_device,priority:1"`
	ClientId              string `json:"client_id" gorm:"type:varchar(64);not null;uniqueIndex:idx_desktop_grants_active_device,priority:2"`
	DeviceIdHash          string `json:"-" gorm:"type:char(64);not null;index;uniqueIndex:idx_desktop_grants_active_device,priority:3"`
	DeviceName            string `json:"device_name" gorm:"type:varchar(128);not null"`
	Platform              string `json:"platform" gorm:"type:varchar(64)"`
	AppVersion            string `json:"app_version" gorm:"type:varchar(32)"`
	TokenId               *int   `json:"-" gorm:"uniqueIndex"`
	CodexTokenId          *int   `json:"-" gorm:"uniqueIndex"`
	Scopes                string `json:"scopes" gorm:"type:varchar(255);not null"`
	Status                string `json:"status" gorm:"type:varchar(32);not null;index"`
	ActiveSlot            *int   `json:"-" gorm:"uniqueIndex:idx_desktop_grants_active_device,priority:4"`
	Version               int64  `json:"-" gorm:"type:bigint;not null;default:1"`
	ConfirmationHash      string `json:"-" gorm:"type:char(64);index"`
	ConfirmationExpiresAt int64  `json:"-" gorm:"type:bigint;not null;default:0;index"`
	ConfirmedTime         int64  `json:"-" gorm:"type:bigint;not null;default:0"`
	CreatedTime           int64  `json:"created_time" gorm:"type:bigint;not null;index"`
	LastUsedTime          int64  `json:"last_used_time" gorm:"type:bigint;not null;default:0"`
	ExpiredTime           int64  `json:"expired_time" gorm:"type:bigint;not null;default:0;index"`
	RevokedTime           int64  `json:"revoked_time" gorm:"type:bigint;not null;default:0"`
	RevokeReason          string `json:"revoke_reason,omitempty" gorm:"type:varchar(64)"`
}

func (DesktopGrant) TableName() string {
	return "desktop_grants"
}

type DesktopGrantTokenPolicy struct {
	Name           string
	Key            string
	Group          string
	ModelLimits    string
	Scopes         string
	ExpiredTime    int64
	UnlimitedQuota bool
}

type DesktopGrantActivationResult struct {
	Grant            *DesktopGrant
	ClaudeToken      *Token
	CodexToken       *Token
	RevokedTokenKeys []string
	RevokedGrants    []DesktopGrant
}

func LockDesktopGrantUserWithTx(tx *gorm.DB, userID int) error {
	if tx == nil || userID <= 0 {
		return ErrDesktopGrantInvalid
	}
	var user User
	return lockForUpdate(tx).
		Select("id").
		Where("id = ?", userID).
		First(&user).Error
}

func CreatePendingDesktopGrantWithTx(tx *gorm.DB, grant *DesktopGrant) error {
	if tx == nil || grant == nil || grant.PublicId == "" || grant.UserId <= 0 ||
		grant.ClientId == "" || grant.DeviceIdHash == "" || grant.DeviceName == "" ||
		grant.Scopes == "" {
		return ErrDesktopGrantInvalid
	}
	if grant.CreatedTime == 0 {
		grant.CreatedTime = time.Now().Unix()
	}
	grant.TokenId = nil
	grant.CodexTokenId = nil
	grant.ActiveSlot = nil
	grant.Version = 1
	grant.Status = DesktopGrantStatusPending
	grant.ConfirmationHash = ""
	grant.ConfirmationExpiresAt = 0
	grant.ConfirmedTime = 0
	grant.ExpiredTime = 0
	grant.RevokedTime = 0
	grant.RevokeReason = ""
	return tx.Create(grant).Error
}

// ActivateDesktopGrantWithTokensTx replaces the active grant for one device
// and creates its Claude and Codex API tokens in the same transaction.
func ActivateDesktopGrantWithTokensTx(
	tx *gorm.DB,
	publicId string,
	userId int,
	claudePolicy DesktopGrantTokenPolicy,
	codexPolicy DesktopGrantTokenPolicy,
) (*DesktopGrantActivationResult, error) {
	now := time.Now().Unix()
	if tx == nil || publicId == "" || userId <= 0 ||
		claudePolicy.Name == "" || claudePolicy.Key == "" || claudePolicy.Scopes == "" ||
		codexPolicy.Name == "" || codexPolicy.Key == "" || codexPolicy.Scopes == "" ||
		claudePolicy.ExpiredTime <= now || codexPolicy.ExpiredTime != claudePolicy.ExpiredTime ||
		codexPolicy.Scopes != claudePolicy.Scopes {
		return nil, ErrDesktopGrantInvalid
	}

	var pending DesktopGrant
	if err := lockForUpdate(tx).
		Where("public_id = ? AND user_id = ? AND status = ?", publicId, userId, DesktopGrantStatusPending).
		First(&pending).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDesktopGrantNotFound
		}
		return nil, err
	}

	revokedGrants, revokedKeys, err := revokeActiveDesktopGrantsForDeviceWithTx(tx, &pending, "reauthorized")
	if err != nil {
		return nil, err
	}

	tokens := []*Token{
		{
			UserId:             userId,
			Key:                claudePolicy.Key,
			Status:             common.TokenStatusEnabled,
			Name:               claudePolicy.Name,
			CreatedTime:        now,
			AccessedTime:       now,
			ExpiredTime:        claudePolicy.ExpiredTime,
			UnlimitedQuota:     claudePolicy.UnlimitedQuota,
			ModelLimitsEnabled: true,
			ModelLimits:        claudePolicy.ModelLimits,
			Group:              claudePolicy.Group,
		},
		{
			UserId:             userId,
			Key:                codexPolicy.Key,
			Status:             common.TokenStatusEnabled,
			Name:               codexPolicy.Name,
			CreatedTime:        now,
			AccessedTime:       now,
			ExpiredTime:        codexPolicy.ExpiredTime,
			UnlimitedQuota:     codexPolicy.UnlimitedQuota,
			ModelLimitsEnabled: true,
			ModelLimits:        codexPolicy.ModelLimits,
			Group:              codexPolicy.Group,
		},
	}
	if err := tx.Create(&tokens).Error; err != nil {
		return nil, err
	}
	claudeToken, codexToken := tokens[0], tokens[1]

	activeSlot := 1
	result := tx.Model(&DesktopGrant{}).
		Where("id = ? AND status = ? AND token_id IS NULL AND codex_token_id IS NULL", pending.Id, DesktopGrantStatusPending).
		Updates(map[string]interface{}{
			"token_id":                claudeToken.Id,
			"codex_token_id":          codexToken.Id,
			"scopes":                  claudePolicy.Scopes,
			"status":                  DesktopGrantStatusActive,
			"active_slot":             activeSlot,
			"last_used_time":          now,
			"expired_time":            claudePolicy.ExpiredTime,
			"confirmation_hash":       "",
			"confirmation_expires_at": 0,
			"confirmed_time":          now,
		})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected != 1 {
		return nil, ErrDesktopGrantInactive
	}

	pending.TokenId = &claudeToken.Id
	pending.CodexTokenId = &codexToken.Id
	pending.Scopes = claudePolicy.Scopes
	pending.Status = DesktopGrantStatusActive
	pending.ActiveSlot = &activeSlot
	pending.LastUsedTime = now
	pending.ExpiredTime = claudePolicy.ExpiredTime
	pending.ConfirmationHash = ""
	pending.ConfirmationExpiresAt = 0
	pending.ConfirmedTime = now
	return &DesktopGrantActivationResult{
		Grant:            &pending,
		ClaudeToken:      claudeToken,
		CodexToken:       codexToken,
		RevokedTokenKeys: revokedKeys,
		RevokedGrants:    revokedGrants,
	}, nil
}

// StageDesktopGrantWithTokensTx creates disabled engine tokens and binds them
// to a one-time confirmation digest. Existing active credentials are left
// untouched until the desktop client confirms durable local storage.
func StageDesktopGrantWithTokensTx(
	tx *gorm.DB,
	publicId string,
	userId int,
	claudePolicy DesktopGrantTokenPolicy,
	codexPolicy DesktopGrantTokenPolicy,
	confirmationHash string,
	confirmationExpiresAt int64,
) (*DesktopGrantActivationResult, error) {
	now := time.Now().Unix()
	if tx == nil || publicId == "" || userId <= 0 || len(confirmationHash) != 64 ||
		confirmationExpiresAt <= now ||
		claudePolicy.Name == "" || claudePolicy.Key == "" || claudePolicy.Scopes == "" ||
		codexPolicy.Name == "" || codexPolicy.Key == "" || codexPolicy.Scopes == "" ||
		claudePolicy.ExpiredTime <= now || codexPolicy.ExpiredTime != claudePolicy.ExpiredTime ||
		codexPolicy.Scopes != claudePolicy.Scopes {
		return nil, ErrDesktopGrantInvalid
	}

	var pending DesktopGrant
	if err := lockForUpdate(tx).
		Where("public_id = ? AND user_id = ? AND status = ?", publicId, userId, DesktopGrantStatusPending).
		First(&pending).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDesktopGrantNotFound
		}
		return nil, err
	}

	tokens := []*Token{
		{
			UserId:             userId,
			Key:                claudePolicy.Key,
			Status:             common.TokenStatusDisabled,
			Name:               claudePolicy.Name,
			CreatedTime:        now,
			AccessedTime:       now,
			ExpiredTime:        claudePolicy.ExpiredTime,
			UnlimitedQuota:     claudePolicy.UnlimitedQuota,
			ModelLimitsEnabled: true,
			ModelLimits:        claudePolicy.ModelLimits,
			Group:              claudePolicy.Group,
		},
		{
			UserId:             userId,
			Key:                codexPolicy.Key,
			Status:             common.TokenStatusDisabled,
			Name:               codexPolicy.Name,
			CreatedTime:        now,
			AccessedTime:       now,
			ExpiredTime:        codexPolicy.ExpiredTime,
			UnlimitedQuota:     codexPolicy.UnlimitedQuota,
			ModelLimitsEnabled: true,
			ModelLimits:        codexPolicy.ModelLimits,
			Group:              codexPolicy.Group,
		},
	}
	if err := tx.Create(&tokens).Error; err != nil {
		return nil, err
	}
	claudeToken, codexToken := tokens[0], tokens[1]

	result := tx.Model(&DesktopGrant{}).
		Where("id = ? AND status = ? AND token_id IS NULL AND codex_token_id IS NULL", pending.Id, DesktopGrantStatusPending).
		Updates(map[string]interface{}{
			"token_id":                claudeToken.Id,
			"codex_token_id":          codexToken.Id,
			"scopes":                  claudePolicy.Scopes,
			"status":                  DesktopGrantStatusAwaitingConfirmation,
			"active_slot":             nil,
			"expired_time":            claudePolicy.ExpiredTime,
			"confirmation_hash":       confirmationHash,
			"confirmation_expires_at": confirmationExpiresAt,
			"confirmed_time":          0,
		})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected != 1 {
		return nil, ErrDesktopGrantInactive
	}

	pending.TokenId = &claudeToken.Id
	pending.CodexTokenId = &codexToken.Id
	pending.Scopes = claudePolicy.Scopes
	pending.Status = DesktopGrantStatusAwaitingConfirmation
	pending.ActiveSlot = nil
	pending.ExpiredTime = claudePolicy.ExpiredTime
	pending.ConfirmationHash = confirmationHash
	pending.ConfirmationExpiresAt = confirmationExpiresAt
	pending.ConfirmedTime = 0
	return &DesktopGrantActivationResult{
		Grant:       &pending,
		ClaudeToken: claudeToken,
		CodexToken:  codexToken,
	}, nil
}

func ConfirmDesktopGrant(
	confirmationHash string,
	maxActiveDevices int,
	giftPlanId int,
) (*DesktopGrantActivationResult, error) {
	if len(confirmationHash) != 64 || maxActiveDevices <= 0 || giftPlanId < 0 {
		return nil, ErrDesktopGrantConfirmationInvalid
	}
	now := time.Now().Unix()
	var activation *DesktopGrantActivationResult
	err := DB.Transaction(func(tx *gorm.DB) error {
		var grant DesktopGrant
		if err := lockForUpdate(tx).
			Where("confirmation_hash = ?", confirmationHash).
			First(&grant).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrDesktopGrantConfirmationInvalid
			}
			return err
		}
		claudeToken, codexToken, err := desktopGrantTokensWithTx(tx, &grant)
		if err != nil {
			return err
		}
		if grant.Status == DesktopGrantStatusActive && grant.ActiveSlot != nil {
			if _, err := EnsurePccAgentGiftSubscriptionWithTx(tx, grant.UserId, giftPlanId); err != nil {
				return err
			}
			activation = &DesktopGrantActivationResult{
				Grant:       &grant,
				ClaudeToken: claudeToken,
				CodexToken:  codexToken,
			}
			return nil
		}
		if grant.ConfirmationExpiresAt <= now {
			return ErrDesktopGrantConfirmationExpired
		}
		if grant.Status != DesktopGrantStatusAwaitingConfirmation || grant.ActiveSlot != nil {
			return ErrDesktopGrantConfirmationInvalid
		}
		if err := LockDesktopGrantUserWithTx(tx, grant.UserId); err != nil {
			return err
		}
		if _, err := EnsurePccAgentGiftSubscriptionWithTx(tx, grant.UserId, giftPlanId); err != nil {
			return err
		}

		var activeCount int64
		if err := tx.Model(&DesktopGrant{}).
			Where("user_id = ? AND status = ? AND active_slot = ? AND expired_time > ?",
				grant.UserId, DesktopGrantStatusActive, 1, now).
			Count(&activeCount).Error; err != nil {
			return err
		}
		var sameDeviceCount int64
		if err := tx.Model(&DesktopGrant{}).
			Where("user_id = ? AND client_id = ? AND device_id_hash = ? AND status = ? AND active_slot = ? AND expired_time > ?",
				grant.UserId, grant.ClientId, grant.DeviceIdHash, DesktopGrantStatusActive, 1, now).
			Count(&sameDeviceCount).Error; err != nil {
			return err
		}
		if activeCount >= int64(maxActiveDevices) && sameDeviceCount == 0 {
			return ErrDesktopGrantDeviceLimit
		}

		revokedGrants, revokedKeys, err := revokeActiveDesktopGrantsForDeviceWithTx(tx, &grant, "reauthorized")
		if err != nil {
			return err
		}
		for _, token := range []*Token{claudeToken, codexToken} {
			if token.Status != common.TokenStatusDisabled || token.ExpiredTime <= now {
				return ErrDesktopGrantInactive
			}
			if err := tx.Model(token).Update("status", common.TokenStatusEnabled).Error; err != nil {
				return err
			}
			token.Status = common.TokenStatusEnabled
		}

		activeSlot := 1
		result := tx.Model(&DesktopGrant{}).
			Where("id = ? AND status = ? AND active_slot IS NULL", grant.Id, DesktopGrantStatusAwaitingConfirmation).
			Updates(map[string]interface{}{
				"status":         DesktopGrantStatusActive,
				"active_slot":    activeSlot,
				"version":        gorm.Expr("version + 1"),
				"last_used_time": now,
				"confirmed_time": now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrDesktopGrantInactive
		}
		grant.Status = DesktopGrantStatusActive
		grant.ActiveSlot = &activeSlot
		grant.Version++
		grant.LastUsedTime = now
		grant.ConfirmedTime = now
		activation = &DesktopGrantActivationResult{
			Grant:            &grant,
			ClaudeToken:      claudeToken,
			CodexToken:       codexToken,
			RevokedTokenKeys: revokedKeys,
			RevokedGrants:    revokedGrants,
		}
		return nil
	})
	return activation, err
}

func desktopGrantTokensWithTx(tx *gorm.DB, grant *DesktopGrant) (*Token, *Token, error) {
	tokenIds := desktopGrantTokenIds(grant)
	if tx == nil || grant == nil || len(tokenIds) != 2 {
		return nil, nil, ErrDesktopGrantInvalid
	}
	var tokens []Token
	if err := lockForUpdate(tx).
		Where("user_id = ? AND id IN ?", grant.UserId, tokenIds).
		Order("id asc").
		Find(&tokens).Error; err != nil {
		return nil, nil, err
	}
	if len(tokens) != 2 || grant.TokenId == nil || grant.CodexTokenId == nil {
		return nil, nil, ErrDesktopGrantInactive
	}
	tokenByID := map[int]*Token{
		tokens[0].Id: &tokens[0],
		tokens[1].Id: &tokens[1],
	}
	claudeToken := tokenByID[*grant.TokenId]
	codexToken := tokenByID[*grant.CodexTokenId]
	if claudeToken == nil || codexToken == nil {
		return nil, nil, ErrDesktopGrantInactive
	}
	return claudeToken, codexToken, nil
}

func revokeActiveDesktopGrantsForDeviceWithTx(
	tx *gorm.DB,
	replacement *DesktopGrant,
	reason string,
) ([]DesktopGrant, []string, error) {
	if tx == nil || replacement == nil || replacement.UserId <= 0 ||
		replacement.ClientId == "" || replacement.DeviceIdHash == "" {
		return nil, nil, ErrDesktopGrantInvalid
	}
	var active []DesktopGrant
	if err := lockForUpdate(tx).
		Where("user_id = ? AND client_id = ? AND device_id_hash = ? AND status = ? AND active_slot = ? AND id <> ?",
			replacement.UserId, replacement.ClientId, replacement.DeviceIdHash,
			DesktopGrantStatusActive, 1, replacement.Id).
		Find(&active).Error; err != nil {
		return nil, nil, err
	}

	now := time.Now().Unix()
	revokedKeys := make([]string, 0, len(active)*2)
	revokedGrants := make([]DesktopGrant, 0, len(active))
	for i := range active {
		lastUsedTime := active[i].LastUsedTime
		for _, tokenId := range desktopGrantTokenIds(&active[i]) {
			var oldToken Token
			if err := lockForUpdate(tx).
				Where("id = ? AND user_id = ?", tokenId, replacement.UserId).
				First(&oldToken).Error; err != nil {
				if !errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, nil, err
				}
				continue
			}
			if oldToken.AccessedTime > lastUsedTime {
				lastUsedTime = oldToken.AccessedTime
			}
			if err := tx.Model(&oldToken).Updates(map[string]interface{}{
				"status":     common.TokenStatusDisabled,
				"deleted_at": time.Now(),
			}).Error; err != nil {
				return nil, nil, err
			}
			revokedKeys = append(revokedKeys, oldToken.Key)
		}
		if err := tx.Model(&DesktopGrant{}).Where("id = ?", active[i].Id).Updates(map[string]interface{}{
			"status":         DesktopGrantStatusRevoked,
			"active_slot":    nil,
			"version":        gorm.Expr("version + 1"),
			"revoked_time":   now,
			"revoke_reason":  reason,
			"last_used_time": lastUsedTime,
		}).Error; err != nil {
			return nil, nil, err
		}
		active[i].Status = DesktopGrantStatusRevoked
		active[i].ActiveSlot = nil
		active[i].Version++
		active[i].RevokedTime = now
		active[i].RevokeReason = reason
		active[i].LastUsedTime = lastUsedTime
		revokedGrants = append(revokedGrants, active[i])
	}
	return revokedGrants, revokedKeys, nil
}

func GetActiveDesktopGrantByTokenId(tokenId int) (*DesktopGrant, error) {
	if tokenId <= 0 {
		return nil, ErrDesktopGrantInvalid
	}
	if common.RedisEnabled {
		entry, err := getDesktopGrantCache(tokenId)
		if err == nil {
			return entry.grant(), nil
		}
		if errors.Is(err, ErrDesktopGrantInactive) {
			return nil, err
		}
	}
	cacheDeadline := desktopGrantCacheDeadline()
	now := time.Now().Unix()
	var grant DesktopGrant
	err := DB.Where("(token_id = ? OR codex_token_id = ?) AND status = ? AND active_slot = ? AND expired_time > ?",
		tokenId, tokenId, DesktopGrantStatusActive, 1, now).First(&grant).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrDesktopGrantInactive
	}
	if err != nil {
		return nil, err
	}
	if common.RedisEnabled {
		entry := desktopGrantCacheEntryFor(&grant, tokenId)
		if cacheErr := writeDesktopGrantCache(entry, cacheDeadline); cacheErr != nil {
			if errors.Is(cacheErr, errDesktopGrantCacheObservationStale) {
				if confirmErr := confirmDesktopGrantActiveSnapshot(&grant, tokenId); confirmErr != nil {
					return nil, confirmErr
				}
			} else if errors.Is(cacheErr, ErrDesktopGrantInactive) {
				return nil, cacheErr
			} else {
				common.SysLog("failed to populate desktop grant cache: " + cacheErr.Error())
			}
		}
	}
	return &grant, err
}

func GetPendingDesktopGrant(publicId string, userId int) (*DesktopGrant, error) {
	if publicId == "" || userId <= 0 {
		return nil, ErrDesktopGrantInvalid
	}
	var grant DesktopGrant
	err := DB.Where("public_id = ? AND user_id = ? AND status = ?", publicId, userId, DesktopGrantStatusPending).
		First(&grant).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrDesktopGrantNotFound
	}
	return &grant, err
}

func ListDesktopGrants(userId int) ([]DesktopGrant, error) {
	if userId <= 0 {
		return nil, ErrDesktopGrantInvalid
	}
	var grants []DesktopGrant
	err := DB.Where("user_id = ? AND status NOT IN ?", userId, []string{
		DesktopGrantStatusPending,
		DesktopGrantStatusAwaitingConfirmation,
	}).
		Order("created_time desc, id desc").
		Find(&grants).Error
	if grants == nil {
		grants = []DesktopGrant{}
	}
	tokenIds := make([]int, 0, len(grants)*2)
	for i := range grants {
		tokenIds = append(tokenIds, desktopGrantTokenIds(&grants[i])...)
	}
	tokenLastUsed := make(map[int]int64, len(tokenIds))
	if len(tokenIds) > 0 {
		var tokens []Token
		if err := DB.Select("id", "accessed_time").
			Where("user_id = ? AND id IN ?", userId, tokenIds).
			Find(&tokens).Error; err != nil {
			return nil, err
		}
		for i := range tokens {
			tokenLastUsed[tokens[i].Id] = tokens[i].AccessedTime
		}
	}
	now := time.Now().Unix()
	for i := range grants {
		for _, tokenId := range desktopGrantTokenIds(&grants[i]) {
			if tokenLastUsed[tokenId] > grants[i].LastUsedTime {
				grants[i].LastUsedTime = tokenLastUsed[tokenId]
			}
		}
		if grants[i].Status == DesktopGrantStatusActive && grants[i].ExpiredTime <= now {
			grants[i].Status = DesktopGrantStatusExpired
		}
	}
	return grants, err
}

func DeleteExpiredPendingDesktopGrants(now time.Time) error {
	cutoff := now.Add(-AuthFlowDefaultCleanupRetention).Unix()
	return DB.Where("status = ? AND created_time < ?", DesktopGrantStatusPending, cutoff).
		Delete(&DesktopGrant{}).Error
}

func RevokeExpiredUnconfirmedDesktopGrants(now time.Time) ([]string, error) {
	if now.IsZero() {
		now = time.Now()
	}
	tokenKeys := make([]string, 0)
	var revokedGrants []DesktopGrant
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := lockForUpdate(tx).
			Where("status = ? AND confirmation_expires_at > 0 AND confirmation_expires_at <= ?",
				DesktopGrantStatusAwaitingConfirmation, now.Unix()).
			Find(&revokedGrants).Error; err != nil {
			return err
		}
		for i := range revokedGrants {
			if err := revokeDesktopGrantWithTx(
				tx,
				&revokedGrants[i],
				"confirmation_expired",
				&tokenKeys,
			); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for i := range revokedGrants {
		if err := publishDesktopGrantState(&revokedGrants[i]); err != nil {
			return tokenKeys, err
		}
	}
	return tokenKeys, nil
}

// RevokeDesktopGrant revokes a grant and its token atomically. It is
// idempotent for already-revoked grants.
func RevokeDesktopGrant(userId int, publicId, reason string) ([]string, error) {
	if userId <= 0 || publicId == "" {
		return nil, ErrDesktopGrantInvalid
	}
	var tokenKeys []string
	var revokedGrant *DesktopGrant
	err := DB.Transaction(func(tx *gorm.DB) error {
		var grant DesktopGrant
		if err := lockForUpdate(tx).Where("user_id = ? AND public_id = ?", userId, publicId).First(&grant).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrDesktopGrantNotFound
			}
			return err
		}
		if grant.Status != DesktopGrantStatusActive &&
			grant.Status != DesktopGrantStatusAwaitingConfirmation {
			return nil
		}
		if err := revokeDesktopGrantWithTx(tx, &grant, reason, &tokenKeys); err != nil {
			return err
		}
		revokedGrant = &grant
		return nil
	})
	if err == nil && revokedGrant != nil {
		err = publishDesktopGrantState(revokedGrant)
	}
	return tokenKeys, err
}

func DeleteRevokedDesktopGrant(userId int, publicId string) ([]string, error) {
	if userId <= 0 || publicId == "" {
		return nil, ErrDesktopGrantInvalid
	}
	var tokenKeys []string
	err := DB.Transaction(func(tx *gorm.DB) error {
		var grant DesktopGrant
		if err := lockForUpdate(tx).
			Where("user_id = ? AND public_id = ?", userId, publicId).
			First(&grant).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrDesktopGrantNotFound
			}
			return err
		}
		if grant.Status != DesktopGrantStatusRevoked {
			return ErrDesktopGrantNotRevoked
		}

		tokenIds := desktopGrantTokenIds(&grant)
		if len(tokenIds) > 0 {
			var tokens []Token
			if err := tx.Select("id", commonKeyCol).
				Where("user_id = ? AND id IN ?", userId, tokenIds).
				Find(&tokens).Error; err != nil {
				return err
			}
			if len(tokens) > 0 {
				if err := tx.Where("user_id = ? AND id IN ?", userId, tokenIds).
					Delete(&Token{}).Error; err != nil {
					return err
				}
				tokenKeys = make([]string, 0, len(tokens))
				for i := range tokens {
					tokenKeys = append(tokenKeys, tokens[i].Key)
				}
			}
		}
		return tx.Delete(&grant).Error
	})
	return tokenKeys, err
}

func RevokeDesktopGrantByTokenId(tokenId int, reason string) (int, []string, error) {
	if tokenId <= 0 {
		return 0, nil, ErrDesktopGrantInvalid
	}
	var userId int
	var tokenKeys []string
	var revokedGrant *DesktopGrant
	err := DB.Transaction(func(tx *gorm.DB) error {
		var grant DesktopGrant
		if err := lockForUpdate(tx).Where("token_id = ? OR codex_token_id = ?", tokenId, tokenId).First(&grant).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrDesktopGrantNotFound
			}
			return err
		}
		userId = grant.UserId
		if grant.Status != DesktopGrantStatusActive &&
			grant.Status != DesktopGrantStatusAwaitingConfirmation {
			return nil
		}
		if err := revokeDesktopGrantWithTx(tx, &grant, reason, &tokenKeys); err != nil {
			return err
		}
		revokedGrant = &grant
		return nil
	})
	if err == nil && revokedGrant != nil {
		err = publishDesktopGrantState(revokedGrant)
	}
	return userId, tokenKeys, err
}

func revokeDesktopGrantWithTx(tx *gorm.DB, grant *DesktopGrant, reason string, tokenKeys *[]string) error {
	if tx == nil || grant == nil {
		return ErrDesktopGrantInvalid
	}
	now := time.Now().Unix()
	lastUsedTime := grant.LastUsedTime
	for _, tokenId := range desktopGrantTokenIds(grant) {
		var token Token
		if err := lockForUpdate(tx).Where("id = ? AND user_id = ?", tokenId, grant.UserId).First(&token).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		} else {
			if token.AccessedTime > lastUsedTime {
				lastUsedTime = token.AccessedTime
			}
			if err := tx.Model(&token).Update("status", common.TokenStatusDisabled).Error; err != nil {
				return err
			}
			*tokenKeys = append(*tokenKeys, token.Key)
		}
	}
	if err := tx.Model(&DesktopGrant{}).Where("id = ?", grant.Id).Updates(map[string]interface{}{
		"status":         DesktopGrantStatusRevoked,
		"active_slot":    nil,
		"version":        gorm.Expr("version + 1"),
		"revoked_time":   now,
		"revoke_reason":  reason,
		"last_used_time": lastUsedTime,
	}).Error; err != nil {
		return err
	}
	grant.Status = DesktopGrantStatusRevoked
	grant.ActiveSlot = nil
	grant.Version++
	grant.RevokedTime = now
	grant.RevokeReason = reason
	grant.LastUsedTime = lastUsedTime
	return nil
}

func revokeAllDesktopGrantsWithTx(tx *gorm.DB, userId int, reason string) ([]DesktopGrant, error) {
	if tx == nil || userId <= 0 {
		return nil, ErrDesktopGrantInvalid
	}
	var grants []DesktopGrant
	if err := lockForUpdate(tx).
		Where("user_id = ? AND status IN ?", userId, []string{
			DesktopGrantStatusActive,
			DesktopGrantStatusAwaitingConfirmation,
		}).
		Find(&grants).Error; err != nil {
		return nil, err
	}
	for i := range grants {
		tokenKeys := make([]string, 0, 2)
		if err := revokeDesktopGrantWithTx(tx, &grants[i], reason, &tokenKeys); err != nil {
			return nil, err
		}
	}
	return grants, nil
}

func RevokeAllDesktopGrants(userId int, reason string) ([]string, error) {
	if userId <= 0 {
		return nil, ErrDesktopGrantInvalid
	}
	tokenKeys := make([]string, 0)
	var revokedGrants []DesktopGrant
	err := DB.Transaction(func(tx *gorm.DB) error {
		var grants []DesktopGrant
		if err := lockForUpdate(tx).
			Where("user_id = ? AND status IN ?", userId, []string{
				DesktopGrantStatusActive,
				DesktopGrantStatusAwaitingConfirmation,
			}).
			Find(&grants).Error; err != nil {
			return err
		}
		for i := range grants {
			if err := revokeDesktopGrantWithTx(tx, &grants[i], reason, &tokenKeys); err != nil {
				return err
			}
		}
		revokedGrants = grants
		return nil
	})
	if err == nil {
		for i := range revokedGrants {
			if publishErr := publishDesktopGrantState(&revokedGrants[i]); publishErr != nil {
				return tokenKeys, publishErr
			}
		}
	}
	return tokenKeys, err
}

func IsDesktopGrantToken(userId, tokenId int) (bool, error) {
	if userId <= 0 || tokenId <= 0 {
		return false, nil
	}
	return HasDesktopGrantToken(userId, []int{tokenId})
}

func HasDesktopGrantToken(userId int, tokenIds []int) (bool, error) {
	if userId <= 0 || len(tokenIds) == 0 {
		return false, nil
	}
	var count int64
	err := DB.Model(&DesktopGrant{}).
		Where("user_id = ? AND (token_id IN ? OR codex_token_id IN ?)", userId, tokenIds, tokenIds).
		Count(&count).Error
	return count > 0, err
}

func HasNonRevokedDesktopGrantToken(userId int, tokenIds []int) (bool, error) {
	if userId <= 0 || len(tokenIds) == 0 {
		return false, nil
	}
	var count int64
	err := DB.Model(&DesktopGrant{}).
		Where("user_id = ? AND (status IS NULL OR status <> ?) AND (token_id IN ? OR codex_token_id IN ?)",
			userId, DesktopGrantStatusRevoked, tokenIds, tokenIds).
		Count(&count).Error
	return count > 0, err
}

func desktopGrantTokenIds(grant *DesktopGrant) []int {
	if grant == nil {
		return nil
	}
	tokenIds := make([]int, 0, 2)
	if grant.TokenId != nil {
		tokenIds = append(tokenIds, *grant.TokenId)
	}
	if grant.CodexTokenId != nil {
		tokenIds = append(tokenIds, *grant.CodexTokenId)
	}
	return tokenIds
}
