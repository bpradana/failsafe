package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/bpradana/failsafe"
	"github.com/bpradana/failsafe/strategies"
)

func main() {
	// Advanced retry with exponential backoff and jitter
	fmt.Println("=== Advanced Retry Example ===")

	ctx := context.Background()

	// Create exponential backoff with jitter
	backoff := strategies.ExponentialBackoffWithJitter(
		100*time.Millisecond, // initial delay
		5*time.Second,        // max delay
		2.0,                  // multiplier
	)

	// Create retrier with advanced configuration
	retrier := failsafe.NewRetrier(
		failsafe.WithMaxAttempts(5),
		failsafe.WithDelayStrategy(backoff),
		failsafe.WithErrorFilter(failsafe.RetryTransientErrors),
		failsafe.WithOnRetry(func(attempt int, err error, nextDelay time.Duration) {
			fmt.Printf("Retry attempt %d failed: %v, next delay: %v\n", attempt, err, nextDelay)
		}),
		failsafe.WithOnFinalError(func(attempt int, err error, nextDelay time.Duration) {
			fmt.Printf("Final retry attempt %d failed: %v\n", attempt, err)
		}),
		failsafe.WithOnSuccess(func(attempt int, err error, nextDelay time.Duration) {
			fmt.Printf("Success on attempt %d\n", attempt)
		}),
	)

	// Simulate a function that fails a few times then succeeds
	attempts := 0
	err := retrier.Retry(ctx, func() error {
		attempts++
		fmt.Printf("Executing attempt %d\n", attempts)

		if attempts < 3 {
			return errors.New("transient error")
		}
		return nil
	})

	if err != nil {
		log.Printf("Final error: %v", err)
	} else {
		fmt.Printf("Operation succeeded!\n")
	}

	// Convenience function example
	fmt.Println("\n=== Convenience Function Example ===")

	attempts = 0
	err = failsafe.RetryWithExponentialBackoff(ctx, func() error {
		attempts++
		fmt.Printf("Convenience retry attempt %d\n", attempts)

		if attempts < 2 {
			return errors.New("simulated failure")
		}
		return nil
	}, 3)

	if err != nil {
		log.Printf("Convenience retry failed: %v", err)
	} else {
		fmt.Printf("Convenience retry succeeded!\n")
	}

	// Advanced async retry example
	fmt.Println("\n=== Advanced Async Retry Example ===")

	// Create async retrier with exponential backoff
	asyncRetrier := failsafe.NewRetrier(
		failsafe.WithMaxAttempts(5),
		failsafe.WithAsyncMode(true),
		failsafe.WithDelayStrategy(strategies.ExponentialBackoffWithJitter(
			100*time.Millisecond,
			2*time.Second,
			1.5,
		)),
		failsafe.WithErrorFilter(failsafe.RetryTransientErrors),
		failsafe.WithOnRetry(func(attempt int, err error, nextDelay time.Duration) {
			fmt.Printf("Async retry attempt %d failed: %v, next delay: %v\n", attempt, err, nextDelay)
		}),
		failsafe.WithOnSuccess(func(attempt int, err error, nextDelay time.Duration) {
			fmt.Printf("Async operation succeeded on attempt %d\n", attempt)
		}),
		failsafe.WithOnFinalError(func(attempt int, err error, nextDelay time.Duration) {
			fmt.Printf("Async operation failed permanently after %d attempts: %v\n", attempt, err)
		}),
	)

	// Simulate background processing task
	fmt.Println("Starting async background task...")
	attempts = 0
	err = asyncRetrier.Retry(ctx, func() error {
		attempts++
		fmt.Printf("Background task attempt %d\n", attempts)

		// Simulate processing that might fail
		if attempts < 3 {
			return errors.New("background task failed")
		}

		// Simulate some work
		time.Sleep(100 * time.Millisecond)
		return nil
	})

	// Show that async mode returns immediately
	fmt.Printf("Async retry returned: %v (background task running)\n", err)

	// Demonstrate concurrent execution
	fmt.Println("Doing other work while background task runs...")
	for i := 1; i <= 3; i++ {
		fmt.Printf("Other work step %d\n", i)
		time.Sleep(500 * time.Millisecond)
	}

	// Wait for async task to complete
	fmt.Println("Waiting for background task to complete...")
	time.Sleep(3 * time.Second)
	fmt.Println("All done!")
}
