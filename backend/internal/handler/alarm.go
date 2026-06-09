package handler

import (
	"net/http"
	"strconv"

	"gas-leak-monitor/internal/model"
	"gas-leak-monitor/internal/repository"

	"github.com/gin-gonic/gin"
)

type AlarmHandler struct {
	pgRepo *repository.PostgresRepo
}

func NewAlarmHandler(pgRepo *repository.PostgresRepo) *AlarmHandler {
	return &AlarmHandler{pgRepo: pgRepo}
}

func (h *AlarmHandler) GetAlarms(c *gin.Context) {
	level := 0
	if l := c.Query("level"); l != "" {
		level, _ = strconv.Atoi(l)
	}

	status := c.Query("status")

	page := 1
	if p := c.Query("page"); p != "" {
		page, _ = strconv.Atoi(p)
		if page < 1 {
			page = 1
		}
	}

	pageSize := 20

	alarms, err := h.pgRepo.GetAlarms(level, status, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if alarms == nil {
		alarms = []model.Alarm{}
	}

	c.JSON(http.StatusOK, gin.H{
		"alarms": alarms,
		"page":   page,
		"size":   pageSize,
	})
}

func (h *AlarmHandler) AcknowledgeAlarm(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid alarm id"})
		return
	}

	if err := h.pgRepo.AcknowledgeAlarm(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "acknowledged"})
}

func (h *AlarmHandler) ResolveAlarm(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid alarm id"})
		return
	}

	if err := h.pgRepo.ResolveAlarm(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "resolved"})
}
