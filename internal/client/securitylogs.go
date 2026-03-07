package client

import (
	"context"
	"net/http"
	"strconv"
)

// SecurityLog represents a security log entry.
type SecurityLog struct {
	ID            string `json:"id"`
	OrgID         string `json:"orgId,omitempty"`
	UserName      string `json:"userName"`
	Action        string `json:"action"`
	ObjectType    string `json:"objectType,omitempty"`
	ObjectName    string `json:"objectName,omitempty"`
	Status        string `json:"status,omitempty"`
	SourceIP      string `json:"sourceIp,omitempty"`
	EntryTime     string `json:"entryTime"`
	AdditionalInfo string `json:"additionalInfo,omitempty"`
}

// SecurityLogListOptions holds query parameters for listing security logs.
type SecurityLogListOptions struct {
	StartTime string
	EndTime   string
	Limit     int
	Skip      int
}

// ListSecurityLogs retrieves security log entries.
func (c *Client) ListSecurityLogs(ctx context.Context, opts SecurityLogListOptions) ([]SecurityLog, error) {
	query := make(map[string]string)
	if opts.StartTime != "" {
		query["startTime"] = opts.StartTime
	}
	if opts.EndTime != "" {
		query["endTime"] = opts.EndTime
	}
	if opts.Limit > 0 {
		query["limit"] = strconv.Itoa(opts.Limit)
	}
	if opts.Skip > 0 {
		query["skip"] = strconv.Itoa(opts.Skip)
	}

	var resp []SecurityLog
	if err := c.doJSONWithQuery(ctx, http.MethodGet, "securityLogs", query, nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}
