package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
)

// TransformationEntry represents a transformation log entry within an activity log entry.
type TransformationEntry struct {
	ID           string `json:"id"`
	TxName       string `json:"txName"`
	TxType       string `json:"txType"`
	SuccessRows  int64  `json:"successRows"`
	AffectedRows int64  `json:"affectedRows,omitempty"`
	FailedRows   int64  `json:"failedRows"`
}

// ActivityLogEntry represents a completed job activity log entry.
type ActivityLogEntry struct {
	ID                   string                `json:"id"`
	Type                 string                `json:"type"`
	ObjectID             string                `json:"objectId,omitempty"`
	ObjectName           string                `json:"objectName"`
	RunID                int64                 `json:"runId"`
	AgentID              string                `json:"agentId,omitempty"`
	RuntimeEnvironmentID string                `json:"runtimeEnvironmentId,omitempty"`
	StartTime            string                `json:"startTime,omitempty"`
	EndTime              string                `json:"endTime,omitempty"`
	StartTimeUtc         string                `json:"startTimeUtc,omitempty"`
	EndTimeUtc           string                `json:"endTimeUtc,omitempty"`
	State                int                   `json:"state"`
	FailedSourceRows     int64                 `json:"failedSourceRows,omitempty"`
	SuccessSourceRows    int64                 `json:"successSourceRows,omitempty"`
	FailedTargetRows     int64                 `json:"failedTargetRows,omitempty"`
	SuccessTargetRows    int64                 `json:"successTargetRows,omitempty"`
	ScheduleName         string                `json:"scheduleName,omitempty"`
	ErrorMsg             string                `json:"errorMsg,omitempty"`
	StartedBy            string                `json:"startedBy,omitempty"`
	RunContextType       string                `json:"runContextType,omitempty"`
	IsStopped            bool                  `json:"isStopped,omitempty"`
	TotalSuccessRows     int64                 `json:"totalSuccessRows,omitempty"`
	TotalFailedRows      int64                 `json:"totalFailedRows,omitempty"`
	StopOnError          bool                  `json:"stopOnError,omitempty"`
	HasStopOnErrorRecord bool                  `json:"hasStopOnErrorRecord,omitempty"`
	ContextExternalID    string                `json:"contextExternalId,omitempty"`
	Entries              []ActivityLogEntry    `json:"entries,omitempty"`
	SubTaskEntries       []ActivityLogEntry    `json:"subTaskEntries,omitempty"`
	LogEntryItemAttrs    map[string]string     `json:"logEntryItemAttrs,omitempty"`
	SessionVariables     map[string]string     `json:"sessionVariables,omitempty"`
	TransformationEntries []TransformationEntry `json:"transformationEntries,omitempty"`
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
