package model

import (
	"database/sql/driver"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
)

type GuardianIPOption struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Description       string `json:"description"`
	QuarterPriceCents int64  `json:"quarter_price_cents"`
	AnnualPriceCents  int64  `json:"annual_price_cents"`
	PriceCents        int64  `json:"price_cents"`
	Enabled           bool   `json:"enabled"`
}

type GuardianIPOptions []GuardianIPOption

func (options GuardianIPOptions) Value() (driver.Value, error) {
	b, err := common.Marshal(options)
	return string(b), err
}
func (options *GuardianIPOptions) Scan(value any) error {
	switch v := value.(type) {
	case string:
		return common.UnmarshalJsonStr(v, options)
	case []byte:
		return common.Unmarshal(v, options)
	case nil:
		*options = GuardianIPOptions{}
		return nil
	default:
		return errors.New("invalid Guardian IP options")
	}
}
func (options GuardianIPOptions) Validate() error {
	if len(options) > 12 {
		return errors.New("IP 加购选项最多 12 项")
	}
	ids := map[string]bool{}
	for i := range options {
		option := &options[i]
		option.Name = strings.TrimSpace(option.Name)
		option.Description = strings.TrimSpace(option.Description)
		if option.ID == "" || len(option.ID) > 64 || ids[option.ID] {
			return errors.New("IP 加购选项标识无效或重复")
		}
		ids[option.ID] = true
		if option.Name == "" || utf8.RuneCountInString(option.Name) > 128 || utf8.RuneCountInString(option.Description) > 1000 {
			return errors.New("IP 加购名称须为 1–128 字，说明最多 1000 字")
		}
		for _, price := range []int64{option.PriceCents, option.QuarterPriceCents, option.AnnualPriceCents} {
			if price < 1 || price > 10000000 {
				return errors.New("IP 月、季、年价格均须在 0.01 至 100000 元之间")
			}
		}
	}
	return nil
}

func (option GuardianIPOption) PeriodPrice(period string) (int64, error) {
	switch period {
	case "monthly":
		return option.PriceCents, nil
	case "quarterly":
		return option.QuarterPriceCents, nil
	case "yearly":
		return option.AnnualPriceCents, nil
	default:
		return 0, errors.New("请选择 IP 月付、季付或年付")
	}
}
