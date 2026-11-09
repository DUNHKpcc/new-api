package controller

import (
	"errors"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
)

func GetMyGuardianReseller(c *gin.Context) {
	reseller, err := model.GetGuardianReseller(c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, reseller)
}

func GetGuardianResellerCatalog(c *gin.Context) {
	products, err := model.GuardianResellerCatalog(c.Param("code"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	methods := []map[string]string{}
	for _, method := range guardianPayMethods() {
		if method["type"] == "alipay" {
			methods = append(methods, method)
		}
	}
	common.ApiSuccess(c, gin.H{"services": products, "pay_methods": methods, "purchase_enabled": GetEpayClient() != nil && operation_setting.IsPaymentComplianceConfirmed() && len(methods) > 0})
}

func GetGuardianResellerOffers(c *gin.Context) {
	id := c.GetInt("id")
	if c.Param("id") != "" {
		var ok bool
		id, ok = guardianPathID(c, "id")
		if !ok {
			return
		}
	}
	offers, err := model.ListGuardianResellerOffers(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	products := []model.GuardianService{}
	query := model.DB.Order("sort_order, id")
	if c.Param("id") == "" {
		ids := []int{}
		for _, offer := range offers {
			ids = append(ids, offer.ServiceID)
		}
		query = query.Where("id IN ?", ids)
	}
	if err := query.Find(&products).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	setting, err := model.GetGuardianResellerSetting()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"offers": offers, "services": products, "setting": setting})
}

func AdminSaveGuardianSupply(c *gin.Context) {
	id, ok := guardianPathID(c, "id")
	if !ok {
		return
	}
	serviceID, ok := guardianPathID(c, "service_id")
	if !ok {
		return
	}
	var req model.GuardianSupplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if err := model.SaveGuardianSupply(id, serviceID, c.GetInt("id"), req); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func SaveGuardianRetail(c *gin.Context) {
	id, ok := guardianPathID(c, "service_id")
	if !ok {
		return
	}
	var req model.GuardianRetailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if err := model.SaveGuardianRetail(c.GetInt("id"), id, req); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func GetGuardianResellerBalance(c *gin.Context) {
	balance, err := model.GetGuardianResellerBalance(c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, balance)
}

func ListGuardianResellerOrders(c *gin.Context) {
	page, size := guardianPagination(c)
	items, total, err := model.ListGuardianResellerOrders(c.GetInt("id"), page, size)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"items": items, "total": total})
}

func AdminListGuardianResellerOrders(c *gin.Context) {
	page, size := guardianPagination(c)
	items, total, err := model.ListGuardianResellerOrders(0, page, size)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"items": items, "total": total})
}

func AdminGetGuardianResellerOrder(c *gin.Context) {
	id, ok := guardianPathID(c, "id")
	if !ok {
		return
	}
	order, err := model.GetGuardianResellerOrder(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, order)
}

func AdminConfirmGuardianFee(c *gin.Context) {
	id, ok := guardianPathID(c, "id")
	if !ok {
		return
	}
	var req struct {
		FeeCents  *int64 `json:"fee_cents" binding:"required"`
		Reference string `json:"reference"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if err := model.ConfirmGuardianResellerFee(id, c.GetInt("id"), model.GuardianFeeConfirmation{FeeCents: *req.FeeCents, Reference: req.Reference}); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func AdminGuardianResellerRefund(c *gin.Context) {
	id, ok := guardianPathID(c, "id")
	if !ok {
		return
	}
	var req model.GuardianResellerRefundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if err := model.UpdateGuardianResellerRefund(id, c.GetInt("id"), req); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func GetGuardianResellerSetting(c *gin.Context) {
	setting, err := model.GetGuardianResellerSetting()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, setting)
}

func AdminSaveGuardianResellerSetting(c *gin.Context) {
	var req struct {
		FeeBPS *int64 `json:"fee_bps" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if err := model.SaveGuardianResellerSetting(c.GetInt("id"), *req.FeeBPS); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func RequestGuardianWithdrawal(c *gin.Context) {
	var req model.GuardianWithdrawalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if err := req.Validate(); err != nil {
		common.ApiError(c, err)
		return
	}
	context, err := common.Marshal(req)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if middleware.RequireSecurityProof(c, service.VerificationOperation{Scope: service.VerificationScopeGuardianWithdrawal, Context: context}) == nil {
		return
	}
	withdrawal, err := model.CreateGuardianWithdrawal(c.GetInt("id"), req)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, withdrawal)
}

func ListGuardianWithdrawals(c *gin.Context) {
	page, size := guardianPagination(c)
	items, total, err := model.ListGuardianWithdrawals(c.GetInt("id"), page, size)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"items": items, "total": total})
}

func AdminListGuardianWithdrawals(c *gin.Context) {
	page, size := guardianPagination(c)
	items, total, err := model.ListGuardianWithdrawals(0, page, size)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"items": items, "total": total})
}

func CancelGuardianWithdrawal(c *gin.Context) {
	id, ok := guardianPathID(c, "id")
	if !ok {
		return
	}
	if err := model.UpdateGuardianWithdrawal(id, c.GetInt("id"), false, model.GuardianWithdrawalUpdate{Status: "cancelled", Note: "代理撤销申请"}); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func AdminUpdateGuardianWithdrawal(c *gin.Context) {
	id, ok := guardianPathID(c, "id")
	if !ok {
		return
	}
	var req model.GuardianWithdrawalUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if err := model.UpdateGuardianWithdrawal(id, c.GetInt("id"), true, req); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func guardianPathID(c *gin.Context, key string) (int, bool) {
	id, err := strconv.Atoi(c.Param(key))
	if err != nil || id <= 0 {
		common.ApiError(c, errors.New("无效的记录 ID"))
		return 0, false
	}
	return id, true
}

func guardianPagination(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("p", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	return max(1, page), min(50, max(1, size))
}

func AdminGetGuardianReseller(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil || userID <= 0 {
		common.ApiErrorMsg(c, "用户 ID 无效")
		return
	}
	reseller, err := model.GetGuardianReseller(userID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, reseller)
}

func AdminSetGuardianReseller(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil || userID <= 0 {
		common.ApiErrorMsg(c, "用户 ID 无效")
		return
	}
	var request model.GuardianResellerAccessRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	reseller, err := model.SetGuardianResellerAccess(userID, c.GetInt("id"), request)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, reseller)
}

func ListMyGuardianCustomerServices(c *gin.Context) {
	items, err := model.ListGuardianCustomerServices(c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, items)
}

func GetGuardianRenewalQuote(c *gin.Context) {
	id, ok := guardianPathID(c, "id")
	if !ok {
		return
	}
	quote, err := model.GetGuardianRenewalQuote(c.GetInt("id"), id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, quote)
}

func AdminListGuardianResellers(c *gin.Context) {
	items, err := model.ListGuardianResellers()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, items)
}
