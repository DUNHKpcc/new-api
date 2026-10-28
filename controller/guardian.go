package controller

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"time"

	"github.com/Calcium-Ion/go-epay/epay"
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
)

func guardianPayMethods() []map[string]string {
	methods := []map[string]string{}
	for _, method := range operation_setting.PayMethods {
		if method["type"] != "" && !slices.Contains([]string{model.PaymentMethodStripe, model.PaymentMethodCreem, model.PaymentMethodWaffo, model.PaymentMethodWaffoPancake}, method["type"]) {
			methods = append(methods, map[string]string{"type": method["type"], "name": method["name"]})
		}
	}
	return methods
}

func GetGuardianCatalog(c *gin.Context) {
	services := []model.GuardianService{}
	if err := model.DB.Where("enabled = ?", true).Order("sort_order, id").Find(&services).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	methods := guardianPayMethods()
	common.ApiSuccess(c, gin.H{"services": services, "pay_methods": methods, "purchase_enabled": GetEpayClient() != nil && operation_setting.IsPaymentComplianceConfirmed() && len(methods) > 0})
}
func AdminListGuardianServices(c *gin.Context) {
	services := []model.GuardianService{}
	if err := model.DB.Order("sort_order, id").Find(&services).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, services)
}
func AdminSaveGuardianService(c *gin.Context) {
	service := model.GuardianService{Enabled: true}
	if err := c.ShouldBindJSON(&service); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if err := service.Validate(); err != nil {
		common.ApiError(c, err)
		return
	}
	service.ID = 0
	service.CreatedAt = time.Now().Unix()
	service.UpdatedAt = service.CreatedAt
	if c.Param("id") != "" {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil || id <= 0 {
			common.ApiErrorMsg(c, "服务 ID 无效")
			return
		}
		var previous model.GuardianService
		if err := model.DB.First(&previous, id).Error; err != nil {
			common.ApiErrorMsg(c, "服务不存在")
			return
		}
		service.ID = id
		service.CreatedAt = previous.CreatedAt
		if err := model.DB.Model(&previous).Select("title", "subtitle", "description", "features", "price_cents", "delivery_note", "badge", "route", "enabled", "sort_order", "updated_at").Updates(&service).Error; err != nil {
			common.ApiError(c, err)
			return
		}
	} else if err := model.DB.Create(&service).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, service)
}
func CreateGuardianOrder(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}
	var req model.GuardianOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if err := req.Validate(); err != nil {
		common.ApiError(c, err)
		return
	}
	if !operation_setting.ContainsPayMethod(req.PaymentMethod) || !slices.ContainsFunc(guardianPayMethods(), func(m map[string]string) bool { return m["type"] == req.PaymentMethod }) {
		common.ApiErrorMsg(c, "支付方式不存在")
		return
	}
	client := GetEpayClient()
	if client == nil {
		common.ApiErrorMsg(c, "当前管理员未配置支付信息")
		return
	}
	callback := service.GetCallbackAddress()
	notifyURL, err := url.Parse(callback + "/api/guardian/epay/notify")
	if err != nil {
		common.ApiErrorMsg(c, "回调地址配置错误")
		return
	}
	returnURL, err := url.Parse(callback + "/api/guardian/epay/return")
	if err != nil {
		common.ApiErrorMsg(c, "回调地址配置错误")
		return
	}
	order, err := model.CreateGuardianOrder(c.GetInt("id"), req)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	uri, params, err := client.Purchase(&epay.PurchaseArgs{Type: req.PaymentMethod, ServiceTradeNo: order.TradeNo, Name: "SVC:" + order.ServiceTitle, Money: fmt.Sprintf("%d.%02d", order.AmountCents/100, order.AmountCents%100), Device: epay.PC, NotifyUrl: notifyURL, ReturnUrl: returnURL})
	if err != nil {
		if expireErr := model.DB.Model(order).Where("status = ?", "pending_payment").Updates(map[string]any{"status": "expired", "updated_at": time.Now().Unix()}).Error; expireErr != nil {
			logger.LogError(c, "Guardian payment initiation cleanup failed: "+expireErr.Error())
		}
		common.ApiErrorMsg(c, "拉起支付失败")
		return
	}
	common.ApiSuccess(c, gin.H{"trade_no": order.TradeNo, "pay_url": uri, "pay_params": params})
}
func ListGuardianOrders(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("p", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	page = max(1, page)
	size = min(50, max(1, size))
	userID := c.GetInt("id")
	status, keyword := "", ""
	admin := c.FullPath() == "/api/guardian/admin/orders"
	if admin {
		userID = 0
		status = c.Query("status")
		keyword = c.Query("keyword")
	}
	orders, total, err := model.ListGuardianOrders(userID, status, keyword, page, size)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if admin {
		common.ApiSuccess(c, gin.H{"items": orders, "total": total})
		return
	}
	items := make([]model.GuardianServiceOrder, 0, len(orders))
	for _, order := range orders {
		items = append(items, order.GuardianServiceOrder)
	}
	common.ApiSuccess(c, gin.H{"items": items, "total": total})
}
func AdminUpdateGuardianOrder(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "订单 ID 无效")
		return
	}
	var req struct {
		Status    string `json:"status"`
		AdminNote string `json:"admin_note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if err := model.UpdateGuardianOrder(id, req.Status, req.AdminNote); err != nil {
		common.ApiError(c, err)
		return
	}
	var order model.GuardianServiceOrder
	if err := model.DB.First(&order, id).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	var user model.User
	if err := model.DB.Select("id", "username").First(&user, order.UserID).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, model.AdminGuardianOrder{GuardianServiceOrder: order, AdminUserID: order.UserID, Username: user.Username})
}

// Both callback transports share signature verification and the same order lock.
func guardianEpayResult(c *gin.Context) string {
	var values url.Values
	if c.Request.Method == http.MethodPost {
		if err := c.Request.ParseForm(); err != nil {
			return "fail"
		}
		values = c.Request.PostForm
	} else {
		values = c.Request.URL.Query()
	}
	params := make(map[string]string, len(values))
	for key := range values {
		params[key] = values.Get(key)
	}
	client := GetEpayClient()
	if len(params) == 0 || client == nil {
		return "fail"
	}
	info, err := client.Verify(params)
	if err != nil || !info.VerifyStatus {
		return "fail"
	}
	if info.TradeStatus != epay.StatusTradeSuccess {
		return "pending"
	}
	LockOrder(info.ServiceTradeNo)
	defer UnlockOrder(info.ServiceTradeNo)
	// Persist only verified business fields; the reusable signature is not retained.
	payload, err := common.Marshal(info)
	if err != nil {
		return "fail"
	}
	if err := model.CompleteGuardianOrder(info.ServiceTradeNo, string(payload), info.Money, info.Type); err != nil {
		logger.LogError(c, "Guardian payment completion failed for "+info.ServiceTradeNo+": "+err.Error())
		return "fail"
	}
	return "success"
}
func GuardianEpayNotify(c *gin.Context) {
	result := guardianEpayResult(c)
	if result != "success" {
		result = "fail"
	}
	c.String(http.StatusOK, result)
}
func GuardianEpayReturn(c *gin.Context) {
	c.Redirect(http.StatusFound, paymentReturnPath("/guardian/orders?pay="+guardianEpayResult(c)))
}
