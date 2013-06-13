package main

import (
	"fmt"
	"flag"
	"log"
	"github.com/project8/gomonarch"
)


func main() {
	var fname = flag.String("in","","Input file name")
	flag.Parse()
	
	m, m_open_err := gomonarch.Open(*fname)
	if m_open_err != nil {
		log.Fatal("could not open file for reading!")
	}

	nc := gomonarch.NumChannels(m)
	rl := gomonarch.RecordLength(m)
	fmt.Printf("\tnumber of channels: %d\n",nc)
	fmt.Printf("\trecord length in bytes: %d\n",rl)
}

func usage() (s string) {
	s = "usage: monarch_info <filename>"
	return
}
