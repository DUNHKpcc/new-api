package router

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestSiteDocuments(t *testing.T) {
	for _, dialect := range []string{"sqlite", "mysql", "postgres"} {
		t.Run(dialect, func(t *testing.T) {
			var driver gorm.Dialector
			dbType := common.DatabaseTypeSQLite
			switch dialect {
			case "sqlite":
				driver = sqlite.Open(filepath.Join(t.TempDir(), "docs.db"))
			case "mysql":
				dsn := os.Getenv("TEST_MYSQL_DSN")
				if dsn == "" {
					t.Skip("TEST_MYSQL_DSN not configured")
				}
				driver = mysql.Open(dsn)
				dbType = common.DatabaseTypeMySQL
			case "postgres":
				dsn := os.Getenv("TEST_POSTGRES_DSN")
				if dsn == "" {
					t.Skip("TEST_POSTGRES_DSN not configured")
				}
				driver = postgres.Open(dsn)
				dbType = common.DatabaseTypePostgreSQL
			}
			db, err := gorm.Open(driver, &gorm.Config{})
			require.NoError(t, err)
			sqlDB, err := db.DB()
			require.NoError(t, err)
			oldDB, oldLog, oldType := model.DB, model.LOG_DB, common.MainDatabaseType()
			oldRedis, oldSecret := common.RedisEnabled, common.SessionSecret
			model.DB, model.LOG_DB = db, db
			common.SetMainDatabaseType(dbType)
			common.RedisEnabled = false
			common.SessionSecret = "docs-test-session-secret"
			t.Cleanup(func() {
				model.DB, model.LOG_DB = oldDB, oldLog
				common.SetMainDatabaseType(oldType)
				common.RedisEnabled = oldRedis
				common.SessionSecret = oldSecret
				_ = sqlDB.Close()
			})
			require.NoError(t, db.AutoMigrate(&model.User{}, &model.UserSession{}, &model.UserAccessToken{}, &model.Option{}, &model.AuditLog{}))
			require.NoError(t, model.EnsureLegacyAccessTokenRetireAt(time.Now().Unix()))
			// Upgrade an existing database, then repeat migration and seed import.
			require.NoError(t, db.AutoMigrate(&model.SiteDocument{}, &model.SiteDocumentImage{}))
			require.NoError(t, model.SeedSiteDocuments(db))
			var seeded model.SiteDocument
			require.NoError(t, db.First(&seeded).Error)
			seeded.Title = "Administrator edit"
			seeded.Published = false
			require.NoError(t, db.Save(&seeded).Error)
			for range 2 {
				require.NoError(t, db.AutoMigrate(&model.SiteDocument{}, &model.SiteDocumentImage{}))
				require.NoError(t, model.SeedSiteDocuments(db))
			}
			var preserved model.SiteDocument
			require.NoError(t, db.First(&preserved, "slug = ?", seeded.Slug).Error)
			assert.Equal(t, "Administrator edit", preserved.Title)
			assert.False(t, preserved.Published)
			users := []model.User{{Username: "docs-admin", Password: "unused", Role: common.RoleAdminUser, Status: common.UserStatusEnabled, AuthVersion: 1, AffCode: "docs-admin"}, {Username: "docs-reader", Password: "unused", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, AuthVersion: 1, AffCode: "docs-reader"}}
			require.NoError(t, db.Create(&users).Error)
			tokens := make([]string, 2)
			for i := range users {
				bundle, err := service.CreateLoginSession(users[i].Id, "password", "127.0.0.1", "docs-test")
				require.NoError(t, err)
				tokens[i] = bundle.AccessToken
			}
			gin.SetMode(gin.TestMode)
			engine := gin.New()
			SetApiRouter(engine)
			request := func(method, path, token, body string) *httptest.ResponseRecorder {
				req := httptest.NewRequest(method, path, strings.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				if token != "" {
					req.Header.Set("Authorization", "Bearer "+token)
				}
				response := httptest.NewRecorder()
				engine.ServeHTTP(response, req)
				return response
			}
			for _, route := range []struct{ method, path string }{{"GET", "/api/site-docs-admin"}, {"GET", "/api/site-docs-admin/example"}, {"POST", "/api/site-docs-admin"}, {"POST", "/api/site-docs-admin/images"}, {"PUT", "/api/site-docs-admin/example"}} {
				assert.Equal(t, http.StatusUnauthorized, request(route.method, route.path, "", "{}").Code)
				assert.Equal(t, http.StatusForbidden, request(route.method, route.path, tokens[1], "{}").Code)
			}
			imageData := "UklGRlAAAABXRUJQVlA4WAoAAAAQAAAAAAAAAAAAQUxQSAIAAAAALlZQOCAoAAAAcAEAnQEqAQABAAIANCWgAnQBQAAA/umt//wQX9Dn9bePDVlqIXIAAA=="
			imageBody := `{"image":"data:image/webp;base64,` + imageData + `"}`
			imageResponse := request("POST", "/api/site-docs-admin/images", tokens[0], imageBody)
			var uploaded struct {
				Success bool
				Data    struct {
					URL string `json:"url"`
				}
			}
			require.NoError(t, common.Unmarshal(imageResponse.Body.Bytes(), &uploaded))
			require.True(t, uploaded.Success, imageResponse.Body.String())
			again := request("POST", "/api/site-docs-admin/images", tokens[0], imageBody)
			assert.Contains(t, again.Body.String(), uploaded.Data.URL)
			var imageCount int64
			require.NoError(t, db.Model(&model.SiteDocumentImage{}).Count(&imageCount).Error)
			assert.EqualValues(t, 1, imageCount)
			served := request("GET", uploaded.Data.URL, "", "")
			expectedImage, err := base64.StdEncoding.DecodeString(imageData)
			require.NoError(t, err)
			assert.Equal(t, expectedImage, served.Body.Bytes())
			assert.Equal(t, "image/webp", served.Header().Get("Content-Type"))
			assert.Contains(t, served.Header().Get("Cache-Control"), "immutable")
			assert.Contains(t, request("POST", "/api/site-docs-admin/images", tokens[0], `{"image":"data:image/png;base64,aGVsbG8="}`).Body.String(), `"success":false`)
			assert.Equal(t, http.StatusNotFound, request("GET", "/api/site-doc-images/invalid", "", "").Code)

			assert.Equal(t, http.StatusNotFound, request("GET", "/api/site-docs/"+seeded.Slug, "", "").Code)
			payload := `{"slug":"example","title":"Example","category":"Guide","content":"# Body\n\nOriginal text","published":true,"sort_order":1}`
			response := request("POST", "/api/site-docs-admin", tokens[0], payload)
			var created struct {
				Success bool
				Data    model.SiteDocument
			}
			require.NoError(t, common.Unmarshal(response.Body.Bytes(), &created))
			require.True(t, created.Success, response.Body.String())
			catalog := request("GET", "/api/site-docs", "", "")
			assert.NotContains(t, catalog.Body.String(), `"content"`)
			assert.Contains(t, catalog.Body.String(), "Example")
			assert.Contains(t, request("GET", "/api/site-docs/example", "", "").Body.String(), "Original text")
			assert.Equal(t, http.StatusConflict, request("POST", "/api/site-docs-admin", tokens[0], payload).Code)
			created.Data.Content = "Changed"
			created.Data.Published = false
			body, err := common.Marshal(created.Data)
			require.NoError(t, err)
			require.Contains(t, request("PUT", "/api/site-docs-admin/example", tokens[0], string(body)).Body.String(), `"success":true`)
			assert.Equal(t, http.StatusConflict, request("PUT", "/api/site-docs-admin/example", tokens[0], string(body)).Code)
			assert.Equal(t, http.StatusNotFound, request("GET", "/api/site-docs/example", "", "").Code)
			assert.NotContains(t, request("GET", "/api/site-docs", "", "").Body.String(), `"slug":"example"`)
			assert.Contains(t, request("GET", "/api/site-docs-admin/example", tokens[0], "").Body.String(), "Changed")
			assert.Contains(t, request("POST", "/api/site-docs-admin", tokens[0], strings.Replace(payload, `"example"`, `"../invalid"`, 1)).Body.String(), `"success":false`)
		})
	}
}
