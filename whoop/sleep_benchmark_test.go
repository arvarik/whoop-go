package whoop

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
)

type mockRoundTripper struct {
	responseBody []byte
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(m.responseBody)),
	}, nil
}

func BenchmarkSleepService_List(b *testing.B) {
	mockBody := []byte(`{"records": [], "next_token": ""}`)
	client := NewClient(WithHTTPClient(&http.Client{
		Transport: &mockRoundTripper{responseBody: mockBody},
	}))

	// Disable rate limiting for benchmark
	client.rateLimiter.isAutoLimiting.Store(false)

	ctx := context.Background()
	opts := &ListOptions{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.Sleep.List(ctx, opts)
		if err != nil {
			b.Fatal(err)
		}
	}
}
