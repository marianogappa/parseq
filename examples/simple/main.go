package main

import (
	"fmt"
	"time"

	"github.com/marianogappa/parseq"
)

type DataMapper struct {
	// you can put private fields here
}

func (p *DataMapper) Map(input int) int {
	// access go routine-private data
	time.Sleep(time.Duration(200) * time.Millisecond) // processing a request takes 200ms
	// process input value, and return result
	return input
}

func main() {
	p, err := parseq.NewWithMapper[int, int](5, &DataMapper{}) // 5 goroutines using the process function
	if err != nil {
		panic(err)
	}

	go p.Start()
	go makeRequests(p)

	for out := range p.Output { // after initial 1s, requests output every ~200ms
		fmt.Print(out, ".") // and output respects input order
	}
	p.Close()
}

func makeRequests(p *parseq.ParSeq[int, int]) {
	counter := 666
	for {
		p.Input <- counter                 // this simulates an incoming request
		time.Sleep(200 * time.Millisecond) // requests come every 200ms
		counter++
	}
}
