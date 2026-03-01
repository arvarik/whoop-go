package whoop

import (
	"context"
	"errors"
	"testing"
)

func TestSleepService_GetByID(t *testing.T) {
	ts := newMockServer(t)
	defer ts.Close()

	client := newMockClient(ts)

	sleep, err := client.Sleep.GetByID(context.Background(), "slp-uuid-789")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sleep.ID != "slp-uuid-789" {
		t.Errorf("expected ID slp-uuid-789, got %s", sleep.ID)
	}
	if sleep.UserID != 999 {
		t.Errorf("expected UserID 999, got %d", sleep.UserID)
	}
	if sleep.Nap {
		t.Error("expected Nap to be false")
	}
	if sleep.Score == nil {
		t.Fatal("expected score to be populated")
	}
	if sleep.Score.RespiratoryRate != 15.5 {
		t.Errorf("expected respiratory rate 15.5, got %f", sleep.Score.RespiratoryRate)
	}
	if sleep.Score.SleepPerformancePercentage != 95.0 {
		t.Errorf("expected sleep performance 95.0, got %f", sleep.Score.SleepPerformancePercentage)
	}
	if sleep.Score.StageSummary == nil {
		t.Fatal("expected stage summary to be populated")
	}
	if sleep.Score.StageSummary.SleepCycleCount != 4 {
		t.Errorf("expected 4 sleep cycles, got %d", sleep.Score.StageSummary.SleepCycleCount)
	}
	if sleep.Score.SleepNeeded == nil {
		t.Fatal("expected sleep needed to be populated")
	}
	if sleep.Score.SleepNeeded.BaselineMilli != 28800000 {
		t.Errorf("expected baseline 28800000, got %d", sleep.Score.SleepNeeded.BaselineMilli)
	}
}

func TestSleepService_List_Pagination(t *testing.T) {
	ts := newMockServer(t)
	defer ts.Close()

	client := newMockClient(ts)

	page1, err := client.Sleep.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error fetching page 1: %v", err)
	}

	if len(page1.Records) != 1 {
		t.Fatalf("expected 1 record on page 1, got %d", len(page1.Records))
	}
	if page1.Records[0].ID != "slp-uuid-789" {
		t.Errorf("expected sleep ID slp-uuid-789, got %s", page1.Records[0].ID)
	}
	if page1.NextToken != "sleep-p2" {
		t.Errorf("expected next token 'sleep-p2', got '%s'", page1.NextToken)
	}

	// Fetch page 2
	page2, err := page1.NextPage(context.Background())
	if err != nil {
		t.Fatalf("unexpected error fetching page 2: %v", err)
	}
	if len(page2.Records) != 0 {
		t.Errorf("expected 0 records on page 2, got %d", len(page2.Records))
	}

	// No more pages
	_, err = page2.NextPage(context.Background())
	if !errors.Is(err, ErrNoNextPage) {
		t.Errorf("expected ErrNoNextPage, got %v", err)
	}
}
