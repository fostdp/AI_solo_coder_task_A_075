package leak

import (
	"math"
	"math/rand"
	"time"

	"gas-leak-monitor/internal/model"
)

const mPerDegLat = 111320.0
const mPerDegLng = 95200.0

func gaussianPlume(srcLat, srcLng, rate, detLat, detLng, windSpeed, windDir float64) float64 {
	dx := (detLng - srcLng) * mPerDegLng
	dy := (detLat - srcLat) * mPerDegLat

	windRad := windDir * math.Pi / 180.0
	x := dx*math.Cos(windRad) + dy*math.Sin(windRad)
	y := -dx*math.Sin(windRad) + dy*math.Cos(windRad)

	if x <= 0 || windSpeed <= 0 {
		return 0
	}

	sigmaY := 0.1 * x
	sigmaZ := 0.05 * x

	if sigmaY <= 0 || sigmaZ <= 0 {
		return 0
	}

	concentration := rate / (2 * math.Pi * windSpeed * sigmaY * sigmaZ)
	concentration *= math.Exp(-y*y/(2*sigmaY*sigmaY))

	return concentration
}

type PSO struct {
	numParticles      int
	maxIter           int
	w                 float64
	c1                float64
	c2                float64
	readings          []model.DetectorReading
	windSpeed         float64
	windDir           float64
	windQuality       WindQuality
	confidencePenalty float64
	latCenter         float64
	lngCenter         float64
	latSpread         float64
	lngSpread         float64
}

type particle struct {
	lat       float64
	lng       float64
	rate      float64
	vLat      float64
	vLng      float64
	vRate     float64
	pBestLat  float64
	pBestLng  float64
	pBestRate float64
	pBestFit  float64
}

func NewPSO(readings []model.DetectorReading, windSpeed, windDir float64, windTimestamp time.Time) *PSO {
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
	latCenter := (latMin + latMax) / 2
	lngCenter := (lngMin + lngMax) / 2
	latSpread := (latMax - latMin)/2 + margin
	lngSpread := (lngMax - lngMin)/2 + margin

	var confidencePenalty float64
	switch wa.Quality {
	case WindQualityGood:
		confidencePenalty = 0.0
	case WindQualityStale:
		confidencePenalty = 0.2
	case WindQualityUnavailable:
		confidencePenalty = 0.4
	}

	return &PSO{
		numParticles:      50,
		maxIter:           100,
		w:                 0.7,
		c1:                1.5,
		c2:                1.5,
		readings:          readings,
		windSpeed:         effectiveSpeed,
		windDir:           effectiveDir,
		latCenter:         latCenter,
		lngCenter:         lngCenter,
		latSpread:         latSpread,
		lngSpread:         lngSpread,
		confidencePenalty: confidencePenalty,
		windQuality:       wa.Quality,
	}
}

func (p *PSO) Run() *model.LeakSourceResult {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	particles := make([]*particle, p.numParticles)
	for i := 0; i < p.numParticles; i++ {
		lat := p.latCenter + (rng.Float64()*2-1)*p.latSpread
		lng := p.lngCenter + (rng.Float64()*2-1)*p.lngSpread
		rate := rng.Float64() * 50 + 0.1
		particles[i] = &particle{
			lat:       lat,
			lng:       lng,
			rate:      rate,
			vLat:      (rng.Float64()*2 - 1) * 0.001,
			vLng:      (rng.Float64()*2 - 1) * 0.001,
			vRate:     (rng.Float64()*2 - 1) * 5,
			pBestLat:  lat,
			pBestLng:  lng,
			pBestRate: rate,
			pBestFit:  math.Inf(1),
		}
	}

	gBestLat := 0.0
	gBestLng := 0.0
	gBestRate := 0.0
	gBestFit := math.Inf(1)

	for iter := 0; iter < p.maxIter; iter++ {
		for _, pt := range particles {
			fit := p.fitness(pt.lat, pt.lng, pt.rate)
			if fit < pt.pBestFit {
				pt.pBestFit = fit
				pt.pBestLat = pt.lat
				pt.pBestLng = pt.lng
				pt.pBestRate = pt.rate
			}
			if fit < gBestFit {
				gBestFit = fit
				gBestLat = pt.lat
				gBestLng = pt.lng
				gBestRate = pt.rate
			}
		}

		for _, pt := range particles {
			r1 := rng.Float64()
			r2 := rng.Float64()
			pt.vLat = p.w*pt.vLat + p.c1*r1*(pt.pBestLat-pt.lat) + p.c2*r2*(gBestLat-pt.lat)
			pt.vLng = p.w*pt.vLng + p.c1*r1*(pt.pBestLng-pt.lng) + p.c2*r2*(gBestLng-pt.lng)
			pt.vRate = p.w*pt.vRate + p.c1*r1*(pt.pBestRate-pt.rate) + p.c2*r2*(gBestRate-pt.rate)

			maxVLat := p.latSpread * 0.1
			maxVLng := p.lngSpread * 0.1
			maxVRate := 5.0

			pt.vLat = math.Max(-maxVLat, math.Min(maxVLat, pt.vLat))
			pt.vLng = math.Max(-maxVLng, math.Min(maxVLng, pt.vLng))
			pt.vRate = math.Max(-maxVRate, math.Min(maxVRate, pt.vRate))

			pt.lat += pt.vLat
			pt.lng += pt.vLng
			pt.rate += pt.vRate
			if pt.rate < 0.01 {
				pt.rate = 0.01
			}
		}
	}

	confidence := p.computeConfidence(gBestFit)
	diffusionRadius := p.computeDiffusionRadius(gBestRate)

	return &model.LeakSourceResult{
		SourceLat:       gBestLat,
		SourceLng:       gBestLng,
		LeakRate:        gBestRate,
		Confidence:      confidence,
		DiffusionRadius: diffusionRadius,
	}
}

func (p *PSO) fitness(lat, lng, rate float64) float64 {
	residual := 0.0
	for _, r := range p.readings {
		predicted := gaussianPlume(lat, lng, rate, r.Lat, r.Lng, p.windSpeed, p.windDir)
		diff := r.Concentration - predicted
		residual += diff * diff
	}
	return residual
}

func (p *PSO) computeConfidence(fit float64) float64 {
	if fit <= 0 {
		conf := 0.99
		conf *= (1.0 - p.confidencePenalty)
		if conf < 0.05 {
			conf = 0.05
		}
		return conf
	}
	conf := 1.0 / (1.0 + math.Sqrt(fit))
	if conf > 0.99 {
		conf = 0.99
	}
	if conf < 0.1 {
		conf = 0.1
	}
	conf *= (1.0 - p.confidencePenalty)
	if conf < 0.05 {
		conf = 0.05
	}
	return conf
}

func (p *PSO) computeDiffusionRadius(rate float64) float64 {
	radius := math.Sqrt(rate) * 10.0
	if p.windQuality == WindQualityUnavailable {
		radius *= 1.5
	} else if p.windQuality == WindQualityStale {
		radius *= 1.25
	}
	return radius
}
