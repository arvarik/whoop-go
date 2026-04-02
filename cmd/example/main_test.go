package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/arvarik/whoop-go/whoop"
)

// signPayload computes the HMAC-SHA256 signature for the given body and secret,
// returning the base64-encoded result expected by whoop.ParseWebhook.
func signPayload(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func TestWebhookHandler(t *testing.T) {
	secret := "test-webhook-secret"
	client := whoop.NewClient()

	tests := []struct {
		name           string
		payload        string
		invalidSig     bool
		jobQueueSize   int
		fillQueue      bool
		expectedStatus int
		expectInQueue  string
	}{
		{
			name:           "Valid workout.updated",
			payload:        `{"id":"workout-123","type":"workout.updated"}`,
			jobQueueSize:   1,
			expectedStatus: http.StatusOK,
			expectInQueue:  "workout-123",
		},
		{
			name:           "Valid other event type",
			payload:        `{"id":"recovery-456","type":"recovery.updated"}`,
			jobQueueSize:   1,
			expectedStatus: http.StatusOK,
			expectInQueue:  "",
		},
		{
			name:           "Invalid signature",
			payload:        `{"id":"workout-789","type":"workout.updated"}`,
			invalidSig:     true,
			jobQueueSize:   1,
			expectedStatus: http.StatusUnauthorized,
			expectInQueue:  "",
		},
		{
			name:           "Full job queue",
			payload:        `{"id":"workout-999","type":"workout.updated"}`,
			jobQueueSize:   1,
			fillQueue:      true,
			expectedStatus: http.StatusOK,
			expectInQueue:  "full", // Special marker to check it didn't block and didn't overwrite
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jobQueue := make(chan string, tt.jobQueueSize)
			if tt.fillQueue {
				jobQueue <- "full"
			}

			handler := webhookHandler(client, secret, jobQueue)

			sig := signPayload([]byte(tt.payload), secret)
			if tt.invalidSig {
				sig = "invalid-signature"
			}

			req := httptest.NewRequest(http.MethodPost, "/whoop/webhook", strings.NewReader(tt.payload))
			req.Header.Set("X-Whoop-Signature", sig)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.expectInQueue == "full" {
				// Verify queue still has "full" and wasn't blocked
				select {
				case val := <-jobQueue:
					if val != "full" {
						t.Errorf("expected 'full' in queue, got %s", val)
					}
				default:
					t.Error("expected something in queue")
				}
			} else if tt.expectInQueue != "" {
				select {
				case val := <-jobQueue:
					if val != tt.expectInQueue {
						t.Errorf("expected %s in queue, got %s", tt.expectInQueue, val)
					}
				default:
					t.Error("expected workout ID in queue, but it was empty")
				}
			} else {
				// Should be empty
				select {
				case val := <-jobQueue:
					t.Errorf("expected empty queue, but got %s", val)
				default:
					// OK
				}
			}
		})
	}
}
