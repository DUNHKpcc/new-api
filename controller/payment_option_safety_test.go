package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfirmPaymentCompliancePersistsCompleteBatch(t *testing.T) {
	db := setupAffiliateRegistrationControllerTest(t)
	for _, key := range []string{
		model.PaymentComplianceConfirmedOptionKey,
		model.PaymentComplianceTermsVersionOptionKey,
		"payment_setting.compliance_confirmed_at",
		"payment_setting.compliance_confirmed_by",
		"payment_setting.compliance_confirmed_ip",
	} {
		require.NoError(t, db.Delete(&model.Option{Key: key}).Error)
	}
	operation_setting.GetPaymentSetting().ComplianceConfirmed = false
	operation_setting.GetPaymentSetting().ComplianceTermsVersion = ""

	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Set("id", 42)
	context.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/option/payment_compliance",
		strings.NewReader(`{"confirmed":true}`),
	)
	context.Request.Header.Set("Content-Type", "application/json")

	ConfirmPaymentCompliance(context)

	var payload struct {
		Success bool `json:"success"`
	}
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &payload))
	assert.True(t, payload.Success)
	var options []model.Option
	require.NoError(t, db.Find(&options).Error)
	values := make(map[string]string, len(options))
	for _, option := range options {
		values[option.Key] = option.Value
	}
	assert.Equal(t, "true", values[model.PaymentComplianceConfirmedOptionKey])
	assert.Equal(t, operation_setting.CurrentComplianceTermsVersion, values[model.PaymentComplianceTermsVersionOptionKey])
	assert.Equal(t, "42", values["payment_setting.compliance_confirmed_by"])
	assert.NotEmpty(t, values["payment_setting.compliance_confirmed_at"])
	assert.NotEmpty(t, values["payment_setting.compliance_confirmed_ip"])
	assert.True(t, operation_setting.IsPaymentComplianceConfirmed())
}

func TestUpdateOptionRejectsEpayCurrencyDriftFromPersistedAffiliateSetting(t *testing.T) {
	db := setupAffiliateRegistrationControllerTest(t)
	persisted := operation_setting.DefaultAffiliateSetting()
	persisted.CommissionEnabled = true
	encoded, err := operation_setting.MarshalAffiliateSetting(persisted)
	require.NoError(t, err)
	require.NoError(t, db.Model(&model.Option{}).
		Where(&model.Option{Key: operation_setting.AffiliateSettingOptionKey}).
		Update("value", encoded).Error)
	require.NoError(t, db.Create(&model.Option{Key: model.EpayCurrencyOptionKey, Value: "CNY"}).Error)

	runtimeSetting := persisted
	runtimeSetting.CommissionEnabled = false
	require.NoError(t, operation_setting.SetAffiliateSetting(runtimeSetting))
	operation_setting.GetPaymentSetting().EpayCurrency = "USD"

	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Request = httptest.NewRequest(
		http.MethodPut,
		"/api/option/",
		strings.NewReader(`{"key":"payment_setting.epay_currency","value":"USD"}`),
	)
	context.Request.Header.Set("Content-Type", "application/json")

	UpdateOption(context)

	var payload struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &payload))
	assert.False(t, payload.Success)
	assert.Contains(t, payload.Message, "不能修改 Epay 结算币种")
	var stored model.Option
	require.NoError(t, db.First(&stored, &model.Option{Key: model.EpayCurrencyOptionKey}).Error)
	assert.Equal(t, "CNY", stored.Value)
}
