package controller

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLotteryItemsAcceptsManagedWebPContent(t *testing.T) {
	image := validResourceDownloadThumbnail()
	value := `[{
		"id":"summer-draw",
		"title":"Summer draw",
		"content":"Join before Friday.",
		"winnerInfo":"",
		"image":"` + image + `",
		"publishDate":"2026-08-09T08:00:00Z"
	}]`

	items, err := parseLotteryItems(value)

	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "summer-draw", items[0].ID)
	assert.Equal(t, image, items[0].Image)
	assert.Empty(t, items[0].WinnerInfo)
}

func TestParseLotteryItemsRejectsInvalidImage(t *testing.T) {
	value := `[{
		"id":"invalid-image",
		"title":"Invalid image",
		"content":"Content",
		"winnerInfo":"Winner #1",
		"image":"data:image/png;base64,aW1hZ2U=",
		"publishDate":"2026-08-09T08:00:00Z"
	}]`

	_, err := parseLotteryItems(value)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "WebP")
}

func TestParseLotteryItemsRejectsNullAndExcessItems(t *testing.T) {
	_, err := parseLotteryItems("null")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JSON 数组")

	item := `{
		"id":"lottery-%d",
		"title":"Lottery",
		"content":"Content",
		"winnerInfo":"",
		"image":"` + validResourceDownloadThumbnail() + `",
		"publishDate":"2026-08-09T08:00:00Z"
	}`
	values := make([]string, maxLotteryItems+1)
	for index := range values {
		values[index] = strings.Replace(item, "%d", string(rune('a'+index)), 1)
	}

	_, err = parseLotteryItems("[" + strings.Join(values, ",") + "]")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "不能超过")
}

func TestGetLotteryItemsSortsNewestFirst(t *testing.T) {
	common.OptionMapRWMutex.Lock()
	mapWasNil := common.OptionMap == nil
	if mapWasNil {
		common.OptionMap = make(map[string]string)
	}
	original := common.OptionMap["LotteryItems"]
	common.OptionMap["LotteryItems"] = `[
		{"id":"older","title":"Older","content":"Content","winnerInfo":"","image":"` + validResourceDownloadThumbnail() + `","publishDate":"2026-08-08T08:00:00Z"},
		{"id":"newer","title":"Newer","content":"Content","winnerInfo":"Winner","image":"` + validResourceDownloadThumbnail() + `","publishDate":"2026-08-09T08:00:00Z"}
	]`
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		if mapWasNil {
			common.OptionMap = nil
		} else {
			common.OptionMap["LotteryItems"] = original
		}
		common.OptionMapRWMutex.Unlock()
	})

	common.OptionMapRWMutex.RLock()
	items := getLotteryItems()
	common.OptionMapRWMutex.RUnlock()

	require.Len(t, items, 2)
	assert.Equal(t, "newer", items[0].ID)
	assert.Equal(t, "older", items[1].ID)
}
