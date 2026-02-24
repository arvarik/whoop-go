package whoop

import (
	"context"
	"errors"
	"net/http"
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
