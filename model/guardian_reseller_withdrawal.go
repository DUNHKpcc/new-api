package model

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"
)

type GuardianResellerWithdrawal struct {
	ID          int     `json:"id"`
	ResellerID  int     `json:"reseller_id" gorm:"uniqueIndex:ux_guardian_withdrawal_key,priority:1"`
	RequestKey  string  `json:"request_key" gorm:"type:varchar(64);uniqueIndex:ux_guardian_withdrawal_key,priority:2"`
	AmountCents int64   `json:"amount_cents"`
	FeeCents    int64   `json:"fee_cents"`
	Account     string  `json:"account" gorm:"type:varchar(128)"`
	PayeeName   string  `json:"payee_name" gorm:"type:varchar(128)"`
	Status      string  `json:"status" gorm:"type:varchar(16);index"`
	Reference   *string `json:"reference" gorm:"type:varchar(128);uniqueIndex"`
	Note        string  `json:"note" gorm:"type:varchar(500)"`
	UpdatedBy   int     `json:"updated_by"`
	CreatedAt   int64   `json:"created_at"`
	UpdatedAt   int64   `json:"updated_at"`
}

type GuardianWithdrawalRequest struct {
	AmountCents int64  `json:"amount_cents"`
	Account     string `json:"account"`
	PayeeName   string `json:"payee_name"`
	RequestKey  string `json:"request_key"`
}

func (r *GuardianWithdrawalRequest) Validate() error {
	r.Account, r.PayeeName, r.RequestKey = strings.TrimSpace(r.Account), strings.TrimSpace(r.PayeeName), strings.TrimSpace(r.RequestKey)
	if r.AmountCents < 100 || r.AmountCents > 100000000 || r.Account == "" || r.PayeeName == "" || utf8.RuneCountInString(r.Account) > 128 || utf8.RuneCountInString(r.PayeeName) > 128 || len(r.RequestKey) < 16 || len(r.RequestKey) > 64 {
		return errors.New("提现至少 1 元，请填写有效的支付宝收款账号、姓名和申请标识")
	}
	return nil
}

// The controller validates and binds this exact request to a one-use security proof.
func CreateGuardianWithdrawal(resellerID int, req GuardianWithdrawalRequest) (*GuardianResellerWithdrawal, error) {
	var withdrawal GuardianResellerWithdrawal
	err := DB.Transaction(func(tx *gorm.DB) error {
		var reseller GuardianReseller
		if err := lockForUpdate(tx).First(&reseller, "user_id = ?", resellerID).Error; err != nil {
			return errors.New("尚未开通 Guardian 代理")
		}
		err := tx.First(&withdrawal, "reseller_id = ? AND request_key = ?", resellerID, req.RequestKey).Error
		if err == nil {
			if withdrawal.AmountCents != req.AmountCents || withdrawal.Account != req.Account || withdrawal.PayeeName != req.PayeeName {
				return errors.New("申请标识已用于另一笔提现")
			}
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		balance, err := guardianResellerBalance(tx, resellerID, time.Now().Unix())
		if err != nil {
			return err
		}
		if req.AmountCents > balance.AvailableCents {
			return errors.New("可提现金额不足，请检查手续费核对、交付等待期及退款冲减")
		}
		now := time.Now().Unix()
		withdrawal = GuardianResellerWithdrawal{ResellerID: resellerID, RequestKey: req.RequestKey, AmountCents: req.AmountCents, Account: req.Account, PayeeName: req.PayeeName, Status: "pending", CreatedAt: now, UpdatedAt: now}
		if err := tx.Create(&withdrawal).Error; err != nil {
			return err
		}
		return recordGuardianResellerOperation(tx, resellerID, resellerID, "withdrawal_requested", map[string]any{"withdrawal_id": withdrawal.ID, "amount_cents": withdrawal.AmountCents})
	})
	return &withdrawal, err
}

func ListGuardianWithdrawals(resellerID, page, size int) ([]GuardianResellerWithdrawal, int64, error) {
	query := DB.Model(&GuardianResellerWithdrawal{})
	if resellerID > 0 {
		query = query.Where("reseller_id = ?", resellerID)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	items := []GuardianResellerWithdrawal{}
	err := query.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error
	return items, total, err
}

type GuardianWithdrawalUpdate struct {
	Status    string `json:"status"`
	FeeCents  int64  `json:"fee_cents"`
	Reference string `json:"reference"`
	Note      string `json:"note"`
}

func UpdateGuardianWithdrawal(id, actorID int, admin bool, req GuardianWithdrawalUpdate) error {
	req.Reference, req.Note = strings.TrimSpace(req.Reference), strings.TrimSpace(req.Note)
	if req.FeeCents < 0 || req.Note == "" || utf8.RuneCountInString(req.Note) > 500 || utf8.RuneCountInString(req.Reference) > 128 {
		return errors.New("请填写有效的提现操作说明及手续费")
	}
	var existing GuardianResellerWithdrawal
	query := DB.Where("id = ?", id)
	if !admin {
		query = query.Where("reseller_id = ?", actorID)
	}
	if err := query.First(&existing).Error; err != nil {
		return errors.New("提现申请不存在")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var reseller GuardianReseller
		if err := lockForUpdate(tx).First(&reseller, "user_id = ?", existing.ResellerID).Error; err != nil {
			return err
		}
		var w GuardianResellerWithdrawal
		if err := lockForUpdate(tx).First(&w, id).Error; err != nil {
			return err
		}
		if !admin {
			if w.Status != "pending" || req.Status != "cancelled" {
				return errors.New("只能撤销尚未进入打款处理的提现申请")
			}
		} else {
			allowed := (w.Status == "pending" && (req.Status == "processing" || req.Status == "rejected")) || (w.Status == "processing" && (req.Status == "paid" || req.Status == "failed"))
			if !allowed {
				return errors.New("不允许的提现状态变更")
			}
			if req.Status == "processing" {
				if req.FeeCents >= w.AmountCents {
					return errors.New("打款手续费必须小于申请金额")
				}
				w.FeeCents = req.FeeCents
			}
			if req.Status == "paid" || req.Status == "failed" {
				if req.Reference == "" {
					return errors.New("请核实外部打款结果并填写交易或失败核查凭证；结果不明请保持处理中")
				}
				w.Reference = &req.Reference
			}
		}
		w.Status, w.Note, w.UpdatedBy, w.UpdatedAt = req.Status, req.Note, actorID, time.Now().Unix()
		if err := tx.Model(&w).Select("status", "fee_cents", "reference", "note", "updated_by", "updated_at").Updates(&w).Error; err != nil {
			return err
		}
		return recordGuardianResellerOperation(tx, actorID, w.ResellerID, "withdrawal_"+req.Status, map[string]any{"withdrawal_id": w.ID, "fee_cents": w.FeeCents, "reference": w.Reference, "note": w.Note})
	})
}
