package operation_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEpayCurrency(t *testing.T) {
	original := paymentSetting.EpayCurrency
	t.Cleanup(func() {
		paymentSetting.EpayCurrency = original
	})

	paymentSetting.EpayCurrency = " usd "
	assert.Equal(t, "USD", GetEpayCurrency())

	paymentSetting.EpayCurrency = "invalid"
	assert.Equal(t, "CNY", GetEpayCurrency())
}

func TestNormalizeEpayCurrency(t *testing.T) {
	currency, err := NormalizeEpayCurrency(" usd ")
	assert.NoError(t, err)
	assert.Equal(t, "USD", currency)

	for _, value := range []string{"", "US", "US1", "USDD"} {
		t.Run(value, func(t *testing.T) {
			_, err := NormalizeEpayCurrency(value)
			assert.Error(t, err)
		})
	}
}
