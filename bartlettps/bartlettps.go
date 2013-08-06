package main

import(
	"flag"
	"fmt"
	"log"
	"math"
	"math/cmplx"
	"github.com/project8/swarm/gomonarch"
	"github.com/project8/swarm/runningstat"
	"github.com/project8/swarm/gomonarch/frame"
	"github.com/project8/swarm/sensors/px1500"
	"github.com/kofron/go-fftw"
)

func main() {
	var infname = flag.String("infile", "", "absolute path of input file")
	var FFTSize = flag.Uint64("fft_size", 1024, "size of FFT to calculate")
	var NFFTs = flag.Uint64("num_ffts", 100, "number of FFTs to calculate")
	flag.Parse()

	if *infname == "" {
		flag.Usage()
		return
	}

	m, m_err := gomonarch.Open(*infname, gomonarch.ReadMode)
	if m_err != nil {
		log.Print("Error opening file for reading!")
		return
	}
	fr, fr_err := frame.NewFramer(m, *FFTSize)
	if fr_err != nil {
		log.Print(fr_err.Error())
		return
	}


	// the running statistics calculator(s)
	stats := make([]runningstat.StatRunner, *FFTSize, *FFTSize)
	for _, val := range stats {
		val.Reset()
	}

	// prepare the arrays, the plan, and the px1500 calibration thingie
	in := fftw.Alloc1d((int)(*FFTSize))
	out := fftw.Alloc1d((int)(*FFTSize))
	plan := fftw.PlanDft1d(in, out, fftw.Forward, fftw.Estimate)
	p := px1500.PX1500{}
	var i uint64 = 0
	for i = 0; i < *NFFTs; i++ {
		f, ok := fr.Advance()
		if ok != nil {
			log.Print("[WARN] Out of data too early...")
			log.Printf("[WARN] only %d FFTs calculated.",i)
			break
		}

		for pos, val := range f.Data {
			in[pos] = complex(p.Calibrate(val),0)
		}
		plan.ExecuteNewArray(in, out)
		for pos, val := range out {
			stats[pos].Update(cmplx.Abs(val))
		}
	}
	norm := 1.0/50.0*2.0*math.Pow(0.5,2.0)/math.Pow(256.0,2.0)
	norm *= 1.0/(float64)(*FFTSize)

	for pos, val := range stats {
		fmt.Printf("%d, %e, %e\n", pos, 
			norm*val.Mean(), 
			norm*val.Variance())
	}
}
