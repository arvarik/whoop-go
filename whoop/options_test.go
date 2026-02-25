package whoop

import (
	"net/http"
	"testing"
	"time"
)

func TestClient_Defaults(t *testing.T) {
	client := NewClient()

	if client.baseURL != defaultBaseURL {
		t.Errorf("expected baseURL %q, got %q", defaultBaseURL, client.baseURL)
	}

	if client.maxRetries != 3 {
		t.Errorf("expected maxRetries %d, got %d", 3, client.maxRetries)
	}

	if client.backoffBase != 1*time.Second {
		t.Errorf("expected backoffBase %v, got %v", 1*time.Second, client.backoffBase)
	}

	if client.backoffMax != 60*time.Second {
		t.Errorf("expected backoffMax %v, got %v", 60*time.Second, client.backoffMax)
	}

	if client.httpClient == nil {
		t.Fatal("expected httpClient to be initialized")
	}

	if client.httpClient.Timeout != 30*time.Second {
		t.Errorf("expected httpClient timeout %v, got %v", 30*time.Second, client.httpClient.Timeout)
	}

	if client.rateLimiter == nil {
		t.Fatal("expected rateLimiter to be initialized")
	}

	if !client.rateLimiter.isAutoLimiting.Load() {
		t.Error("expected rateLimiter auto limiting to be enabled by default")
	}
}

func TestClient_Options(t *testing.T) {
	customHTTPClient := &http.Client{Timeout: 10 * time.Second}
	customBaseURL := "https://api.example.com"
	customToken := "my-secret-token"

	client := NewClient(
		WithHTTPClient(customHTTPClient),
		WithMaxRetries(5),
		WithBackoffBase(500*time.Millisecond),
		WithBackoffMax(10*time.Second),
		WithToken(customToken),
		WithBaseURL(customBaseURL),
		WithRateLimiting(false),
	)

	if client.httpClient != customHTTPClient {
		t.Errorf("expected custom httpClient, got different instance")
	}

	if client.maxRetries != 5 {
		t.Errorf("expected maxRetries %d, got %d", 5, client.maxRetries)
	}

	if client.backoffBase != 500*time.Millisecond {
		t.Errorf("expected backoffBase %v, got %v", 500*time.Millisecond, client.backoffBase)
	}

	if client.backoffMax != 10*time.Second {
		t.Errorf("expected backoffMax %v, got %v", 10*time.Second, client.backoffMax)
	}

	if client.token != customToken {
		t.Errorf("expected token %q, got %q", customToken, client.token)
	}

	if client.baseURL != customBaseURL {
		t.Errorf("expected baseURL %q, got %q", customBaseURL, client.baseURL)
	}

	if client.rateLimiter.isAutoLimiting.Load() {
		t.Error("expected rateLimiter auto limiting to be disabled")
	}
}
