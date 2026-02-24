package whoop

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestClient_Do_ContextCancellation(t *testing.T) {
	ts := newMockServer(t)
	defer ts.Close()

	client := newMockClient(ts)

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/delay", nil)

	// Context with immediate 1 millisecond execution cancellation
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := client.Do(ctx, req)
	duration := time.Since(start)

	if err == nil {
		t.Fatal("expected context deadline exceeded error, got nil")
	}

	// Make sure the request correctly aborted and returned quickly
	if duration > 100*time.Millisecond {
		t.Errorf("request took too long to abort on cancelled context: %v", duration)
	}
}

func TestClient_Do_CustomErrorMapping(t *testing.T) {
	ts := newMockServer(t)
	defer ts.Close()

	client := newMockClient(ts)

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/403-generator", nil)
	_, err := client.Do(context.Background(), req)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Verify the AuthError structure marshalled cleanly from mapHTTPError
	if authErr, ok := err.(*AuthError); ok {
		if authErr.StatusCode != http.StatusForbidden {
			t.Errorf("expected status 403, got %d", authErr.StatusCode)
		}
	} else {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}
