package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type TopUp struct {
	Id                      int     `json:"id"`
	UserId                  int     `json:"user_id" gorm:"index"`
	Amount                  int64   `json:"amount"`
	Money                   float64 `json:"money"`
	TradeNo                 string  `json:"trade_no" gorm:"unique;type:varchar(255);index"`
	PaymentMethod           string  `json:"payment_method" gorm:"type:varchar(50)"`
	PaymentProvider         string  `json:"payment_provider" gorm:"type:varchar(50);default:'';uniqueIndex:idx_top_ups_provider_trade,priority:1"`
	ExpectedAmountMinor     int64   `json:"-"`
	PaidAmountMinor         int64   `json:"-"`
	PaidCurrency            string  `json:"-" gorm:"type:varchar(3)"`
	ProviderTradeNo         *string `json:"-" gorm:"type:varchar(255);uniqueIndex:idx_top_ups_provider_trade,priority:2"`
	QuotaAmount             int     `json:"-"`
	UnitPriceSnapshot       string  `json:"-" gorm:"type:varchar(64)"`
	QuotaPerUnitSnapshot    string  `json:"-" gorm:"type:varchar(64)"`
	TopUpGroupRatioSnapshot string  `json:"-" gorm:"type:varchar(64)"`
	AmountDiscountSnapshot  string  `json:"-" gorm:"type:varchar(64)"`
	CommissionEligible      bool    `json:"-"`
	CompletionSource        string  `json:"-" gorm:"type:varchar(32)"`
	CreateTime              int64   `json:"create_time"`
	CompleteTime            int64   `json:"complete_time"`
	Status                  string  `json:"status"`
}

type VerifiedTopUpPaymentTotal struct {
	PaymentProvider string `json:"payment_provider"`
	Currency        string `json:"currency" gorm:"column:paid_currency"`
	AmountMinor     int64  `json:"amount_minor,string" gorm:"column:paid_amount_minor"`
}

const (
	PaymentMethodStripe       = "stripe"
	PaymentMethodCreem        = "creem"
	PaymentMethodWaffo        = "waffo"
	PaymentMethodWaffoPancake = "waffo_pancake"
	PaymentMethodBalance      = "balance"
)

const (
	CompletionSourceWebhook = "webhook"
	CompletionSourceAdmin   = "admin"
)

const (
	PaymentProviderEpay         = "epay"
	PaymentProviderStripe       = "stripe"
	PaymentProviderCreem        = "creem"
	PaymentProviderWaffo        = "waffo"
	PaymentProviderWaffoPancake = "waffo_pancake"
	PaymentProviderBalance      = "balance"
)

var (
	ErrPaymentMethodMismatch         = errors.New("payment method mismatch")
	ErrTopUpNotFound                 = errors.New("topup not found")
	ErrTopUpStatusInvalid            = errors.New("topup status invalid")
	ErrEpayProviderTradeNoRequired   = errors.New("Epay provider trade number is required")
	ErrEpayPaidAmountInvalid         = errors.New("Epay paid amount must be positive")
	ErrEpayPaidAmountMismatch        = errors.New("Epay paid amount does not match the order")
	ErrEpayOrderSnapshotInvalid      = errors.New("Epay order settlement snapshot is invalid")
	ErrEpayCompletedEvidenceMismatch = errors.New("completed Epay order evidence does not match callback")
	ErrInvalidTopUpQuota             = errors.New("invalid top-up quota")
	ErrTopUpQuotaLimitExceeded       = errors.New("top-up quota limit exceeded")
)

type EpaySettlement struct {
	TradeNo         string
	ProviderTradeNo string
	PaidAmountMinor int64
	PaymentMethod   string
	CompletedAt     int64
}

type EpaySettlementResult struct {
	TopUpID           int
	UserID            int
	QuotaAdded        int
	CommissionID      int64
	CommissionCreated bool
	AlreadyCompleted  bool
}

func validPositiveDecimalSnapshot(value string) bool {
	parsed, err := decimal.NewFromString(value)
	return err == nil && parsed.GreaterThan(decimal.Zero)
}

// CompleteEpayTopUp atomically records verified payment evidence, completes the
// order, and credits the user. Database locks and unique constraints provide
// correctness across multiple application instances.
func CompleteEpayTopUp(input EpaySettlement) (EpaySettlementResult, error) {
	input.TradeNo = strings.TrimSpace(input.TradeNo)
	input.ProviderTradeNo = strings.TrimSpace(input.ProviderTradeNo)
	input.PaymentMethod = strings.TrimSpace(input.PaymentMethod)
	if input.TradeNo == "" {
		return EpaySettlementResult{}, ErrTopUpNotFound
	}
	if input.ProviderTradeNo == "" {
		return EpaySettlementResult{}, ErrEpayProviderTradeNoRequired
	}
	if input.PaidAmountMinor <= 0 {
		return EpaySettlementResult{}, ErrEpayPaidAmountInvalid
	}
	if input.CompletedAt <= 0 {
		input.CompletedAt = common.GetTimestamp()
	}

	result := EpaySettlementResult{}
	err := DB.Transaction(func(tx *gorm.DB) error {
		topUp := TopUp{}
		if err := lockForUpdate(tx).Where("trade_no = ?", input.TradeNo).First(&topUp).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrTopUpNotFound
			}
			return err
		}
		result.TopUpID = topUp.Id
		result.UserID = topUp.UserId

		if topUp.PaymentProvider != PaymentProviderEpay {
			return ErrPaymentMethodMismatch
		}
		if input.PaymentMethod == "" || topUp.PaymentMethod == "" || input.PaymentMethod != topUp.PaymentMethod {
			return ErrPaymentMethodMismatch
		}
		if topUp.Status == common.TopUpStatusSuccess {
			if topUp.CompletionSource != CompletionSourceAdmin &&
				topUp.ProviderTradeNo != nil &&
				*topUp.ProviderTradeNo == input.ProviderTradeNo &&
				topUp.PaidAmountMinor == input.PaidAmountMinor {
				result.AlreadyCompleted = true
				return nil
			}
			return ErrEpayCompletedEvidenceMismatch
		}
		if topUp.Status != common.TopUpStatusPending {
			return ErrTopUpStatusInvalid
		}
		if topUp.ExpectedAmountMinor <= 0 ||
			topUp.QuotaAmount <= 0 ||
			len(topUp.PaidCurrency) != 3 ||
			!validPositiveDecimalSnapshot(topUp.UnitPriceSnapshot) ||
			!validPositiveDecimalSnapshot(topUp.QuotaPerUnitSnapshot) ||
			!validPositiveDecimalSnapshot(topUp.TopUpGroupRatioSnapshot) ||
			!validPositiveDecimalSnapshot(topUp.AmountDiscountSnapshot) {
			return ErrEpayOrderSnapshotInvalid
		}
		if input.PaidAmountMinor != topUp.ExpectedAmountMinor {
			return ErrEpayPaidAmountMismatch
		}
		affiliateConfig, err := affiliateCommissionConfigWithTx(tx)
		if err != nil {
			return err
		}

		providerTradeNo := input.ProviderTradeNo
		paymentMethod := topUp.PaymentMethod
		if input.PaymentMethod != "" {
			paymentMethod = input.PaymentMethod
		}
		updates := map[string]interface{}{
			"paid_amount_minor": input.PaidAmountMinor,
			"provider_trade_no": &providerTradeNo,
			"payment_method":    paymentMethod,
			"completion_source": CompletionSourceWebhook,
			"complete_time":     input.CompletedAt,
			"status":            common.TopUpStatusSuccess,
		}
		orderUpdate := tx.Model(&TopUp{}).
			Where("id = ? AND status = ?", topUp.Id, common.TopUpStatusPending).
			Updates(updates)
		if orderUpdate.Error != nil {
			return orderUpdate.Error
		}
		if orderUpdate.RowsAffected != 1 {
			return ErrTopUpStatusInvalid
		}
		if err := creditTopUpQuota(tx, topUp.UserId, topUp.QuotaAmount, nil); err != nil {
			return err
		}
		commission, created, err := CreateAffiliateCommissionForTopUpWithTx(tx, AffiliateCommissionSettlement{
			TopUp:           &topUp,
			ProviderTradeNo: input.ProviderTradeNo,
			PaidAmountMinor: input.PaidAmountMinor,
			PaidCurrency:    topUp.PaidCurrency,
			CompletedAt:     input.CompletedAt,
			Config:          affiliateConfig,
		})
		if err != nil {
			return err
		}
		if commission != nil {
			result.CommissionID = commission.Id
			result.CommissionCreated = created
		}
		if _, err := EnqueueAffiliateEventWithTx(
			tx,
			fmt.Sprintf("payment.topup_completed:%d", topUp.Id),
			topUp.UserId,
			"payment.topup_completed",
			LogTypeTopup,
			"Online top-up completed",
			map[string]interface{}{
				"paid_amount_minor": fmt.Sprintf("%d", input.PaidAmountMinor),
				"paid_currency":     topUp.PaidCurrency,
				"payment_method":    paymentMethod,
				"quota_added":       topUp.QuotaAmount,
				"topup_id":          topUp.Id,
			},
			input.CompletedAt,
		); err != nil {
			return err
		}
		result.QuotaAdded = topUp.QuotaAmount
		return nil
	})
	if err != nil {
		return EpaySettlementResult{}, err
	}
	if !result.AlreadyCompleted {
		syncCreditUserQuotaCache(result.UserID, result.QuotaAdded, "Epay topup")
	}
	return result, nil
}

func (topUp *TopUp) Insert() error {
	var err error
	err = DB.Create(topUp).Error
	return err
}

func topUpQuotaMaxCurrent(creditedQuota int) (int, error) {
	if creditedQuota <= 0 || creditedQuota >= common.MaxQuota {
		return 0, ErrInvalidTopUpQuota
	}
	return common.MaxQuota - 1 - creditedQuota, nil
}

// ValidateTopUpQuotaCapacity performs the user-facing pre-payment check. The
// settlement path repeats the same invariant with an atomic conditional
// update, because the wallet balance can change after checkout creation.
func ValidateTopUpQuotaCapacity(userId int, creditedQuota int) error {
	maxCurrentQuota, err := topUpQuotaMaxCurrent(creditedQuota)
	if err != nil {
		return err
	}

	var user User
	if err := DB.Select("quota").Where("id = ?", userId).First(&user).Error; err != nil {
		return err
	}
	if user.Quota > maxCurrentQuota {
		return ErrTopUpQuotaLimitExceeded
	}
	return nil
}

// creditTopUpQuota atomically enforces the int32 wallet ceiling while adding
// quota. Keeping the predicate and increment in one UPDATE prevents two
// concurrent callbacks from both passing a separate read/check.
func creditTopUpQuota(tx *gorm.DB, userId int, creditedQuota int, updates map[string]interface{}) error {
	maxCurrentQuota, err := topUpQuotaMaxCurrent(creditedQuota)
	if err != nil {
		return err
	}

	updateFields := make(map[string]interface{}, len(updates)+1)
	for key, value := range updates {
		updateFields[key] = value
	}
	updateFields["quota"] = gorm.Expr("quota + ?", creditedQuota)

	result := tx.Model(&User{}).
		Where("id = ? AND quota <= ?", userId, maxCurrentQuota).
		Updates(updateFields)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 1 {
		return nil
	}

	var count int64
	if err := tx.Model(&User{}).Where("id = ?", userId).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return gorm.ErrRecordNotFound
	}
	return ErrTopUpQuotaLimitExceeded
}

func (topUp *TopUp) InsertEpayWithCurrencySnapshot() error {
	if topUp == nil || topUp.PaymentProvider != PaymentProviderEpay {
		return ErrEpayOrderSnapshotInvalid
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		values, err := optionValuesWithTx(tx, EpayCurrencyOptionKey)
		if err != nil {
			return err
		}
		currency, err := epayCurrencyFromOptionValues(values)
		if err != nil {
			return err
		}
		topUp.PaidCurrency = currency
		return tx.Create(topUp).Error
	})
}

func (topUp *TopUp) Update() error {
	var err error
	err = DB.Save(topUp).Error
	return err
}

func prepareTopUpProviderTradeUniqueIndex() error {
	if !DB.Migrator().HasTable(&TopUp{}) || !DB.Migrator().HasColumn(&TopUp{}, "provider_trade_no") {
		return nil
	}
	if err := DB.Model(&TopUp{}).
		Where("provider_trade_no = ?", "").
		UpdateColumn("provider_trade_no", nil).Error; err != nil {
		return fmt.Errorf("normalize empty topup provider trade numbers: %w", err)
	}

	type duplicateProviderTrade struct {
		PaymentProvider string
		ProviderTradeNo string
		Count           int64
	}
	duplicate := duplicateProviderTrade{}
	if err := DB.Model(&TopUp{}).
		Select("payment_provider, provider_trade_no, COUNT(*) AS count").
		Where("provider_trade_no IS NOT NULL").
		Group("payment_provider, provider_trade_no").
		Having("COUNT(*) > 1").
		Limit(1).
		Scan(&duplicate).Error; err != nil {
		return fmt.Errorf("inspect duplicate topup provider trade numbers: %w", err)
	}
	if duplicate.Count > 1 {
		return fmt.Errorf(
			"duplicate provider trade number requires manual audit: provider=%q provider_trade_no=%q count=%d",
			duplicate.PaymentProvider,
			duplicate.ProviderTradeNo,
			duplicate.Count,
		)
	}
	return nil
}

func GetTopUpById(id int) *TopUp {
	var topUp *TopUp
	var err error
	err = DB.Where("id = ?", id).First(&topUp).Error
	if err != nil {
		return nil
	}
	return topUp
}

func GetTopUpByTradeNo(tradeNo string) *TopUp {
	var topUp *TopUp
	var err error
	err = DB.Where("trade_no = ?", tradeNo).First(&topUp).Error
	if err != nil {
		return nil
	}
	return topUp
}

// GetVerifiedTopUpPaymentTotals returns provider-confirmed payment totals without
// converting them to quota. Passing no providers includes every verified provider.
func verifiedTopUpPayments(db *gorm.DB) *gorm.DB {
	return db.Model(&TopUp{}).Where(
		"top_ups.status = ? AND top_ups.completion_source = ? AND top_ups.paid_amount_minor > 0 AND top_ups.paid_currency <> ? AND top_ups.provider_trade_no IS NOT NULL AND top_ups.provider_trade_no <> ?",
		common.TopUpStatusSuccess,
		CompletionSourceWebhook,
		"",
		"",
	)
}

func verifiedEpayPayments(db *gorm.DB, currency string) *gorm.DB {
	return verifiedTopUpPayments(db).Where(
		"top_ups.payment_provider = ? AND top_ups.paid_currency = ?",
		PaymentProviderEpay,
		currency,
	)
}

func GetVerifiedTopUpPaymentTotals(userId int, providers ...string) ([]VerifiedTopUpPaymentTotal, error) {
	totals := make([]VerifiedTopUpPaymentTotal, 0)
	query := verifiedTopUpPayments(DB).
		Select("payment_provider, paid_currency, SUM(paid_amount_minor) AS paid_amount_minor").
		Where("top_ups.user_id = ?", userId)
	if len(providers) > 0 {
		query = query.Where("top_ups.payment_provider IN ?", providers)
	}
	err := query.Group("payment_provider, paid_currency").
		Order("payment_provider ASC, paid_currency ASC").
		Scan(&totals).Error
	return totals, err
}

func UpdatePendingTopUpStatus(tradeNo string, expectedPaymentProvider string, targetStatus string) error {
	if tradeNo == "" {
		return errors.New("未提供支付单号")
	}

	refCol := "`trade_no`"
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		refCol = `"trade_no"`
	}

	return DB.Transaction(func(tx *gorm.DB) error {
		topUp := &TopUp{}
		if err := lockForUpdate(tx).Where(refCol+" = ?", tradeNo).First(topUp).Error; err != nil {
			return ErrTopUpNotFound
		}
		if expectedPaymentProvider != "" && topUp.PaymentProvider != expectedPaymentProvider {
			return ErrPaymentMethodMismatch
		}
		if topUp.Status != common.TopUpStatusPending {
			return ErrTopUpStatusInvalid
		}

		topUp.Status = targetStatus
		return tx.Save(topUp).Error
	})
}

func Recharge(referenceId string, customerId string, callerIp string) (err error) {
	if referenceId == "" {
		return errors.New("未提供支付单号")
	}

	var quota int
	topUp := &TopUp{}

	refCol := "`trade_no`"
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		refCol = `"trade_no"`
	}

	err = DB.Transaction(func(tx *gorm.DB) error {
		err := lockForUpdate(tx).Where(refCol+" = ?", referenceId).First(topUp).Error
		if err != nil {
			return errors.New("充值订单不存在")
		}

		if topUp.PaymentProvider != PaymentProviderStripe {
			return ErrPaymentMethodMismatch
		}

		if topUp.Status != common.TopUpStatusPending {
			return errors.New("充值订单状态错误")
		}

		topUp.CompleteTime = common.GetTimestamp()
		topUp.Status = common.TopUpStatusSuccess
		err = tx.Save(topUp).Error
		if err != nil {
			return err
		}

		quota, err = common.QuotaFromDecimalStrict(
			decimal.NewFromFloat(topUp.Money).Mul(decimal.NewFromFloat(common.QuotaPerUnit)),
		)
		if err != nil || quota <= 0 {
			return ErrInvalidTopUpQuota
		}
		return creditTopUpQuota(tx, topUp.UserId, quota, map[string]interface{}{
			"stripe_customer": customerId,
		})
	})

	if err != nil {
		common.SysError("topup failed: " + err.Error())
		return errors.New("充值失败，请稍后重试")
	}
	syncCreditUserQuotaCache(topUp.UserId, quota, "Stripe topup")

	RecordTopupLog(topUp.UserId, fmt.Sprintf("使用在线充值成功，充值金额: %v，支付金额：%d", logger.FormatQuota(quota), topUp.Amount), callerIp, topUp.PaymentMethod, PaymentMethodStripe)

	return nil
}

// topUpQueryWindowSeconds 限制充值记录查询的时间窗口（秒）。
const topUpQueryWindowSeconds int64 = 30 * 24 * 60 * 60

// topUpQueryCutoff 返回允许查询的最早 create_time（秒级 Unix 时间戳）。
func topUpQueryCutoff() int64 {
	return common.GetTimestamp() - topUpQueryWindowSeconds
}

func GetUserTopUps(userId int, pageInfo *common.PageInfo) (topups []*TopUp, total int64, err error) {
	// Start transaction
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	cutoff := topUpQueryCutoff()

	// Get total count within transaction
	err = tx.Model(&TopUp{}).Where("user_id = ? AND create_time >= ?", userId, cutoff).Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// Get paginated topups within same transaction
	err = tx.Where("user_id = ? AND create_time >= ?", userId, cutoff).Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&topups).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// Commit transaction
	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	return topups, total, nil
}

func applyTopUpCompletionRange(query *gorm.DB, startTime int64, endTime int64) *gorm.DB {
	if startTime <= 0 || endTime <= startTime {
		return query
	}
	// Pending orders do not have a completion timestamp yet. Keep them visible
	// in the month in which they were created so administrators can complete
	// them from the ranged order list.
	return query.Where(
		"(complete_time >= ? AND complete_time < ?) OR (status = ? AND create_time >= ? AND create_time < ?)",
		startTime,
		endTime,
		common.TopUpStatusPending,
		startTime,
		endTime,
	)
}

// GetAllTopUps 获取全平台的充值记录（管理员使用，可按完成时间筛选，待支付订单按创建时间回退）。
func GetAllTopUps(pageInfo *common.PageInfo, startTime int64, endTime int64) (topups []*TopUp, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	query := applyTopUpCompletionRange(tx.Model(&TopUp{}), startTime, endTime)
	if err = query.Count(&total).Error; err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = query.Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&topups).Error; err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	return topups, total, nil
}

// searchTopUpCountHardLimit 搜索充值记录时 COUNT 的安全上限，
// 防止对超大表执行无界 COUNT 触发 DoS。
const searchTopUpCountHardLimit = 10000

// SearchUserTopUps 按订单号搜索某用户的充值记录
func SearchUserTopUps(userId int, keyword string, pageInfo *common.PageInfo) (topups []*TopUp, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	query := tx.Model(&TopUp{}).Where("user_id = ? AND create_time >= ?", userId, topUpQueryCutoff())
	if keyword != "" {
		pattern, perr := sanitizeLikePattern(keyword)
		if perr != nil {
			tx.Rollback()
			return nil, 0, perr
		}
		query = query.Where("trade_no LIKE ? ESCAPE '!'", pattern)
	}

	if err = query.Limit(searchTopUpCountHardLimit).Count(&total).Error; err != nil {
		tx.Rollback()
		common.SysError("failed to count search topups: " + err.Error())
		return nil, 0, errors.New("搜索充值记录失败")
	}

	if err = query.Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&topups).Error; err != nil {
		tx.Rollback()
		common.SysError("failed to search topups: " + err.Error())
		return nil, 0, errors.New("搜索充值记录失败")
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}
	return topups, total, nil
}

// SearchAllTopUps 按订单号和完成时间搜索全平台充值记录（管理员使用，待支付订单按创建时间回退）。
func SearchAllTopUps(keyword string, pageInfo *common.PageInfo, startTime int64, endTime int64) (topups []*TopUp, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	query := applyTopUpCompletionRange(tx.Model(&TopUp{}), startTime, endTime)
	if keyword != "" {
		pattern, perr := sanitizeLikePattern(keyword)
		if perr != nil {
			tx.Rollback()
			return nil, 0, perr
		}
		query = query.Where("trade_no LIKE ? ESCAPE '!'", pattern)
	}

	if err = query.Limit(searchTopUpCountHardLimit).Count(&total).Error; err != nil {
		tx.Rollback()
		common.SysError("failed to count search topups: " + err.Error())
		return nil, 0, errors.New("搜索充值记录失败")
	}

	if err = query.Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&topups).Error; err != nil {
		tx.Rollback()
		common.SysError("failed to search topups: " + err.Error())
		return nil, 0, errors.New("搜索充值记录失败")
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}
	return topups, total, nil
}

// ManualCompleteTopUp 管理员手动完成订单并给用户充值
func ManualCompleteTopUp(tradeNo string, callerIp string) error {
	if tradeNo == "" {
		return errors.New("未提供订单号")
	}

	refCol := "`trade_no`"
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		refCol = `"trade_no"`
	}

	var userId int
	var quotaToAdd int
	var payMoney float64
	var paymentMethod string
	var alreadyCompleted bool

	err := DB.Transaction(func(tx *gorm.DB) error {
		topUp := &TopUp{}
		// 行级锁，避免并发补单
		if err := lockForUpdate(tx).Where(refCol+" = ?", tradeNo).First(topUp).Error; err != nil {
			return errors.New("充值订单不存在")
		}

		// 幂等处理：已成功直接返回
		if topUp.Status == common.TopUpStatusSuccess {
			alreadyCompleted = true
			return nil
		}

		if topUp.Status != common.TopUpStatusPending {
			return errors.New("订单状态不是待支付，无法补单")
		}

		// 计算应充值额度：
		// - Stripe 订单：Money 代表经分组倍率换算后的美元数量，直接 * QuotaPerUnit
		// - 其他订单（如易支付）：Amount 为美元数量，* QuotaPerUnit
		var quotaErr error
		if topUp.QuotaAmount > 0 {
			quotaToAdd = topUp.QuotaAmount
		} else if topUp.PaymentProvider == PaymentProviderStripe {
			quotaToAdd, quotaErr = common.QuotaFromDecimalStrict(
				decimal.NewFromFloat(topUp.Money).Mul(decimal.NewFromFloat(common.QuotaPerUnit)),
			)
		} else {
			quotaToAdd, quotaErr = common.QuotaFromDecimalStrict(
				decimal.NewFromInt(topUp.Amount).Mul(decimal.NewFromFloat(common.QuotaPerUnit)),
			)
		}
		if quotaErr != nil || quotaToAdd <= 0 {
			return ErrInvalidTopUpQuota
		}

		// 标记完成
		topUp.CompleteTime = common.GetTimestamp()
		topUp.CompletionSource = CompletionSourceAdmin
		topUp.Status = common.TopUpStatusSuccess
		if err := tx.Save(topUp).Error; err != nil {
			return err
		}

		// 增加用户额度（立即写库，保持一致性）
		if err := creditTopUpQuota(tx, topUp.UserId, quotaToAdd, nil); err != nil {
			return err
		}

		userId = topUp.UserId
		payMoney = topUp.Money
		paymentMethod = topUp.PaymentMethod
		return nil
	})

	if err != nil {
		return err
	}
	if alreadyCompleted {
		return nil
	}
	syncCreditUserQuotaCache(userId, quotaToAdd, "manual topup")

	RecordTopupLog(userId, fmt.Sprintf("管理员补单成功，充值金额: %v，支付金额：%f", logger.FormatQuota(quotaToAdd), payMoney), callerIp, paymentMethod, "admin")
	return nil
}

func RechargeCreem(referenceId string, customerEmail string, customerName string, callerIp string) (err error) {
	if referenceId == "" {
		return errors.New("未提供支付单号")
	}

	var quota int
	topUp := &TopUp{}

	refCol := "`trade_no`"
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		refCol = `"trade_no"`
	}

	err = DB.Transaction(func(tx *gorm.DB) error {
		err := lockForUpdate(tx).Where(refCol+" = ?", referenceId).First(topUp).Error
		if err != nil {
			return errors.New("充值订单不存在")
		}

		if topUp.PaymentProvider != PaymentProviderCreem {
			return ErrPaymentMethodMismatch
		}

		if topUp.Status != common.TopUpStatusPending {
			return errors.New("充值订单状态错误")
		}

		topUp.CompleteTime = common.GetTimestamp()
		topUp.Status = common.TopUpStatusSuccess
		err = tx.Save(topUp).Error
		if err != nil {
			return err
		}

		// Creem 直接使用 Amount 作为充值额度（整数）
		quota, err = common.QuotaFromDecimalStrict(decimal.NewFromInt(topUp.Amount))
		if err != nil || quota <= 0 {
			return ErrInvalidTopUpQuota
		}

		// 构建更新字段，优先使用邮箱，如果邮箱为空则使用用户名
		updateFields := map[string]interface{}{}

		// 如果有客户邮箱，尝试更新用户邮箱（仅当用户邮箱为空时）
		if customerEmail != "" {
			// 先检查用户当前邮箱是否为空
			var user User
			err = tx.Where("id = ?", topUp.UserId).First(&user).Error
			if err != nil {
				return err
			}

			// 如果用户邮箱为空，则更新为支付时使用的邮箱
			if user.Email == "" {
				updateFields["email"] = customerEmail
			}
		}

		return creditTopUpQuota(tx, topUp.UserId, quota, updateFields)
	})

	if err != nil {
		common.SysError("creem topup failed: " + err.Error())
		return errors.New("充值失败，请稍后重试")
	}
	syncCreditUserQuotaCache(topUp.UserId, quota, "Creem topup")

	RecordTopupLog(topUp.UserId, fmt.Sprintf("使用Creem充值成功，充值额度: %v，支付金额：%.2f", quota, topUp.Money), callerIp, topUp.PaymentMethod, PaymentMethodCreem)

	return nil
}

func RechargeWaffo(tradeNo string, callerIp string) (err error) {
	if tradeNo == "" {
		return errors.New("未提供支付单号")
	}

	var quotaToAdd int
	topUp := &TopUp{}

	refCol := "`trade_no`"
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		refCol = `"trade_no"`
	}

	err = DB.Transaction(func(tx *gorm.DB) error {
		err := lockForUpdate(tx).Where(refCol+" = ?", tradeNo).First(topUp).Error
		if err != nil {
			return errors.New("充值订单不存在")
		}

		if topUp.PaymentProvider != PaymentProviderWaffo {
			return ErrPaymentMethodMismatch
		}

		if topUp.Status == common.TopUpStatusSuccess {
			return nil // 幂等：已成功直接返回
		}

		if topUp.Status != common.TopUpStatusPending {
			return errors.New("充值订单状态错误")
		}

		quotaToAdd, err = common.QuotaFromDecimalStrict(
			decimal.NewFromInt(topUp.Amount).Mul(decimal.NewFromFloat(common.QuotaPerUnit)),
		)
		if err != nil || quotaToAdd <= 0 {
			return ErrInvalidTopUpQuota
		}

		topUp.CompleteTime = common.GetTimestamp()
		topUp.Status = common.TopUpStatusSuccess
		if err := tx.Save(topUp).Error; err != nil {
			return err
		}

		return creditTopUpQuota(tx, topUp.UserId, quotaToAdd, nil)
	})

	if err != nil {
		common.SysError("waffo topup failed: " + err.Error())
		return errors.New("充值失败，请稍后重试")
	}
	syncCreditUserQuotaCache(topUp.UserId, quotaToAdd, "Waffo topup")

	if quotaToAdd > 0 {
		RecordTopupLog(topUp.UserId, fmt.Sprintf("Waffo充值成功，充值额度: %v，支付金额: %.2f", logger.FormatQuota(quotaToAdd), topUp.Money), callerIp, topUp.PaymentMethod, PaymentMethodWaffo)
	}

	return nil
}

func RechargeWaffoPancake(tradeNo string) (err error) {
	if tradeNo == "" {
		return errors.New("未提供支付单号")
	}

	var quotaToAdd int
	topUp := &TopUp{}

	refCol := "`trade_no`"
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		refCol = `"trade_no"`
	}

	err = DB.Transaction(func(tx *gorm.DB) error {
		err := lockForUpdate(tx).Where(refCol+" = ?", tradeNo).First(topUp).Error
		if err != nil {
			return errors.New("充值订单不存在")
		}

		if topUp.PaymentProvider != PaymentProviderWaffoPancake {
			return ErrPaymentMethodMismatch
		}

		if topUp.Status == common.TopUpStatusSuccess {
			return nil
		}

		if topUp.Status != common.TopUpStatusPending {
			return errors.New("充值订单状态错误")
		}

		quotaToAdd, err = common.QuotaFromDecimalStrict(
			decimal.NewFromInt(topUp.Amount).Mul(decimal.NewFromFloat(common.QuotaPerUnit)),
		)
		if err != nil || quotaToAdd <= 0 {
			return ErrInvalidTopUpQuota
		}

		topUp.CompleteTime = common.GetTimestamp()
		topUp.Status = common.TopUpStatusSuccess
		if err := tx.Save(topUp).Error; err != nil {
			return err
		}

		return creditTopUpQuota(tx, topUp.UserId, quotaToAdd, nil)
	})

	if err != nil {
		common.SysError("waffo pancake topup failed: " + err.Error())
		return errors.New("充值失败，请稍后重试")
	}
	syncCreditUserQuotaCache(topUp.UserId, quotaToAdd, "Waffo Pancake topup")

	if quotaToAdd > 0 {
		RecordLog(topUp.UserId, LogTypeTopup, fmt.Sprintf("Waffo Pancake充值成功，充值额度: %v，支付金额: %.2f", logger.FormatQuota(quotaToAdd), topUp.Money))
	}

	return nil
}
