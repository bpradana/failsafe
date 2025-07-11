package failsafe

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RetryConfig holds configuration for retry behavior
type RetryConfig struct {
	MaxAttempts   int
	DelayStrategy DelayStrategy
	ErrorFilter   ErrorFilter
	OnRetry       Hook
	OnFinalError  Hook
	OnSuccess     Hook
}

// Retrier provides retry functionality with configurable options
type Retrier struct {
	config RetryConfig
	mu     sync.RWMutex
}

// RetryOption represents a configuration option for Retrier
type RetryOption func(*RetryConfig)

// EnhancedRetrier with middleware support
type EnhancedRetrier struct {
	*Retrier
	middlewares []Middleware
	mu          sync.RWMutex
}

// NewRetrier creates a new retrier with the given options
func NewRetrier(options ...RetryOption) *Retrier {
	config := RetryConfig{
		MaxAttempts:   DefaultMaxAttempts,
		DelayStrategy: &FixedDelay{Delay: DefaultDelay},
		ErrorFilter:   RetryAllErrors,
	}

	for _, option := range options {
		option(&config)
	}

	return &Retrier{config: config}
}

// NewEnhancedRetrier creates a new enhanced retrier with middleware support
func NewEnhancedRetrier(options ...RetryOption) *EnhancedRetrier {
	return &EnhancedRetrier{
		Retrier: NewRetrier(options...),
	}
}

// Retry executes the given function with retry logic
func (r *Retrier) Retry(ctx context.Context, fn RetryFunc) error {
	r.mu.RLock()
	config := r.config
	r.mu.RUnlock()

	var lastErr error
	var lastDelay time.Duration

	config.DelayStrategy.Reset()

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return fmt.Errorf("retry cancelled: %w", ctx.Err())
		default:
		}

		err := fn()
		if err == nil {
			if config.OnSuccess != nil {
				config.OnSuccess(attempt, nil, 0)
			}
			return nil
		}

		lastErr = err

		if !config.ErrorFilter(err) {
			if config.OnFinalError != nil {
				config.OnFinalError(attempt, err, 0)
			}
			return fmt.Errorf("non-retryable error: %w", err)
		}

		if attempt == config.MaxAttempts {
			if config.OnFinalError != nil {
				config.OnFinalError(attempt, err, 0)
			}
			return fmt.Errorf("max attempts reached: %w", err)
		}

		delay := config.DelayStrategy.NextDelay(attempt, lastDelay)
		lastDelay = delay

		if config.OnRetry != nil {
			config.OnRetry(attempt, err, delay)
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("retry cancelled: %w", ctx.Err())
		case <-time.After(delay):
		}
	}

	return lastErr
}

// AddMiddleware adds middleware to the enhanced retrier
func (e *EnhancedRetrier) AddMiddleware(middleware Middleware) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.middlewares = append(e.middlewares, middleware)
}

// Retry executes the given function with middleware and retry logic
func (e *EnhancedRetrier) Retry(ctx context.Context, fn RetryFunc) error {
	return e.executeWithMiddleware(ctx, fn, 0)
}

func (e *EnhancedRetrier) executeWithMiddleware(ctx context.Context, fn RetryFunc, middlewareIndex int) error {
	e.mu.RLock()
	middlewares := e.middlewares
	e.mu.RUnlock()

	if middlewareIndex >= len(middlewares) {
		return e.Retrier.Retry(ctx, fn)
	}

	middleware := middlewares[middlewareIndex]
	return middleware.Execute(ctx, fn, func(ctx context.Context, fn func() error) error {
		return e.executeWithMiddleware(ctx, fn, middlewareIndex+1)
	})
}

// UpdateConfig updates the retrier configuration
func (r *Retrier) UpdateConfig(options ...RetryOption) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, option := range options {
		option(&r.config)
	}
}

// GetConfig returns a copy of the current configuration
func (r *Retrier) GetConfig() RetryConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.config
}
