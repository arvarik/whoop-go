package whoop

import (
	"context"
	"net/http"
	"testing"
)

// TestClient_Do_UnsafeHeaderModification verifies that the original request's headers
// are not modified by the client.Do method.
func TestClient_Do_UnsafeHeaderModification(t *testing.T) {
	// Create a dummy request with a custom header
	req, err := http.NewRequest("GET", "https://example.com", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("X-Custom-Header", "original-value")

	// Create a client
	client := NewClient(WithToken("test-token"))

	// Create a context
	ctx := context.Background()

	// Use a mock transport to avoid actual network calls
	mockTransport := &safetyCheckTransport{}
	client.httpClient.Transport = mockTransport

	// Execute the request
	_, _ = client.Do(ctx, req)

	// Check if "Authorization" header is present in the original request
	if req.Header.Get("Authorization") != "" {
		t.Errorf("Original request header was modified! Authorization: %s", req.Header.Get("Authorization"))
	}

	if req.Header.Get("User-Agent") != "" {
		t.Errorf("Original request header was modified! User-Agent: %s", req.Header.Get("User-Agent"))
	}
}

type safetyCheckTransport struct{}

func (m *safetyCheckTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: 200,
		Body:       http.NoBody,
	}, nil
}
