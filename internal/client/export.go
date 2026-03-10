package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

// ExportObject specifies an object to include in an export.
type ExportObject struct {
	ID                  string `json:"id"`
	IncludeDependencies bool   `json:"includeDependencies,omitempty"`
}

// ExportRequest is the request body for creating an export job.
type ExportRequest struct {
	Name    string         `json:"name"`
	Objects []ExportObject `json:"objects"`
}

// JobStatus represents the status of an async job.
type JobStatus struct {
	State   string `json:"state"`
	Message string `json:"message,omitempty"`
}

// JobObject represents an object within a job.
type JobObject struct {
	ID     string    `json:"id"`
	Name   string    `json:"name"`
	Type   string    `json:"type"`
	Status JobStatus `json:"status"`
}

// ExportJob represents an export job.
type ExportJob struct {
	ID         string      `json:"id"`
	CreateTime string      `json:"createTime,omitempty"`
	Name       string      `json:"name"`
	Status     JobStatus   `json:"status"`
	StartTime  string      `json:"startTime,omitempty"`
	EndTime    string      `json:"endTime,omitempty"`
	Objects    []JobObject `json:"objects,omitempty"`
}

// CreateExport starts an export job.
func (c *Client) CreateExport(ctx context.Context, req *ExportRequest) (*ExportJob, error) {
	var resp ExportJob
	if err := c.doJSON(ctx, http.MethodPost, fmt.Sprintf("%s/export", BaseAPIPathV3), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetExportStatus retrieves the status of an export job.
func (c *Client) GetExportStatus(ctx context.Context, jobID string, expand bool) (*ExportJob, error) {
	query := make(map[string]string)
	if expand {
		query["expand"] = "objects"
	}

	var resp ExportJob
	if err := c.doJSONWithQuery(ctx, http.MethodGet, fmt.Sprintf("%s/export/%s", BaseAPIPathV3, jobID), query, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DownloadExportPackage downloads the export ZIP package.
func (c *Client) DownloadExportPackage(ctx context.Context, jobID string, dest io.Writer) error {
	path := fmt.Sprintf("%s/export/%s/package", BaseAPIPathV3, jobID)
	body, err := c.doRaw(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	defer body.Close()

	if _, err := io.Copy(dest, body); err != nil {
		return fmt.Errorf("downloading export package: %w", err)
	}
	return nil
}
