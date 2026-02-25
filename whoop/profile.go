package whoop

import (
	"context"
)

// BasicProfile represents the user's basic profile information.
type BasicProfile struct {
	UserID    int    `json:"user_id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// BodyMeasurement represents the user's physical body measurements.
type BodyMeasurement struct {
	HeightMeter    float64 `json:"height_meter"`
	WeightKilogram float64 `json:"weight_kilogram"`
	MaxHeartRate   int     `json:"max_heart_rate"`
}

// UserService handles communication with the user related methods.
type UserService struct {
	client *Client
}

// GetBasicProfile fetches the athlete's basic profile.
func (s *UserService) GetBasicProfile(ctx context.Context) (*BasicProfile, error) {
	var profile BasicProfile
	if err := s.client.get(ctx, "/user/profile/basic", &profile); err != nil {
		return nil, err
	}

	return &profile, nil
}

// GetBodyMeasurement fetches the athlete's body measurements.
func (s *UserService) GetBodyMeasurement(ctx context.Context) (*BodyMeasurement, error) {
	var measurement BodyMeasurement
	if err := s.client.get(ctx, "/user/measurement/body", &measurement); err != nil {
		return nil, err
	}

	return &measurement, nil
}
