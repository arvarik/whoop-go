package whoop

import (
	"fmt"
	"net/http"
)

// APIError represents an error returned by the WHOOP API.
type APIError struct {
	StatusCode int
	Message    string
	URL        string
	Err        error // Underlying error, if any
}

// Error implements the error interface.
func (e *APIError) Error() string {
	msg := fmt.Sprintf("whoop api error: %d - %s at %s", e.StatusCode, e.Message, e.URL)
	if e.Err != nil {
		msg += fmt.Sprintf(" (%v)", e.Err)
	}
	return msg
}

// Unwrap implements errors.Unwrap so the underlying error can be extracted.
func (e *APIError) Unwrap() error {
	return e.Err
}

// RateLimitError represents an error indicating that the client is rate-limited.
// It can occur locally before the request is made or as a response from the API.
type RateLimitError struct {
	RetryAfter int // Suggested retry after duration in seconds, if provided by the API
	Err        error
}

// Error implements the error interface.
func (e *RateLimitError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("whoop rate limit exceeded: retry after %d seconds", e.RetryAfter)
	}
	if e.Err != nil {
		return fmt.Sprintf("whoop rate limit exceeded: %v", e.Err)
	}
	return "whoop rate limit exceeded"
}

// Unwrap implements errors.Unwrap.
func (e *RateLimitError) Unwrap() error {
	return e.Err
}

// AuthError represents an authentication or authorization failure (401, 403).
type AuthError struct {
	StatusCode int
	Message    string
	Err        error
}

// Error implements the error interface.
func (e *AuthError) Error() string {
	msg := fmt.Sprintf("whoop auth error (%d): %s", e.StatusCode, e.Message)
	if e.Err != nil {
		msg += fmt.Sprintf(" - %v", e.Err)
	}
	return msg
}

// Unwrap implements errors.Unwrap.
func (e *AuthError) Unwrap() error {
	return e.Err
}

// mapHTTPError is a helper to convert an unsuccessful HTTP response to an appropriate custom error.
func mapHTTPError(resp *http.Response, body []byte) error {
	baseErr := &APIError{
		StatusCode: resp.StatusCode,
		Message:    string(body),
		URL:        resp.Request.URL.String(),
	}

	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return &AuthError{
			StatusCode: resp.StatusCode,
			Message:    "authentication failed or forbidden",
			Err:        baseErr,
		}
	case http.StatusTooManyRequests:
		return &RateLimitError{
			Err: baseErr,
		}
	default:
		return baseErr
	}
}
