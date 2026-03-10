package whoop

import (
	"context"
	"encoding/json"
	"net/http"
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
func (s *UserService) GetBasicProfile(ctx context.Context) (profile *BasicProfile, err error) {
	req, err := http.NewRequest(http.MethodGet, s.client.baseURL+"/user/profile/basic", nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	var p BasicProfile
	if err = json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return nil, err
	}

	return &p, nil
}

// GetBodyMeasurement fetches the athlete's body measurements.
func (s *UserService) GetBodyMeasurement(ctx context.Context) (measurement *BodyMeasurement, err error) {
	req, err := http.NewRequest(http.MethodGet, s.client.baseURL+"/user/measurement/body", nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	var m BodyMeasurement
	if err = json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return nil, err
	}

	return &m, nil
}
