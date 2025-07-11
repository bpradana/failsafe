package failsafe

import (
	"context"
	"errors"
	"testing"
	"time"
)

func BenchmarkRetrier_Success(b *testing.B) {
	retrier := NewRetrier(WithMaxAttempts(1))
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := retrier.Retry(ctx, func() error {
			return nil
		})
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}

func BenchmarkRetrier_FailureAndRetry(b *testing.B) {
	retrier := NewRetrier(WithMaxAttempts(3))
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		attempts := 0
		err := retrier.Retry(ctx, func() error {
			attempts++
			if attempts < 2 {
				return errors.New("temporary failure")
			}
			return nil
		})
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}

func BenchmarkRetrier_WithFixedDelay(b *testing.B) {
	retrier := NewRetrier(
		WithMaxAttempts(3),
		WithDelayStrategy(&FixedDelay{Delay: 1 * time.Millisecond}),
	)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		attempts := 0
		err := retrier.Retry(ctx, func() error {
			attempts++
			if attempts < 2 {
				return errors.New("temporary failure")
			}
			return nil
		})
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}

func BenchmarkRetrier_WithHooks(b *testing.B) {
	retrier := NewRetrier(
		WithMaxAttempts(2),
		WithOnRetry(func(ctx context.Context, attempt int, err error, nextDelay time.Duration) {
			// Simulate some work in the hook
		}),
		WithOnSuccess(func(ctx context.Context, attempt int, err error, nextDelay time.Duration) {
			// Simulate some work in the hook
		}),
	)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		attempts := 0
		err := retrier.Retry(ctx, func() error {
			attempts++
			if attempts < 2 {
				return errors.New("temporary failure")
			}
			return nil
		})
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}

func BenchmarkRetryWithResult_Success(b *testing.B) {
	retrier := NewRetrier(WithMaxAttempts(1))
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := RetryWithResult(ctx, retrier, func() (int, error) {
			return 42, nil
		})
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
		if result != 42 {
			b.Fatalf("Unexpected result: %d", result)
		}
	}
}

func BenchmarkRetryWithResult_FailureAndRetry(b *testing.B) {
	retrier := NewRetrier(WithMaxAttempts(3))
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		attempts := 0
		result, err := RetryWithResult(ctx, retrier, func() (string, error) {
			attempts++
			if attempts < 2 {
				return "", errors.New("temporary failure")
			}
			return "success", nil
		})
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
		if result != "success" {
			b.Fatalf("Unexpected result: %s", result)
		}
	}
}

func BenchmarkEnhancedRetrier_NoMiddleware(b *testing.B) {
	retrier := NewEnhancedRetrier(WithMaxAttempts(1))
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := retrier.Retry(ctx, func() error {
			return nil
		})
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}

func BenchmarkEnhancedRetrier_WithMiddleware(b *testing.B) {
	retrier := NewEnhancedRetrier(WithMaxAttempts(1))
	middleware := &benchmarkMiddleware{}
	retrier.AddMiddleware(middleware)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := retrier.Retry(ctx, func() error {
			return nil
		})
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}

func BenchmarkEnhancedRetrier_MultipleMiddleware(b *testing.B) {
	retrier := NewEnhancedRetrier(WithMaxAttempts(1))

	// Add multiple middleware
	for i := 0; i < 5; i++ {
		middleware := &benchmarkMiddleware{}
		retrier.AddMiddleware(middleware)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := retrier.Retry(ctx, func() error {
			return nil
		})
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}

func BenchmarkRetryWithExponentialBackoff(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		attempts := 0
		err := RetryWithExponentialBackoff(ctx, func() error {
			attempts++
			if attempts < 2 {
				return errors.New("temporary failure")
			}
			return nil
		}, 3)
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}

func BenchmarkConfigUpdate(b *testing.B) {
	retrier := NewRetrier(WithMaxAttempts(1))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		retrier.UpdateConfig(WithMaxAttempts(i%10 + 1))
	}
}

func BenchmarkConfigGet(b *testing.B) {
	retrier := NewRetrier(WithMaxAttempts(3))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config := retrier.GetConfig()
		_ = config.MaxAttempts
	}
}

func BenchmarkConcurrentRetries(b *testing.B) {
	retrier := NewRetrier(WithMaxAttempts(1))
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := retrier.Retry(ctx, func() error {
				return nil
			})
			if err != nil {
				b.Fatalf("Unexpected error: %v", err)
			}
		}
	})
}

func BenchmarkConcurrentRetriesWithResult(b *testing.B) {
	retrier := NewRetrier(WithMaxAttempts(1))
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			result, err := RetryWithResult(ctx, retrier, func() (int, error) {
				return 42, nil
			})
			if err != nil {
				b.Fatalf("Unexpected error: %v", err)
			}
			if result != 42 {
				b.Fatalf("Unexpected result: %d", result)
			}
		}
	})
}

func BenchmarkErrorFiltering(b *testing.B) {
	retrier := NewRetrier(
		WithMaxAttempts(1),
		WithErrorFilter(func(err error) bool {
			// Simulate some filtering logic
			return err.Error() != "non-retryable"
		}),
	)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := retrier.Retry(ctx, func() error {
			return nil
		})
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}

// Helper middleware for benchmarking
type benchmarkMiddleware struct{}

func (m *benchmarkMiddleware) Execute(ctx context.Context, fn func() error, next func(context.Context, func() error) error) error {
	// Simulate minimal middleware work
	return next(ctx, fn)
}

// Memory allocation benchmarks
func BenchmarkMemoryAllocation_BasicRetry(b *testing.B) {
	retrier := NewRetrier(WithMaxAttempts(1))
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := retrier.Retry(ctx, func() error {
			return nil
		})
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}

func BenchmarkMemoryAllocation_RetryWithResult(b *testing.B) {
	retrier := NewRetrier(WithMaxAttempts(1))
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		result, err := RetryWithResult(ctx, retrier, func() (int, error) {
			return 42, nil
		})
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
		if result != 42 {
			b.Fatalf("Unexpected result: %d", result)
		}
	}
}

func BenchmarkMemoryAllocation_EnhancedRetrier(b *testing.B) {
	retrier := NewEnhancedRetrier(WithMaxAttempts(1))
	middleware := &benchmarkMiddleware{}
	retrier.AddMiddleware(middleware)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := retrier.Retry(ctx, func() error {
			return nil
		})
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}

// Comparison benchmarks
func BenchmarkCompare_DirectCall(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := func() error {
			return nil
		}()
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}

func BenchmarkCompare_RetryWithOneAttempt(b *testing.B) {
	retrier := NewRetrier(WithMaxAttempts(1))
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := retrier.Retry(ctx, func() error {
			return nil
		})
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}
