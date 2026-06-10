package receiver

import (
	"context"
	"log"
	"math"
	"time"

	"gas-leak-monitor/internal/model"
	"gas-leak-monitor/internal/module"
	"gas-leak-monitor/internal/repository"
)

type LaserReceiver struct {
	influxRepo  *repository.InfluxRepo
	pgRepo      *repository.PostgresRepo
	bus         *module.MessageBus
	detectorMap map[string]*model.Detector
	sensorMap   map[string]*model.Sensor
}

func NewLaserReceiver(influxRepo *repository.InfluxRepo, pgRepo *repository.PostgresRepo, bus *module.MessageBus) *LaserReceiver {
	r := &LaserReceiver{
		influxRepo:  influxRepo,
		pgRepo:      pgRepo,
		bus:         bus,
		detectorMap: make(map[string]*model.Detector),
		sensorMap:   make(map[string]*model.Sensor),
	}
	r.loadDevices()
	return r
}

func (r *LaserReceiver) loadDevices() {
	detectors, err := r.pgRepo.GetDetectors()
	if err != nil {
		log.Printf("receiver: failed to load detectors: %v", err)
		return
	}
	for _, d := range detectors {
		r.detectorMap[d.ID] = d
	}
	sensors, err := r.pgRepo.GetSensors()
	if err != nil {
		log.Printf("receiver: failed to load sensors: %v", err)
		return
	}
	for _, s := range sensors {
		r.sensorMap[s.ID] = s
	}
}

func (r *LaserReceiver) IngestDetectorData(ctx context.Context, data model.DetectorData) error {
	validated := r.validateDetectorData(data)
	if validated.Valid {
		if err := r.influxRepo.WriteDetectorData(ctx, data.DetectorID, data.Concentration, data.Timestamp); err != nil {
			log.Printf("receiver: influx write error for %s: %v", data.DetectorID, err)
		}
	}
	select {
	case r.bus.ValidatedDetectorData <- validated:
	default:
		log.Printf("receiver: detector data channel full, dropping %s", data.DetectorID)
	}
	return nil
}

func (r *LaserReceiver) IngestSensorData(ctx context.Context, data model.SensorData) error {
	validated := r.validateSensorData(data)
	if validated.Valid {
		if err := r.influxRepo.WriteSensorData(ctx, data.SensorID, data.Type, data.Value, data.Timestamp); err != nil {
			log.Printf("receiver: influx write error for sensor %s: %v", data.SensorID, err)
		}
	}
	select {
	case r.bus.ValidatedSensorData <- validated:
	default:
		log.Printf("receiver: sensor data channel full, dropping %s", data.SensorID)
	}
	return nil
}

func (r *LaserReceiver) validateDetectorData(data model.DetectorData) *module.ValidatedDetectorData {
	result := &module.ValidatedDetectorData{
		DetectorID:    data.DetectorID,
		Concentration: data.Concentration,
		Timestamp:     data.Timestamp,
		Valid:         true,
	}
	if data.Timestamp.IsZero() {
		result.Timestamp = time.Now()
	}
	if data.DetectorID == "" {
		result.Valid = false
		result.Reason = "empty_detector_id"
		return result
	}
	if _, exists := r.detectorMap[data.DetectorID]; !exists {
		result.Valid = false
		result.Reason = "unknown_detector"
		return result
	}
	if data.Concentration < 0 {
		result.Valid = false
		result.Reason = "negative_concentration"
		return result
	}
	if data.Concentration > 100 {
		result.Valid = false
		result.Reason = "concentration_exceeds_100"
		return result
	}
	if math.IsNaN(data.Concentration) || math.IsInf(data.Concentration, 0) {
		result.Valid = false
		result.Reason = "invalid_concentration_value"
		return result
	}
	return result
}

func (r *LaserReceiver) validateSensorData(data model.SensorData) *module.ValidatedSensorData {
	result := &module.ValidatedSensorData{
		SensorID:  data.SensorID,
		Type:      data.Type,
		Value:     data.Value,
		Timestamp: data.Timestamp,
		Valid:     true,
	}
	if data.Timestamp.IsZero() {
		result.Timestamp = time.Now()
	}
	if data.SensorID == "" {
		result.Valid = false
		result.Reason = "empty_sensor_id"
		return result
	}
	if _, exists := r.sensorMap[data.SensorID]; !exists {
		result.Valid = false
		result.Reason = "unknown_sensor"
		return result
	}
	if math.IsNaN(data.Value) || math.IsInf(data.Value, 0) {
		result.Valid = false
		result.Reason = "invalid_value"
		return result
	}
	return result
}

func (r *LaserReceiver) GetDetectors() ([]*model.Detector, error) {
	return r.pgRepo.GetDetectors()
}

func (r *LaserReceiver) GetSensors() ([]*model.Sensor, error) {
	return r.pgRepo.GetSensors()
}

func (r *LaserReceiver) GetDetectorHistory(ctx context.Context, id string, dur time.Duration, interval string) ([]map[string]interface{}, error) {
	return r.influxRepo.QueryDetectorHistory(ctx, id, dur, interval)
}

func (r *LaserReceiver) GetDetectorHealth(id string) (*model.DetectorHealth, error) {
	return r.pgRepo.GetDetectorHealth(id)
}

func (r *LaserReceiver) GetLatestConcentrations(ctx context.Context) (map[string]float64, error) {
	return r.influxRepo.QueryLatestConcentrations(ctx)
}
