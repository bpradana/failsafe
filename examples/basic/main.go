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
}
