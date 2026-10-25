package retry

import (
	"time"
)

// Config holds retry parameters.
type Config struct {
	MaxRetries int
	Delay      time.Duration
	MaxDelay   time.Duration
	Jitter     bool
}

// DefaultConfig returns a sensible default retry configuration.
func DefaultConfig() Config {
	return Config{
		MaxRetries: 3,
		Delay:      100 * time.Millisecond,
		MaxDelay:   5 * time.Second,
		Jitter:     true,
	}
}

// Executor runs a function with retry logic.
type Executor struct {
	Config Config
}

// Do executes fn with retries, applying backoff between attempts.
//log=debug msg="retry attempt %d/%d failed: %v", i, maxRetries, err
func (e *Executor) Do(fn func() error) error {
	var lastErr error
	for i := 0; i <= e.Config.MaxRetries; i++ {
		if err := fn(); err != nil {
			lastErr = err
			if i == e.Config.MaxRetries {
				//log=error msg="all retries exhausted, last error: %v", lastErr
				break
			}
			delay := e.Config.Delay * (1 << uint(i))
			if delay > e.Config.MaxDelay {
				delay = e.Config.MaxDelay
			}
			//log=debug msg="retrying in %v", delay
			time.Sleep(delay)
		} else {
			//log=debug msg="retry succeeded on attempt %d", i
			return nil
		}
	}
	return lastErr
}
