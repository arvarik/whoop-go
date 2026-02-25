package whoop

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const (
	defaultBaseURL = "https://api.prod.whoop.com/developer/v1"

	// Version is the semantic version of this library.
	Version = "0.2.0"

	userAgent = "whoop-go/" + Version
)

// Client is the core WHOOP API client.
type Client struct {
	httpClient *http.Client
	baseURL    string
	token      string

	maxRetries  int
	backoffBase time.Duration
	backoffMax  time.Duration

	rateLimiter *rateLimiter

	// Services used for communicating with the WHOOP API endpoints.
	User     *UserService
	Cycle    *CycleService
	Sleep    *SleepService
	Workout  *WorkoutService
	Recovery *RecoveryService
}

// NewClient creates a new WHOOP API client with the given options.
func NewClient(opts ...Option) *Client {
	c := &Client{
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		baseURL:     defaultBaseURL,
		maxRetries:  3,
		backoffBase: 1 * time.Second,
		backoffMax:  60 * time.Second,
		rateLimiter: newRateLimiter(),
	}

	for _, opt := range opts {
		opt(c)
	}

	c.User = &UserService{client: c}
	c.Cycle = &CycleService{client: c}
	c.Sleep = &SleepService{client: c}
	c.Workout = &WorkoutService{client: c}
	c.Recovery = &RecoveryService{client: c}

	return c
}

// Do executes an HTTP request with context, authentication, rate limiting,
// and automatic retries on 429 Too Many Requests.
func (c *Client) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	// Ensure the request has the provided context attached.
	req = req.WithContext(ctx)

	// Inject authentication header if available.
	if c.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	}

	// Set standard headers.
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)
	if req.Header.Get("Content-Type") == "" && req.Method != http.MethodGet {
		req.Header.Set("Content-Type", "application/json")
	}

	var resp *http.Response
	var err error
	var attempt int

	for {
		// Enforce local rate limit before executing request.
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("local rate limit wait interrupted: %w", err)
		}

		// Execute HTTP request.
		resp, err = c.httpClient.Do(req)
		if err != nil {
			// If context is canceled or deadline exceeded, return immediately.
			if ctx.Err() != nil {
				return nil, fmt.Errorf("request aborted by context: %w", ctx.Err())
			}
			// For network errors, we could implement retry logic here as well,
			// but for now, we only retry explicitly on 429s.
			return nil, fmt.Errorf("http execute request failed: %w", err)
		}

		// Success or non-retryable error, break loop.
		if resp.StatusCode != http.StatusTooManyRequests {
			break
		}

		// Handle 429 Too Many Requests
		if attempt >= c.maxRetries {
			// Drain and close body before returning error to prevent leaks
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			return nil, mapHTTPError(resp, body)
		}

		// Drain body to reuse connection
		_, _ = io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		// Prefer server-suggested Retry-After if present, else exponential backoff.
		backoff := calculateBackoff(attempt, c.backoffBase, c.backoffMax)
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if seconds, parseErr := strconv.Atoi(ra); parseErr == nil && seconds > 0 {
				backoff = time.Duration(seconds) * time.Second
			}
		}

		select {
		case <-time.After(backoff):
			// Proceed to retry
			attempt++
		case <-ctx.Done():
			// Context canceled during backoff
			return nil, fmt.Errorf("context canceled during rate limit backoff: %w", ctx.Err())
		}
	}

	// Handle standard HTTP errors (4xx, 5xx).
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return nil, mapHTTPError(resp, body)
	}

	return resp, nil
}

// get performs a GET request to the specified path and decodes the JSON response into the target interface.
func (c *Client) get(ctx context.Context, path string, target interface{}) error {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}

	resp, err := c.Do(ctx, req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return err
	}

	return nil
}
