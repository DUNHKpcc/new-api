package controller

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

const maxExternalRevenueAmountMinor int64 = 1_000_000_000_000_000

var externalRevenueSources = map[string]struct{}{
	"xianyu": {},
	"wechat": {},
	"alipay": {},
	"other":  {},
}

type externalRevenueMutationRequest struct {
	Source          string `json:"source"`
	SourceLabel     string `json:"source_label"`
	ExternalOrderNo string `json:"external_order_no"`
	AmountMinor     string `json:"amount_minor"`
	Currency        string `json:"currency"`
	OccurredAt      int64  `json:"occurred_at"`
	Note            string `json:"note"`
	Version         int64  `json:"version"`
}

type externalRevenueVoidRequest struct {
	Version int64 `json:"version"`
}

func normalizeExternalRevenueRequest(req externalRevenueMutationRequest, adminId int, now int64) (*model.ExternalRevenueRecord, error) {
	req.Source = strings.ToLower(strings.TrimSpace(req.Source))
	if _, ok := externalRevenueSources[req.Source]; !ok {
		return nil, errors.New("unsupported external revenue source")
	}
	req.SourceLabel = strings.TrimSpace(req.SourceLabel)
	if req.Source == "other" && req.SourceLabel == "" {
		return nil, errors.New("source label is required for other revenue")
	}
	if req.Source != "other" {
		req.SourceLabel = ""
	}
	if len(req.SourceLabel) > 64 {
		return nil, errors.New("source label is too long")
	}

	amountMinor, err := strconv.ParseInt(strings.TrimSpace(req.AmountMinor), 10, 64)
	if err != nil || amountMinor <= 0 || amountMinor > maxExternalRevenueAmountMinor {
		return nil, errors.New("amount must be a positive minor-unit integer within the supported range")
	}
	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	if len(req.Currency) != 3 {
		return nil, errors.New("currency must be a three-letter code")
	}
	for _, char := range req.Currency {
		if char < 'A' || char > 'Z' {
			return nil, errors.New("currency must contain only letters")
		}
	}
	if req.OccurredAt < 946684800 || req.OccurredAt > now+86400 {
		return nil, errors.New("occurred time is outside the supported range")
	}
	req.Note = strings.TrimSpace(req.Note)
	if len(req.Note) > 500 {
		return nil, errors.New("note is too long")
	}

	orderNo := strings.TrimSpace(req.ExternalOrderNo)
	if len(orderNo) > 128 {
		return nil, errors.New("external order number is too long")
	}
	var orderNoPointer *string
	if orderNo != "" {
		orderNoPointer = &orderNo
	}
	return &model.ExternalRevenueRecord{
		Source:          req.Source,
		SourceLabel:     req.SourceLabel,
		ExternalOrderNo: orderNoPointer,
		AmountMinor:     amountMinor,
		Currency:        req.Currency,
		OccurredAt:      req.OccurredAt,
		Note:            req.Note,
		Status:          model.ExternalRevenueStatusActive,
		Version:         1,
		CreatedBy:       adminId,
		UpdatedBy:       adminId,
		CreateTime:      now,
		UpdateTime:      now,
	}, nil
}

func externalRevenueError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, model.ErrExternalRevenueNotFound):
		common.ApiErrorMsg(c, "第三方收入记录不存在")
	case errors.Is(err, model.ErrExternalRevenueDuplicate):
		common.ApiErrorMsg(c, "该平台订单号已存在")
	case errors.Is(err, model.ErrExternalRevenueVersionConflict):
		common.ApiErrorMsg(c, "记录已被其他管理员修改，请刷新后重试")
	case errors.Is(err, model.ErrExternalRevenueVoided):
		common.ApiErrorMsg(c, "已作废的记录不能修改")
	default:
		common.ApiError(c, err)
	}
}

func ListExternalRevenue(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	filter := model.ExternalRevenueListFilter{
		Source:   strings.ToLower(strings.TrimSpace(c.Query("source"))),
		Currency: strings.ToUpper(strings.TrimSpace(c.Query("currency"))),
		Status:   strings.ToLower(strings.TrimSpace(c.Query("status"))),
		Keyword:  strings.TrimSpace(c.Query("keyword")),
	}
	if filter.Source != "" {
		if _, ok := externalRevenueSources[filter.Source]; !ok {
			common.ApiErrorMsg(c, "第三方收入来源无效")
			return
		}
	}
	if filter.Currency != "" && len(filter.Currency) != 3 {
		common.ApiErrorMsg(c, "币种代码无效")
		return
	}
	if filter.Status != "" && filter.Status != model.ExternalRevenueStatusActive && filter.Status != model.ExternalRevenueStatusVoided {
		common.ApiErrorMsg(c, "记录状态无效")
		return
	}
	if len(filter.Keyword) > 128 {
		common.ApiErrorMsg(c, "搜索关键词过长")
		return
	}
	if value := c.Query("start_time"); value != "" {
		filter.StartTime, _ = strconv.ParseInt(value, 10, 64)
	}
	if value := c.Query("end_time"); value != "" {
		filter.EndTime, _ = strconv.ParseInt(value, 10, 64)
	}

	records, total, err := model.ListExternalRevenue(pageInfo, filter)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(records)
	common.ApiSuccess(c, pageInfo)
}

func CreateExternalRevenue(c *gin.Context) {
	var req externalRevenueMutationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	now := time.Now().Unix()
	record, err := normalizeExternalRevenueRequest(req, c.GetInt("id"), now)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	if err := model.CreateExternalRevenue(record); err != nil {
		externalRevenueError(c, err)
		return
	}
	recordManageAudit(c, "external_revenue.create", map[string]interface{}{
		"id": record.Id, "source": record.Source, "currency": record.Currency, "amount_minor": record.AmountMinor,
	})
	common.ApiSuccess(c, record)
}

func UpdateExternalRevenue(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "记录 ID 无效")
		return
	}
	var req externalRevenueMutationRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Version <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	now := time.Now().Unix()
	record, err := normalizeExternalRevenueRequest(req, c.GetInt("id"), now)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	record.Id = id
	if err := model.UpdateExternalRevenue(record, req.Version); err != nil {
		externalRevenueError(c, err)
		return
	}
	recordManageAudit(c, "external_revenue.update", map[string]interface{}{
		"id": record.Id, "source": record.Source, "currency": record.Currency, "amount_minor": record.AmountMinor,
	})
	common.ApiSuccess(c, record)
}

func VoidExternalRevenue(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "记录 ID 无效")
		return
	}
	var req externalRevenueVoidRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Version <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if err := model.VoidExternalRevenue(id, req.Version, c.GetInt("id"), time.Now().Unix()); err != nil {
		externalRevenueError(c, err)
		return
	}
	recordManageAudit(c, "external_revenue.void", map[string]interface{}{"id": id})
	common.ApiSuccess(c, nil)
}

func GetExternalRevenueSummary(c *gin.Context) {
	startTime, endTime, timezoneOffset, ok := revenueSummaryRange(c)
	if !ok {
		return
	}
	summary, err := model.GetExternalRevenueSummary(startTime, endTime, timezoneOffset)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, summary)
}

func revenueSummaryRange(c *gin.Context) (int64, int64, int, bool) {
	startTime, startErr := strconv.ParseInt(c.Query("start_time"), 10, 64)
	endTime, endErr := strconv.ParseInt(c.Query("end_time"), 10, 64)
	timezoneOffset, timezoneErr := strconv.Atoi(c.DefaultQuery("timezone_offset", "0"))
	if startErr != nil || endErr != nil || timezoneErr != nil || startTime <= 0 || endTime <= startTime || endTime-startTime > 366*86400 || timezoneOffset < -840 || timezoneOffset > 840 {
		common.ApiErrorMsg(c, "统计时间范围无效")
		return 0, 0, 0, false
	}
	return startTime, endTime, timezoneOffset, true
}

func GetPlatformRevenueSummary(c *gin.Context) {
	startTime, endTime, timezoneOffset, ok := revenueSummaryRange(c)
	if !ok {
		return
	}
	summary, err := model.GetPlatformRevenueSummary(startTime, endTime, timezoneOffset)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, summary)
}
