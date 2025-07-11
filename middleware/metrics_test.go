package middleware

import (
	"context"
	"errors"
	"testing"
)

func TestMetricsMiddleware_Success(t *testing.T) {
	var attemptCount int
	var successCount int
	var failureCount int
	var finalAttempts int

	middleware := NewMetricsMiddleware(
		func(attempt int) {
			attemptCount++
		},
		func(totalAttempts int) {
			successCount++
			finalAttempts = totalAttempts
		},
		func(totalAttempts int, err error) {
			failureCount++
			finalAttempts = totalAttempts
		},
	)

	ctx := context.Background()

	callCount := 0
	fn := func() error {
		callCount++
		return nil
	}

	next := func(ctx context.Context, fn func() error) error {
		return fn()
	}

	err := middleware.Execute(ctx, fn, next)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if callCount != 1 {
		t.Errorf("Expected function to be called once, got %d", callCount)
	}
	if attemptCount != 1 {
		t.Errorf("Expected 1 attempt recorded, got %d", attemptCount)
	}
	if successCount != 1 {
		t.Errorf("Expected 1 success recorded, got %d", successCount)
	}
	if failureCount != 0 {
		t.Errorf("Expected 0 failures recorded, got %d", failureCount)
	}
	if finalAttempts != 1 {
		t.Errorf("Expected final attempts to be 1, got %d", finalAttempts)
	}
}

func TestMetricsMiddleware_Failure(t *testing.T) {
	var attemptCount int
	var successCount int
	var failureCount int
	var finalAttempts int
	var finalErr error

	middleware := NewMetricsMiddleware(
		func(attempt int) {
			attemptCount++
		},
		func(totalAttempts int) {
			successCount++
			finalAttempts = totalAttempts
		},
		func(totalAttempts int, err error) {
			failureCount++
			finalAttempts = totalAttempts
			finalErr = err
		},
	)

	ctx := context.Background()

	testError := errors.New("test error")
	fn := func() error {
		return testError
	}

	next := func(ctx context.Context, fn func() error) error {
		return fn()
	}

	err := middleware.Execute(ctx, fn, next)

	if err != testError {
		t.Errorf("Expected test error, got %v", err)
	}
	if attemptCount != 1 {
		t.Errorf("Expected 1 attempt recorded, got %d", attemptCount)
	}
	if successCount != 0 {
		t.Errorf("Expected 0 successes recorded, got %d", successCount)
	}
	if failureCount != 1 {
		t.Errorf("Expected 1 failure recorded, got %d", failureCount)
	}
	if finalAttempts != 1 {
		t.Errorf("Expected final attempts to be 1, got %d", finalAttempts)
	}
	if finalErr != testError {
		t.Errorf("Expected final error to be test error, got %v", finalErr)
	}
}

func TestMetricsMiddleware_MultipleAttempts(t *testing.T) {
	var attemptCalls []int
	var successCount int
	var failureCount int
	var finalAttempts int

	middleware := NewMetricsMiddleware(
		func(attempt int) {
			attemptCalls = append(attemptCalls, attempt)
		},
		func(totalAttempts int) {
			successCount++
			finalAttempts = totalAttempts
		},
		func(totalAttempts int, err error) {
			failureCount++
			finalAttempts = totalAttempts
		},
	)

	ctx := context.Background()

	callCount := 0
	fn := func() error {
		callCount++
		if callCount < 3 {
			return errors.New("temporary error")
		}
		return nil
	}

	next := func(ctx context.Context, fn func() error) error {
		// Simulate multiple attempts
		for i := 0; i < 3; i++ {
			if err := fn(); err != nil && i == 2 {
				return err
			} else if err == nil {
				return nil
			}
		}
		return errors.New("max attempts reached")
	}

	err := middleware.Execute(ctx, fn, next)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(attemptCalls) != 3 {
		t.Errorf("Expected 3 attempt calls, got %d", len(attemptCalls))
	}
	if successCount != 1 {
		t.Errorf("Expected 1 success, got %d", successCount)
	}
	if failureCount != 0 {
		t.Errorf("Expected 0 failures, got %d", failureCount)
	}
	if finalAttempts != 3 {
		t.Errorf("Expected final attempts to be 3, got %d", finalAttempts)
	}

	// Check attempt sequence
	expectedSequence := []int{1, 2, 3}
	for i, expected := range expectedSequence {
		if i >= len(attemptCalls) || attemptCalls[i] != expected {
			t.Errorf("Expected attempt sequence %v, got %v", expectedSequence, attemptCalls)
			break
		}
	}
}

func TestMetricsMiddleware_NilCallbacks(t *testing.T) {
	// Test with nil callbacks to ensure no panics
	middleware := NewMetricsMiddleware(nil, nil, nil)

	ctx := context.Background()

	fn := func() error {
		return nil
	}

	next := func(ctx context.Context, fn func() error) error {
		return fn()
	}

	err := middleware.Execute(ctx, fn, next)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestMetricsMiddleware_PartialCallbacks(t *testing.T) {
	var attemptCount int

	// Test with only onAttempt callback
	middleware := NewMetricsMiddleware(
		func(attempt int) {
			attemptCount++
		},
		nil,
		nil,
	)

	ctx := context.Background()

	fn := func() error {
		return nil
	}

	next := func(ctx context.Context, fn func() error) error {
		return fn()
	}

	err := middleware.Execute(ctx, fn, next)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if attemptCount != 1 {
		t.Errorf("Expected 1 attempt recorded, got %d", attemptCount)
	}
}

func TestMetricsMiddleware_NextError(t *testing.T) {
	var attemptCount int
	var successCount int
	var failureCount int

	middleware := NewMetricsMiddleware(
		func(attempt int) {
			attemptCount++
		},
		func(totalAttempts int) {
			successCount++
		},
		func(totalAttempts int, err error) {
			failureCount++
		},
	)

	ctx := context.Background()

	fn := func() error {
		return nil
	}

	nextError := errors.New("next error")
	next := func(ctx context.Context, fn func() error) error {
		return nextError
	}

	err := middleware.Execute(ctx, fn, next)

	if err != nextError {
		t.Errorf("Expected next error, got %v", err)
	}
	if attemptCount != 1 {
		t.Errorf("Expected 1 attempt recorded, got %d", attemptCount)
	}
	if successCount != 0 {
		t.Errorf("Expected 0 successes recorded, got %d", successCount)
	}
	if failureCount != 1 {
		t.Errorf("Expected 1 failure recorded, got %d", failureCount)
	}
}

func TestNewMetricsMiddleware(t *testing.T) {
	onAttempt := func(int) {}
	onSuccess := func(int) {}
	onFailure := func(int, error) {}

	middleware := NewMetricsMiddleware(onAttempt, onSuccess, onFailure)

	if middleware.OnAttempt == nil {
		t.Error("OnAttempt callback not set")
	}
	if middleware.OnSuccess == nil {
		t.Error("OnSuccess callback not set")
	}
	if middleware.OnFailure == nil {
		t.Error("OnFailure callback not set")
	}
}
