package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/bpradana/failsafe"
	"github.com/bpradana/failsafe/middleware"
	"github.com/bpradana/failsafe/strategies"
)

func main() {
	fmt.Println("=== Async Mode Examples ===")

	ctx := context.Background()

	// Example 1: Basic async retry
	fmt.Println("\n--- Basic Async Retry ---")
	basicAsyncExample(ctx)

	// Example 2: Async with middleware
	fmt.Println("\n--- Async with Middleware ---")
	asyncMiddlewareExample(ctx)

	// Example 3: Multiple concurrent async operations
	fmt.Println("\n--- Multiple Concurrent Async Operations ---")
	multipleAsyncExample(ctx)

	// Example 4: Background task processing
	fmt.Println("\n--- Background Task Processing ---")
	backgroundTaskExample(ctx)

	fmt.Println("\n=== All Examples Complete ===")
}

func basicAsyncExample(ctx context.Context) {
	// Create async retrier
	retrier := failsafe.NewRetrier(
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

	// Start async operation
	fmt.Println("Starting async operation...")
	attempts := 0
	err := retrier.Retry(ctx, func() error {
		attempts++
		fmt.Printf("Async attempt %d\n", attempts)

		if attempts < 3 {
			return errors.New("simulated failure")
		}
		return nil
	})

	fmt.Printf("Async retry returned: %v (operation running in background)\n", err)

	// Wait for completion
	time.Sleep(2 * time.Second)
}

func asyncMiddlewareExample(ctx context.Context) {
	// Create enhanced retrier with async mode
	retrier := failsafe.NewEnhancedRetrier(
		failsafe.WithMaxAttempts(3),
		failsafe.WithAsyncMode(true),
		failsafe.WithDelayStrategy(&failsafe.FixedDelay{Delay: 300 * time.Millisecond}),
	)

	// Add metrics middleware
	metricsMiddleware := middleware.NewMetricsMiddleware(
		func(attempt int) {
			fmt.Printf("Async Metrics: Starting attempt %d\n", attempt)
		},
		func(totalAttempts int) {
			fmt.Printf("Async Metrics: Success after %d attempts\n", totalAttempts)
		},
		func(totalAttempts int, err error) {
			fmt.Printf("Async Metrics: Failed after %d attempts: %v\n", totalAttempts, err)
		},
	)

	retrier.AddMiddleware(metricsMiddleware)

	// Start async operation with middleware
	fmt.Println("Starting async operation with middleware...")
	attempts := 0
	err := retrier.Retry(ctx, func() error {
		attempts++
		fmt.Printf("Middleware async attempt %d\n", attempts)

		if attempts < 2 {
			return errors.New("middleware async failure")
		}
		return nil
	})

	fmt.Printf("Async middleware retry returned: %v\n", err)

	// Wait for completion
	time.Sleep(2 * time.Second)
}

func multipleAsyncExample(ctx context.Context) {
	var wg sync.WaitGroup

	// Create multiple async retriers
	for i := 1; i <= 3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			retrier := failsafe.NewRetrier(
				failsafe.WithMaxAttempts(3),
				failsafe.WithAsyncMode(true),
				failsafe.WithDelayStrategy(strategies.ExponentialBackoffWithJitter(
					100*time.Millisecond,
					1*time.Second,
					1.5,
				)),
				failsafe.WithOnSuccess(func(ctx context.Context, attempt int, err error, nextDelay time.Duration) {
					fmt.Printf("Task %d succeeded on attempt %d\n", id, attempt)
				}),
				failsafe.WithOnFinalError(func(ctx context.Context, attempt int, err error, nextDelay time.Duration) {
					fmt.Printf("Task %d failed after %d attempts: %v\n", id, attempt, err)
				}),
			)

			fmt.Printf("Starting concurrent task %d\n", id)
			attempts := 0
			retrier.Retry(ctx, func() error {
				attempts++
				fmt.Printf("Task %d attempt %d\n", id, attempts)

				// Different failure patterns for different tasks
				if id == 1 && attempts < 2 {
					return errors.New("task 1 failure")
				}
				if id == 2 && attempts < 3 {
					return errors.New("task 2 failure")
				}
				if id == 3 && attempts < 1 {
					return errors.New("task 3 failure")
				}
				return nil
			})
		}(i)
	}

	// Wait for all tasks to start
	time.Sleep(100 * time.Millisecond)
	fmt.Println("All concurrent tasks started...")

	// Wait for all tasks to complete
	wg.Wait()
	time.Sleep(3 * time.Second)
	fmt.Println("All concurrent tasks completed")
}

func backgroundTaskExample(ctx context.Context) {
	// Simulate a background task processor
	taskQueue := []string{"task1", "task2", "task3", "task4"}

	retrier := failsafe.NewRetrier(
		failsafe.WithMaxAttempts(5),
		failsafe.WithAsyncMode(true),
		failsafe.WithDelayStrategy(strategies.ExponentialBackoffWithJitter(
			200*time.Millisecond,
			2*time.Second,
			1.8,
		)),
		failsafe.WithOnRetry(func(ctx context.Context, attempt int, err error, nextDelay time.Duration) {
			fmt.Printf("Background task retry attempt %d: %v, next delay: %v\n", attempt, err, nextDelay)
		}),
		failsafe.WithOnSuccess(func(ctx context.Context, attempt int, err error, nextDelay time.Duration) {
			fmt.Printf("Background task completed successfully on attempt %d\n", attempt)
		}),
		failsafe.WithOnFinalError(func(ctx context.Context, attempt int, err error, nextDelay time.Duration) {
			fmt.Printf("Background task failed permanently after %d attempts: %v\n", attempt, err)
			// In a real application, you might want to:
			// - Log the failure
			// - Send an alert
			// - Move the task to a dead letter queue
		}),
	)

	// Process tasks in background
	for _, task := range taskQueue {
		fmt.Printf("Queuing background task: %s\n", task)

		// Capture task variable for closure
		taskName := task
		retrier.Retry(ctx, func() error {
			return processBackgroundTask(taskName)
		})
	}

	fmt.Println("All background tasks queued, continuing with other work...")

	// Simulate doing other work
	for i := 1; i <= 5; i++ {
		fmt.Printf("Doing other work step %d\n", i)
		time.Sleep(500 * time.Millisecond)
	}

	// Wait for background tasks to complete
	fmt.Println("Waiting for background tasks to complete...")
	time.Sleep(5 * time.Second)
}

func processBackgroundTask(taskName string) error {
	fmt.Printf("Processing background task: %s\n", taskName)

	// Simulate some work
	time.Sleep(100 * time.Millisecond)

	// Simulate different failure rates for different tasks
	switch taskName {
	case "task1":
		// This task always succeeds
		return nil
	case "task2":
		// This task fails once then succeeds
		if time.Now().UnixNano()%2 == 0 {
			return errors.New("task2 temporary failure")
		}
		return nil
	case "task3":
		// This task fails twice then succeeds
		if time.Now().UnixNano()%3 != 0 {
			return errors.New("task3 temporary failure")
		}
		return nil
	case "task4":
		// This task fails most of the time
		if time.Now().UnixNano()%5 != 0 {
			return errors.New("task4 frequent failure")
		}
		return nil
	default:
		return nil
	}
}
