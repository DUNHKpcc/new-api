package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"net/url"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/system_setting"
)

type guardianReceiptHandler struct{}

func (guardianReceiptHandler) Type() string            { return "guardian_receipts" }
func (guardianReceiptHandler) Enabled() bool           { return model.HasGuardianReceiptWork() }
func (guardianReceiptHandler) Interval() time.Duration { return 15 * time.Second }
func (guardianReceiptHandler) NewPayload() any         { return nil }
func init()                                            { RegisterSystemTaskHandler(guardianReceiptHandler{}) }
func (guardianReceiptHandler) Run(ctx context.Context, task *model.SystemTask, runnerID string) {
	err := DispatchGuardianReceipts(ctx)
	status, message := model.SystemTaskStatusSucceeded, ""
	if err != nil {
		status, message = model.SystemTaskStatusFailed, "Guardian receipt notification processing failed"
	}
	if err := model.FinishSystemTask(task.TaskID, runnerID, status, nil, message); err != nil {
		common.SysError("could not finish Guardian receipt task")
	}
}

var guardianReceiptTemplate = template.Must(template.New("guardian-receipt").Parse(`<html><body><h1>Claude Guardian 付款成功 / Payment received</h1><p>{{.Title}}</p><p>订单 / Order: {{.TradeNo}}</p><p>实付 / Paid: CNY {{.Amount}}</p><p>付款时间 / Payment time: {{.PaidAt}}</p><p><a href="{{.URL}}">查看付款收据 / View receipt</a>（请登录 / Sign in required）</p><p>本收据用于记录订单付款，不作为税务发票。</p></body></html>`))

func BuildGuardianReceiptMessage(receipt model.GuardianReceipt) (common.CampaignEmailMessage, error) {
	common.OptionMapRWMutex.RLock()
	address := system_setting.ServerAddress
	common.OptionMapRWMutex.RUnlock()
	base, err := url.Parse(address)
	if err != nil || base.Scheme != "https" || base.Hostname() == "" || base.User != nil || base.RawQuery != "" || base.Fragment != "" {
		return common.CampaignEmailMessage{}, errors.New("configure an HTTPS server address")
	}
	base.Path = strings.TrimRight(base.Path, "/") + "/guardian/receipt"
	base.RawQuery = url.Values{"order": {fmt.Sprint(receipt.OrderID)}}.Encode()
	amount := fmt.Sprintf("%d.%02d", receipt.AmountCents/100, receipt.AmountCents%100)
	var body bytes.Buffer
	err = guardianReceiptTemplate.Execute(&body, struct{ Title, TradeNo, Amount, PaidAt, URL string }{receipt.ServiceTitle, receipt.TradeNo, amount, time.Unix(receipt.PaidAt, 0).UTC().Format(time.RFC3339), base.String()})
	if err != nil {
		return common.CampaignEmailMessage{}, err
	}
	return common.CampaignEmailMessage{Subject: "Claude Guardian 付款收据 / Payment receipt", Receiver: receipt.Email, HTMLBody: body.String(), TextBody: fmt.Sprintf("%s\nOrder: %s\nCNY %s\n%s", receipt.ServiceTitle, receipt.TradeNo, amount, base.String()), MessageID: fmt.Sprintf("<guardian-%d-%d@%s>", receipt.OrderID, receipt.RequestedAt, base.Hostname())}, nil
}

func DispatchGuardianReceipts(ctx context.Context) error {
	if err := model.PrepareGuardianReceipts(); err != nil {
		return err
	}
	for range 20 {
		if err := ctx.Err(); err != nil {
			return err
		}
		receipt, err := model.ClaimGuardianReceipt(time.Now().Unix())
		if err != nil {
			return err
		}
		if receipt == nil {
			return nil
		}
		message, err := BuildGuardianReceiptMessage(*receipt)
		if err == nil {
			err = common.SendNotificationEmail(ctx, message)
		}
		status := "sent"
		if err != nil {
			status = "failed"
			var smtpErr *common.CampaignEmailDeliveryError
			if errors.As(err, &smtpErr) {
				if smtpErr.Uncertain {
					status = "uncertain"
				} else if smtpErr.Temporary {
					status = "pending"
				}
			}
		}
		if err := model.FinishGuardianReceiptEmail(*receipt, status); err != nil {
			return err
		}
	}
	return nil
}
