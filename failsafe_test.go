package failsafe

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRetrier_Retry_Success(t *testing.T) {
	retrier := NewRetrier(WithMaxAttempts(3))
	ctx := context.Background()

	attempts := 0
	err := retrier.Retry(ctx, func() error {
		attempts++
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", attempts)
	}
}

func TestRetrier_Retry_EventualSuccess(t *testing.T) {
	retrier := NewRetrier(WithMaxAttempts(3))
	ctx := context.Background()

	attempts := 0
	err := retrier.Retry(ctx, func() error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary failure")
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestRetrier_Retry_MaxAttemptsExceeded(t *testing.T) {
	retrier := NewRetrier(WithMaxAttempts(2))
	ctx := context.Background()

	attempts := 0
	err := retrier.Retry(ctx, func() error {
		attempts++
		return errors.New("persistent failure")
	})

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
	if !strings.Contains(err.Error(), "persistent failure") {
		t.Errorf("Expected error to contain 'persistent failure', got %v", err)
	}
}

func TestRetrier_Retry_ContextCancellation(t *testing.T) {
	retrier := NewRetrier(WithMaxAttempts(5))
	ctx, cancel := context.WithCancel(context.Background())

	attempts := 0
	err := retrier.Retry(ctx, func() error {
		attempts++
		if attempts == 2 {
			cancel()
		}
		return errors.New("failure")
	})

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context cancellation error, got %v", err)
	}
}

func TestRetrier_Retry_ContextTimeout(t *testing.T) {
	retrier := NewRetrier(
		WithMaxAttempts(10),
		WithDelayStrategy(&FixedDelay{Delay: 100 * time.Millisecond}),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	attempts := 0
	err := retrier.Retry(ctx, func() error {
		attempts++
		return errors.New("failure")
	})

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected timeout error, got %v", err)
	}
}

func TestRetrier_Retry_NonRetryableError(t *testing.T) {
	retrier := NewRetrier(
		WithMaxAttempts(3),
		WithErrorFilter(func(err error) bool {
			return !strings.Contains(err.Error(), "non-retryable")
		}),
	)
	ctx := context.Background()

	attempts := 0
	nonRetryableErr := errors.New("non-retryable")
	err := retrier.Retry(ctx, func() error {
		attempts++
		return nonRetryableErr
	})

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", attempts)
	}
}

func TestRetrier_Hooks(t *testing.T) {
	var onRetryCallCount int
	var onSuccessCallCount int
	var onFinalErrorCallCount int

	retrier := NewRetrier(
		WithMaxAttempts(3),
		WithOnRetry(func(ctx context.Context, attempt int, err error, nextDelay time.Duration) {
			onRetryCallCount++
		}),
		WithOnSuccess(func(ctx context.Context, attempt int, err error, nextDelay time.Duration) {
			onSuccessCallCount++
		}),
		WithOnFinalError(func(ctx context.Context, attempt int, err error, nextDelay time.Duration) {
			onFinalErrorCallCount++
		}),
	)

	ctx := context.Background()

	// Test success hooks
	attempts := 0
	err := retrier.Retry(ctx, func() error {
		attempts++
		if attempts < 2 {
			return errors.New("temporary failure")
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if onRetryCallCount != 1 {
		t.Errorf("Expected 1 retry hook call, got %d", onRetryCallCount)
	}
	if onSuccessCallCount != 1 {
		t.Errorf("Expected 1 success hook call, got %d", onSuccessCallCount)
	}
	if onFinalErrorCallCount != 0 {
		t.Errorf("Expected 0 final error hook calls, got %d", onFinalErrorCallCount)
	}

	// Reset counters and test failure hooks
	onRetryCallCount = 0
	onSuccessCallCount = 0
	onFinalErrorCallCount = 0

	err = retrier.Retry(ctx, func() error {
		return errors.New("persistent failure")
	})

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if onRetryCallCount != 2 {
		t.Errorf("Expected 2 retry hook calls, got %d", onRetryCallCount)
	}
	if onSuccessCallCount != 0 {
		t.Errorf("Expected 0 success hook calls, got %d", onSuccessCallCount)
	}
	if onFinalErrorCallCount != 1 {
		t.Errorf("Expected 1 final error hook call, got %d", onFinalErrorCallCount)
	}
}

func TestRetrier_UpdateConfig(t *testing.T) {
	retrier := NewRetrier(WithMaxAttempts(2))

	// Initial config
	config := retrier.GetConfig()
	if config.MaxAttempts != 2 {
		t.Errorf("Expected MaxAttempts 2, got %d", config.MaxAttempts)
	}

	// Update config
	retrier.UpdateConfig(WithMaxAttempts(5))
	config = retrier.GetConfig()
	if config.MaxAttempts != 5 {
		t.Errorf("Expected MaxAttempts 5, got %d", config.MaxAttempts)
	}
}

func TestFixedDelay(t *testing.T) {
	delay := &FixedDelay{Delay: 100 * time.Millisecond}

	// Test NextDelay returns the same delay regardless of attempt
	if nextDelay := delay.NextDelay(1, 0); nextDelay != 100*time.Millisecond {
		t.Errorf("Expected 100ms, got %v", nextDelay)
	}
	if nextDelay := delay.NextDelay(5, 50*time.Millisecond); nextDelay != 100*time.Millisecond {
		t.Errorf("Expected 100ms, got %v", nextDelay)
	}

	// Test Reset does nothing
	delay.Reset()
	if nextDelay := delay.NextDelay(1, 0); nextDelay != 100*time.Millisecond {
		t.Errorf("Expected 100ms after reset, got %v", nextDelay)
	}
}

func TestRetryAllErrors(t *testing.T) {
	testCases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, true},
		{"generic error", errors.New("test error"), true},
		{"context canceled", context.Canceled, true},
		{"context timeout", context.DeadlineExceeded, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := RetryAllErrors(tc.err); got != tc.want {
				t.Errorf("RetryAllErrors(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestRetryTransientErrors(t *testing.T) {
	testCases := []struct {
		name string
		err  error
		want bool
	}{
		{"generic error", errors.New("test error"), true},
		{"context canceled", context.Canceled, false},
		{"context timeout", context.DeadlineExceeded, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := RetryTransientErrors(tc.err); got != tc.want {
				t.Errorf("RetryTransientErrors(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestNewRetrier_DefaultConfig(t *testing.T) {
	retrier := NewRetrier()
	config := retrier.GetConfig()

	if config.MaxAttempts != DefaultMaxAttempts {
		t.Errorf("Expected MaxAttempts %d, got %d", DefaultMaxAttempts, config.MaxAttempts)
	}
	if config.DelayStrategy == nil {
		t.Error("Expected DelayStrategy to be set")
	}
	if config.ErrorFilter == nil {
		t.Error("Expected ErrorFilter to be set")
	}
}

func TestConcurrentRetries(t *testing.T) {
	retrier := NewRetrier(WithMaxAttempts(3))
	ctx := context.Background()

	const numGoroutines = 10
	errChan := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			err := retrier.Retry(ctx, func() error {
				return nil
			})
			errChan <- err
		}()
	}

	for i := 0; i < numGoroutines; i++ {
		if err := <-errChan; err != nil {
			t.Errorf("Concurrent retry failed: %v", err)
		}
	}
}

// Async Mode Tests

func TestRetrier_AsyncMode_ReturnsImmediately(t *testing.T) {
	retrier := NewRetrier(
		WithMaxAttempts(3),
		WithAsyncMode(true),
		WithDelayStrategy(&FixedDelay{Delay: 100 * time.Millisecond}),
	)
	ctx := context.Background()

	start := time.Now()
	err := retrier.Retry(ctx, func() error {
		time.Sleep(200 * time.Millisecond) // Simulate work
		return nil
	})
	duration := time.Since(start)

	// Should return immediately (nil error in async mode)
	if err != nil {
		t.Errorf("Expected nil error in async mode, got %v", err)
	}

	// Should return much faster than the work duration
	if duration > 50*time.Millisecond {
		t.Errorf("Expected immediate return, took %v", duration)
	}
}

func TestRetrier_AsyncMode_Success(t *testing.T) {
	var successCalled bool
	var successAttempt int

	retrier := NewRetrier(
		WithMaxAttempts(3),
		WithAsyncMode(true),
		WithDelayStrategy(&FixedDelay{Delay: 10 * time.Millisecond}),
		WithOnSuccess(func(ctx context.Context, attempt int, err error, nextDelay time.Duration) {
			successCalled = true
			successAttempt = attempt
		}),
	)
	ctx := context.Background()

	attempts := 0
	err := retrier.Retry(ctx, func() error {
		attempts++
		if attempts < 2 {
			return errors.New("temporary failure")
		}
		return nil
	})

	// Should return nil immediately
	if err != nil {
		t.Errorf("Expected nil error in async mode, got %v", err)
	}

	// Wait for async operation to complete
	time.Sleep(100 * time.Millisecond)

	if !successCalled {
		t.Error("Expected success hook to be called")
	}
	if successAttempt != 2 {
		t.Errorf("Expected success on attempt 2, got %d", successAttempt)
	}
}

func TestRetrier_AsyncMode_Failure(t *testing.T) {
	var finalErrorCalled bool
	var finalErrorAttempt int

	retrier := NewRetrier(
		WithMaxAttempts(2),
		WithAsyncMode(true),
		WithDelayStrategy(&FixedDelay{Delay: 10 * time.Millisecond}),
		WithOnFinalError(func(ctx context.Context, attempt int, err error, nextDelay time.Duration) {
			finalErrorCalled = true
			finalErrorAttempt = attempt
		}),
	)
	ctx := context.Background()

	err := retrier.Retry(ctx, func() error {
		return errors.New("persistent failure")
	})

	// Should return nil immediately
	if err != nil {
		t.Errorf("Expected nil error in async mode, got %v", err)
	}

	// Wait for async operation to complete
	time.Sleep(100 * time.Millisecond)

	if !finalErrorCalled {
		t.Error("Expected final error hook to be called")
	}
	if finalErrorAttempt != 2 {
		t.Errorf("Expected final error on attempt 2, got %d", finalErrorAttempt)
	}
}

func TestRetrier_AsyncMode_ContextCancellation(t *testing.T) {
	var finalErrorCalled bool
	var finalErrorMessage string

	retrier := NewRetrier(
		WithMaxAttempts(5),
		WithAsyncMode(true),
		WithDelayStrategy(&FixedDelay{Delay: 50 * time.Millisecond}),
		WithOnFinalError(func(ctx context.Context, attempt int, err error, nextDelay time.Duration) {
			finalErrorCalled = true
			finalErrorMessage = err.Error()
		}),
	)

	ctx, cancel := context.WithCancel(context.Background())

	err := retrier.Retry(ctx, func() error {
		return errors.New("will be cancelled")
	})

	// Should return nil immediately
	if err != nil {
		t.Errorf("Expected nil error in async mode, got %v", err)
	}

	// Cancel context after a short delay
	time.Sleep(10 * time.Millisecond)
	cancel()

	// Wait for async operation to handle cancellation
	time.Sleep(100 * time.Millisecond)

	if !finalErrorCalled {
		t.Error("Expected final error hook to be called on cancellation")
	}
	if !strings.Contains(finalErrorMessage, "cancelled") {
		t.Errorf("Expected cancellation error message, got %s", finalErrorMessage)
	}
}

func TestRetrier_AsyncMode_WithRetryHooks(t *testing.T) {
	var retryCallCount int
	var retryAttempts []int

	retrier := NewRetrier(
		WithMaxAttempts(3),
		WithAsyncMode(true),
		WithDelayStrategy(&FixedDelay{Delay: 10 * time.Millisecond}),
		WithOnRetry(func(ctx context.Context, attempt int, err error, nextDelay time.Duration) {
			retryCallCount++
			retryAttempts = append(retryAttempts, attempt)
		}),
		WithOnSuccess(func(ctx context.Context, attempt int, err error, nextDelay time.Duration) {
			// Final success
		}),
	)
	ctx := context.Background()

	attempts := 0
	err := retrier.Retry(ctx, func() error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary failure")
		}
		return nil
	})

	// Should return nil immediately
	if err != nil {
		t.Errorf("Expected nil error in async mode, got %v", err)
	}

	// Wait for async operation to complete
	time.Sleep(100 * time.Millisecond)

	if retryCallCount != 2 {
		t.Errorf("Expected 2 retry hook calls, got %d", retryCallCount)
	}

	expectedAttempts := []int{1, 2}
	if len(retryAttempts) != len(expectedAttempts) {
		t.Errorf("Expected retry attempts %v, got %v", expectedAttempts, retryAttempts)
	}
	for i, expected := range expectedAttempts {
		if i < len(retryAttempts) && retryAttempts[i] != expected {
			t.Errorf("Expected retry attempt %d at index %d, got %d", expected, i, retryAttempts[i])
		}
	}
}

func TestRetrier_AsyncMode_NonRetryableError(t *testing.T) {
	var finalErrorCalled bool
	var attempts int

	retrier := NewRetrier(
		WithMaxAttempts(3),
		WithAsyncMode(true),
		WithDelayStrategy(&FixedDelay{Delay: 10 * time.Millisecond}),
		WithErrorFilter(func(err error) bool {
			return !strings.Contains(err.Error(), "non-retryable")
		}),
		WithOnFinalError(func(ctx context.Context, attempt int, err error, nextDelay time.Duration) {
			finalErrorCalled = true
		}),
	)
	ctx := context.Background()

	err := retrier.Retry(ctx, func() error {
		attempts++
		return errors.New("non-retryable error")
	})

	// Should return nil immediately
	if err != nil {
		t.Errorf("Expected nil error in async mode, got %v", err)
	}

	// Wait for async operation to complete
	time.Sleep(100 * time.Millisecond)

	if !finalErrorCalled {
		t.Error("Expected final error hook to be called for non-retryable error")
	}
	if attempts != 1 {
		t.Errorf("Expected 1 attempt for non-retryable error, got %d", attempts)
	}
}

func TestRetrier_SyncMode_Default(t *testing.T) {
	retrier := NewRetrier(
		WithMaxAttempts(3),
		// AsyncMode defaults to false
		WithDelayStrategy(&FixedDelay{Delay: 10 * time.Millisecond}),
	)
	ctx := context.Background()

	attempts := 0
	start := time.Now()
	err := retrier.Retry(ctx, func() error {
		attempts++
		if attempts < 2 {
			return errors.New("temporary failure")
		}
		return nil
	})
	duration := time.Since(start)

	// Should block until completion
	if err != nil {
		t.Errorf("Expected no error in sync mode, got %v", err)
	}
	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}

	// Should take at least the delay time
	if duration < 10*time.Millisecond {
		t.Errorf("Expected sync mode to take at least 10ms, took %v", duration)
	}
}

func TestRetrier_AsyncMode_MultipleOperations(t *testing.T) {
	var successCount int
	var mutex sync.Mutex

	retrier := NewRetrier(
		WithMaxAttempts(2),
		WithAsyncMode(true),
		WithDelayStrategy(&FixedDelay{Delay: 10 * time.Millisecond}),
		WithOnSuccess(func(ctx context.Context, attempt int, err error, nextDelay time.Duration) {
			mutex.Lock()
			successCount++
			mutex.Unlock()
		}),
	)
	ctx := context.Background()

	// Start multiple async operations
	const numOperations = 5
	for i := 0; i < numOperations; i++ {
		err := retrier.Retry(ctx, func() error {
			time.Sleep(5 * time.Millisecond) // Simulate work
			return nil
		})
		if err != nil {
			t.Errorf("Expected nil error for operation %d, got %v", i, err)
		}
	}

	// Wait for all operations to complete
	time.Sleep(200 * time.Millisecond)

	mutex.Lock()
	finalCount := successCount
	mutex.Unlock()

	if finalCount != numOperations {
		t.Errorf("Expected %d successful operations, got %d", numOperations, finalCount)
	}
}
