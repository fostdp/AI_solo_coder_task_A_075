package handler

import (
	"net/http"
	"time"

	"gas-leak-monitor/internal/model"
	"gas-leak-monitor/internal/repository"

	"github.com/gin-gonic/gin"
)

type SensorHandler struct {
	influxRepo *repository.InfluxRepo
	pgRepo     *repository.PostgresRepo
}

func NewSensorHandler(influxRepo *repository.InfluxRepo, pgRepo *repository.PostgresRepo) *SensorHandler {
	return &SensorHandler{
		influxRepo: influxRepo,
		pgRepo:     pgRepo,
	}
}

func (h *SensorHandler) GetSensors(c *gin.Context) {
	sensors, err := h.pgRepo.GetSensors()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sensors)
}

func (h *SensorHandler) PostSensorData(c *gin.Context) {
	var data model.SensorData
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if data.Timestamp.IsZero() {
		data.Timestamp = time.Now()
	}

	if err := h.influxRepo.WriteSensorData(c.Request.Context(), data.SensorID, data.Type, data.Value, data.Timestamp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
