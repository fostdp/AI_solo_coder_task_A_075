package controller

import (
	"log"
	"time"

	"gas-leak-monitor/internal/model"
	"gas-leak-monitor/internal/module"
	"gas-leak-monitor/internal/mqtt"
	"gas-leak-monitor/internal/repository"
)

type EmergencyController struct {
	pgRepo     *repository.PostgresRepo
	mqttClient *mqtt.Client
	bus        *module.MessageBus
}

func NewEmergencyController(pgRepo *repository.PostgresRepo, mqttClient *mqtt.Client, bus *module.MessageBus) *EmergencyController {
	ec := &EmergencyController{
		pgRepo:     pgRepo,
		mqttClient: mqttClient,
		bus:        bus,
	}
	go ec.consumeControlCommands()
	go ec.consumeMQTTResults()
	return ec
}

func (ec *EmergencyController) consumeControlCommands() {
	for cmd := range ec.bus.ControlCommand {
		ec.executeCommand(cmd)
	}
}

func (ec *EmergencyController) executeCommand(cmd *module.ControlCommand) {
	switch cmd.DeviceType {
	case "valve":
		cmdID := ec.mqttClient.PublishValveControl(cmd.PartitionID, cmd.Action)
		log.Printf("controller: valve %s %s, command_id=%s, source=%s", cmd.PartitionID, cmd.Action, cmdID, cmd.Source)
		ec.pgRepo.CreateControlLog(model.ControlLog{
			TargetType: "valve",
			TargetID:   cmd.PartitionID,
			Action:     cmd.Action,
			Operator:   cmd.Source,
			CreatedAt:  time.Now(),
		})
		ec.bus.WSBroadcast <- &module.WSBroadcast{
			Type: "control",
			Payload: map[string]interface{}{
				"type":       "valve",
				"id":         cmd.PartitionID,
				"action":     cmd.Action,
				"command_id": cmdID,
				"status":     "pending",
			},
		}
	case "fan":
		cmdID := ec.mqttClient.PublishFanControl(cmd.PartitionID, cmd.Action)
		log.Printf("controller: fan %s %s, command_id=%s, source=%s", cmd.PartitionID, cmd.Action, cmdID, cmd.Source)
		ec.pgRepo.CreateControlLog(model.ControlLog{
			TargetType: "fan",
			TargetID:   cmd.PartitionID,
			Action:     cmd.Action,
			Operator:   cmd.Source,
			CreatedAt:  time.Now(),
		})
		ec.bus.WSBroadcast <- &module.WSBroadcast{
			Type: "control",
			Payload: map[string]interface{}{
				"type":       "fan",
				"id":         cmd.PartitionID,
				"action":     cmd.Action,
				"command_id": cmdID,
				"status":     "pending",
			},
		}
	case "evacuation":
		ec.mqttClient.PublishEvacuation(cmd.PartitionID)
		log.Printf("controller: evacuation notification for %s, source=%s", cmd.PartitionID, cmd.Source)
		ec.pgRepo.CreateControlLog(model.ControlLog{
			TargetType: "notification",
			TargetID:   cmd.PartitionID,
			Action:     "evacuation",
			Operator:   cmd.Source,
			CreatedAt:  time.Now(),
		})
		ec.bus.WSBroadcast <- &module.WSBroadcast{
			Type: "notification",
			Payload: map[string]interface{}{
				"partition_id": cmd.PartitionID,
				"message":      "Evacuate immediately",
			},
		}
	}
}

func (ec *EmergencyController) consumeMQTTResults() {
	for result := range ec.mqttClient.Results() {
		log.Printf("controller: MQTT result: id=%s device=%s/%s status=%s retries=%d",
			result.CommandID, result.DeviceType, result.PartitionID, result.Status, result.Retries)

		ec.bus.WSBroadcast <- &module.WSBroadcast{
			Type: "control_update",
			Payload: map[string]interface{}{
				"command_id":   result.CommandID,
				"partition_id": result.PartitionID,
				"device_type":  result.DeviceType,
				"action":       result.Action,
				"status":       string(result.Status),
				"retries":      result.Retries,
				"timestamp":    result.Timestamp,
			},
		}

		if result.Status == mqtt.CommandAcked {
			if result.DeviceType == "valve" {
				status := "open"
				if result.Action == "close" {
					status = "closed"
				}
				ec.pgRepo.UpdatePartitionValve(result.PartitionID, status)
			} else if result.DeviceType == "fan" {
				status := "stopped"
				if result.Action == "start" {
					status = "running"
				}
				ec.pgRepo.UpdatePartitionFan(result.PartitionID, status)
			}
		}
	}
}

func (ec *EmergencyController) ManualValveControl(partitionID, action string) string {
	resultCh := make(chan string, 1)
	go func() {
		cmdID := ec.mqttClient.PublishValveControl(partitionID, action)
		ec.pgRepo.CreateControlLog(model.ControlLog{
			TargetType: "valve",
			TargetID:   partitionID,
			Action:     action,
			Operator:   "manual",
			CreatedAt:  time.Now(),
		})
		ec.bus.WSBroadcast <- &module.WSBroadcast{
			Type: "control",
			Payload: map[string]interface{}{
				"type":       "valve",
				"id":         partitionID,
				"action":     action,
				"command_id": cmdID,
				"status":     "pending",
			},
		}
		resultCh <- cmdID
	}()
	return <-resultCh
}

func (ec *EmergencyController) ManualFanControl(partitionID, action string) string {
	resultCh := make(chan string, 1)
	go func() {
		cmdID := ec.mqttClient.PublishFanControl(partitionID, action)
		ec.pgRepo.CreateControlLog(model.ControlLog{
			TargetType: "fan",
			TargetID:   partitionID,
			Action:     action,
			Operator:   "manual",
			CreatedAt:  time.Now(),
		})
		ec.bus.WSBroadcast <- &module.WSBroadcast{
			Type: "control",
			Payload: map[string]interface{}{
				"type":       "fan",
				"id":         partitionID,
				"action":     action,
				"command_id": cmdID,
				"status":     "pending",
			},
		}
		resultCh <- cmdID
	}()
	return <-resultCh
}

func (ec *EmergencyController) ManualNotify(partitionID, message string) {
	ec.mqttClient.Publish("notification/evacuation", 1, message)
	ec.pgRepo.CreateControlLog(model.ControlLog{
		TargetType: "notification",
		TargetID:   partitionID,
		Action:     "evacuation",
		Operator:   "manual",
		CreatedAt:  time.Now(),
	})
}

func (ec *EmergencyController) GetPartitions() ([]*model.Partition, error) {
	return ec.pgRepo.GetPartitions()
}
