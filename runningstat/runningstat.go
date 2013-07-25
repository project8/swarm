package runningstat

import (
	"math"
)

type StatRunner struct {
	n_upd uint64

	/*
        Running mean, variance, and standard deviation are calculated
        using the algorithm presented in Welford '62.
        */
	μ, σ, σsq float64
}

func (r *StatRunner) Reset() (*StatRunner) {
	r.μ = 0 
	r.σ = 0
	r.σsq = 0
	r.n_upd = 0
	return r
}

func (r *StatRunner) Update(x float64) (*StatRunner) {
	r.n_upd++
	if r.n_upd == 1 {
		r.μ = x
	} else {
		var lastm = r.μ
		r.μ += (x - r.μ)/float64(r.n_upd)
		r.σsq += (x - lastm)*(x - r.μ)
	}
	return r
}

func (r *StatRunner) Mean() float64 {
	return r.μ
}

func (r *StatRunner) Variance() float64 {
	if r.n_upd > 1 {
		return r.σsq/float64(r.n_upd - 1)
	} else {
		return 0
	}
}

func (r *StatRunner) StdDev() float64 {
	if r.n_upd > 1 {
		return math.Sqrt(r.σsq/float64(r.n_upd - 1))
	} else {
		return 0
	}
}
