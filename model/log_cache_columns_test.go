package model

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordConsumeLogPersistsStableCacheTokenColumns(t *testing.T) {
	truncateTables(t)
	user := &User{
		Username:    "cache-column-user",
		Password:    "password",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		AuthVersion: 1,
	}
	require.NoError(t, DB.Create(user).Error)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest("POST", "/v1/messages", nil)
	context.Set("username", user.Username)
	context.Set(common.RequestIdKey, "cache-column-request")

	RecordConsumeLog(context, user.Id, RecordConsumeLogParams{
		PromptTokens:     2,
		CompletionTokens: 3,
		ModelName:        "cache-column-model",
		Quota:            11,
		Other: map[string]interface{}{
			"cache_tokens":          float64(5),
			"cache_creation_tokens": float64(7),
			"cache_write_tokens":    float64(13),
		},
	})

	var stored Log
	require.NoError(t, LOG_DB.Where("request_id = ?", "cache-column-request").First(&stored).Error)
	assert.Equal(t, 5, stored.CacheTokens)
	assert.Equal(t, 13, stored.CacheCreationTokens)
	assert.JSONEq(t,
		`{"cache_tokens":5,"cache_creation_tokens":7,"cache_write_tokens":13}`,
		stored.Other,
	)
}

func TestConsumeLogCacheTokenColumnsRejectOutOfRangeCounts(t *testing.T) {
	cacheTokens, cacheCreationTokens := consumeLogCacheTokenCounts(map[string]interface{}{
		"cache_tokens":       float64(common.MaxQuota + 1),
		"cache_write_tokens": float64(-1),
	})

	assert.Zero(t, cacheTokens)
	assert.Zero(t, cacheCreationTokens)
}
