package strategies

import (
	"sync"
	"time"
)

// FixedDelay implements a fixed delay strategy
type FixedDelay struct {
	Delay time.Duration
}

func (f *FixedDelay) NextDelay(attempt int, lastDelay time.Duration) time.Duration {
	return f.Delay
}

func (f *FixedDelay) Reset() {}

// ExponentialBackoff implements exponential backoff delay strategy
type ExponentialBackoff struct {
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	mu           sync.Mutex
}

func (e *ExponentialBackoff) NextDelay(attempt int, lastDelay time.Duration) time.Duration {
	e.mu.Lock()
	defer e.mu.Unlock()

	if attempt == 1 {
		return e.InitialDelay
	}

	delay := time.Duration(float64(lastDelay) * e.Multiplier)
	if delay > e.MaxDelay {
		delay = e.MaxDelay
	}
	return delay
}

func (e *ExponentialBackoff) Reset() {}

// Constructor functions
func NewFixedDelay(delay time.Duration) *FixedDelay {
	return &FixedDelay{Delay: delay}
}

func NewExponentialBackoff(initial, max time.Duration, multiplier float64) *ExponentialBackoff {
	return &ExponentialBackoff{
		InitialDelay: initial,
		MaxDelay:     max,
		Multiplier:   multiplier,
	}
}
