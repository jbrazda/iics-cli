package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
)

// Schedule represents an IICS schedule.
type Schedule struct {
	ID          string `json:"id,omitempty"`
	OrgID       string `json:"orgId,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status,omitempty"`
	CreateTime  string `json:"createTime,omitempty"`
	UpdateTime  string `json:"updateTime,omitempty"`
	CreatedBy   string `json:"createdBy,omitempty"`
	UpdatedBy   string `json:"updatedBy,omitempty"`
	StartTime   string `json:"startTime,omitempty"`
	EndTime     string `json:"endTime,omitempty"`
	Interval    string `json:"interval,omitempty"`
	Frequency   int    `json:"frequency,omitempty"`
	Timezone    string `json:"timezone,omitempty"`
	DayOfWeek   string `json:"dayOfWeek,omitempty"`
	DayOfMonth  string `json:"dayOfMonth,omitempty"`
	WeekOfMonth string `json:"weekOfMonth,omitempty"`
	RangeStartTime string `json:"rangeStartTime,omitempty"`
	RangeEndTime   string `json:"rangeEndTime,omitempty"`
}

// ScheduleListOptions holds query parameters for listing schedules.
type ScheduleListOptions struct {
	Limit int
	Skip  int
}

// ListSchedules retrieves schedules.
func (c *Client) ListSchedules(ctx context.Context, opts ScheduleListOptions) ([]Schedule, error) {
	query := make(map[string]string)
	if opts.Limit > 0 {
		query["limit"] = strconv.Itoa(opts.Limit)
	}
	if opts.Skip > 0 {
		query["skip"] = strconv.Itoa(opts.Skip)
	}

	var resp []Schedule
	if err := c.doJSONWithQuery(ctx, http.MethodGet, "public/core/v3/schedule", query, nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetSchedule retrieves a single schedule by ID.
func (c *Client) GetSchedule(ctx context.Context, id string) (*Schedule, error) {
	var resp Schedule
	if err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("public/core/v3/schedule/%s", id), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreateSchedule creates a new schedule.
func (c *Client) CreateSchedule(ctx context.Context, schedule *Schedule) (*Schedule, error) {
	var resp Schedule
	if err := c.doJSON(ctx, http.MethodPost, "public/core/v3/schedule", schedule, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateSchedule updates an existing schedule.
func (c *Client) UpdateSchedule(ctx context.Context, id string, schedule *Schedule) (*Schedule, error) {
	var resp Schedule
	if err := c.doJSON(ctx, http.MethodPut, fmt.Sprintf("public/core/v3/schedule/%s", id), schedule, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteSchedule deletes a schedule by ID.
func (c *Client) DeleteSchedule(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, fmt.Sprintf("public/core/v3/schedule/%s", id), nil, nil)
}
