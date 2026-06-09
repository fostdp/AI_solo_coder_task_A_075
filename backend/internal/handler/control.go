package handler

import (
	"net/http"
	"time"

	"gas-leak-monitor/internal/model"
	"gas-leak-monitor/internal/mqtt"
	"gas-leak-monitor/internal/repository"
	"gas-leak-monitor/internal/ws"

	"github.com/gin-gonic/gin"
)

type ControlHandler struct {
	pgRepo     *repository.PostgresRepo
	mqttClient *mqtt.Client
	hub        *ws.Hub
}

func NewControlHandler(pgRepo *repository.PostgresRepo, mqttClient *mqtt.Client, hub *ws.Hub) *ControlHandler {
	return &ControlHandler{
		pgRepo:     pgRepo,
		mqttClient: mqttClient,
		hub:        hub,
	}
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

	h.mqttClient.PublishValveControl(req.PartitionID, req.Action)

	status := "open"
	if req.Action == "close" {
		status = "closed"
	}
	h.pgRepo.UpdatePartitionValve(req.PartitionID, status)
	h.pgRepo.CreateControlLog(model.ControlLog{
		TargetType: "valve",
		TargetID:   req.PartitionID,
		Action:     req.Action,
		Operator:   "manual",
		CreatedAt:  time.Now(),
	})

	h.hub.Broadcast(model.WSMessage{
		Type: "control",
		Payload: map[string]interface{}{
			"type":   "valve",
			"id":     req.PartitionID,
			"action": req.Action,
		},
	})

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
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

	h.mqttClient.PublishFanControl(req.PartitionID, req.Action)

	status := "stopped"
	if req.Action == "start" {
		status = "running"
	}
	h.pgRepo.UpdatePartitionFan(req.PartitionID, status)
	h.pgRepo.CreateControlLog(model.ControlLog{
		TargetType: "fan",
		TargetID:   req.PartitionID,
		Action:     req.Action,
		Operator:   "manual",
		CreatedAt:  time.Now(),
	})

	h.hub.Broadcast(model.WSMessage{
		Type: "control",
		Payload: map[string]interface{}{
			"type":   "fan",
			"id":     req.PartitionID,
			"action": req.Action,
		},
	})

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *ControlHandler) SendNotification(c *gin.Context) {
	var req NotifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.mqttClient.Publish("notification/evacuation", 1, req.Message)
	h.pgRepo.CreateControlLog(model.ControlLog{
		TargetType: "notification",
		TargetID:   req.PartitionID,
		Action:     "evacuation",
		Operator:   "manual",
		CreatedAt:  time.Now(),
	})

	h.hub.Broadcast(model.WSMessage{
		Type: "notification",
		Payload: map[string]interface{}{
			"partition_id": req.PartitionID,
			"message":      req.Message,
		},
	})

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *ControlHandler) GetPartitions(c *gin.Context) {
	partitions, err := h.pgRepo.GetPartitions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, partitions)
}
