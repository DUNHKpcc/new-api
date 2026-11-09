package middleware

import (
	"net/http"

	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func GuardianResellerRequired(c *gin.Context) {
	profile, err := model.GetGuardianReseller(c.GetInt("id"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"success": false, "message": "代理资料暂时不可用"})
		return
	}
	if profile == nil {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"success": false, "message": "尚未开通 Guardian 代理"})
		return
	}
	c.Next()
}
