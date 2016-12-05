package parseq

import (
	"testing"
	"time"

	"math/rand"

	"github.com/MarianoGappa/parseq"
)

func TestOutperformsSequential(t *testing.T) {
	p := parseq.NewParSeq(5, processAfter(50*time.Millisecond))

	go p.Run()
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
	p := parseq.NewParSeq(5, processAfterRandom(r))

	go p.Run()
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

	a := <-p.Output
	b := <-p.Output
	c := <-p.Output
	d := <-p.Output
	e := <-p.Output

	if a.(int) != 666 ||
		b.(int) != 667 ||
		c.(int) != 668 ||
		d.(int) != 669 ||
		e.(int) != 670 {
		t.Error("output came out out of order: ", a, b, c, d, e)
	}

	p.Close()
}

func processAfter(d time.Duration) func(interface{}) interface{} {
	return func(v interface{}) interface{} {
		time.Sleep(d)
		return v
	}
}

func processAfterRandom(r *rand.Rand) func(interface{}) interface{} {
	return func(v interface{}) interface{} {
		time.Sleep(time.Duration(r.Intn(41)+10) * time.Millisecond) //sleep between 10ms and 50ms
		return v
	}
}
