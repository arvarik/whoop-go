package whoop

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
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

func TestMapHTTPError(t *testing.T) {
	testURL, _ := url.Parse("https://api.whoop.com/test")

	tests := []struct {
		name       string
		statusCode int
		header     http.Header
		body       string
		wantErr    func(*testing.T, error)
	}{
		{
			name:       "401 Unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       "unauthorized",
			wantErr: func(t *testing.T, err error) {
				var authErr *AuthError
				if !errors.As(err, &authErr) {
					t.Fatalf("expected AuthError, got %T", err)
				}
				if authErr.StatusCode != 401 {
					t.Errorf("expected status 401, got %d", authErr.StatusCode)
				}
				// Verify it wraps APIError
				var apiErr *APIError
				if !errors.As(err, &apiErr) {
					t.Error("expected wrapped APIError")
				}
			},
		},
		{
			name:       "403 Forbidden",
			statusCode: http.StatusForbidden,
			body:       "forbidden",
			wantErr: func(t *testing.T, err error) {
				var authErr *AuthError
				if !errors.As(err, &authErr) {
					t.Fatalf("expected AuthError, got %T", err)
				}
			},
		},
		{
			name:       "429 Too Many Requests (valid header)",
			statusCode: http.StatusTooManyRequests,
			header:     http.Header{"Retry-After": []string{"30"}},
			body:       "too many",
			wantErr: func(t *testing.T, err error) {
				var rlErr *RateLimitError
				if !errors.As(err, &rlErr) {
					t.Fatalf("expected RateLimitError, got %T", err)
				}
				if rlErr.RetryAfter != 30 {
					t.Errorf("expected RetryAfter 30, got %d", rlErr.RetryAfter)
				}
			},
		},
		{
			name:       "429 Too Many Requests (missing header)",
			statusCode: http.StatusTooManyRequests,
			body:       "too many",
			wantErr: func(t *testing.T, err error) {
				var rlErr *RateLimitError
				if !errors.As(err, &rlErr) {
					t.Fatalf("expected RateLimitError, got %T", err)
				}
				if rlErr.RetryAfter != 0 {
					t.Errorf("expected RetryAfter 0, got %d", rlErr.RetryAfter)
				}
			},
		},
		{
			name:       "429 Too Many Requests (invalid header)",
			statusCode: http.StatusTooManyRequests,
			header:     http.Header{"Retry-After": []string{"invalid"}},
			body:       "too many",
			wantErr: func(t *testing.T, err error) {
				var rlErr *RateLimitError
				if !errors.As(err, &rlErr) {
					t.Fatalf("expected RateLimitError, got %T", err)
				}
				if rlErr.RetryAfter != 0 {
					t.Errorf("expected RetryAfter 0, got %d", rlErr.RetryAfter)
				}
			},
		},
		{
			name:       "404 Not Found",
			statusCode: http.StatusNotFound,
			body:       "not found",
			wantErr: func(t *testing.T, err error) {
				var apiErr *APIError
				if !errors.As(err, &apiErr) {
					t.Fatalf("expected APIError, got %T", err)
				}
				if apiErr.StatusCode != 404 {
					t.Errorf("expected status 404, got %d", apiErr.StatusCode)
				}
				if apiErr.Message != "not found" {
					t.Errorf("expected message 'not found', got %q", apiErr.Message)
				}
			},
		},
		{
			name:       "500 Internal Server Error",
			statusCode: http.StatusInternalServerError,
			body:       "internal error",
			wantErr: func(t *testing.T, err error) {
				var apiErr *APIError
				if !errors.As(err, &apiErr) {
					t.Fatalf("expected APIError, got %T", err)
				}
				if apiErr.StatusCode != 500 {
					t.Errorf("expected status 500, got %d", apiErr.StatusCode)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				StatusCode: tt.statusCode,
				Header:     tt.header,
				Request:    &http.Request{URL: testURL},
			}
			if resp.Header == nil {
				resp.Header = make(http.Header)
			}

			err := mapHTTPError(resp, []byte(tt.body))
			tt.wantErr(t, err)
		})
	}
}
