package handler

import (
	"net/http"
	"strconv"
	"time"

	"gas-leak-monitor/internal/model"
	"gas-leak-monitor/internal/receiver"
	"gas-leak-monitor/internal/alarm"

	"github.com/gin-gonic/gin"
)

type DetectorHandler struct {
	rcv *receiver.LaserReceiver
}

func NewDetectorHandler(rcv *receiver.LaserReceiver) *DetectorHandler {
	return &DetectorHandler{rcv: rcv}
}

type DetectorWithConcentration struct {
	*model.Detector
	Concentration float64 `json:"concentration"`
}

func (h *DetectorHandler) GetDetectors(c *gin.Context) {
	detectors, err := h.rcv.GetDetectors()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	concentrations, _ := h.rcv.GetLatestConcentrations(c.Request.Context())
	result := make([]DetectorWithConcentration, len(detectors))
	for i, d := range detectors {
		conc := concentrations[d.ID]
		result[i] = DetectorWithConcentration{Detector: d, Concentration: conc}
	}
	c.JSON(http.StatusOK, result)
}

func (h *DetectorHandler) GetDetectorHistory(c *gin.Context) {
	id := c.Param("id")
	history, err := h.rcv.GetDetectorHistory(c.Request.Context(), id, time.Hour, "1m")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, history)
}

func (h *DetectorHandler) GetDetectorHealth(c *gin.Context) {
	id := c.Param("id")
	health, err := h.rcv.GetDetectorHealth(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "health data not found"})
		return
	}
	c.JSON(http.StatusOK, health)
}

func (h *DetectorHandler) PostDetectorData(c *gin.Context) {
	var data model.DetectorData
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if data.Timestamp.IsZero() {
		data.Timestamp = time.Now()
	}
	if err := h.rcv.IngestDetectorData(c.Request.Context(), data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

type SensorHandler struct {
	rcv *receiver.LaserReceiver
}

func NewSensorHandler(rcv *receiver.LaserReceiver) *SensorHandler {
	return &SensorHandler{rcv: rcv}
}

func (h *SensorHandler) GetSensors(c *gin.Context) {
	sensors, err := h.rcv.GetSensors()
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
	if err := h.rcv.IngestSensorData(c.Request.Context(), data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

type AlarmHandler struct {
	router *alarm.AlarmRouter
}

func NewAlarmHandler(router *alarm.AlarmRouter) *AlarmHandler {
	return &AlarmHandler{router: router}
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
	alarms, err := h.router.GetAlarms(level, status, page, pageSize)
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
	if err := h.router.AcknowledgeAlarm(id); err != nil {
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
	if err := h.router.ResolveAlarm(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "resolved"})
}
