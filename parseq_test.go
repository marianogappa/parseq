package parseq_test

import (
	"reflect"
	"runtime"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/marianogappa/parseq"

	"math/rand"
)

func TestOutperformsSequential(t *testing.T) {
	p := parseq.New(5, processAfter(50*time.Millisecond))

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
	elapsed := time.Now().Sub(start)

	if elapsed > 150*time.Millisecond { // 150ms for
		t.Error("test took too long; parallel strategy is ineffective!")
	}
	t.Log("min(elapsed) was 90ms; took", elapsed, "against sequential time of 250ms")

	p.Close()
}

func TestOrderedOutput(t *testing.T) {
	r := rand.New(rand.NewSource(99))
	p := parseq.New(5, processAfterRandom(r))

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

	if a.(int) != 666 ||
		b.(int) != 667 ||
		c.(int) != 668 ||
		d.(int) != 669 ||
		e.(int) != 670 {
		t.Error("output came out out of order: ", a, b, c, d, e)
	}
}

func TestCloseNotEatingResult(t *testing.T) {
	p := parseq.New(5, processAfter(50*time.Millisecond))
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
		r := <-p.Output
		if rint, ok := (r).(int); ok {
			res[i] = rint
		} else {
			t.Errorf("received unexpected type %s", reflect.TypeOf(r))
			return
		}
	}

	// close first
	p.Close()

	// then try reading last `parallelism + 1` results
	for i := sendTotal - 2; i < sendTotal; i++ {
		r := <-p.Output
		if rint, ok := (r).(int); ok {
			res[i] = rint
		} else {
			t.Errorf("received unexpected type %s", reflect.TypeOf(r))
			return
		}
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
var result interface{}

func BenchmarkNoop(b *testing.B) {
	numThreads := runtime.NumCPU()

	p := parseq.New(numThreads, noopProcessor)
	go p.Start()

	go func() {
		for n := 0; n < b.N; n++ {
			p.Input <- true
		}
		p.Close()
	}()

	for r := range p.Output {
		result = r
	}
}

func noopProcessor(v interface{}) interface{} {
	return v
}

func processAfter(d time.Duration) parseq.ProcessFunc {
	return func(v interface{}) interface{} {
		time.Sleep(d)
		return v
	}
}

func processAfterRandom(r *rand.Rand) parseq.ProcessFunc {
	var mu sync.Mutex
	return func(v interface{}) interface{} {
		mu.Lock()
		rnd := r.Intn(41)
		mu.Unlock()
		time.Sleep(time.Duration(rnd+10) * time.Millisecond) //sleep between 10ms and 50ms
		return v
	}
}
