package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/bpradana/failsafe"
)

func main() {
	// Basic retry example
	fmt.Println("=== Basic Retry Example ===")

	ctx := context.Background()

	// Create a simple retrier
	retrier := failsafe.NewRetrier(
		failsafe.WithMaxAttempts(3),
		failsafe.WithDelayStrategy(&failsafe.FixedDelay{Delay: 1 * time.Second}),
	)

	// Simulate a function that fails twice then succeeds
	attempts := 0
	err := retrier.Retry(ctx, func() error {
		attempts++
		fmt.Printf("Attempt %d\n", attempts)

		if attempts < 3 {
			return errors.New("simulated failure")
		}
		return nil
	})

	if err != nil {
		log.Printf("Final error: %v", err)
	} else {
		fmt.Printf("Success after %d attempts!\n", attempts)
	}

	// Retry with result example
	fmt.Println("\n=== Retry with Result Example ===")

	attempts = 0
	result, err := failsafe.RetryWithResult(ctx, retrier, func() (string, error) {
		attempts++
		fmt.Printf("Attempt %d\n", attempts)

		if attempts < 2 {
			return "", errors.New("simulated failure")
		}
		return "success!", nil
	})

	if err != nil {
		log.Printf("Final error: %v", err)
	} else {
		fmt.Printf("Result: %s (after %d attempts)\n", result, attempts)
	}

	// Async retry example
	fmt.Println("\n=== Async Retry Example ===")

	// Create async retrier
	asyncRetrier := failsafe.NewRetrier(
		failsafe.WithMaxAttempts(3),
		failsafe.WithAsyncMode(true),
		failsafe.WithDelayStrategy(&failsafe.FixedDelay{Delay: 500 * time.Millisecond}),
		failsafe.WithOnRetry(func(ctx context.Context, attempt int, err error, nextDelay time.Duration) {
			fmt.Printf("Async retry attempt %d failed: %v, next delay: %v\n", attempt, err, nextDelay)
		}),
		failsafe.WithOnSuccess(func(ctx context.Context, attempt int, err error, nextDelay time.Duration) {
			fmt.Printf("Async operation succeeded on attempt %d\n", attempt)
		}),
		failsafe.WithOnFinalError(func(ctx context.Context, attempt int, err error, nextDelay time.Duration) {
			fmt.Printf("Async operation failed after %d attempts: %v\n", attempt, err)
		}),
	)

	// Start async operation - returns immediately
	fmt.Println("Starting async operation...")
	attempts = 0
	err = asyncRetrier.Retry(ctx, func() error {
		attempts++
		fmt.Printf("Async attempt %d\n", attempts)

		if attempts < 3 {
			return errors.New("async simulated failure")
		}
		return nil
	})

	// err will be nil in async mode
	fmt.Printf("Async retry returned: %v (operation running in background)\n", err)

	// Wait a bit to see the async operation complete
	fmt.Println("Waiting for async operation to complete...")
	time.Sleep(3 * time.Second)
	fmt.Println("Done!")
}
