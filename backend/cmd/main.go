package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gas-leak-monitor/internal/alarm"
	"gas-leak-monitor/internal/config"
	"gas-leak-monitor/internal/controller"
	"gas-leak-monitor/internal/handler"
	"gas-leak-monitor/internal/locator"
	"gas-leak-monitor/internal/metrics"
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

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(metrics.MetricsMiddleware())
	r.Use(gin.Logger())

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
	r.GET("/metrics", metrics.PrometheusHandler())

	debugGroup := r.Group("/debug")
	{
		debugGroup.GET("/pprof/", gin.WrapF(pprof.Index))
		debugGroup.GET("/pprof/cmdline", gin.WrapF(pprof.Cmdline))
		debugGroup.GET("/pprof/profile", gin.WrapF(pprof.Profile))
		debugGroup.GET("/pprof/symbol", gin.WrapF(pprof.Symbol))
		debugGroup.GET("/pprof/trace", gin.WrapF(pprof.Trace))
		debugGroup.GET("/pprof/heap", gin.WrapH(pprof.Handler("heap")))
		debugGroup.GET("/pprof/goroutine", gin.WrapH(pprof.Handler("goroutine")))
		debugGroup.GET("/pprof/block", gin.WrapH(pprof.Handler("block")))
		debugGroup.GET("/pprof/mutex", gin.WrapH(pprof.Handler("mutex")))
	}

	go func() {
		log.Println("Pprof server starting on :6060")
		mux := http.NewServeMux()
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
		if err := http.ListenAndServe(":6060", mux); err != nil {
			log.Printf("Pprof server error: %v", err)
		}
	}()

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
	if url := os.Getenv("INFLUXDB_URL"); url != "" {
		cfg.InfluxDB.URL = url
	}
	if url := os.Getenv("POSTGRES_URL"); url != "" {
		cfg.Postgres.URL = url
	}
	if broker := os.Getenv("MQTT_BROKER"); broker != "" {
		cfg.MQTT.Broker = broker
	}
	return cfg
}

func runWSForwarder(hub *ws.Hub, bus *module.MessageBus) {
	for msg := range bus.WSBroadcast {
		hub.Broadcast(model.WSMessage{
			Type:    msg.Type,
			Payload: msg.Payload,
		})
		metrics.WSMessagesBroadcast.Inc()
	}
}
