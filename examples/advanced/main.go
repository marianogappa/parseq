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
	delay int
}

func (p *DataMapper) Map(input int) int {
	// access go routine-private data
	time.Sleep(time.Duration(p.delay) * time.Millisecond) // processing a request takes 1s
	// process input value, and return result
	return input
}

func main() {
	parallelism := 5

	// Create mapper upfront. Set private data here
	mappers := make([]parseq.Mapper[int, int], parallelism)
	for i := 0; i < 5; i++ {
		mappers[i] = &DataMapper{
			delay: i * 5,
		}
	}

	p, err := parseq.NewWithMapperSlice(parallelism, mappers) // 5 goroutines using the process function
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
