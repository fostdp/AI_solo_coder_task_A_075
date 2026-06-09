package leak

import (
	"math"
	"time"

	"gas-leak-monitor/internal/model"
)

type WindQuality int

const (
	WindQualityGood      WindQuality = iota
	WindQualityStale
	WindQualityUnavailable
)

type WindAssessment struct {
	Quality     WindQuality
	Speed       float64
	Direction   float64
	Staleness   time.Duration
}

func AssessWindQuality(windSpeed, windDir float64, windTimestamp time.Time) WindAssessment {
	staleness := time.Since(windTimestamp)

	if windSpeed < 0 || windDir < 0 || windDir > 360 || windTimestamp.IsZero() {
		return WindAssessment{
			Quality:   WindQualityUnavailable,
			Speed:     0,
			Direction: 0,
			Staleness: staleness,
		}
	}

	if staleness > 10*time.Minute {
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

func selectEffectiveWind(wa WindAssessment, readings []model.DetectorReading) (float64, float64) {
	switch wa.Quality {
	case WindQualityGood:
		return wa.Speed, wa.Direction

	case WindQualityStale:
		concentrationWeightedDir := estimateWindFromGradient(readings)
		blendedDir := blendDirections(wa.Direction, concentrationWeightedDir, 0.3)
		return wa.Speed * 0.7, blendedDir

	case WindQualityUnavailable:
		fallBackDir := estimateWindFromGradient(readings)
		fallBackSpeed := estimateSpeedFromSpread(readings)
		return fallBackSpeed, fallBackDir
	}

	return 1.0, 0.0
}

func estimateWindFromGradient(readings []model.DetectorReading) float64 {
	if len(readings) < 2 {
		return 90.0
	}

	maxConc := 0.0
	maxLat := 0.0
	maxLng := 0.0
	totalConc := 0.0
	totalLat := 0.0
	totalLng := 0.0

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
		return 90.0
	}

	centerLat := totalLat / totalConc
	centerLng := totalLng / totalConc

	dx := (centerLng - maxLng) * mPerDegLng
	dy := (centerLat - maxLat) * mPerDegLat

	angle := math.Atan2(dy, dx) * 180.0 / math.Pi
	if angle < 0 {
		angle += 360
	}

	return angle
}

func estimateSpeedFromSpread(readings []model.DetectorReading) float64 {
	if len(readings) < 2 {
		return 1.5
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

	spreadM := math.Sqrt(math.Pow((maxLat-minLat)*mPerDegLat, 2) + math.Pow((maxLng-minLng)*mPerDegLng, 2))
	estimatedSpeed := 0.5 + spreadM/500.0
	if estimatedSpeed > 5.0 {
		estimatedSpeed = 5.0
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

type Bayesian struct {
	readings       []model.DetectorReading
	windSpeed      float64
	windDir        float64
	windQuality    WindQuality
	gridRes        int
	latMin         float64
	latMax         float64
	lngMin         float64
	lngMax         float64
	confidencePenalty float64
}

func NewBayesian(readings []model.DetectorReading, windSpeed, windDir float64, windTimestamp time.Time) *Bayesian {
	wa := AssessWindQuality(windSpeed, windDir, windTimestamp)
	effectiveSpeed, effectiveDir := selectEffectiveWind(wa, readings)

	latMin, latMax := readings[0].Lat, readings[0].Lat
	lngMin, lngMax := readings[0].Lng, readings[0].Lng
	for _, r := range readings {
		if r.Lat < latMin {
			latMin = r.Lat
		}
		if r.Lat > latMax {
			latMax = r.Lat
		}
		if r.Lng < lngMin {
			lngMin = r.Lng
		}
		if r.Lng > lngMax {
			lngMax = r.Lng
		}
	}

	margin := 0.005

	var penalty float64
	switch wa.Quality {
	case WindQualityGood:
		penalty = 0.0
	case WindQualityStale:
		penalty = 0.2
	case WindQualityUnavailable:
		penalty = 0.4
	}

	return &Bayesian{
		readings:          readings,
		windSpeed:         effectiveSpeed,
		windDir:           effectiveDir,
		windQuality:       wa.Quality,
		gridRes:           50,
		latMin:            latMin - margin,
		latMax:            latMax + margin,
		lngMin:            lngMin - margin,
		lngMax:            lngMax + margin,
		confidencePenalty: penalty,
	}
}

func (b *Bayesian) Run() *model.LeakSourceResult {
	dLat := (b.latMax - b.latMin) / float64(b.gridRes)
	dLng := (b.lngMax - b.lngMin) / float64(b.gridRes)

	bestLat := 0.0
	bestLng := 0.0
	bestRate := 0.0
	bestLogPosterior := math.Inf(-1)

	for i := 0; i <= b.gridRes; i++ {
		for j := 0; j <= b.gridRes; j++ {
			lat := b.latMin + float64(i)*dLat
			lng := b.lngMin + float64(j)*dLng

			rate, logPost := b.computeMAP(lat, lng)
			if logPost > bestLogPosterior {
				bestLogPosterior = logPost
				bestLat = lat
				bestLng = lng
				bestRate = rate
			}
		}
	}

	confidence := b.computeConfidence(bestLogPosterior)

	diffusionRadius := math.Sqrt(bestRate) * 10.0
	if b.windQuality == WindQualityUnavailable {
		diffusionRadius *= 1.5
	} else if b.windQuality == WindQualityStale {
		diffusionRadius *= 1.25
	}

	return &model.LeakSourceResult{
		SourceLat:       bestLat,
		SourceLng:       bestLng,
		LeakRate:        bestRate,
		Confidence:      confidence,
		DiffusionRadius: diffusionRadius,
	}
}

func (b *Bayesian) computeConfidence(logPost float64) float64 {
	conf := 1.0 / (1.0 + math.Exp(-logPost/100.0))
	if conf > 0.99 {
		conf = 0.99
	}
	if conf < 0.1 {
		conf = 0.1
	}

	conf *= (1.0 - b.confidencePenalty)
	if conf < 0.05 {
		conf = 0.05
	}
	return conf
}

func (b *Bayesian) computeMAP(lat, lng float64) (float64, float64) {
	bestRate := 1.0
	bestLL := math.Inf(-1)

	for rate := 0.1; rate <= 100.0; rate += 0.5 {
		ll := b.logLikelihood(lat, lng, rate)
		if ll > bestLL {
			bestLL = ll
			bestRate = rate
		}
	}

	return bestRate, bestLL
}

func (b *Bayesian) logLikelihood(lat, lng, rate float64) float64 {
	ll := 0.0
	sigma := 1.0 + float64(b.windQuality)*0.5
	for _, r := range b.readings {
		predicted := gaussianPlume(lat, lng, rate, r.Lat, r.Lng, b.windSpeed, b.windDir)
		diff := r.Concentration - predicted
		ll -= 0.5 * diff * diff / (sigma * sigma)
	}
	return ll
}
