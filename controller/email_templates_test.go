package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderEmailVerificationContentUsesPccAgentTheme(t *testing.T) {
	content := renderEmailVerificationContent(
		"New API",
		"123456",
		"https://example.com/pcc-agent-logo.png",
		10,
	)

	assert.Contains(t, content, `alt="DPCC API"`)
	assert.Contains(t, content, `src="https://example.com/pcc-agent-logo.png"`)
	assert.Contains(t, content, "DPCC API")
	assert.NotContains(t, content, ">PccAgent<")
	assert.Contains(t, content, "DPCC API 基于 New API 提供")
	assert.Contains(t, content, "验证你的邮箱")
	assert.Contains(t, content, "123456")
	assert.Contains(t, content, "10 分钟")
	assert.Contains(t, content, "#faf9f5")
	assert.Contains(t, content, "#d97757")
}

func TestRenderEmailVerificationContentEscapesDynamicValues(t *testing.T) {
	content := renderEmailVerificationContent(
		`New API <script>alert("x")</script>`,
		`123<&>`,
		`https://example.com/logo.png?next="unsafe"`,
		10,
	)

	assert.NotContains(t, content, "<script>")
	assert.NotContains(t, content, `next="unsafe"`)
	assert.Contains(
		t,
		content,
		`New API &lt;script&gt;alert(&#34;x&#34;)&lt;/script&gt;`,
	)
	assert.Contains(t, content, "123&lt;&amp;&gt;")
	assert.Contains(t, content, "next=&#34;unsafe&#34;")
}
