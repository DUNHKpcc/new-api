package model

import (
	"errors"

	"gorm.io/gorm"
)

type GuardianCustomerService struct {
	ID             int    `json:"id"`
	UserID         int    `json:"-" gorm:"index"`
	ServiceID      int    `json:"service_id"`
	InitialOrderID int    `json:"initial_order_id" gorm:"uniqueIndex"`
	Title          string `json:"title" gorm:"type:varchar(128)"`
	ResellerID     int    `json:"reseller_id"`
	CreatedAt      int64  `json:"created_at"`
	UpdatedAt      int64  `json:"updated_at"`
}

func ListGuardianCustomerServices(userID int) ([]GuardianCustomerService, error) {
	items := []GuardianCustomerService{}
	err := DB.Where("user_id = ?", userID).Order("id DESC").Find(&items).Error
	return items, err
}

// Attribution belongs to the renewed service, never to a link supplied by the browser.
func guardianRenewalSource(tx *gorm.DB, userID int, req *GuardianOrderRequest) error {
	var item GuardianCustomerService
	if err := tx.First(&item, "id = ? AND user_id = ?", req.CustomerServiceID, userID).Error; err != nil {
		return errors.New("续费服务不存在")
	}
	if item.ServiceID != req.ServiceID {
		return errors.New("续费商品与原服务不一致")
	}
	req.ResellerCode = ""
	if item.ResellerID == 0 {
		return nil
	}
	var reseller GuardianReseller
	if err := tx.First(&reseller, "user_id = ?", item.ResellerID).Error; err != nil {
		return err
	}
	if reseller.Status != "active" {
		if !req.AcceptDirectRenewal {
			return errors.New("原代理已暂停，请明确确认转为平台直销续费")
		}
		req.renewalPreviousResellerID = item.ResellerID
		return nil
	}
	req.ResellerCode = reseller.Code
	return nil
}

type GuardianRenewalQuote struct {
	Service                    GuardianService `json:"service"`
	ResellerCode               string          `json:"reseller_code"`
	DirectConfirmationRequired bool            `json:"direct_confirmation_required"`
}

func GetGuardianRenewalQuote(userID, id int) (*GuardianRenewalQuote, error) {
	var item GuardianCustomerService
	if err := DB.First(&item, "id = ? AND user_id = ?", id, userID).Error; err != nil {
		return nil, errors.New("续费服务不存在")
	}
	var quote GuardianRenewalQuote
	if err := DB.First(&quote.Service, item.ServiceID).Error; err != nil {
		return nil, err
	}
	if !quote.Service.Enabled {
		return nil, errors.New("服务暂未开放续费，请联系平台")
	}
	if item.ResellerID == 0 {
		return &quote, nil
	}
	reseller, err := GetGuardianReseller(item.ResellerID)
	if err != nil {
		return nil, err
	}
	if reseller == nil {
		return nil, errors.New("服务归属异常，请联系平台")
	}
	if reseller.Status != "active" {
		quote.DirectConfirmationRequired = true
		return &quote, nil
	}
	products, err := GuardianResellerCatalog(reseller.Code)
	if err != nil {
		return nil, err
	}
	for _, product := range products {
		if product.ID == item.ServiceID {
			quote.Service, quote.ResellerCode = product, reseller.Code
			return &quote, nil
		}
	}
	return nil, errors.New("原代理尚未提供有效续费报价，请联系平台处理")
}
