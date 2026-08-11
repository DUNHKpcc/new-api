package model

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const legacyEpayEvidenceMigrationMarker = "migration.epay_evidence_v1"

type legacyEpayEvidenceMigrationResult struct {
	SuccessBackfilled int64
	SuccessBlocked    int64
}

func ensureNoLegacyPendingEpayOrders() error {
	if !DB.Migrator().HasTable(&TopUp{}) {
		return nil
	}

	query := DB.Model(&TopUp{}).Where(
		"payment_provider = ? AND status = ?",
		PaymentProviderEpay,
		common.TopUpStatusPending,
	)
	requiredSnapshotColumns := []string{
		"expected_amount_minor",
		"quota_amount",
		"paid_currency",
		"unit_price_snapshot",
		"quota_per_unit_snapshot",
		"top_up_group_ratio_snapshot",
		"amount_discount_snapshot",
		"commission_eligible",
	}
	for _, column := range requiredSnapshotColumns {
		if !DB.Migrator().HasColumn(&TopUp{}, column) {
			var count int64
			if err := query.Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				return fmt.Errorf("%d legacy pending Epay orders must be drained before upgrade", count)
			}
			return nil
		}
	}

	var count int64
	err := query.Where(
		"expected_amount_minor IS NULL OR expected_amount_minor <= 0 OR quota_amount IS NULL OR quota_amount <= 0 OR paid_currency IS NULL OR paid_currency = ? OR unit_price_snapshot IS NULL OR unit_price_snapshot = ? OR quota_per_unit_snapshot IS NULL OR quota_per_unit_snapshot = ? OR top_up_group_ratio_snapshot IS NULL OR top_up_group_ratio_snapshot = ? OR amount_discount_snapshot IS NULL OR amount_discount_snapshot = ? OR commission_eligible IS NULL OR commission_eligible = ?",
		"",
		"",
		"",
		"",
		"",
		false,
	).Count(&count).Error
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("%d legacy pending Epay orders must be drained before upgrade", count)
	}
	return nil
}

// backfillLegacyEpayEvidence marks historical successes only when their stored
// provider evidence and order amount agree exactly.
func backfillLegacyEpayEvidence() (legacyEpayEvidenceMigrationResult, error) {
	result := legacyEpayEvidenceMigrationResult{}
	if !DB.Migrator().HasTable(&TopUp{}) || !DB.Migrator().HasTable(&Option{}) {
		return result, nil
	}

	err := DB.Transaction(func(tx *gorm.DB) error {
		var marker Option
		err := tx.Where(commonKeyCol+" = ?", legacyEpayEvidenceMigrationMarker).First(&marker).Error
		if err == nil {
			return nil
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		var successful []TopUp
		if err := tx.Where(
			"payment_provider = ? AND status = ? AND (completion_source IS NULL OR completion_source = ?) AND paid_amount_minor > 0 AND paid_currency <> ? AND provider_trade_no IS NOT NULL AND provider_trade_no <> ?",
			PaymentProviderEpay,
			common.TopUpStatusSuccess,
			"",
			"",
			"",
		).Find(&successful).Error; err != nil {
			return err
		}
		for i := range successful {
			order := &successful[i]
			if order.Money <= 0 || math.IsNaN(order.Money) || math.IsInf(order.Money, 0) ||
				strings.TrimSpace(order.PaymentMethod) == "" || len(strings.TrimSpace(order.PaidCurrency)) != 3 ||
				order.ProviderTradeNo == nil || strings.TrimSpace(*order.ProviderTradeNo) == "" {
				result.SuccessBlocked++
				continue
			}
			expectedMinor := decimal.NewFromFloat(order.Money).Round(2).Mul(decimal.NewFromInt(100))
			if !expectedMinor.IsInteger() || expectedMinor.LessThanOrEqual(decimal.Zero) ||
				expectedMinor.GreaterThan(decimal.NewFromInt(math.MaxInt64)) ||
				expectedMinor.IntPart() != order.PaidAmountMinor {
				result.SuccessBlocked++
				continue
			}
			update := tx.Model(&TopUp{}).
				Where("id = ? AND (completion_source IS NULL OR completion_source = ?)", order.Id, "").
				Update("completion_source", CompletionSourceWebhook)
			if update.Error != nil {
				return update.Error
			}
			result.SuccessBackfilled += update.RowsAffected
		}

		marker = Option{Key: legacyEpayEvidenceMigrationMarker, Value: "1"}
		return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&marker).Error
	})
	return result, err
}
