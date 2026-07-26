package model

import (
	"errors"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	desktopUsageDayLimit         = 3660
	desktopUsageModelLimit       = 500
	desktopUsageLegacyCacheLimit = 100000
	desktopUsageActivityLimit    = 30000
	desktopUsageTaskGapSeconds   = 30 * 60
)

type DesktopUsageAggregate struct {
	RequestCount        int64 `json:"request_count"`
	Quota               int64 `json:"quota"`
	PromptTokens        int64 `json:"prompt_tokens"`
	CompletionTokens    int64 `json:"completion_tokens"`
	CacheTokens         int64 `json:"cache_tokens"`
	CacheCreationTokens int64 `json:"cache_creation_tokens"`
	StreamCount         int64 `json:"stream_count"`
}

type DesktopUsageDayAggregate struct {
	DayStart int64 `json:"day_start"`
	DesktopUsageAggregate
}

type DesktopUsageModelAggregate struct {
	ModelName string `json:"model_name"`
	DesktopUsageAggregate
}

type DesktopUsageAggregateResult struct {
	StartTimestamp     int64
	EndTimestamp       int64
	Totals             DesktopUsageAggregate
	Days               []DesktopUsageDayAggregate
	Models             []DesktopUsageModelAggregate
	DaysTruncated      bool
	ModelsTruncated    bool
	LegacyTruncated    bool
	ActivityTruncated  bool
	LongestTaskSeconds int64
}

type DesktopUsageCursor struct {
	ID        int
	CreatedAt int64
	RequestID string
}

func desktopUsageAggregateSelect(prefix string) string {
	return "COUNT(*) AS request_count, " +
		"COALESCE(SUM(" + prefix + "quota), 0) AS quota, " +
		"COALESCE(SUM(" + prefix + "prompt_tokens), 0) AS prompt_tokens, " +
		"COALESCE(SUM(" + prefix + "completion_tokens), 0) AS completion_tokens, " +
		"COALESCE(SUM(" + prefix + "cache_tokens), 0) AS cache_tokens, " +
		"COALESCE(SUM(" + prefix + "cache_creation_tokens), 0) AS cache_creation_tokens, " +
		"COALESCE(SUM(CASE WHEN " + prefix + "is_stream THEN 1 ELSE 0 END), 0) AS stream_count"
}

func desktopUsageAggregateQuery(userID int, startTimestamp, endTimestamp int64) (*gorm.DB, error) {
	if LOG_DB == nil || userID <= 0 || startTimestamp < 0 || endTimestamp < 0 {
		return nil, errors.New("invalid desktop usage aggregate parameters")
	}
	if endTimestamp == 0 {
		endTimestamp = time.Now().Unix()
	}
	if startTimestamp > endTimestamp {
		return nil, errors.New("desktop usage start timestamp is after end timestamp")
	}
	query := LOG_DB.Model(&Log{}).
		Where("user_id = ? AND type = ? AND created_at <= ?", userID, LogTypeConsume, endTimestamp)
	if startTimestamp > 0 {
		query = query.Where("created_at >= ?", startTimestamp)
	}
	return query, nil
}

func GetDesktopUsageAggregate(userID int, startTimestamp, endTimestamp int64) (*DesktopUsageAggregateResult, error) {
	if endTimestamp == 0 {
		endTimestamp = time.Now().Unix()
	}
	query, err := desktopUsageAggregateQuery(userID, startTimestamp, endTimestamp)
	if err != nil {
		return nil, err
	}
	result := &DesktopUsageAggregateResult{
		StartTimestamp: startTimestamp,
		EndTimestamp:   endTimestamp,
		Days:           []DesktopUsageDayAggregate{},
		Models:         []DesktopUsageModelAggregate{},
	}
	if err := query.Select(desktopUsageAggregateSelect("")).Scan(&result.Totals).Error; err != nil {
		return nil, err
	}

	const dayExpression = "created_at - (created_at % 86400)"
	if err := query.
		Select(dayExpression + " AS day_start, " + desktopUsageAggregateSelect("")).
		Group(dayExpression).
		Order("day_start ASC").
		Limit(desktopUsageDayLimit + 1).
		Scan(&result.Days).Error; err != nil {
		return nil, err
	}
	if len(result.Days) > desktopUsageDayLimit {
		result.Days = result.Days[:desktopUsageDayLimit]
		result.DaysTruncated = true
	}

	modelQuery, err := desktopUsageAggregateQuery(userID, startTimestamp, endTimestamp)
	if err != nil {
		return nil, err
	}
	if err := modelQuery.
		Select("model_name, " + desktopUsageAggregateSelect("")).
		Group("model_name").
		Order("quota DESC, model_name ASC").
		Limit(desktopUsageModelLimit + 1).
		Scan(&result.Models).Error; err != nil {
		return nil, err
	}
	if len(result.Models) > desktopUsageModelLimit {
		result.Models = result.Models[:desktopUsageModelLimit]
		result.ModelsTruncated = true
	}

	activityQuery, err := desktopUsageAggregateQuery(userID, startTimestamp, endTimestamp)
	if err != nil {
		return nil, err
	}
	var activity []struct {
		CreatedAt int64
	}
	if err := activityQuery.
		Select("created_at").
		Order("created_at ASC").
		Limit(desktopUsageActivityLimit + 1).
		Scan(&activity).Error; err != nil {
		return nil, err
	}
	if len(activity) > desktopUsageActivityLimit {
		activity = activity[:desktopUsageActivityLimit]
		result.ActivityTruncated = true
	}
	if len(activity) > 0 {
		taskStart := activity[0].CreatedAt
		previous := activity[0].CreatedAt
		for i := 1; i < len(activity); i++ {
			createdAt := activity[i].CreatedAt
			if createdAt-previous > desktopUsageTaskGapSeconds {
				taskStart = createdAt
			}
			if duration := createdAt - taskStart; duration > result.LongestTaskSeconds {
				result.LongestTaskSeconds = duration
			}
			previous = createdAt
		}
	}

	legacyQuery, err := desktopUsageAggregateQuery(userID, startTimestamp, endTimestamp)
	if err != nil {
		return nil, err
	}
	var legacyLogs []Log
	if err := legacyQuery.
		Select("created_at", "model_name", "cache_tokens", "cache_creation_tokens", "other").
		Where("(cache_tokens = 0 OR cache_creation_tokens = 0) AND other <> ''").
		Limit(desktopUsageLegacyCacheLimit + 1).
		Find(&legacyLogs).Error; err != nil {
		return nil, err
	}
	if len(legacyLogs) > desktopUsageLegacyCacheLimit {
		legacyLogs = legacyLogs[:desktopUsageLegacyCacheLimit]
		result.LegacyTruncated = true
	}
	daysByStart := make(map[int64]*DesktopUsageDayAggregate, len(result.Days))
	for i := range result.Days {
		daysByStart[result.Days[i].DayStart] = &result.Days[i]
	}
	modelsByName := make(map[string]*DesktopUsageModelAggregate, len(result.Models))
	for i := range result.Models {
		modelsByName[result.Models[i].ModelName] = &result.Models[i]
	}
	for i := range legacyLogs {
		other, _ := common.StrToMap(legacyLogs[i].Other)
		cacheTokens := 0
		if legacyLogs[i].CacheTokens == 0 {
			cacheTokens = positiveLogTokenCount(other["cache_tokens"])
		}
		cacheCreationTokens := 0
		if legacyLogs[i].CacheCreationTokens == 0 {
			cacheCreationTokens = positiveLogTokenCount(other["cache_write_tokens"])
			if cacheCreationTokens == 0 {
				cacheCreationTokens = positiveLogTokenCount(other["cache_creation_tokens"])
			}
		}
		if cacheTokens == 0 && cacheCreationTokens == 0 {
			continue
		}
		result.Totals.CacheTokens += int64(cacheTokens)
		result.Totals.CacheCreationTokens += int64(cacheCreationTokens)
		dayStart := legacyLogs[i].CreatedAt - (legacyLogs[i].CreatedAt % 86400)
		if day := daysByStart[dayStart]; day != nil {
			day.CacheTokens += int64(cacheTokens)
			day.CacheCreationTokens += int64(cacheCreationTokens)
		}
		if model := modelsByName[legacyLogs[i].ModelName]; model != nil {
			model.CacheTokens += int64(cacheTokens)
			model.CacheCreationTokens += int64(cacheCreationTokens)
		}
	}
	return result, nil
}

func GetDesktopUsageLogsByCursor(
	userID int,
	cursor DesktopUsageCursor,
	limit int,
	startTimestamp int64,
	endTimestamp int64,
) ([]*Log, bool, error) {
	if LOG_DB == nil || userID <= 0 || limit <= 0 || limit > 100 ||
		startTimestamp < 0 || endTimestamp < 0 || (endTimestamp > 0 && startTimestamp > endTimestamp) {
		return nil, false, errors.New("invalid desktop usage cursor parameters")
	}
	query := LOG_DB.Model(&Log{}).Where("user_id = ? AND type = ?", userID, LogTypeConsume)
	if startTimestamp > 0 {
		query = query.Where("created_at >= ?", startTimestamp)
	}
	if endTimestamp > 0 {
		query = query.Where("created_at <= ?", endTimestamp)
	}
	if common.UsingLogDatabase(common.DatabaseTypeClickHouse) {
		if cursor.CreatedAt > 0 {
			query = query.Where(
				"created_at < ? OR (created_at = ? AND request_id < ?)",
				cursor.CreatedAt,
				cursor.CreatedAt,
				cursor.RequestID,
			)
		}
		query = query.Order(clickHouseLogOrder(""))
	} else {
		if cursor.ID > 0 {
			query = query.Where("id < ?", cursor.ID)
		}
		query = query.Order("id desc")
	}
	var logs []*Log
	if err := query.Limit(limit + 1).Find(&logs).Error; err != nil {
		return nil, false, err
	}
	hasMore := len(logs) > limit
	if hasMore {
		logs = logs[:limit]
	}
	return logs, hasMore, nil
}
