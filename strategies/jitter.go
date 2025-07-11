package strategies

import (
	"math/rand"
	"sync"
	"time"
)

// JitterStrategy wraps a base delay strategy with jitter
type JitterStrategy struct {
	BaseStrategy DelayStrategy
	JitterFunc   func(time.Duration) time.Duration
	mu           sync.RWMutex
}

// DelayStrategy interface for jitter package
type DelayStrategy interface {
	NextDelay(attempt int, lastDelay time.Duration) time.Duration
	Reset()
}

func (j *JitterStrategy) NextDelay(attempt int, lastDelay time.Duration) time.Duration {
	j.mu.RLock()
	defer j.mu.RUnlock()

	baseDelay := j.BaseStrategy.NextDelay(attempt, lastDelay)
	return j.JitterFunc(baseDelay)
}

func (j *JitterStrategy) Reset() {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.BaseStrategy.Reset()
}

// Common jitter functions
var (
	UniformJitter = func(delay time.Duration) time.Duration {
		jitter := time.Duration(rand.Float64() * float64(delay))
		return delay + jitter
	}

	ExponentialJitter = func(delay time.Duration) time.Duration {
		jitter := time.Duration(rand.Float64() * float64(delay) * 0.1)
		return delay + jitter
	}
)

// Constructor functions
func NewJitterStrategy(base DelayStrategy, jitterFunc func(time.Duration) time.Duration) *JitterStrategy {
	return &JitterStrategy{
		BaseStrategy: base,
		JitterFunc:   jitterFunc,
	}
}

// Utility function for exponential backoff with jitter
func ExponentialBackoffWithJitter(initial, max time.Duration, multiplier float64) DelayStrategy {
	backoff := NewExponentialBackoff(initial, max, multiplier)
	return NewJitterStrategy(backoff, ExponentialJitter)
}
