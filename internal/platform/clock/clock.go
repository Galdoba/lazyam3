package clock

import "time"

// Clock abstracts time for testability.
type Clock interface {
	Now() time.Time
	Since(t time.Time) time.Duration
	After(d time.Duration) <-chan time.Time
	Ticker(d time.Duration) *Ticker
}

// RealClock is the real-world clock implementation.
type RealClock struct{}

func (RealClock) Now() time.Time                         { return time.Now() }
func (RealClock) Since(t time.Time) time.Duration        { return time.Since(t) }
func (RealClock) After(d time.Duration) <-chan time.Time { return time.After(d) }

func (RealClock) Ticker(d time.Duration) *Ticker {
	return &Ticker{c: time.NewTicker(d)}
}

// Ticker wraps time.Ticker.
type Ticker struct {
	c *time.Ticker
}

func (t *Ticker) C() <-chan time.Time   { return t.c.C }
func (t *Ticker) Stop()                 { t.c.Stop() }
func (t *Ticker) Reset(d time.Duration) { t.c.Reset(d) }
