package strategies

import (
	"testing"
	"time"
)

func TestFixedDelay_NextDelay(t *testing.T) {
	delay := NewFixedDelay(100 * time.Millisecond)

	testCases := []struct {
		name      string
		attempt   int
		lastDelay time.Duration
		want      time.Duration
	}{
		{"first attempt", 1, 0, 100 * time.Millisecond},
		{"second attempt", 2, 100 * time.Millisecond, 100 * time.Millisecond},
		{"fifth attempt", 5, 100 * time.Millisecond, 100 * time.Millisecond},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := delay.NextDelay(tc.attempt, tc.lastDelay); got != tc.want {
				t.Errorf("NextDelay(%d, %v) = %v, want %v", tc.attempt, tc.lastDelay, got, tc.want)
			}
		})
	}
}

func TestFixedDelay_Reset(t *testing.T) {
	delay := NewFixedDelay(100 * time.Millisecond)
	delay.Reset()

	if got := delay.NextDelay(1, 0); got != 100*time.Millisecond {
		t.Errorf("NextDelay after reset = %v, want %v", got, 100*time.Millisecond)
	}
}

func TestExponentialBackoff_NextDelay(t *testing.T) {
	backoff := NewExponentialBackoff(100*time.Millisecond, 1*time.Second, 2.0)

	testCases := []struct {
		name      string
		attempt   int
		lastDelay time.Duration
		want      time.Duration
	}{
		{"first attempt", 1, 0, 100 * time.Millisecond},
		{"second attempt", 2, 100 * time.Millisecond, 200 * time.Millisecond},
		{"third attempt", 3, 200 * time.Millisecond, 400 * time.Millisecond},
		{"max delay reached", 4, 400 * time.Millisecond, 800 * time.Millisecond},
		{"exceeds max delay", 5, 800 * time.Millisecond, 1 * time.Second},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := backoff.NextDelay(tc.attempt, tc.lastDelay)
			if got != tc.want {
				t.Errorf("NextDelay(%d, %v) = %v, want %v", tc.attempt, tc.lastDelay, got, tc.want)
			}
		})
	}
}

func TestExponentialBackoff_MaxDelay(t *testing.T) {
	backoff := NewExponentialBackoff(100*time.Millisecond, 500*time.Millisecond, 2.0)

	// Test that delay is capped at MaxDelay
	delay1 := backoff.NextDelay(1, 0)      // 100ms
	delay2 := backoff.NextDelay(2, delay1) // 200ms
	delay3 := backoff.NextDelay(3, delay2) // 400ms
	delay4 := backoff.NextDelay(4, delay3) // should be capped at 500ms
	delay5 := backoff.NextDelay(5, delay4) // should be capped at 500ms

	expectedDelays := []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
		400 * time.Millisecond,
		500 * time.Millisecond,
		500 * time.Millisecond,
	}

	delays := []time.Duration{delay1, delay2, delay3, delay4, delay5}

	for i, expected := range expectedDelays {
		if delays[i] != expected {
			t.Errorf("Delay %d: got %v, want %v", i+1, delays[i], expected)
		}
	}
}

func TestExponentialBackoff_Reset(t *testing.T) {
	backoff := NewExponentialBackoff(100*time.Millisecond, 1*time.Second, 2.0)

	// Use the backoff a few times
	backoff.NextDelay(1, 0)
	backoff.NextDelay(2, 100*time.Millisecond)

	// Reset and test that it starts from the beginning
	backoff.Reset()
	if got := backoff.NextDelay(1, 0); got != 100*time.Millisecond {
		t.Errorf("NextDelay after reset = %v, want %v", got, 100*time.Millisecond)
	}
}

func TestExponentialBackoff_ConcurrentAccess(t *testing.T) {
	backoff := NewExponentialBackoff(100*time.Millisecond, 1*time.Second, 2.0)

	const numGoroutines = 10
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() { done <- true }()

			// Each goroutine performs several NextDelay calls
			for j := 1; j <= 5; j++ {
				backoff.NextDelay(j, time.Duration(j-1)*100*time.Millisecond)
			}
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

func TestConstructors(t *testing.T) {
	// Test NewFixedDelay
	fixed := NewFixedDelay(500 * time.Millisecond)
	if fixed.Delay != 500*time.Millisecond {
		t.Errorf("NewFixedDelay: got %v, want %v", fixed.Delay, 500*time.Millisecond)
	}

	// Test NewExponentialBackoff
	backoff := NewExponentialBackoff(100*time.Millisecond, 1*time.Second, 2.5)
	if backoff.InitialDelay != 100*time.Millisecond {
		t.Errorf("NewExponentialBackoff InitialDelay: got %v, want %v", backoff.InitialDelay, 100*time.Millisecond)
	}
	if backoff.MaxDelay != 1*time.Second {
		t.Errorf("NewExponentialBackoff MaxDelay: got %v, want %v", backoff.MaxDelay, 1*time.Second)
	}
	if backoff.Multiplier != 2.5 {
		t.Errorf("NewExponentialBackoff Multiplier: got %v, want %v", backoff.Multiplier, 2.5)
	}
}

func TestExponentialBackoff_EdgeCases(t *testing.T) {
	testCases := []struct {
		name       string
		initial    time.Duration
		max        time.Duration
		multiplier float64
		attempt    int
		lastDelay  time.Duration
		want       time.Duration
	}{
		{
			name:       "zero initial delay",
			initial:    0,
			max:        1 * time.Second,
			multiplier: 2.0,
			attempt:    1,
			lastDelay:  0,
			want:       0,
		},
		{
			name:       "multiplier of 1.0",
			initial:    100 * time.Millisecond,
			max:        1 * time.Second,
			multiplier: 1.0,
			attempt:    3,
			lastDelay:  100 * time.Millisecond,
			want:       100 * time.Millisecond,
		},
		{
			name:       "very small multiplier",
			initial:    100 * time.Millisecond,
			max:        1 * time.Second,
			multiplier: 1.1,
			attempt:    2,
			lastDelay:  100 * time.Millisecond,
			want:       110 * time.Millisecond,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			backoff := NewExponentialBackoff(tc.initial, tc.max, tc.multiplier)
			got := backoff.NextDelay(tc.attempt, tc.lastDelay)
			if got != tc.want {
				t.Errorf("NextDelay(%d, %v) = %v, want %v", tc.attempt, tc.lastDelay, got, tc.want)
			}
		})
	}
}
