package controller

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

const (
	maxLotteryItems      = 6
	maxLotteryImageBytes = 300 * 1024
	maxLotteryConfig     = maxLotteryItems * ((maxLotteryImageBytes*4+2)/3 + 8*1024)
)

type LotteryItem struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	WinnerInfo  string `json:"winnerInfo"`
	Image       string `json:"image"`
	PublishDate string `json:"publishDate"`
}

func parseLotteryItems(value string) ([]LotteryItem, error) {
	if strings.TrimSpace(value) == "" {
		return []LotteryItem{}, nil
	}
	if len(value) > maxLotteryConfig {
		return nil, errors.New("抽奖配置过大")
	}

	var items []LotteryItem
	if err := common.UnmarshalJsonStr(value, &items); err != nil || items == nil {
		return nil, errors.New("抽奖配置必须是有效的 JSON 数组")
	}
	if len(items) > maxLotteryItems {
		return nil, fmt.Errorf("抽奖内容不能超过 %d 条", maxLotteryItems)
	}

	ids := make(map[string]struct{}, len(items))
	for index, item := range items {
		itemNumber := index + 1
		if item.ID == "" || len(item.ID) > 64 {
			return nil, fmt.Errorf("第 %d 条抽奖内容的标识无效", itemNumber)
		}
		if _, exists := ids[item.ID]; exists {
			return nil, fmt.Errorf("第 %d 条抽奖内容的标识重复", itemNumber)
		}
		ids[item.ID] = struct{}{}

		if strings.TrimSpace(item.Title) == "" || utf8.RuneCountInString(item.Title) > 80 {
			return nil, fmt.Errorf("第 %d 条抽奖内容的标题无效", itemNumber)
		}
		if strings.TrimSpace(item.Content) == "" || utf8.RuneCountInString(item.Content) > 500 {
			return nil, fmt.Errorf("第 %d 条抽奖内容的正文无效", itemNumber)
		}
		if utf8.RuneCountInString(item.WinnerInfo) > 500 {
			return nil, fmt.Errorf("第 %d 条抽奖内容的中奖信息不能超过 500 个字符", itemNumber)
		}
		if _, err := time.Parse(time.RFC3339, item.PublishDate); err != nil {
			return nil, fmt.Errorf("第 %d 条抽奖内容的发布时间无效", itemNumber)
		}
		if err := validateManagedWebPDataURL(item.Image, maxLotteryImageBytes); err != nil {
			return nil, fmt.Errorf("第 %d 条抽奖内容的图片%s", itemNumber, err.Error())
		}
	}

	return items, nil
}

func getLotteryItems() []LotteryItem {
	items, err := parseLotteryItems(common.OptionMap["LotteryItems"])
	if err != nil {
		common.SysError("invalid LotteryItems option: " + err.Error())
		return []LotteryItem{}
	}

	sort.SliceStable(items, func(left, right int) bool {
		leftTime, leftErr := time.Parse(time.RFC3339, items[left].PublishDate)
		rightTime, rightErr := time.Parse(time.RFC3339, items[right].PublishDate)
		if leftErr != nil || rightErr != nil {
			return false
		}
		return leftTime.After(rightTime)
	})
	return items
}

func GetLotteryItems(c *gin.Context) {
	common.OptionMapRWMutex.RLock()
	items := getLotteryItems()
	common.OptionMapRWMutex.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    items,
	})
}
