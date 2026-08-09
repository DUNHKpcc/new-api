package controller

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"

	"github.com/gin-gonic/gin"
)

const (
	maxResourceDownloadItems  = 12
	maxResourceThumbnailBytes = 300 * 1024
	maxResourceDownloadConfig = maxResourceDownloadItems * ((maxResourceThumbnailBytes*4+2)/3 + 4*1024)
)

type ResourceDownloadItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	URL         string `json:"url"`
	Thumbnail   string `json:"thumbnail"`
}

func parseResourceDownloadItems(value string) ([]ResourceDownloadItem, error) {
	if strings.TrimSpace(value) == "" {
		return []ResourceDownloadItem{}, nil
	}
	if len(value) > maxResourceDownloadConfig {
		return nil, errors.New("资源下载配置过大")
	}

	var items []ResourceDownloadItem
	if err := common.UnmarshalJsonStr(value, &items); err != nil {
		return nil, errors.New("资源下载配置必须是有效的 JSON 数组")
	}
	if items == nil {
		return nil, errors.New("资源下载配置必须是有效的 JSON 数组")
	}
	if len(items) > maxResourceDownloadItems {
		return nil, fmt.Errorf("资源下载项不能超过 %d 个", maxResourceDownloadItems)
	}

	ids := make(map[string]struct{}, len(items))
	for index, item := range items {
		itemNumber := index + 1
		if item.ID == "" || len(item.ID) > 64 {
			return nil, fmt.Errorf("第 %d 个资源的标识无效", itemNumber)
		}
		if _, exists := ids[item.ID]; exists {
			return nil, fmt.Errorf("第 %d 个资源的标识重复", itemNumber)
		}
		ids[item.ID] = struct{}{}

		if strings.TrimSpace(item.Name) == "" || utf8.RuneCountInString(item.Name) > 80 {
			return nil, fmt.Errorf("第 %d 个资源的名称无效", itemNumber)
		}
		if utf8.RuneCountInString(item.Description) > 200 {
			return nil, fmt.Errorf("第 %d 个资源的描述不能超过 200 个字符", itemNumber)
		}
		if len(item.URL) > 2048 {
			return nil, fmt.Errorf("第 %d 个资源的下载链接过长", itemNumber)
		}
		parsedUrl, err := url.ParseRequestURI(item.URL)
		if err != nil || (parsedUrl.Scheme != "https" && parsedUrl.Scheme != "http") || parsedUrl.Host == "" || parsedUrl.User != nil {
			return nil, fmt.Errorf("第 %d 个资源的下载链接必须是有效的 HTTP(S) 地址", itemNumber)
		}

		if err := validateManagedWebPDataURL(item.Thumbnail, maxResourceThumbnailBytes); err != nil {
			return nil, fmt.Errorf("第 %d 个资源的缩略图%s", itemNumber, err.Error())
		}
	}

	return items, nil
}

func GetResourceDownloads(c *gin.Context) {
	common.OptionMapRWMutex.RLock()
	value := common.OptionMap["ResourceDownloadItems"]
	common.OptionMapRWMutex.RUnlock()

	items, err := parseResourceDownloadItems(value)
	if err != nil {
		common.SysError("invalid ResourceDownloadItems option: " + err.Error())
		common.ApiErrorMsg(c, "资源下载配置无效")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    items,
	})
}
