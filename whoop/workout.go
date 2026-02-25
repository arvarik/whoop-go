package whoop

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Workout represents a tracked workout session.
type Workout struct {
	ID             int           `json:"id"`
	UserID         int           `json:"user_id"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
	Start          time.Time     `json:"start"`
	End            time.Time     `json:"end"`
	TimezoneOffset string        `json:"timezone_offset"`
	SportID        int           `json:"sport_id"`
	Score          *WorkoutScore `json:"score,omitempty"`
}

// WorkoutScore details the cardiovascular output of a given workout.
type WorkoutScore struct {
	Strain              float64        `json:"strain"`
	AverageHeartRate    int            `json:"average_heart_rate"`
	MaxHeartRate        int            `json:"max_heart_rate"`
	Kilojoule           float64        `json:"kilojoule"`
	PercentRecorded     float64        `json:"percent_recorded"`
	DistanceMeter       float64        `json:"distance_meter"`
	AltitudeGainMeter   float64        `json:"altitude_gain_meter"`
	AltitudeChangeMeter float64        `json:"altitude_change_meter"`
	ZoneDuration        *ZoneDurations `json:"zone_duration"`
}

// ZoneDurations breaks down the duration spent in different heart rate zones.
type ZoneDurations struct {
	ZoneZeroMilli  int `json:"zone_zero_milli"`
	ZoneOneMilli   int `json:"zone_one_milli"`
	ZoneTwoMilli   int `json:"zone_two_milli"`
	ZoneThreeMilli int `json:"zone_three_milli"`
	ZoneFourMilli  int `json:"zone_four_milli"`
	ZoneFiveMilli  int `json:"zone_five_milli"`
}

// WorkoutService handles communication with the workout related methods.
type WorkoutService struct {
	client *Client
}

// GetByID fetches a single workout session by its ID.
func (s *WorkoutService) GetByID(ctx context.Context, id int) (*Workout, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/activity/workout/%d", s.client.baseURL, id), nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var item Workout
	if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
		return nil, err
	}

	return &item, nil
}

// List fetches a paginated collection of workout sessions.
func (s *WorkoutService) List(ctx context.Context, opts *ListOptions) (*WorkoutPage, error) {
	page, err := list[Workout](ctx, s.client, "/activity/workout", opts)
	if err != nil {
		return nil, err
	}

	return &WorkoutPage{
		Records:   page.Records,
		NextToken: page.NextToken,
		service:   s,
		opts:      opts,
	}, nil
}

// WorkoutPage represents a paginated set of Workouts.
type WorkoutPage struct {
	Records   []Workout
	NextToken string

	service *WorkoutService
	opts    *ListOptions
}

// NextPage fetches the subsequent page of Workouts based on NextToken.
func (p *WorkoutPage) NextPage(ctx context.Context) (*WorkoutPage, error) {
	if p.NextToken == "" {
		return nil, ErrNoNextPage
	}

	nextOpts := &ListOptions{}
	if p.opts != nil {
		*nextOpts = *p.opts
	}
	nextOpts.NextToken = p.NextToken

	return p.service.List(ctx, nextOpts)
}
