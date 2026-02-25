package whoop

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type cycleMockTransport struct{}

func (m *cycleMockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(`{"records": [], "next_token": ""}`)),
		Header:     make(http.Header),
	}, nil
}

func BenchmarkCycleService_List(b *testing.B) {
	client := NewClient(
		WithHTTPClient(&http.Client{Transport: &cycleMockTransport{}}),
		WithRateLimiting(false),
	)

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.Cycle.List(ctx, nil)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}
