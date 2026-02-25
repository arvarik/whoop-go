package whoop

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

func TestRateLimit_ExponentialBackoff(t *testing.T) {
	ts := newMockServer(t)
	defer ts.Close()

	// Short backoffs to keep the test rapid.
	// MaxRetries = 2
	// Base = 100ms
	// Max = 500ms
	client := newMockClient(ts,
		WithMaxRetries(2),
		WithBackoffBase(100*time.Millisecond),
		WithBackoffMax(500*time.Millisecond),
	)

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/429-generator", nil)

	start := time.Now()
	_, err := client.Do(context.Background(), req)
	duration := time.Since(start)

	if err == nil {
		t.Fatal("expected error from persistent 429, got nil")
	}

	var rateLimitErr *RateLimitError
	if !errors.As(err, &rateLimitErr) {
		t.Fatalf("expected RateLimitError, got %T: %v", err, err)
	}

	// We expect roughly:
	// Attempt 0 -> gets 429 -> backoff ~100ms
	// Attempt 1 -> gets 429 -> backoff ~200ms
	// Attempt 2 -> gets 429 -> aborts (max retries 2 reached)
	// Total wait time should be at minimum close to 0ms (due to jitter) but generally
	// > 0 and less than ~300ms. We'll simply verify the jitter calculation ran.

	if duration < 1*time.Millisecond {
		t.Errorf("expected backoff delay to take time, duration was essentially instantaneous: %v", duration)
	}
}

func TestCalculateBackoff_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		attempt int
		base    time.Duration
		max     time.Duration
	}{
		{name: "zero base", attempt: 0, base: 0, max: time.Minute},
		{name: "negative base", attempt: 0, base: -time.Second, max: time.Minute},
		{name: "negative max", attempt: 0, base: time.Second, max: -time.Minute},
		{name: "high attempt capped by max", attempt: 100, base: time.Second, max: 5 * time.Second},
		{name: "normal case", attempt: 2, base: time.Second, max: time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backoff := calculateBackoff(tt.attempt, tt.base, tt.max)
			if backoff < 0 {
				t.Errorf("backoff should never be negative, got %v", backoff)
			}

			// Determine effective max accounting for defaults
			effectiveMax := tt.max
			if effectiveMax <= 0 {
				effectiveMax = 60 * time.Second
			}
			if backoff > effectiveMax {
				t.Errorf("backoff %v exceeded max %v", backoff, effectiveMax)
			}
		})
	}
}

func TestCalculateBackoff_JitterDistribution(t *testing.T) {
	// Run multiple iterations to verify jitter produces varied results.
	seen := make(map[time.Duration]bool)
	for i := 0; i < 50; i++ {
		d := calculateBackoff(1, 100*time.Millisecond, time.Second)
		seen[d] = true
	}
	// With jitter, we should see more than one unique value in 50 runs.
	if len(seen) < 2 {
		t.Errorf("expected jitter to produce varied backoffs, got %d unique values", len(seen))
	}
}

var sink int64

func BenchmarkCalculateBackoff(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		var s time.Duration
		for pb.Next() {
			s = calculateBackoff(1, time.Second, 10*time.Second)
		}
		atomic.AddInt64(&sink, int64(s))
	})
}
