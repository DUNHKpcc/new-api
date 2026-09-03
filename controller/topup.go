package controller

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/Calcium-Ion/go-epay/epay"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
)

const epayMinorUnitsPerMajor = int64(100)

func parseEpayPaidAmountMinor(money string) (int64, error) {
	amount, err := decimal.NewFromString(money)
	if err != nil || amount.LessThanOrEqual(decimal.Zero) {
		return 0, fmt.Errorf("invalid Epay paid amount")
	}
	scaled := amount.Mul(decimal.NewFromInt(epayMinorUnitsPerMajor))
	if !scaled.IsInteger() || scaled.GreaterThan(decimal.NewFromInt(math.MaxInt64)) {
		return 0, fmt.Errorf("invalid Epay paid amount precision or range")
	}
	return scaled.IntPart(), nil
}

type epayOrderPricing struct {
	PayMoney                string
	ExpectedAmountMinor     int64
	QuotaAmount             int
	LegacyAmount            int64
	UnitPriceSnapshot       string
	QuotaPerUnitSnapshot    string
	TopUpGroupRatioSnapshot string
	AmountDiscountSnapshot  string
}

func buildEpayOrderPricing(amount int64, group string) (epayOrderPricing, error) {
	if amount <= 0 || operation_setting.Price <= 0 ||
		math.IsNaN(operation_setting.Price) || math.IsInf(operation_setting.Price, 0) ||
		common.QuotaPerUnit <= 0 || math.IsNaN(common.QuotaPerUnit) || math.IsInf(common.QuotaPerUnit, 0) {
		return epayOrderPricing{}, errors.New("invalid Epay order pricing input")
	}

	quotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
	valueAmount := decimal.NewFromInt(amount)
	quotaAmountDecimal := valueAmount.Mul(quotaPerUnit)
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		quotaAmountDecimal = valueAmount
		valueAmount = valueAmount.Div(quotaPerUnit)
	}
	quotaAmount, clamp := common.QuotaFromDecimalChecked(quotaAmountDecimal)
	if clamp != nil || quotaAmount <= 0 {
		return epayOrderPricing{}, errors.New("invalid Epay order quota amount")
	}

	groupRatio := common.GetTopupGroupRatio(group)
	if groupRatio == 0 {
		groupRatio = 1
	}
	if groupRatio < 0 || math.IsNaN(groupRatio) || math.IsInf(groupRatio, 0) {
		return epayOrderPricing{}, errors.New("invalid Epay topup group ratio")
	}
	discount := 1.0
	if configured, ok := operation_setting.GetPaymentSetting().AmountDiscount[int(amount)]; ok && configured > 0 {
		discount = configured
	}
	if math.IsNaN(discount) || math.IsInf(discount, 0) {
		return epayOrderPricing{}, errors.New("invalid Epay amount discount")
	}

	unitPrice := decimal.NewFromFloat(operation_setting.Price)
	groupRatioDecimal := decimal.NewFromFloat(groupRatio)
	discountDecimal := decimal.NewFromFloat(discount)
	payMoney := valueAmount.
		Mul(unitPrice).
		Mul(groupRatioDecimal).
		Mul(discountDecimal).
		Round(2)
	if payMoney.LessThan(decimal.NewFromInt(1).Div(decimal.NewFromInt(epayMinorUnitsPerMajor))) {
		return epayOrderPricing{}, errors.New("Epay paid amount is below one minor unit")
	}
	payMoneyString := payMoney.StringFixed(2)
	expectedAmountMinor, err := parseEpayPaidAmountMinor(payMoneyString)
	if err != nil {
		return epayOrderPricing{}, err
	}

	return epayOrderPricing{
		PayMoney:                payMoneyString,
		ExpectedAmountMinor:     expectedAmountMinor,
		QuotaAmount:             quotaAmount,
		LegacyAmount:            valueAmount.Truncate(0).IntPart(),
		UnitPriceSnapshot:       unitPrice.String(),
		QuotaPerUnitSnapshot:    quotaPerUnit.String(),
		TopUpGroupRatioSnapshot: groupRatioDecimal.String(),
		AmountDiscountSnapshot:  discountDecimal.String(),
	}, nil
}

func GetTopUpInfo(c *gin.Context) {
	complianceConfirmed := operation_setting.IsPaymentComplianceConfirmed()

	// 获取支付方式
	payMethods := operation_setting.PayMethods
	if !complianceConfirmed {
		payMethods = []map[string]string{}
	}

	// 如果启用了 Stripe 支付，添加到支付方法列表
	if isStripeTopUpEnabled() {
		// 检查是否已经包含 Stripe
		hasStripe := false
		for _, method := range payMethods {
			if method["type"] == "stripe" {
				hasStripe = true
				break
			}
		}

		if !hasStripe {
			stripeMethod := map[string]string{
				"name":      "Stripe",
				"type":      "stripe",
				"color":     "#635BFF",
				"min_topup": strconv.Itoa(setting.StripeMinTopUp),
			}
			payMethods = append(payMethods, stripeMethod)
		}
	}

	// Waffo Pancake is displayed above the standard Waffo gateway.
	enableWaffoPancake := isWaffoPancakeTopUpEnabled()
	if enableWaffoPancake {
		hasWaffoPancake := false
		for _, method := range payMethods {
			if method["type"] == model.PaymentMethodWaffoPancake {
				hasWaffoPancake = true
				break
			}
		}

		if !hasWaffoPancake {
			payMethods = append(payMethods, map[string]string{
				"name":      "Waffo Pancake",
				"type":      model.PaymentMethodWaffoPancake,
				"color":     "#F97316",
				"min_topup": strconv.Itoa(setting.WaffoPancakeMinTopUp),
			})
		}
	}

	// 如果启用了 Waffo 支付，添加到支付方法列表
	enableWaffo := isWaffoTopUpEnabled()
	if enableWaffo {
		hasWaffo := false
		for _, method := range payMethods {
			if method["type"] == model.PaymentMethodWaffo {
				hasWaffo = true
				break
			}
		}

		if !hasWaffo {
			waffoMethod := map[string]string{
				"name":      "Waffo (Global Payment)",
				"type":      model.PaymentMethodWaffo,
				"color":     "#3B82F6",
				"min_topup": strconv.Itoa(setting.WaffoMinTopUp),
			}
			payMethods = append(payMethods, waffoMethod)
		}
	}

	data := gin.H{
		"enable_online_topup":              isEpayTopUpEnabled(),
		"enable_stripe_topup":              isStripeTopUpEnabled(),
		"enable_creem_topup":               isCreemTopUpEnabled(),
		"enable_waffo_topup":               enableWaffo,
		"enable_waffo_pancake_topup":       enableWaffoPancake,
		"enable_redemption":                complianceConfirmed,
		"payment_compliance_confirmed":     complianceConfirmed,
		"payment_compliance_terms_version": operation_setting.CurrentComplianceTermsVersion,
		"waffo_pay_methods": func() interface{} {
			if enableWaffo {
				return setting.GetWaffoPayMethods()
			}
			return nil
		}(),
		"creem_products":          setting.CreemProducts,
		"pay_methods":             payMethods,
		"min_topup":               operation_setting.MinTopUp,
		"stripe_min_topup":        setting.StripeMinTopUp,
		"waffo_min_topup":         setting.WaffoMinTopUp,
		"waffo_pancake_min_topup": setting.WaffoPancakeMinTopUp,
		"amount_options":          operation_setting.GetPaymentSetting().AmountOptions,
		"discount":                operation_setting.GetPaymentSetting().AmountDiscount,
		"topup_link":              common.TopUpLink,
	}
	common.ApiSuccess(c, data)
}

type EpayRequest struct {
	Amount        int64  `json:"amount"`
	PaymentMethod string `json:"payment_method"`
}

type AmountRequest struct {
	Amount int64 `json:"amount"`
}

func GetEpayClient() *epay.Client {
	if operation_setting.PayAddress == "" || operation_setting.EpayId == "" || operation_setting.EpayKey == "" {
		return nil
	}
	withUrl, err := epay.NewClient(&epay.Config{
		PartnerID: operation_setting.EpayId,
		Key:       operation_setting.EpayKey,
	}, operation_setting.PayAddress)
	if err != nil {
		return nil
	}
	return withUrl
}

func getMinTopup() int64 {
	minTopup := operation_setting.MinTopUp
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		dMinTopup := decimal.NewFromInt(int64(minTopup))
		dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
		minTopup = common.QuotaFromDecimal(dMinTopup.Mul(dQuotaPerUnit))
	}
	return int64(minTopup)
}

func getTopUpQuota(amount int64) (int, error) {
	if common.QuotaPerUnit <= 0 || math.IsNaN(common.QuotaPerUnit) || math.IsInf(common.QuotaPerUnit, 0) {
		return 0, errors.New("充值数量无效")
	}
	quota := decimal.NewFromInt(amount)
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		quotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
		quota = decimal.NewFromInt(quota.Div(quotaPerUnit).IntPart()).Mul(quotaPerUnit)
	} else {
		quota = quota.Mul(decimal.NewFromFloat(common.QuotaPerUnit))
	}
	return common.QuotaFromDecimalStrict(quota)
}

func getMaxTopUpAmount() int64 {
	if common.QuotaPerUnit <= 0 || math.IsNaN(common.QuotaPerUnit) || math.IsInf(common.QuotaPerUnit, 0) {
		return 0
	}
	quotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
	maxStoredAmount := decimal.NewFromInt(common.MaxQuota - 1).
		Div(quotaPerUnit).
		Floor()
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		return maxStoredAmount.Add(decimal.NewFromInt(1)).
			Mul(quotaPerUnit).
			Ceil().
			Sub(decimal.NewFromInt(1)).
			IntPart()
	}
	return maxStoredAmount.IntPart()
}

func validateCreditedQuota(quota decimal.Decimal) (int, error) {
	value, err := common.QuotaFromDecimalStrict(quota)
	if err != nil {
		return 0, errors.New("充值额度超出系统可表示范围")
	}
	if value <= 0 {
		return 0, errors.New("充值额度必须大于 0")
	}
	return value, nil
}

func validateTopUpQuota(amount int64) (int, error) {
	quota, err := getTopUpQuota(amount)
	if err == nil && quota > 0 {
		return quota, nil
	}
	maxAmount := getMaxTopUpAmount()
	if maxAmount > 0 && amount > maxAmount {
		return 0, fmt.Errorf("单笔充值数量不能大于 %d", maxAmount)
	}
	return 0, errors.New("充值数量无效")
}

func rejectInvalidCreditedQuota(c *gin.Context, userId int, quota decimal.Decimal) bool {
	creditedQuota, err := validateCreditedQuota(quota)
	if err == nil {
		err = model.ValidateTopUpQuotaCapacity(userId, creditedQuota)
	}
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": err.Error()})
		return true
	}
	return false
}

func rejectInvalidTopUpQuota(c *gin.Context, userId int, amount int64) bool {
	creditedQuota, err := validateTopUpQuota(amount)
	if err == nil {
		err = model.ValidateTopUpQuotaCapacity(userId, creditedQuota)
	}
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": err.Error()})
		return true
	}
	return false
}

func RequestEpay(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}
	var req EpayRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "参数错误"})
		return
	}
	if req.Amount < getMinTopup() {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %d", getMinTopup())})
		return
	}

	id := c.GetInt("id")
	group, err := model.GetUserGroup(id, true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "获取用户分组失败"})
		return
	}
	pricing, err := buildEpayOrderPricing(req.Amount, group)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("易支付 订单价格计算失败 user_id=%d amount=%d error=%q", id, req.Amount, err.Error()))
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "充值金额无效"})
		return
	}
	if rejectInvalidCreditedQuota(c, id, decimal.NewFromInt(int64(pricing.QuotaAmount))) {
		return
	}

	if !operation_setting.ContainsPayMethod(req.PaymentMethod) {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "支付方式不存在"})
		return
	}

	callBackAddress := service.GetCallbackAddress()
	returnUrl, _ := url.Parse(paymentReturnPath("/usage-logs"))
	notifyUrl, _ := url.Parse(callBackAddress + "/api/user/epay/notify")
	tradeNo := fmt.Sprintf("%s%d", common.GetRandomString(6), time.Now().Unix())
	tradeNo = fmt.Sprintf("USR%dNO%s", id, tradeNo)
	client := GetEpayClient()
	if client == nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "当前管理员未配置支付信息"})
		return
	}
	uri, params, err := client.Purchase(&epay.PurchaseArgs{
		Type:           req.PaymentMethod,
		ServiceTradeNo: tradeNo,
		Name:           fmt.Sprintf("TUC%d", req.Amount),
		Money:          pricing.PayMoney,
		Device:         epay.PC,
		NotifyUrl:      notifyUrl,
		ReturnUrl:      returnUrl,
	})
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("易支付 拉起支付失败 user_id=%d trade_no=%s payment_method=%s amount=%d error=%q", id, tradeNo, req.PaymentMethod, req.Amount, err.Error()))
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "拉起支付失败"})
		return
	}
	payMoneyDecimal, err := decimal.NewFromString(pricing.PayMoney)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("易支付 订单金额快照解析失败 user_id=%d trade_no=%s error=%q", id, tradeNo, err.Error()))
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "创建订单失败"})
		return
	}
	payMoney, _ := payMoneyDecimal.Float64()
	topUp := &model.TopUp{
		UserId:                  id,
		Amount:                  pricing.LegacyAmount,
		Money:                   payMoney,
		TradeNo:                 tradeNo,
		PaymentMethod:           req.PaymentMethod,
		PaymentProvider:         model.PaymentProviderEpay,
		ExpectedAmountMinor:     pricing.ExpectedAmountMinor,
		QuotaAmount:             pricing.QuotaAmount,
		UnitPriceSnapshot:       pricing.UnitPriceSnapshot,
		QuotaPerUnitSnapshot:    pricing.QuotaPerUnitSnapshot,
		TopUpGroupRatioSnapshot: pricing.TopUpGroupRatioSnapshot,
		AmountDiscountSnapshot:  pricing.AmountDiscountSnapshot,
		CommissionEligible:      true,
		CreateTime:              time.Now().Unix(),
		Status:                  common.TopUpStatusPending,
	}
	err = topUp.InsertEpayWithCurrencySnapshot()
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("易支付 创建充值订单失败 user_id=%d trade_no=%s payment_method=%s amount=%d error=%q", id, tradeNo, req.PaymentMethod, req.Amount, err.Error()))
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "创建订单失败"})
		return
	}
	logger.LogInfo(c.Request.Context(), fmt.Sprintf("易支付 充值订单创建成功 user_id=%d trade_no=%s payment_method=%s amount=%d expected_amount_minor=%d currency=%s", id, tradeNo, req.PaymentMethod, req.Amount, pricing.ExpectedAmountMinor, topUp.PaidCurrency))
	c.JSON(http.StatusOK, gin.H{"message": "success", "data": params, "url": uri})
}

// tradeNo lock
var orderLocks sync.Map
var createLock sync.Mutex

// refCountedMutex 带引用计数的互斥锁，确保最后一个使用者才从 map 中删除
type refCountedMutex struct {
	mu       sync.Mutex
	refCount int
}

// LockOrder 尝试对给定订单号加锁
func LockOrder(tradeNo string) {
	createLock.Lock()
	var rcm *refCountedMutex
	if v, ok := orderLocks.Load(tradeNo); ok {
		rcm = v.(*refCountedMutex)
	} else {
		rcm = &refCountedMutex{}
		orderLocks.Store(tradeNo, rcm)
	}
	rcm.refCount++
	createLock.Unlock()
	rcm.mu.Lock()
}

// UnlockOrder 释放给定订单号的锁
func UnlockOrder(tradeNo string) {
	v, ok := orderLocks.Load(tradeNo)
	if !ok {
		return
	}
	rcm := v.(*refCountedMutex)
	rcm.mu.Unlock()

	createLock.Lock()
	rcm.refCount--
	if rcm.refCount == 0 {
		orderLocks.Delete(tradeNo)
	}
	createLock.Unlock()
}

func EpayNotify(c *gin.Context) {
	if !isEpayWebhookEnabled() {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("易支付 webhook 被拒绝 reason=webhook_disabled path=%q client_ip=%s", c.Request.RequestURI, c.ClientIP()))
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	var params map[string]string

	if c.Request.Method == "POST" {
		// POST 请求：从 POST body 解析参数
		if err := c.Request.ParseForm(); err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("易支付 webhook POST 表单解析失败 path=%q client_ip=%s error=%q", c.Request.RequestURI, c.ClientIP(), err.Error()))
			_, _ = c.Writer.Write([]byte("fail"))
			return
		}
		params = lo.Reduce(lo.Keys(c.Request.PostForm), func(r map[string]string, t string, i int) map[string]string {
			r[t] = c.Request.PostForm.Get(t)
			return r
		}, map[string]string{})
	} else {
		// GET 请求：从 URL Query 解析参数
		params = lo.Reduce(lo.Keys(c.Request.URL.Query()), func(r map[string]string, t string, i int) map[string]string {
			r[t] = c.Request.URL.Query().Get(t)
			return r
		}, map[string]string{})
	}
	if len(params) == 0 {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("易支付 webhook 参数为空 path=%q client_ip=%s", c.Request.RequestURI, c.ClientIP()))
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}
	client := GetEpayClient()
	if client == nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("易支付 client 未初始化 path=%q client_ip=%s", c.Request.RequestURI, c.ClientIP()))
		_, err := c.Writer.Write([]byte("fail"))
		if err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("易支付 webhook 响应写入失败 path=%q client_ip=%s error=%q", c.Request.RequestURI, c.ClientIP(), err.Error()))
		}
		return
	}
	verifyInfo, verifyErr := client.Verify(params)
	if verifyErr != nil || verifyInfo == nil || !verifyInfo.VerifyStatus {
		if verifyErr != nil {
			logger.LogWarn(c.Request.Context(), fmt.Sprintf("易支付 webhook 验签失败 path=%q client_ip=%s error=%q", c.Request.RequestURI, c.ClientIP(), verifyErr.Error()))
		} else {
			logger.LogWarn(c.Request.Context(), fmt.Sprintf("易支付 webhook 验签失败 path=%q client_ip=%s verify_status=false", c.Request.RequestURI, c.ClientIP()))
		}
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}
	if strings.TrimSpace(params["pid"]) != strings.TrimSpace(operation_setting.EpayId) {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("易支付 webhook 商户号不匹配 trade_no=%s client_ip=%s", verifyInfo.ServiceTradeNo, c.ClientIP()))
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}
	logger.LogInfo(c.Request.Context(), fmt.Sprintf("易支付 webhook 验签成功 trade_no=%s callback_type=%s trade_status=%s client_ip=%s", verifyInfo.ServiceTradeNo, verifyInfo.Type, verifyInfo.TradeStatus, c.ClientIP()))

	if verifyInfo.TradeStatus != epay.StatusTradeSuccess {
		logger.LogInfo(c.Request.Context(), fmt.Sprintf("易支付 webhook 忽略非成功事件 trade_no=%s callback_type=%s trade_status=%s client_ip=%s", verifyInfo.ServiceTradeNo, verifyInfo.Type, verifyInfo.TradeStatus, c.ClientIP()))
		_, _ = c.Writer.Write([]byte("success"))
		return
	}
	paidAmountMinor, paidAmountErr := parseEpayPaidAmountMinor(verifyInfo.Money)
	if paidAmountErr != nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("易支付 webhook 实付金额无效 trade_no=%s client_ip=%s error=%q", verifyInfo.ServiceTradeNo, c.ClientIP(), paidAmountErr.Error()))
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}
	settlement, settlementErr := model.CompleteEpayTopUp(model.EpaySettlement{
		TradeNo:         verifyInfo.ServiceTradeNo,
		ProviderTradeNo: verifyInfo.TradeNo,
		PaidAmountMinor: paidAmountMinor,
		PaymentMethod:   verifyInfo.Type,
		CompletedAt:     common.GetTimestamp(),
	})
	if settlementErr != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("易支付 webhook 结算失败 trade_no=%s client_ip=%s error=%q", verifyInfo.ServiceTradeNo, c.ClientIP(), settlementErr.Error()))
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}
	if !settlement.AlreadyCompleted {
		logger.LogInfo(c.Request.Context(), fmt.Sprintf("易支付 充值结算成功 trade_no=%s user_id=%d topup_id=%d client_ip=%s quota_to_add=%d", verifyInfo.ServiceTradeNo, settlement.UserID, settlement.TopUpID, c.ClientIP(), settlement.QuotaAdded))
	}
	_, _ = c.Writer.Write([]byte("success"))
}

func RequestAmount(c *gin.Context) {
	var req AmountRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "参数错误"})
		return
	}

	if req.Amount < getMinTopup() {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %d", getMinTopup())})
		return
	}
	id := c.GetInt("id")
	group, err := model.GetUserGroup(id, true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "获取用户分组失败"})
		return
	}
	pricing, err := buildEpayOrderPricing(req.Amount, group)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("易支付 订单金额计算失败 user_id=%d amount=%d error=%q", id, req.Amount, err.Error()))
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "充值金额无效"})
		return
	}
	if rejectInvalidCreditedQuota(c, id, decimal.NewFromInt(int64(pricing.QuotaAmount))) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "success", "data": pricing.PayMoney})
}

func GetUserTopUps(c *gin.Context) {
	userId := c.GetInt("id")
	pageInfo := common.GetPageQuery(c)
	keyword := c.Query("keyword")

	var (
		topups []*model.TopUp
		total  int64
		err    error
	)
	if keyword != "" {
		topups, total, err = model.SearchUserTopUps(userId, keyword, pageInfo)
	} else {
		topups, total, err = model.GetUserTopUps(userId, pageInfo)
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(topups)
	common.ApiSuccess(c, pageInfo)
}

func GetUserTopUpSummary(c *gin.Context) {
	totals, err := model.GetVerifiedTopUpPaymentTotals(c.GetInt("id"), model.PaymentProviderEpay)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	defaultCurrency, err := model.GetEpayCurrencySnapshot()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"basis":             "provider_verified_payment",
		"covered_providers": []string{model.PaymentProviderEpay},
		"default_currency":  defaultCurrency,
		"totals":            totals,
	})
}

// GetAllTopUps 管理员获取全平台充值记录
func GetAllTopUps(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	keyword := c.Query("keyword")
	var startTime int64
	var endTime int64
	hasStartTime := c.Query("start_time") != ""
	hasEndTime := c.Query("end_time") != ""
	if hasStartTime != hasEndTime {
		common.ApiErrorMsg(c, "订单时间范围无效")
		return
	}
	if hasStartTime {
		var ok bool
		startTime, endTime, _, ok = revenueSummaryRange(c)
		if !ok {
			return
		}
	}

	var (
		topups []*model.TopUp
		total  int64
		err    error
	)
	if keyword != "" {
		topups, total, err = model.SearchAllTopUps(keyword, pageInfo, startTime, endTime)
	} else {
		topups, total, err = model.GetAllTopUps(pageInfo, startTime, endTime)
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(topups)
	common.ApiSuccess(c, pageInfo)
}

type AdminCompleteTopupRequest struct {
	TradeNo string `json:"trade_no"`
}

// AdminCompleteTopUp 管理员补单接口
func AdminCompleteTopUp(c *gin.Context) {
	var req AdminCompleteTopupRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.TradeNo == "" {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	// 订单级互斥，防止并发补单
	LockOrder(req.TradeNo)
	defer UnlockOrder(req.TradeNo)

	if err := model.ManualCompleteTopUp(req.TradeNo, c.ClientIP()); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}
