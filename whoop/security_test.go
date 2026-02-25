package whoop

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestMapHTTPError_BodyTruncation(t *testing.T) {
	t.Run("Large Body", func(t *testing.T) {
		longBody := strings.Repeat("A", 2000)
		resp := &http.Response{
			StatusCode: 500,
			Request:    &http.Request{URL: &url.URL{Path: "/test"}},
		}
		err := mapHTTPError(resp, []byte(longBody))

		var apiErr *APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("expected APIError, got %T", err)
		}

		// Should be truncated to 1000 + 3 ("...")
		expectedLen := 1003
		if len(apiErr.Message) != expectedLen {
			t.Errorf("expected message length %d, got %d", expectedLen, len(apiErr.Message))
		}
		if !strings.HasSuffix(apiErr.Message, "...") {
			t.Error("expected message to end with '...'")
		}
	})

	t.Run("Short Body", func(t *testing.T) {
		shortBody := "short error message"
		resp := &http.Response{
			StatusCode: 400,
			Request:    &http.Request{URL: &url.URL{Path: "/test"}},
		}
		err := mapHTTPError(resp, []byte(shortBody))

		var apiErr *APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("expected APIError, got %T", err)
		}

		if apiErr.Message != shortBody {
			t.Errorf("expected message %q, got %q", shortBody, apiErr.Message)
		}
	})
}
