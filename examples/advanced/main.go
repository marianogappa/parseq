package main

import (
	"fmt"
	"time"

	"github.com/marianogappa/parseq"
)

type DataMapper struct {
	// You can put private fields here.
	// For demonstative purposes we'll use this to
	// set an artificial delay used in the Map func.
	delay time.Duration
}

func (p *DataMapper) Map(input int) int {
	// Delay execution by a user-provided duration to simultate time spent
	// on processing/mapping input data. Replace this with your own code.
	time.Sleep(p.delay)

	// Process input value, and return result.
	// You can access go routine-private data here.
	return input
}

func main() {
	parallelism := 5

	// Create mapper upfront. Set private data here
	mappers := make([]parseq.Mapper[int, int], parallelism)
	for i := 0; i < 5; i++ {
		mappers[i] = &DataMapper{
			delay: time.Duration(i*50) * time.Millisecond,
		}
	}

	p, err := parseq.NewWithMapperSlice(mappers) // 5 goroutines using the process function
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
		p.Input <- counter // this simulates an incoming request
		counter++
	}
}
