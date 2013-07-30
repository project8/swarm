package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"github.com/project8/swarm/gomonarch"
	"github.com/project8/swarm/gomonarch/frame"
	"github.com/project8/swarm/runningstat"
	"github.com/project8/swarm/sensors/cernox"
	"github.com/project8/swarm/sensors/px1500"
	"github.com/kofron/go-fftw"
	"github.com/kofron/gogsl/fit"
	"io/ioutil"
	"net/http"
	"strings"
	"math"
	"math/cmplx"
	"log"
)

const (
	FileNameStr      string       = `*output file name: `
	NumCodes int = 1 << 8
	kB float64 = 1.3806488e-23
)

type Config struct {
	DBHost     string `json:"couch_host"`
	DBPort     uint   `json:"couch_port"`
	DBName     string `json:"db_name"`
	RunTag     string   `json:"target_run"`
	FFTSize    int   `json:"fft_size"`
	NAvg       int   `json:"num_averages"`
	NWorkers   int   `json:"num_workers"`
	DataLocation string `json:"local_data_dir"`
	PSOutFile string `json:"power_spectrum_out_filename"`
	FitOutFile string `json:"fit_out_filename"`
	RawOutFile string `json:"raw_out_filename"`
	FFTWPlan *fftw.Plan
}

type ViewDoc struct {
	Key string `json:"key"`
	Value RunDoc
}

type ViewResult struct {
	TotalRows uint `json:"total_rows"`
	Offset uint `json:"offset"`
	Docs []ViewDoc `json:"rows"`
}

type MantisDoc struct {
	TimeStamp string `json:"timestamp"`
	Result    string `json:"result"`
	Filename  string
}

type DripValue struct {
	RawValue string `json:"result"`
	FinalValue string `json:"final"`
}

type RunDoc struct {
	RunNumber    int       `json:"run_number"`
	RunSequence  int       `json:"sequence_number"`
	Timestamp    string    `json:"run_timestamp"`
	KH3Temp      DripValue    `json:"kh3_temp"`
	KH2Temp      DripValue    `json:"kh2_temp"`
	WGCellTemp   DripValue    `json:"waveguide_cell_body_temp"`
	TerminatorTemp DripValue  `json:"terminator_temp"`
	MantisResult MantisDoc `json:"mantis"`
}

type CouchHost struct {
	Host string
	Port uint
}

type Database struct {
	Name string
	URL  string
}

type View struct {
	DesignDoc string
	Name string
	URL string
}

type Calculation struct {
	PowerStats []runningstat.StatRunner
	NyquistFreq, PhysTemp, KH2Temp float64
}

func (c *CouchHost) URL() string {
	return fmt.Sprintf("http://%s:%d", c.Host, c.Port)
}

func (c *CouchHost) NewDatabase(name string) *Database {
	d := Database{Name: name, URL: fmt.Sprintf("%s/%s", c.URL(), name)}
	return &d
}

func (d *Database) NewView(design, name string) *View {
	u := fmt.Sprintf("/_design/%s/_view/%s",design, name)
	return &View{Name: name, DesignDoc: design, URL: u}

}

func d2a(input []byte, output []complex128) {
	var v float64
	for pos, _ := range input {
		v = -0.25 + 0.5*float64(input[pos])/float64(NumCodes)
		(output)[pos] = complex(v, 0.0)
	}
}

func parseMantisResult(result *MantisDoc) (*MantisDoc, error) {
	n := strings.Index(result.Result, FileNameStr)
	if n > 0 {
		substr := result.Result[(n + len(FileNameStr)):len(result.Result)]
		m := strings.Index(substr, "\n")
		if m > 0 {
			result.Filename = substr[0:m]
		}
	}
	return result, nil
}

func ProcessRuns(docs []ViewDoc, c *Config, result chan<- []Calculation) {
	var calc Calculation
	results := make([]Calculation, 0, len(docs))
	termCal := cernox.Cernox{CalPts: cernox.Cernox87821}
	for _, doc := range docs {
		var term_temp, amp_temp float64
		mr, _ := parseMantisResult(&doc.Value.MantisResult)
		fmt.Sscanf(doc.Value.TerminatorTemp.RawValue, "%g OHM", &term_temp)
		fmt.Sscanf(doc.Value.KH2Temp.FinalValue, "%g K", &amp_temp)
		
		// try to open the file.
		m, e := gomonarch.Open(mr.Filename, gomonarch.ReadMode)
		if e != nil {
			log.Printf("[ERR] couldn't open %s, skipping.", mr.Filename)
		} else {
			f_nyq := m.AcqRate()/2.0
			term_temp = termCal.Calibrate(term_temp)
			if math.IsInf(term_temp,0) {
				log.Printf("[ERR] bad terminator temp, skipping.")
			} else {
				s, _ := Bartlett(m,c)
				calc = Calculation{PhysTemp: term_temp, 
					PowerStats: s,
					KH2Temp: amp_temp,
					NyquistFreq: f_nyq,}
				results = append(results, calc)
			}
		}
	}

	result <- results
}

func Bartlett(m *gomonarch.Monarch, c *Config) (s []runningstat.StatRunner, e error) {
	// to calculate running statistics
	s = make([]runningstat.StatRunner, c.FFTSize, c.FFTSize)
	for _, val := range s {
		val.Reset()
	}

	fr, fr_err := frame.NewFramer(m, uint64(c.FFTSize))
	if fr_err != nil {
		e = fr_err
		return
	}

	in := fftw.Alloc1d(c.FFTSize)
	out := fftw.Alloc1d(c.FFTSize)
	plan := c.FFTWPlan

	p := px1500.PX1500{}

	for i := 0; i < c.NAvg; i++ {
		f, ok := fr.Advance()
		if ok != nil {
			e = ok
			break
		}

		for pos, v := range f.Data {
			in[pos] = complex(p.Calibrate(v), 0)
		}

		plan.ExecuteNewArray(in, out)
		for p := 0; p < c.FFTSize; p++ {
		    s[p].Update(cmplx.Abs(out[p]))
		}
	}

	return s, nil
}

func est_frac_err(f *fit.LinearFit, x, y []float64) (err float64) {
	sse := 0.0
	for p, v := range y {
		sse += math.Pow((f.Y0 + x[p]*f.Slope) - v,2.0)
	}
	// mean squared error for 2 parameter linear fit
	mse := sse/(float64)(len(x) - 2)

	// now spread in x parameter
	x_bar := 0.0
	for _, v := range x {
		x_bar += v
	}
	x_bar /= float64(len(x))

	x_s := 0.0
	for _, v := range x {
		x_s += math.Pow((v - x_bar),2.0)
	}

	i_contrib := 1/math.Pow(f.Y0,2.0)
	i_contrib *= (1/(float64)(len(x)) + math.Pow(x_bar,2.0)/x_s)

	m_contrib := 1/math.Pow(f.Slope,2.0)
	m_contrib *= 1/x_s

	err = math.Sqrt(mse*(i_contrib + m_contrib))
	return
}

/*
Grab the results of the by_run_tag view with the key set to the run tag
of interest.  Parse out the filenames, and grab the temperature data from
the runs.  Spawn off goroutines to open those files, calculate averaged power
spectra, and return the average power, the variance of that power, and the
temperature.  Either:

1) Use a waitgroup to synchronize waiting on the goroutines.
2) Keep a fixed number of goroutines "waiting" to get the next available 
filename from the main thread.  Once all filenames are consumed, move on.

Choose a reference temperature.  Divide all temps by that reference temp, and
all powers by the power calculated at that reference temperature.
*/
func main() {
	var conf = flag.String("config", "", "path to configuration file")
	flag.Parse()

	var env = &Config{}

	if *conf == "" {
		panic("must have conf file!")
	} else {
		bytes, err := ioutil.ReadFile(*conf)
		if err != nil {
			panic("couldn't read config file!")
		}
		err = json.Unmarshal(bytes, env)
	}

	fs := "http://%s:%d/%s/_design/%s/_view/by_run_tag?%s"
	url := fmt.Sprintf(fs,env.DBHost,env.DBPort,env.DBName,"general","key=\"noise_temperature_tone\"")
	r, get_err := http.Get(url)
	if get_err != nil {
		panic("couldn't fetch run data!")
	} 
	data, _ := ioutil.ReadAll(r.Body)
	
	var v ViewResult
	er := json.Unmarshal(data, &v)
	if er != nil {
		panic(er)
	}

	// As in all things, we need a plan.
	in := fftw.Alloc1d(env.FFTSize)
	out := fftw.Alloc1d(env.FFTSize)
	plan := fftw.PlanDft1d(in, out, fftw.Forward, fftw.Estimate)
	env.FFTWPlan = plan

	//  Figure out how to split up work.
	n := env.NWorkers
	m := len(v.Docs)

	// Divide m by n first
	num_per := int(math.Floor(float64(m)/float64(n)))

	// Send out the work.
	ch := make(chan []Calculation, n)
	var idx0, idx1 int
	for i := 0; i < (n-1); i++ {
		idx0 = i*num_per
		idx1 = (i+1)*num_per
		go ProcessRuns(v.Docs[idx0:idx1], env, ch)
	}
	go ProcessRuns(v.Docs[(idx1+1):len(v.Docs)], env, ch)

	var result []Calculation
	results := make([]Calculation, 0, len(v.Docs))
	for i := 0; i < n; i++ {
		result = <- ch
		results = append(results, result...)
	}

	t := make([]float64, len(results), len(results))
	p := make([]float64, len(results), len(results))

	// Power spectrum normalization
	norm := 1.0/50.0*2.0*math.Pow(0.5,2.0)/math.Pow(256.0,2.0)
	norm *= 1.0/(float64)(env.FFTSize)

	// Loop over bins, at each bin use all physical temps,
	// plus the mean power in the bin at each temp for the
	// X,Y pairs.  Then linear fit.  Write results to the fit
	// output file, and raw data to the raw output file.
	fit_out, fit_f_err := os.Create(env.FitOutFile)
	if fit_f_err != nil {
		log.Print("[ERR] couldn't open fit file for writing!")
		return
	}
	defer fit_out.Close()

	raw_out, raw_f_err := os.Create(env.RawOutFile)
	if raw_f_err != nil {
		log.Print("[ERR] couldn't open raw data file for writing!")
		return
	}
	defer raw_out.Close()
	fmt.Fprintf(raw_out, "bin, freq, phys_temp, power, norm_t, norm_p\n")

	n_t := make([]float64, env.FFTSize, env.FFTSize)
	fmt.Fprintf(fit_out, "bin, freq, icept, slope, temp, sum_squares\n")
	for bin := 0; bin < env.FFTSize/2; bin++ {
		t0, p0 := results[0].PhysTemp, results[0].PowerStats[bin].Mean()
		f_nyq := results[0].NyquistFreq
		freq := (float64)(bin)/(float64)(env.FFTSize)*f_nyq
		
		for pos, res := range results {
			t[pos] = res.PhysTemp/t0
			p[pos] = res.PowerStats[bin].Mean()/p0
			fmt.Fprintf(raw_out,
				"%d, %e, %e, %e, %e, %e\n",
				bin,
				freq,
				res.PhysTemp,
				res.PowerStats[bin].Mean(),
				t[pos],
				p[pos])
		}

		f, fit_err := fit.FitLinear(&t,&p,1,1)
		if fit_err != nil {
			log.Print("[ERR] fit failed, skipping.")
		} else {

			frac_err := est_frac_err(f, t, p)
			γ := f.Y0/f.Slope
			n_t[bin] = γ*t0
			

			fmt.Fprintf(fit_out,
				"%d, %e, %e, %e, %e, %e, %e\n",
				bin,
				freq,
				f.Y0,
				f.Slope,
				n_t[bin],
				f.SumSq,
				frac_err*n_t[bin])
		}
	}

	// Now we dump the raw power spectra, calibrated with the power in 
	// Watts.  This is a loop over the results, as each one has its own
	// power spectrum.  A gain calculation could go here too.

	ps_out, ps_out_err := os.Create(env.PSOutFile)
	if ps_out_err != nil {
		log.Print("[ERR] Couldn't open PS file for writing!")
	}
	defer ps_out.Close()

	fmt.Fprintf(ps_out, "result, fft_bin, gain, power, power_norm\n")
	for res := 0; res < len(results); res++ {
		f_nyq := results[res].NyquistFreq
		for bin := 0; bin < env.FFTSize/2; bin++ {
			// We can calculate gain now here.
			// We know P = G*k*B*(T0 + Ta).
			// We found Ta above, T0 is just the physical
			// temperature.
			// So P/(kBT) = G.
			pow := results[res].PowerStats[bin].Mean()*norm
			bw := f_nyq/(float64)(env.FFTSize)
			g := pow/(kB*(n_t[bin] + results[res].PhysTemp)*bw)

			fmt.Fprintf(ps_out,
				"%d, %d, %e, %e, %e\n",
				res,
				bin,
				g,
				results[res].PowerStats[bin].Mean(),
				norm*results[res].PowerStats[bin].Mean())
		}
	}

}
