package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type emailCampaignInput struct {
	Subject  string `json:"subject"`
	Body     string `json:"body"`
	Category string `json:"category"`
	Group    string `json:"group"`
}

func emailCampaignError(c *gin.Context, err error) {
	status, message := http.StatusInternalServerError, "could not process email campaign request"
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		status, message = http.StatusNotFound, "email campaign or subscription was not found"
	case errors.Is(err, model.ErrInvalidEmailCampaign):
		status, message = http.StatusBadRequest, "use a subject of at most 200 characters, a plain-text body of at most 20000 UTF-8 bytes, and a valid category"
	case errors.Is(err, model.ErrEmailCampaignNotDraft):
		status, message = http.StatusConflict, "only drafts can be edited or queued"
	case errors.Is(err, model.ErrEmailAudienceEmpty):
		status, message = http.StatusBadRequest, "no subscribed users with valid email addresses match this audience"
	}
	c.JSON(status, gin.H{"success": false, "message": message})
}

func emailCampaignID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid email campaign id"})
		return 0, false
	}
	return id, true
}

func emailCampaignPage(c *gin.Context) (page, size int) {
	page, _ = strconv.Atoi(c.DefaultQuery("p", "1"))
	size, _ = strconv.Atoi(c.DefaultQuery("page_size", "20"))
	return min(max(page, 1), 1000000), min(max(size, 1), 100)
}

func GetEmailCampaignConfig(c *gin.Context) {
	config, err := model.GetEmailDispatchConfig()
	if err != nil {
		emailCampaignError(c, err)
		return
	}
	smtpReady, addressReady := service.EmailCampaignReadiness()
	common.ApiSuccess(c, gin.H{"rate_per_minute": config.RatePerMinute, "smtp_configured": smtpReady, "server_address_ready": addressReady})
}

func UpdateEmailCampaignConfig(c *gin.Context) {
	var input struct {
		RatePerMinute int `json:"rate_per_minute"`
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
	if c.ShouldBindJSON(&input) != nil || input.RatePerMinute < 1 || input.RatePerMinute > 120 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "rate_per_minute must be between 1 and 120"})
		return
	}
	if err := model.UpdateEmailDispatchRate(input.RatePerMinute); err != nil {
		emailCampaignError(c, err)
		return
	}
	GetEmailCampaignConfig(c)
}

func ListEmailCampaigns(c *gin.Context) {
	page, size := emailCampaignPage(c)
	items, total, err := model.ListEmailCampaigns((page-1)*size, size)
	if err != nil {
		emailCampaignError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"items": items, "total": total, "p": page, "page_size": size})
}

func CreateEmailCampaign(c *gin.Context) {
	var input emailCampaignInput
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 128*1024)
	if c.ShouldBindJSON(&input) != nil {
		emailCampaignError(c, model.ErrInvalidEmailCampaign)
		return
	}
	campaign := &model.EmailCampaign{Subject: strings.TrimSpace(input.Subject), Body: input.Body, Category: input.Category, TargetGroup: strings.TrimSpace(input.Group), CreatedBy: c.GetInt("id")}
	if err := model.CreateEmailCampaign(campaign); err != nil {
		emailCampaignError(c, err)
		return
	}
	common.ApiSuccess(c, campaign)
}

func UpdateEmailCampaign(c *gin.Context) {
	id, ok := emailCampaignID(c)
	if !ok {
		return
	}
	var input emailCampaignInput
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 128*1024)
	if c.ShouldBindJSON(&input) != nil {
		emailCampaignError(c, model.ErrInvalidEmailCampaign)
		return
	}
	if err := model.UpdateEmailCampaign(id, strings.TrimSpace(input.Subject), input.Body, input.Category, strings.TrimSpace(input.Group)); err != nil {
		emailCampaignError(c, err)
		return
	}
	campaign, err := model.GetEmailCampaign(id)
	if err != nil {
		emailCampaignError(c, err)
		return
	}
	common.ApiSuccess(c, campaign)
}

func PreviewEmailCampaign(c *gin.Context) {
	id, ok := emailCampaignID(c)
	if !ok {
		return
	}
	campaign, err := model.GetEmailCampaign(id)
	if err != nil {
		emailCampaignError(c, err)
		return
	}
	count, err := model.PreviewEmailCampaignRecipients(campaign.Category, campaign.TargetGroup)
	if err != nil {
		emailCampaignError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"eligible": count})
}

func QueueEmailCampaign(c *gin.Context) {
	id, ok := emailCampaignID(c)
	if !ok {
		return
	}
	smtpReady, addressReady := service.EmailCampaignReadiness()
	if !smtpReady || !addressReady {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "configure SMTP and an HTTPS server address before queueing emails"})
		return
	}
	campaign, err := model.QueueEmailCampaign(id)
	if err != nil {
		emailCampaignError(c, err)
		return
	}
	// The periodic scheduler also scans the durable queue if this wakeup fails.
	if _, _, err := service.EnqueueSystemTask(service.EmailDispatchTaskType, nil); err != nil {
		common.SysError("email campaign queued; immediate dispatcher wakeup failed")
	}
	common.ApiSuccess(c, campaign)
}

func CancelEmailCampaign(c *gin.Context) {
	id, ok := emailCampaignID(c)
	if !ok {
		return
	}
	if err := model.CancelEmailCampaign(id); err != nil {
		emailCampaignError(c, err)
		return
	}
	campaign, err := model.GetEmailCampaign(id)
	if err != nil {
		emailCampaignError(c, err)
		return
	}
	common.ApiSuccess(c, campaign)
}

func ListEmailCampaignDeliveries(c *gin.Context) {
	id, ok := emailCampaignID(c)
	if !ok {
		return
	}
	page, size := emailCampaignPage(c)
	status := c.Query("status")
	switch status {
	case "", model.EmailDeliveryPending, model.EmailDeliverySending, model.EmailDeliveryRetry, model.EmailDeliverySent, model.EmailDeliveryFailed, model.EmailDeliverySkipped, model.EmailDeliveryUncertain, model.EmailDeliveryCancelled:
	default:
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid delivery status"})
		return
	}
	items, total, err := model.ListEmailDeliveries(id, (page-1)*size, size, status)
	if err != nil {
		emailCampaignError(c, err)
		return
	}
	for i := range items {
		items[i].NextAttemptAt /= 1000
	}
	common.ApiSuccess(c, gin.H{"items": items, "total": total, "p": page, "page_size": size})
}

func GetEmailSubscriptions(c *gin.Context) {
	preferences, err := model.GetEmailPreferences(c.GetInt("id"))
	if err != nil {
		emailCampaignError(c, err)
		return
	}
	common.ApiSuccess(c, preferences)
}

func UpdateEmailSubscriptions(c *gin.Context) {
	var input struct {
		Promotion *bool `json:"promotion"`
		Platform  *bool `json:"platform"`
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
	if c.ShouldBindJSON(&input) != nil || input.Promotion == nil || input.Platform == nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "both subscription preferences are required"})
		return
	}
	if err := model.SetEmailPreferences(c.GetInt("id"), *input.Promotion, *input.Platform); err != nil {
		emailCampaignError(c, err)
		return
	}
	GetEmailSubscriptions(c)
}

func GetEmailUnsubscribe(c *gin.Context) {
	c.Header("Referrer-Policy", "no-referrer")
	info, err := model.GetEmailUnsubscribeInfo(c.Query("token"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "this unsubscribe link is invalid"})
		return
	}
	common.ApiSuccess(c, gin.H{"category": info.Category, "unsubscribed": !info.Subscribed})
}

func UnsubscribeEmail(c *gin.Context) {
	c.Header("Referrer-Policy", "no-referrer")
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
	confirmed := false
	switch c.ContentType() {
	case "application/json":
		var input struct {
			Confirm bool `json:"confirm"`
		}
		confirmed = c.ShouldBindJSON(&input) == nil && input.Confirm
	case "application/x-www-form-urlencoded", "multipart/form-data":
		confirmed = c.PostForm("List-Unsubscribe") == "One-Click"
	}
	if !confirmed {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "unsubscribe confirmation is required"})
		return
	}
	if err := model.UnsubscribeEmail(c.Query("token")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "this unsubscribe link is invalid"})
		return
	}
	GetEmailUnsubscribe(c)
}
