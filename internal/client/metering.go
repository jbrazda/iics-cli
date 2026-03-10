package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

// MeteringData represents metering/usage data.
type MeteringData struct {
	ID        string                 `json:"id,omitempty"`
	OrgID     string                 `json:"orgId,omitempty"`
	Type      string                 `json:"type,omitempty"`
	StartDate string                 `json:"startDate,omitempty"`
	EndDate   string                 `json:"endDate,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// GetMeteringData retrieves metering data.
func (c *Client) GetMeteringData(ctx context.Context, meteringType, startDate, endDate string) (*MeteringData, error) {
	query := make(map[string]string)
	if meteringType != "" {
		query["type"] = meteringType
	}
	if startDate != "" {
		query["startDate"] = startDate
	}
	if endDate != "" {
		query["endDate"] = endDate
	}

	var resp MeteringData
	if err := c.doJSONWithQuery(ctx, http.MethodGet, BaseAPIPathV3+"/metering", query, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DownloadMeteringReport downloads a metering report.
func (c *Client) DownloadMeteringReport(ctx context.Context, reportID string, dest io.Writer) error {
	path := fmt.Sprintf("%s/metering/%s/report", BaseAPIPathV3, reportID)
	body, err := c.doRaw(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	defer body.Close()

	if _, err := io.Copy(dest, body); err != nil {
		return fmt.Errorf("downloading metering report: %w", err)
	}
	return nil
}
