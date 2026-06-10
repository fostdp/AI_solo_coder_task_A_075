package alarm

import (
	"fmt"
	"log"
	"time"

	"gas-leak-monitor/internal/config"
	"gas-leak-monitor/internal/model"
	"gas-leak-monitor/internal/module"
	"gas-leak-monitor/internal/repository"
)

type AlarmRouter struct {
	cfg       *config.AlarmConfig
	pgRepo    *repository.PostgresRepo
	bus       *module.MessageBus
	detectorMap map[string]*model.Detector
}

func NewAlarmRouter(cfg *config.AlarmConfig, pgRepo *repository.PostgresRepo, bus *module.MessageBus) *AlarmRouter {
	ar := &AlarmRouter{
		cfg:       cfg,
		pgRepo:    pgRepo,
		bus:       bus,
		detectorMap: make(map[string]*model.Detector),
	}
	ar.loadDetectors()
	go ar.run()
	return ar
}

func (ar *AlarmRouter) loadDetectors() {
	detectors, err := ar.pgRepo.GetDetectors()
	if err != nil {
		log.Printf("alarm_router: failed to load detectors: %v", err)
		return
	}
	for _, d := range detectors {
		ar.detectorMap[d.ID] = d
	}
}

func (ar *AlarmRouter) run() {
	for data := range ar.bus.ValidatedDetectorData {
		if !data.Valid {
			continue
		}
		ar.evaluate(data)
	}
}

func (ar *AlarmRouter) evaluate(data *module.ValidatedDetectorData) {
	var level int
	var message string

	if data.Concentration > ar.cfg.Level3Threshold {
		level = 3
		message = "Critical gas leak detected"
	} else if data.Concentration > ar.cfg.Level2Threshold {
		level = 2
		message = "High gas concentration detected"
	} else if data.Concentration > ar.cfg.Level1Threshold {
		level = 1
		message = "Elevated gas concentration detected"
	} else {
		return
	}

	alarm := model.Alarm{
		DetectorID:    data.DetectorID,
		Level:         level,
		Concentration: data.Concentration,
		Status:        "active",
		Message:       message,
		CreatedAt:     time.Now(),
	}
	if err := ar.pgRepo.CreateAlarm(alarm); err != nil {
		log.Printf("alarm_router: failed to create alarm: %v", err)
	}

	ar.bus.WSBroadcast <- &module.WSBroadcast{
		Type:    "alarm",
		Payload: alarm,
	}

	ar.bus.WSBroadcast <- &module.WSBroadcast{
		Type: "detector_update",
		Payload: map[string]interface{}{
			"detector_id":   data.DetectorID,
			"concentration": data.Concentration,
			"timestamp":     data.Timestamp,
		},
	}

	if level >= 2 {
		detector, exists := ar.detectorMap[data.DetectorID]
		if !exists {
			log.Printf("alarm_router: detector %s not found for auto-response", data.DetectorID)
			return
		}

		ar.bus.ControlCommand <- &module.ControlCommand{
			DeviceType:  "valve",
			PartitionID: detector.PartitionID,
			Action:      "close",
			Source:       "auto",
		}

		ar.bus.ControlCommand <- &module.ControlCommand{
			DeviceType:  "fan",
			PartitionID: detector.PartitionID,
			Action:      "start",
			Source:       "auto",
		}

		ar.bus.ControlCommand <- &module.ControlCommand{
			DeviceType:  "evacuation",
			PartitionID: detector.PartitionID,
			Action:      "evacuate",
			Source:       "auto",
		}

		ar.bus.WSBroadcast <- &module.WSBroadcast{
			Type: "control",
			Payload: map[string]interface{}{
				"action": "auto_response",
				"detail": fmt.Sprintf("Valve closed, fan started for partition of detector %s", data.DetectorID),
			},
		}
	}

	log.Printf("[SMS] Alarm Level %d: %s at detector %s (concentration: %.2f%%LEL)", level, message, data.DetectorID, data.Concentration)
}

func (ar *AlarmRouter) GetAlarms(level int, status string, page, pageSize int) ([]model.Alarm, error) {
	return ar.pgRepo.GetAlarms(level, status, page, pageSize)
}

func (ar *AlarmRouter) AcknowledgeAlarm(id int64) error {
	return ar.pgRepo.AcknowledgeAlarm(id)
}

func (ar *AlarmRouter) ResolveAlarm(id int64) error {
	return ar.pgRepo.ResolveAlarm(id)
}
