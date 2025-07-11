package internal

import (
	"errors"
	"testing"
)

func TestRetryError_Error(t *testing.T) {
	originalErr := errors.New("original error")
	retryErr := NewRetryError(3, originalErr)
	
	expected := "retry failed after 3 attempts: original error"
	if retryErr.Error() != expected {
		t.Errorf("Expected error message '%s', got '%s'", expected, retryErr.Error())
	}
}

func TestRetryError_Unwrap(t *testing.T) {
	originalErr := errors.New("original error")
	retryErr := NewRetryError(3, originalErr)
	
	if retryErr.Unwrap() != originalErr {
		t.Error("Expected Unwrap to return the original error")
	}
}

func TestRetryError_Fields(t *testing.T) {
	originalErr := errors.New("test error")
	retryErr := NewRetryError(5, originalErr)
	
	if retryErr.Attempts != 5 {
		t.Errorf("Expected Attempts 5, got %d", retryErr.Attempts)
	}
	
	if retryErr.LastErr != originalErr {
		t.Error("Expected LastErr to be the original error")
	}
}

func TestRetryError_WithNilError(t *testing.T) {
	retryErr := NewRetryError(2, nil)
	
	expected := "retry failed after 2 attempts: <nil>"
	if retryErr.Error() != expected {
		t.Errorf("Expected error message '%s', got '%s'", expected, retryErr.Error())
	}
	
	if retryErr.Unwrap() != nil {
		t.Error("Expected Unwrap to return nil")
	}
}

func TestRetryError_IsErrorInterface(t *testing.T) {
	retryErr := NewRetryError(1, errors.New("test"))
	
	// Test that it implements the error interface
	var err error = retryErr
	if err == nil {
		t.Error("RetryError should implement error interface")
	}
}

func TestRetryError_ErrorsIs(t *testing.T) {
	originalErr := errors.New("original error")
	retryErr := NewRetryError(3, originalErr)
	
	if !errors.Is(retryErr, originalErr) {
		t.Error("Expected errors.Is to return true for the wrapped error")
	}
}

func TestRetryError_ErrorsAs(t *testing.T) {
	originalErr := &CustomError{Message: "custom error"}
	retryErr := NewRetryError(3, originalErr)
	
	var customErr *CustomError
	if !errors.As(retryErr, &customErr) {
		t.Error("Expected errors.As to return true for the wrapped error type")
	}
	
	if customErr.Message != "custom error" {
		t.Errorf("Expected custom error message 'custom error', got '%s'", customErr.Message)
	}
}

func TestRetryError_ZeroAttempts(t *testing.T) {
	originalErr := errors.New("test error")
	retryErr := NewRetryError(0, originalErr)
	
	expected := "retry failed after 0 attempts: test error"
	if retryErr.Error() != expected {
		t.Errorf("Expected error message '%s', got '%s'", expected, retryErr.Error())
	}
}

func TestRetryError_LargeAttempts(t *testing.T) {
	originalErr := errors.New("test error")
	retryErr := NewRetryError(1000, originalErr)
	
	expected := "retry failed after 1000 attempts: test error"
	if retryErr.Error() != expected {
		t.Errorf("Expected error message '%s', got '%s'", expected, retryErr.Error())
	}
}

// Helper type for testing error unwrapping
type CustomError struct {
	Message string
}

func (e *CustomError) Error() string {
	return e.Message
}