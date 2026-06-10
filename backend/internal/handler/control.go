package handler

import (
	"net/http"
	"time"

	"gas-leak-monitor/internal/controller"
	"gas-leak-monitor/internal/model"

	"github.com/gin-gonic/gin"
)

type ControlHandler struct {
	ec *controller.EmergencyController
}

func NewControlHandler(ec *controller.EmergencyController) *ControlHandler {
	return &ControlHandler{ec: ec}
}

type ValveControlRequest struct {
	PartitionID string `json:"partition_id"`
	Action      string `json:"action"`
}

type FanControlRequest struct {
	PartitionID string `json:"partition_id"`
	Action      string `json:"action"`
}

type NotifyRequest struct {
	PartitionID string `json:"partition_id"`
	Message     string `json:"message"`
}

func (h *ControlHandler) ControlValve(c *gin.Context) {
	var req ValveControlRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Action != "open" && req.Action != "close" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "action must be 'open' or 'close'"})
		return
	}
	cmdID := h.ec.ManualValveControl(req.PartitionID, req.Action)
	c.JSON(http.StatusOK, gin.H{
		"status":     "sent",
		"command_id": cmdID,
	})
}

func (h *ControlHandler) ControlFan(c *gin.Context) {
	var req FanControlRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Action != "start" && req.Action != "stop" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "action must be 'start' or 'stop'"})
		return
	}
	cmdID := h.ec.ManualFanControl(req.PartitionID, req.Action)
	c.JSON(http.StatusOK, gin.H{
		"status":     "sent",
		"command_id": cmdID,
	})
}

func (h *ControlHandler) SendNotification(c *gin.Context) {
	var req NotifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.ec.ManualNotify(req.PartitionID, req.Message)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *ControlHandler) GetPartitions(c *gin.Context) {
	partitions, err := h.ec.GetPartitions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, partitions)
}

func (h *ControlHandler) CreateControlLog(c *gin.Context) {
	var log model.ControlLog
	if err := c.ShouldBindJSON(&log); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	log.CreatedAt = time.Now()
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
