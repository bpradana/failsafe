package failsafe

import (
	"context"
	"errors"
	"time"

	"github.com/bpradana/failsafe/strategies"
)

// Option functions for configuring the retrier
func WithMaxAttempts(max int) RetryOption {
	return func(c *RetryConfig) {
		c.MaxAttempts = max
	}
}

func WithDelayStrategy(strategy DelayStrategy) RetryOption {
	return func(c *RetryConfig) {
		c.DelayStrategy = strategy
	}
}

func WithErrorFilter(filter ErrorFilter) RetryOption {
	return func(c *RetryConfig) {
		c.ErrorFilter = filter
	}
}

func WithOnRetry(hook Hook) RetryOption {
	return func(c *RetryConfig) {
		c.OnRetry = hook
	}
}

func WithOnFinalError(hook Hook) RetryOption {
	return func(c *RetryConfig) {
		c.OnFinalError = hook
	}
}

func WithOnSuccess(hook Hook) RetryOption {
	return func(c *RetryConfig) {
		c.OnSuccess = hook
	}
}

func WithAsyncMode(async bool) RetryOption {
	return func(c *RetryConfig) {
		c.AsyncMode = async
	}
}

// Common error filters
var (
	RetryAllErrors = func(err error) bool { return true }

	RetryTransientErrors = func(err error) bool {
		if errors.Is(err, context.DeadlineExceeded) ||
			errors.Is(err, context.Canceled) {
			return false
		}
		return true
	}
)

// Convenience functions for common use cases
func RetryWithExponentialBackoff(ctx context.Context, fn RetryFunc, maxAttempts int) error {
	retrier := NewRetrier(
		WithMaxAttempts(maxAttempts),
		WithDelayStrategy(strategies.ExponentialBackoffWithJitter(100*time.Millisecond, 5*time.Second, 2.0)),
		WithErrorFilter(RetryTransientErrors),
	)

	return retrier.Retry(ctx, fn)
}

func RetryWithLogging(ctx context.Context, fn RetryFunc, maxAttempts int, logger func(string, ...interface{})) error {
	retrier := NewRetrier(
		WithMaxAttempts(maxAttempts),
		WithDelayStrategy(strategies.ExponentialBackoffWithJitter(100*time.Millisecond, 5*time.Second, 2.0)),
		WithOnRetry(func(attempt int, err error, nextDelay time.Duration) {
			logger("Retry attempt %d failed: %v, next delay: %v", attempt, err, nextDelay)
		}),
		WithOnFinalError(func(attempt int, err error, nextDelay time.Duration) {
			logger("Final retry attempt %d failed: %v", attempt, err)
		}),
		WithOnSuccess(func(attempt int, err error, nextDelay time.Duration) {
			logger("Success on attempt %d", attempt)
		}),
	)

	return retrier.Retry(ctx, fn)
}
