package middleware

import (
	"context"
	"sync"
)

// MetricsMiddleware adds metrics collection to retry operations
type MetricsMiddleware struct {
	OnAttempt func(attempt int)
	OnSuccess func(totalAttempts int)
	OnFailure func(totalAttempts int, err error)
	mu        sync.Mutex
	attempts  int
}

// CallOnAttempt is called for each attempt
func (m *MetricsMiddleware) CallOnAttempt(attempt int) {
	if m.OnAttempt != nil {
		m.OnAttempt(attempt)
	}
}

// CallOnSuccess is called on successful completion
func (m *MetricsMiddleware) CallOnSuccess(totalAttempts int) {
	if m.OnSuccess != nil {
		m.OnSuccess(totalAttempts)
	}
}

// CallOnFailure is called on final failure
func (m *MetricsMiddleware) CallOnFailure(totalAttempts int, err error) {
	if m.OnFailure != nil {
		m.OnFailure(totalAttempts, err)
	}
}

func (m *MetricsMiddleware) Execute(ctx context.Context, fn func() error, next func(context.Context, func() error) error) error {
	m.mu.Lock()
	m.attempts = 0
	m.mu.Unlock()

	wrappedFn := func() error {
		m.mu.Lock()
		m.attempts++
		currentAttempts := m.attempts
		m.mu.Unlock()

		if m.OnAttempt != nil {
			m.OnAttempt(currentAttempts)
		}
		return fn()
	}

	err := next(ctx, wrappedFn)

	m.mu.Lock()
	totalAttempts := m.attempts
	m.mu.Unlock()

	// In async mode, next() returns nil immediately without calling wrappedFn
	// So we shouldn't count this as an attempt or call success/failure callbacks
	// The async execution will handle the callbacks directly
	if totalAttempts == 0 && err == nil {
		// This is async mode - don't count as attempt or success/failure
		return err
	}

	// If attempts is 0 and there's an error, it means next() returned an error without calling wrappedFn
	// We should still record it as an attempt and failure
	if totalAttempts == 0 {
		totalAttempts = 1
		if m.OnAttempt != nil {
			m.OnAttempt(totalAttempts)
		}
	}

	if err != nil {
		if m.OnFailure != nil {
			m.OnFailure(totalAttempts, err)
		}
	} else {
		if m.OnSuccess != nil {
			m.OnSuccess(totalAttempts)
		}
	}

	return err
}

// NewMetricsMiddleware creates a new metrics middleware
func NewMetricsMiddleware(onAttempt func(int), onSuccess func(int), onFailure func(int, error)) *MetricsMiddleware {
	return &MetricsMiddleware{
		OnAttempt: onAttempt,
		OnSuccess: onSuccess,
		OnFailure: onFailure,
	}
}
