package model

import "time"

type Detector struct {
	ID          string    `json:"id" db:"id"`
	PartitionID string    `json:"partition_id" db:"partition_id"`
	PositionKm  float64   `json:"position_km" db:"position_km"`
	Lat         float64   `json:"lat" db:"lat"`
	Lng         float64   `json:"lng" db:"lng"`
	Status      string    `json:"status" db:"status"`
	Type        string    `json:"type" db:"type"`
	InstalledAt time.Time `json:"installed_at" db:"installed_at"`
}

type Sensor struct {
	ID          string    `json:"id" db:"id"`
	PartitionID string    `json:"partition_id" db:"partition_id"`
	Type        string    `json:"type" db:"type"`
	Lat         float64   `json:"lat" db:"lat"`
	Lng         float64   `json:"lng" db:"lng"`
	Status      string    `json:"status" db:"status"`
	InstalledAt time.Time `json:"installed_at" db:"installed_at"`
}

type Partition struct {
	ID          string  `json:"id" db:"id"`
	Name        string  `json:"name" db:"name"`
	StartKm     float64 `json:"start_km" db:"start_km"`
	EndKm       float64 `json:"end_km" db:"end_km"`
	ValveStatus string  `json:"valve_status" db:"valve_status"`
	FanStatus   string  `json:"fan_status" db:"fan_status"`
}

type Alarm struct {
	ID             int64      `json:"id" db:"id"`
	DetectorID     string     `json:"detector_id" db:"detector_id"`
	Level          int        `json:"level" db:"level"`
	Concentration  float64    `json:"concentration" db:"concentration"`
	Status         string     `json:"status" db:"status"`
	Message        string     `json:"message" db:"message"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	AcknowledgedAt *time.Time `json:"acknowledged_at" db:"acknowledged_at"`
	ResolvedAt     *time.Time `json:"resolved_at" db:"resolved_at"`
}

type LeakEvent struct {
	ID              int64     `json:"id" db:"id"`
	SourceLat       float64   `json:"source_lat" db:"source_lat"`
	SourceLng       float64   `json:"source_lng" db:"source_lng"`
	LeakRate        float64   `json:"leak_rate" db:"leak_rate"`
	Confidence      float64   `json:"confidence" db:"confidence"`
	DiffusionRadius float64   `json:"diffusion_radius" db:"diffusion_radius"`
	Method          string    `json:"method" db:"method"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

type ControlLog struct {
	ID         int64     `json:"id" db:"id"`
	TargetType string    `json:"target_type" db:"target_type"`
	TargetID   string    `json:"target_id" db:"target_id"`
	Action     string    `json:"action" db:"action"`
	Operator   string    `json:"operator" db:"operator"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

type WindData struct {
	Speed     float64   `json:"speed"`
	Direction float64   `json:"direction"`
	Timestamp time.Time `json:"timestamp"`
}

type DetectorHealth struct {
	ID              string    `json:"id" db:"id"`
	DetectorID      string    `json:"detector_id" db:"detector_id"`
	BatteryLevel    float64   `json:"battery_level" db:"battery_level"`
	SignalStrength  float64   `json:"signal_strength" db:"signal_strength"`
	LastCalibration time.Time `json:"last_calibration" db:"last_calibration"`
	Status          string    `json:"status" db:"status"`
}

type WSMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type DetectorData struct {
	DetectorID    string    `json:"detector_id"`
	Concentration float64   `json:"concentration"`
	Timestamp     time.Time `json:"timestamp"`
}

type SensorData struct {
	SensorID  string    `json:"sensor_id"`
	Type      string    `json:"type"`
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

type LeakSourceResult struct {
	SourceLat       float64 `json:"source_lat"`
	SourceLng       float64 `json:"source_lng"`
	LeakRate        float64 `json:"leak_rate"`
	Confidence      float64 `json:"confidence"`
	DiffusionRadius float64 `json:"diffusion_radius"`
}

type LocateRequest struct {
	DetectorReadings []DetectorReading `json:"detector_readings"`
	WindSpeed        float64           `json:"wind_speed"`
	WindDirection    float64           `json:"wind_direction"`
	Method           string            `json:"method"`
}

type DetectorReading struct {
	DetectorID    string  `json:"detector_id"`
	Lat           float64 `json:"lat"`
	Lng           float64 `json:"lng"`
	Concentration float64 `json:"concentration"`
}
