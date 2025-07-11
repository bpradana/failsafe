package failsafe

import (
	"context"
	"time"
)

// RetryFunc represents a function that can be retried
type RetryFunc func() error

// RetryFuncWithResult represents a function that returns a result and error
type RetryFuncWithResult[T any] func() (T, error)

// DelayStrategy defines how delays are calculated between retries
type DelayStrategy interface {
	NextDelay(attempt int, lastDelay time.Duration) time.Duration
	Reset()
}

// ErrorFilter determines if an error should trigger a retry
type ErrorFilter func(error) bool

// Hook represents a callback function for retry events
type Hook func(ctx context.Context, attempt int, err error, nextDelay time.Duration)

// Middleware interface for extensible patterns
type Middleware interface {
	Execute(ctx context.Context, fn func() error, next func(context.Context, func() error) error) error
}

// Circuit breaker interface for future extensibility
type CircuitBreaker interface {
	Allow() bool
	RecordSuccess()
	RecordFailure()
	State() string
}

// Rate limiter interface for future extensibility
type RateLimiter interface {
	Allow() bool
	Wait(ctx context.Context) error
}

// Default configurations
var (
	DefaultMaxAttempts = 3
	DefaultDelay       = 1 * time.Second
)

// FixedDelay implements a fixed delay strategy
type FixedDelay struct {
	Delay time.Duration
}

func (f *FixedDelay) NextDelay(attempt int, lastDelay time.Duration) time.Duration {
	return f.Delay
}

func (f *FixedDelay) Reset() {}
