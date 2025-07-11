package failsafe

import "context"

// RetryWithResult executes the given function with retry logic and returns a result
func RetryWithResult[T any](ctx context.Context, r *Retrier, fn RetryFuncWithResult[T]) (T, error) {
	var result T

	err := r.Retry(ctx, func() error {
		var err error
		result, err = fn()
		return err
	})

	if err != nil {
		var zero T
		return zero, err
	}

	return result, nil
}
