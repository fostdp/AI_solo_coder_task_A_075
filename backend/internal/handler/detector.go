package handler

import (
	"net/http"
	"time"

	"gas-leak-monitor/internal/model"
	"gas-leak-monitor/internal/repository"
	"gas-leak-monitor/internal/rule"

	"github.com/gin-gonic/gin"
)

type DetectorHandler struct {
	influxRepo *repository.InfluxRepo
	pgRepo     *repository.PostgresRepo
	engine     *rule.Engine
}

func NewDetectorHandler(influxRepo *repository.InfluxRepo, pgRepo *repository.PostgresRepo, engine *rule.Engine) *DetectorHandler {
	return &DetectorHandler{
		influxRepo: influxRepo,
		pgRepo:     pgRepo,
		engine:     engine,
	}
}

type DetectorWithConcentration struct {
	*model.Detector
	Concentration float64 `json:"concentration"`
}

func (h *DetectorHandler) GetDetectors(c *gin.Context) {
	detectors, err := h.pgRepo.GetDetectors()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	concentrations, _ := h.influxRepo.QueryLatestConcentrations(c.Request.Context())

	result := make([]DetectorWithConcentration, len(detectors))
	for i, d := range detectors {
		conc := concentrations[d.ID]
		result[i] = DetectorWithConcentration{Detector: d, Concentration: conc}
	}

	c.JSON(http.StatusOK, result)
}

func (h *DetectorHandler) GetDetectorHistory(c *gin.Context) {
	id := c.Param("id")
	history, err := h.influxRepo.QueryDetectorHistory(c.Request.Context(), id, time.Hour, "1m")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, history)
}

func (h *DetectorHandler) GetDetectorHealth(c *gin.Context) {
	id := c.Param("id")
	health, err := h.pgRepo.GetDetectorHealth(id)
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

	if err := h.influxRepo.WriteDetectorData(c.Request.Context(), data.DetectorID, data.Concentration, data.Timestamp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.engine.DataChan() <- data

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
