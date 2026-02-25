package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/arvarik/whoop-go/whoop"
)

// MockTransport simulates a slow API response
type MockTransport struct {
	Delay time.Duration
}

func (m *MockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	time.Sleep(m.Delay)
	// Return a dummy workout response
	resp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(`{"id": 123, "score": {"strain": 10.5}}`)),
		Header:     make(http.Header),
	}
	return resp, nil
}

// webhookHandlerOld replicates the original unbounded goroutine behavior
func webhookHandlerOld(client *whoop.Client, webhookSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		event, err := whoop.ParseWebhook(r, webhookSecret)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		if event.Type == "workout.updated" {
			go processWorkout(client, event.ID)
		}
	}
}

func BenchmarkWebhookConcurrency(b *testing.B) {
	// Setup client with slow transport
	mockClient := &http.Client{
		Transport: &MockTransport{Delay: 50 * time.Millisecond}, // Slow enough to accumulate
	}
	client := whoop.NewClient(whoop.WithHTTPClient(mockClient), whoop.WithRateLimiting(false))
	secret := "secret"

	// Create request payload
	payload := `{"type": "workout.updated", "id": 123, "user_id": 456, "trace_id": "abc"}`

	// Compute signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	b.Run("Unbounded", func(b *testing.B) {
		// Create handler
		handler := webhookHandlerOld(client, secret)
		server := httptest.NewServer(handler)
		defer server.Close()

		initialGoroutines := runtime.NumGoroutine()

		// Run parallel requests
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				req, _ := http.NewRequest("POST", server.URL, strings.NewReader(payload))
				req.Header.Set("X-Whoop-Signature", sig)
				resp, err := http.DefaultClient.Do(req)
				if err == nil {
					resp.Body.Close()
				}
			}
		})

		// Give a tiny bit of time for goroutines to spawn if needed, but they are spawned before handler returns
		time.Sleep(1 * time.Millisecond)

		finalGoroutines := runtime.NumGoroutine()
		b.Logf("Unbounded: Goroutines start=%d, end=%d, delta=%d", initialGoroutines, finalGoroutines, finalGoroutines-initialGoroutines)
	})

	b.Run("Bounded", func(b *testing.B) {
		jobQueue := make(chan int, 100)
		for i := 0; i < 5; i++ {
			go worker(client, jobQueue)
		}

		handler := webhookHandler(client, secret, jobQueue)
		server := httptest.NewServer(handler)
		defer server.Close()

		initialGoroutines := runtime.NumGoroutine()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				req, _ := http.NewRequest("POST", server.URL, strings.NewReader(payload))
				req.Header.Set("X-Whoop-Signature", sig)
				resp, err := http.DefaultClient.Do(req)
				if err == nil {
					resp.Body.Close()
				}
			}
		})

		time.Sleep(1 * time.Millisecond)

		finalGoroutines := runtime.NumGoroutine()
		b.Logf("Bounded: Goroutines start=%d, end=%d, delta=%d", initialGoroutines, finalGoroutines, finalGoroutines-initialGoroutines)
	})
}
