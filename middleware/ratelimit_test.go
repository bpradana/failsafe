package middleware

import (
	"context"
	"errors"
	"testing"
	"time"
)

// Mock rate limiter for testing
type mockRateLimiter struct {
	allowCalls []bool
	allowIndex int
	waitError  error
	waitDelay  time.Duration
}

func (m *mockRateLimiter) Allow() bool {
	if m.allowIndex >= len(m.allowCalls) {
		return true
	}
	result := m.allowCalls[m.allowIndex]
	m.allowIndex++
	return result
}

func (m *mockRateLimiter) Wait(ctx context.Context) error {
	if m.waitDelay > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(m.waitDelay):
		}
	}
	return m.waitError
}

func TestRateLimitMiddleware_Allow(t *testing.T) {
	rl := &mockRateLimiter{
		allowCalls: []bool{true},
	}

	middleware := NewRateLimitMiddleware(rl)
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
}

func TestRateLimitMiddleware_Deny(t *testing.T) {
	rl := &mockRateLimiter{
		allowCalls: []bool{false},
	}

	middleware := NewRateLimitMiddleware(rl)
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
		t.Error("Expected rate limit error")
	}
	if err.Error() != "rate limit exceeded" {
		t.Errorf("Expected rate limit error, got %v", err)
	}
	if called {
		t.Error("Expected function not to be called")
	}
}

func TestRateLimitMiddleware_WaitError(t *testing.T) {
	waitError := errors.New("wait failed")
	rl := &mockRateLimiter{
		waitError: waitError,
	}

	middleware := NewRateLimitMiddleware(rl)
	ctx := context.Background()

	fn := func() error {
		return nil
	}

	next := func(ctx context.Context, fn func() error) error {
		return fn()
	}

	err := middleware.Execute(ctx, fn, next)

	if err == nil {
		t.Error("Expected wait error")
	}
	if !errors.Is(err, waitError) {
		t.Errorf("Expected wait error, got %v", err)
	}
}

func TestRateLimitMiddleware_WaitTimeout(t *testing.T) {
	rl := &mockRateLimiter{
		waitDelay: 100 * time.Millisecond,
	}

	middleware := NewRateLimitMiddleware(rl)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	fn := func() error {
		return nil
	}

	next := func(ctx context.Context, fn func() error) error {
		return fn()
	}

	err := middleware.Execute(ctx, fn, next)

	if err == nil {
		t.Error("Expected timeout error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected timeout error, got %v", err)
	}
}

func TestRateLimitMiddleware_WaitSuccess(t *testing.T) {
	rl := &mockRateLimiter{
		allowCalls: []bool{true},
		waitDelay:  10 * time.Millisecond,
	}

	middleware := NewRateLimitMiddleware(rl)
	ctx := context.Background()

	called := false
	fn := func() error {
		called = true
		return nil
	}

	next := func(ctx context.Context, fn func() error) error {
		return fn()
	}

	start := time.Now()
	err := middleware.Execute(ctx, fn, next)
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !called {
		t.Error("Expected function to be called")
	}
	if duration < 10*time.Millisecond {
		t.Errorf("Expected wait delay, got %v", duration)
	}
}

func TestRateLimitMiddleware_FunctionError(t *testing.T) {
	rl := &mockRateLimiter{
		allowCalls: []bool{true},
	}

	middleware := NewRateLimitMiddleware(rl)
	ctx := context.Background()

	testError := errors.New("function error")
	fn := func() error {
		return testError
	}

	next := func(ctx context.Context, fn func() error) error {
		return fn()
	}

	err := middleware.Execute(ctx, fn, next)

	if err != testError {
		t.Errorf("Expected function error, got %v", err)
	}
}

func TestRateLimitMiddleware_MultipleOperations(t *testing.T) {
	rl := &mockRateLimiter{
		allowCalls: []bool{true, false, true},
	}

	middleware := NewRateLimitMiddleware(rl)
	ctx := context.Background()

	next := func(ctx context.Context, fn func() error) error {
		return fn()
	}

	// First operation - allowed
	err := middleware.Execute(ctx, func() error { return nil }, next)
	if err != nil {
		t.Errorf("First operation failed: %v", err)
	}

	// Second operation - rate limited
	err = middleware.Execute(ctx, func() error { return nil }, next)
	if err == nil || err.Error() != "rate limit exceeded" {
		t.Errorf("Second operation: expected rate limit error, got %v", err)
	}

	// Third operation - allowed
	err = middleware.Execute(ctx, func() error { return nil }, next)
	if err != nil {
		t.Errorf("Third operation failed: %v", err)
	}
}

func TestNewRateLimitMiddleware(t *testing.T) {
	rl := &mockRateLimiter{}
	middleware := NewRateLimitMiddleware(rl)

	if middleware.RateLimiter != rl {
		t.Error("RateLimiter not set correctly")
	}
}
