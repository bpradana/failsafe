package failsafe

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"failsafe/middleware"
	"failsafe/strategies"
)

// Integration tests that combine multiple components
func TestIntegration_CompleteRetryWorkflow(t *testing.T) {
	// Create a comprehensive retry configuration
	retrier := NewRetrier(
		WithMaxAttempts(5),
		WithDelayStrategy(strategies.NewExponentialBackoff(10*time.Millisecond, 100*time.Millisecond, 2.0)),
		WithErrorFilter(RetryTransientErrors),
		WithOnRetry(func(attempt int, err error, nextDelay time.Duration) {
			// Log retry attempt
		}),
		WithOnSuccess(func(attempt int, err error, nextDelay time.Duration) {
			// Log success
		}),
		WithOnFinalError(func(attempt int, err error, nextDelay time.Duration) {
			// Log final failure
		}),
	)

	ctx := context.Background()

	// Test successful retry after failures
	attempts := 0
	err := retrier.Retry(ctx, func() error {
		attempts++
		if attempts < 3 {
			return errors.New("transient error")
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected success after retries, got %v", err)
	}
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestIntegration_EnhancedRetrierWithMiddleware(t *testing.T) {
	// Create enhanced retrier with multiple middleware
	retrier := NewEnhancedRetrier(
		WithMaxAttempts(3),
		WithDelayStrategy(strategies.NewFixedDelay(5*time.Millisecond)),
	)

	// Add metrics middleware
	var totalAttempts int
	var successCount int
	var failureCount int

	metricsMiddleware := middleware.NewMetricsMiddleware(
		func(attempt int) {
			totalAttempts++
		},
		func(attempts int) {
			successCount++
		},
		func(attempts int, err error) {
			failureCount++
		},
	)

	retrier.AddMiddleware(metricsMiddleware)

	ctx := context.Background()

	// Test successful operation
	attempts := 0
	err := retrier.Retry(ctx, func() error {
		attempts++
		if attempts < 2 {
			return errors.New("temporary failure")
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected success, got %v", err)
	}
	if totalAttempts != 2 {
		t.Errorf("Expected 2 total attempts recorded, got %d", totalAttempts)
	}
	if successCount != 1 {
		t.Errorf("Expected 1 success recorded, got %d", successCount)
	}
	if failureCount != 0 {
		t.Errorf("Expected 0 failures recorded, got %d", failureCount)
	}
}

func TestIntegration_GenericRetryWithComplexTypes(t *testing.T) {
	retrier := NewRetrier(WithMaxAttempts(3))
	ctx := context.Background()

	// Test with complex struct
	type ComplexResult struct {
		Data   map[string]interface{}
		Status int
		Items  []string
	}

	attempts := 0
	result, err := RetryWithResult(ctx, retrier, func() (ComplexResult, error) {
		attempts++
		if attempts < 2 {
			return ComplexResult{}, errors.New("temporary failure")
		}
		return ComplexResult{
			Data:   map[string]interface{}{"key": "value"},
			Status: 200,
			Items:  []string{"item1", "item2"},
		}, nil
	})

	if err != nil {
		t.Errorf("Expected success, got %v", err)
	}
	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
	if result.Status != 200 {
		t.Errorf("Expected status 200, got %d", result.Status)
	}
	if len(result.Items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(result.Items))
	}
}

func TestIntegration_ContextCancellationWithMiddleware(t *testing.T) {
	retrier := NewEnhancedRetrier(
		WithMaxAttempts(10),
		WithDelayStrategy(strategies.NewFixedDelay(50*time.Millisecond)),
	)

	// Add middleware that tracks cancellation
	var cancelledInMiddleware bool
	cancellationMiddleware := &contextCancellationMiddleware{
		cancelled: &cancelledInMiddleware,
	}
	retrier.AddMiddleware(cancellationMiddleware)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	attempts := 0
	err := retrier.Retry(ctx, func() error {
		attempts++
		return errors.New("always fail")
	})

	if err == nil {
		t.Error("Expected timeout error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected timeout error, got %v", err)
	}
	if !cancelledInMiddleware {
		t.Error("Expected middleware to detect context cancellation")
	}
}

func TestIntegration_ConcurrentRetriesWithSharedResources(t *testing.T) {
	retrier := NewRetrier(
		WithMaxAttempts(3),
		WithDelayStrategy(strategies.NewFixedDelay(1*time.Millisecond)),
	)

	ctx := context.Background()

	// Shared resource with controlled failure
	var sharedResource struct {
		mu        sync.Mutex
		callCount int
		failUntil int
	}
	sharedResource.failUntil = 50 // Fail first 50 calls

	const numGoroutines = 10
	const retriesPerGoroutine = 10

	var wg sync.WaitGroup
	errChan := make(chan error, numGoroutines*retriesPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for j := 0; j < retriesPerGoroutine; j++ {
				err := retrier.Retry(ctx, func() error {
					sharedResource.mu.Lock()
					defer sharedResource.mu.Unlock()

					sharedResource.callCount++
					if sharedResource.callCount <= sharedResource.failUntil {
						return errors.New("resource not ready")
					}
					return nil
				})

				errChan <- err
			}
		}()
	}

	wg.Wait()
	close(errChan)

	// Check results
	successCount := 0
	failureCount := 0

	for err := range errChan {
		if err == nil {
			successCount++
		} else {
			failureCount++
		}
	}

	totalOperations := numGoroutines * retriesPerGoroutine
	if successCount+failureCount != totalOperations {
		t.Errorf("Expected %d total operations, got %d", totalOperations, successCount+failureCount)
	}

	// Most operations should succeed once resource becomes available
	if successCount < totalOperations/2 {
		t.Errorf("Expected at least %d successes, got %d", totalOperations/2, successCount)
	}
}

func TestIntegration_ConvenienceFunctionWithRealScenario(t *testing.T) {
	ctx := context.Background()

	// Simulate a service that becomes available after a few attempts
	var serviceCallCount int

	err := RetryWithExponentialBackoff(ctx, func() error {
		serviceCallCount++

		// Simulate service startup time
		if serviceCallCount < 3 {
			return errors.New("service not ready")
		}

		// Simulate occasional transient failures
		if serviceCallCount == 3 {
			return errors.New("temporary network error")
		}

		return nil
	}, 5)

	if err != nil {
		t.Errorf("Expected success, got %v", err)
	}
	if serviceCallCount != 4 {
		t.Errorf("Expected 4 service calls, got %d", serviceCallCount)
	}
}

// Custom error types for testing
type TransientError struct {
	Message string
}

func (e TransientError) Error() string {
	return e.Message
}

type PermanentError struct {
	Message string
}

func (e PermanentError) Error() string {
	return e.Message
}

func TestIntegration_ComplexErrorFiltering(t *testing.T) {

	// Custom error filter
	retrier := NewRetrier(
		WithMaxAttempts(5),
		WithErrorFilter(func(err error) bool {
			// Only retry transient errors
			var transientErr TransientError
			return errors.As(err, &transientErr)
		}),
	)

	ctx := context.Background()

	// Test with transient error followed by success
	attempts := 0
	err := retrier.Retry(ctx, func() error {
		attempts++
		if attempts < 3 {
			return TransientError{Message: "transient failure"}
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected success with transient errors, got %v", err)
	}
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}

	// Test with permanent error (should not retry)
	attempts = 0
	err = retrier.Retry(ctx, func() error {
		attempts++
		return PermanentError{Message: "permanent failure"}
	})

	if err == nil {
		t.Error("Expected permanent error")
	}
	if attempts != 1 {
		t.Errorf("Expected 1 attempt (no retry), got %d", attempts)
	}
}

func TestIntegration_DynamicConfigurationUpdate(t *testing.T) {
	retrier := NewRetrier(WithMaxAttempts(2))
	ctx := context.Background()

	// Initial retry should fail after 2 attempts
	attempts := 0
	err := retrier.Retry(ctx, func() error {
		attempts++
		return errors.New("persistent failure")
	})

	if err == nil {
		t.Error("Expected failure with 2 attempts")
	}
	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}

	// Update configuration to allow more attempts
	retrier.UpdateConfig(WithMaxAttempts(5))

	// Now retry should succeed with more attempts
	attempts = 0
	err = retrier.Retry(ctx, func() error {
		attempts++
		if attempts < 4 {
			return errors.New("temporary failure")
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected success with updated config, got %v", err)
	}
	if attempts != 4 {
		t.Errorf("Expected 4 attempts, got %d", attempts)
	}
}

// Helper middleware for testing context cancellation
type contextCancellationMiddleware struct {
	cancelled *bool
}

func (m *contextCancellationMiddleware) Execute(ctx context.Context, fn func() error, next func(context.Context, func() error) error) error {
	err := next(ctx, fn)
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		*m.cancelled = true
	}
	return err
}

func TestIntegration_CompleteFailsafeWorkflow(t *testing.T) {
	// This test demonstrates a complete real-world scenario
	// combining all major features of the failsafe package

	// Create enhanced retrier with comprehensive configuration
	retrier := NewEnhancedRetrier(
		WithMaxAttempts(5),
		WithDelayStrategy(strategies.ExponentialBackoffWithJitter(
			10*time.Millisecond,
			100*time.Millisecond,
			2.0,
		)),
		WithErrorFilter(RetryTransientErrors),
		WithOnRetry(func(attempt int, err error, nextDelay time.Duration) {
			// In a real scenario, this would log the retry attempt
		}),
		WithOnSuccess(func(attempt int, err error, nextDelay time.Duration) {
			// In a real scenario, this would log the success
		}),
		WithOnFinalError(func(attempt int, err error, nextDelay time.Duration) {
			// In a real scenario, this would log the final failure
		}),
	)

	// Add metrics middleware
	var metricsData struct {
		totalAttempts int
		successes     int
		failures      int
	}

	metricsMiddleware := middleware.NewMetricsMiddleware(
		func(attempt int) {
			metricsData.totalAttempts++
		},
		func(totalAttempts int) {
			metricsData.successes++
		},
		func(totalAttempts int, err error) {
			metricsData.failures++
		},
	)

	retrier.AddMiddleware(metricsMiddleware)

	ctx := context.Background()

	// Simulate a complex operation that might fail
	type ServiceResponse struct {
		ID      string
		Status  string
		Results []interface{}
	}

	attempts := 0
	var result ServiceResponse
	err := retrier.Retry(ctx, func() error {
		attempts++

		// Simulate various failure scenarios
		switch attempts {
		case 1:
			return errors.New("network timeout")
		case 2:
			return errors.New("service unavailable")
		case 3:
			// Success case
			result = ServiceResponse{
				ID:      "response-123",
				Status:  "success",
				Results: []interface{}{"data1", "data2"},
			}
			return nil
		default:
			return errors.New("unexpected error")
		}
	})

	// Verify the operation succeeded
	if err != nil {
		t.Errorf("Expected success, got %v", err)
	}

	if result.ID != "response-123" {
		t.Errorf("Expected ID 'response-123', got %s", result.ID)
	}

	if result.Status != "success" {
		t.Errorf("Expected status 'success', got %s", result.Status)
	}

	if len(result.Results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(result.Results))
	}

	// Verify metrics were collected
	if metricsData.totalAttempts != 3 {
		t.Errorf("Expected 3 total attempts in metrics, got %d", metricsData.totalAttempts)
	}

	if metricsData.successes != 1 {
		t.Errorf("Expected 1 success in metrics, got %d", metricsData.successes)
	}

	if metricsData.failures != 0 {
		t.Errorf("Expected 0 failures in metrics, got %d", metricsData.failures)
	}

	// Verify the actual retry attempts
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}
