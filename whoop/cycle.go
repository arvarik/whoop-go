package whoop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Cycle represents a physiological cycle (typically an awake period to the next awake period).
type Cycle struct {
	ID             int       `json:"id"`
	UserID         int       `json:"user_id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Start          time.Time `json:"start"`
	End            time.Time `json:"end,omitempty"`
	TimezoneOffset string    `json:"timezone_offset"`
	Score          *Score    `json:"score,omitempty"`
}

// Score summarizes physiological strains within a Cycle.
type Score struct {
	Strain           float64 `json:"strain"`
	Kilojoule        float64 `json:"kilojoule"`
	AverageHeartRate int     `json:"average_heart_rate"`
	MaxHeartRate     int     `json:"max_heart_rate"`
}

// CycleService handles communication with the cycle related methods.
type CycleService struct {
	client *Client
}

// GetByID fetches a single cycle by its ID.
func (s *CycleService) GetByID(ctx context.Context, id int) (*Cycle, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/cycle/%d", s.client.baseURL, id), nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var cycle Cycle
	if err := json.NewDecoder(resp.Body).Decode(&cycle); err != nil {
		return nil, err
	}

	return &cycle, nil
}

// List fetches a paginated collection of cycles.
func (s *CycleService) List(ctx context.Context, opts *ListOptions) (*CyclePage, error) {
	u, err := url.Parse(s.client.baseURL + "/cycle")
	if err != nil {
		return nil, err
	}

	opts.encode(u)

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var page paginatedResponse[Cycle]
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return nil, err
	}

	return &CyclePage{
		Records:   page.Records,
		NextToken: page.NextToken,
		service:   s,
		opts:      opts,
	}, nil
}

// CyclePage represents a paginated set of Cycles.
type CyclePage struct {
	Records   []Cycle
	NextToken string

	service *CycleService
	opts    *ListOptions
}

// NextPage fetches the subsequent page of cycles based on NextToken.
// Returns an error if there is no next page.
func (p *CyclePage) NextPage(ctx context.Context) (*CyclePage, error) {
	if p.NextToken == "" {
		return nil, errors.New("no next page available")
	}

	// Copy existing options or initialize if nil
	nextOpts := &ListOptions{}
	if p.opts != nil {
		*nextOpts = *p.opts
	}
	nextOpts.NextToken = p.NextToken

	return p.service.List(ctx, nextOpts)
}
