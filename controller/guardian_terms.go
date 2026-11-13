package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func GetGuardianAfterSalesTerms(c *gin.Context) {
	terms, err := model.GetGuardianAfterSalesTerms(model.DB)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, terms)
}

func AdminUpdateGuardianAfterSalesTerms(c *gin.Context) {
	var terms model.GuardianAfterSalesTerms
	if err := c.ShouldBindJSON(&terms); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if err := model.SaveGuardianAfterSalesTerms(terms); err != nil {
		common.ApiError(c, err)
		return
	}
	GetGuardianAfterSalesTerms(c)
}
