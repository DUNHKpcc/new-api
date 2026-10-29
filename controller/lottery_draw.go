package controller

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func ListLotteries(c *gin.Context) {
	items := []model.Lottery{}
	query := model.DB.Order("id DESC").Limit(50)
	if c.Query("before") != "" {
		query = query.Where("id < ?", c.Query("before"))
	}
	if err := query.Find(&items).Error; err != nil {
		c.JSON(500, gin.H{"success": false, "message": "could not load lotteries"})
		return
	}
	c.JSON(200, gin.H{"success": true, "data": items})
}
func SaveLottery(c *gin.Context) {
	var l model.Lottery
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16000)
	if err := c.ShouldBindJSON(&l); err != nil {
		c.JSON(400, gin.H{"success": false, "message": "invalid lottery"})
		return
	}
	if err := model.SaveLottery(&l); err != nil {
		c.JSON(400, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(200, gin.H{"success": true, "data": l})
}
func ChangeLottery(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(400, gin.H{"success": false, "message": "invalid lottery ID"})
		return
	}
	if err := model.TransitionLottery(id, c.Param("action")); err != nil {
		c.JSON(400, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(200, gin.H{"success": true})
}
func JoinLottery(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(400, gin.H{"success": false, "message": "invalid lottery ID"})
		return
	}
	if err := model.JoinLottery(id, c.GetInt("id")); err != nil {
		c.JSON(400, gin.H{"success": false, "message": err.Error()})
		return
	}
	// The scheduled worker retries if this draw fails; entry remains persisted.
	_ = model.DrawLottery(id, time.Now().Unix())
	c.JSON(200, gin.H{"success": true})
}
func GetLotteryEntry(c *gin.Context) {
	var entry model.LotteryEntry
	err := model.DB.Where("lottery_id = ? AND user_id = ?", c.Param("id"), c.GetInt("id")).First(&entry).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(200, gin.H{"success": true, "data": nil})
		return
	}
	if err != nil {
		c.JSON(500, gin.H{"success": false, "message": "could not load lottery entry"})
		return
	}
	c.JSON(200, gin.H{"success": true, "data": entry})
}
func ListLotteryWinners(c *gin.Context) {
	entries := []model.LotteryEntry{}
	if err := model.DB.Where("lottery_id = ? AND winner = ?", c.Param("id"), true).Order("id").Limit(100).Find(&entries).Error; err != nil {
		c.JSON(500, gin.H{"success": false, "message": "could not load winners"})
		return
	}
	c.JSON(200, gin.H{"success": true, "data": entries})
}
func UpdateLotteryWinner(c *gin.Context) {
	var request struct {
		Action string `json:"action"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, gin.H{"success": false, "message": "invalid action"})
		return
	}
	q := model.DB.Model(&model.LotteryEntry{}).Where("id = ? AND lottery_id = ? AND winner = ?", c.Param("entry"), c.Param("id"), true)
	var result *gorm.DB
	switch request.Action {
	case "fulfill":
		result = q.Update("fulfilled", true)
	case "retry":
		result = q.Where("email_status IN ?", []string{"failed", "uncertain"}).Updates(map[string]any{"email_status": "pending", "email_attempts": 0, "email_next_at": 0})
	default:
		c.JSON(400, gin.H{"success": false, "message": "invalid action"})
		return
	}
	if result.Error != nil || result.RowsAffected != 1 {
		c.JSON(409, gin.H{"success": false, "message": "winner record cannot be updated"})
		return
	}
	c.JSON(200, gin.H{"success": true})
}
