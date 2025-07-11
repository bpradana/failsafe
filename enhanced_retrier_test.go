package failsafe

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// Mock middleware for testing
type mockMiddleware struct {
	called     bool
	shouldFail bool
	mu         sync.Mutex
}

func (m *mockMiddleware) Execute(ctx context.Context, fn func() error, next func(context.Context, func() error) error) error {
	m.mu.Lock()
	m.called = true
	m.mu.Unlock()

	if m.shouldFail {
		return errors.New("middleware error")
	}

	return next(ctx, fn)
}

func (m *mockMiddleware) wasCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.called
}

func (m *mockMiddleware) reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.called = false
}

func TestEnhancedRetrier_NoMiddleware(t *testing.T) {
	retrier := NewEnhancedRetrier(WithMaxAttempts(3))
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

func TestEnhancedRetrier_SingleMiddleware(t *testing.T) {
	retrier := NewEnhancedRetrier(WithMaxAttempts(3))
	middleware := &mockMiddleware{}
	retrier.AddMiddleware(middleware)

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
	if !middleware.wasCalled() {
		t.Error("Expected middleware to be called")
	}
}

func TestEnhancedRetrier_MultipleMiddleware(t *testing.T) {
	retrier := NewEnhancedRetrier(WithMaxAttempts(3))

	middleware1 := &mockMiddleware{}
	middleware2 := &mockMiddleware{}
	middleware3 := &mockMiddleware{}

	retrier.AddMiddleware(middleware1)
	retrier.AddMiddleware(middleware2)
	retrier.AddMiddleware(middleware3)

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

	// All middleware should be called
	if !middleware1.wasCalled() {
		t.Error("Expected middleware1 to be called")
	}
	if !middleware2.wasCalled() {
		t.Error("Expected middleware2 to be called")
	}
	if !middleware3.wasCalled() {
		t.Error("Expected middleware3 to be called")
	}
}

func TestEnhancedRetrier_MiddlewareFailure(t *testing.T) {
	retrier := NewEnhancedRetrier(WithMaxAttempts(3))

	middleware1 := &mockMiddleware{}
	middleware2 := &mockMiddleware{shouldFail: true}
	middleware3 := &mockMiddleware{}

	retrier.AddMiddleware(middleware1)
	retrier.AddMiddleware(middleware2)
	retrier.AddMiddleware(middleware3)

	ctx := context.Background()

	attempts := 0
	err := retrier.Retry(ctx, func() error {
		attempts++
		return nil
	})

	if err == nil {
		t.Error("Expected middleware error")
	}
	if err.Error() != "middleware error" {
		t.Errorf("Expected 'middleware error', got %v", err)
	}

	// First two middleware should be called, third should not
	if !middleware1.wasCalled() {
		t.Error("Expected middleware1 to be called")
	}
	if !middleware2.wasCalled() {
		t.Error("Expected middleware2 to be called")
	}
	if middleware3.wasCalled() {
		t.Error("Expected middleware3 NOT to be called")
	}

	// Function should not be called due to middleware failure
	if attempts != 0 {
		t.Errorf("Expected 0 attempts due to middleware failure, got %d", attempts)
	}
}

func TestEnhancedRetrier_MiddlewareOrder(t *testing.T) {
	retrier := NewEnhancedRetrier(WithMaxAttempts(3))

	var executionOrder []string

	middleware1 := &orderTrackingMiddleware{name: "middleware1", order: &executionOrder}
	middleware2 := &orderTrackingMiddleware{name: "middleware2", order: &executionOrder}
	middleware3 := &orderTrackingMiddleware{name: "middleware3", order: &executionOrder}

	retrier.AddMiddleware(middleware1)
	retrier.AddMiddleware(middleware2)
	retrier.AddMiddleware(middleware3)

	ctx := context.Background()

	err := retrier.Retry(ctx, func() error {
		executionOrder = append(executionOrder, "function")
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	expectedOrder := []string{"middleware1", "middleware2", "middleware3", "function"}
	if len(executionOrder) != len(expectedOrder) {
		t.Errorf("Expected order length %d, got %d", len(expectedOrder), len(executionOrder))
	}

	for i, expected := range expectedOrder {
		if i >= len(executionOrder) || executionOrder[i] != expected {
			t.Errorf("Expected execution order %v, got %v", expectedOrder, executionOrder)
			break
		}
	}
}

func TestEnhancedRetrier_ConcurrentMiddleware(t *testing.T) {
	retrier := NewEnhancedRetrier(WithMaxAttempts(1))
	middleware := &mockMiddleware{}
	retrier.AddMiddleware(middleware)

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

func TestEnhancedRetrier_MiddlewareWithRetryLogic(t *testing.T) {
	retrier := NewEnhancedRetrier(WithMaxAttempts(3))
	middleware := &mockMiddleware{}
	retrier.AddMiddleware(middleware)

	ctx := context.Background()

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
	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
	if !middleware.wasCalled() {
		t.Error("Expected middleware to be called")
	}
}

func TestEnhancedRetrier_AddMiddlewareAfterCreation(t *testing.T) {
	retrier := NewEnhancedRetrier(WithMaxAttempts(1))

	middleware1 := &mockMiddleware{}
	middleware2 := &mockMiddleware{}

	// Add middleware after creation
	retrier.AddMiddleware(middleware1)

	ctx := context.Background()

	err := retrier.Retry(ctx, func() error {
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !middleware1.wasCalled() {
		t.Error("Expected middleware1 to be called")
	}

	// Add another middleware
	middleware1.reset()
	retrier.AddMiddleware(middleware2)

	err = retrier.Retry(ctx, func() error {
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !middleware1.wasCalled() {
		t.Error("Expected middleware1 to be called")
	}
	if !middleware2.wasCalled() {
		t.Error("Expected middleware2 to be called")
	}
}

// Helper middleware for testing execution order
type orderTrackingMiddleware struct {
	name  string
	order *[]string
}

func (m *orderTrackingMiddleware) Execute(ctx context.Context, fn func() error, next func(context.Context, func() error) error) error {
	*m.order = append(*m.order, m.name)
	return next(ctx, fn)
}

func TestEnhancedRetrier_MiddlewareContextPropagation(t *testing.T) {
	retrier := NewEnhancedRetrier(WithMaxAttempts(1))

	middleware := &contextCheckingMiddleware{}
	retrier.AddMiddleware(middleware)

	ctx := context.WithValue(context.Background(), "test-key", "test-value")

	err := retrier.Retry(ctx, func() error {
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !middleware.contextReceived {
		t.Error("Expected context to be propagated to middleware")
	}
}

// Helper middleware for testing context propagation
type contextCheckingMiddleware struct {
	contextReceived bool
}

func (m *contextCheckingMiddleware) Execute(ctx context.Context, fn func() error, next func(context.Context, func() error) error) error {
	if ctx.Value("test-key") == "test-value" {
		m.contextReceived = true
	}
	return next(ctx, fn)
}

func TestNewEnhancedRetrier_InheritsConfig(t *testing.T) {
	retrier := NewEnhancedRetrier(
		WithMaxAttempts(5),
		WithDelayStrategy(&FixedDelay{Delay: 100 * time.Millisecond}),
	)

	config := retrier.GetConfig()
	if config.MaxAttempts != 5 {
		t.Errorf("Expected MaxAttempts 5, got %d", config.MaxAttempts)
	}

	if config.DelayStrategy == nil {
		t.Error("Expected DelayStrategy to be set")
	}
}
