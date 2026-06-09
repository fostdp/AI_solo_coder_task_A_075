package leak

import (
	"math"

	"gas-leak-monitor/internal/model"
)

type Bayesian struct {
	readings  []model.DetectorReading
	windSpeed float64
	windDir   float64
	gridRes   int
	latMin    float64
	latMax    float64
	lngMin    float64
	lngMax    float64
}

func NewBayesian(readings []model.DetectorReading, windSpeed, windDir float64) *Bayesian {
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
	return &Bayesian{
		readings:  readings,
		windSpeed: windSpeed,
		windDir:   windDir,
		gridRes:   50,
		latMin:    latMin - margin,
		latMax:    latMax + margin,
		lngMin:    lngMin - margin,
		lngMax:    lngMax + margin,
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

	diffusionRadius := math.Sqrt(bestRate) * 10.0
	confidence := 1.0 / (1.0 + math.Exp(-bestLogPosterior/100.0))
	if confidence > 0.99 {
		confidence = 0.99
	}
	if confidence < 0.1 {
		confidence = 0.1
	}

	return &model.LeakSourceResult{
		SourceLat:       bestLat,
		SourceLng:       bestLng,
		LeakRate:        bestRate,
		Confidence:      confidence,
		DiffusionRadius: diffusionRadius,
	}
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
	sigma := 1.0
	for _, r := range b.readings {
		predicted := gaussianPlume(lat, lng, rate, r.Lat, r.Lng, b.windSpeed, b.windDir)
		diff := r.Concentration - predicted
		ll -= 0.5 * diff * diff / (sigma * sigma)
	}
	return ll
}
