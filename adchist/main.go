package main

import(
	"github.com/project8/swarm/gomonarch"
	"code.google.com/p/plotinum/plotter"
	"code.google.com/p/plotinum/plot"
)

func main () {
	m, err := gomonarch.Open("/Users/project8/quicktest.egg",gomonarch.ReadMode)
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

	if err := p.Save(4,4,"adc_hist.png"); err != nil {
		panic(err)
	}
}
