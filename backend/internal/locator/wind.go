package locator

import (
	"math"
	"time"

	"gas-leak-monitor/internal/config"
	"gas-leak-monitor/internal/model"
)

type WindQuality int

const (
	WindQualityGood        WindQuality = 0
	WindQualityStale       WindQuality = 1
	WindQualityUnavailable WindQuality = 2
)

type WindAssessment struct {
	Quality   WindQuality
	Speed     float64
	Direction float64
	Staleness time.Duration
}

type WindEstimator struct {
	cfg config.WindConfig
}

func NewWindEstimator(cfg config.WindConfig) *WindEstimator {
	return &WindEstimator{cfg: cfg}
}

func (e *WindEstimator) AssessQuality(windSpeed, windDir float64, windTimestamp time.Time) WindAssessment {
	staleness := time.Since(windTimestamp)
	if windSpeed < 0 || windDir < 0 || windDir > 360 || windTimestamp.IsZero() {
		return WindAssessment{
			Quality:   WindQualityUnavailable,
			Speed:     0,
			Direction: 0,
			Staleness: staleness,
		}
	}
	if staleness > time.Duration(e.cfg.StaleThresholdMin)*time.Minute {
		return WindAssessment{
			Quality:   WindQualityStale,
			Speed:     windSpeed,
			Direction: windDir,
			Staleness: staleness,
		}
	}
	return WindAssessment{
		Quality:   WindQualityGood,
		Speed:     windSpeed,
		Direction: windDir,
		Staleness: staleness,
	}
}

func (e *WindEstimator) SelectEffective(wa WindAssessment, readings []model.DetectorReading, gauss *GaussianParams) (float64, float64) {
	switch wa.Quality {
	case WindQualityGood:
		return wa.Speed, wa.Direction
	case WindQualityStale:
		concentrationWeightedDir := e.estimateWindFromGradient(readings, gauss)
		blendedDir := blendDirections(wa.Direction, concentrationWeightedDir, e.cfg.GradientBlendWeight)
		return wa.Speed * e.cfg.StaleSpeedFactor, blendedDir
	case WindQualityUnavailable:
		fallBackDir := e.estimateWindFromGradient(readings, gauss)
		fallBackSpeed := e.estimateSpeedFromSpread(readings, gauss)
		return fallBackSpeed, fallBackDir
	}
	return e.cfg.DefaultSpeed, e.cfg.DefaultDirection
}

func (e *WindEstimator) estimateWindFromGradient(readings []model.DetectorReading, gauss *GaussianParams) float64 {
	if len(readings) < 2 {
		return e.cfg.DefaultDirection
	}
	maxConc := 0.0
	maxLat, maxLng := 0.0, 0.0
	totalConc, totalLat, totalLng := 0.0, 0.0, 0.0
	for _, r := range readings {
		if r.Concentration > maxConc {
			maxConc = r.Concentration
			maxLat = r.Lat
			maxLng = r.Lng
		}
		totalConc += r.Concentration
		totalLat += r.Lat * r.Concentration
		totalLng += r.Lng * r.Concentration
	}
	if totalConc == 0 {
		return e.cfg.DefaultDirection
	}
	centerLat := totalLat / totalConc
	centerLng := totalLng / totalConc
	dx := (centerLng - maxLng) * gauss.MPPerDegLng
	dy := (centerLat - maxLat) * gauss.MPPerDegLat
	angle := math.Atan2(dy, dx) * 180.0 / math.Pi
	if angle < 0 {
		angle += 360
	}
	return angle
}

func (e *WindEstimator) estimateSpeedFromSpread(readings []model.DetectorReading, gauss *GaussianParams) float64 {
	if len(readings) < 2 {
		return e.cfg.DefaultSpeed
	}
	minLat, maxLat := readings[0].Lat, readings[0].Lat
	minLng, maxLng := readings[0].Lng, readings[0].Lng
	for _, r := range readings {
		if r.Lat < minLat {
			minLat = r.Lat
		}
		if r.Lat > maxLat {
			maxLat = r.Lat
		}
		if r.Lng < minLng {
			minLng = r.Lng
		}
		if r.Lng > maxLng {
			maxLng = r.Lng
		}
	}
	spreadM := math.Sqrt(math.Pow((maxLat-minLat)*gauss.MPPerDegLat, 2)+math.Pow((maxLng-minLng)*gauss.MPPerDegLng, 2))
	estimatedSpeed := e.cfg.FallbackSpeedMin + spreadM/e.cfg.FallbackSpeedSpreadDiv
	if estimatedSpeed > e.cfg.FallbackSpeedMax {
		estimatedSpeed = e.cfg.FallbackSpeedMax
	}
	return estimatedSpeed
}

func blendDirections(d1, d2 float64, w2 float64) float64 {
	w1 := 1.0 - w2
	r1 := d1 * math.Pi / 180.0
	r2 := d2 * math.Pi / 180.0
	x := w1*math.Cos(r1) + w2*math.Cos(r2)
	y := w1*math.Sin(r1) + w2*math.Sin(r2)
	result := math.Atan2(y, x) * 180.0 / math.Pi
	if result < 0 {
		result += 360
	}
	return result
}
