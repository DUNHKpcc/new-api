package model

import (
	"errors"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	DesktopGrantStatusPending = "pending"
	DesktopGrantStatusActive  = "active"
	DesktopGrantStatusRevoked = "revoked"
	DesktopGrantStatusExpired = "expired"
)

var (
	ErrDesktopGrantInvalid  = errors.New("desktop grant is invalid")
	ErrDesktopGrantNotFound = errors.New("desktop grant was not found")
	ErrDesktopGrantInactive = errors.New("desktop grant is inactive")
)

// DesktopGrant records a user's explicit authorization of a public desktop
// client. DeviceIDHash is an HMAC; the raw installation identifier is never
// persisted.
type DesktopGrant struct {
	Id           int64  `json:"-" gorm:"primaryKey"`
	PublicId     string `json:"public_id" gorm:"type:varchar(64);not null;uniqueIndex"`
	UserId       int    `json:"-" gorm:"not null;index;uniqueIndex:idx_desktop_grants_active_device,priority:1"`
	ClientId     string `json:"client_id" gorm:"type:varchar(64);not null;uniqueIndex:idx_desktop_grants_active_device,priority:2"`
	DeviceIdHash string `json:"-" gorm:"type:char(64);not null;index;uniqueIndex:idx_desktop_grants_active_device,priority:3"`
	DeviceName   string `json:"device_name" gorm:"type:varchar(128);not null"`
	Platform     string `json:"platform" gorm:"type:varchar(64)"`
	AppVersion   string `json:"app_version" gorm:"type:varchar(32)"`
	TokenId      *int   `json:"-" gorm:"uniqueIndex"`
	CodexTokenId *int   `json:"-" gorm:"uniqueIndex"`
	Scopes       string `json:"scopes" gorm:"type:varchar(255);not null"`
	Status       string `json:"status" gorm:"type:varchar(16);not null;index"`
	ActiveSlot   *int   `json:"-" gorm:"uniqueIndex:idx_desktop_grants_active_device,priority:4"`
	CreatedTime  int64  `json:"created_time" gorm:"type:bigint;not null;index"`
	LastUsedTime int64  `json:"last_used_time" gorm:"type:bigint;not null;default:0"`
	ExpiredTime  int64  `json:"expired_time" gorm:"type:bigint;not null;default:0;index"`
	RevokedTime  int64  `json:"revoked_time" gorm:"type:bigint;not null;default:0"`
	RevokeReason string `json:"revoke_reason,omitempty" gorm:"type:varchar(64)"`
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
	grant.Status = DesktopGrantStatusPending
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

	var active []DesktopGrant
	if err := lockForUpdate(tx).
		Where("user_id = ? AND client_id = ? AND device_id_hash = ? AND status = ? AND active_slot = ?",
			pending.UserId, pending.ClientId, pending.DeviceIdHash, DesktopGrantStatusActive, 1).
		Find(&active).Error; err != nil {
		return nil, err
	}

	revokedKeys := make([]string, 0, len(active)*2)
	for i := range active {
		lastUsedTime := active[i].LastUsedTime
		for _, tokenId := range desktopGrantTokenIds(&active[i]) {
			var oldToken Token
			if err := lockForUpdate(tx).Where("id = ? AND user_id = ?", tokenId, userId).First(&oldToken).Error; err != nil {
				if !errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, err
				}
			} else {
				if oldToken.AccessedTime > lastUsedTime {
					lastUsedTime = oldToken.AccessedTime
				}
				if err := tx.Model(&oldToken).Updates(map[string]interface{}{
					"status":     common.TokenStatusDisabled,
					"deleted_at": time.Now(),
				}).Error; err != nil {
					return nil, err
				}
				revokedKeys = append(revokedKeys, oldToken.Key)
			}
		}
		if err := tx.Model(&DesktopGrant{}).Where("id = ?", active[i].Id).Updates(map[string]interface{}{
			"status":         DesktopGrantStatusRevoked,
			"active_slot":    nil,
			"revoked_time":   now,
			"revoke_reason":  "reauthorized",
			"last_used_time": lastUsedTime,
		}).Error; err != nil {
			return nil, err
		}
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
			"token_id":       claudeToken.Id,
			"codex_token_id": codexToken.Id,
			"scopes":         claudePolicy.Scopes,
			"status":         DesktopGrantStatusActive,
			"active_slot":    activeSlot,
			"last_used_time": now,
			"expired_time":   claudePolicy.ExpiredTime,
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
	return &DesktopGrantActivationResult{
		Grant:            &pending,
		ClaudeToken:      claudeToken,
		CodexToken:       codexToken,
		RevokedTokenKeys: revokedKeys,
	}, nil
}

func GetActiveDesktopGrantByTokenId(tokenId int) (*DesktopGrant, error) {
	if tokenId <= 0 {
		return nil, ErrDesktopGrantInvalid
	}
	now := time.Now().Unix()
	var grant DesktopGrant
	err := DB.Where("(token_id = ? OR codex_token_id = ?) AND status = ? AND active_slot = ? AND expired_time > ?",
		tokenId, tokenId, DesktopGrantStatusActive, 1, now).First(&grant).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrDesktopGrantInactive
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
	err := DB.Where("user_id = ? AND status <> ?", userId, DesktopGrantStatusPending).
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

// RevokeDesktopGrant revokes a grant and its token atomically. It is
// idempotent for already-revoked grants.
func RevokeDesktopGrant(userId int, publicId, reason string) ([]string, error) {
	if userId <= 0 || publicId == "" {
		return nil, ErrDesktopGrantInvalid
	}
	var tokenKeys []string
	err := DB.Transaction(func(tx *gorm.DB) error {
		var grant DesktopGrant
		if err := lockForUpdate(tx).Where("user_id = ? AND public_id = ?", userId, publicId).First(&grant).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrDesktopGrantNotFound
			}
			return err
		}
		if grant.Status != DesktopGrantStatusActive || grant.ActiveSlot == nil {
			return nil
		}
		return revokeDesktopGrantWithTx(tx, &grant, reason, &tokenKeys)
	})
	return tokenKeys, err
}

func RevokeDesktopGrantByTokenId(tokenId int, reason string) (int, []string, error) {
	if tokenId <= 0 {
		return 0, nil, ErrDesktopGrantInvalid
	}
	var userId int
	var tokenKeys []string
	err := DB.Transaction(func(tx *gorm.DB) error {
		var grant DesktopGrant
		if err := lockForUpdate(tx).Where("token_id = ? OR codex_token_id = ?", tokenId, tokenId).First(&grant).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrDesktopGrantNotFound
			}
			return err
		}
		userId = grant.UserId
		if grant.Status != DesktopGrantStatusActive || grant.ActiveSlot == nil {
			return nil
		}
		return revokeDesktopGrantWithTx(tx, &grant, reason, &tokenKeys)
	})
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
	return tx.Model(&DesktopGrant{}).Where("id = ?", grant.Id).Updates(map[string]interface{}{
		"status":         DesktopGrantStatusRevoked,
		"active_slot":    nil,
		"revoked_time":   now,
		"revoke_reason":  reason,
		"last_used_time": lastUsedTime,
	}).Error
}

func revokeAllDesktopGrantsWithTx(tx *gorm.DB, userId int, reason string) error {
	if tx == nil || userId <= 0 {
		return ErrDesktopGrantInvalid
	}
	var grants []DesktopGrant
	if err := lockForUpdate(tx).
		Where("user_id = ? AND status = ? AND active_slot = ?", userId, DesktopGrantStatusActive, 1).
		Find(&grants).Error; err != nil {
		return err
	}
	for i := range grants {
		tokenKeys := make([]string, 0, 2)
		if err := revokeDesktopGrantWithTx(tx, &grants[i], reason, &tokenKeys); err != nil {
			return err
		}
	}
	return nil
}

func RevokeAllDesktopGrants(userId int, reason string) ([]string, error) {
	if userId <= 0 {
		return nil, ErrDesktopGrantInvalid
	}
	tokenKeys := make([]string, 0)
	err := DB.Transaction(func(tx *gorm.DB) error {
		var grants []DesktopGrant
		if err := lockForUpdate(tx).
			Where("user_id = ? AND status = ? AND active_slot = ?", userId, DesktopGrantStatusActive, 1).
			Find(&grants).Error; err != nil {
			return err
		}
		for i := range grants {
			if err := revokeDesktopGrantWithTx(tx, &grants[i], reason, &tokenKeys); err != nil {
				return err
			}
		}
		return nil
	})
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
