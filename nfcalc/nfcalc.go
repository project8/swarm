package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/project8/swarm/gomonarch"
	"github.com/project8/swarm/runningstat"
	"github.com/kofron/go-fftw"
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
	FreqBin float64 `json:"freq"`
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
	PhysTemp, PowerMean, PowerVariance, KH2Temp float64
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

var Cernox87821 = []Point2d{
	// log(R), log(T)
	Point2d{X: math.Log(56.21), Y: math.Log(276.33)},
	Point2d{X: math.Log(133.62), Y: math.Log(77.0)},
	Point2d{X: math.Log(1764.0), Y: math.Log(4.2)},
}

type Calibrator interface {
	Calibrate(float64) float64
}

type Point2d struct {
	X, Y float64
}

type Cernox struct {
	CalPts []Point2d
}

func (c *Cernox) Calibrate(Ω float64) (K float64) {
	logΩ := math.Log(Ω)
	pt1, pt2 := find_interval(logΩ, c.CalPts)
	slope, icept := linear_fit(pt1, pt2)
	logT := interpolate(slope,icept, logΩ)
	return math.Exp(logT)
}

func interpolate(m, b, x float64) float64 {
	return (m*x + b)
}

func linear_fit(pt1, pt2 Point2d) (m, b float64) {
	m = (pt2.Y - pt1.Y)/(pt2.X - pt1.X)
	b = pt2.Y - m*pt2.X
	return
}

func find_interval(pt float64, points []Point2d) (Point2d,Point2d) {
	// Assumes points are sorted from low to high T!
	// If the first point is larger than the first element
	// in points, return the first element.  If not, iterate
	// through until we find the appropriate one.  If we get
	// all the way to the end, return the last element.
	if pt < points[0].X {
		return points[0], points[1]
	} else {
		for i := 1; i < len(points) - 1; i++ {
			if (pt >= points[i].X && pt < points[i+1].X ) {
				return points[i], points[i+1]
			} 
		}
	}
	return points[len(points)-2], points[len(points)-1]
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
	results := make([]Calculation, 0, 10)
	termCal := Cernox{CalPts: Cernox87821}
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
			term_temp = termCal.Calibrate(term_temp)
			if math.IsInf(term_temp,0) {
				log.Printf("[ERR] bad terminator temp, skipping.")
			} else {
				mean, variance, _ := Bartlett(m,c)
				calc = Calculation{PhysTemp: term_temp, 
					PowerMean: mean, 
					PowerVariance: variance,
					KH2Temp: amp_temp}
				results = append(results, calc)
			}
		}
	}

	result <- results
}

func Bartlett(m *gomonarch.Monarch, c *Config) (mean, v float64, e error) {
	// to calculate running statistics
	var stats runningstat.StatRunner
	stats.Reset()

	// First, which bin are we interested in?
	f_acq := gomonarch.AcqRate(m)
	f_nyq := f_acq/2
	f_roi := int(math.Trunc(c.FreqBin/f_nyq*float64(c.FFTSize)))

	// We need to know when we are going to "overflow" a record.
	r_len := int(gomonarch.RecordLength(m))

	// we aren't interested in the entire power spectrum, so instead
	// we will use the method due to Knuth to calculate running mean
	// and standard deviation as we accumulate power spectra.  this
	// holds the amount of data we need to keep around to a single
	// power spectrum and two floats.
	// the recurrence is 
	//      m_k = m_(k-1) + (x_k - m_(k-1)/k)
	//      s_k = s_(k-1) + (x_k - m_(k-1))*(x_k - m_k)
	in := fftw.Alloc1d(c.FFTSize)
	out := fftw.Alloc1d(c.FFTSize)
	plan := c.FFTWPlan
	r, er := gomonarch.NextRecord(m)
	if e == nil {
		// we need to initialize the running calculations.  this is a little
		// awkward but it's not that bad.
		d2a(r.Data[0:c.FFTSize], in)
		plan.ExecuteNewArray(in, out)
		stats.Update(cmplx.Abs(out[f_roi]))
		
		var l int = 1
		var x float64
		var idx0, idx1 int
		for k := 1; k < c.NAvg; k++ {
			idx0 = l*c.FFTSize 
			idx1 = (l+1)*c.FFTSize 
			if idx1 > r_len {
				r, er = gomonarch.NextRecord(m)
				if er != nil {
					e = er
					return
				} 
				l = 0
				idx0 = 0
				idx1 = c.FFTSize
			}
			d2a(r.Data[idx0:idx1],in)
			plan.ExecuteNewArray(in, out)
			
			// OK, now we grab the bin we care about and re-calculate
			// the running mean and variance.
			x = cmplx.Abs(out[f_roi])
			stats.Update(x)
			l++
		}
	} else {
		mean = stats.Mean()
		v = stats.Variance()
		e = er
	}
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
	for _, res := range results {
	    fmt.Printf("%v, %v, %v, %v\n",res.PhysTemp, res.KH2Temp, res.PowerMean, res.PowerVariance)
	}	
}

//"result": "mantis enviguration:\n  *output file name: /data/june2013_anti_00001_00000.egg\n  *digitizer rate: 500(MHz)\n  *run duration: 60000(ms)\n  *channel mode: 1(number of channels)\n  *record size: 2097152(bytes)\n  *buffer count: 640(entries)\n\npx1500 statistics:\n  * records taken: 14306\n  * acquisitions taken: 1\n  * live time: 60.0058(sec)\n  * dead time: 0(sec)\n  * total data read: 28612(Mb)\n  * average acquisition rate: 476.821(Mb/sec)\n\nwriter statistics:\n  * records written: 14306\n  * data written: 28612(Mb)\n  * live time: 60.0222(sec)\n  * average write rate: 476.69(Mb/sec)\n",
