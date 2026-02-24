package whoop

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// newMockServer creates an httptest.Server configured to respond dynamically
// to specific WHOOP API routes with literal mock JSON payloads.
func newMockServer(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	// 1. Cycle - GetByID Mock
	mux.HandleFunc("/cycle/123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": 123,
			"user_id": 999,
			"created_at": "2026-02-24T12:00:00Z",
			"updated_at": "2026-02-24T13:00:00Z",
			"start": "2026-02-24T05:00:00Z",
			"end": "2026-02-24T10:00:00Z",
			"timezone_offset": "-08:00",
			"score": {
				"strain": 12.4,
				"kilojoule": 2048.5,
				"average_heart_rate": 65,
				"max_heart_rate": 185
			}
		}`))
	})

	// 2. Workout - List Mock (Paginated)
	mux.HandleFunc("/activity/workout", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		token := r.URL.Query().Get("nextToken")
		if token == "" {
			_, _ = w.Write([]byte(`{
				"records": [
					{
						"id": 456,
						"user_id": 999,
						"created_at": "2026-02-24T14:00:00Z",
						"updated_at": "2026-02-24T15:00:00Z",
						"start": "2026-02-24T14:00:00Z",
						"end": "2026-02-24T15:00:00Z",
						"timezone_offset": "-08:00",
						"sport_id": 1,
						"score": {
							"strain": 14.2,
							"average_heart_rate": 150,
							"max_heart_rate": 190,
							"kilojoule": 700.5,
							"percent_recorded": 99.9,
							"distance_meter": 5000.0,
							"altitude_gain_meter": 100.0,
							"altitude_change_meter": 10.0,
							"zone_duration": {
								"zone_zero_milli": 1000,
								"zone_one_milli": 2000,
								"zone_two_milli": 3000,
								"zone_three_milli": 4000,
								"zone_four_milli": 5000,
								"zone_five_milli": 6000
							}
						}
					}
				],
				"next_token": "page2"
			}`))
		} else if token == "page2" {
			_, _ = w.Write([]byte(`{
				"records": [],
				"next_token": ""
			}`))
		} else {
			t.Fatalf("unexpected token requested: %s", token)
		}
	})

	// 3. Rate Limit Explicit Mock (always 429)
	mux.HandleFunc("/429-generator", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error": "Too Many Requests"}`))
	})

	// 4. Broken Endpoint Mock (Auth Error)
	mux.HandleFunc("/403-generator", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error": "Forbidden"}`))
	})

	// 5. Context Cancellation Delay Mock
	mux.HandleFunc("/delay", func(w http.ResponseWriter, r *http.Request) {
		// Used to simulate contexts failing during network reads
		// Select blocks until handler context is canceled
		<-r.Context().Done()
	})

	return httptest.NewServer(mux)
}

// newMockClient builds a generic unauthenticated WHOOP client
// connected directly to the `mockServer` base URL.
func newMockClient(ts *httptest.Server, opts ...Option) *Client {
	baseURL := ts.URL
	defaultOpts := []Option{
		// Shorter backoff logic so tests don't permanently stall
		WithMaxRetries(3),
	}
	defaultOpts = append(defaultOpts, opts...)

	client := NewClient(defaultOpts...)

	// Override private base url post-instantiation
	client.baseURL = baseURL

	return client
}
