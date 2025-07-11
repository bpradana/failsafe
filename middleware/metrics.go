package middleware

import (
	"context"
)

// MetricsMiddleware adds metrics collection to retry operations
type MetricsMiddleware struct {
	OnAttempt func(attempt int)
	OnSuccess func(totalAttempts int)
	OnFailure func(totalAttempts int, err error)
}

func (m *MetricsMiddleware) Execute(ctx context.Context, fn func() error, next func(context.Context, func() error) error) error {
	var attempts int

	wrappedFn := func() error {
		attempts++
		if m.OnAttempt != nil {
			m.OnAttempt(attempts)
		}
		return fn()
	}

	err := next(ctx, wrappedFn)

	// If attempts is 0, it means next() returned an error without calling wrappedFn
	// We should still record it as an attempt and failure
	if attempts == 0 {
		attempts = 1
		if m.OnAttempt != nil {
			m.OnAttempt(attempts)
		}
	}

	if err != nil {
		if m.OnFailure != nil {
			m.OnFailure(attempts, err)
		}
	} else {
		if m.OnSuccess != nil {
			m.OnSuccess(attempts)
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
