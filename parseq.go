package parseq

import "sync"

type par struct {
	Input  chan Processable
	Output chan interface{}

	parallelism int
	order       int64
	unresolved  []int64
	l           sync.Mutex
	work        chan input
	outs        chan output
}

func NewPar(parallelism int) par {
	return par{
		Input:  make(chan Processable, parallelism),
		Output: make(chan interface{}),

		parallelism: parallelism,
		work:        make(chan input, parallelism),
		outs:        make(chan output, parallelism),
	}
}

func (p *par) Run() {
	go p.readRequests()
	go p.orderResults()

	for i := 0; i < p.parallelism; i++ {
		go p.processRequests()
	}
}

func (p *par) readRequests() {
	for r := range p.Input {
		p.order++
		p.l.Lock()
		p.unresolved = append(p.unresolved, p.order)
		p.l.Unlock()
		p.work <- input{order: p.order, request: r}
	}
}

func (p *par) processRequests() {
	for r := range p.work {
		p.outs <- output{order: r.order, product: r.request.Process()}
	}
}

func (p *par) orderResults() {
	rtBuf := make(map[int64]bool)
	for pr := range p.outs {
		rtBuf[pr.order] = true
	loop:
		if len(p.unresolved) > 0 {
			u := p.unresolved[0]
			if rtBuf[u] {
				delete(rtBuf, u)
				p.l.Lock()
				p.unresolved = p.unresolved[1:]
				p.l.Unlock()
				p.Output <- pr.product
				goto loop
			}
		}
	}
}

type Processable interface {
	Process() interface{}
}

type input struct {
	request Processable
	order   int64
}

type output struct {
	product interface{}
	order   int64
}
