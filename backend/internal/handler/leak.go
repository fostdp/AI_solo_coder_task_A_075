package handler

import (
	"net/http"
	"time"

	"gas-leak-monitor/internal/leak"
	"gas-leak-monitor/internal/model"
	"gas-leak-monitor/internal/repository"

	"github.com/gin-gonic/gin"
)

type LeakHandler struct {
	pgRepo *repository.PostgresRepo
}

func NewLeakHandler(pgRepo *repository.PostgresRepo) *LeakHandler {
	return &LeakHandler{pgRepo: pgRepo}
}

func (h *LeakHandler) LocateLeak(c *gin.Context) {
	var req model.LocateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.DetectorReadings) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no detector readings provided"})
		return
	}

	if req.WindSpeed <= 0 {
		req.WindSpeed = 1.0
	}

	var result *model.LeakSourceResult
	method := req.Method
	if method == "" {
		method = "pso"
	}

	switch method {
	case "bayesian":
		b := leak.NewBayesian(req.DetectorReadings, req.WindSpeed, req.WindDirection, req.WindTimestamp)
		result = b.Run()
		method = "bayesian"
	default:
		p := leak.NewPSO(req.DetectorReadings, req.WindSpeed, req.WindDirection, req.WindTimestamp)
		result = p.Run()
		method = "pso"
	}

	event := model.LeakEvent{
		SourceLat:       result.SourceLat,
		SourceLng:       result.SourceLng,
		LeakRate:        result.LeakRate,
		Confidence:      result.Confidence,
		DiffusionRadius: result.DiffusionRadius,
		Method:          method,
		CreatedAt:       time.Now(),
	}
	h.pgRepo.CreateLeakEvent(event)

	c.JSON(http.StatusOK, result)
}

func (h *LeakHandler) GetLatestLeak(c *gin.Context) {
	event, err := h.pgRepo.GetLatestLeakEvent()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no leak events found"})
		return
	}
	c.JSON(http.StatusOK, event)
}
