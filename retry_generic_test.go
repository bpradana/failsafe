package failsafe

import (
	"context"
	"errors"
	"testing"
)

func TestRetryWithResult_Success(t *testing.T) {
	retrier := NewRetrier(WithMaxAttempts(3))
	ctx := context.Background()

	attempts := 0
	result, err := RetryWithResult(ctx, retrier, func() (string, error) {
		attempts++
		return "success", nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != "success" {
		t.Errorf("Expected result 'success', got %s", result)
	}
	if attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", attempts)
	}
}

func TestRetryWithResult_EventualSuccess(t *testing.T) {
	retrier := NewRetrier(WithMaxAttempts(3))
	ctx := context.Background()

	attempts := 0
	result, err := RetryWithResult(ctx, retrier, func() (int, error) {
		attempts++
		if attempts < 3 {
			return 0, errors.New("temporary failure")
		}
		return 42, nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != 42 {
		t.Errorf("Expected result 42, got %d", result)
	}
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestRetryWithResult_Failure(t *testing.T) {
	retrier := NewRetrier(WithMaxAttempts(2))
	ctx := context.Background()

	attempts := 0
	result, err := RetryWithResult(ctx, retrier, func() (string, error) {
		attempts++
		return "", errors.New("persistent failure")
	})

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if result != "" {
		t.Errorf("Expected empty result, got %s", result)
	}
	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
}

func TestRetryWithResult_PartialResult(t *testing.T) {
	retrier := NewRetrier(WithMaxAttempts(2))
	ctx := context.Background()

	attempts := 0
	result, err := RetryWithResult(ctx, retrier, func() ([]int, error) {
		attempts++
		if attempts == 1 {
			return []int{1, 2}, errors.New("partial failure")
		}
		return []int{}, errors.New("complete failure")
	})

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if len(result) != 0 {
		t.Errorf("Expected empty result, got %v", result)
	}
	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
}

func TestRetryWithResult_ContextCancellation(t *testing.T) {
	retrier := NewRetrier(WithMaxAttempts(5))
	ctx, cancel := context.WithCancel(context.Background())

	attempts := 0
	result, err := RetryWithResult(ctx, retrier, func() (bool, error) {
		attempts++
		if attempts == 2 {
			cancel()
		}
		return false, errors.New("failure")
	})

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context cancellation error, got %v", err)
	}
	if result != false {
		t.Errorf("Expected false result, got %v", result)
	}
}

func TestRetryWithResult_DifferentTypes(t *testing.T) {
	retrier := NewRetrier(WithMaxAttempts(1))
	ctx := context.Background()

	// Test with string
	stringResult, err := RetryWithResult(ctx, retrier, func() (string, error) {
		return "test", nil
	})
	if err != nil || stringResult != "test" {
		t.Errorf("String test failed: result=%s, err=%v", stringResult, err)
	}

	// Test with int
	intResult, err := RetryWithResult(ctx, retrier, func() (int, error) {
		return 123, nil
	})
	if err != nil || intResult != 123 {
		t.Errorf("Int test failed: result=%d, err=%v", intResult, err)
	}

	// Test with struct
	type TestStruct struct {
		Name string
		Age  int
	}

	structResult, err := RetryWithResult(ctx, retrier, func() (TestStruct, error) {
		return TestStruct{Name: "John", Age: 30}, nil
	})
	if err != nil || structResult.Name != "John" || structResult.Age != 30 {
		t.Errorf("Struct test failed: result=%+v, err=%v", structResult, err)
	}

	// Test with slice
	sliceResult, err := RetryWithResult(ctx, retrier, func() ([]int, error) {
		return []int{1, 2, 3}, nil
	})
	if err != nil || len(sliceResult) != 3 {
		t.Errorf("Slice test failed: result=%v, err=%v", sliceResult, err)
	}

	// Test with map
	mapResult, err := RetryWithResult(ctx, retrier, func() (map[string]int, error) {
		return map[string]int{"key": 42}, nil
	})
	if err != nil || mapResult["key"] != 42 {
		t.Errorf("Map test failed: result=%v, err=%v", mapResult, err)
	}
}

func TestRetryWithResult_Pointer(t *testing.T) {
	retrier := NewRetrier(WithMaxAttempts(1))
	ctx := context.Background()

	type TestStruct struct {
		Value int
	}

	// Test with pointer
	obj := &TestStruct{Value: 100}
	result, err := RetryWithResult(ctx, retrier, func() (*TestStruct, error) {
		return obj, nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != obj {
		t.Errorf("Expected same pointer, got different")
	}
	if result.Value != 100 {
		t.Errorf("Expected value 100, got %d", result.Value)
	}
}

func TestRetryWithResult_Interface(t *testing.T) {
	retrier := NewRetrier(WithMaxAttempts(1))
	ctx := context.Background()

	// Test with interface{}
	result, err := RetryWithResult(ctx, retrier, func() (interface{}, error) {
		return "interface test", nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != "interface test" {
		t.Errorf("Expected 'interface test', got %v", result)
	}
}

func TestRetryWithResult_ZeroValues(t *testing.T) {
	retrier := NewRetrier(WithMaxAttempts(1))
	ctx := context.Background()

	// Test that zero values are returned correctly on error
	result, err := RetryWithResult(ctx, retrier, func() (int, error) {
		return 42, errors.New("test error")
	})

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if result != 0 {
		t.Errorf("Expected zero value 0, got %d", result)
	}

	// Test with string
	stringResult, err := RetryWithResult(ctx, retrier, func() (string, error) {
		return "value", errors.New("test error")
	})

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if stringResult != "" {
		t.Errorf("Expected empty string, got %s", stringResult)
	}

	// Test with slice
	sliceResult, err := RetryWithResult(ctx, retrier, func() ([]int, error) {
		return []int{1, 2, 3}, errors.New("test error")
	})

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if sliceResult != nil {
		t.Errorf("Expected nil slice, got %v", sliceResult)
	}
}

func TestRetryWithResult_ConcurrentAccess(t *testing.T) {
	retrier := NewRetrier(WithMaxAttempts(1))
	ctx := context.Background()

	const numGoroutines = 10
	results := make(chan int, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(value int) {
			result, err := RetryWithResult(ctx, retrier, func() (int, error) {
				return value, nil
			})
			results <- result
			errors <- err
		}(i)
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		err := <-errors
		if err != nil {
			t.Errorf("Concurrent test failed: %v", err)
		}
	}

	// Check that we got all expected results
	receivedResults := make(map[int]bool)
	for i := 0; i < numGoroutines; i++ {
		result := <-results
		receivedResults[result] = true
	}

	for i := 0; i < numGoroutines; i++ {
		if !receivedResults[i] {
			t.Errorf("Missing result for value %d", i)
		}
	}
}
