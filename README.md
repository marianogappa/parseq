# ParSeq

Parallel processing with sequential output (respecting order of input).

## Should I use this?

Probably not! Don't be clever. Only use it if:

1. the rate of input is higher than the rate of output on the system (i.e. it queues up)
2. the processing of input can be parallelised, and overall throughput increases by doing so
3. the order of output of the system needs to respect order of input

## Usage

```
package main

import (
	"fmt"
	"time"

	"github.com/MarianoGappa/parseq"
)

func main() {
	p := parseq.New(5, process)			// 5 goroutines using the process function

	go p.Start()
	go makeRequests(p)

	for out := range p.Output {			// after initial 1s, requests output every ~200ms
		fmt.Print(out.(int), ".")		// and output respects input order
	}
}

func makeRequests(p parseq.ParSeq) {
	counter := 666
	for {
		p.Input <- counter			// this simulates an incoming request
		time.Sleep(200 * time.Millisecond)	// requests come every 200ms
		counter++
	}
}

func process(value interface{}) interface{} {
	time.Sleep(1 * time.Second)			// processing a request takes 1s
	return value
}
```
