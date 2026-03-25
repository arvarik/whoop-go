package whoop

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// ErrNoNextPage is returned by NextPage when there are no more pages to fetch.
var ErrNoNextPage = errors.New("no next page available")

// ListOptions specifies the optional parameters to various List methods that support pagination.
type ListOptions struct {
	// Maximum number of items to return. The API limits this to 50.
	Limit int `url:"limit,omitempty"`

	// Earliest date of data to fetch (inclusive). Time is ISO-8601 formatted.
	Start *time.Time `url:"start,omitempty"`

	// Latest date of data to fetch (inclusive). Time is ISO-8601 formatted.
	End *time.Time `url:"end,omitempty"`

	// Token used to fetch the next page of results. Usually handled automatically by the paginator.
	NextToken string `url:"nextToken,omitempty"`
}

// encode safely encodes ListOptions into query parameters.
func (o *ListOptions) encode(u *url.URL) {
	if o == nil {
		return
	}

	q := u.Query()
	if o.Limit > 0 {
		q.Set("limit", strconv.Itoa(o.Limit))
	}
	if o.Start != nil {
		q.Set("start", o.Start.Format(time.RFC3339))
	}
	if o.End != nil {
		q.Set("end", o.End.Format(time.RFC3339))
	}
	if o.NextToken != "" {
		q.Set("nextToken", o.NextToken)
	}

	u.RawQuery = q.Encode()
}

// nextPageOpts creates a new ListOptions with the given next token,
// copying existing options if present.
func nextPageOpts(opts *ListOptions, token string) *ListOptions {
	nextOpts := &ListOptions{}
	if opts != nil {
		*nextOpts = *opts
	}
	nextOpts.NextToken = token
	return nextOpts
}

// paginatedResponse represents the raw JSON wrapping a WHOOP collection array.
type paginatedResponse[T any] struct {
	Records   []T    `json:"records"`
	NextToken string `json:"next_token"`
}

// getPaginated executes a generic GET request for a paginated resource.
func getPaginated[T any](ctx context.Context, client *Client, u *url.URL, opts *ListOptions) (pageRes *paginatedResponse[T], err error) {
	// Copy the URL so that encoding options doesn't modify the cached base
	reqURL := *u
	opts.encode(&reqURL)

	req, err := http.NewRequest(http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	var p paginatedResponse[T]
	if err = json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return nil, err
	}

	return &p, nil
}
