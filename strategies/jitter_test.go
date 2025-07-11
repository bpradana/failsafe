package strategies

import (
	"testing"
	"time"
)

func TestJitterStrategy_NextDelay(t *testing.T) {
	baseStrategy := NewFixedDelay(100 * time.Millisecond)
	jitterFunc := func(delay time.Duration) time.Duration {
		return delay + 50*time.Millisecond // Add fixed jitter for testing
	}

	jitter := NewJitterStrategy(baseStrategy, jitterFunc)

	testCases := []struct {
		name      string
		attempt   int
		lastDelay time.Duration
		want      time.Duration
	}{
		{"first attempt", 1, 0, 150 * time.Millisecond},
		{"second attempt", 2, 100 * time.Millisecond, 150 * time.Millisecond},
		{"third attempt", 3, 100 * time.Millisecond, 150 * time.Millisecond},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := jitter.NextDelay(tc.attempt, tc.lastDelay)
			if got != tc.want {
				t.Errorf("NextDelay(%d, %v) = %v, want %v", tc.attempt, tc.lastDelay, got, tc.want)
			}
		})
	}
}

func TestJitterStrategy_Reset(t *testing.T) {
	baseStrategy := NewExponentialBackoff(100*time.Millisecond, 1*time.Second, 2.0)
	jitterFunc := func(delay time.Duration) time.Duration {
		return delay
	}

	jitter := NewJitterStrategy(baseStrategy, jitterFunc)

	// Use the strategy a few times
	jitter.NextDelay(1, 0)
	jitter.NextDelay(2, 100*time.Millisecond)

	// Reset and verify base strategy is reset
	jitter.Reset()
	got := jitter.NextDelay(1, 0)
	if got != 100*time.Millisecond {
		t.Errorf("NextDelay after reset = %v, want %v", got, 100*time.Millisecond)
	}
}

func TestJitterStrategy_ConcurrentAccess(t *testing.T) {
	baseStrategy := NewFixedDelay(100 * time.Millisecond)
	jitterFunc := func(delay time.Duration) time.Duration {
		return delay + 10*time.Millisecond
	}

	jitter := NewJitterStrategy(baseStrategy, jitterFunc)

	const numGoroutines = 10
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() { done <- true }()

			// Each goroutine performs several NextDelay calls
			for j := 1; j <= 5; j++ {
				jitter.NextDelay(j, time.Duration(j-1)*100*time.Millisecond)
			}
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

func TestUniformJitter(t *testing.T) {
	baseDelay := 100 * time.Millisecond

	// Test multiple times to ensure jitter is within expected range
	for i := 0; i < 10; i++ {
		jitteredDelay := UniformJitter(baseDelay)

		// Uniform jitter should add 0-100% of the base delay
		if jitteredDelay < baseDelay {
			t.Errorf("UniformJitter(%v) = %v, should be >= base delay", baseDelay, jitteredDelay)
		}
		if jitteredDelay > baseDelay*2 {
			t.Errorf("UniformJitter(%v) = %v, should be <= 2x base delay", baseDelay, jitteredDelay)
		}
	}
}

func TestExponentialJitter(t *testing.T) {
	baseDelay := 100 * time.Millisecond

	// Test multiple times to ensure jitter is within expected range
	for i := 0; i < 10; i++ {
		jitteredDelay := ExponentialJitter(baseDelay)

		// Exponential jitter should add 0-10% of the base delay
		if jitteredDelay < baseDelay {
			t.Errorf("ExponentialJitter(%v) = %v, should be >= base delay", baseDelay, jitteredDelay)
		}
		if jitteredDelay > time.Duration(float64(baseDelay)*1.1) {
			t.Errorf("ExponentialJitter(%v) = %v, should be <= 1.1x base delay", baseDelay, jitteredDelay)
		}
	}
}

func TestExponentialBackoffWithJitter(t *testing.T) {
	strategy := ExponentialBackoffWithJitter(100*time.Millisecond, 1*time.Second, 2.0)

	// Test that it returns a JitterStrategy
	jitterStrategy, ok := strategy.(*JitterStrategy)
	if !ok {
		t.Error("ExponentialBackoffWithJitter should return a JitterStrategy")
	}

	// Test that the base strategy is ExponentialBackoff
	_, ok = jitterStrategy.BaseStrategy.(*ExponentialBackoff)
	if !ok {
		t.Error("Base strategy should be ExponentialBackoff")
	}

	// Test that delays increase with jitter
	delay1 := strategy.NextDelay(1, 0)
	delay2 := strategy.NextDelay(2, delay1)
	delay3 := strategy.NextDelay(3, delay2)

	// First delay should be around 100ms (plus jitter)
	if delay1 < 100*time.Millisecond {
		t.Errorf("First delay %v should be >= 100ms", delay1)
	}
	if delay1 > 110*time.Millisecond {
		t.Errorf("First delay %v should be <= 110ms (100ms + 10%% jitter)", delay1)
	}

	// Delays should generally increase (accounting for jitter)
	if delay2 < 180*time.Millisecond { // ~200ms - some margin for jitter
		t.Errorf("Second delay %v should be around 200ms", delay2)
	}
	if delay3 < 360*time.Millisecond { // ~400ms - some margin for jitter
		t.Errorf("Third delay %v should be around 400ms", delay3)
	}
}

func TestJitterStrategy_WithExponentialBackoff(t *testing.T) {
	baseStrategy := NewExponentialBackoff(100*time.Millisecond, 1*time.Second, 2.0)
	jitterFunc := func(delay time.Duration) time.Duration {
		return delay + 25*time.Millisecond // Add fixed jitter for predictable testing
	}

	jitter := NewJitterStrategy(baseStrategy, jitterFunc)

	// Test sequence of delays
	delay1 := jitter.NextDelay(1, 0)      // 100ms + 25ms = 125ms
	delay2 := jitter.NextDelay(2, delay1) // 125ms * 2.0 + 25ms = 275ms
	delay3 := jitter.NextDelay(3, delay2) // 275ms * 2.0 + 25ms = 575ms

	expectedDelays := []time.Duration{
		125 * time.Millisecond,
		275 * time.Millisecond,
		575 * time.Millisecond,
	}

	delays := []time.Duration{delay1, delay2, delay3}

	for i, expected := range expectedDelays {
		if delays[i] != expected {
			t.Errorf("Delay %d: got %v, want %v", i+1, delays[i], expected)
		}
	}
}

func TestNewJitterStrategy(t *testing.T) {
	baseStrategy := NewFixedDelay(100 * time.Millisecond)
	jitterFunc := func(delay time.Duration) time.Duration {
		return delay * 2
	}

	jitter := NewJitterStrategy(baseStrategy, jitterFunc)

	if jitter.BaseStrategy != baseStrategy {
		t.Error("BaseStrategy not set correctly")
	}

	// Test that jitter function is applied
	result := jitter.NextDelay(1, 0)
	expected := 200 * time.Millisecond // 100ms * 2
	if result != expected {
		t.Errorf("NextDelay = %v, want %v", result, expected)
	}
}
