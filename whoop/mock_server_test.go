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

	// 6. User - BasicProfile Mock
	mux.HandleFunc("/user/profile/basic", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"user_id": 999,
			"email": "athlete@example.com",
			"first_name": "Jane",
			"last_name": "Doe"
		}`))
	})

	// 7. User - BodyMeasurement Mock
	mux.HandleFunc("/user/measurement/body", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"height_meter": 1.75,
			"weight_kilogram": 70.5,
			"max_heart_rate": 195
		}`))
	})

	// 8. Sleep - GetByID Mock
	mux.HandleFunc("/activity/sleep/789", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": 789,
			"user_id": 999,
			"created_at": "2026-02-24T06:00:00Z",
			"updated_at": "2026-02-24T07:00:00Z",
			"start": "2026-02-23T22:00:00Z",
			"end": "2026-02-24T06:00:00Z",
			"timezone_offset": "-08:00",
			"nap": false,
			"score": {
				"stage_summary": {
					"total_in_bed_time_milli": 28800000,
					"total_awake_time_milli": 3600000,
					"total_no_data_time_milli": 0,
					"total_light_sleep_time_milli": 10800000,
					"total_slow_wave_sleep_time_milli": 7200000,
					"total_rem_sleep_time_milli": 7200000,
					"sleep_cycle_count": 4,
					"disturbance_count": 2
				},
				"sleep_needed": {
					"baseline_milli": 28800000,
					"need_from_sleep_debt_milli": 1800000,
					"need_from_recent_strain_milli": 900000,
					"need_from_recent_nap_milli": 0
				},
				"respiratory_rate": 15.5,
				"sleep_performance_percentage": 95.0,
				"sleep_consistency_percentage": 88.0,
				"sleep_efficiency_percentage": 92.0
			}
		}`))
	})

	// 9. Sleep - List Mock (Paginated)
	mux.HandleFunc("/activity/sleep", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		token := r.URL.Query().Get("nextToken")
		if token == "" {
			_, _ = w.Write([]byte(`{
				"records": [{"id": 789, "user_id": 999, "nap": false}],
				"next_token": "sleep-p2"
			}`))
		} else if token == "sleep-p2" {
			_, _ = w.Write([]byte(`{
				"records": [],
				"next_token": ""
			}`))
		}
	})

	// 10. Recovery - GetByID Mock
	mux.HandleFunc("/cycle/123/recovery", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"cycle_id": 123,
			"sleep_id": 789,
			"user_id": 999,
			"created_at": "2026-02-24T06:00:00Z",
			"updated_at": "2026-02-24T07:00:00Z",
			"score": {
				"user_calibrating": false,
				"recovery_score": 85.5,
				"resting_heart_rate": 52.0,
				"hrv_rmssd_milli": 65.3,
				"spo2_percentage": 97.0,
				"skin_temp_celsius": 33.5
			}
		}`))
	})

	// 11. Recovery - List Mock (Paginated)
	mux.HandleFunc("/recovery", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		token := r.URL.Query().Get("nextToken")
		if token == "" {
			_, _ = w.Write([]byte(`{
				"records": [{"cycle_id": 123, "sleep_id": 789, "user_id": 999}],
				"next_token": "rec-p2"
			}`))
		} else if token == "rec-p2" {
			_, _ = w.Write([]byte(`{
				"records": [],
				"next_token": ""
			}`))
		}
	})

	// 12. Workout - GetByID Mock
	mux.HandleFunc("/activity/workout/456", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
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
		}`))
	})

	return httptest.NewServer(mux)
}

// newMockClient builds a generic unauthenticated WHOOP client
// connected directly to the mockServer base URL.
func newMockClient(ts *httptest.Server, opts ...Option) *Client {
	defaultOpts := []Option{
		WithBaseURL(ts.URL),
		// Shorter backoff logic so tests don't permanently stall
		WithMaxRetries(3),
	}
	defaultOpts = append(defaultOpts, opts...)
	return NewClient(defaultOpts...)
}
