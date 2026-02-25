package whoop

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// WebhookEvent represents a "Skinny Webhook" payload from WHOOP.
type WebhookEvent struct {
	UserID  int    `json:"user_id"`
	ID      int    `json:"id"`
	Type    string `json:"type"`
	TraceID string `json:"trace_id"`
}

// maxWebhookBodySize is the maximum allowed size for an incoming webhook payload (1 MB).
const maxWebhookBodySize = 1 << 20

// ParseWebhook reads and verifies an incoming HTTP request from a WHOOP Webhook.
// It validates the X-Whoop-Signature HMAC-SHA256 using the provided secret key.
// The request body is capped at 1 MB to prevent memory exhaustion. Ensure your
// HTTP handler does NOT consume r.Body before passing it to this function.
func ParseWebhook(r *http.Request, secret string) (*WebhookEvent, error) {
	if r.Method != http.MethodPost {
		return nil, errors.New("webhook must be a POST request")
	}

	headerSig := r.Header.Get("X-Whoop-Signature")
	if headerSig == "" {
		return nil, errors.New("missing X-Whoop-Signature header")
	}

	// Cap the body read to prevent memory exhaustion from oversized payloads.
	body, err := io.ReadAll(io.LimitReader(r.Body, maxWebhookBodySize))
	defer func() { _ = r.Body.Close() }()
	if err != nil {
		return nil, fmt.Errorf("failed to read webhook body: %w", err)
	}

	// Calculate HMAC SHA256 signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedSig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	// Validate signature to ensure payload integrity
	if !hmac.Equal([]byte(headerSig), []byte(expectedSig)) {
		return nil, errors.New("invalid webhook signature")
	}

	var event WebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, fmt.Errorf("failed to parse webhook json: %w", err)
	}

	return &event, nil
}
