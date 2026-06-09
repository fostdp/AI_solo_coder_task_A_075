package repository

import (
	"context"
	"fmt"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

type InfluxRepo struct {
	client   influxdb2.Client
	org      string
	bucket   string
	writeAPI api.WriteAPIBlocking
	queryAPI api.QueryAPI
}

func NewInfluxRepo(client influxdb2.Client, org, bucket string) *InfluxRepo {
	return &InfluxRepo{
		client:   client,
		org:      org,
		bucket:   bucket,
		writeAPI: client.WriteAPIBlocking(org, bucket),
		queryAPI: client.QueryAPI(org),
	}
}

func (r *InfluxRepo) WriteDetectorData(ctx context.Context, detectorID string, concentration float64, timestamp time.Time) error {
	p := influxdb2.NewPointWithMeasurement("detector_data").
		AddTag("detector_id", detectorID).
		AddField("concentration", concentration).
		SetTime(timestamp)
	return r.writeAPI.WritePoint(ctx, p)
}

func (r *InfluxRepo) QueryDetectorHistory(ctx context.Context, detectorID string, start time.Duration, interval string) ([]map[string]interface{}, error) {
	startTime := time.Now().Add(-start).Format(time.RFC3339Nano)
	flux := fmt.Sprintf(`
from(bucket: "%s")
  |> range(start: %s)
  |> filter(fn: (r) => r["_measurement"] == "detector_data")
  |> filter(fn: (r) => r["detector_id"] == "%s")
  |> filter(fn: (r) => r["_field"] == "concentration")
  |> aggregateWindow(every: %s, fn: mean, createEmpty: false)
  |> yield(name: "mean")
`, r.bucket, startTime, detectorID, interval)

	result, err := r.queryAPI.Query(ctx, flux)
	if err != nil {
		return nil, err
	}

	var records []map[string]interface{}
	for result.Next() {
		record := map[string]interface{}{
			"time":          result.Time(),
			"concentration": result.Value(),
			"detector_id":   result.Record().ValueByKey("detector_id"),
		}
		records = append(records, record)
	}
	if result.Err() != nil {
		return nil, result.Err()
	}
	return records, nil
}

func (r *InfluxRepo) QueryLatestConcentrations(ctx context.Context) (map[string]float64, error) {
	flux := fmt.Sprintf(`
from(bucket: "%s")
  |> range(start: -5m)
  |> filter(fn: (r) => r["_measurement"] == "detector_data")
  |> filter(fn: (r) => r["_field"] == "concentration")
  |> last()
`, r.bucket)

	result, err := r.queryAPI.Query(ctx, flux)
	if err != nil {
		return nil, err
	}

	concentrations := make(map[string]float64)
	for result.Next() {
		detectorID, ok := result.Record().ValueByKey("detector_id").(string)
		if !ok {
			continue
		}
		val := result.Value()
		if f, ok := val.(float64); ok {
			concentrations[detectorID] = f
		}
	}
	if result.Err() != nil {
		return nil, result.Err()
	}
	return concentrations, nil
}

func (r *InfluxRepo) WriteSensorData(ctx context.Context, sensorID string, sensorType string, value float64, timestamp time.Time) error {
	p := influxdb2.NewPointWithMeasurement("sensor_data").
		AddTag("sensor_id", sensorID).
		AddTag("sensor_type", sensorType).
		AddField("value", value).
		SetTime(timestamp)
	return r.writeAPI.WritePoint(ctx, p)
}

func (r *InfluxRepo) WriteWindData(ctx context.Context, speed float64, direction float64, timestamp time.Time) error {
	p := influxdb2.NewPointWithMeasurement("wind_data").
		AddField("speed", speed).
		AddField("direction", direction).
		SetTime(timestamp)
	return r.writeAPI.WritePoint(ctx, p)
}
