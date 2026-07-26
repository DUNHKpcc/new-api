package controller

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v81/webhook"
)

func TestCreemWebhookSignatureFixture(t *testing.T) {
	const (
		payload   = `{"eventType":"checkout.completed","object":{"id":"ch_contract"}}`
		secret    = "creem_contract_secret"
		signature = "652923242afa0e3fb556a1f1b7f39d819e2c74b5c663b5685f7f7ff749319023"
	)

	assert.Equal(t, signature, generateCreemSignature(payload, secret))
	assert.True(t, verifyCreemSignature(payload, signature, secret))
	assert.False(t, verifyCreemSignature(payload+" ", signature, secret))
	assert.False(t, verifyCreemSignature(payload, signature[:len(signature)-1]+"0", secret))
}

func TestStripeWebhookSignatureFixture(t *testing.T) {
	const (
		payload   = `{"id":"evt_contract","object":"event","type":"checkout.session.completed","data":{"object":{"id":"cs_contract"}}}`
		secret    = "whsec_contract_secret"
		timestamp = int64(1785024000)
	)
	mac := hmac.New(sha256.New, []byte(secret))
	_, err := mac.Write([]byte(fmt.Sprintf("%d.%s", timestamp, payload)))
	require.NoError(t, err)
	signature := fmt.Sprintf("t=%d,v1=%x", timestamp, mac.Sum(nil))
	options := webhook.ConstructEventOptions{
		IgnoreAPIVersionMismatch: true,
		IgnoreTolerance:          true,
	}

	event, err := webhook.ConstructEventWithOptions([]byte(payload), signature, secret, options)
	require.NoError(t, err)
	assert.Equal(t, "evt_contract", event.ID)
	_, err = webhook.ConstructEventWithOptions([]byte(payload+" "), signature, secret, options)
	require.Error(t, err)
}
