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

const EmailDispatchTaskType = "email_dispatch"

var campaignEmailTemplate = template.Must(template.New("campaign-email").Parse(`<!doctype html>
<html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"></head>
<body style="margin:0;background:#f5f5f5;font-family:Arial,sans-serif;color:#202020">
<table role="presentation" width="100%"><tr><td align="center" style="padding:24px 12px">
<table role="presentation" width="100%" style="width:100%;max-width:560px;table-layout:fixed;background:white;border:1px solid #ddd"><tr><td style="padding:28px">
<img src="{{.LogoURL}}" width="48" height="48" alt="{{.SystemName}}" style="display:block;width:48px;height:48px;border:0">
<p style="color:#666;font-size:14px">{{.SystemName}} · {{.Category}}</p>
<h1 style="font-size:24px;line-height:1.4">{{.Subject}}</h1>
{{range .Paragraphs}}<p style="line-height:1.8;overflow-wrap:anywhere">{{.}}</p>{{end}}
<hr style="border:0;border-top:1px solid #eee;margin:28px 0">
<p style="font-size:12px;color:#666">此邮件由 {{.SystemName}} 发送。<a href="{{.UnsubscribeURL}}">退订此类邮件 / Unsubscribe</a></p>
</td></tr></table></td></tr></table></body></html>`))

// EmailCampaignReadiness reports only whether the existing configuration is usable;
// credentials and sender addresses never appear in the management response.
func EmailCampaignReadiness() (smtpConfigured, serverAddressReady bool) {
	common.OptionMapRWMutex.RLock()
	from := common.SMTPFrom
	if from == "" {
		from = common.SMTPAccount
	}
	_, addressErr := model.NormalizeCampaignEmail(from)
	smtpConfigured = common.SMTPServer != "" && common.SMTPPort > 0 && common.SMTPPort <= 65535 && addressErr == nil
	address := system_setting.ServerAddress
	common.OptionMapRWMutex.RUnlock()
	parsed, err := url.Parse(address)
	serverAddressReady = err == nil && parsed.Scheme == "https" && parsed.Hostname() != "" && parsed.User == nil && parsed.RawQuery == "" && parsed.Fragment == ""
	return
}

func BuildEmailCampaignMessage(job *model.EmailSendJob) (common.CampaignEmailMessage, error) {
	common.OptionMapRWMutex.RLock()
	address, systemName := system_setting.ServerAddress, common.SystemName
	common.OptionMapRWMutex.RUnlock()
	base, err := url.Parse(address)
	if err != nil || base.Scheme != "https" || base.Hostname() == "" || base.User != nil || base.RawQuery != "" || base.Fragment != "" {
		return common.CampaignEmailMessage{}, errors.New("configure an HTTPS server address before sending campaigns")
	}
	apiURL := *base
	apiURL.Path = strings.TrimRight(base.Path, "/") + "/api/email/unsubscribe"
	apiURL.RawQuery = url.Values{"token": {job.UnsubscribeToken}}.Encode()
	pageURL := *base
	pageURL.Path = strings.TrimRight(base.Path, "/") + "/email/unsubscribe"
	// A fragment keeps the manual-link capability out of page access logs and
	// referrers. The standalone API URL remains available for mail-client POSTs.
	pageURL.Fragment = url.Values{"token": {job.UnsubscribeToken}}.Encode()
	logoURL := *base
	logoURL.Path = strings.TrimRight(base.Path, "/") + "/pcc-agent-logo.png"
	category := "平台信息 / Platform updates"
	if job.Campaign.Category == model.EmailCategoryPromotion {
		category = "促销信息 / Promotions"
	}
	var htmlBody bytes.Buffer
	err = campaignEmailTemplate.Execute(&htmlBody, struct {
		SystemName     string
		Category       string
		Subject        string
		Paragraphs     []string
		UnsubscribeURL string
		LogoURL        string
	}{systemName, category, job.Campaign.Subject, strings.Split(strings.ReplaceAll(job.Campaign.Body, "\r\n", "\n"), "\n"), pageURL.String(), logoURL.String()})
	if err != nil {
		return common.CampaignEmailMessage{}, errors.New("could not render campaign email")
	}
	return common.CampaignEmailMessage{
		Subject: job.Campaign.Subject, Receiver: job.Delivery.Email,
		HTMLBody: htmlBody.String(), TextBody: job.Campaign.Body + "\n\n" + systemName + " · " + category + "\n退订 / Unsubscribe: " + pageURL.String(),
		MessageID: job.Delivery.MessageID, UnsubscribeURL: apiURL.String(),
	}, nil
}

type emailDispatchHandler struct{}

func (emailDispatchHandler) Type() string            { return EmailDispatchTaskType }
func (emailDispatchHandler) Enabled() bool           { return model.HasPendingEmailDeliveries() }
func (emailDispatchHandler) Interval() time.Duration { return 15 * time.Second }
func (emailDispatchHandler) NewPayload() any         { return nil }

func init() { RegisterSystemTaskHandler(emailDispatchHandler{}) }

func (emailDispatchHandler) Run(ctx context.Context, task *model.SystemTask, runnerID string) {
	processed, err := DispatchEmailCampaigns(ctx, task.TaskID)
	status, message := model.SystemTaskStatusSucceeded, ""
	if err != nil {
		status, message = model.SystemTaskStatusFailed, "email dispatch interrupted; delivery records contain individual outcomes"
	}
	if finishErr := model.FinishSystemTask(task.TaskID, runnerID, status, map[string]int{"processed": processed}, message); finishErr != nil {
		common.SysError("could not finish email dispatch task")
	}
}

// DispatchEmailCampaigns drains a bounded run under the system-task lease. The
// database claim also reserves a persisted rate slot, including across restarts.
func DispatchEmailCampaigns(ctx context.Context, workerID string) (int, error) {
	smtpReady, addressReady := EmailCampaignReadiness()
	if !smtpReady || !addressReady {
		return 0, errors.New("email campaign configuration is incomplete")
	}
	processed := 0
	deadline := time.Now().Add(2 * time.Minute)
	for processed < 100 && time.Now().Before(deadline) {
		if err := ctx.Err(); err != nil {
			return processed, err
		}
		delivery, err := model.ClaimEmailDelivery(workerID, time.Now().UnixMilli(), 60000)
		if err != nil {
			return processed, err
		}
		if delivery == nil {
			config, err := model.GetEmailDispatchConfig()
			if err != nil {
				return processed, err
			}
			wait := time.Until(time.UnixMilli(config.NextSendAt))
			if wait <= 0 || wait > time.Minute || !model.HasPendingEmailDeliveries() {
				break
			}
			timer := time.NewTimer(wait)
			select {
			case <-ctx.Done():
				timer.Stop()
				return processed, ctx.Err()
			case <-timer.C:
			}
			continue
		}
		if err := DeliverEmailCampaign(ctx, delivery, workerID); err != nil {
			return processed, err
		}
		processed++
	}
	return processed, nil
}

// DeliverEmailCampaign rechecks recipient eligibility immediately before SMTP.
// An ambiguous SMTP acknowledgement must never be automatically retried.
func DeliverEmailCampaign(ctx context.Context, delivery *model.EmailDelivery, workerID string) error {
	job, err := model.PrepareEmailDelivery(delivery.ID, workerID)
	if err != nil || job == nil {
		return err
	}
	message, err := BuildEmailCampaignMessage(job)
	if err == nil {
		// Database waits may have consumed part of the claim lease. Never start
		// or continue SMTP beyond the lease that still grants this worker ownership.
		sendCtx, cancel := context.WithDeadline(ctx, time.UnixMilli(job.Delivery.LockedUntil))
		err = common.SendCampaignEmail(sendCtx, message)
		cancel()
	}
	status, safeError, nextAttempt := model.EmailDeliverySent, "", int64(0)
	if err != nil {
		status, safeError = model.EmailDeliveryFailed, "email could not be submitted; check SMTP configuration"
		var deliveryErr *common.CampaignEmailDeliveryError
		if errors.As(err, &deliveryErr) {
			safeError = deliveryErr.Error()
			switch {
			case deliveryErr.Uncertain:
				status = model.EmailDeliveryUncertain
			case deliveryErr.Temporary && job.Delivery.Attempts < 3:
				status = model.EmailDeliveryRetry
				nextAttempt = time.Now().Add(time.Duration(job.Delivery.Attempts*job.Delivery.Attempts) * time.Minute).UnixMilli()
			}
		}
	}
	if err := model.FinishEmailDelivery(delivery.ID, workerID, status, safeError, nextAttempt); err != nil {
		return fmt.Errorf("could not persist email delivery %d outcome: %w", delivery.ID, err)
	}
	return nil
}
