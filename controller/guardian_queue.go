package controller

import (
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
)

// guardianOrderView adds the buyer's own queue place; other buyers' orders are never exposed.
type guardianOrderView struct {
	model.GuardianServiceOrder
	QueueAhead     *int   `json:"queue_ahead"`
	QueueStartDate string `json:"queue_start_date"`
}

type adminGuardianQueueItem struct {
	model.AdminGuardianOrder
	QueueAhead     int    `json:"queue_ahead"`
	QueueStartDate string `json:"queue_start_date"`
}

func AdminGetGuardianQueue(c *gin.Context) {
	now := time.Now()
	queue, err := model.LoadGuardianQueue(operation_setting.GetGuardianQueueSetting(), now)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	inProgress, err := model.AdminGuardianOrders(queue.InProgress)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	waitingOrders, err := model.AdminGuardianOrders(queue.Waiting)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	waiting := make([]adminGuardianQueueItem, 0, len(waitingOrders))
	for _, order := range waitingOrders {
		slot := queue.Slots[order.ID]
		waiting = append(waiting, adminGuardianQueueItem{AdminGuardianOrder: order, QueueAhead: slot.Ahead, QueueStartDate: slot.StartDate})
	}
	common.ApiSuccess(c, gin.H{"setting": queue.Setting, "paused": queue.Paused, "started_today": queue.StartedToday, "in_progress": inProgress, "waiting": waiting})
}

func AdminUpdateGuardianQueueSetting(c *gin.Context) {
	var req operation_setting.GuardianQueueSetting
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if req.DailyCapacity < 0 || req.DailyCapacity > 1000 || req.ExtraSlots < 0 || req.ExtraSlots > 1000 {
		common.ApiErrorMsg(c, "每日上限和额外占位需在 0 到 1000 之间")
		return
	}
	if !req.Paused {
		req.ResumeAt = 0
	}
	if req.Paused && req.ResumeAt != 0 && req.ResumeAt <= time.Now().Unix() {
		common.ApiErrorMsg(c, "恢复时间需晚于当前时间")
		return
	}
	if err := model.UpdateOptionsBulk(map[string]string{
		"guardian_queue.daily_capacity": strconv.Itoa(req.DailyCapacity),
		"guardian_queue.extra_slots":    strconv.Itoa(req.ExtraSlots),
		"guardian_queue.paused":         strconv.FormatBool(req.Paused),
		"guardian_queue.resume_at":      strconv.FormatInt(req.ResumeAt, 10),
	}); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, operation_setting.GetGuardianQueueSetting())
}

func AdminMoveGuardianQueueOrder(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "订单 ID 无效")
		return
	}
	var req struct {
		Position int `json:"position"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Position < 1 {
		common.ApiErrorMsg(c, "排位无效")
		return
	}
	if err := model.MoveGuardianQueueOrder(id, req.Position); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}
