package middleware

import (
	"context"
	"errors"
	"testing"
)

// Mock circuit breaker for testing
type mockCircuitBreaker struct {
	allowCalls   []bool
	allowIndex   int
	successCount int
	failureCount int
	state        string
}

func (m *mockCircuitBreaker) Allow() bool {
	if m.allowIndex >= len(m.allowCalls) {
		return true
	}
	result := m.allowCalls[m.allowIndex]
	m.allowIndex++
	return result
}

func (m *mockCircuitBreaker) RecordSuccess() {
	m.successCount++
}

func (m *mockCircuitBreaker) RecordFailure() {
	m.failureCount++
}

func (m *mockCircuitBreaker) State() string {
	return m.state
}

func TestCircuitBreakerMiddleware_Allow(t *testing.T) {
	cb := &mockCircuitBreaker{
		allowCalls: []bool{true},
		state:      "closed",
	}

	middleware := NewCircuitBreakerMiddleware(cb)
	ctx := context.Background()

	called := false
	fn := func() error {
		called = true
		return nil
	}

	next := func(ctx context.Context, fn func() error) error {
		return fn()
	}

	err := middleware.Execute(ctx, fn, next)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !called {
		t.Error("Expected function to be called")
	}
	if cb.successCount != 1 {
		t.Errorf("Expected 1 success recorded, got %d", cb.successCount)
	}
	if cb.failureCount != 0 {
		t.Errorf("Expected 0 failures recorded, got %d", cb.failureCount)
	}
}

func TestCircuitBreakerMiddleware_Deny(t *testing.T) {
	cb := &mockCircuitBreaker{
		allowCalls: []bool{false},
		state:      "open",
	}

	middleware := NewCircuitBreakerMiddleware(cb)
	ctx := context.Background()

	called := false
	fn := func() error {
		called = true
		return nil
	}

	next := func(ctx context.Context, fn func() error) error {
		return fn()
	}

	err := middleware.Execute(ctx, fn, next)

	if err == nil {
		t.Error("Expected circuit breaker error")
	}
	if err.Error() != "circuit breaker is open" {
		t.Errorf("Expected circuit breaker error, got %v", err)
	}
	if called {
		t.Error("Expected function not to be called")
	}
}

func TestCircuitBreakerMiddleware_RecordFailure(t *testing.T) {
	cb := &mockCircuitBreaker{
		allowCalls: []bool{true},
		state:      "closed",
	}

	middleware := NewCircuitBreakerMiddleware(cb)
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
	if cb.successCount != 0 {
		t.Errorf("Expected 0 successes recorded, got %d", cb.successCount)
	}
	if cb.failureCount != 1 {
		t.Errorf("Expected 1 failure recorded, got %d", cb.failureCount)
	}
}

func TestCircuitBreakerMiddleware_RecordSuccess(t *testing.T) {
	cb := &mockCircuitBreaker{
		allowCalls: []bool{true},
		state:      "closed",
	}

	middleware := NewCircuitBreakerMiddleware(cb)
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
	if cb.successCount != 1 {
		t.Errorf("Expected 1 success recorded, got %d", cb.successCount)
	}
	if cb.failureCount != 0 {
		t.Errorf("Expected 0 failures recorded, got %d", cb.failureCount)
	}
}

func TestCircuitBreakerMiddleware_MultipleOperations(t *testing.T) {
	cb := &mockCircuitBreaker{
		allowCalls: []bool{true, true, false, true},
		state:      "half-open",
	}

	middleware := NewCircuitBreakerMiddleware(cb)
	ctx := context.Background()

	next := func(ctx context.Context, fn func() error) error {
		return fn()
	}

	// First operation - success
	err := middleware.Execute(ctx, func() error { return nil }, next)
	if err != nil {
		t.Errorf("First operation failed: %v", err)
	}

	// Second operation - failure
	testError := errors.New("test error")
	err = middleware.Execute(ctx, func() error { return testError }, next)
	if err != testError {
		t.Errorf("Second operation: expected test error, got %v", err)
	}

	// Third operation - circuit breaker denies
	err = middleware.Execute(ctx, func() error { return nil }, next)
	if err == nil || err.Error() != "circuit breaker is open" {
		t.Errorf("Third operation: expected circuit breaker error, got %v", err)
	}

	// Fourth operation - success
	err = middleware.Execute(ctx, func() error { return nil }, next)
	if err != nil {
		t.Errorf("Fourth operation failed: %v", err)
	}

	// Check final counts
	if cb.successCount != 2 {
		t.Errorf("Expected 2 successes, got %d", cb.successCount)
	}
	if cb.failureCount != 1 {
		t.Errorf("Expected 1 failure, got %d", cb.failureCount)
	}
}

func TestNewCircuitBreakerMiddleware(t *testing.T) {
	cb := &mockCircuitBreaker{state: "closed"}
	middleware := NewCircuitBreakerMiddleware(cb)

	if middleware.CircuitBreaker != cb {
		t.Error("CircuitBreaker not set correctly")
	}
}
