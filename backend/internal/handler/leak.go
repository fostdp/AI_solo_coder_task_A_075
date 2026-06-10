package handler

import (
	"net/http"
	"time"

	"gas-leak-monitor/internal/model"
	"gas-leak-monitor/internal/locator"
	"gas-leak-monitor/internal/module"
	"gas-leak-monitor/internal/repository"

	"github.com/gin-gonic/gin"
)

type LeakHandler struct {
	pgRepo  *repository.PostgresRepo
	locator *locator.LeakLocator
	bus     *module.MessageBus
}

func NewLeakHandler(pgRepo *repository.PostgresRepo, loc *locator.LeakLocator, bus *module.MessageBus) *LeakHandler {
	return &LeakHandler{pgRepo: pgRepo, locator: loc, bus: bus}
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

	resultChan := make(chan *model.LeakSourceResult, 1)
	h.bus.LocateRequest <- &module.LocateRequestMsg{
		Readings:      req.DetectorReadings,
		WindSpeed:     req.WindSpeed,
		WindDirection: req.WindDirection,
		WindTimestamp:  req.WindTimestamp,
		Method:        req.Method,
		ResultChan:    resultChan,
	}

	result := <-resultChan
	if result == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "localization failed"})
		return
	}

	method := req.Method
	if method == "" {
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
