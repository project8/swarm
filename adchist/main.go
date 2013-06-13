package main

import(
	"fmt"
	"github.com/project8/gomonarch"
)

func main () {
	m, err := gomonarch.Open("/Users/kofron/quicktest.egg")
	if err == nil {
		defer gomonarch.Close(m)
	}

	hist := make([]int, 256, 256)
	rec1, recerr := gomonarch.NextRecord(m)
	if recerr == nil {
		for _, recval := range(rec1) {
			hist[recval] += 1
		}
	}
	for pos, val := range(hist) {
		fmt.Printf("%d, %d\n", pos, val)
	}
}
