# Failsafe

A robust, production-ready Go library for handling retries, circuit breakers, and failure resilience patterns with comprehensive middleware support.

## Features

- **Flexible Retry Strategies**: Fixed delay, exponential backoff, and custom delay strategies
- **Jitter Support**: Reduce thundering herd problems with configurable jitter
- **Middleware Architecture**: Extensible middleware system for cross-cutting concerns
- **Circuit Breaker Pattern**: Prevent cascading failures with built-in circuit breaker support
- **Rate Limiting**: Control request rates with pluggable rate limiters
- **Metrics Collection**: Built-in metrics middleware for monitoring retry behavior
- **Generic Result Handling**: Type-safe retry operations with generic result types
- **Context Support**: Full context cancellation and timeout support
- **Thread-Safe**: Concurrent-safe operations with proper synchronization
- **Error Filtering**: Configurable error filters for selective retry logic
- **Hook System**: Comprehensive callback system for retry events
- **Async Mode**: Non-blocking retry operations with background execution

## Installation

```bash
go get github.com/bpradana/failsafe
```

## Quick Start

### Basic Retry

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "time"

    "github.com/bpradana/failsafe"
)

func main() {
    ctx := context.Background()
    
    // Create a simple retrier
    retrier := failsafe.NewRetrier(
        failsafe.WithMaxAttempts(3),
        failsafe.WithDelayStrategy(&failsafe.FixedDelay{Delay: 1 * time.Second}),
    )
    
    // Retry a function
    err := retrier.Retry(ctx, func() error {
        // Your operation here
        return errors.New("temporary failure")
    })
    
    if err != nil {
        fmt.Printf("Operation failed: %v\n", err)
    }
}
```

### Retry with Result

```go
result, err := failsafe.RetryWithResult(ctx, retrier, func() (string, error) {
    // Your operation that returns a result
    return "success", nil
})
```

## Configuration Options

### Retry Options

- `WithMaxAttempts(max int)`: Set maximum number of retry attempts
- `WithDelayStrategy(strategy DelayStrategy)`: Configure delay behavior between retries
- `WithErrorFilter(filter ErrorFilter)`: Filter which errors should trigger retries
- `WithOnRetry(hook Hook)`: Execute callback on each retry attempt
- `WithOnFinalError(hook Hook)`: Execute callback when all retries are exhausted
- `WithOnSuccess(hook Hook)`: Execute callback on successful completion
- `WithAsyncMode(async bool)`: Enable/disable asynchronous retry execution

### Error Filters

```go
// Retry all errors
failsafe.RetryAllErrors

// Retry only transient errors (excludes context cancellation/timeout)
failsafe.RetryTransientErrors

// Custom error filter
customFilter := func(err error) bool {
    return !errors.Is(err, MyPermanentError)
}
```

## Delay Strategies

### Fixed Delay

```go
strategy := &failsafe.FixedDelay{Delay: 1 * time.Second}
```

### Exponential Backoff

```go
strategy := strategies.NewExponentialBackoff(
    100*time.Millisecond, // initial delay
    5*time.Second,        // max delay
    2.0,                  // multiplier
)
```

### Exponential Backoff with Jitter

```go
strategy := strategies.ExponentialBackoffWithJitter(
    100*time.Millisecond, // initial delay
    5*time.Second,        // max delay
    2.0,                  // multiplier
)
```

### Custom Jitter Strategy

```go
baseStrategy := strategies.NewExponentialBackoff(100*time.Millisecond, 5*time.Second, 2.0)
jitterStrategy := strategies.NewJitterStrategy(baseStrategy, strategies.UniformJitter)
```

## Advanced Usage

### Comprehensive Retry Configuration

```go
retrier := failsafe.NewRetrier(
    failsafe.WithMaxAttempts(5),
    failsafe.WithDelayStrategy(strategies.ExponentialBackoffWithJitter(
        100*time.Millisecond,
        5*time.Second,
        2.0,
    )),
    failsafe.WithErrorFilter(failsafe.RetryTransientErrors),
    failsafe.WithOnRetry(func(attempt int, err error, nextDelay time.Duration) {
        log.Printf("Retry attempt %d failed: %v, next delay: %v", attempt, err, nextDelay)
    }),
    failsafe.WithOnFinalError(func(attempt int, err error, nextDelay time.Duration) {
        log.Printf("Final retry attempt %d failed: %v", attempt, err)
    }),
    failsafe.WithOnSuccess(func(attempt int, err error, nextDelay time.Duration) {
        log.Printf("Success on attempt %d", attempt)
    }),
)
```

### Enhanced Retrier with Middleware

```go
// Create enhanced retrier with middleware support
retrier := failsafe.NewEnhancedRetrier(
    failsafe.WithMaxAttempts(3),
    failsafe.WithDelayStrategy(&failsafe.FixedDelay{Delay: 500 * time.Millisecond}),
)

// Add metrics middleware
metricsMiddleware := middleware.NewMetricsMiddleware(
    func(attempt int) {
        fmt.Printf("Starting attempt %d\n", attempt)
    },
    func(totalAttempts int) {
        fmt.Printf("Success after %d attempts\n", totalAttempts)
    },
    func(totalAttempts int, err error) {
        fmt.Printf("Failed after %d attempts: %v\n", totalAttempts, err)
    },
)

retrier.AddMiddleware(metricsMiddleware)
```

### Asynchronous Retry Operations

```go
// Create retrier with async mode enabled
retrier := failsafe.NewRetrier(
    failsafe.WithMaxAttempts(3),
    failsafe.WithAsyncMode(true),
    failsafe.WithDelayStrategy(strategies.ExponentialBackoffWithJitter(
        100*time.Millisecond,
        5*time.Second,
        2.0,
    )),
    failsafe.WithOnSuccess(func(attempt int, err error, nextDelay time.Duration) {
        log.Printf("Async operation succeeded on attempt %d", attempt)
    }),
    failsafe.WithOnFinalError(func(attempt int, err error, nextDelay time.Duration) {
        log.Printf("Async operation failed after %d attempts: %v", attempt, err)
    }),
)

// This returns immediately, operation runs in background
err := retrier.Retry(ctx, func() error {
    // Your potentially long-running operation
    return performDatabaseOperation()
})

// err will be nil if async mode is enabled
// Use hooks to monitor the actual operation results
```

### Sync vs Async Mode

```go
// Synchronous mode (default)
syncRetrier := failsafe.NewRetrier(
    failsafe.WithMaxAttempts(3),
    failsafe.WithAsyncMode(false), // explicit, but this is the default
)

// Blocks until completion or failure
err := syncRetrier.Retry(ctx, operation)
if err != nil {
    // Handle error immediately
}

// Asynchronous mode
asyncRetrier := failsafe.NewRetrier(
    failsafe.WithMaxAttempts(3),
    failsafe.WithAsyncMode(true),
    failsafe.WithOnSuccess(func(attempt int, err error, nextDelay time.Duration) {
        // Handle successful completion
    }),
    failsafe.WithOnFinalError(func(attempt int, err error, nextDelay time.Duration) {
        // Handle final failure
    }),
)

// Returns immediately, operation runs in background
asyncRetrier.Retry(ctx, operation) // Always returns nil in async mode
```

## Middleware

### Built-in Middleware

#### Metrics Middleware

```go
metricsMiddleware := middleware.NewMetricsMiddleware(
    onAttempt func(int),
    onSuccess func(int),
    onFailure func(int, error),
)
```

#### Circuit Breaker Middleware

```go
// Implement CircuitBreaker interface
type MyCircuitBreaker struct {
    // Your circuit breaker implementation
}

func (cb *MyCircuitBreaker) Allow() bool { /* implementation */ }
func (cb *MyCircuitBreaker) RecordSuccess() { /* implementation */ }
func (cb *MyCircuitBreaker) RecordFailure() { /* implementation */ }
func (cb *MyCircuitBreaker) State() string { /* implementation */ }

// Use with middleware
cbMiddleware := middleware.NewCircuitBreakerMiddleware(circuitBreaker)
retrier.AddMiddleware(cbMiddleware)
```

#### Rate Limit Middleware

```go
// Implement RateLimiter interface
type MyRateLimiter struct {
    // Your rate limiter implementation
}

func (rl *MyRateLimiter) Allow() bool { /* implementation */ }
func (rl *MyRateLimiter) Wait(ctx context.Context) error { /* implementation */ }

// Use with middleware
rlMiddleware := middleware.NewRateLimitMiddleware(rateLimiter)
retrier.AddMiddleware(rlMiddleware)
```

### Custom Middleware

```go
type LoggingMiddleware struct {
    Logger *log.Logger
}

func (l *LoggingMiddleware) Execute(
    ctx context.Context,
    fn func() error,
    next func(context.Context, func() error) error,
) error {
    l.Logger.Println("Starting retry operation")
    
    err := next(ctx, fn)
    
    if err != nil {
        l.Logger.Printf("Retry operation failed: %v", err)
    } else {
        l.Logger.Println("Retry operation succeeded")
    }
    
    return err
}
```

## Convenience Functions

### Exponential Backoff Retry

```go
err := failsafe.RetryWithExponentialBackoff(ctx, func() error {
    // Your operation
    return nil
}, 3) // max attempts
```

### Retry with Logging

```go
err := failsafe.RetryWithLogging(ctx, func() error {
    // Your operation
    return nil
}, 3, log.Printf) // max attempts, logger function
```

## Dynamic Configuration

### Runtime Configuration Updates

```go
// Update retrier configuration at runtime
retrier.UpdateConfig(
    failsafe.WithMaxAttempts(5),
    failsafe.WithDelayStrategy(strategies.NewFixedDelay(2*time.Second)),
)

// Get current configuration
config := retrier.GetConfig()
fmt.Printf("Max attempts: %d\n", config.MaxAttempts)
```

## Error Handling

### Error Types

The library wraps errors to provide context about retry failures:

- `"max attempts reached: %w"` - All retry attempts exhausted
- `"non-retryable error: %w"` - Error filtered out by error filter
- `"retry cancelled: %w"` - Context cancellation during retry
- `"circuit breaker is open"` - Circuit breaker preventing execution
- `"rate limit exceeded"` - Rate limit preventing execution

### Error Filtering Examples

```go
// Retry only specific error types
retrySpecificErrors := func(err error) bool {
    var netErr *net.OpError
    var timeoutErr *TimeoutError
    return errors.As(err, &netErr) || errors.As(err, &timeoutErr)
}

retrier := failsafe.NewRetrier(
    failsafe.WithErrorFilter(retrySpecificErrors),
)
```

## Testing

### Test Utilities

```go
// Create retrier for testing with no delays
testRetrier := failsafe.NewRetrier(
    failsafe.WithMaxAttempts(3),
    failsafe.WithDelayStrategy(&failsafe.FixedDelay{Delay: 0}),
)
```

### Mock Implementations

```go
// Mock circuit breaker for testing
type MockCircuitBreaker struct {
    allowCalls bool
    state      string
}

func (m *MockCircuitBreaker) Allow() bool { return m.allowCalls }
func (m *MockCircuitBreaker) RecordSuccess() {}
func (m *MockCircuitBreaker) RecordFailure() {}
func (m *MockCircuitBreaker) State() string { return m.state }
```

## Performance Considerations

### Memory Usage

- Retriers are lightweight and can be reused across multiple operations
- Middleware chain execution has minimal overhead
- Delay strategies maintain minimal state

### Concurrency

- All components are thread-safe and can be used concurrently
- Configuration updates are atomic and don't affect ongoing operations
- Middleware execution is sequential but non-blocking

### Best Practices

1. **Reuse Retriers**: Create retriers once and reuse them for similar operations
2. **Configure Timeouts**: Always use context with timeouts for bounded retry operations
3. **Monitor Metrics**: Use metrics middleware to track retry behavior in production
4. **Appropriate Jitter**: Use jitter to prevent thundering herd problems
5. **Error Classification**: Implement proper error filters to avoid retrying permanent failures
6. **Async Mode Usage**: Use async mode for fire-and-forget operations with proper hooks for monitoring

## Examples

### HTTP Client with Retry

```go
func httpClientWithRetry() error {
    retrier := failsafe.NewRetrier(
        failsafe.WithMaxAttempts(3),
        failsafe.WithDelayStrategy(strategies.ExponentialBackoffWithJitter(
            100*time.Millisecond,
            5*time.Second,
            2.0,
        )),
        failsafe.WithErrorFilter(func(err error) bool {
            // Retry on network errors but not on 4xx client errors
            var netErr *net.OpError
            if errors.As(err, &netErr) {
                return true
            }
            
            var httpErr *HTTPError
            if errors.As(err, &httpErr) {
                return httpErr.StatusCode >= 500
            }
            
            return false
        }),
    )
    
    return retrier.Retry(ctx, func() error {
        resp, err := http.Get("https://api.example.com/data")
        if err != nil {
            return err
        }
        defer resp.Body.Close()
        
        if resp.StatusCode >= 400 {
            return &HTTPError{StatusCode: resp.StatusCode}
        }
        
        return nil
    })
}
```

### Database Operation with Circuit Breaker

```go
func databaseOperationWithCircuitBreaker() error {
    circuitBreaker := &MyCircuitBreaker{
        failureThreshold: 5,
        resetTimeout:     30 * time.Second,
    }
    
    retrier := failsafe.NewEnhancedRetrier(
        failsafe.WithMaxAttempts(3),
        failsafe.WithDelayStrategy(strategies.NewFixedDelay(1*time.Second)),
    )
    
    retrier.AddMiddleware(middleware.NewCircuitBreakerMiddleware(circuitBreaker))
    
    return retrier.Retry(ctx, func() error {
        // Database operation
        return db.QueryRow("SELECT * FROM users").Scan(&user)
    })
}
```

### Async Background Processing

```go
func asyncBackgroundProcessing() {
    // Create async retrier for background tasks
    asyncRetrier := failsafe.NewRetrier(
        failsafe.WithMaxAttempts(5),
        failsafe.WithAsyncMode(true),
        failsafe.WithDelayStrategy(strategies.ExponentialBackoffWithJitter(
            500*time.Millisecond,
            30*time.Second,
            1.5,
        )),
        failsafe.WithOnSuccess(func(attempt int, err error, nextDelay time.Duration) {
            log.Printf("Background task completed successfully after %d attempts", attempt)
            // Update task status in database
            updateTaskStatus(taskID, "completed")
        }),
        failsafe.WithOnFinalError(func(attempt int, err error, nextDelay time.Duration) {
            log.Printf("Background task failed permanently after %d attempts: %v", attempt, err)
            // Mark task as failed and notify administrators
            updateTaskStatus(taskID, "failed")
            sendAlertToAdmins(fmt.Sprintf("Task %s failed: %v", taskID, err))
        }),
    )
    
    // Start background processing - returns immediately
    asyncRetrier.Retry(ctx, func() error {
        return processLargeDataset()
    })
    
    // Continue with other work while background task runs
    log.Println("Background task started, continuing with other work...")
}
```

## Project Structure

```
failsafe/
├── failsafe.go          # Core types and interfaces
├── retrier.go           # Main retry implementation
├── config.go            # Configuration options
├── retry_generic.go     # Generic retry with result
├── strategies/          # Delay strategies
│   ├── delay.go         # Fixed and exponential backoff
│   └── jitter.go        # Jitter implementations
├── middleware/          # Middleware implementations
│   ├── circuitbreaker.go
│   ├── ratelimit.go
│   └── metrics.go
├── internal/            # Internal error handling
│   └── errors.go
└── examples/            # Usage examples
    ├── basic/
    ├── advanced/
    └── middleware/
```

## API Reference

### Core Types

```go
// RetryFunc represents a function that can be retried
type RetryFunc func() error

// RetryFuncWithResult represents a function that returns a result and error
type RetryFuncWithResult[T any] func() (T, error)

// DelayStrategy defines how delays are calculated between retries
type DelayStrategy interface {
    NextDelay(attempt int, lastDelay time.Duration) time.Duration
    Reset()
}

// ErrorFilter determines if an error should trigger a retry
type ErrorFilter func(error) bool

// Hook represents a callback function for retry events
type Hook func(attempt int, err error, nextDelay time.Duration)

// Middleware interface for extensible patterns
type Middleware interface {
    Execute(ctx context.Context, fn func() error, next func(context.Context, func() error) error) error
}
```

### Main Functions

```go
// NewRetrier creates a new retrier with the given options
func NewRetrier(options ...RetryOption) *Retrier

// NewEnhancedRetrier creates a new enhanced retrier with middleware support
func NewEnhancedRetrier(options ...RetryOption) *EnhancedRetrier

// RetryWithResult executes the given function with retry logic and returns a result
func RetryWithResult[T any](ctx context.Context, r *Retrier, fn RetryFuncWithResult[T]) (T, error)

// RetryWithExponentialBackoff convenience function for exponential backoff
func RetryWithExponentialBackoff(ctx context.Context, fn RetryFunc, maxAttempts int) error

// RetryWithLogging convenience function with logging
func RetryWithLogging(ctx context.Context, fn RetryFunc, maxAttempts int, logger func(string, ...interface{})) error
```

## Running Tests

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run benchmarks
go test -bench=. ./...

# Run integration tests
go test -tags=integration ./...

# Run tests with coverage
go test -cover ./...
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Write tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

MIT License - see [LICENSE](LICENSE) file for details.
