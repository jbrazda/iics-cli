package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
)

// ImportUploadResponse is the response from uploading an import package.
type ImportUploadResponse struct {
	JobID         string    `json:"jobId"`
	JobStatus     JobStatus `json:"jobStatus"`
	ChecksumValid bool     `json:"checksumValid"`
}

// ImportSpecification defines import parameters.
type ImportSpecification struct {
	DefaultConflictResolution string                `json:"defaultConflictResolution"`
	ObjectSpecification       []ObjectSpecification `json:"objectSpecification,omitempty"`
}

// ObjectSpecification defines per-object import parameters.
type ObjectSpecification struct {
	SourceObjectID     string `json:"sourceObjectId"`
	ConflictResolution string `json:"conflictResolution"`
}

// ImportStartRequest is the request body for starting an import job.
type ImportStartRequest struct {
	Name                string              `json:"name"`
	ImportSpecification ImportSpecification `json:"importSpecification"`
}

// ImportJob represents an import job.
type ImportJob struct {
	ID         string      `json:"id"`
	CreateTime string      `json:"createTime,omitempty"`
	Name       string      `json:"name"`
	Status     JobStatus   `json:"status"`
	StartTime  string      `json:"startTime,omitempty"`
	EndTime    string      `json:"endTime,omitempty"`
	Objects    []JobObject `json:"objects,omitempty"`
}

// UploadImportPackage uploads a ZIP package for import.
func (c *Client) UploadImportPackage(ctx context.Context, filename string, reader io.Reader) (*ImportUploadResponse, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("package", filename)
	if err != nil {
		return nil, fmt.Errorf("creating form file: %w", err)
	}

	if _, err := io.Copy(part, reader); err != nil {
		return nil, fmt.Errorf("copying file data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("closing multipart writer: %w", err)
	}

	url := c.apiURL("public/core/v3/import/package")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &buf)
	if err != nil {
		return nil, fmt.Errorf("creating upload request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading upload response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, newAPIError(resp, respData)
	}

	var uploadResp ImportUploadResponse
	if err := parseJSON(respData, &uploadResp); err != nil {
		return nil, fmt.Errorf("parsing upload response: %w", err)
	}

	return &uploadResp, nil
}

// StartImport starts an import job.
func (c *Client) StartImport(ctx context.Context, jobID string, req *ImportStartRequest) (*ImportJob, error) {
	var resp ImportJob
	if err := c.doJSON(ctx, http.MethodPost, fmt.Sprintf("public/core/v3/import/%s", jobID), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetImportStatus retrieves the status of an import job.
func (c *Client) GetImportStatus(ctx context.Context, jobID string, expand bool) (*ImportJob, error) {
	query := make(map[string]string)
	if expand {
		query["expand"] = "objects"
	}

	var resp ImportJob
	if err := c.doJSONWithQuery(ctx, http.MethodGet, fmt.Sprintf("public/core/v3/import/%s", jobID), query, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DownloadImportLog downloads the import job log.
func (c *Client) DownloadImportLog(ctx context.Context, jobID string, dest io.Writer) error {
	path := fmt.Sprintf("public/core/v3/import/%s/log", jobID)
	body, err := c.doRaw(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	defer body.Close()

	if _, err := io.Copy(dest, body); err != nil {
		return fmt.Errorf("downloading import log: %w", err)
	}
	return nil
}
