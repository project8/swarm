package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/project8/swarm/gomonarch"
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

func Bartlett(m *gomonarch.Monarch, c *Config) (mean, v float64, e error) {
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
	plan := fftw.PlanDft1d(in, out, fftw.Forward, fftw.Estimate)
	r, er := gomonarch.NextRecord(m)
	if e == nil {
		// we need to initialize the running calculations.  this is a little
		// awkward but it's not that bad.
		d2a(r.Data[0:c.FFTSize], in)
		plan.Execute()
		mean = cmplx.Abs(out[f_roi])
		v = 0
		
		var l int = 1
		var x, lastm float64
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
			plan.Execute()
			
			// OK, now we grab the bin we care about and re-calculate
			// the running mean and variance.
			x = cmplx.Abs(out[f_roi])
			lastm = mean
			mean = mean + (x - mean)/float64(k)
			v = v + (x - lastm)*(x - mean)
			l++
		}
		
		v /= float64(c.NAvg - 1)
	} else {
		mean = 0
		v = 0
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
	
	for _, run := range v.Docs {
		var temp float64
		mr, _ := parseMantisResult(&run.Value.MantisResult)
		fmt.Sscanf(run.Value.TerminatorTemp.FinalValue, "%g K", &temp)
		// try to open the file.
		m, e := gomonarch.Open(mr.Filename, gomonarch.ReadMode)
		if e != nil {
			log.Printf("[ERR] couldn't open %s, skipping.", mr.Filename)
		} else {
			mean, variance, _ := Bartlett(m,env)
			fmt.Printf("%v, %v, %v\n", temp, mean, variance)
		}
	}
}

//"result": "mantis enviguration:\n  *output file name: /data/june2013_anti_00001_00000.egg\n  *digitizer rate: 500(MHz)\n  *run duration: 60000(ms)\n  *channel mode: 1(number of channels)\n  *record size: 2097152(bytes)\n  *buffer count: 640(entries)\n\npx1500 statistics:\n  * records taken: 14306\n  * acquisitions taken: 1\n  * live time: 60.0058(sec)\n  * dead time: 0(sec)\n  * total data read: 28612(Mb)\n  * average acquisition rate: 476.821(Mb/sec)\n\nwriter statistics:\n  * records written: 14306\n  * data written: 28612(Mb)\n  * live time: 60.0222(sec)\n  * average write rate: 476.69(Mb/sec)\n",
