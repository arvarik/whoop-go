package whoop

import (
	"context"
	"errors"
	"testing"
)

func TestWorkoutService_GetByID(t *testing.T) {
	ts := newMockServer(t)
	defer ts.Close()

	client := newMockClient(ts)

	workout, err := client.Workout.GetByID(context.Background(), 456)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if workout.ID != 456 {
		t.Errorf("expected ID 456, got %d", workout.ID)
	}
	if workout.UserID != 999 {
		t.Errorf("expected UserID 999, got %d", workout.UserID)
	}
	if workout.SportID != 1 {
		t.Errorf("expected SportID 1, got %d", workout.SportID)
	}
	if workout.Score == nil {
		t.Fatal("expected score to be populated")
	}
	if workout.Score.Strain != 14.2 {
		t.Errorf("expected strain 14.2, got %f", workout.Score.Strain)
	}
	if workout.Score.MaxHeartRate != 190 {
		t.Errorf("expected max heart rate 190, got %d", workout.Score.MaxHeartRate)
	}
	if workout.Score.DistanceMeter != 5000.0 {
		t.Errorf("expected distance 5000.0, got %f", workout.Score.DistanceMeter)
	}
	if workout.Score.ZoneDuration == nil {
		t.Fatal("expected zone duration to be populated")
	}
	if workout.Score.ZoneDuration.ZoneFiveMilli != 6000 {
		t.Errorf("expected zone five 6000, got %d", workout.Score.ZoneDuration.ZoneFiveMilli)
	}
}

func TestWorkoutService_List_Pagination(t *testing.T) {
	ts := newMockServer(t)
	defer ts.Close()

	client := newMockClient(ts)

	page1, err := client.Workout.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error fetching page 1: %v", err)
	}

	if len(page1.Records) != 1 {
		t.Fatalf("expected 1 record on page 1, got %d", len(page1.Records))
	}
	if page1.Records[0].ID != 456 {
		t.Errorf("expected workout ID 456, got %d", page1.Records[0].ID)
	}
	if page1.NextToken != "page2" {
		t.Errorf("expected next token 'page2', got '%s'", page1.NextToken)
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
