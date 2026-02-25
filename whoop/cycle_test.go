package whoop

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCycleService_GetByID_MockIntegration(t *testing.T) {
	ts := newMockServer(t)
	defer ts.Close()

	client := newMockClient(ts)

	cycle, err := client.Cycle.GetByID(context.Background(), 123)
	if err != nil {
		t.Fatalf("unexpected error fetching cycle: %v", err)
	}

	if cycle.ID != 123 {
		t.Errorf("expected cycle ID 123, got %d", cycle.ID)
	}
	if cycle.UserID != 999 {
		t.Errorf("expected cycle UserID 999, got %d", cycle.UserID)
	}
	if cycle.Score == nil {
		t.Fatal("expected cycle score to be populated, got nil")
	}
	if cycle.Score.Strain != 12.4 {
		t.Errorf("expected cycle Strain 12.4, got %f", cycle.Score.Strain)
	}
	if cycle.TimezoneOffset != "-08:00" {
		t.Errorf("expected timezone offset -08:00, got %s", cycle.TimezoneOffset)
	}
}

// TestCycleService_List_OptionsIntegration verifies that ListOptions are correctly
// encoded into query parameters when calling CycleService.List.
func TestCycleService_List_OptionsIntegration(t *testing.T) {
	// Define test parameters
	start := time.Date(2023, 10, 26, 0, 0, 0, 0, time.UTC)
	end := time.Date(2023, 10, 27, 0, 0, 0, 0, time.UTC)
	limit := 10
	nextToken := "next_token_123"

	// Create a mock server that verifies the request query parameters
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the path
		if r.URL.Path != "/cycle" {
			t.Errorf("expected path /cycle, got %s", r.URL.Path)
		}

		// Verify query parameters
		q := r.URL.Query()
		if got := q.Get("limit"); got != "10" {
			t.Errorf("expected limit=10, got %s", got)
		}

		expectedStart := start.Format(time.RFC3339)
		if got := q.Get("start"); got != expectedStart {
			t.Errorf("expected start=%s, got %s", expectedStart, got)
		}

		expectedEnd := end.Format(time.RFC3339)
		if got := q.Get("end"); got != expectedEnd {
			t.Errorf("expected end=%s, got %s", expectedEnd, got)
		}

		if got := q.Get("nextToken"); got != nextToken {
			t.Errorf("expected nextToken=%s, got %s", nextToken, got)
		}

		// Return a minimal valid response to ensure the method succeeds
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"records": [], "next_token": ""}`))
	}))
	defer ts.Close()

	// Initialize the client with the mock server URL
	client := NewClient(WithBaseURL(ts.URL))

	// Prepare options
	opts := &ListOptions{
		Limit:     limit,
		Start:     &start,
		End:       &end,
		NextToken: nextToken,
	}

	// Execute List
	_, err := client.Cycle.List(context.Background(), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
