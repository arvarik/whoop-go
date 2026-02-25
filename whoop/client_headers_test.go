package whoop

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_Do_Headers(t *testing.T) {
	testCases := []struct {
		name              string
		token             string
		method            string
		customHeaders     map[string]string
		expectedHeaders   map[string]string
		unexpectedHeaders []string
	}{
		{
			name:   "With Token",
			token:  "test-token",
			method: http.MethodGet,
			expectedHeaders: map[string]string{
				"Authorization": "Bearer test-token",
			},
		},
		{
			name:   "Without Token",
			token:  "",
			method: http.MethodGet,
			unexpectedHeaders: []string{
				"Authorization",
			},
		},
		{
			name:   "Standard Headers",
			token:  "",
			method: http.MethodGet,
			expectedHeaders: map[string]string{
				"Accept":     "application/json",
				"User-Agent": userAgent,
			},
		},
		{
			name:   "Content-Type Default (POST)",
			token:  "",
			method: http.MethodPost,
			expectedHeaders: map[string]string{
				"Content-Type": "application/json",
			},
		},
		{
			name:   "Content-Type Default (GET)",
			token:  "",
			method: http.MethodGet,
			unexpectedHeaders: []string{
				"Content-Type",
			},
		},
		{
			name:   "Custom Content-Type",
			token:  "",
			method: http.MethodPost,
			customHeaders: map[string]string{
				"Content-Type": "application/xml",
			},
			expectedHeaders: map[string]string{
				"Content-Type": "application/xml",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock server
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Validate expected headers
				for key, expectedValue := range tc.expectedHeaders {
					actualValue := r.Header.Get(key)
					if actualValue != expectedValue {
						t.Errorf("header %s: expected %q, got %q", key, expectedValue, actualValue)
					}
				}

				// Validate unexpected headers
				for _, key := range tc.unexpectedHeaders {
					if value := r.Header.Get(key); value != "" {
						t.Errorf("header %s: expected empty, got %q", key, value)
					}
				}

				w.WriteHeader(http.StatusOK)
			}))
			defer ts.Close()

			// Configure client
			opts := []Option{
				WithBaseURL(ts.URL),
			}
			if tc.token != "" {
				opts = append(opts, WithToken(tc.token))
			}

			client := NewClient(opts...)

			// Create request
			req, err := http.NewRequest(tc.method, ts.URL, nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			// Add custom headers
			for k, v := range tc.customHeaders {
				req.Header.Set(k, v)
			}

			// Execute request
			_, err = client.Do(context.Background(), req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
