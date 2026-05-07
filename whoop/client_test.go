package whoop

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestClientStringRedaction(t *testing.T) {
	token := "my-secret-token"
	client := &Client{
		token:   token,
		baseURL: "https://example.com",
	}

	formats := []string{"%+v", "%#v", "%v", "%s"}

	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			output := fmt.Sprintf(format, client)

			if strings.Contains(output, token) {
				t.Errorf("Security check failed: Token leaked in %s output: %s", format, output)
			}

			if !strings.Contains(output, "token:<REDACTED>") {
				t.Errorf("Expected output to contain redacted token placeholder for %s, got: %s", format, output)
			}

			if !strings.Contains(output, "baseURL:https://example.com") {
				t.Errorf("Expected output to contain baseURL for %s, got: %s", format, output)
			}
		})
	}
}

func TestClient_Get(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			if r.URL.Path != "/test" {
				t.Errorf("expected path /test, got %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"foo": "bar"}`))
		}))
		defer ts.Close()

		client := NewClient(WithBaseURL(ts.URL))
		var result struct {
			Foo string `json:"foo"`
		}
		err := client.Get(context.Background(), "/test", &result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Foo != "bar" {
			t.Errorf("expected bar, got %s", result.Foo)
		}
	})

	t.Run("ServerError", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error": "not found"}`))
		}))
		defer ts.Close()

		client := NewClient(WithBaseURL(ts.URL))
		err := client.Get(context.Background(), "/not-found", nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("MalformedJSON", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{invalid json}`))
		}))
		defer ts.Close()

		client := NewClient(WithBaseURL(ts.URL))
		var result map[string]any
		err := client.Get(context.Background(), "/malformed", &result)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("NilTarget", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"foo": "bar"}`))
		}))
		defer ts.Close()

		client := NewClient(WithBaseURL(ts.URL))
		err := client.Get(context.Background(), "/nil", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
