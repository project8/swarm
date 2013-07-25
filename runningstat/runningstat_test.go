package runningstat

import (
	"math"
	"testing"
	"log"
)

func TestMean(t *testing.T) {
	r := StatRunner{}
	r.Reset()

	for i := 0; i < 100; i++ {
		r.Update(float64(i))
	}

	m_err := math.Abs(r.Mean() - 49.5)/49.5
	if m_err > 1.0e-3 {
		log.Printf("error in mean calculation (%f) exceeds tolerance.\n", m_err)
		t.Fail()
	} else {
		log.Printf("mean calculation succeeded (%v, %v=tgt)\n",49.5,r.Mean())
	}
}

func TestVariance(t *testing.T) {
	r := StatRunner{}
	r.Reset()

	for i := 0; i < 100; i++ {
		r.Update(float64(i))
	}

	var_tgt := 841.0 + 2.0/3.0
	var_err := math.Abs(r.Variance() - var_tgt)/var_tgt
	if var_err > 1.0e-3 {
		log.Printf("error in variance (%f) exceeds tol of 1e-3.\n", var_err)
	} else {
		log.Printf("variance OK (%v, %v=tgt)\n",r.Variance(),var_tgt)
	}
}

func TestStdDev(t *testing.T) {
	r := StatRunner{}
	r.Reset()

	for i := 0; i < 100; i++ {
		r.Update(float64(i))
	}

	sd_tgt := 29.011491975882016
	sd_err := math.Abs(r.StdDev() - sd_tgt)/sd_tgt
	if sd_err > 1.0e-3 {
		log.Printf("error in stddev (%f) exceeds tol of 1e-3.\n", sd_err)
	} else {
		log.Printf("stddev OK (%v, %v=tgt)\n",r.StdDev(),sd_tgt)
	}
}
