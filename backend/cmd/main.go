package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gas-leak-monitor/internal/handler"
	"gas-leak-monitor/internal/mqtt"
	"gas-leak-monitor/internal/repository"
	"gas-leak-monitor/internal/rule"
	"gas-leak-monitor/internal/ws"

	"github.com/gin-gonic/gin"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	_ "github.com/lib/pq"
)

func main() {
	influxURL := getEnv("INFLUX_URL", "http://localhost:8086")
	influxToken := getEnv("INFLUX_TOKEN", "my-token")
	influxOrg := getEnv("INFLUX_ORG", "gas-monitor")
	influxBucket := getEnv("INFLUX_BUCKET", "gas-data")

	influxClient := influxdb2.NewClient(influxURL, influxToken)
	defer influxClient.Close()
	influxRepo := repository.NewInfluxRepo(influxClient, influxOrg, influxBucket)

	pgConnStr := getEnv("POSTGRES_URL", "postgres://postgres:postgres@localhost:5432/gas_monitor?sslmode=disable")
	pgDB, err := sql.Open("postgres", pgConnStr)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer pgDB.Close()

	if err := pgDB.Ping(); err != nil {
		log.Fatalf("Failed to ping PostgreSQL: %v", err)
	}

	pgRepo := repository.NewPostgresRepo(pgDB)
	pgRepo.InitializeSeedData()

	hub := ws.NewHub()
	go hub.Run()

	mqttBroker := getEnv("MQTT_BROKER", "tcp://localhost:1883")
	mqttClientID := getEnv("MQTT_CLIENT_ID", "gas-monitor-backend")
	mqttCli := mqtt.NewClient(mqttBroker, mqttClientID)
	mqttCli.Connect()
	defer mqttCli.Disconnect()

	engine := rule.NewEngine(pgRepo, hub, mqttCli)
	go engine.Run()

	detectorHandler := handler.NewDetectorHandler(influxRepo, pgRepo, engine)
	sensorHandler := handler.NewSensorHandler(influxRepo, pgRepo)
	alarmHandler := handler.NewAlarmHandler(pgRepo)
	leakHandler := handler.NewLeakHandler(pgRepo)
	controlHandler := handler.NewControlHandler(pgRepo, mqttCli, hub)
	wsHandler := handler.NewWSHandler(hub)

	r := gin.Default()

	api := r.Group("/api")
	{
		api.GET("/detectors", detectorHandler.GetDetectors)
		api.GET("/detectors/:id/history", detectorHandler.GetDetectorHistory)
		api.GET("/detectors/:id/health", detectorHandler.GetDetectorHealth)
		api.POST("/detectors/data", detectorHandler.PostDetectorData)

		api.GET("/sensors", sensorHandler.GetSensors)
		api.POST("/sensors/data", sensorHandler.PostSensorData)

		api.GET("/alarms", alarmHandler.GetAlarms)
		api.PUT("/alarms/:id/acknowledge", alarmHandler.AcknowledgeAlarm)
		api.PUT("/alarms/:id/resolve", alarmHandler.ResolveAlarm)

		api.POST("/leak/locate", leakHandler.LocateLeak)
		api.GET("/leak/latest", leakHandler.GetLatestLeak)

		api.POST("/control/valve", controlHandler.ControlValve)
		api.POST("/control/fan", controlHandler.ControlFan)
		api.POST("/control/notify", controlHandler.SendNotification)
		api.GET("/partitions", controlHandler.GetPartitions)
	}

	r.GET("/ws", wsHandler.HandleWebSocket)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	go func() {
		log.Println("Server starting on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
