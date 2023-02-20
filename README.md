# ParSeq 
[![Build Status](https://img.shields.io/travis/MarianoGappa/parseq.svg)](https://travis-ci.org/MarianoGappa/parseq) 
[![GoDoc](https://godoc.org/github.com/MarianoGappa/parseq?status.svg)](https://godoc.org/github.com/MarianoGappa/parseq) 
[![GitHub license](https://img.shields.io/badge/license-MIT-blue.svg)](https://raw.githubusercontent.com/MarianoGappa/parseq/master/LICENSE)

The parseq package provides a simple interface for processing a stream in parallel,
with configurable level of parallelism, while still outputting a sequential stream
that respects the order of input.

This package requires Go 1.18+ due to use of generics.

## WAT?

In 99% of the cases, you read messages from a channel and process them sequentially. This is fine. Note that if the channel is unbuffered, your goroutine blocks the next message for the length of the time it takes to process each message. If processing is a lengthy operation, it doesn't matter if the channel is buffered or not, except for the initial period.

```go
for msg := range channel {
	process(msg)
}
```

![Sequential](sequential.png)

If this throttling is problematic, you need to parallelise. When you parallelise, results can come out of order. This can be a problem (or not):

```go
outputChannel := make(chan product)
workers := make(chan request, 3)

for i := 1; i <= 3; i++ {
	go newWorker(workers, outputChannel)
}

for msg := range inputChannel {
	workers <- msg
}
```

![Parallel unordered](parallel_unordered.png)

ParSeq goes just a little further from that model: it saves the order on the input side and holds a buffer on the output side to preserver order:

![Parallel ordered](parallel_ordered.png)

## Should I use this?

Probably not! Don't be clever. Only use it if:

1. the rate of input is higher than the rate of output on the system (i.e. it queues up)
2. the processing of input can be parallelised, and overall throughput increases by doing so
3. the order of output of the system needs to respect order of input

## Usage

```go
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
	time.Sleep(time.Duration(200) * time.Millisecond) // processing a request takes 1s
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
```

Sometimes it is necessary to keep per-thread/go-routine state, or other data critical for mapping the input data, that must not be shared between threads. In this case, instead of providing a single mapper instance, you can provide a slice of mappers. The size of the slice must match the `parallelism` used when creating the `parseq` instance.

```go
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
```
