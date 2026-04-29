package whoop

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

// Sleep represents a single sleep event.
type Sleep struct {
	ID             string      `json:"id"`
	CycleID        int         `json:"cycle_id"`
	V1ID           *int        `json:"v1_id,omitempty"`
	UserID         int         `json:"user_id"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
	Start          time.Time   `json:"start"`
	End            time.Time   `json:"end"`
	TimezoneOffset string      `json:"timezone_offset"`
	Nap            bool        `json:"nap"`
	ScoreState     string      `json:"score_state"`
	Score          *SleepScore `json:"score,omitempty"`
}

// SleepScore provides calculated metrics for a Sleep.
type SleepScore struct {
	StageSummary               *StageSummary `json:"stage_summary"`
	SleepNeeded                *SleepNeeded  `json:"sleep_needed"`
	RespiratoryRate            float64       `json:"respiratory_rate"`
	SleepPerformancePercentage float64       `json:"sleep_performance_percentage"`
	SleepConsistencyPercentage float64       `json:"sleep_consistency_percentage"`
	SleepEfficiencyPercentage  float64       `json:"sleep_efficiency_percentage"`
}

// StageSummary breaks down durations spent in different sleep stages.
type StageSummary struct {
	TotalInBedTimeMilli         int `json:"total_in_bed_time_milli"`
	TotalAwakeTimeMilli         int `json:"total_awake_time_milli"`
	TotalNoDataTimeMilli        int `json:"total_no_data_time_milli"`
	TotalLightSleepTimeMilli    int `json:"total_light_sleep_time_milli"`
	TotalSlowWaveSleepTimeMilli int `json:"total_slow_wave_sleep_time_milli"`
	TotalRemSleepTimeMilli      int `json:"total_rem_sleep_time_milli"`
	SleepCycleCount             int `json:"sleep_cycle_count"`
	DisturbanceCount            int `json:"disturbance_count"`
}

// SleepNeeded defines baseline and calculated sleep needs for the individual.
type SleepNeeded struct {
	BaselineMilli             int `json:"baseline_milli"`
	NeedFromSleepDebtMilli    int `json:"need_from_sleep_debt_milli"`
	NeedFromRecentStrainMilli int `json:"need_from_recent_strain_milli"`
	NeedFromRecentNapMilli    int `json:"need_from_recent_nap_milli"`
}

// SleepService handles communication with the sleep related methods.
type SleepService struct {
	client *Client
}

// GetByID fetches a single sleep event by its UUID.
func (s *SleepService) GetByID(ctx context.Context, id string) (*Sleep, error) {
	var item Sleep
	if err := s.client.Get(ctx, fmt.Sprintf("/activity/sleep/%s", url.PathEscape(id)), &item); err != nil {
		return nil, err
	}

	return &item, nil
}

// List fetches a paginated collection of sleep events.
func (s *SleepService) List(ctx context.Context, opts *ListOptions) (*SleepPage, error) {
	page, err := getPaginated[Sleep](ctx, s.client, "/activity/sleep", opts)
	if err != nil {
		return nil, err
	}

	return &SleepPage{
		Records:   page.Records,
		NextToken: page.NextToken,
		service:   s,
		opts:      opts,
	}, nil
}

// SleepPage represents a paginated set of Sleep activities.
type SleepPage struct {
	Records   []Sleep
	NextToken string

	service *SleepService
	opts    *ListOptions
}

// NextPage fetches the subsequent page of Sleep events based on NextToken.
func (p *SleepPage) NextPage(ctx context.Context) (*SleepPage, error) {
	if p.NextToken == "" {
		return nil, ErrNoNextPage
	}

	return p.service.List(ctx, nextPageOpts(p.opts, p.NextToken))
}
