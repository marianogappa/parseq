### Readme coming soon, you can use it like this:
(note how even though messages take 5s to process, output comes out every second, and it comes out ordered)

```
package main

import (
	"fmt"
	"time"

	"github.com/MarianoGappa/parseq"
)

func main() {
	p := parseq.NewParSeq(5)
	go p.Run()

	go makeRequests(p.Input, 1*time.Second)

	for out := range p.Output {
		tp := out.(testProduct)
		fmt.Print(tp.ord, "-")
	}
}

func makeRequests(ins chan parseq.Processable, requestEvery time.Duration) {
	counter := int64(665)
	for {
		counter++
		ins <- testRequest{counter}
		time.Sleep(requestEvery)
	}
}

type testRequest struct{ ord int64 }
type testProduct struct{ ord int64 }

func (r testRequest) Process() interface{} {
	time.Sleep(5 * time.Second)
	return testProduct{r.ord}
}
```
