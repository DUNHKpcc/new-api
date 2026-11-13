package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// One current assessment per purchaser, independent of their number of orders.
type GuardianAccountStatus struct {
	UserID        int   `json:"user_id" gorm:"primaryKey;autoIncrement:false"`
	Banned        bool  `json:"banned"`
	PhoneRequired bool  `json:"phone_required"`
	KYCRequired   bool  `json:"kyc_required"`
	UpdatedAt     int64 `json:"updated_at"`
	UpdatedBy     int   `json:"-"`
}

type AdminGuardianAccountStatus struct {
	GuardianAccountStatus
	Username string `json:"username"`
}

type GuardianAccountStatistics struct {
	HealthyUsers       int64 `json:"healthy_users"`
	AllIssuesUsers     int64 `json:"all_issues_users"`
	PurchasedUsers     int64 `json:"purchased_users"`
	RecordedUsers      int64 `json:"recorded_users"`
	BannedUsers        int64 `json:"banned_users"`
	PhoneRequiredUsers int64 `json:"phone_required_users"`
	KYCRequiredUsers   int64 `json:"kyc_required_users"`
	UpdatedAt          int64 `json:"updated_at"`
}

func guardianPurchasers(db *gorm.DB) *gorm.DB {
	return db.Model(&GuardianServiceOrder{}).Select("DISTINCT user_id").Where("status IN ?", []string{"paid", "scheduled", "in_progress", "delivered"})
}

func GetGuardianAccountStatistics() (GuardianAccountStatistics, error) {
	var result GuardianAccountStatistics
	err := DB.Table("(?) AS buyers", guardianPurchasers(DB)).
		Joins("LEFT JOIN guardian_account_statuses AS assessments ON assessments.user_id = buyers.user_id").
		Select(`COUNT(*) AS purchased_users, COUNT(assessments.user_id) AS recorded_users,
   COALESCE(SUM(CASE WHEN assessments.banned = ? THEN 1 ELSE 0 END), 0) AS banned_users,
   COALESCE(SUM(CASE WHEN assessments.phone_required = ? THEN 1 ELSE 0 END), 0) AS phone_required_users,
   COALESCE(SUM(CASE WHEN assessments.kyc_required = ? THEN 1 ELSE 0 END), 0) AS kyc_required_users,
   COALESCE(SUM(CASE WHEN assessments.banned = ? AND assessments.phone_required = ? AND assessments.kyc_required = ? THEN 1 ELSE 0 END), 0) AS healthy_users,
   COALESCE(SUM(CASE WHEN assessments.banned = ? AND assessments.phone_required = ? AND assessments.kyc_required = ? THEN 1 ELSE 0 END), 0) AS all_issues_users,
   COALESCE(MAX(assessments.updated_at), 0) AS updated_at`, true, true, true, false, false, false, true, true, true).Scan(&result).Error
	if err != nil {
		return result, err
	}
	var baseSales int64
	if err := DB.Model(&GuardianService{}).Select("COALESCE(SUM(base_sales), 0)").Scan(&baseSales).Error; err != nil {
		return result, err
	}
	result.PurchasedUsers += baseSales
	result.RecordedUsers += baseSales
	result.HealthyUsers += baseSales
	return result, nil
}

func ListGuardianAccountStatuses(page, size int, keyword string) ([]AdminGuardianAccountStatus, int64, error) {
	query := DB.Table("users").Where("users.id IN (?)", guardianPurchasers(DB))
	if keyword != "" {
		query = query.Where("users.username LIKE ?", "%"+keyword+"%")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	rows := []AdminGuardianAccountStatus{}
	err := query.Joins("LEFT JOIN guardian_account_statuses AS assessments ON assessments.user_id = users.id").
		Select("users.id AS user_id, users.username, assessments.banned, assessments.phone_required, assessments.kyc_required, assessments.updated_at").
		Order("users.id DESC").Offset((page - 1) * size).Limit(size).Scan(&rows).Error
	return rows, total, err
}

func SaveGuardianAccountStatus(status *GuardianAccountStatus) error {
	var count int64
	if err := guardianPurchasers(DB).Where("user_id = ?", status.UserID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return errors.New("只能登记有有效已付款订单的 Guardian 用户")
	}
	status.UpdatedAt = time.Now().Unix()
	return DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"banned", "phone_required", "kyc_required", "updated_at", "updated_by"}),
	}).Create(status).Error
}
