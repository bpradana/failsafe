package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/bpradana/failsafe"
	"github.com/bpradana/failsafe/middleware"
)

func main() {
	fmt.Println("=== Middleware Example ===")

	ctx := context.Background()

	// Create enhanced retrier with middleware support
	retrier := failsafe.NewEnhancedRetrier(
		failsafe.WithMaxAttempts(3),
		failsafe.WithDelayStrategy(&failsafe.FixedDelay{Delay: 500 * time.Millisecond}),
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
		fmt.Printf("Operation succeeded with middleware!\n")
	}
}
