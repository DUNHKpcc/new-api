package model

import (
	"errors"
	"time"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// One current assessment per purchaser, independent of their number of orders.
type GuardianAccountStatus struct {
	HistoricalCategory string `json:"historical_category" gorm:"type:varchar(32)"`
	UserID             int    `json:"user_id" gorm:"primaryKey;autoIncrement:false"`
	Banned             bool   `json:"banned"`
	PhoneRequired      bool   `json:"phone_required"`
	KYCRequired        bool   `json:"kyc_required"`
	UpdatedAt          int64  `json:"updated_at"`
	UpdatedBy          int    `json:"-"`
}

type AdminGuardianAccountStatus struct {
	GuardianAccountStatus
	Username string `json:"username"`
}

type GuardianAccountStatistics struct {
	HistoricalUsers    int64 `json:"historical_users"`
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
	history, err := GetGuardianHistoricalAccounts()
	if err != nil {
		return result, err
	}
	result.HistoricalUsers = history.TotalUsers
	counts := history.Counts()
	links, err := guardianHistoricalLinkCounts(DB.Where("user_id IN (?)", guardianPurchasers(DB)))
	if err != nil {
		return result, err
	}
	for category, count := range links {
		counts[category] -= count
	}
	for category, count := range counts {
		result.PurchasedUsers += count
		result.RecordedUsers += count
		switch category {
		case "healthy":
			result.HealthyUsers += count
		case "all_three":
			result.AllIssuesUsers += count
		}
		switch category {
		case "banned_only", "banned_kyc", "banned_phone", "all_three":
			result.BannedUsers += count
		}
		switch category {
		case "kyc_only", "banned_kyc", "kyc_phone", "all_three":
			result.KYCRequiredUsers += count
		}
		switch category {
		case "phone_only", "banned_phone", "kyc_phone", "all_three":
			result.PhoneRequiredUsers += count
		}
	}
	return result, nil
}

func ListGuardianAccountStatuses(page, size int, keyword string) ([]AdminGuardianAccountStatus, int64, error) {
	query := DB.Table("users").Where("users.id IN (?) OR users.id IN (?)", guardianPurchasers(DB), DB.Model(&GuardianAccountStatus{}).Select("user_id").Where("historical_category <> ?", ""))
	if keyword != "" {
		query = query.Where("users.username LIKE ?", "%"+keyword+"%")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	rows := []AdminGuardianAccountStatus{}
	err := query.Joins("LEFT JOIN guardian_account_statuses AS assessments ON assessments.user_id = users.id").
		Select("users.id AS user_id, users.username, assessments.banned, assessments.phone_required, assessments.kyc_required, assessments.updated_at, assessments.historical_category").
		Order("users.id DESC").Offset((page - 1) * size).Limit(size).Scan(&rows).Error
	return rows, total, err
}

func SaveGuardianAccountStatus(status *GuardianAccountStatus) error {
	var count int64
	if err := guardianPurchasers(DB).Where("user_id = ?", status.UserID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		var linked int64
		if err := DB.Model(&GuardianAccountStatus{}).Where("user_id = ? AND historical_category <> ?", status.UserID, "").Count(&linked).Error; err != nil {
			return err
		}
		if linked == 0 {
			return errors.New("只能登记有有效已付款订单或历史关联的 Guardian 用户")
		}
	}
	if status.HistoricalCategory != "" {
		history, err := GetGuardianHistoricalAccounts()
		if err != nil {
			return err
		}
		capacity, valid := history.Counts()[status.HistoricalCategory]
		if !valid {
			return errors.New("历史状态分组无效")
		}
		links, err := guardianHistoricalLinkCounts(DB.Where("user_id <> ?", status.UserID))
		if err != nil {
			return err
		}
		if links[status.HistoricalCategory] >= capacity {
			return errors.New("此历史分组没有可关联的用户名额")
		}
	}
	status.UpdatedAt = time.Now().Unix()
	return DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"banned", "phone_required", "kyc_required", "historical_category", "updated_at", "updated_by"}),
	}).Create(status).Error
}

const guardianHistoricalAccountsKey = "GuardianHistoricalAccounts"

type GuardianHistoricalAccounts struct {
	TotalUsers  int64 `json:"total_users"`
	BannedOnly  int64 `json:"banned_only"`
	KYCOnly     int64 `json:"kyc_only"`
	PhoneOnly   int64 `json:"phone_only"`
	BannedKYC   int64 `json:"banned_kyc"`
	BannedPhone int64 `json:"banned_phone"`
	KYCPhone    int64 `json:"kyc_phone"`
	AllThree    int64 `json:"all_three"`
}

func (h GuardianHistoricalAccounts) IssueUsers() int64 {
	return h.BannedOnly + h.KYCOnly + h.PhoneOnly + h.BannedKYC + h.BannedPhone + h.KYCPhone + h.AllThree
}

func (h GuardianHistoricalAccounts) Validate() error {
	for _, count := range []int64{h.TotalUsers, h.BannedOnly, h.KYCOnly, h.PhoneOnly, h.BannedKYC, h.BannedPhone, h.KYCPhone, h.AllThree} {
		if count < 0 || count > 1000000000 {
			return errors.New("人数必须为 0 到 1000000000 之间的整数")
		}
	}
	if h.IssueUsers() > h.TotalUsers {
		return errors.New("异常人数合计不能超过基础用户数")
	}
	baseSales, err := GuardianBaseSales()
	if err != nil {
		return err
	}
	if h.TotalUsers > baseSales {
		return errors.New("基础用户数不能超过基础销量，请先调整服务基础销量")
	}
	links, err := guardianHistoricalLinkCounts(DB)
	if err != nil {
		return err
	}
	counts := h.Counts()
	for category, count := range links {
		if counts[category] < count {
			return errors.New("历史分组人数不能少于已关联的用户数，请先调整用户关联")
		}
	}
	return nil
}

func GetGuardianHistoricalAccounts() (GuardianHistoricalAccounts, error) {
	var history GuardianHistoricalAccounts
	var option Option
	err := DB.Where(&Option{Key: guardianHistoricalAccountsKey}).Take(&option).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return history, nil
	}
	if err != nil {
		return history, err
	}
	err = common.UnmarshalJsonStr(option.Value, &history)
	return history, err
}

func SaveGuardianHistoricalAccounts(history GuardianHistoricalAccounts) error {
	data, err := common.Marshal(history)
	if err != nil {
		return err
	}
	return UpdateOption(guardianHistoricalAccountsKey, string(data))
}

func GuardianBaseSales() (int64, error) {
	var total int64
	err := DB.Model(&GuardianService{}).Select("COALESCE(SUM(base_sales), 0)").Scan(&total).Error
	return total, err
}

func ValidateGuardianBaseSalesChange(previous, next int64) error {
	if next >= previous {
		return nil
	}
	total, err := GuardianBaseSales()
	if err != nil {
		return err
	}
	history, err := GetGuardianHistoricalAccounts()
	if err != nil {
		return err
	}
	if total-previous+next < history.TotalUsers {
		return errors.New("基础销量不能少于基础用户数，请先调整历史用户配置")
	}
	return nil
}

func (h GuardianHistoricalAccounts) Counts() map[string]int64 {
	return map[string]int64{
		"healthy": h.TotalUsers - h.IssueUsers(), "banned_only": h.BannedOnly, "kyc_only": h.KYCOnly,
		"phone_only": h.PhoneOnly, "banned_kyc": h.BannedKYC, "banned_phone": h.BannedPhone,
		"kyc_phone": h.KYCPhone, "all_three": h.AllThree,
	}
}

func guardianHistoricalLinkCounts(query *gorm.DB) (map[string]int64, error) {
	var rows []struct {
		HistoricalCategory string
		Count              int64
	}
	err := query.Model(&GuardianAccountStatus{}).Where("historical_category <> ?", "").Select("historical_category, COUNT(*) AS count").Group("historical_category").Scan(&rows).Error
	counts := make(map[string]int64, len(rows))
	for _, row := range rows {
		counts[row.HistoricalCategory] = row.Count
	}
	return counts, err
}
