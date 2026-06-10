package locator

import (
	"math"
	"math/rand"
	"time"

	"gas-leak-monitor/internal/config"
	"gas-leak-monitor/internal/model"
	"gas-leak-monitor/internal/module"
)

type LeakLocator struct {
	cfg    *config.LeakConfig
	gauss  *GaussianParams
	wind   *WindEstimator
	bus    *module.MessageBus
}

type GaussianParams struct {
	MPPerDegLat  float64
	MPPerDegLng  float64
	SigmaYFactor float64
	SigmaZFactor float64
}

func NewLeakLocator(cfg *config.LeakConfig, bus *module.MessageBus) *LeakLocator {
	gauss := &GaussianParams{
		MPPerDegLat:  cfg.Gaussian.MPPerDegLat,
		MPPerDegLng:  cfg.Gaussian.MPPerDegLng,
		SigmaYFactor: cfg.Gaussian.SigmaYFactor,
		SigmaZFactor: cfg.Gaussian.SigmaZFactor,
	}
	wind := NewWindEstimator(cfg.Wind)
	return &LeakLocator{
		cfg:   cfg,
		gauss: gauss,
		wind:  wind,
		bus:   bus,
	}
}

func (l *LeakLocator) RunPSO(readings []model.DetectorReading, windSpeed, windDir float64, windTimestamp time.Time) *model.LeakSourceResult {
	wa := l.wind.AssessQuality(windSpeed, windDir, windTimestamp)
	effectiveSpeed, effectiveDir := l.wind.SelectEffective(wa, readings, l.gauss)

	latMin, latMax, lngMin, lngMax := readingBounds(readings, l.cfg.PSO.Margin)
	latCenter := (latMin + latMax) / 2
	lngCenter := (lngMin + lngMax) / 2
	latSpread := (latMax - latMin) / 2
	lngSpread := (lngMax - lngMin) / 2

	var confidencePenalty float64
	switch wa.Quality {
	case WindQualityGood:
		confidencePenalty = 0.0
	case WindQualityStale:
		confidencePenalty = 0.2
	case WindQualityUnavailable:
		confidencePenalty = 0.4
	}

	pso := &PSOSolver{
		numParticles:      l.cfg.PSO.NumParticles,
		maxIter:           l.cfg.PSO.MaxIterations,
		w:                 l.cfg.PSO.InertiaWeight,
		c1:                l.cfg.PSO.CognitiveCoeff,
		c2:                l.cfg.PSO.SocialCoeff,
		rateMin:           l.cfg.PSO.RateMin,
		rateMax:           l.cfg.PSO.RateMax,
		maxVRate:          l.cfg.PSO.MaxVRate,
		readings:          readings,
		windSpeed:         effectiveSpeed,
		windDir:           effectiveDir,
		gauss:             l.gauss,
		windQuality:       wa.Quality,
		confidencePenalty: confidencePenalty,
		latCenter:         latCenter,
		lngCenter:         lngCenter,
		latSpread:         latSpread,
		lngSpread:         lngSpread,
		diffusion:         l.cfg.Diffusion,
	}

	return pso.Solve()
}

func (l *LeakLocator) RunBayesian(readings []model.DetectorReading, windSpeed, windDir float64, windTimestamp time.Time) *model.LeakSourceResult {
	wa := l.wind.AssessQuality(windSpeed, windDir, windTimestamp)
	effectiveSpeed, effectiveDir := l.wind.SelectEffective(wa, readings, l.gauss)

	latMin, latMax, lngMin, lngMax := readingBounds(readings, l.cfg.Bayesian.Margin)

	var confidencePenalty float64
	switch wa.Quality {
	case WindQualityGood:
		confidencePenalty = 0.0
	case WindQualityStale:
		confidencePenalty = 0.2
	case WindQualityUnavailable:
		confidencePenalty = 0.4
	}

	bayes := &BayesianSolver{
		readings:          readings,
		windSpeed:         effectiveSpeed,
		windDir:           effectiveDir,
		gauss:             l.gauss,
		windQuality:       wa.Quality,
		confidencePenalty: confidencePenalty,
		gridRes:           l.cfg.Bayesian.GridResolution,
		rateMin:           l.cfg.Bayesian.RateMin,
		rateMax:           l.cfg.Bayesian.RateMax,
		rateStep:          l.cfg.Bayesian.RateStep,
		sigmaBase:         l.cfg.Bayesian.SigmaBase,
		sigmaPerQuality:   l.cfg.Bayesian.SigmaPerQualityLevel,
		latMin:            latMin,
		latMax:            latMax,
		lngMin:            lngMin,
		lngMax:            lngMax,
		diffusion:         l.cfg.Diffusion,
	}

	return bayes.Solve()
}

func (l *LeakLocator) Locate(req *module.LocateRequestMsg) *model.LeakSourceResult {
	if len(req.Readings) == 0 {
		return nil
	}
	ws := req.WindSpeed
	if ws <= 0 {
		ws = l.cfg.Wind.DefaultSpeed
	}

	var result *model.LeakSourceResult
	method := req.Method
	if method == "" {
		method = "pso"
	}

	switch method {
	case "bayesian":
		result = l.RunBayesian(req.Readings, ws, req.WindDirection, req.WindTimestamp)
		method = "bayesian"
	default:
		result = l.RunPSO(req.Readings, ws, req.WindDirection, req.WindTimestamp)
		method = "pso"
	}

	return result
}

func (l *LeakLocator) RunWorker() {
	for req := range l.bus.LocateRequest {
		result := l.Locate(req)
		select {
		case req.ResultChan <- result:
		default:
		}
	}
}

func gaussianPlume(gauss *GaussianParams, srcLat, srcLng, rate, detLat, detLng, windSpeed, windDir float64) float64 {
	dx := (detLng - srcLng) * gauss.MPPerDegLng
	dy := (detLat - srcLat) * gauss.MPPerDegLat
	windRad := windDir * math.Pi / 180.0
	x := dx*math.Cos(windRad) + dy*math.Sin(windRad)
	y := -dx*math.Sin(windRad) + dy*math.Cos(windRad)
	if x <= 0 || windSpeed <= 0 {
		return 0
	}
	sigmaY := gauss.SigmaYFactor * x
	sigmaZ := gauss.SigmaZFactor * x
	if sigmaY <= 0 || sigmaZ <= 0 {
		return 0
	}
	concentration := rate / (2 * math.Pi * windSpeed * sigmaY * sigmaZ)
	concentration *= math.Exp(-y*y/(2*sigmaY*sigmaY))
	return concentration
}

func readingBounds(readings []model.DetectorReading, margin float64) (float64, float64, float64, float64) {
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
	return latMin - margin, latMax + margin, lngMin - margin, lngMax + margin
}

type PSOSolver struct {
	numParticles      int
	maxIter           int
	w                 float64
	c1                float64
	c2                float64
	rateMin           float64
	rateMax           float64
	maxVRate          float64
	readings          []model.DetectorReading
	windSpeed         float64
	windDir           float64
	gauss             *GaussianParams
	windQuality       WindQuality
	confidencePenalty float64
	latCenter         float64
	lngCenter         float64
	latSpread         float64
	lngSpread         float64
	diffusion         config.DiffusionConfig
}

type particle struct {
	lat, lng, rate                          float64
	vLat, vLng, vRate                       float64
	pBestLat, pBestLng, pBestRate, pBestFit float64
}

func (p *PSOSolver) Solve() *model.LeakSourceResult {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	particles := make([]*particle, p.numParticles)
	for i := 0; i < p.numParticles; i++ {
		lat := p.latCenter + (rng.Float64()*2-1)*p.latSpread
		lng := p.lngCenter + (rng.Float64()*2-1)*p.lngSpread
		rate := rng.Float64()*p.rateMax + p.rateMin
		particles[i] = &particle{
			lat: lat, lng: lng, rate: rate,
			vLat: (rng.Float64()*2 - 1) * 0.001,
			vLng: (rng.Float64()*2 - 1) * 0.001,
			vRate: (rng.Float64()*2 - 1) * p.maxVRate,
			pBestLat: lat, pBestLng: lng, pBestRate: rate,
			pBestFit: math.Inf(1),
		}
	}
	gBestLat, gBestLng, gBestRate := 0.0, 0.0, 0.0
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
			r1, r2 := rng.Float64(), rng.Float64()
			pt.vLat = p.w*pt.vLat + p.c1*r1*(pt.pBestLat-pt.lat) + p.c2*r2*(gBestLat-pt.lat)
			pt.vLng = p.w*pt.vLng + p.c1*r1*(pt.pBestLng-pt.lng) + p.c2*r2*(gBestLng-pt.lng)
			pt.vRate = p.w*pt.vRate + p.c1*r1*(pt.pBestRate-pt.rate) + p.c2*r2*(gBestRate-pt.rate)
			maxVLat := p.latSpread * 0.1
			maxVLng := p.lngSpread * 0.1
			pt.vLat = math.Max(-maxVLat, math.Min(maxVLat, pt.vLat))
			pt.vLng = math.Max(-maxVLng, math.Min(maxVLng, pt.vLng))
			pt.vRate = math.Max(-p.maxVRate, math.Min(p.maxVRate, pt.vRate))
			pt.lat += pt.vLat
			pt.lng += pt.vLng
			pt.rate += pt.vRate
			if pt.rate < p.rateMin {
				pt.rate = p.rateMin
			}
		}
	}
	confidence := p.computeConfidence(gBestFit)
	diffusionRadius := computeDiffusionRadius(gBestRate, p.windQuality, p.diffusion)
	return &model.LeakSourceResult{
		SourceLat:       gBestLat,
		SourceLng:       gBestLng,
		LeakRate:        gBestRate,
		Confidence:      confidence,
		DiffusionRadius: diffusionRadius,
	}
}

func (p *PSOSolver) fitness(lat, lng, rate float64) float64 {
	residual := 0.0
	for _, r := range p.readings {
		predicted := gaussianPlume(p.gauss, lat, lng, rate, r.Lat, r.Lng, p.windSpeed, p.windDir)
		diff := r.Concentration - predicted
		residual += diff * diff
	}
	return residual
}

func (p *PSOSolver) computeConfidence(fit float64) float64 {
	var conf float64
	if fit <= 0 {
		conf = 0.99
	} else {
		conf = 1.0 / (1.0 + math.Sqrt(fit))
		if conf > 0.99 {
			conf = 0.99
		}
		if conf < 0.1 {
			conf = 0.1
		}
	}
	conf *= (1.0 - p.confidencePenalty)
	if conf < 0.05 {
		conf = 0.05
	}
	return conf
}

type BayesianSolver struct {
	readings          []model.DetectorReading
	windSpeed         float64
	windDir           float64
	gauss             *GaussianParams
	windQuality       WindQuality
	confidencePenalty float64
	gridRes           int
	rateMin           float64
	rateMax           float64
	rateStep          float64
	sigmaBase         float64
	sigmaPerQuality   float64
	latMin, latMax    float64
	lngMin, lngMax    float64
	diffusion         config.DiffusionConfig
}

func (b *BayesianSolver) Solve() *model.LeakSourceResult {
	dLat := (b.latMax - b.latMin) / float64(b.gridRes)
	dLng := (b.lngMax - b.lngMin) / float64(b.gridRes)
	bestLat, bestLng, bestRate := 0.0, 0.0, 0.0
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
	diffusionRadius := computeDiffusionRadius(bestRate, b.windQuality, b.diffusion)
	return &model.LeakSourceResult{
		SourceLat:       bestLat,
		SourceLng:       bestLng,
		LeakRate:        bestRate,
		Confidence:      confidence,
		DiffusionRadius: diffusionRadius,
	}
}

func (b *BayesianSolver) computeMAP(lat, lng float64) (float64, float64) {
	bestRate := 1.0
	bestLL := math.Inf(-1)
	for rate := b.rateMin; rate <= b.rateMax; rate += b.rateStep {
		ll := b.logLikelihood(lat, lng, rate)
		if ll > bestLL {
			bestLL = ll
			bestRate = rate
		}
	}
	return bestRate, bestLL
}

func (b *BayesianSolver) logLikelihood(lat, lng, rate float64) float64 {
	ll := 0.0
	sigma := b.sigmaBase + float64(b.windQuality)*b.sigmaPerQuality
	for _, r := range b.readings {
		predicted := gaussianPlume(b.gauss, lat, lng, rate, r.Lat, r.Lng, b.windSpeed, b.windDir)
		diff := r.Concentration - predicted
		ll -= 0.5 * diff * diff / (sigma * sigma)
	}
	return ll
}

func (b *BayesianSolver) computeConfidence(logPost float64) float64 {
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

func computeDiffusionRadius(rate float64, wq WindQuality, dc config.DiffusionConfig) float64 {
	radius := math.Sqrt(rate) * dc.RadiusFactor
	if wq == WindQualityUnavailable {
		radius *= dc.UnavailableMultiplier
	} else if wq == WindQualityStale {
		radius *= dc.StaleMultiplier
	}
	return radius
}
