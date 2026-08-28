/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
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

const maxRevenueCostAmountMinor int64 = 1_000_000_000_000_000

type revenueCostMutationRequest struct {
	Month       string `json:"month"`
	Category    string `json:"category"`
	Description string `json:"description"`
	AmountMinor string `json:"amount_minor"`
	Currency    string `json:"currency"`
	Version     int64  `json:"version"`
}

type revenueCostPageResponse struct {
	Items            []*model.RevenueCostRecord `json:"items"`
	Total            int                        `json:"total"`
	TotalAmountMinor int64                      `json:"total_amount_minor,string"`
	Page             int                        `json:"page"`
	PageSize         int                        `json:"page_size"`
}

func normalizeRevenueCostMonth(value string) (string, error) {
	month := strings.TrimSpace(value)
	parsed, err := time.Parse("2006-01", month)
	if err != nil || parsed.Format("2006-01") != month {
		return "", errors.New("month must use YYYY-MM format")
	}
	return month, nil
}

func normalizeRevenueCostCurrency(value string) (string, error) {
	currency := strings.ToUpper(strings.TrimSpace(value))
	if len(currency) != 3 {
		return "", errors.New("currency must be a three-letter code")
	}
	for _, char := range currency {
		if char < 'A' || char > 'Z' {
			return "", errors.New("currency must contain only letters")
		}
	}
	return currency, nil
}

func normalizeRevenueCostRequest(req revenueCostMutationRequest, adminId int, now int64) (*model.RevenueCostRecord, error) {
	month, err := normalizeRevenueCostMonth(req.Month)
	if err != nil {
		return nil, err
	}
	category := strings.ToLower(strings.TrimSpace(req.Category))
	if !model.IsRevenueCostCategory(category) {
		return nil, errors.New("unsupported revenue cost category")
	}
	amountMinor, err := strconv.ParseInt(strings.TrimSpace(req.AmountMinor), 10, 64)
	if err != nil || amountMinor < 0 || amountMinor > maxRevenueCostAmountMinor {
		return nil, errors.New("amount must be a non-negative minor-unit integer within the supported range")
	}
	currency, err := normalizeRevenueCostCurrency(req.Currency)
	if err != nil {
		return nil, err
	}
	description := strings.TrimSpace(req.Description)
	if len(description) > 500 {
		return nil, errors.New("description is too long")
	}
	return &model.RevenueCostRecord{
		Month:       month,
		Category:    category,
		Description: description,
		AmountMinor: amountMinor,
		Currency:    currency,
		Version:     1,
		CreatedBy:   adminId,
		UpdatedBy:   adminId,
		CreateTime:  now,
		UpdateTime:  now,
	}, nil
}

func revenueCostError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, model.ErrRevenueCostNotFound):
		common.ApiErrorMsg(c, "成本记录不存在")
	case errors.Is(err, model.ErrRevenueCostVersionConflict):
		common.ApiErrorMsg(c, "成本记录已被其他管理员修改，请刷新后重试")
	case errors.Is(err, model.ErrRevenueCostAmountInvalid):
		common.ApiErrorMsg(c, "成本金额超出支持范围")
	default:
		common.ApiError(c, err)
	}
}

func ListRevenueCosts(c *gin.Context) {
	filter := model.RevenueCostListFilter{
		Month:    strings.TrimSpace(c.Query("month")),
		Category: strings.ToLower(strings.TrimSpace(c.Query("category"))),
		Currency: strings.ToUpper(strings.TrimSpace(c.Query("currency"))),
	}
	if filter.Month != "" {
		if _, err := normalizeRevenueCostMonth(filter.Month); err != nil {
			common.ApiErrorMsg(c, err.Error())
			return
		}
	}
	if filter.Category != "" && !model.IsRevenueCostCategory(filter.Category) {
		common.ApiErrorMsg(c, "成本类别无效")
		return
	}
	if filter.Currency != "" {
		var err error
		filter.Currency, err = normalizeRevenueCostCurrency(filter.Currency)
		if err != nil {
			common.ApiErrorMsg(c, err.Error())
			return
		}
	}

	pageInfo := common.GetPageQuery(c)
	records, total, totalAmountMinor, err := model.ListRevenueCosts(pageInfo, filter)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, revenueCostPageResponse{
		Items:            records,
		Total:            int(total),
		TotalAmountMinor: totalAmountMinor,
		Page:             pageInfo.GetPage(),
		PageSize:         pageInfo.GetPageSize(),
	})
}

func CreateRevenueCost(c *gin.Context) {
	var req revenueCostMutationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	now := time.Now().Unix()
	record, err := normalizeRevenueCostRequest(req, c.GetInt("id"), now)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	if err := model.CreateRevenueCost(record); err != nil {
		revenueCostError(c, err)
		return
	}
	recordManageAudit(c, "revenue_cost.create", map[string]interface{}{
		"id": record.Id, "month": record.Month, "category": record.Category,
		"currency": record.Currency, "amount_minor": record.AmountMinor,
	})
	common.ApiSuccess(c, record)
}

func UpdateRevenueCost(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "记录 ID 无效")
		return
	}
	var req revenueCostMutationRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Version <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	now := time.Now().Unix()
	record, err := normalizeRevenueCostRequest(req, c.GetInt("id"), now)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	record.Id = id
	if err := model.UpdateRevenueCost(record, req.Version); err != nil {
		revenueCostError(c, err)
		return
	}
	recordManageAudit(c, "revenue_cost.update", map[string]interface{}{
		"id": record.Id, "month": record.Month, "category": record.Category,
		"currency": record.Currency, "amount_minor": record.AmountMinor,
	})
	common.ApiSuccess(c, record)
}
