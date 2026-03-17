package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
)

// ActivityLogEntry represents a completed job activity log entry.
type ActivityLogEntry struct {
	ID                   string             `json:"id"`
	Type                 string             `json:"type"`
	ObjectID             string             `json:"objectId"`
	ObjectName           string             `json:"objectName"`
	RunID                int64              `json:"runId"`
	AgentID              string             `json:"agentId"`
	RuntimeEnvironmentID string             `json:"runtimeEnvironmentId"`
	StartTime            string             `json:"startTime"`
	EndTime              string             `json:"endTime"`
	StartTimeUtc         string             `json:"startTimeUtc"`
	EndTimeUtc           string             `json:"endTimeUtc"`
	State                int                `json:"state"`
	FailedSourceRows     int64              `json:"failedSourceRows"`
	SuccessSourceRows    int64              `json:"successSourceRows"`
	FailedTargetRows     int64              `json:"failedTargetRows"`
	SuccessTargetRows    int64              `json:"successTargetRows"`
	ScheduleName         string             `json:"scheduleName"`
	ErrorMsg             string             `json:"errorMsg"`
	StartedBy            string             `json:"startedBy"`
	RunContextType       string             `json:"runContextType"`
	IsStopped            bool               `json:"isStopped"`
	Entries              []ActivityLogEntry `json:"entries"`
}

// ActivityLogListOptions holds query parameters for listing activity logs.
type ActivityLogListOptions struct {
	RunID    int64
	TaskID   string
	Offset   int
	RowLimit int
}

// ListActivityLogs retrieves activity log entries for completed jobs.
func (c *Client) ListActivityLogs(ctx context.Context, opts ActivityLogListOptions) ([]ActivityLogEntry, error) {
	query := make(map[string]string)
	if opts.RunID > 0 {
		query["runId"] = strconv.FormatInt(opts.RunID, 10)
	}
	if opts.TaskID != "" {
		query["taskId"] = opts.TaskID
	}
	if opts.Offset > 0 {
		query["offset"] = strconv.Itoa(opts.Offset)
	}
	if opts.RowLimit > 0 {
		query["rowLimit"] = strconv.Itoa(opts.RowLimit)
	}

	var resp []ActivityLogEntry
	if err := c.doJSONWithQuery(ctx, http.MethodGet, BaseAPIPathV2+"/activity/activityLog", query, nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetActivityLog retrieves a single activity log entry by ID.
func (c *Client) GetActivityLog(ctx context.Context, id string) (*ActivityLogEntry, error) {
	var entry ActivityLogEntry
	if err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("%s/activity/activityLog/%s", BaseAPIPathV2, id), nil, &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}
