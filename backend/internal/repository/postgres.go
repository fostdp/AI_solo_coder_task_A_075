package repository

import (
	"database/sql"
	"fmt"
	"math"
	"time"

	"gas-leak-monitor/internal/model"
)

type PostgresRepo struct {
	db *sql.DB
}

func NewPostgresRepo(db *sql.DB) *PostgresRepo {
	return &PostgresRepo{db: db}
}

func (r *PostgresRepo) InitializeSeedData() {
	r.createTables()
	var count int
	r.db.QueryRow("SELECT COUNT(*) FROM partitions").Scan(&count)
	if count > 0 {
		return
	}
	for i := 0; i < 30; i++ {
		id := fmt.Sprintf("P-%03d", i+1)
		name := fmt.Sprintf("Partition %d", i+1)
		startKm := float64(i)
		endKm := float64(i + 1)
		r.db.Exec(
			"INSERT INTO partitions (id, name, start_km, end_km, valve_status, fan_status) VALUES ($1, $2, $3, $4, $5, $6)",
			id, name, startKm, endKm, "open", "stopped",
		)
	}
	for i := 0; i < 300; i++ {
		id := fmt.Sprintf("D-%03d", i+1)
		km := float64(i) * 0.1
		partitionIdx := i / 10
		if partitionIdx >= 30 {
			partitionIdx = 29
		}
		partitionID := fmt.Sprintf("P-%03d", partitionIdx+1)
		lat, lng := tunnelCoords(km)
		r.db.Exec(
			"INSERT INTO detectors (id, partition_id, position_km, lat, lng, status, type, installed_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
			id, partitionID, km, lat, lng, "online", "CH4", time.Now(),
		)
	}
	sensorConfigs := []struct {
		sensorType string
		count      int
	}{
		{"O2", 20},
		{"temperature", 15},
		{"humidity", 15},
	}
	idx := 0
	for _, cfg := range sensorConfigs {
		for j := 0; j < cfg.count; j++ {
			id := fmt.Sprintf("S-%03d", idx+1)
			km := float64(j) * 30.0 / float64(cfg.count)
			partitionIdx := int(km)
			if partitionIdx >= 30 {
				partitionIdx = 29
			}
			partitionID := fmt.Sprintf("P-%03d", partitionIdx+1)
			lat, lng := tunnelCoords(km)
			r.db.Exec(
				"INSERT INTO sensors (id, partition_id, type, lat, lng, status, installed_at) VALUES ($1, $2, $3, $4, $5, $6, $7)",
				id, partitionID, cfg.sensorType, lat, lng, "online", time.Now(),
			)
			idx++
		}
	}
}

func tunnelCoords(km float64) (lat, lng float64) {
	t := km / 30.0
	lng = 121.4737 + km/95.2 + 0.002*math.Sin(t*math.Pi*3)
	lat = 31.2304 + 0.003*math.Sin(t*math.Pi*2.5) + 0.001*math.Cos(t*math.Pi*1.5)
	return
}

func (r *PostgresRepo) createTables() {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS partitions (
			id VARCHAR(10) PRIMARY KEY,
			name VARCHAR(100),
			start_km FLOAT,
			end_km FLOAT,
			valve_status VARCHAR(20) DEFAULT 'open',
			fan_status VARCHAR(20) DEFAULT 'stopped'
		)`,
		`CREATE TABLE IF NOT EXISTS detectors (
			id VARCHAR(10) PRIMARY KEY,
			partition_id VARCHAR(10) REFERENCES partitions(id),
			position_km FLOAT,
			lat FLOAT,
			lng FLOAT,
			status VARCHAR(20),
			type VARCHAR(20),
			installed_at TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS sensors (
			id VARCHAR(10) PRIMARY KEY,
			partition_id VARCHAR(10) REFERENCES partitions(id),
			type VARCHAR(20),
			lat FLOAT,
			lng FLOAT,
			status VARCHAR(20),
			installed_at TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS alarms (
			id SERIAL PRIMARY KEY,
			detector_id VARCHAR(10),
			level INT,
			concentration FLOAT,
			status VARCHAR(20) DEFAULT 'active',
			message TEXT,
			created_at TIMESTAMP DEFAULT NOW(),
			acknowledged_at TIMESTAMP,
			resolved_at TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS leak_events (
			id SERIAL PRIMARY KEY,
			source_lat FLOAT,
			source_lng FLOAT,
			leak_rate FLOAT,
			confidence FLOAT,
			diffusion_radius FLOAT,
			method VARCHAR(20),
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS control_logs (
			id SERIAL PRIMARY KEY,
			target_type VARCHAR(20),
			target_id VARCHAR(10),
			action VARCHAR(20),
			operator VARCHAR(50),
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS detector_health (
			id VARCHAR(10) PRIMARY KEY,
			detector_id VARCHAR(10) REFERENCES detectors(id),
			battery_level FLOAT,
			signal_strength FLOAT,
			last_calibration TIMESTAMP,
			status VARCHAR(20)
		)`,
	}
	for _, q := range queries {
		r.db.Exec(q)
	}
}

func (r *PostgresRepo) GetDetectors() ([]*model.Detector, error) {
	rows, err := r.db.Query("SELECT id, partition_id, position_km, lat, lng, status, type, installed_at FROM detectors ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var detectors []*model.Detector
	for rows.Next() {
		d := &model.Detector{}
		err := rows.Scan(&d.ID, &d.PartitionID, &d.PositionKm, &d.Lat, &d.Lng, &d.Status, &d.Type, &d.InstalledAt)
		if err != nil {
			return nil, err
		}
		detectors = append(detectors, d)
	}
	return detectors, nil
}

func (r *PostgresRepo) GetDetectorByID(id string) (*model.Detector, error) {
	d := &model.Detector{}
	err := r.db.QueryRow(
		"SELECT id, partition_id, position_km, lat, lng, status, type, installed_at FROM detectors WHERE id = $1",
		id,
	).Scan(&d.ID, &d.PartitionID, &d.PositionKm, &d.Lat, &d.Lng, &d.Status, &d.Type, &d.InstalledAt)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (r *PostgresRepo) GetSensors() ([]*model.Sensor, error) {
	rows, err := r.db.Query("SELECT id, partition_id, type, lat, lng, status, installed_at FROM sensors ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sensors []*model.Sensor
	for rows.Next() {
		s := &model.Sensor{}
		err := rows.Scan(&s.ID, &s.PartitionID, &s.Type, &s.Lat, &s.Lng, &s.Status, &s.InstalledAt)
		if err != nil {
			return nil, err
		}
		sensors = append(sensors, s)
	}
	return sensors, nil
}

func (r *PostgresRepo) GetPartitions() ([]*model.Partition, error) {
	rows, err := r.db.Query("SELECT id, name, start_km, end_km, valve_status, fan_status FROM partitions ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var partitions []*model.Partition
	for rows.Next() {
		p := &model.Partition{}
		err := rows.Scan(&p.ID, &p.Name, &p.StartKm, &p.EndKm, &p.ValveStatus, &p.FanStatus)
		if err != nil {
			return nil, err
		}
		partitions = append(partitions, p)
	}
	return partitions, nil
}

func (r *PostgresRepo) GetPartitionByID(id string) (*model.Partition, error) {
	p := &model.Partition{}
	err := r.db.QueryRow(
		"SELECT id, name, start_km, end_km, valve_status, fan_status FROM partitions WHERE id = $1",
		id,
	).Scan(&p.ID, &p.Name, &p.StartKm, &p.EndKm, &p.ValveStatus, &p.FanStatus)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (r *PostgresRepo) UpdatePartitionValve(partitionID string, status string) error {
	_, err := r.db.Exec("UPDATE partitions SET valve_status = $1 WHERE id = $2", status, partitionID)
	return err
}

func (r *PostgresRepo) UpdatePartitionFan(partitionID string, status string) error {
	_, err := r.db.Exec("UPDATE partitions SET fan_status = $1 WHERE id = $2", status, partitionID)
	return err
}

func (r *PostgresRepo) CreateAlarm(alarm model.Alarm) error {
	_, err := r.db.Exec(
		"INSERT INTO alarms (detector_id, level, concentration, status, message, created_at) VALUES ($1, $2, $3, $4, $5, $6)",
		alarm.DetectorID, alarm.Level, alarm.Concentration, alarm.Status, alarm.Message, alarm.CreatedAt,
	)
	return err
}

func (r *PostgresRepo) GetAlarms(level int, status string, page, pageSize int) ([]model.Alarm, error) {
	query := "SELECT id, detector_id, level, concentration, status, message, created_at, acknowledged_at, resolved_at FROM alarms WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if level > 0 {
		query += fmt.Sprintf(" AND level = $%d", argIdx)
		args = append(args, level)
		argIdx++
	}
	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}

	offset := (page - 1) * pageSize
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, pageSize, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alarms []model.Alarm
	for rows.Next() {
		var a model.Alarm
		err := rows.Scan(&a.ID, &a.DetectorID, &a.Level, &a.Concentration, &a.Status, &a.Message, &a.CreatedAt, &a.AcknowledgedAt, &a.ResolvedAt)
		if err != nil {
			return nil, err
		}
		alarms = append(alarms, a)
	}
	return alarms, nil
}

func (r *PostgresRepo) AcknowledgeAlarm(id int64) error {
	now := time.Now()
	_, err := r.db.Exec(
		"UPDATE alarms SET status = 'acknowledged', acknowledged_at = $1 WHERE id = $2",
		now, id,
	)
	return err
}

func (r *PostgresRepo) ResolveAlarm(id int64) error {
	now := time.Now()
	_, err := r.db.Exec(
		"UPDATE alarms SET status = 'resolved', resolved_at = $1 WHERE id = $2",
		now, id,
	)
	return err
}

func (r *PostgresRepo) CreateLeakEvent(event model.LeakEvent) error {
	_, err := r.db.Exec(
		"INSERT INTO leak_events (source_lat, source_lng, leak_rate, confidence, diffusion_radius, method, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		event.SourceLat, event.SourceLng, event.LeakRate, event.Confidence, event.DiffusionRadius, event.Method, event.CreatedAt,
	)
	return err
}

func (r *PostgresRepo) GetLatestLeakEvent() (*model.LeakEvent, error) {
	e := &model.LeakEvent{}
	err := r.db.QueryRow(
		"SELECT id, source_lat, source_lng, leak_rate, confidence, diffusion_radius, method, created_at FROM leak_events ORDER BY created_at DESC LIMIT 1",
	).Scan(&e.ID, &e.SourceLat, &e.SourceLng, &e.LeakRate, &e.Confidence, &e.DiffusionRadius, &e.Method, &e.CreatedAt)
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (r *PostgresRepo) CreateControlLog(log model.ControlLog) error {
	_, err := r.db.Exec(
		"INSERT INTO control_logs (target_type, target_id, action, operator, created_at) VALUES ($1, $2, $3, $4, $5)",
		log.TargetType, log.TargetID, log.Action, log.Operator, log.CreatedAt,
	)
	return err
}

func (r *PostgresRepo) GetDetectorHealth(detectorID string) (*model.DetectorHealth, error) {
	h := &model.DetectorHealth{}
	err := r.db.QueryRow(
		"SELECT id, detector_id, battery_level, signal_strength, last_calibration, status FROM detector_health WHERE detector_id = $1",
		detectorID,
	).Scan(&h.ID, &h.DetectorID, &h.BatteryLevel, &h.SignalStrength, &h.LastCalibration, &h.Status)
	if err != nil {
		return nil, err
	}
	return h, nil
}
