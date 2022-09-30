package parseq_test

import (
	"runtime"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/marianogappa/parseq"

	"math/rand"
)

func TestOutperformsSequential(t *testing.T) {
	p, err := parseq.NewWithMapper[int, int](5, &ConstantDelayMapper{
		delay: 50 * time.Millisecond,
	})
	if err != nil {
		panic(err)
	}

	go p.Start()
	go func() {
		p.Input <- 666
		time.Sleep(10 * time.Millisecond)
		p.Input <- 667
		time.Sleep(10 * time.Millisecond)
		p.Input <- 668
		time.Sleep(10 * time.Millisecond)
		p.Input <- 669
		time.Sleep(10 * time.Millisecond)
		p.Input <- 670
	}()

	start := time.Now()
	<-p.Output // min(elapsed)=50ms
	<-p.Output // min(elapsed)=60ms
	<-p.Output // min(elapsed)=70ms
	<-p.Output // min(elapsed)=80ms
	<-p.Output // min(elapsed)=90ms
	elapsed := time.Since(start)

	if elapsed > 150*time.Millisecond { // 150ms for
		t.Error("test took too long; parallel strategy is ineffective!")
	}
	t.Log("min(elapsed) was 90ms; took", elapsed, "against sequential time of 250ms")

	p.Close()
}

func TestOrderedOutput(t *testing.T) {
	r := rand.New(rand.NewSource(99))
	p, err := parseq.NewWithMapper[int, int](5, &RandomDelayMapper{
		r: r,
	})
	if err != nil {
		panic(err)
	}

	go p.Start()
	go func() {
		p.Input <- 666
		time.Sleep(10 * time.Millisecond)
		p.Input <- 667
		time.Sleep(10 * time.Millisecond)
		p.Input <- 668
		time.Sleep(10 * time.Millisecond)
		p.Input <- 669
		time.Sleep(10 * time.Millisecond)
		p.Input <- 670
		p.Close()
	}()

	a := <-p.Output
	b := <-p.Output
	c := <-p.Output
	d := <-p.Output
	time.Sleep(10 * time.Millisecond)
	e := <-p.Output

	if a != 666 ||
		b != 667 ||
		c != 668 ||
		d != 669 ||
		e != 670 {
		t.Error("output came out out of order: ", a, b, c, d, e)
	}
}

func TestCloseNotEatingResult(t *testing.T) {
	p, err := parseq.NewWithMapper[int, int](5, &ConstantDelayMapper{
		delay: 50 * time.Millisecond,
	})
	if err != nil {
		panic(err)
	}
	sendTotal := 5

	go p.Start()
	go func() {
		for i := 666; i < 666+sendTotal; i++ {
			p.Input <- i
			time.Sleep(10 * time.Millisecond)
		}
	}()

	res := make([]int, sendTotal)
	for i := 0; i < sendTotal-2; i++ {
		res[i] = <-p.Output
	}

	// close first
	p.Close()

	// then try reading last `parallelism + 1` results
	for i := sendTotal - 2; i < sendTotal; i++ {
		res[i] = <-p.Output
	}

	isSorted := sort.SliceIsSorted(res, func(i, j int) bool {
		return res[i] < res[j]
	})
	if !isSorted {
		t.Error("output is not sorted")
	}
}

// Assign benchmark results to global var so the compiler won't optimize
// away parts of the benchmark test.
var result int

func BenchmarkNoop(b *testing.B) {
	numThreads := runtime.NumCPU()

	p, err := parseq.NewWithMapper[int, int](numThreads, &NoopMapper{})
	if err != nil {
		panic(err)
	}
	go p.Start()

	go func() {
		for n := 0; n < b.N; n++ {
			p.Input <- 0
		}
		p.Close()
	}()

	for r := range p.Output {
		result = r
	}
}

type NoopMapper struct{}

func (p *NoopMapper) Map(i int) int {
	return i
}

type ConstantDelayMapper struct {
	delay time.Duration
}

func (p *ConstantDelayMapper) Map(i int) int {
	time.Sleep(time.Duration(p.delay))
	return i
}

type RandomDelayMapper struct {
	r  *rand.Rand
	mu sync.Mutex
}

func (p *RandomDelayMapper) Map(i int) int {
	p.mu.Lock()
	rnd := p.r.Intn(41)
	p.mu.Unlock()
	time.Sleep(time.Duration(rnd+10) * time.Millisecond) //sleep between
	return i
}
