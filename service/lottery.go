package service

import (
	"context"
	"errors"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
)

type lotteryDrawHandler struct{}

func (lotteryDrawHandler) Type() string { return "lottery_draw" }
func (lotteryDrawHandler) Enabled() bool {
	var n int64
	return model.DB.Model(&model.Lottery{}).Where("status = ?", "active").Limit(1).Count(&n).Error == nil && n > 0
}
func (lotteryDrawHandler) Interval() time.Duration { return 30 * time.Second }
func (lotteryDrawHandler) NewPayload() any         { return nil }
func (lotteryDrawHandler) Run(ctx context.Context, task *model.SystemTask, runnerID string) {
	err := model.ProcessDueLotteries(time.Now().Unix())
	if err != nil {
		failSystemTask(task, runnerID, err)
		return
	}
	if err := model.FinishSystemTask(task.TaskID, runnerID, model.SystemTaskStatusSucceeded, nil, ""); err != nil {
		common.SysError("could not finish lottery draw task")
	}
}

type lotteryMailHandler struct{}

func (lotteryMailHandler) Type() string { return "lottery_mail" }
func (lotteryMailHandler) Enabled() bool {
	var n int64
	return model.DB.Model(&model.LotteryEntry{}).Where("email_status IN ? AND email_next_at <= ?", []string{"pending", "sending"}, time.Now().Unix()).Limit(1).Count(&n).Error == nil && n > 0
}
func (lotteryMailHandler) Interval() time.Duration { return 30 * time.Second }
func (lotteryMailHandler) NewPayload() any         { return nil }
func init() {
	RegisterSystemTaskHandler(lotteryDrawHandler{})
	RegisterSystemTaskHandler(lotteryMailHandler{})
}

func (lotteryMailHandler) Run(ctx context.Context, task *model.SystemTask, runnerID string) {
	err := DispatchLotteryEmails(ctx, common.SendNotificationEmail)
	if err != nil {
		failSystemTask(task, runnerID, err)
		return
	}
	if err := model.FinishSystemTask(task.TaskID, runnerID, model.SystemTaskStatusSucceeded, nil, ""); err != nil {
		common.SysError("could not finish lottery mail task")
	}
}

// DispatchLotteryEmails reuses the bounded SMTP transport. An expired send lease
// is marked uncertain instead of automatically sending a possible duplicate.
func DispatchLotteryEmails(ctx context.Context, send func(context.Context, common.CampaignEmailMessage) error) error {
	var entries []model.LotteryEntry
	now := time.Now().Unix()
	if err := model.DB.WithContext(ctx).Where("email_status IN ? AND email_next_at <= ?", []string{"pending", "sending"}, now).Order("id").Limit(20).Find(&entries).Error; err != nil {
		return err
	}
	for _, entry := range entries {
		now = time.Now().Unix()
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.EmailStatus == "sending" {
			if err := model.DB.Model(&model.LotteryEntry{}).Where("id = ? AND email_status = ? AND email_next_at <= ? AND email_attempts = ?", entry.ID, "sending", now, entry.EmailAttempts).Update("email_status", "uncertain").Error; err != nil {
				return err
			}
			continue
		}
		claim := model.DB.WithContext(ctx).Model(&model.LotteryEntry{}).Where("id = ? AND email_status = ? AND email_next_at <= ? AND email_attempts = ?", entry.ID, "pending", now, entry.EmailAttempts).Updates(map[string]any{"email_status": "sending", "email_attempts": gorm.Expr("email_attempts + 1"), "email_next_at": now + 120})
		if claim.Error != nil {
			return claim.Error
		}
		if claim.RowsAffected == 0 {
			continue
		}
		var user model.User
		var lottery model.Lottery
		err := model.DB.Select("id", "email").First(&user, entry.UserID).Error
		if err == nil {
			err = model.DB.First(&lottery, entry.LotteryID).Error
		}
		if err == nil {
			common.OptionMapRWMutex.RLock()
			from := common.SMTPFrom
			if from == "" {
				from = common.SMTPAccount
			}
			common.OptionMapRWMutex.RUnlock()
			sender, normalizeErr := model.NormalizeCampaignEmail(from)
			if normalizeErr != nil {
				err = normalizeErr
			} else {
				domain := sender[strings.LastIndex(sender, "@")+1:]
				body := fmt.Sprintf("恭喜你在「%s」抽奖中中奖！\n奖品：%s\n请登录站内查看中奖记录，并联系客服领取奖品。", lottery.Title, lottery.Prize)
				err = send(ctx, common.CampaignEmailMessage{Subject: "抽奖中奖通知 / Lottery winner", Receiver: user.Email, TextBody: body, HTMLBody: "<p>" + strings.ReplaceAll(html.EscapeString(body), "\n", "</p><p>") + "</p>", MessageID: fmt.Sprintf("<lottery-%d-%d@%s>", entry.LotteryID, entry.ID, domain)})
			}
		}
		state, next := "sent", int64(0)
		if err != nil {
			state = "pending"
			next = time.Now().Unix() + 300
			if entry.EmailAttempts+1 >= 5 {
				state = "failed"
			}
			var deliveryErr *common.CampaignEmailDeliveryError
			if errors.As(err, &deliveryErr) && deliveryErr.Uncertain {
				state = "uncertain"
			}
		}
		if err := model.DB.Model(&model.LotteryEntry{}).Where("id = ? AND email_status = ? AND email_attempts = ?", entry.ID, "sending", entry.EmailAttempts+1).Updates(map[string]any{"email_status": state, "email_next_at": next}).Error; err != nil {
			return err
		}
	}
	return nil
}
