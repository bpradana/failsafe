package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/bpradana/failsafe"
	"github.com/bpradana/failsafe/middleware"
)

// SimpleCircuitBreaker implements a basic circuit breaker
type SimpleCircuitBreaker struct {
	mu          sync.Mutex
	failures    int
	lastFailure time.Time
	threshold   int
	timeout     time.Duration
	state       string
}

func NewSimpleCircuitBreaker(threshold int, timeout time.Duration) *SimpleCircuitBreaker {
	return &SimpleCircuitBreaker{
		threshold: threshold,
		timeout:   timeout,
		state:     "closed",
	}
}

func (cb *SimpleCircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == "open" {
		if time.Since(cb.lastFailure) > cb.timeout {
			cb.state = "half-open"
			return true
		}
		return false
	}
	return true
}

func (cb *SimpleCircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures = 0
	cb.state = "closed"
}

func (cb *SimpleCircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures++
	cb.lastFailure = time.Now()
	if cb.failures >= cb.threshold {
		cb.state = "open"
	}
}

func (cb *SimpleCircuitBreaker) State() string {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// SimpleRateLimiter implements a basic token bucket rate limiter
type SimpleRateLimiter struct {
	mu       sync.Mutex
	tokens   int
	capacity int
	rate     time.Duration
	lastFill time.Time
}

func NewSimpleRateLimiter(capacity int, rate time.Duration) *SimpleRateLimiter {
	return &SimpleRateLimiter{
		tokens:   capacity,
		capacity: capacity,
		rate:     rate,
		lastFill: time.Now(),
	}
}

func (rl *SimpleRateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.refill()
	if rl.tokens > 0 {
		rl.tokens--
		return true
	}
	return false
}

func (rl *SimpleRateLimiter) Wait(ctx context.Context) error {
	for {
		if rl.Allow() {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(rl.rate):
			continue
		}
	}
}

func (rl *SimpleRateLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(rl.lastFill)
	tokensToAdd := int(elapsed / rl.rate)

	if tokensToAdd > 0 {
		rl.tokens = minInt(rl.capacity, rl.tokens+tokensToAdd)
		rl.lastFill = now
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
	fmt.Println("=== Middleware Examples ===")

	metricsExample()
	fmt.Println()
	circuitBreakerExample()
	fmt.Println()
	rateLimitExample()
	fmt.Println()
	combinedExample()
}

func metricsExample() {
	fmt.Println("--- Metrics Middleware Example ---")

	ctx := context.Background()

	// Create enhanced retrier with middleware support
	retrier := failsafe.NewEnhancedRetrier(
		failsafe.WithMaxAttempts(3),
		failsafe.WithDelayStrategy(&failsafe.FixedDelay{Delay: 200 * time.Millisecond}),
	)

	// Add metrics middleware
	metricsMiddleware := middleware.NewMetricsMiddleware(
		func(attempt int) {
			fmt.Printf("Metrics: Starting attempt %d\n", attempt)
		},
		func(totalAttempts int) {
			fmt.Printf("Metrics: Success after %d attempts\n", totalAttempts)
		},
		func(totalAttempts int, err error) {
			fmt.Printf("Metrics: Failed after %d attempts: %v\n", totalAttempts, err)
		},
	)

	retrier.AddMiddleware(metricsMiddleware)

	// Simulate a function that fails once then succeeds
	attempts := 0
	err := retrier.Retry(ctx, func() error {
		attempts++
		fmt.Printf("Executing function attempt %d\n", attempts)

		if attempts < 2 {
			return errors.New("simulated failure")
		}
		return nil
	})

	if err != nil {
		log.Printf("Final error: %v", err)
	} else {
		fmt.Printf("Operation succeeded with metrics middleware!\n")
	}
}

func circuitBreakerExample() {
	fmt.Println("--- Circuit Breaker Middleware Example ---")

	ctx := context.Background()

	// Create circuit breaker with threshold of 2 failures and 1 second timeout
	cb := NewSimpleCircuitBreaker(2, 1*time.Second)

	// Create retrier with circuit breaker middleware
	retrier := failsafe.NewEnhancedRetrier(
		failsafe.WithMaxAttempts(5),
		failsafe.WithDelayStrategy(&failsafe.FixedDelay{Delay: 100 * time.Millisecond}),
	)

	cbMiddleware := middleware.NewCircuitBreakerMiddleware(cb)
	retrier.AddMiddleware(cbMiddleware)

	// Simulate multiple failing operations to trigger circuit breaker
	for i := 0; i < 5; i++ {
		fmt.Printf("Operation %d - Circuit breaker state: %s\n", i+1, cb.State())

		err := retrier.Retry(ctx, func() error {
			return errors.New("operation failed")
		})

		if err != nil {
			fmt.Printf("Operation %d failed: %v\n", i+1, err)
		}

		time.Sleep(100 * time.Millisecond)
	}

	// Wait for circuit breaker to enter half-open state
	fmt.Println("Waiting for circuit breaker to enter half-open state...")
	time.Sleep(1200 * time.Millisecond)

	// Try a successful operation to close the circuit
	fmt.Printf("Attempting recovery - Circuit breaker state: %s\n", cb.State())
	err := retrier.Retry(ctx, func() error {
		fmt.Println("Operation succeeded - circuit breaker should close")
		return nil
	})

	if err != nil {
		fmt.Printf("Recovery failed: %v\n", err)
	} else {
		fmt.Printf("Recovery succeeded - Circuit breaker state: %s\n", cb.State())
	}
}

func rateLimitExample() {
	fmt.Println("--- Rate Limit Middleware Example ---")

	ctx := context.Background()

	// Create rate limiter with capacity of 3 tokens, refilling every 500ms
	rl := NewSimpleRateLimiter(3, 500*time.Millisecond)

	// Create retrier with rate limit middleware
	retrier := failsafe.NewEnhancedRetrier(
		failsafe.WithMaxAttempts(2),
		failsafe.WithDelayStrategy(&failsafe.FixedDelay{Delay: 100 * time.Millisecond}),
	)

	rlMiddleware := middleware.NewRateLimitMiddleware(rl)
	retrier.AddMiddleware(rlMiddleware)

	// Simulate multiple operations that exceed rate limit
	for i := 0; i < 6; i++ {
		start := time.Now()
		fmt.Printf("Operation %d starting at %v\n", i+1, start.Format("15:04:05.000"))

		err := retrier.Retry(ctx, func() error {
			fmt.Printf("Operation %d executing\n", i+1)
			return nil
		})

		elapsed := time.Since(start)
		if err != nil {
			fmt.Printf("Operation %d failed after %v: %v\n", i+1, elapsed, err)
		} else {
			fmt.Printf("Operation %d succeeded after %v\n", i+1, elapsed)
		}
	}
}

func combinedExample() {
	fmt.Println("--- Combined Middleware Example ---")

	ctx := context.Background()

	// Create retrier with multiple middleware
	retrier := failsafe.NewEnhancedRetrier(
		failsafe.WithMaxAttempts(3),
		failsafe.WithDelayStrategy(&failsafe.FixedDelay{Delay: 200 * time.Millisecond}),
	)

	// Add metrics middleware
	metricsMiddleware := middleware.NewMetricsMiddleware(
		func(attempt int) {
			fmt.Printf("Metrics: Attempt %d\n", attempt)
		},
		func(totalAttempts int) {
			fmt.Printf("Metrics: Success after %d attempts\n", totalAttempts)
		},
		func(totalAttempts int, err error) {
			fmt.Printf("Metrics: Failed after %d attempts\n", totalAttempts)
		},
	)

	// Add circuit breaker middleware
	cb := NewSimpleCircuitBreaker(2, 1*time.Second)
	cbMiddleware := middleware.NewCircuitBreakerMiddleware(cb)

	// Add rate limiter middleware
	rl := NewSimpleRateLimiter(2, 300*time.Millisecond)
	rlMiddleware := middleware.NewRateLimitMiddleware(rl)

	// Add all middleware (order matters - rate limit first, then circuit breaker, then metrics)
	retrier.AddMiddleware(rlMiddleware)
	retrier.AddMiddleware(cbMiddleware)
	retrier.AddMiddleware(metricsMiddleware)

	// Simulate operations with combined middleware
	attempts := 0
	err := retrier.Retry(ctx, func() error {
		attempts++
		fmt.Printf("Executing combined operation attempt %d\n", attempts)

		if attempts < 2 {
			return errors.New("simulated failure")
		}
		return nil
	})

	if err != nil {
		log.Printf("Combined operation failed: %v", err)
	} else {
		fmt.Printf("Combined operation succeeded with all middleware!\n")
	}
}
