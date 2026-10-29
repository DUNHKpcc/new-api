package controller

import (
	"errors"
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
func AdminCreateGuardianService(c *gin.Context) {
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
	if err := model.DB.Create(&service).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, service)
}

// The payload replaces the whole service, so an omitted enabled means delisted.
func AdminUpdateGuardianService(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "服务 ID 无效")
		return
	}
	var service model.GuardianService
	if err := c.ShouldBindJSON(&service); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if err := service.Validate(); err != nil {
		common.ApiError(c, err)
		return
	}
	var previous model.GuardianService
	if err := model.DB.First(&previous, id).Error; err != nil {
		common.ApiErrorMsg(c, "服务不存在")
		return
	}
	service.ID = id
	service.CreatedAt = previous.CreatedAt
	service.UpdatedAt = time.Now().Unix()
	if err := model.DB.Model(&previous).Select("title", "subtitle", "description", "features", "price_cents", "delivery_note", "badge", "route", "enabled", "sort_order", "updated_at").Updates(&service).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, service)
}

// New and resumed payments sign the same cashier arguments for the order.
func purchaseGuardianOrder(order *model.GuardianServiceOrder) (string, map[string]string, error) {
	client := GetEpayClient()
	if client == nil {
		return "", nil, errors.New("epay client is not configured")
	}
	callback := service.GetCallbackAddress()
	notifyURL, err := url.Parse(callback + "/api/guardian/epay/notify")
	if err != nil {
		return "", nil, err
	}
	returnURL, err := url.Parse(callback + "/api/guardian/epay/return")
	if err != nil {
		return "", nil, err
	}
	return client.Purchase(&epay.PurchaseArgs{Type: order.PaymentMethod, ServiceTradeNo: order.TradeNo, Name: "SVC:" + order.ServiceTitle, Money: fmt.Sprintf("%d.%02d", order.AmountCents/100, order.AmountCents%100), Device: epay.PC, NotifyUrl: notifyURL, ReturnUrl: returnURL})
}

func guardianPayMethodEnabled(method string) bool {
	return slices.ContainsFunc(guardianPayMethods(), func(m map[string]string) bool { return m["type"] == method })
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
	if !guardianPayMethodEnabled(req.PaymentMethod) {
		common.ApiErrorMsg(c, "支付方式不存在")
		return
	}
	if GetEpayClient() == nil {
		common.ApiErrorMsg(c, "当前管理员未配置支付信息")
		return
	}
	order, err := model.CreateGuardianOrder(c.GetInt("id"), req)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	uri, params, err := purchaseGuardianOrder(order)
	if err != nil {
		logger.LogError(c, "Guardian payment initiation failed: "+err.Error())
		if expireErr := model.ExpireGuardianPaymentInitiation(order.ID); expireErr != nil {
			logger.LogError(c, "Guardian payment initiation cleanup failed: "+expireErr.Error())
		}
		common.ApiErrorMsg(c, "拉起支付失败")
		return
	}
	common.ApiSuccess(c, gin.H{"trade_no": order.TradeNo, "pay_url": uri, "pay_params": params})
}
func PayGuardianOrder(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "订单 ID 无效")
		return
	}
	order, err := model.GetPayableGuardianOrder(c.GetInt("id"), id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !guardianPayMethodEnabled(order.PaymentMethod) {
		common.ApiErrorMsg(c, "该支付方式已停用，请取消后重新下单")
		return
	}
	uri, params, err := purchaseGuardianOrder(order)
	if err != nil {
		logger.LogError(c, "Guardian payment resume failed: "+err.Error())
		common.ApiErrorMsg(c, "拉起支付失败")
		return
	}
	common.ApiSuccess(c, gin.H{"trade_no": order.TradeNo, "pay_url": uri, "pay_params": params})
}
func CancelGuardianOrder(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "订单 ID 无效")
		return
	}
	order, err := model.CancelGuardianOrder(c.GetInt("id"), id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, order)
}
func ListMyGuardianOrders(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("p", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	orders, total, err := model.ListUserGuardianOrders(c.GetInt("id"), max(1, page), min(50, max(1, size)))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"items": orders, "total": total})
}
func AdminListGuardianOrders(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("p", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	orders, total, err := model.ListAdminGuardianOrders(c.Query("status"), c.Query("keyword"), max(1, page), min(50, max(1, size)), c.Query("payment_anomaly") == "true")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"items": orders, "total": total})
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
	order, err := model.UpdateGuardianOrder(id, c.GetInt("id"), req.Status, req.AdminNote)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	// The change is committed; a deleted buyer only leaves the username blank.
	var user model.User
	if err := model.DB.Unscoped().Select("id", "username").Where("id = ?", order.UserID).Find(&user).Error; err != nil {
		logger.LogError(c, "Guardian order buyer lookup failed: "+err.Error())
	}
	common.ApiSuccess(c, model.AdminGuardianOrder{GuardianServiceOrder: *order, PaymentAnomaly: order.HasPaymentAnomaly, AdminUserID: order.UserID, Username: user.Username})
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

func UpdateGuardianContact(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "订单 ID 无效")
		return
	}
	var req model.GuardianContactUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	order, err := model.UpdateGuardianContact(c.GetInt("id"), id, req)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, order)
}

func AdminGuardianOrderHistory(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "订单 ID 无效")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("p", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	page, size = max(1, page), min(50, max(1, size))
	var total int64
	var items any
	if c.Query("kind") == "payments" {
		rows := []model.GuardianPaymentAnomaly{}
		query := model.DB.Model(&model.GuardianPaymentAnomaly{}).Where("order_id = ?", id)
		err = query.Count(&total).Error
		if err == nil {
			err = query.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&rows).Error
		}
		items = rows
	} else {
		rows := []model.GuardianOrderEvent{}
		query := model.DB.Model(&model.GuardianOrderEvent{}).Where("order_id = ?", id)
		err = query.Count(&total).Error
		if err == nil {
			err = query.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&rows).Error
		}
		items = rows
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"items": items, "total": total})
}
