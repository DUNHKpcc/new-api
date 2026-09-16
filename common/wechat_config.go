package common

import (
	"net/url"
	"strings"
)

// WeChatDirectOAuthConfigured reports whether the Open Platform OAuth
// credentials are complete enough for the authorization-code flow.
func WeChatDirectOAuthConfigured() bool {
	return strings.TrimSpace(WeChatAppId) != "" &&
		strings.TrimSpace(WeChatAppSecret) != ""
}

// WeChatServerBridgeConfigured reports whether the compatibility verification
// service can be called safely. The bridge is administrator-configured, but we
// still require an absolute HTTP(S) URL and its shared token before exposing
// the login path.
func WeChatServerBridgeConfigured() bool {
	address := strings.TrimSpace(WeChatServerAddress)
	if address == "" || strings.TrimSpace(WeChatServerToken) == "" {
		return false
	}
	parsed, err := url.Parse(address)
	if err != nil || parsed.Host == "" || parsed.User != nil {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	// A base address must not carry a caller-controlled query or fragment;
	// the bridge endpoint adds its own code query below.
	return parsed.RawQuery == "" && parsed.Fragment == ""
}

// WeChatAuthConfigured reports whether at least one complete, supported WeChat
// login contract is available.
func WeChatAuthConfigured() bool {
	return WeChatDirectOAuthConfigured() || WeChatServerBridgeConfigured()
}
