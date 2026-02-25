package whoop

import (
	"net/http"
	"time"
)

// Option is a functional option for configuring the Client.
type Option func(*Client)

// WithHTTPClient sets the underlying HTTP client used for requests.
// If this is not provided, a default http.Client is used.
func WithHTTPClient(c *http.Client) Option {
	return func(client *Client) {
		client.httpClient = c
	}
}

// WithMaxRetries sets the maximum number of retries for 429 Too Many Requests responses.
// By default, the client will retry up to 3 times.
func WithMaxRetries(retries int) Option {
	return func(client *Client) {
		client.maxRetries = retries
	}
}

// WithBackoffBase sets the base duration for exponential backoff during retries.
// By default, this is 1 second.
func WithBackoffBase(base time.Duration) Option {
	return func(client *Client) {
		client.backoffBase = base
	}
}

// WithBackoffMax sets the maximum duration for exponential backoff during retries.
// By default, this is 60 seconds.
func WithBackoffMax(max time.Duration) Option {
	return func(client *Client) {
		client.backoffMax = max
	}
}

// WithToken sets the OAuth2 access token for authentication.
// This will automatically set the Authorization: Bearer <token> header on all requests.
func WithToken(token string) Option {
	return func(client *Client) {
		client.token = token
	}
}

// WithBaseURL overrides the default WHOOP API base URL.
// This is primarily useful for testing or connecting to a proxy.
func WithBaseURL(url string) Option {
	return func(client *Client) {
		client.baseURL = url
	}
}

// WithRateLimiting enables or disables client-side rate limiting.
// This is primarily used for testing and benchmarking.
func WithRateLimiting(enabled bool) Option {
	return func(client *Client) {
		client.rateLimiter.SetAutoLimiting(enabled)
	}
}
