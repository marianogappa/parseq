package parseq

import "sync"

type ParSeq struct {
	Input  chan interface{}
	Output chan interface{}

	parallelism int
	order       int64
	unresolved  []int64
	l           sync.Mutex
	work        chan input
	outs        chan output
	process     func(interface{}) interface{}
}

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

func (p *ParSeq) Start() {
	go p.readRequests()
	go p.orderResults()

	for i := 0; i < p.parallelism; i++ {
		go p.processRequests()
	}
}

func (p *ParSeq) Close() {
	close(p.Input)
	close(p.Output)
	close(p.work)
	close(p.outs)
}

func (p *ParSeq) readRequests() {
	for r := range p.Input {
		p.order++
		p.l.Lock()
		p.unresolved = append(p.unresolved, p.order)
		p.l.Unlock()
		p.work <- input{order: p.order, request: r}
	}
}

func (p *ParSeq) processRequests() {
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
}

type input struct {
	request interface{}
	order   int64
}

type output struct {
	product interface{}
	order   int64
}
