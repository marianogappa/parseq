// The parseq package provides a simple interface for processing a stream in parallel,
// with configurable level of parallelism, while still outputting a sequential stream
// that respects the order of input.
package parseq

import (
	"sync"
)

type ParSeq struct {
	// Input is the channel the client code should send to.
	Input chan interface{}

	// Output is the channel the client code should recieve from. Output order respects
	// input order. Output is usually casted to a more useful type.
	Output chan interface{}

	parallelism int
	work        []chan interface{}
	outs        []chan interface{}
	process     ProcessFunc
	wg          sync.WaitGroup
}

type ProcessFunc func(interface{}) interface{}

// New returns a new ParSeq. Processing doesn't begin until the Start method is called.
// ParSeq is concurrency-safe; multiple ParSeqs can run in parallel.
// `parallelism` determines how many goroutines read from the Input channel, and each
// of the goroutines uses the `process` function to process the inputs.
func New(parallelism int, process ProcessFunc) *ParSeq {
	work := make([]chan interface{}, parallelism)
	outs := make([]chan interface{}, parallelism)
	for i := 0; i < parallelism; i++ {
		work[i] = make(chan interface{}, parallelism)
		outs[i] = make(chan interface{}, parallelism)
	}

	return &ParSeq{
		Input:  make(chan interface{}, parallelism),
		Output: make(chan interface{}, parallelism),

		parallelism: parallelism,
		work:        work,
		outs:        outs,
		process:     process,
	}
}

// Start begins consuming the Input channel and producing to the Output channel.
// It starts n+2 goroutines, n being the level of parallelism, so Close should be
// called to exit the goroutines after processing has finished.
func (p *ParSeq) Start() {
	go p.readRequests()
	go p.orderResults()

	for i := 0; i < p.parallelism; i++ {
		p.wg.Add(1)
		go p.processRequests(p.work[i], p.outs[i], p.process)
	}

	go func() {
		p.wg.Wait()
		for _, o := range p.outs {
			close(o)
		}
	}()
}

// Close waits for all queued messages to process, and stops the ParSeq.
// This ParSeq cannot be used after calling Close(). You must not send
// to the Input channel after calling Close().
func (p *ParSeq) Close() {
	close(p.Input)
	p.wg.Wait()
}

func (p *ParSeq) readRequests() {
	i := 0
	for r := range p.Input {
		p.work[i%p.parallelism] <- r
		i++
		if i >= p.parallelism {
			i = 0
		}
	}
	for _, w := range p.work {
		close(w)
	}
}

func (p *ParSeq) processRequests(in chan interface{}, out chan interface{}, process ProcessFunc) {
	defer p.wg.Done()

	for r := range in {
		out <- process(r)
	}
}

func (p *ParSeq) orderResults() {
	for {
		for i := 0; i < p.parallelism; i++ {
			val, ok := <-p.outs[i]
			if !ok {
				close(p.Output)
				return
			}
			p.Output <- val
		}
	}
}
