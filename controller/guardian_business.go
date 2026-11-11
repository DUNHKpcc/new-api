package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func GetGuardianReceipt(c *gin.Context)       { guardianReceipt(c, false, false) }
func AdminGetGuardianReceipt(c *gin.Context)  { guardianReceipt(c, true, false) }
func SendGuardianReceipt(c *gin.Context)      { guardianReceipt(c, false, true) }
func AdminSendGuardianReceipt(c *gin.Context) { guardianReceipt(c, true, true) }
func guardianReceipt(c *gin.Context, admin, send bool) {
	id, ok := guardianPathID(c, "id")
	if !ok {
		return
	}
	if send {
		if err := model.RequestGuardianReceiptEmail(id, c.GetInt("id"), admin); err != nil {
			common.ApiError(c, err)
			return
		}
		common.ApiSuccess(c, nil)
		return
	}
	receipt, err := model.GetGuardianReceipt(id, c.GetInt("id"), admin)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, receipt)
}
func AdminListGuardianTierSupplies(c *gin.Context) {
	items, err := model.ListGuardianTierSupplies()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, items)
}
func AdminChangeGuardianTier(c *gin.Context) {
	var req model.GuardianTierChange
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	result, err := model.ChangeGuardianTier(c.GetInt("id"), req)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func GetGuardianCheckoutContext(c *gin.Context) {
	var order model.GuardianServiceOrder
	result := model.DB.Select("discovery_channel").Where("user_id = ? AND discovery_channel <> ?", c.GetInt("id"), "").Order("id DESC").Limit(1).Find(&order)
	if result.Error != nil {
		common.ApiError(c, result.Error)
		return
	}
	common.ApiSuccess(c, gin.H{"discovery_channel": order.DiscoveryChannel})
}
