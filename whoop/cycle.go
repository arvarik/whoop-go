package whoop

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"time"
)

// Cycle represents a physiological cycle (typically an awake period to the next awake period).
type Cycle struct {
	ID             int        `json:"id"`
	UserID         int        `json:"user_id"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	Start          time.Time  `json:"start"`
	End            *time.Time `json:"end"`
	TimezoneOffset string     `json:"timezone_offset"`
	ScoreState     string     `json:"score_state"`
	Score          *Score     `json:"score,omitempty"`
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

	listURLOnce sync.Once
	listURL     *url.URL
	listURLErr  error
}

// GetByID fetches a single cycle by its ID.
func (s *CycleService) GetByID(ctx context.Context, id int) (*Cycle, error) {
	var cycle Cycle
	if err := s.client.Get(ctx, fmt.Sprintf("/cycle/%d", id), &cycle); err != nil {
		return nil, err
	}

	return &cycle, nil
}

// List fetches a paginated collection of cycles.
func (s *CycleService) List(ctx context.Context, opts *ListOptions) (*CyclePage, error) {
	s.listURLOnce.Do(func() {
		s.listURL, s.listURLErr = url.Parse(s.client.baseURL + "/cycle")
	})
	if s.listURLErr != nil {
		return nil, s.listURLErr
	}

	page, err := getPaginated[Cycle](ctx, s.client, s.listURL, opts)
	if err != nil {
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
		return nil, ErrNoNextPage
	}

	return p.service.List(ctx, nextPageOpts(p.opts, p.NextToken))
}
