package whoop

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// signPayload computes the HMAC-SHA256 signature for the given body and secret,
// returning the base64-encoded result expected by ParseWebhook.
func signPayload(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func TestParseWebhook_ValidSignature(t *testing.T) {
	secret := "test-webhook-secret"
	payload := `{"user_id":999,"id":456,"type":"workout.updated","trace_id":"abc-def"}`
	sig := signPayload([]byte(payload), secret)

	req := httptest.NewRequest(http.MethodPost, "/whoop/webhook", strings.NewReader(payload))
	req.Header.Set("X-Whoop-Signature", sig)

	event, err := ParseWebhook(req, secret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if event.UserID != 999 {
		t.Errorf("expected UserID 999, got %d", event.UserID)
	}
	if event.ID != 456 {
		t.Errorf("expected ID 456, got %d", event.ID)
	}
	if event.Type != "workout.updated" {
		t.Errorf("expected type 'workout.updated', got %s", event.Type)
	}
	if event.TraceID != "abc-def" {
		t.Errorf("expected trace_id 'abc-def', got %s", event.TraceID)
	}
}

func TestParseWebhook_InvalidSignature(t *testing.T) {
	payload := `{"user_id":999,"id":456,"type":"workout.updated"}`

	req := httptest.NewRequest(http.MethodPost, "/whoop/webhook", strings.NewReader(payload))
	req.Header.Set("X-Whoop-Signature", "definitely-wrong-signature")

	_, err := ParseWebhook(req, "my-secret")
	if err == nil {
		t.Fatal("expected error for invalid signature, got nil")
	}
	if !strings.Contains(err.Error(), "invalid webhook signature") {
		t.Errorf("expected 'invalid webhook signature' error, got: %v", err)
	}
}

func TestParseWebhook_MissingSignatureHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/whoop/webhook", strings.NewReader(`{}`))
	// No X-Whoop-Signature header set

	_, err := ParseWebhook(req, "my-secret")
	if err == nil {
		t.Fatal("expected error for missing signature header, got nil")
	}
	if !strings.Contains(err.Error(), "missing X-Whoop-Signature") {
		t.Errorf("expected 'missing X-Whoop-Signature' error, got: %v", err)
	}
}

func TestParseWebhook_WrongHTTPMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/whoop/webhook", nil)

	_, err := ParseWebhook(req, "my-secret")
	if err == nil {
		t.Fatal("expected error for non-POST request, got nil")
	}
	if !strings.Contains(err.Error(), "POST") {
		t.Errorf("expected error about POST method, got: %v", err)
	}
}

func TestParseWebhook_OversizedBody(t *testing.T) {
	secret := "test-secret"
	// Create a body larger than maxWebhookBodySize (1 MB)
	oversizedPayload := bytes.Repeat([]byte("A"), maxWebhookBodySize+1024)

	// Sign only the truncated portion that LimitReader would produce
	truncated := oversizedPayload[:maxWebhookBodySize]
	sig := signPayload(truncated, secret)

	req := httptest.NewRequest(http.MethodPost, "/whoop/webhook", bytes.NewReader(oversizedPayload))
	req.Header.Set("X-Whoop-Signature", sig)

	// The body will be silently truncated by LimitReader, so signature
	// validation against the full body will fail or JSON parse will fail.
	// Either way it should not OOM and should return an error.
	_, err := ParseWebhook(req, secret)
	// We only care that it doesn't panic; it will fail on JSON parse
	// since the truncated body is not valid JSON.
	if err == nil {
		t.Fatal("expected error for oversized body, got nil")
	}
}

func TestParseWebhook_InvalidJSON(t *testing.T) {
	secret := "test-secret"
	invalidJSON := `{not valid json}`
	sig := signPayload([]byte(invalidJSON), secret)

	req := httptest.NewRequest(http.MethodPost, "/whoop/webhook", strings.NewReader(invalidJSON))
	req.Header.Set("X-Whoop-Signature", sig)

	_, err := ParseWebhook(req, secret)
	if err == nil {
		t.Fatal("expected error for invalid JSON body, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse webhook json") {
		t.Errorf("expected JSON parse error, got: %v", err)
	}
}

func TestParseWebhook_EmptyBody(t *testing.T) {
	secret := "test-secret"
	emptyBody := ""
	sig := signPayload([]byte(emptyBody), secret)

	req := httptest.NewRequest(http.MethodPost, "/whoop/webhook", strings.NewReader(emptyBody))
	req.Header.Set("X-Whoop-Signature", sig)

	_, err := ParseWebhook(req, secret)
	if err == nil {
		t.Fatal("expected error for empty body (EOF on decode), got nil")
	}
}

func TestParseWebhook_BodyAlreadyConsumed(t *testing.T) {
	secret := "test-secret"
	req := httptest.NewRequest(http.MethodPost, "/whoop/webhook", strings.NewReader(`{"id":1}`))
	req.Header.Set("X-Whoop-Signature", "some-sig")

	// Pre-consume the body
	_, _ = io.ReadAll(req.Body)

	_, err := ParseWebhook(req, secret)
	if err == nil {
		t.Fatal("expected error for pre-consumed body, got nil")
	}
}
