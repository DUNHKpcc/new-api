package controller

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validResourceDownloadThumbnail() string {
	const thumbnail = "UklGRlAAAABXRUJQVlA4WAoAAAAQAAAAAAAAAAAAQUxQSAIAAAAALlZQOCAoAAAAcAEAnQEqAQABAAIANCWgAnQBQAAA/umt//wQX9Dn9bePDVlqIXIAAA=="
	return "data:image/webp;base64," + thumbnail
}

func TestParseResourceDownloadItemsAcceptsWebPDownloads(t *testing.T) {
	thumbnail := validResourceDownloadThumbnail()
	value := `[{
		"id":"desktop-client",
		"name":"Desktop Client",
		"description":"Stable release",
		"url":"https://example.com/download/client",
		"thumbnail":"` + thumbnail + `"
	}]`

	items, err := parseResourceDownloadItems(value)

	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "desktop-client", items[0].ID)
	assert.Equal(t, thumbnail, items[0].Thumbnail)
}

func TestParseResourceDownloadItemsRejectsUnsafeLinks(t *testing.T) {
	value := `[{
		"id":"unsafe",
		"name":"Unsafe",
		"description":"",
		"url":"javascript:alert(1)",
		"thumbnail":"` + validResourceDownloadThumbnail() + `"
	}]`

	_, err := parseResourceDownloadItems(value)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP(S)")
}

func TestParseResourceDownloadItemsRequiresWebPThumbnails(t *testing.T) {
	value := `[{
		"id":"png-image",
		"name":"PNG image",
		"description":"",
		"url":"https://example.com/download",
		"thumbnail":"data:image/png;base64,aW1hZ2U="
	}]`

	_, err := parseResourceDownloadItems(value)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "WebP")
}

func TestParseResourceDownloadItemsRejectsNullConfiguration(t *testing.T) {
	_, err := parseResourceDownloadItems("null")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "JSON 数组")
}

func TestParseResourceDownloadItemsRejectsMalformedWebPThumbnails(t *testing.T) {
	webpHeader := []byte("RIFF\x04\x00\x00\x00WEBP")
	value := `[{
		"id":"broken-image",
		"name":"Broken image",
		"description":"",
		"url":"https://example.com/download",
		"thumbnail":"data:image/webp;base64,` + base64.StdEncoding.EncodeToString(webpHeader) + `"
	}]`

	_, err := parseResourceDownloadItems(value)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "有效的 WebP")
}

func TestParseResourceDownloadItemsCapsItemCount(t *testing.T) {
	item := `{
		"id":"%d",
		"name":"Resource",
		"description":"",
		"url":"https://example.com/download",
		"thumbnail":"` + validResourceDownloadThumbnail() + `"
	}`
	values := make([]string, maxResourceDownloadItems+1)
	for index := range values {
		values[index] = strings.Replace(item, "%d", string(rune('a'+index)), 1)
	}

	_, err := parseResourceDownloadItems("[" + strings.Join(values, ",") + "]")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "不能超过")
}

func TestParseResourceDownloadItemsAcceptsMaxSizedConfiguration(t *testing.T) {
	const webpDataPrefix = "data:image/webp;base64,"
	thumbnailBytes, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(validResourceDownloadThumbnail(), webpDataPrefix))
	require.NoError(t, err)
	thumbnailBytes = append(thumbnailBytes, make([]byte, maxResourceThumbnailBytes-len(thumbnailBytes))...)
	thumbnail := webpDataPrefix + base64.StdEncoding.EncodeToString(thumbnailBytes)

	values := make([]string, maxResourceDownloadItems)
	for index := range values {
		values[index] = fmt.Sprintf(`{
			"id":"resource-%d",
			"name":"Resource",
			"description":"",
			"url":"https://example.com/download",
			"thumbnail":"%s"
		}`, index, thumbnail)
	}

	items, err := parseResourceDownloadItems("[" + strings.Join(values, ",") + "]")

	require.NoError(t, err)
	assert.Len(t, items, maxResourceDownloadItems)
}
