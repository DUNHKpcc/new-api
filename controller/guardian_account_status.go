package controller

import (
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func GetGuardianAccountStatistics(c *gin.Context) {
	result, err := model.GetGuardianAccountStatistics()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func AdminListGuardianAccountStatuses(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("p"))
	size, _ := strconv.Atoi(c.Query("page_size"))
	if size <= 0 {
		size = 20
	}
	items, total, err := model.ListGuardianAccountStatuses(max(1, page), min(50, size), strings.TrimSpace(c.Query("keyword")))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"items": items, "total": total})
}

func AdminUpdateGuardianAccountStatus(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil || userID <= 0 {
		common.ApiErrorMsg(c, "用户 ID 无效")
		return
	}
	var request struct {
		Banned        *bool `json:"banned" binding:"required"`
		PhoneRequired *bool `json:"phone_required" binding:"required"`
		KYCRequired   *bool `json:"kyc_required" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	status := model.GuardianAccountStatus{UserID: userID, Banned: *request.Banned, PhoneRequired: *request.PhoneRequired, KYCRequired: *request.KYCRequired, UpdatedBy: c.GetInt("id")}
	if err := model.SaveGuardianAccountStatus(&status); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, status)
}
