package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	HTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gas_monitor_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gas_monitor_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	DetectorDataReceived = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "gas_monitor_detector_data_received_total",
			Help: "Total detector data points received",
		},
	)

	DetectorDataInvalid = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "gas_monitor_detector_data_invalid_total",
			Help: "Total invalid detector data points",
		},
	)

	SensorDataReceived = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "gas_monitor_sensor_data_received_total",
			Help: "Total sensor data points received",
		},
	)

	InfluxDBWriteErrors = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "gas_monitor_influxdb_write_errors_total",
			Help: "Total InfluxDB write errors",
		},
	)

	InfluxDBBatchSize = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "gas_monitor_influxdb_pending_batch_size",
			Help: "Current InfluxDB pending batch size",
		},
	)

	AlarmsTriggered = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gas_monitor_alarms_triggered_total",
			Help: "Total alarms triggered by level",
		},
		[]string{"level"},
	)

	ControlCommandsSent = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gas_monitor_control_commands_sent_total",
			Help: "Total control commands sent by type",
		},
		[]string{"type", "action"},
	)

	MQTTCommandTimeouts = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "gas_monitor_mqtt_command_timeouts_total",
			Help: "Total MQTT command timeouts",
		},
	)

	MQTTCommandAcked = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "gas_monitor_mqtt_command_acked_total",
			Help: "Total MQTT commands acknowledged",
		},
	)

	LeakLocateDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gas_monitor_leak_locate_duration_seconds",
			Help:    "Leak source localization duration",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30},
		},
		[]string{"method"},
	)

	LeakLocateConfidence = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gas_monitor_leak_locate_confidence",
			Help: "Leak source localization confidence",
		},
		[]string{"method"},
	)

	WSClientsConnected = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "gas_monitor_ws_clients_connected",
			Help: "Number of connected WebSocket clients",
		},
	)

	WSMessagesBroadcast = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "gas_monitor_ws_messages_broadcast_total",
			Help: "Total WebSocket messages broadcast",
		},
	)
)

func init() {
	prometheus.MustRegister(
		HTTPRequestsTotal,
		HTTPRequestDuration,
		DetectorDataReceived,
		DetectorDataInvalid,
		SensorDataReceived,
		InfluxDBWriteErrors,
		InfluxDBBatchSize,
		AlarmsTriggered,
		ControlCommandsSent,
		MQTTCommandTimeouts,
		MQTTCommandAcked,
		LeakLocateDuration,
		LeakLocateConfidence,
		WSClientsConnected,
		WSMessagesBroadcast,
	)
}

func PrometheusHandler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())
		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}
		HTTPRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		HTTPRequestDuration.WithLabelValues(c.Request.Method, path).Observe(duration)
	}
}
