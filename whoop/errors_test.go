package whoop

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestAPIError_Error(t *testing.T) {
	err := &APIError{
		StatusCode: 500,
		Message:    "internal failure",
		URL:        "https://api.whoop.com/v1/cycle",
	}

	got := err.Error()
	if !strings.Contains(got, "500") {
		t.Errorf("expected error to contain status code 500, got: %s", got)
	}
	if !strings.Contains(got, "internal failure") {
		t.Errorf("expected error to contain message, got: %s", got)
	}
	if !strings.Contains(got, "api.whoop.com") {
		t.Errorf("expected error to contain URL, got: %s", got)
	}
}

func TestAPIError_Error_WithWrapped(t *testing.T) {
	inner := fmt.Errorf("connection refused")
	err := &APIError{
		StatusCode: 502,
		Message:    "bad gateway",
		URL:        "https://api.whoop.com/v1/cycle",
		Err:        inner,
	}

	got := err.Error()
	if !strings.Contains(got, "connection refused") {
		t.Errorf("expected error to contain wrapped error, got: %s", got)
	}
}

func TestAPIError_Unwrap(t *testing.T) {
	inner := fmt.Errorf("original error")
	err := &APIError{Err: inner}

	if !errors.Is(err, inner) {
		t.Error("expected errors.Is to find inner error")
	}
}

func TestRateLimitError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      RateLimitError
		contains string
	}{
		{
			name:     "with retry-after",
			err:      RateLimitError{RetryAfter: 30},
			contains: "retry after 30 seconds",
		},
		{
			name:     "with wrapped error",
			err:      RateLimitError{Err: fmt.Errorf("underlying")},
			contains: "underlying",
		},
		{
			name:     "bare",
			err:      RateLimitError{},
			contains: "rate limit exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if !strings.Contains(got, tt.contains) {
				t.Errorf("expected error to contain %q, got: %s", tt.contains, got)
			}
		})
	}
}

func TestRateLimitError_Unwrap(t *testing.T) {
	inner := fmt.Errorf("api error")
	err := &RateLimitError{Err: inner}

	if !errors.Is(err, inner) {
		t.Error("expected errors.Is to find inner error")
	}
}

func TestAuthError_Error(t *testing.T) {
	err := &AuthError{
		StatusCode: 401,
		Message:    "token expired",
	}

	got := err.Error()
	if !strings.Contains(got, "401") {
		t.Errorf("expected error to contain 401, got: %s", got)
	}
	if !strings.Contains(got, "token expired") {
		t.Errorf("expected error to contain message, got: %s", got)
	}
}

func TestAuthError_Unwrap(t *testing.T) {
	inner := fmt.Errorf("original")
	err := &AuthError{Err: inner}

	if !errors.Is(err, inner) {
		t.Error("expected errors.Is to find inner error")
	}
}

func TestErrorsAs_TypeAssertions(t *testing.T) {
	// Verify errors.As works correctly for each custom error type.
	apiErr := &APIError{StatusCode: 404, Message: "not found", URL: "/test"}
	authErr := &AuthError{StatusCode: 403, Message: "forbidden", Err: apiErr}
	rlErr := &RateLimitError{RetryAfter: 10, Err: apiErr}

	var target *APIError
	if !errors.As(authErr, &target) {
		t.Error("expected errors.As to find APIError wrapped in AuthError")
	}

	var target2 *APIError
	if !errors.As(rlErr, &target2) {
		t.Error("expected errors.As to find APIError wrapped in RateLimitError")
	}
}
