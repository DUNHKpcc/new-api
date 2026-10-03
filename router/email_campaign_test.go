package router

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/http/httptest"
	"net/mail"
	"net/textproto"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// This fixture only binds loopback and never authenticates against a real SMTP
// server. rcptReply models a safe rejection; dropAck models uncertain delivery.
func campaignSMTPFixture(t *testing.T, rcptReply string, dropAck bool) <-chan string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })
	common.SMTPServer = "127.0.0.1"
	common.SMTPPort = listener.Addr().(*net.TCPAddr).Port
	messages := make(chan string, 1)
	go func() {
		defer close(messages)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
		reader := textproto.NewReader(bufio.NewReader(conn))
		_, _ = io.WriteString(conn, "220 localhost ESMTP\r\n")
		for {
			line, err := reader.ReadLine()
			if err != nil {
				return
			}
			switch {
			case strings.HasPrefix(line, "EHLO"), strings.HasPrefix(line, "HELO"), strings.HasPrefix(line, "MAIL FROM:"):
				_, _ = io.WriteString(conn, "250 localhost\r\n")
			case strings.HasPrefix(line, "RCPT TO:"):
				_, _ = io.WriteString(conn, rcptReply+"\r\n")
			case line == "DATA":
				_, _ = io.WriteString(conn, "354 send message\r\n")
				data, err := reader.ReadDotBytes()
				if err != nil {
					return
				}
				messages <- string(data)
				if dropAck {
					return
				}
				_, _ = io.WriteString(conn, "250 accepted\r\n")
			case line == "QUIT":
				_, _ = io.WriteString(conn, "221 bye\r\n")
				return
			default:
				_, _ = io.WriteString(conn, "250 ok\r\n")
			}
		}
	}()
	return messages
}

func TestEmailCampaignAPIAndDispatch(t *testing.T) {
	previousDB, previousLogDB, previousType := model.DB, model.LOG_DB, common.MainDatabaseType()
	previousRedis, previousSecret := common.RedisEnabled, common.SessionSecret
	previousCriticalLimit := common.CriticalRateLimitEnable
	previousSMTPServer, previousSMTPPort := common.SMTPServer, common.SMTPPort
	previousSMTPAccount, previousSMTPToken, previousSMTPFrom, previousSMTPSSL := common.SMTPAccount, common.SMTPToken, common.SMTPFrom, common.SMTPSSLEnabled
	previousAddress := system_setting.ServerAddress
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "mail.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.UserSession{}, &model.UserAccessToken{}, &model.Option{}, &model.AuditLog{}, &model.SystemTask{}, &model.EmailCampaign{}, &model.EmailDelivery{}, &model.EmailSubscription{}, &model.EmailDispatchConfig{}))
	model.DB, model.LOG_DB = db, db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	common.RedisEnabled, common.SessionSecret = false, "campaign-integration-secret"
	common.CriticalRateLimitEnable = true
	common.SMTPAccount, common.SMTPToken, common.SMTPFrom, common.SMTPSSLEnabled = "", "", "sender@example.test", false
	system_setting.ServerAddress = "https://mail.example.test"
	require.NoError(t, model.EnsureLegacyAccessTokenRetireAt(time.Now().Unix()))
	t.Cleanup(func() {
		model.DB, model.LOG_DB = previousDB, previousLogDB
		common.SetMainDatabaseType(previousType)
		common.RedisEnabled, common.SessionSecret = previousRedis, previousSecret
		common.CriticalRateLimitEnable = previousCriticalLimit
		common.SMTPServer, common.SMTPPort = previousSMTPServer, previousSMTPPort
		common.SMTPAccount, common.SMTPToken, common.SMTPFrom, common.SMTPSSLEnabled = previousSMTPAccount, previousSMTPToken, previousSMTPFrom, previousSMTPSSL
		system_setting.ServerAddress = previousAddress
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
	})
	users := []model.User{
		{Username: "campaign-root", Password: "unused", Role: common.RoleRootUser, Status: common.UserStatusEnabled, Group: "root", AuthVersion: 1, AffCode: "mail-root"},
		{Username: "campaign-user", Password: "unused", Email: "reader@example.test", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", AuthVersion: 1, AffCode: "mail-user"},
		{Username: "campaign-admin", Password: "unused", Role: common.RoleAdminUser, Status: common.UserStatusEnabled, Group: "admin", AuthVersion: 1, AffCode: "mail-admin"},
	}
	require.NoError(t, db.Create(&users).Error)
	tokens := make([]string, len(users))
	for i := range users {
		bundle, err := service.CreateLoginSession(users[i].Id, "password", "127.0.0.1", "campaign-test")
		require.NoError(t, err)
		tokens[i] = bundle.AccessToken
	}
	suffix, err := common.GenerateRandomCharsKey(43)
	require.NoError(t, err)
	pat := model.AccessTokenPrefix + suffix
	grant := &model.UserAccessToken{Name: "root PAT", TokenHash: model.AccessTokenFingerprint(pat), TokenHint: model.AccessTokenHint(pat)}
	require.NoError(t, grant.SetScopes([]string{"ops:read", "ops:write"}))
	require.NoError(t, model.CreateUserAccessToken(users[0].Id, grant, service.AccessTokenMaxPerUser))
	gin.SetMode(gin.TestMode)
	previousWriter := gin.DefaultWriter
	var accessLog bytes.Buffer
	gin.DefaultWriter = &accessLog
	t.Cleanup(func() { gin.DefaultWriter = previousWriter })
	engine := gin.New()
	middleware.SetUpLogger(engine)
	SetApiRouter(engine)
	request := func(method, path, token, contentType, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.RemoteAddr = "198.51.100.23:12345"
		req.Header.Set("Content-Type", contentType)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		result := httptest.NewRecorder()
		engine.ServeHTTP(result, req)
		return result
	}
	for _, endpoint := range []struct{ method, path string }{{"GET", "/api/email-campaign/"}, {"POST", "/api/email-campaign/"}, {"POST", "/api/email-campaign/1/queue"}, {"PUT", "/api/email-campaign/config"}, {"GET", "/api/email-campaign/1/deliveries"}} {
		for _, credential := range []struct {
			token  string
			status int
		}{{"", 401}, {tokens[1], 403}, {tokens[2], 403}, {pat, 403}} {
			response := request(endpoint.method, endpoint.path, credential.token, "application/json", `{}`)
			assert.Equal(t, credential.status, response.Code, "%s %s: %s", endpoint.method, endpoint.path, response.Body.String())
		}
	}
	response := request("PUT", "/api/user/self/email-subscriptions", tokens[1], "application/json", `{"promotion":true,"platform":true}`)
	require.Equal(t, 200, response.Code, response.Body.String())
	response = request("PUT", "/api/user/self/email-subscriptions", tokens[1], "application/json", `{"promotion":false}`)
	assert.Equal(t, 400, response.Code)
	response = request("GET", "/api/user/self/email-subscriptions", pat, "", "")
	assert.Equal(t, 403, response.Code)
	messages := campaignSMTPFixture(t, "250 recipient ok", false)
	response = request("POST", "/api/email-campaign/", tokens[0], "application/json", `{"subject":"Platform update","body":"<script>alert(1)</script>\nHello","category":"platform","group":"default"}`)
	require.Equal(t, 200, response.Code, response.Body.String())
	var created struct {
		Data model.EmailCampaign `json:"data"`
	}
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &created))
	path := "/api/email-campaign/" + strconv.FormatInt(created.Data.ID, 10)
	response = request("GET", path+"/preview", tokens[0], "", "")
	require.Equal(t, 200, response.Code, response.Body.String())
	assert.Contains(t, response.Body.String(), `"eligible":1`)
	for range 2 {
		response = request("POST", path+"/queue", tokens[0], "application/json", `{}`)
		require.Equal(t, 200, response.Code, response.Body.String())
	}
	processed, err := service.DispatchEmailCampaigns(context.Background(), "integration-worker")
	require.NoError(t, err)
	assert.Equal(t, 1, processed, "repeat queue request must not duplicate email")
	message := <-messages
	parsed, err := mail.ReadMessage(strings.NewReader(message))
	require.NoError(t, err)
	assert.Equal(t, "List-Unsubscribe=One-Click", parsed.Header.Get("List-Unsubscribe-Post"))
	unsubscribeURL, err := url.Parse(strings.Trim(parsed.Header.Get("List-Unsubscribe"), "<>"))
	require.NoError(t, err)
	_, params, err := mime.ParseMediaType(parsed.Header.Get("Content-Type"))
	require.NoError(t, err)
	parts := multipart.NewReader(parsed.Body, params["boundary"])
	var htmlBody string
	for {
		part, err := parts.NextPart()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		content, err := io.ReadAll(part)
		require.NoError(t, err)
		if strings.HasPrefix(part.Header.Get("Content-Type"), "text/html") {
			htmlBody = string(content)
		}
	}
	assert.Contains(t, htmlBody, "&lt;script&gt;alert(1)&lt;/script&gt;")
	assert.Contains(t, htmlBody, "/email/unsubscribe#token=")
	response = request("GET", path+"/deliveries", tokens[0], "", "")
	require.Equal(t, 200, response.Code)
	assert.Contains(t, response.Body.String(), `"status":"sent"`)
	assert.NotContains(t, response.Body.String(), "unsubscribe_token")
	response = request("GET", unsubscribeURL.RequestURI(), "", "", "")
	assert.Equal(t, 200, response.Code)
	assert.Contains(t, response.Header().Get("Cache-Control"), "no-store")
	assert.Contains(t, response.Body.String(), `"unsubscribed":false`)
	// Mail-client link checks must not exhaust the shared authentication quota.
	for range 21 {
		response = request("GET", unsubscribeURL.RequestURI(), "", "", "")
		require.Equal(t, 200, response.Code)
	}
	response = request("POST", unsubscribeURL.RequestURI(), "", "application/json", `{}`)
	assert.Equal(t, 400, response.Code)
	for range 2 {
		response = request("POST", unsubscribeURL.RequestURI(), "", "application/x-www-form-urlencoded", "List-Unsubscribe=One-Click")
		assert.Equal(t, 200, response.Code)
		assert.Contains(t, response.Body.String(), `"unsubscribed":true`)
	}
	assert.NotContains(t, accessLog.String(), unsubscribeURL.Query().Get("token"))
	preferences, err := model.GetEmailPreferences(users[1].Id)
	require.NoError(t, err)
	assert.Equal(t, &model.EmailPreferences{Promotion: true, Platform: false}, preferences)

	for _, test := range []struct {
		name, reply, status string
		drop                bool
	}{
		{"temporary rejection", "451 try later", model.EmailDeliveryRetry, false},
		{"permanent rejection", "550 unavailable", model.EmailDeliveryFailed, false},
		{"lost acknowledgement", "250 ok", model.EmailDeliveryUncertain, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			campaignSMTPFixture(t, test.reply, test.drop)
			campaign := model.EmailCampaign{Subject: test.name, Body: "hello", Category: model.EmailCategoryPromotion, TargetGroup: "default", CreatedBy: users[0].Id}
			require.NoError(t, model.CreateEmailCampaign(&campaign))
			_, err := model.QueueEmailCampaign(campaign.ID)
			require.NoError(t, err)
			require.NoError(t, db.Model(&model.EmailDispatchConfig{}).Where("id = 1").Update("next_send_at", 0).Error)
			delivery, err := model.ClaimEmailDelivery("result-worker", time.Now().UnixMilli(), 60000)
			require.NoError(t, err)
			require.NotNil(t, delivery)
			require.NoError(t, service.DeliverEmailCampaign(context.Background(), delivery, "result-worker"))
			var stored model.EmailDelivery
			require.NoError(t, db.First(&stored, delivery.ID).Error)
			assert.Equal(t, test.status, stored.Status)
			assert.Equal(t, test.status == model.EmailDeliveryRetry, stored.NextAttemptAt > 0)
			assert.NotContains(t, stored.LastError, "example.test")
			response = request("GET", fmt.Sprintf("/api/email-campaign/%d/deliveries", campaign.ID), tokens[0], "", "")
			assert.Equal(t, 200, response.Code)
		})
	}
}
