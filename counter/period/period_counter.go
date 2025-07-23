package period

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/tutils/tnet/counter"
)

var _ counter.Counter = &periodCounter{}

type periodCounter struct {
	value              int64
	period             time.Duration
	epoch              time.Time
	increaceRatePerSec int64

	lastValue int64
	lastTime  time.Time
	mut       sync.Mutex
}

func NewPeriodCounter(period time.Duration) counter.Counter {
	now := time.Now()
	c := &periodCounter{
		period:   period,
		epoch:    now,
		lastTime: now,
	}
	return c
}

// Value implements Counter.
func (c *periodCounter) Value() int64 {
	return atomic.LoadInt64(&c.value)
}

// IncreaceRatePerSec implements Counter.
func (c *periodCounter) IncreaceRatePerSec() int64 {
	return atomic.LoadInt64(&c.increaceRatePerSec)
}

// Add implements Counter.
func (c *periodCounter) Add(bytes int64) {
	atomic.AddInt64(&c.value, bytes)
	c.check()
}

func (c *periodCounter) check() {
	c.mut.Lock()
	defer c.mut.Unlock()

	elapsed := time.Since(c.lastTime)
	if elapsed < c.period {
		return
	}

	now := time.Now()
	value := c.Value()
	atomic.StoreInt64(&c.increaceRatePerSec, int64(float64(value-c.lastValue)/elapsed.Seconds()))
	c.lastValue = value
	c.lastTime = now
}
