package model

import (
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/console_setting"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func useFrontendOptionMigrationDB(t *testing.T) *gorm.DB {
	t.Helper()
	previousDB := DB
	previousType := common.MainDatabaseType()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Option{}))
	DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	t.Cleanup(func() {
		DB = previousDB
		common.SetMainDatabaseType(previousType)
	})
	return db
}

func requireOptionValue(t *testing.T, db *gorm.DB, key string) string {
	t.Helper()
	var option Option
	require.NoError(t, db.Where(&Option{Key: key}).First(&option).Error)
	return option.Value
}

func requireOptionMissing(t *testing.T, db *gorm.DB, key string) {
	t.Helper()
	var option Option
	assert.ErrorIs(t, db.Where(&Option{Key: key}).First(&option).Error, gorm.ErrRecordNotFound)
}

func TestMigrateRetiredFrontendOptionsMigratesValidValuesIdempotently(t *testing.T) {
	db := useFrontendOptionMigrationDB(t)
	legacy := []Option{
		{Key: retiredThemeOptionKey, Value: "classic"},
		{Key: "ApiInfo", Value: `[{"url":"https://api.example.com","route":"primary","description":"API","color":"blue"}]`},
		{Key: "Announcements", Value: `[{"content":"maintenance","publishDate":"2026-07-20T00:00:00Z","type":"warning"}]`},
		{Key: "FAQ", Value: `[{"title":"Question","content":"Answer"}]`},
		{Key: "UptimeKumaUrl", Value: "https://status.example.com"},
		{Key: "UptimeKumaSlug", Value: "status"},
	}
	require.NoError(t, db.Create(&legacy).Error)

	require.NoError(t, MigrateRetiredFrontendOptions())
	assert.Equal(t, "default", requireOptionValue(t, db, retiredThemeOptionKey))
	assert.JSONEq(t, legacy[1].Value, requireOptionValue(t, db, "console_setting.api_info"))
	assert.Equal(t, legacy[2].Value, requireOptionValue(t, db, "console_setting.announcements"))
	assert.JSONEq(t, `[{"question":"Question","answer":"Answer"}]`, requireOptionValue(t, db, "console_setting.faq"))
	assert.JSONEq(t, `[{
		"id":1,"categoryName":"old","url":"https://status.example.com","slug":"status","description":""
	}]`, requireOptionValue(t, db, "console_setting.uptime_kuma_groups"))
	for _, key := range []string{"ApiInfo", "Announcements", "FAQ", "UptimeKumaUrl", "UptimeKumaSlug"} {
		requireOptionMissing(t, db, key)
	}

	before, err := AllOption()
	require.NoError(t, err)
	require.NoError(t, MigrateRetiredFrontendOptions())
	after, err := AllOption()
	require.NoError(t, err)
	assert.ElementsMatch(t, before, after)
}

func TestMigrateRetiredFrontendOptionsStripsExistingAnnouncementImages(t *testing.T) {
	db := useFrontendOptionMigrationDB(t)
	ordinary := `[{"id":7,"content":"legacy image","publishDate":"2026-08-01T00:00:00Z","type":"warning","image":"data:image/webp;base64,AAAA"}]`
	global := `[{"id":8,"content":"global image","publishDate":"2026-08-02T00:00:00Z","type":"success","image":"data:image/webp;base64,BBBB"}]`
	require.NoError(t, db.Create(&[]Option{
		{Key: "console_setting.announcements", Value: ordinary},
		{Key: "console_setting.global_notifications", Value: global},
	}).Error)

	require.NoError(t, MigrateRetiredFrontendOptions())
	assert.JSONEq(t, `[{"id":7,"content":"legacy image","publishDate":"2026-08-01T00:00:00Z","type":"warning"}]`, requireOptionValue(t, db, "console_setting.announcements"))
	assert.JSONEq(t, global, requireOptionValue(t, db, "console_setting.global_notifications"))
}

func TestAnnouncementImagePolicyAppliesToSingleAndBulkOptionWrites(t *testing.T) {
	db := useFrontendOptionMigrationDB(t)
	previousMap := common.OptionMap
	common.OptionMap = map[string]string{}
	previousConsoleSetting := *console_setting.GetConsoleSetting()
	t.Cleanup(func() {
		common.OptionMap = previousMap
		*console_setting.GetConsoleSetting() = previousConsoleSetting
	})

	ordinary := `[{"id":11,"content":"ordinary","publishDate":"2026-08-01T00:00:00Z","type":"default","image":"data:image/webp;base64,AAAA"}]`
	require.NoError(t, UpdateOption("console_setting.announcements", ordinary))
	assert.JSONEq(t, `[{"id":11,"content":"ordinary","publishDate":"2026-08-01T00:00:00Z","type":"default"}]`, requireOptionValue(t, db, "console_setting.announcements"))

	global := `[{"id":12,"content":"global","publishDate":"2026-08-02T00:00:00Z","type":"success","image":"data:image/webp;base64,BBBB"}]`
	require.NoError(t, UpdateOptionsBulk(map[string]string{
		"console_setting.announcements":        ordinary,
		"console_setting.global_notifications": global,
	}))
	assert.JSONEq(t, `[{"id":11,"content":"ordinary","publishDate":"2026-08-01T00:00:00Z","type":"default"}]`, requireOptionValue(t, db, "console_setting.announcements"))
	assert.JSONEq(t, global, requireOptionValue(t, db, "console_setting.global_notifications"))
}

func TestLegacyConsoleListMigrationCapsAPIInfoAndFAQ(t *testing.T) {
	apiInfo := make([]map[string]any, 51)
	faq := make([]map[string]any, 51)
	for i := range apiInfo {
		apiInfo[i] = map[string]any{
			"url":         fmt.Sprintf("https://api-%d.example.com", i),
			"route":       fmt.Sprintf("route-%d", i),
			"description": "API",
			"color":       "blue",
		}
		faq[i] = map[string]any{"title": fmt.Sprintf("Question %d", i), "content": "Answer"}
	}
	apiBytes, err := common.Marshal(apiInfo)
	require.NoError(t, err)
	faqBytes, err := common.Marshal(faq)
	require.NoError(t, err)

	migratedAPI, err := transformLegacyAPIInfo(string(apiBytes))
	require.NoError(t, err)
	migratedFAQ, err := transformLegacyFAQ(string(faqBytes))
	require.NoError(t, err)
	var apiResult []map[string]any
	require.NoError(t, common.UnmarshalJsonStr(migratedAPI, &apiResult))
	var faqResult []map[string]any
	require.NoError(t, common.UnmarshalJsonStr(migratedFAQ, &faqResult))
	assert.Len(t, apiResult, 50)
	assert.Len(t, faqResult, 50)
}

func TestMigrateRetiredFrontendOptionsPreservesMalformedValuesAndContinues(t *testing.T) {
	db := useFrontendOptionMigrationDB(t)
	legacy := []Option{
		{Key: "ApiInfo", Value: `{invalid`},
		{Key: "FAQ", Value: `[{"question":"Question","answer":"Answer"}]`},
		{Key: "UptimeKumaUrl", Value: "https://status.example.com"},
	}
	require.NoError(t, db.Create(&legacy).Error)

	require.NoError(t, MigrateRetiredFrontendOptions())
	assert.Equal(t, `{invalid`, requireOptionValue(t, db, "ApiInfo"))
	requireOptionMissing(t, db, "console_setting.api_info")
	requireOptionMissing(t, db, "FAQ")
	assert.JSONEq(t, legacy[1].Value, requireOptionValue(t, db, "console_setting.faq"))
	assert.Equal(t, "https://status.example.com", requireOptionValue(t, db, "UptimeKumaUrl"))
	requireOptionMissing(t, db, "console_setting.uptime_kuma_groups")
}

func TestMigrateRetiredFrontendOptionsPreservesMixedInvalidFAQ(t *testing.T) {
	db := useFrontendOptionMigrationDB(t)
	legacyFAQ := `[{"question":"Valid question","answer":"Valid answer"},{"question":"Missing answer"}]`
	require.NoError(t, db.Create(&Option{Key: "FAQ", Value: legacyFAQ}).Error)

	require.NoError(t, MigrateRetiredFrontendOptions())
	assert.Equal(t, legacyFAQ, requireOptionValue(t, db, "FAQ"))
	requireOptionMissing(t, db, "console_setting.faq")
}

func TestMigrateRetiredFrontendOptionsKeepsAuthoritativeTargets(t *testing.T) {
	db := useFrontendOptionMigrationDB(t)
	options := []Option{
		{Key: "ApiInfo", Value: `{invalid`},
		{Key: "console_setting.api_info", Value: `[{"url":"https://new.example.com"}]`},
		{Key: "UptimeKumaUrl", Value: "https://old.example.com"},
		{Key: "UptimeKumaSlug", Value: "old"},
		{Key: "console_setting.uptime_kuma_groups", Value: `[{"url":"https://new.example.com"}]`},
	}
	require.NoError(t, db.Create(&options).Error)

	require.NoError(t, MigrateRetiredFrontendOptions())
	assert.Equal(t, options[1].Value, requireOptionValue(t, db, "console_setting.api_info"))
	assert.Equal(t, options[4].Value, requireOptionValue(t, db, "console_setting.uptime_kuma_groups"))
	for _, key := range []string{"ApiInfo", "UptimeKumaUrl", "UptimeKumaSlug"} {
		requireOptionMissing(t, db, key)
	}
}

func TestMigrateRetiredFrontendOptionsKeepsEmptyAuthoritativeTargets(t *testing.T) {
	db := useFrontendOptionMigrationDB(t)
	options := []Option{
		{Key: "ApiInfo", Value: `[{"url":"https://old.example.com"}]`},
		{Key: "console_setting.api_info", Value: ""},
		{Key: "UptimeKumaUrl", Value: "https://old.example.com"},
		{Key: "UptimeKumaSlug", Value: "old"},
		{Key: "console_setting.uptime_kuma_groups", Value: ""},
	}
	require.NoError(t, db.Create(&options).Error)

	require.NoError(t, MigrateRetiredFrontendOptions())
	assert.Empty(t, requireOptionValue(t, db, "console_setting.api_info"))
	assert.Empty(t, requireOptionValue(t, db, "console_setting.uptime_kuma_groups"))
	for _, key := range []string{"ApiInfo", "UptimeKumaUrl", "UptimeKumaSlug"} {
		requireOptionMissing(t, db, key)
	}
}

func TestRetiredThemeOptionIsPersistedButNotPublished(t *testing.T) {
	db := useFrontendOptionMigrationDB(t)
	previousMap := common.OptionMap
	t.Cleanup(func() { common.OptionMap = previousMap })
	common.OptionMap = map[string]string{}

	require.NoError(t, UpdateOption(retiredThemeOptionKey, "default"))
	assert.Equal(t, "default", requireOptionValue(t, db, retiredThemeOptionKey))
	_, published := common.OptionMap[retiredThemeOptionKey]
	assert.False(t, published)
}
