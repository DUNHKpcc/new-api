package model

import (
	"errors"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

type PlatformRevenueTotal struct {
	Currency      string `json:"currency"`
	AmountMinor   int64  `json:"amount_minor,string"`
	Count         int64  `json:"count"`
	VerifiedCount int64  `json:"verified_count"`
}

type PlatformRevenueMethodTotal struct {
	PaymentMethod string `json:"payment_method"`
	Currency      string `json:"currency"`
	AmountMinor   int64  `json:"amount_minor,string"`
	Count         int64  `json:"count"`
}

type PlatformRevenueTimelinePoint struct {
	Date          string `json:"date"`
	PaymentMethod string `json:"payment_method"`
	Currency      string `json:"currency"`
	AmountMinor   int64  `json:"amount_minor,string"`
	Count         int64  `json:"count"`
}

type PlatformRevenueSummary struct {
	Totals             []PlatformRevenueTotal         `json:"totals"`
	ByPaymentMethod    []PlatformRevenueMethodTotal   `json:"by_payment_method"`
	Timeline           []PlatformRevenueTimelinePoint `json:"timeline"`
	DefaultCurrency    string                         `json:"default_currency"`
	StartTime          int64                          `json:"start_time"`
	EndTime            int64                          `json:"end_time"`
	Timezone           int                            `json:"timezone_offset"`
	NormalizationBasis string                         `json:"normalization_basis"`
}

func platformRevenueAmount(topUp *TopUp, fallbackCurrency string) (int64, string, bool, error) {
	if topUp.PaidAmountMinor > 0 && len(topUp.PaidCurrency) == 3 {
		return topUp.PaidAmountMinor, strings.ToUpper(topUp.PaidCurrency), true, nil
	}
	if math.IsNaN(topUp.Money) || math.IsInf(topUp.Money, 0) {
		return 0, "", false, errors.New("platform revenue amount is not finite")
	}
	amount := decimal.NewFromFloat(topUp.Money).Mul(decimal.NewFromInt(100)).Round(0)
	if amount.IsNegative() || amount.GreaterThan(decimal.NewFromInt(math.MaxInt64)) {
		return 0, "", false, errors.New("platform revenue amount exceeds supported range")
	}
	return amount.IntPart(), fallbackCurrency, false, nil
}

func GetPlatformRevenueSummary(startTime int64, endTime int64, timezoneOffset int) (*PlatformRevenueSummary, error) {
	defaultCurrency, err := GetEpayCurrencySnapshot()
	if err != nil {
		return nil, err
	}
	defaultCurrency = strings.ToUpper(defaultCurrency)
	var topUps []*TopUp
	err = DB.Where("status = ? AND complete_time >= ? AND complete_time < ?", "success", startTime, endTime).
		Order("complete_time asc, id asc").Find(&topUps).Error
	if err != nil {
		return nil, err
	}

	totalMap := map[string]*PlatformRevenueTotal{}
	methodMap := map[string]*PlatformRevenueMethodTotal{}
	timelineMap := map[string]*PlatformRevenueTimelinePoint{}
	location := time.FixedZone("admin", timezoneOffset*60)
	for _, topUp := range topUps {
		amountMinor, currency, verified, amountErr := platformRevenueAmount(topUp, defaultCurrency)
		if amountErr != nil {
			return nil, amountErr
		}
		method := strings.TrimSpace(topUp.PaymentMethod)
		if method == "" {
			method = "unknown"
		}
		date := time.Unix(topUp.CompleteTime, 0).In(location).Format("2006-01-02")
		methodKey := strings.Join([]string{method, currency}, "\x00")
		timelineKey := strings.Join([]string{date, method, currency}, "\x00")
		if totalMap[currency] == nil {
			totalMap[currency] = &PlatformRevenueTotal{Currency: currency}
		}
		if methodMap[methodKey] == nil {
			methodMap[methodKey] = &PlatformRevenueMethodTotal{PaymentMethod: method, Currency: currency}
		}
		if timelineMap[timelineKey] == nil {
			timelineMap[timelineKey] = &PlatformRevenueTimelinePoint{Date: date, PaymentMethod: method, Currency: currency}
		}
		if totalMap[currency].AmountMinor > math.MaxInt64-amountMinor ||
			methodMap[methodKey].AmountMinor > math.MaxInt64-amountMinor ||
			timelineMap[timelineKey].AmountMinor > math.MaxInt64-amountMinor {
			return nil, errors.New("platform revenue summary exceeds supported range")
		}
		totalMap[currency].AmountMinor += amountMinor
		totalMap[currency].Count++
		if verified {
			totalMap[currency].VerifiedCount++
		}
		methodMap[methodKey].AmountMinor += amountMinor
		methodMap[methodKey].Count++
		timelineMap[timelineKey].AmountMinor += amountMinor
		timelineMap[timelineKey].Count++
	}

	summary := &PlatformRevenueSummary{
		Totals:             make([]PlatformRevenueTotal, 0, len(totalMap)),
		ByPaymentMethod:    make([]PlatformRevenueMethodTotal, 0, len(methodMap)),
		Timeline:           make([]PlatformRevenueTimelinePoint, 0, len(timelineMap)),
		DefaultCurrency:    defaultCurrency,
		StartTime:          startTime,
		EndTime:            endTime,
		Timezone:           timezoneOffset,
		NormalizationBasis: "verified_settlement_or_recorded_order_amount",
	}
	for _, total := range totalMap {
		summary.Totals = append(summary.Totals, *total)
	}
	for _, total := range methodMap {
		summary.ByPaymentMethod = append(summary.ByPaymentMethod, *total)
	}
	for _, point := range timelineMap {
		summary.Timeline = append(summary.Timeline, *point)
	}
	sort.Slice(summary.Totals, func(i, j int) bool { return summary.Totals[i].Currency < summary.Totals[j].Currency })
	sort.Slice(summary.ByPaymentMethod, func(i, j int) bool {
		if summary.ByPaymentMethod[i].Currency == summary.ByPaymentMethod[j].Currency {
			return summary.ByPaymentMethod[i].PaymentMethod < summary.ByPaymentMethod[j].PaymentMethod
		}
		return summary.ByPaymentMethod[i].Currency < summary.ByPaymentMethod[j].Currency
	})
	sort.Slice(summary.Timeline, func(i, j int) bool {
		if summary.Timeline[i].Date == summary.Timeline[j].Date {
			return summary.Timeline[i].PaymentMethod < summary.Timeline[j].PaymentMethod
		}
		return summary.Timeline[i].Date < summary.Timeline[j].Date
	})
	return summary, nil
}
