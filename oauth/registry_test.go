package oauth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomProviderCannotReplaceOrUnregisterBuiltInWeChat(t *testing.T) {
	builtIn := GetProvider("wechat")
	require.NotNil(t, builtIn)

	RegisterCustom("wechat", &GenericOAuthProvider{})
	assert.Same(t, builtIn, GetProvider("wechat"))

	UnregisterCustomProvider("wechat")
	assert.Same(t, builtIn, GetProvider("wechat"))
	assert.False(t, IsCustomProvider("wechat"))
}
