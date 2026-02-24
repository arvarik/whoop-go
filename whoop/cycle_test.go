package whoop

import (
	"context"
	"testing"
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
