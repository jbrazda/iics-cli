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
	ChecksumValid bool      `json:"checksumValid"`
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

// ImportObjectRef is a source or target object reference within an import job object entry.
type ImportObjectRef struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// ImportJobObject is a single entry in the ImportJob.Objects list.
// The import status API returns a nested sourceObject/targetObject structure,
// unlike the export API which returns flat JobObject entries.
type ImportJobObject struct {
	SourceObject ImportObjectRef `json:"sourceObject"`
	TargetObject ImportObjectRef `json:"targetObject"`
	Status       JobStatus       `json:"status"`
}

// FlatImportObject is a display-ready flattened view of an ImportJobObject.
// All fields from sourceObject, targetObject, and status are exposed so that
// any combination can be selected via --object-status-fields.
type FlatImportObject struct {
	SourceID          string `json:"sourceId"`
	SourceName        string `json:"sourceName"`
	SourcePath        string `json:"sourcePath"`
	SourceType        string `json:"sourceType"`
	SourceDescription string `json:"sourceDescription"`
	TargetID          string `json:"targetId"`
	TargetName        string `json:"targetName"`
	TargetPath        string `json:"targetPath"`
	TargetType        string `json:"targetType"`
	TargetDescription string `json:"targetDescription"`
	State             string `json:"state"`
	Message           string `json:"message"`
}

// FlattenImportObjects converts a slice of ImportJobObject to a flat display slice.
func FlattenImportObjects(objects []ImportJobObject) []FlatImportObject {
	flat := make([]FlatImportObject, len(objects))
	for i, o := range objects {
		flat[i] = FlatImportObject{
			SourceID:          o.SourceObject.ID,
			SourceName:        o.SourceObject.Name,
			SourcePath:        o.SourceObject.Path,
			SourceType:        o.SourceObject.Type,
			SourceDescription: o.SourceObject.Description,
			TargetID:          o.TargetObject.ID,
			TargetName:        o.TargetObject.Name,
			TargetPath:        o.TargetObject.Path,
			TargetType:        o.TargetObject.Type,
			TargetDescription: o.TargetObject.Description,
			State:             o.Status.State,
			Message:           o.Status.Message,
		}
	}
	return flat
}

// ImportJob represents an import job.
type ImportJob struct {
	ID         string            `json:"id"`
	CreateTime string            `json:"createTime,omitempty"`
	Name       string            `json:"name"`
	Status     JobStatus         `json:"status"`
	StartTime  string            `json:"startTime,omitempty"`
	EndTime    string            `json:"endTime,omitempty"`
	Objects    []ImportJobObject `json:"objects,omitempty"`
}

// UploadImportPackage uploads a ZIP package for import.
func (c *Client) UploadImportPackage(ctx context.Context, filename string, reader io.Reader) (*ImportUploadResponse, error) {
	if err := c.ensureSession(ctx); err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("package", filename)
	if err != nil {
		return nil, fmt.Errorf("creating form file: %w", err)
	}

	_, err = io.Copy(part, reader)
	if err != nil {
		return nil, fmt.Errorf("copying file data: %w", err)
	}

	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("closing multipart writer: %w", err)
	}

	url := c.apiURL(fmt.Sprintf("%s/import/package", BaseAPIPathV3))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &buf)
	if err != nil {
		return nil, fmt.Errorf("creating upload request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

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
	if err := c.doJSON(ctx, http.MethodPost, fmt.Sprintf("%s/import/%s", BaseAPIPathV3, jobID), req, &resp); err != nil {
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
	if err := c.doJSONWithQuery(ctx, http.MethodGet, fmt.Sprintf("%s/import/%s", BaseAPIPathV3, jobID), query, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DownloadImportLog downloads the import job log.
func (c *Client) DownloadImportLog(ctx context.Context, jobID string, dest io.Writer) error {
	path := fmt.Sprintf("%s/import/%s/log", BaseAPIPathV3, jobID)
	body, err := c.doRaw(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	defer func() { _ = body.Close() }()

	if _, err := io.Copy(dest, body); err != nil {
		return fmt.Errorf("downloading import log: %w", err)
	}
	return nil
}
