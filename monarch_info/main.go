package main

import (
	"fmt"
	"io"
	"flag"
	"log"
	"github.com/project8/swarm/gomonarch"
)


func main() {
	var fname = flag.String("in","","Input file name")
	flag.Parse()
	
	m, m_open_err := gomonarch.Open(*fname, gomonarch.ReadMode)
	if m_open_err != nil {
		log.Fatal("could not open file for reading:")
		log.Fatal(m_open_err)
	}
	defer gomonarch.Close(m)
	

	nc := gomonarch.NumChannels(m)
	rl := gomonarch.RecordLength(m)
	fmt.Printf("info for file named <%s>\n",*fname)
	fmt.Printf("\tnumber of channels: %d\n",nc)
	fmt.Printf("\trecord length in bytes: %d\n",rl)
	nr := 0
	var err error
	for _, err = gomonarch.NextRecord(m); err == nil; _, err = gomonarch.NextRecord(m) {
		nr += 1
	}
	if err != io.EOF {
		fmt.Printf("scanning file terminated abnormally...")
		log.Fatal(err)
	}
	fmt.Printf("\trecords contained: %d\n",nr)
}

func usage() (s string) {
	s = "usage: monarch_info <filename>"
	return
}
