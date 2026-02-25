package whoop

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func BenchmarkParseWebhook(b *testing.B) {
	secret := "benchmark-secret"
	// Create a 512KB payload
	size := 512 * 1024
	data := make([]byte, size)

	prefix := []byte(`{"user_id":123,"id":456,"type":"test","data":"`)
	suffix := []byte(`"}`)
	fillSize := size - len(prefix) - len(suffix)

	copy(data, prefix)
	for i := 0; i < fillSize; i++ {
		data[len(prefix)+i] = 'a'
	}
	copy(data[len(data)-len(suffix):], suffix)

	sig := signPayload(data, secret)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/whoop/webhook", bytes.NewReader(data))
		req.Header.Set("X-Whoop-Signature", sig)

		_, err := ParseWebhook(req, secret)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

func BenchmarkParseWebhookSmall(b *testing.B) {
	secret := "benchmark-secret"
	// Small payload (typical webhook)
	payload := `{"user_id":12345,"id":67890,"type":"workout.updated","trace_id":"abcdef-123456-7890"}`
	data := []byte(payload)
	sig := signPayload(data, secret)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/whoop/webhook", bytes.NewReader(data))
		req.Header.Set("X-Whoop-Signature", sig)

		_, err := ParseWebhook(req, secret)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}
