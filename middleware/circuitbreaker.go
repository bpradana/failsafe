package middleware

import (
	"context"
	"errors"
)

// CircuitBreaker interface
type CircuitBreaker interface {
	Allow() bool
	RecordSuccess()
	RecordFailure()
	State() string
}

// CircuitBreakerMiddleware wraps retry logic with circuit breaker pattern
type CircuitBreakerMiddleware struct {
	CircuitBreaker CircuitBreaker
}

func (c *CircuitBreakerMiddleware) Execute(ctx context.Context, fn func() error, next func(context.Context, func() error) error) error {
	if !c.CircuitBreaker.Allow() {
		return errors.New("circuit breaker is open")
	}

	wrappedFn := func() error {
		err := fn()
		if err != nil {
			c.CircuitBreaker.RecordFailure()
		} else {
			c.CircuitBreaker.RecordSuccess()
		}
		return err
	}

	return next(ctx, wrappedFn)
}

// NewCircuitBreakerMiddleware creates a new circuit breaker middleware
func NewCircuitBreakerMiddleware(cb CircuitBreaker) *CircuitBreakerMiddleware {
	return &CircuitBreakerMiddleware{
		CircuitBreaker: cb,
	}
}
