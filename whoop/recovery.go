package whoop

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// Recovery represents the quantified recovery status of the user for a given cycle.
type Recovery struct {
	CycleID   int            `json:"cycle_id"`
	SleepID   int            `json:"sleep_id"`
	UserID    int            `json:"user_id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	Score     *RecoveryScore `json:"score,omitempty"`
}

// RecoveryScore contains the metrics formulating the recovery calculation.
type RecoveryScore struct {
	UserCalibrating  bool    `json:"user_calibrating"`
	RecoveryScore    float64 `json:"recovery_score"`
	RestingHeartRate float64 `json:"resting_heart_rate"`
	HrvRmssdMilli    float64 `json:"hrv_rmssd_milli"`
	Spo2Percentage   float64 `json:"spo2_percentage"`
	SkinTempCelsius  float64 `json:"skin_temp_celsius"`
}

// RecoveryService handles communication with the recovery related methods.
type RecoveryService struct {
	client *Client

	listURLOnce sync.Once
	listURL     *url.URL
	listURLErr  error
}

// GetByID fetches a single recovery score by cycle ID.
func (s *RecoveryService) GetByID(ctx context.Context, cycleID int) (*Recovery, error) {
	var item Recovery
	if err := s.client.get(ctx, fmt.Sprintf("/cycle/%d/recovery", cycleID), &item); err != nil {
		return nil, err
	}

	return &item, nil
}

// List fetches a paginated collection of recovery records.
func (s *RecoveryService) List(ctx context.Context, opts *ListOptions) (*RecoveryPage, error) {
	s.listURLOnce.Do(func() {
		s.listURL, s.listURLErr = url.Parse(s.client.baseURL + "/recovery")
	})
	if s.listURLErr != nil {
		return nil, s.listURLErr
	}

	u := *s.listURL
	opts.encode(&u)

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var page paginatedResponse[Recovery]
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return nil, err
	}

	return &RecoveryPage{
		Records:   page.Records,
		NextToken: page.NextToken,
		service:   s,
		opts:      opts,
	}, nil
}

// RecoveryPage represents a paginated set of Recoveries.
type RecoveryPage struct {
	Records   []Recovery
	NextToken string

	service *RecoveryService
	opts    *ListOptions
}

// NextPage fetches the subsequent page of Recoveries based on NextToken.
func (p *RecoveryPage) NextPage(ctx context.Context) (*RecoveryPage, error) {
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
