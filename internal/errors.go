package internal

import "fmt"

// RetryError represents an error that occurred during retry attempts
type RetryError struct {
	Attempts int
	LastErr  error
}

func (e *RetryError) Error() string {
	return fmt.Sprintf("retry failed after %d attempts: %v", e.Attempts, e.LastErr)
}

func (e *RetryError) Unwrap() error {
	return e.LastErr
}

// NewRetryError creates a new retry error
func NewRetryError(attempts int, lastErr error) *RetryError {
	return &RetryError{
		Attempts: attempts,
		LastErr:  lastErr,
	}
}