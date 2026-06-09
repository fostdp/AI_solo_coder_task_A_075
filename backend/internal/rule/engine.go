package rule

import (
	"fmt"
	"log"
	"time"

	"gas-leak-monitor/internal/model"
	"gas-leak-monitor/internal/mqtt"
	"gas-leak-monitor/internal/repository"
	"gas-leak-monitor/internal/ws"
)

const (
	Level1Threshold = 10.0
	Level2Threshold = 20.0
	Level3Threshold = 50.0
)

type Engine struct {
	pgRepo     *repository.PostgresRepo
	hub        *ws.Hub
	mqttClient *mqtt.Client
	dataChan   chan model.DetectorData
}

func NewEngine(pgRepo *repository.PostgresRepo, hub *ws.Hub, mqttClient *mqtt.Client) *Engine {
	return &Engine{
		pgRepo:     pgRepo,
		hub:        hub,
		mqttClient: mqttClient,
		dataChan:   make(chan model.DetectorData, 100),
	}
}

func (e *Engine) DataChan() chan<- model.DetectorData {
	return e.dataChan
}

func (e *Engine) Run() {
	for data := range e.dataChan {
		e.evaluate(data)
	}
}

func (e *Engine) evaluate(data model.DetectorData) {
	var level int
	var message string

	if data.Concentration > Level3Threshold {
		level = 3
		message = "Critical gas leak detected"
	} else if data.Concentration > Level2Threshold {
		level = 2
		message = "High gas concentration detected"
	} else if data.Concentration > Level1Threshold {
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
	err := e.pgRepo.CreateAlarm(alarm)
	if err != nil {
		log.Printf("Failed to create alarm: %v", err)
	}

	e.mqttClient.PublishAlarm(level, fmt.Sprintf("%s at %s (%.2f%%LEL)", message, data.DetectorID, data.Concentration))

	log.Printf("[SMS] Alarm Level %d: %s at detector %s (concentration: %.2f%%LEL)", level, message, data.DetectorID, data.Concentration)

	if level >= 2 {
		detector, err := e.pgRepo.GetDetectorByID(data.DetectorID)
		if err == nil && detector != nil {
			e.mqttClient.PublishValveControl(detector.PartitionID, "close")
			e.pgRepo.UpdatePartitionValve(detector.PartitionID, "closed")
			e.pgRepo.CreateControlLog(model.ControlLog{
				TargetType: "valve",
				TargetID:   detector.PartitionID,
				Action:     "close",
				Operator:   "auto",
				CreatedAt:  time.Now(),
			})

			e.mqttClient.PublishFanControl(detector.PartitionID, "start")
			e.pgRepo.UpdatePartitionFan(detector.PartitionID, "running")
			e.pgRepo.CreateControlLog(model.ControlLog{
				TargetType: "fan",
				TargetID:   detector.PartitionID,
				Action:     "start",
				Operator:   "auto",
				CreatedAt:  time.Now(),
			})

			e.mqttClient.PublishEvacuation(detector.PartitionID)
		}
	}

	e.hub.Broadcast(model.WSMessage{
		Type:    "alarm",
		Payload: alarm,
	})

	if level >= 2 {
		e.hub.Broadcast(model.WSMessage{
			Type: "control",
			Payload: map[string]interface{}{
				"action": "auto_response",
				"detail": fmt.Sprintf("Valve closed, fan started for partition of detector %s", data.DetectorID),
			},
		})
	}
}
