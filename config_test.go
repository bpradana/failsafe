package failsafe

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestWithMaxAttempts(t *testing.T) {
	config := RetryConfig{}
	option := WithMaxAttempts(5)
	option(&config)

	if config.MaxAttempts != 5 {
		t.Errorf("Expected MaxAttempts 5, got %d", config.MaxAttempts)
	}
}

func TestWithDelayStrategy(t *testing.T) {
	config := RetryConfig{}
	strategy := &FixedDelay{Delay: 100 * time.Millisecond}
	option := WithDelayStrategy(strategy)
	option(&config)

	if config.DelayStrategy != strategy {
		t.Error("DelayStrategy not set correctly")
	}
}

func TestWithErrorFilter(t *testing.T) {
	config := RetryConfig{}
	filter := func(err error) bool { return false }
	option := WithErrorFilter(filter)
	option(&config)

	if config.ErrorFilter == nil {
		t.Error("ErrorFilter not set")
	}

	// Test the filter function
	if config.ErrorFilter(errors.New("test")) != false {
		t.Error("ErrorFilter not working correctly")
	}
}

func TestWithOnRetry(t *testing.T) {
	ctx := context.Background()
	config := RetryConfig{}
	called := false
	hook := func(ctx context.Context, attempt int, err error, nextDelay time.Duration) {
		called = true
	}
	option := WithOnRetry(hook)
	option(&config)

	if config.OnRetry == nil {
		t.Error("OnRetry hook not set")
	}

	// Test the hook function
	config.OnRetry(ctx, 1, errors.New("test"), time.Second)
	if !called {
		t.Error("OnRetry hook not called")
	}
}

func TestWithOnFinalError(t *testing.T) {
	ctx := context.Background()
	config := RetryConfig{}
	called := false
	hook := func(ctx context.Context, attempt int, err error, nextDelay time.Duration) {
		called = true
	}
	option := WithOnFinalError(hook)
	option(&config)

	if config.OnFinalError == nil {
		t.Error("OnFinalError hook not set")
	}

	// Test the hook function
	config.OnFinalError(ctx, 3, errors.New("test"), 0)
	if !called {
		t.Error("OnFinalError hook not called")
	}
}

func TestWithOnSuccess(t *testing.T) {
	ctx := context.Background()
	config := RetryConfig{}
	called := false
	hook := func(ctx context.Context, attempt int, err error, nextDelay time.Duration) {
		called = true
	}
	option := WithOnSuccess(hook)
	option(&config)

	if config.OnSuccess == nil {
		t.Error("OnSuccess hook not set")
	}

	// Test the hook function
	config.OnSuccess(ctx, 2, nil, 0)
	if !called {
		t.Error("OnSuccess hook not called")
	}
}

func TestRetryWithExponentialBackoff(t *testing.T) {
	ctx := context.Background()

	attempts := 0
	err := RetryWithExponentialBackoff(ctx, func() error {
		attempts++
		if attempts < 2 {
			return errors.New("temporary failure")
		}
		return nil
	}, 3)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
}

func TestRetryWithExponentialBackoff_Failure(t *testing.T) {
	ctx := context.Background()

	attempts := 0
	err := RetryWithExponentialBackoff(ctx, func() error {
		attempts++
		return errors.New("persistent failure")
	}, 2)

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
}

func TestRetryWithExponentialBackoff_NonRetryableError(t *testing.T) {
	ctx := context.Background()

	attempts := 0
	err := RetryWithExponentialBackoff(ctx, func() error {
		attempts++
		return context.Canceled
	}, 3)

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if attempts != 1 {
		t.Errorf("Expected 1 attempt (non-retryable), got %d", attempts)
	}
}

func TestRetryWithLogging(t *testing.T) {
	ctx := context.Background()

	var logMessages []string
	logger := func(format string, args ...interface{}) {
		logMessages = append(logMessages, format)
	}

	attempts := 0
	err := RetryWithLogging(ctx, func() error {
		attempts++
		if attempts < 2 {
			return errors.New("temporary failure")
		}
		return nil
	}, 3, logger)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}

	// Check that logging occurred
	if len(logMessages) == 0 {
		t.Error("Expected log messages, got none")
	}

	// Should have retry log and success log
	expectedMessages := []string{
		"Retry attempt %d failed: %v, next delay: %v",
		"Success on attempt %d",
	}

	for i, expected := range expectedMessages {
		if i >= len(logMessages) {
			t.Errorf("Expected log message %d: %s", i, expected)
		} else if logMessages[i] != expected {
			t.Errorf("Expected log message %d: %s, got %s", i, expected, logMessages[i])
		}
	}
}

func TestRetryWithLogging_Failure(t *testing.T) {
	ctx := context.Background()

	var logMessages []string
	logger := func(format string, args ...interface{}) {
		logMessages = append(logMessages, format)
	}

	attempts := 0
	err := RetryWithLogging(ctx, func() error {
		attempts++
		return errors.New("persistent failure")
	}, 2, logger)

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}

	// Check that logging occurred
	if len(logMessages) == 0 {
		t.Error("Expected log messages, got none")
	}

	// Should have retry log and final error log
	expectedMessages := []string{
		"Retry attempt %d failed: %v, next delay: %v",
		"Final retry attempt %d failed: %v",
	}

	for i, expected := range expectedMessages {
		if i >= len(logMessages) {
			t.Errorf("Expected log message %d: %s", i, expected)
		} else if logMessages[i] != expected {
			t.Errorf("Expected log message %d: %s, got %s", i, expected, logMessages[i])
		}
	}
}

func TestMultipleOptions(t *testing.T) {
	retrier := NewRetrier(
		WithMaxAttempts(5),
		WithDelayStrategy(&FixedDelay{Delay: 200 * time.Millisecond}),
		WithErrorFilter(func(err error) bool { return true }),
	)

	config := retrier.GetConfig()

	if config.MaxAttempts != 5 {
		t.Errorf("Expected MaxAttempts 5, got %d", config.MaxAttempts)
	}

	if config.DelayStrategy == nil {
		t.Error("DelayStrategy not set")
	}

	if config.ErrorFilter == nil {
		t.Error("ErrorFilter not set")
	}

	// Test that the delay strategy works
	delay := config.DelayStrategy.NextDelay(1, 0)
	if delay != 200*time.Millisecond {
		t.Errorf("Expected delay 200ms, got %v", delay)
	}
}

func TestOptionsOverride(t *testing.T) {
	retrier := NewRetrier(
		WithMaxAttempts(3),
		WithMaxAttempts(5), // Should override the previous value
	)

	config := retrier.GetConfig()
	if config.MaxAttempts != 5 {
		t.Errorf("Expected MaxAttempts 5 (overridden), got %d", config.MaxAttempts)
	}
}

func TestConfigurationImmutability(t *testing.T) {
	retrier := NewRetrier(WithMaxAttempts(3))

	// Get config and modify it
	config := retrier.GetConfig()
	config.MaxAttempts = 10

	// Original config should be unchanged
	originalConfig := retrier.GetConfig()
	if originalConfig.MaxAttempts != 3 {
		t.Errorf("Expected original MaxAttempts 3, got %d", originalConfig.MaxAttempts)
	}
}

func TestWithAsyncMode(t *testing.T) {
	config := RetryConfig{}

	// Test enabling async mode
	option := WithAsyncMode(true)
	option(&config)

	if !config.AsyncMode {
		t.Error("Expected AsyncMode to be true")
	}

	// Test disabling async mode
	option = WithAsyncMode(false)
	option(&config)

	if config.AsyncMode {
		t.Error("Expected AsyncMode to be false")
	}
}

func TestAsyncModeIntegration(t *testing.T) {
	// Test that async mode works with other configurations
	retrier := NewRetrier(
		WithMaxAttempts(3),
		WithAsyncMode(true),
		WithDelayStrategy(&FixedDelay{Delay: 10 * time.Millisecond}),
		WithErrorFilter(RetryTransientErrors),
	)

	config := retrier.GetConfig()

	if config.MaxAttempts != 3 {
		t.Errorf("Expected MaxAttempts 3, got %d", config.MaxAttempts)
	}

	if !config.AsyncMode {
		t.Error("Expected AsyncMode to be true")
	}

	if config.DelayStrategy == nil {
		t.Error("Expected DelayStrategy to be set")
	}

	if config.ErrorFilter == nil {
		t.Error("Expected ErrorFilter to be set")
	}
}

func TestAsyncModeWithConvenienceFunctions(t *testing.T) {
	ctx := context.Background()

	// Test that convenience functions work with async mode
	// Note: These functions create their own retriers, so they don't use async mode
	// This test ensures they still work correctly

	attempts := 0
	err := RetryWithExponentialBackoff(ctx, func() error {
		attempts++
		if attempts < 2 {
			return errors.New("temporary failure")
		}
		return nil
	}, 3)

	if err != nil {
		t.Errorf("Expected no error from convenience function, got %v", err)
	}

	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
}
