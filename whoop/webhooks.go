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
	ID      string `json:"id"`
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
	limitedBody := io.LimitReader(r.Body, maxWebhookBodySize)
	defer func() { _ = r.Body.Close() }()

	// Calculate HMAC SHA256 signature
	mac := hmac.New(sha256.New, []byte(secret))

	// TeeReader writes to mac as it reads from limitedBody
	tee := io.TeeReader(limitedBody, mac)

	var event WebhookEvent
	// Decode JSON from the TeeReader
	jsonErr := json.NewDecoder(tee).Decode(&event)

	// Consume any remaining body to ensure HMAC is calculated over the entire body
	// This is crucial because json.Decoder might stop reading after the JSON object ends,
	// or if there is whitespace/trailing garbage that is part of the signature.
	if _, err := io.Copy(io.Discard, tee); err != nil {
		return nil, fmt.Errorf("failed to read webhook body: %w", err)
	}

	expectedSig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	// Validate signature to ensure payload integrity
	if !hmac.Equal([]byte(headerSig), []byte(expectedSig)) {
		return nil, errors.New("invalid webhook signature")
	}

	if jsonErr != nil {
		return nil, fmt.Errorf("failed to parse webhook json: %w", jsonErr)
	}

	return &event, nil
}
