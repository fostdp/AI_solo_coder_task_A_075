package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	Server   ServerConfig   `json:"server"`
	InfluxDB InfluxDBConfig `json:"influxdb"`
	Postgres PostgresConfig `json:"postgres"`
	MQTT     MQTTConfig     `json:"mqtt"`
	Alarm    AlarmConfig    `json:"alarm"`
	Leak     LeakConfig     `json:"leak"`
}

type ServerConfig struct {
	Port int `json:"port"`
}

type InfluxDBConfig struct {
	URL             string `json:"url"`
	Token           string `json:"token"`
	Org             string `json:"org"`
	Bucket          string `json:"bucket"`
	BatchSize       int    `json:"batch_size"`
	FlushIntervalMs int    `json:"flush_interval_ms"`
}

type PostgresConfig struct {
	URL                string `json:"url"`
	MaxOpenConns       int    `json:"max_open_conns"`
	MaxIdleConns       int    `json:"max_idle_conns"`
	ConnMaxLifetimeSec int    `json:"conn_max_lifetime_sec"`
}

type MQTTConfig struct {
	Broker           string `json:"broker"`
	ClientID         string `json:"client_id"`
	AckTimeoutSec    int    `json:"ack_timeout_sec"`
	MaxRetries       int    `json:"max_retries"`
	RetryIntervalSec int    `json:"retry_interval_sec"`
}

type AlarmConfig struct {
	Level1Threshold float64 `json:"level1_threshold"`
	Level2Threshold float64 `json:"level2_threshold"`
	Level3Threshold float64 `json:"level3_threshold"`
}

type LeakConfig struct {
	PSO       PSOConfig       `json:"pso"`
	Bayesian  BayesianConfig  `json:"bayesian"`
	Wind      WindConfig      `json:"wind"`
	Diffusion DiffusionConfig `json:"diffusion"`
	Gaussian  GaussianConfig  `json:"gaussian"`
}

type PSOConfig struct {
	NumParticles    int     `json:"num_particles"`
	MaxIterations   int     `json:"max_iterations"`
	InertiaWeight   float64 `json:"inertia_weight"`
	CognitiveCoeff  float64 `json:"cognitive_coeff"`
	SocialCoeff     float64 `json:"social_coeff"`
	Margin          float64 `json:"margin"`
	RateMin         float64 `json:"rate_min"`
	RateMax         float64 `json:"rate_max"`
	MaxVRate        float64 `json:"max_vrate"`
}

type BayesianConfig struct {
	GridResolution       int     `json:"grid_resolution"`
	RateMin              float64 `json:"rate_min"`
	RateMax              float64 `json:"rate_max"`
	RateStep             float64 `json:"rate_step"`
	Margin               float64 `json:"margin"`
	SigmaBase            float64 `json:"sigma_base"`
	SigmaPerQualityLevel float64 `json:"sigma_per_quality_level"`
}

type WindConfig struct {
	StaleThresholdMin       int     `json:"stale_threshold_min"`
	StaleSpeedFactor        float64 `json:"stale_speed_factor"`
	GradientBlendWeight     float64 `json:"gradient_blend_weight"`
	FallbackSpeedMin        float64 `json:"fallback_speed_min"`
	FallbackSpeedMax        float64 `json:"fallback_speed_max"`
	FallbackSpeedSpreadDiv  float64 `json:"fallback_speed_spread_div"`
	DefaultDirection        float64 `json:"default_direction"`
	DefaultSpeed            float64 `json:"default_speed"`
}

type DiffusionConfig struct {
	RadiusFactor          float64 `json:"radius_factor"`
	StaleMultiplier       float64 `json:"stale_multiplier"`
	UnavailableMultiplier float64 `json:"unavailable_multiplier"`
}

type GaussianConfig struct {
	MPPerDegLat  float64 `json:"m_per_deg_lat"`
	MPPerDegLng  float64 `json:"m_per_deg_lng"`
	SigmaYFactor float64 `json:"sigma_y_factor"`
	SigmaZFactor float64 `json:"sigma_z_factor"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func Defaults() *Config {
	return &Config{
		Server: ServerConfig{
			Port: 8080,
		},
		InfluxDB: InfluxDBConfig{
			URL:             "http://localhost:8086",
			Token:           "my-token",
			Org:             "gas-monitor",
			Bucket:          "gas-data",
			BatchSize:       500,
			FlushIntervalMs: 500,
		},
		Postgres: PostgresConfig{
			URL:                "postgres://postgres:postgres@localhost:5432/gas_monitor?sslmode=disable",
			MaxOpenConns:       25,
			MaxIdleConns:       10,
			ConnMaxLifetimeSec: 300,
		},
		MQTT: MQTTConfig{
			Broker:           "tcp://localhost:1883",
			ClientID:         "gas-monitor-backend",
			AckTimeoutSec:    5,
			MaxRetries:       3,
			RetryIntervalSec: 2,
		},
		Alarm: AlarmConfig{
			Level1Threshold: 10.0,
			Level2Threshold: 20.0,
			Level3Threshold: 50.0,
		},
		Leak: LeakConfig{
			PSO: PSOConfig{
				NumParticles:   50,
				MaxIterations:  100,
				InertiaWeight:  0.7,
				CognitiveCoeff: 1.5,
				SocialCoeff:    1.5,
				Margin:         0.005,
				RateMin:        0.01,
				RateMax:        50.0,
				MaxVRate:       5.0,
			},
			Bayesian: BayesianConfig{
				GridResolution:       50,
				RateMin:              0.1,
				RateMax:              100.0,
				RateStep:             0.5,
				Margin:               0.005,
				SigmaBase:            1.0,
				SigmaPerQualityLevel: 0.5,
			},
			Wind: WindConfig{
				StaleThresholdMin:      10,
				StaleSpeedFactor:       0.7,
				GradientBlendWeight:    0.3,
				FallbackSpeedMin:       0.5,
				FallbackSpeedMax:       5.0,
				FallbackSpeedSpreadDiv: 500.0,
				DefaultDirection:       90.0,
				DefaultSpeed:           1.0,
			},
			Diffusion: DiffusionConfig{
				RadiusFactor:          10.0,
				StaleMultiplier:       1.25,
				UnavailableMultiplier: 1.5,
			},
			Gaussian: GaussianConfig{
				MPPerDegLat:  111320.0,
				MPPerDegLng:  95200.0,
				SigmaYFactor: 0.1,
				SigmaZFactor: 0.05,
			},
		},
	}
}
