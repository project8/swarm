package main

import(
	"fmt"
	"github.com/project8/swarm/gomonarch"
	"flag"
	"log"
	"github.com/project8/swarm/gomonarch"
	"code.google.com/p/plotinum/plotter"
	"code.google.com/p/plotinum/plot"
)

func main () {
	var fname = flag.String("file_in","","Input file name")
	var iname = flag.String("image_out","adc_hist.png","Output image file name")
	flag.Parse()

	if *fname == "" {
		log.Print("Input filename must be specified!")
		return
	}

	m, err := gomonarch.Open(*fname,gomonarch.ReadMode)
	if err == nil {
		defer gomonarch.Close(m)
	}

	hist := make(plotter.Values, 100000)
	rec1, recerr := gomonarch.NextRecord(m)
	if recerr == nil {
		for pos, _ := range(hist) {
			hist[pos] += float64(rec1.Data[pos])
		}
	}

	p, err := plot.New()
	if err != nil {
		panic(err)
	}
	p.Title.Text = "ADC Counts"
	h, err := plotter.NewHist(hist, 100)
	if err != nil {
		panic(err)
	}
	p.Add(h)

	if err := p.Save(8,8,*iname); err != nil {
		panic(err)
	}
}
