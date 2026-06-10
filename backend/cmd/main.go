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

	"gas-leak-monitor/internal/alarm"
	"gas-leak-monitor/internal/config"
	"gas-leak-monitor/internal/controller"
	"gas-leak-monitor/internal/handler"
	"gas-leak-monitor/internal/locator"
	"gas-leak-monitor/internal/model"
	"gas-leak-monitor/internal/module"
	"gas-leak-monitor/internal/mqtt"
	"gas-leak-monitor/internal/receiver"
	"gas-leak-monitor/internal/repository"
	"gas-leak-monitor/internal/ws"

	"github.com/gin-gonic/gin"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	_ "github.com/lib/pq"
)

func main() {
	cfg := loadConfig()

	influxClient := influxdb2.NewClient(cfg.InfluxDB.URL, cfg.InfluxDB.Token)
	defer influxClient.Close()
	influxRepo := repository.NewInfluxRepo(influxClient, cfg.InfluxDB.Org, cfg.InfluxDB.Bucket, cfg.InfluxDB.BatchSize, cfg.InfluxDB.FlushIntervalMs)
	defer influxRepo.Close()

	pgDB, err := sql.Open("postgres", cfg.Postgres.URL)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer pgDB.Close()
	pgDB.SetMaxOpenConns(cfg.Postgres.MaxOpenConns)
	pgDB.SetMaxIdleConns(cfg.Postgres.MaxIdleConns)
	pgDB.SetConnMaxLifetime(time.Duration(cfg.Postgres.ConnMaxLifetimeSec) * time.Second)

	if err := pgDB.Ping(); err != nil {
		log.Fatalf("Failed to ping PostgreSQL: %v", err)
	}

	pgRepo := repository.NewPostgresRepo(pgDB)
	pgRepo.InitializeSeedData()

	hub := ws.NewHub()
	go hub.Run()

	mqttClient := mqtt.NewClient(cfg.MQTT.Broker, cfg.MQTT.ClientID)
	mqttClient.Connect()
	defer mqttClient.Disconnect()

	bus := module.NewMessageBus()

	rcv := receiver.NewLaserReceiver(influxRepo, pgRepo, bus)
	loc := locator.NewLeakLocator(&cfg.Leak, bus)
	ec := controller.NewEmergencyController(pgRepo, mqttClient, bus)
	ar := alarm.NewAlarmRouter(&cfg.Alarm, pgRepo, bus)

	go loc.RunWorker()
	go runWSForwarder(hub, bus)

	detectorHandler := handler.NewDetectorHandler(rcv)
	sensorHandler := handler.NewSensorHandler(rcv)
	alarmHandler := handler.NewAlarmHandler(ar)
	leakHandler := handler.NewLeakHandler(pgRepo, loc, bus)
	controlHandler := handler.NewControlHandler(ec)
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

func loadConfig() *config.Config {
	cfgPath := "config.json"
	if p := os.Getenv("CONFIG_PATH"); p != "" {
		cfgPath = p
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Printf("Config file not found at %s, using defaults", cfgPath)
		cfg = config.Defaults()
	}
	return cfg
}

func runWSForwarder(hub *ws.Hub, bus *module.MessageBus) {
	for msg := range bus.WSBroadcast {
		hub.Broadcast(model.WSMessage{
			Type:    msg.Type,
			Payload: msg.Payload,
		})
	}
}
