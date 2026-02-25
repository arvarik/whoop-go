package whoop

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type mockTransport struct{}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(`{"records": [], "next_token": ""}`)),
		Header:     make(http.Header),
	}, nil
}

func BenchmarkRecoveryService_List(b *testing.B) {
	client := NewClient(
		WithHTTPClient(&http.Client{Transport: &mockTransport{}}),
		WithRateLimiting(false),
	)

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.Recovery.List(ctx, nil)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}
