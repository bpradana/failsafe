package middleware

import (
	"context"
	"errors"
	"fmt"
)

// RateLimiter interface
type RateLimiter interface {
	Allow() bool
	Wait(ctx context.Context) error
}

// RateLimitMiddleware wraps retry logic with rate limiting
type RateLimitMiddleware struct {
	RateLimiter RateLimiter
}

func (r *RateLimitMiddleware) Execute(ctx context.Context, fn func() error, next func(context.Context, func() error) error) error {
	if err := r.RateLimiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limit wait failed: %w", err)
	}

	wrappedFn := func() error {
		if !r.RateLimiter.Allow() {
			return errors.New("rate limit exceeded")
		}
		return fn()
	}

	return next(ctx, wrappedFn)
}

// NewRateLimitMiddleware creates a new rate limit middleware
func NewRateLimitMiddleware(rl RateLimiter) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		RateLimiter: rl,
	}
}
