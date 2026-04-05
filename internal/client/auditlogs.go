package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
)

// AuditLog represents a single audit log entry returned by GET /api/v2/auditlog.
type AuditLog struct {
	ID           string `json:"id,omitempty"`
	Version      int    `json:"version,omitempty"`
	OrgID        string `json:"orgId,omitempty"`
	Username     string `json:"username,omitempty"`
	EntryTime    string `json:"entryTime,omitempty"`
	EntryTimeUTC string `json:"entryTimeUTC,omitempty"`
	ObjectID     string `json:"objectId,omitempty"`
	ObjectName   string `json:"objectName,omitempty"`
	Category     string `json:"category,omitempty"`
	Event        string `json:"event,omitempty"`
	EventParam   string `json:"eventParam,omitempty"`
	Message      string `json:"message,omitempty"`
}

// AuditLogListOptions holds optional query parameters for listing audit logs.
// Limit maps to the API batchSize parameter; Skip maps to the API batchId parameter.
type AuditLogListOptions struct {
	Limit int // 0 = not set; API returns most recent 200 when omitted
	Skip  int // 0-based batch number; only sent when Limit > 0
}

// ListAuditLogs retrieves audit log entries from the V2 auditlog endpoint.
// When opts.Limit is 0, no pagination parameters are sent and the API returns
// the most recent 200 entries.
func (c *Client) ListAuditLogs(ctx context.Context, opts AuditLogListOptions) ([]AuditLog, error) {
	query := make(map[string]string)
	if opts.Limit > 0 {
		query["batchSize"] = strconv.Itoa(opts.Limit)
		query["batchId"] = strconv.Itoa(opts.Skip)
	}
	var resp []AuditLog
	if err := c.doJSONWithQuery(ctx, http.MethodGet, fmt.Sprintf("%s/auditlog", BaseAPIPathV2), query, nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}
