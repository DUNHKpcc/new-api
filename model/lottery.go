package model

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// Lottery rules become immutable when published, so entrants see the rules used at draw time.
type Lottery struct {
	ID              int    `json:"id"`
	Title           string `json:"title" gorm:"type:varchar(160)"`
	Content         string `json:"content" gorm:"type:text"`
	Prize           string `json:"prize" gorm:"type:varchar(500)"`
	WinnerCount     int    `json:"winner_count"`
	DrawMode        string `json:"draw_mode" gorm:"type:varchar(16)"`
	TargetCount     int    `json:"target_count"`
	DrawAt          int64  `json:"draw_at" gorm:"index:idx_lottery_due,priority:2"`
	RequireRecharge bool   `json:"require_recharge"`
	MinInvites      int    `json:"min_invites"`
	RequiredGroup   string `json:"required_group" gorm:"type:varchar(64)"`
	EmailNotify     bool   `json:"email_notify"`
	Status          string `json:"status" gorm:"type:varchar(16);index:idx_lottery_due,priority:1"`
	EntryCount      int    `json:"entry_count"`
	CreatedAt       int64  `json:"created_at"`
	DrawnAt         int64  `json:"drawn_at"`
}

type LotteryEntry struct {
	ID            int    `json:"id"`
	LotteryID     int    `json:"lottery_id" gorm:"uniqueIndex:idx_lottery_user,priority:1;index:idx_lottery_winners,priority:1"`
	UserID        int    `json:"user_id" gorm:"uniqueIndex:idx_lottery_user,priority:2"`
	CreatedAt     int64  `json:"created_at"`
	Winner        bool   `json:"winner" gorm:"index:idx_lottery_winners,priority:2"`
	EmailStatus   string `json:"email_status" gorm:"type:varchar(16);index:idx_lottery_mail,priority:1"`
	EmailAttempts int    `json:"email_attempts"`
	EmailNextAt   int64  `json:"-" gorm:"index:idx_lottery_mail,priority:2"`
	Fulfilled     bool   `json:"fulfilled"`
}

func (l *Lottery) Validate(now int64) error {
	l.Title, l.Content, l.Prize = strings.TrimSpace(l.Title), strings.TrimSpace(l.Content), strings.TrimSpace(l.Prize)
	l.RequiredGroup = strings.TrimSpace(l.RequiredGroup)
	if l.Title == "" || utf8.RuneCountInString(l.Title) > 80 || l.Content == "" || len(l.Content) > 6000 || l.Prize == "" || utf8.RuneCountInString(l.Prize) > 250 || l.WinnerCount < 1 || l.WinnerCount > 100 || l.MinInvites < 0 || l.MinInvites > 100000 || len(l.RequiredGroup) > 64 {
		return errors.New("invalid lottery rules")
	}
	if l.DrawMode != "full" && l.DrawMode != "time" && l.DrawMode != "either" {
		return errors.New("invalid draw mode")
	}
	if l.DrawMode != "time" && (l.TargetCount < l.WinnerCount || l.TargetCount > 100000) {
		return errors.New("participant target must cover all winners and not exceed 100000")
	}
	if l.DrawMode != "full" && l.DrawAt <= now {
		return errors.New("draw time must be in the future")
	}
	if l.DrawMode == "full" {
		l.DrawAt = 0
	}
	if l.DrawMode == "time" {
		l.TargetCount = 0
	}
	return nil
}

func SaveLottery(l *Lottery) error {
	if err := l.Validate(time.Now().Unix()); err != nil {
		return err
	}
	if l.ID == 0 {
		l.Status = "draft"
		l.EntryCount = 0
		l.DrawnAt = 0
		l.CreatedAt = time.Now().Unix()
		return DB.Create(l).Error
	}
	// Select only editable fields; never trust submitted counters or state.
	r := DB.Model(&Lottery{}).Where("id = ? AND status = ?", l.ID, "draft").Select("title", "content", "prize", "winner_count", "draw_mode", "target_count", "draw_at", "require_recharge", "min_invites", "required_group", "email_notify").Updates(l)
	if r.Error != nil {
		return r.Error
	}
	if r.RowsAffected == 0 {
		var n int64
		if err := DB.Model(&Lottery{}).Where("id = ? AND status = ?", l.ID, "draft").Count(&n).Error; err != nil {
			return err
		}
		if n == 0 {
			return errors.New("only drafts can be edited")
		}
	}
	return nil
}

func TransitionLottery(id int, action string) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		// Acquire the write lock before reading, including on SQLite.
		r := tx.Model(&Lottery{}).Where("id = ?", id).UpdateColumn("entry_count", gorm.Expr("entry_count"))
		if r.Error != nil {
			return r.Error
		}

		var l Lottery
		if err := tx.First(&l, id).Error; err != nil {
			return err
		}
		switch action {
		case "publish":
			if l.Status != "draft" {
				return errors.New("only drafts can be published")
			}
			if err := l.Validate(time.Now().Unix()); err != nil {
				return err
			}
			l.Status = "active"
		case "cancel":
			if l.Status != "draft" && l.Status != "active" {
				return errors.New("lottery already closed")
			}
			l.Status = "cancelled"
		default:
			return errors.New("invalid lottery action")
		}
		return tx.Model(&l).Update("status", l.Status).Error
	})
}

func JoinLottery(id, userID int) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		r := tx.Model(&Lottery{}).Where("id = ?", id).UpdateColumn("entry_count", gorm.Expr("entry_count"))
		if r.Error != nil {
			return r.Error
		}

		var l Lottery
		if err := tx.First(&l, id).Error; err != nil {
			return err
		}
		if l.Status != "active" {
			return errors.New("lottery is not open")
		}
		now := time.Now().Unix()
		if (l.DrawAt > 0 && now >= l.DrawAt) || (l.TargetCount > 0 && l.EntryCount >= l.TargetCount) || l.EntryCount >= 100000 {
			return errors.New("lottery entry is closed")
		}
		var existing int64
		if err := tx.Model(&LotteryEntry{}).Where("lottery_id = ? AND user_id = ?", id, userID).Count(&existing).Error; err != nil {
			return err
		}
		if existing > 0 {
			return errors.New("already entered this lottery")
		}
		var u User
		if err := tx.Select("id", "status", "email", "group").First(&u, userID).Error; err != nil {
			return err
		}
		if u.Status != common.UserStatusEnabled {
			return errors.New("account is disabled")
		}
		if l.RequiredGroup != "" && u.Group != l.RequiredGroup {
			return errors.New("required user group not met")
		}
		if l.RequireRecharge {
			var n int64
			if err := tx.Model(&TopUp{}).Where("user_id = ? AND status = ?", userID, common.TopUpStatusSuccess).Count(&n).Error; err != nil {
				return err
			}
			if n == 0 {
				return errors.New("a completed recharge is required")
			}
		}
		if l.MinInvites > 0 {
			var n int64
			if err := tx.Model(&User{}).Where("inviter_id = ? AND status = ?", userID, common.UserStatusEnabled).Count(&n).Error; err != nil {
				return err
			}
			if n < int64(l.MinInvites) {
				return errors.New("minimum invited users not met")
			}
		}
		if l.EmailNotify {
			if _, err := NormalizeCampaignEmail(u.Email); err != nil {
				return errors.New("bind an email address before entering")
			}
		}
		entry := LotteryEntry{LotteryID: id, UserID: userID, CreatedAt: now, EmailStatus: "none"}
		if err := tx.Create(&entry).Error; err != nil {
			return err
		}
		return tx.Model(&l).UpdateColumn("entry_count", gorm.Expr("entry_count + 1")).Error
	})
}

// DrawLottery atomically closes entries and records unique winners. Repeated workers are no-ops.
func DrawLottery(id int, now int64) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		r := tx.Model(&Lottery{}).Where("id = ?", id).UpdateColumn("entry_count", gorm.Expr("entry_count"))
		if r.Error != nil {
			return r.Error
		}

		var l Lottery
		if err := tx.First(&l, id).Error; err != nil {
			return err
		}
		if l.Status != "active" {
			return nil
		}
		if !(l.DrawAt > 0 && l.DrawAt <= now) && !(l.TargetCount > 0 && l.EntryCount >= l.TargetCount) {
			return nil
		}
		var ids []int
		if err := tx.Model(&LotteryEntry{}).Where("lottery_id = ?", id).Order("id").Pluck("id", &ids).Error; err != nil {
			return err
		}
		// Partial Fisher-Yates uses crypto/rand; every participant has equal probability.
		for i := range min(l.WinnerCount, len(ids)) {
			n, err := rand.Int(rand.Reader, big.NewInt(int64(len(ids)-i)))
			if err != nil {
				return err
			}
			j := i + int(n.Int64())
			ids[i], ids[j] = ids[j], ids[i]
			state := "none"
			if l.EmailNotify {
				state = "pending"
			}
			if err := tx.Model(&LotteryEntry{}).Where("id = ?", ids[i]).Updates(map[string]any{"winner": true, "email_status": state}).Error; err != nil {
				return err
			}
		}
		return tx.Model(&l).Updates(map[string]any{"status": "drawn", "drawn_at": now}).Error
	})
}

func ProcessDueLotteries(now int64) error {
	var ids []int
	if err := DB.Model(&Lottery{}).Where("status = ? AND ((draw_at > 0 AND draw_at <= ?) OR (target_count > 0 AND entry_count >= target_count))", "active", now).Order("id").Limit(50).Pluck("id", &ids).Error; err != nil {
		return err
	}
	for _, id := range ids {
		if err := DrawLottery(id, now); err != nil {
			return fmt.Errorf("lottery %d draw failed: %w", id, err)
		}
	}
	return nil
}
