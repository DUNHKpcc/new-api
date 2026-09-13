package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWeChatConfigurationReadinessRequiresCompleteContracts(t *testing.T) {
	previousAppID, previousAppSecret := WeChatAppId, WeChatAppSecret
	previousAddress, previousToken := WeChatServerAddress, WeChatServerToken
	t.Cleanup(func() {
		WeChatAppId, WeChatAppSecret = previousAppID, previousAppSecret
		WeChatServerAddress, WeChatServerToken = previousAddress, previousToken
	})

	WeChatAppId, WeChatAppSecret = "wx-app", "wx-secret"
	WeChatServerAddress, WeChatServerToken = "", ""
	assert.True(t, WeChatDirectOAuthConfigured())
	assert.False(t, WeChatServerBridgeConfigured())
	assert.True(t, WeChatAuthConfigured())

	WeChatAppId, WeChatAppSecret = "", ""
	WeChatServerAddress, WeChatServerToken = "https://bridge.example/base", "bridge-token"
	assert.False(t, WeChatDirectOAuthConfigured())
	assert.True(t, WeChatServerBridgeConfigured())
	assert.True(t, WeChatAuthConfigured())

	WeChatServerAddress = "bridge.example/base"
	assert.False(t, WeChatServerBridgeConfigured())
	WeChatServerAddress = "https://bridge.example/base?unexpected=query"
	assert.False(t, WeChatServerBridgeConfigured())
}
