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
	AsyncMode     bool
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

	if config.AsyncMode {
		go r.retryAsync(ctx, fn)
		return nil
	}

	return r.retrySync(ctx, fn)
}

// retrySync executes the function synchronously with retry logic
func (r *Retrier) retrySync(ctx context.Context, fn RetryFunc) error {
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

// retryAsync executes the function asynchronously with retry logic
func (r *Retrier) retryAsync(ctx context.Context, fn RetryFunc) {
	r.mu.RLock()
	config := r.config
	r.mu.RUnlock()

	var lastDelay time.Duration

	config.DelayStrategy.Reset()

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			if config.OnFinalError != nil {
				config.OnFinalError(attempt, fmt.Errorf("retry cancelled: %w", ctx.Err()), 0)
			}
			return
		default:
		}

		err := fn()
		if err == nil {
			if config.OnSuccess != nil {
				config.OnSuccess(attempt, nil, 0)
			}
			return
		}

		if !config.ErrorFilter(err) {
			if config.OnFinalError != nil {
				config.OnFinalError(attempt, err, 0)
			}
			return
		}

		if attempt == config.MaxAttempts {
			if config.OnFinalError != nil {
				config.OnFinalError(attempt, err, 0)
			}
			return
		}

		delay := config.DelayStrategy.NextDelay(attempt, lastDelay)
		lastDelay = delay

		if config.OnRetry != nil {
			config.OnRetry(attempt, err, delay)
		}

		select {
		case <-ctx.Done():
			if config.OnFinalError != nil {
				config.OnFinalError(attempt, fmt.Errorf("retry cancelled: %w", ctx.Err()), 0)
			}
			return
		case <-time.After(delay):
		}
	}
}

// retryAsyncWithMiddleware executes the function asynchronously with retry logic and middleware support
func (e *EnhancedRetrier) retryAsyncWithMiddleware(ctx context.Context, fn RetryFunc) {
	e.executeWithMiddlewareAsync(ctx, fn, 0)
}

func (e *EnhancedRetrier) executeWithMiddlewareAsync(ctx context.Context, fn RetryFunc, middlewareIndex int) {
	e.mu.RLock()
	middlewares := e.middlewares
	config := e.config
	e.mu.RUnlock()

	if middlewareIndex >= len(middlewares) {
		// Execute the actual retry logic with hooks
		var lastDelay time.Duration
		config.DelayStrategy.Reset()

		var totalAttempts int
		var finalErr error

		for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
			totalAttempts = attempt

			select {
			case <-ctx.Done():
				finalErr = fmt.Errorf("retry cancelled: %w", ctx.Err())
				if config.OnFinalError != nil {
					config.OnFinalError(attempt, finalErr, 0)
				}
				break
			default:
			}

			err := fn()
			if err == nil {
				if config.OnSuccess != nil {
					config.OnSuccess(attempt, nil, 0)
				}
				// Call middleware success callbacks
				e.callMiddlewareSuccess(totalAttempts)
				return
			}

			if !config.ErrorFilter(err) {
				finalErr = err
				if config.OnFinalError != nil {
					config.OnFinalError(attempt, err, 0)
				}
				break
			}

			if attempt == config.MaxAttempts {
				finalErr = err
				if config.OnFinalError != nil {
					config.OnFinalError(attempt, err, 0)
				}
				break
			}

			delay := config.DelayStrategy.NextDelay(attempt, lastDelay)
			lastDelay = delay

			if config.OnRetry != nil {
				config.OnRetry(attempt, err, delay)
			}

			select {
			case <-ctx.Done():
				finalErr = fmt.Errorf("retry cancelled: %w", ctx.Err())
				if config.OnFinalError != nil {
					config.OnFinalError(attempt, finalErr, 0)
				}
				break
			case <-time.After(delay):
			}
		}

		// If we got here, there was an error
		if finalErr != nil {
			e.callMiddlewareFailure(totalAttempts, finalErr)
		}
		return
	}

	// Skip middleware execution in async mode to avoid double-counting
	// The async execution will handle the middleware callbacks directly
	e.executeWithMiddlewareAsync(ctx, fn, middlewareIndex+1)
}

func (e *EnhancedRetrier) callMiddlewareAttempt(attempt int) {
	e.mu.RLock()
	middlewares := e.middlewares
	e.mu.RUnlock()

	for _, middleware := range middlewares {
		// Use reflection to call CallOnAttempt callback
		if hasCallOnAttempt, ok := middleware.(interface{ CallOnAttempt(int) }); ok {
			hasCallOnAttempt.CallOnAttempt(attempt)
		}
	}
}

func (e *EnhancedRetrier) callMiddlewareSuccess(totalAttempts int) {
	e.mu.RLock()
	middlewares := e.middlewares
	e.mu.RUnlock()

	for _, middleware := range middlewares {
		// Use reflection to call CallOnSuccess callback
		if hasCallOnSuccess, ok := middleware.(interface{ CallOnSuccess(int) }); ok {
			hasCallOnSuccess.CallOnSuccess(totalAttempts)
		}
	}
}

func (e *EnhancedRetrier) callMiddlewareFailure(totalAttempts int, err error) {
	e.mu.RLock()
	middlewares := e.middlewares
	e.mu.RUnlock()

	for _, middleware := range middlewares {
		// Use reflection to call CallOnFailure callback
		if hasCallOnFailure, ok := middleware.(interface{ CallOnFailure(int, error) }); ok {
			hasCallOnFailure.CallOnFailure(totalAttempts, err)
		}
	}
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
	config := e.config
	e.mu.RUnlock()

	if middlewareIndex >= len(middlewares) {
		if config.AsyncMode {
			go e.retryAsyncWithMiddleware(ctx, fn)
			return nil
		}
		return e.Retrier.retrySync(ctx, fn)
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
