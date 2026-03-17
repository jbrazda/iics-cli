package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// SecurityLog represents a security log entry.
type SecurityLog struct {
	ID             string `json:"id"`
	OrgID          string `json:"orgId,omitempty"`
	Actor          string `json:"actor"`
	EntryTime      string `json:"entryTime"`
	ObjectID       string `json:"objectId,omitempty"`
	ObjectName     string `json:"objectName,omitempty"`
	ActionCategory string `json:"actionCategory,omitempty"`
	ActionEvent    string `json:"actionEvent,omitempty"`
}

// securityLogListResponse is the API wrapper returned by GET /public/core/v3/securityLog.
type securityLogListResponse struct {
	Entries []SecurityLog `json:"entries"`
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

	var filters []string
	if opts.StartTime != "" {
		filters = append(filters, fmt.Sprintf(`entryTime>="%s"`, opts.StartTime))
	}
	if opts.EndTime != "" {
		filters = append(filters, fmt.Sprintf(`entryTime<="%s"`, opts.EndTime))
	}
	if len(filters) > 0 {
		query["q"] = strings.Join(filters, ";")
	}

	if opts.Limit > 0 {
		query["limit"] = strconv.Itoa(opts.Limit)
	}
	if opts.Skip > 0 {
		query["skip"] = strconv.Itoa(opts.Skip)
	}

	var wrapper securityLogListResponse
	if err := c.doJSONWithQuery(ctx, http.MethodGet, BaseAPIPathV3+"/securityLog", query, nil, &wrapper); err != nil {
		return nil, err
	}
	return wrapper.Entries, nil
}
