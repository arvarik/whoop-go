package whoop

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func BenchmarkCycleGetByID_Sprintf(b *testing.B) {
	id := 123
	for i := 0; i < b.N; i++ {
		_ = fmt.Sprintf("/cycle/%d", id)
	}
}

func BenchmarkCycleGetByID_Itoa(b *testing.B) {
	id := 123
	for i := 0; i < b.N; i++ {
		_ = "/cycle/" + strconv.Itoa(id)
	}
}

func BenchmarkCycleService_GetByID(b *testing.B) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": 123}`))
	}))
	defer ts.Close()

	client := NewClient(WithBaseURL(ts.URL))
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.Cycle.GetByID(ctx, 123)
	}
}
