
package check

import (
	"fmt"
	"runtime"
	"time"
)

var memStats runtime.MemStats

type timer struct {
	start     time.Time 
	duration  time.Duration
	N         int
	bytes     int64
	timerOn   bool
	benchTime time.Duration

	startAllocs uint64
	startBytes  uint64

	netAllocs uint64
	netBytes  uint64
}

func (c *C) StartTimer() {
	if !c.timerOn {
		c.start = time.Now()
		c.timerOn = true

		runtime.ReadMemStats(&memStats)
		c.startAllocs = memStats.Mallocs
		c.startBytes = memStats.TotalAlloc
	}
}

func (c *C) StopTimer() {
	if c.timerOn {
		c.duration += time.Now().Sub(c.start)
		c.timerOn = false
		runtime.ReadMemStats(&memStats)
		c.netAllocs += memStats.Mallocs - c.startAllocs
		c.netBytes += memStats.TotalAlloc - c.startBytes
	}
}

func (c *C) ResetTimer() {
	if c.timerOn {
		c.start = time.Now()
		runtime.ReadMemStats(&memStats)
		c.startAllocs = memStats.Mallocs
		c.startBytes = memStats.TotalAlloc
	}
	c.duration = 0
	c.netAllocs = 0
	c.netBytes = 0
}

func (c *C) SetBytes(n int64) {
	c.bytes = n
}

func (c *C) nsPerOp() int64 {
	if c.N <= 0 {
		return 0
	}
	return c.duration.Nanoseconds() / int64(c.N)
}

func (c *C) mbPerSec() float64 {
	if c.bytes <= 0 || c.duration <= 0 || c.N <= 0 {
		return 0
	}
	return (float64(c.bytes) * float64(c.N) / 1e6) / c.duration.Seconds()
}

func (c *C) timerString() string {
	if c.N <= 0 {
		return fmt.Sprintf("%3.3fs", float64(c.duration.Nanoseconds())/1e9)
	}
	mbs := c.mbPerSec()
	mb := ""
	if mbs != 0 {
		mb = fmt.Sprintf("\t%7.2f MB/s", mbs)
	}
	nsop := c.nsPerOp()
	ns := fmt.Sprintf("%10d ns/op", nsop)
	if c.N > 0 && nsop < 100 {

		if nsop < 10 {
			ns = fmt.Sprintf("%13.2f ns/op", float64(c.duration.Nanoseconds())/float64(c.N))
		} else {
			ns = fmt.Sprintf("%12.1f ns/op", float64(c.duration.Nanoseconds())/float64(c.N))
		}
	}
	memStats := ""
	if c.benchMem {
		allocedBytes := fmt.Sprintf("%8d B/op", int64(c.netBytes)/int64(c.N))
		allocs := fmt.Sprintf("%8d allocs/op", int64(c.netAllocs)/int64(c.N))
		memStats = fmt.Sprintf("\t%s\t%s", allocedBytes, allocs)
	}
	return fmt.Sprintf("%8d\t%s%s%s", c.N, ns, mb, memStats)
}

func min(x, y int) int {
	if x > y {
		return y
	}
	return x
}

func max(x, y int) int {
	if x < y {
		return y
	}
	return x
}

func roundDown10(n int) int {
	var tens = 0

	for n > 10 {
		n = n / 10
		tens++
	}

	result := 1
	for i := 0; i < tens; i++ {
		result *= 10
	}
	return result
}

func roundUp(n int) int {
	base := roundDown10(n)
	if n < (2 * base) {
		return 2 * base
	}
	if n < (5 * base) {
		return 5 * base
	}
	return 10 * base
}
