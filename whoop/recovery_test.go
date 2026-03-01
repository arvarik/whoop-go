package whoop

import (
	"context"
	"errors"
	"testing"
)

func TestRecoveryService_GetByID(t *testing.T) {
	ts := newMockServer(t)
	defer ts.Close()

	client := newMockClient(ts)

	recovery, err := client.Recovery.GetByID(context.Background(), 123)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if recovery.CycleID != 123 {
		t.Errorf("expected CycleID 123, got %d", recovery.CycleID)
	}
	if recovery.SleepID != "slp-uuid-789" {
		t.Errorf("expected SleepID slp-uuid-789, got %s", recovery.SleepID)
	}
	if recovery.UserID != 999 {
		t.Errorf("expected UserID 999, got %d", recovery.UserID)
	}
	if recovery.Score == nil {
		t.Fatal("expected score to be populated")
	}
	if recovery.Score.RecoveryScore != 85.5 {
		t.Errorf("expected recovery score 85.5, got %f", recovery.Score.RecoveryScore)
	}
	if recovery.Score.RestingHeartRate != 52.0 {
		t.Errorf("expected resting heart rate 52.0, got %f", recovery.Score.RestingHeartRate)
	}
	if recovery.Score.HrvRmssdMilli != 65.3 {
		t.Errorf("expected HRV 65.3, got %f", recovery.Score.HrvRmssdMilli)
	}
	if recovery.Score.Spo2Percentage != 97.0 {
		t.Errorf("expected SpO2 97.0, got %f", recovery.Score.Spo2Percentage)
	}
	if recovery.Score.SkinTempCelsius != 33.5 {
		t.Errorf("expected skin temp 33.5, got %f", recovery.Score.SkinTempCelsius)
	}
}

func TestRecoveryService_List_Pagination(t *testing.T) {
	ts := newMockServer(t)
	defer ts.Close()

	client := newMockClient(ts)

	page1, err := client.Recovery.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error fetching page 1: %v", err)
	}

	if len(page1.Records) != 1 {
		t.Fatalf("expected 1 record on page 1, got %d", len(page1.Records))
	}
	if page1.Records[0].CycleID != 123 {
		t.Errorf("expected cycle ID 123, got %d", page1.Records[0].CycleID)
	}

	page2, err := page1.NextPage(context.Background())
	if err != nil {
		t.Fatalf("unexpected error fetching page 2: %v", err)
	}
	if len(page2.Records) != 0 {
		t.Errorf("expected 0 records on page 2, got %d", len(page2.Records))
	}

	_, err = page2.NextPage(context.Background())
	if !errors.Is(err, ErrNoNextPage) {
		t.Errorf("expected ErrNoNextPage, got %v", err)
	}
}
