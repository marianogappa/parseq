// The parseq package provides a simple interface for processing a stream in parallel,
// with configurable level of parallelism, while still outputting a sequential stream
// that respects the order of input.
package parseq

import "sync"

type ParSeq struct {
	// Input is the channel the client code should send to.
	Input chan interface{}

	// Output is the channel the client code should recieve from. Output order respects
	// input order. Output is usually casted to a more useful type.
	Output chan interface{}

	parallelism int
	order       int64
	unresolved  []int64
	l           sync.Mutex
	work        chan input
	outs        chan output
	process     func(interface{}) interface{}
}

// New returns a new ParSeq. Processing doesn't begin until the Start method is called.
// ParSeq is concurrency-safe; multiple ParSeqs can run in parallel.
// `parallelism` determines how many goroutines read from the Input channel, and each
// of the goroutines uses the `process` function to process the inputs.
func New(parallelism int, process func(interface{}) interface{}) ParSeq {
	return ParSeq{
		Input:  make(chan interface{}, parallelism),
		Output: make(chan interface{}),

		parallelism: parallelism,
		work:        make(chan input, parallelism),
		outs:        make(chan output, parallelism),
		process:     process,
	}
}

// Start begins consuming the Input channel and producing to the Output channel.
// It starts n+2 goroutines, n being the level of parallelism, so Close should be
// called to exit the goroutines after processing has finished.
func (p *ParSeq) Start() {
	go p.readRequests()
	go p.orderResults()

	var wg sync.WaitGroup
	for i := 0; i < p.parallelism; i++ {
		wg.Add(1)
		go p.processRequests(&wg)
	}

	go func(wg *sync.WaitGroup) {
		wg.Wait()
		close(p.outs)
	}(&wg)
}

// Close waits for all queued messages to process, and stops the ParSeq.
// This ParSeq cannot be used after calling Close(). You must not send
// to the Input channel after calling Close().
func (p *ParSeq) Close() {
	close(p.Input)
	<-p.Output
}

func (p *ParSeq) readRequests() {
	for r := range p.Input {
		p.order++
		p.l.Lock()
		p.unresolved = append(p.unresolved, p.order)
		p.l.Unlock()
		p.work <- input{order: p.order, request: r}
	}
	close(p.work)
}

func (p *ParSeq) processRequests(wg *sync.WaitGroup) {
	defer wg.Done()

	for r := range p.work {
		p.outs <- output{order: r.order, product: p.process(r.request)}
	}
}

func (p *ParSeq) orderResults() {
	rtBuf := make(map[int64]interface{})
	for pr := range p.outs {
		rtBuf[pr.order] = pr.product
	loop:
		if len(p.unresolved) > 0 {
			u := p.unresolved[0]
			if rtBuf[u] != nil {
				p.l.Lock()
				p.unresolved = p.unresolved[1:]
				p.l.Unlock()
				p.Output <- rtBuf[u]
				delete(rtBuf, u)
				goto loop
			}
		}
	}
	close(p.Output)
}

type input struct {
	request interface{}
	order   int64
}

type output struct {
	product interface{}
	order   int64
}
