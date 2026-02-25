package whoop

import (
	"context"
	"testing"
)

func TestUserService_GetBasicProfile(t *testing.T) {
	ts := newMockServer(t)
	defer ts.Close()

	client := newMockClient(ts)

	profile, err := client.User.GetBasicProfile(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if profile.UserID != 999 {
		t.Errorf("expected UserID 999, got %d", profile.UserID)
	}
	if profile.Email != "athlete@example.com" {
		t.Errorf("expected email 'athlete@example.com', got %s", profile.Email)
	}
	if profile.FirstName != "Jane" {
		t.Errorf("expected first name 'Jane', got %s", profile.FirstName)
	}
	if profile.LastName != "Doe" {
		t.Errorf("expected last name 'Doe', got %s", profile.LastName)
	}
}

func TestUserService_GetBodyMeasurement(t *testing.T) {
	ts := newMockServer(t)
	defer ts.Close()

	client := newMockClient(ts)

	m, err := client.User.GetBodyMeasurement(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if m.HeightMeter != 1.75 {
		t.Errorf("expected height 1.75, got %f", m.HeightMeter)
	}
	if m.WeightKilogram != 70.5 {
		t.Errorf("expected weight 70.5, got %f", m.WeightKilogram)
	}
	if m.MaxHeartRate != 195 {
		t.Errorf("expected max heart rate 195, got %d", m.MaxHeartRate)
	}
}
