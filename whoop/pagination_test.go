package whoop

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestListOptionsEncoding(t *testing.T) {
	tm := time.Date(2026, 2, 24, 0, 0, 0, 0, time.UTC)
	opts := &ListOptions{
		Limit:     25,
		Start:     &tm,
		NextToken: "abc123token",
	}

	u, _ := url.Parse("https://api.example.com/data")
	opts.encode(u)

	q := u.Query()
	if q.Get("limit") != "25" {
		t.Errorf("expected limit=25, got %s", q.Get("limit"))
	}
	if q.Get("start") != "2026-02-24T00:00:00Z" {
		t.Errorf("expected start=2026-02-24T00:00:00Z, got %s", q.Get("start"))
	}
	if q.Get("end") != "" {
		t.Errorf("expected empty end, got %s", q.Get("end"))
	}
	if q.Get("nextToken") != "abc123token" {
		t.Errorf("expected nextToken=abc123token, got %s", q.Get("nextToken"))
	}
}

func TestListOptionsEncoding_Nil(t *testing.T) {
	u, _ := url.Parse("https://api.example.com/data")
	var opts *ListOptions
	opts.encode(u)

	if u.RawQuery != "" {
		t.Errorf("expected no query params for nil opts, got %s", u.RawQuery)
	}
}

func TestServiceInitialization(t *testing.T) {
	client := NewClient()

	if client.User == nil {
		t.Error("expected client.User to be initialized")
	}
	if client.Cycle == nil {
		t.Error("expected client.Cycle to be initialized")
	}
	if client.Sleep == nil {
		t.Error("expected client.Sleep to be initialized")
	}
	if client.Workout == nil {
		t.Error("expected client.Workout to be initialized")
	}
	if client.Recovery == nil {
		t.Error("expected client.Recovery to be initialized")
	}
}

func TestCycleService_List_Pagination(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/cycle", func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("nextToken")
		w.Header().Set("Content-Type", "application/json")

		switch token {
		case "":
			// First page
			_, _ = w.Write([]byte(`{
				"records": [{"id": 1, "user_id": 123}],
				"next_token": "page2"
			}`))
		case "page2":
			// Second page
			_, _ = w.Write([]byte(`{
				"records": [{"id": 2, "user_id": 123}],
				"next_token": ""
			}`))
		}
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	client := NewClient(WithBaseURL(ts.URL))

	// Fetch Page 1
	page1, err := client.Cycle.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error fetching page 1: %v", err)
	}

	if len(page1.Records) != 1 || page1.Records[0].ID != 1 {
		t.Errorf("expected page 1 to have cycle ID 1")
	}
	if page1.NextToken != "page2" {
		t.Errorf("expected next token to be 'page2', got '%s'", page1.NextToken)
	}

	// Fetch Page 2 using NextPage iterator
	page2, err := page1.NextPage(context.Background())
	if err != nil {
		t.Fatalf("unexpected error fetching page 2: %v", err)
	}

	if len(page2.Records) != 1 || page2.Records[0].ID != 2 {
		t.Errorf("expected page 2 to have cycle ID 2")
	}
	if page2.NextToken != "" {
		t.Errorf("expected empty next token, got '%s'", page2.NextToken)
	}

	// Fetch Page 3 (should fail with sentinel error)
	_, err = page2.NextPage(context.Background())
	if !errors.Is(err, ErrNoNextPage) {
		t.Errorf("expected ErrNoNextPage, got %v", err)
	}
}
