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
}
