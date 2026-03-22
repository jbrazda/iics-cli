package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// publishRequest is the JSON:API request body for start publish/unpublish.
type publishRequest struct {
	Data publishRequestData `json:"data"`
}

type publishRequestData struct {
	Type       string                   `json:"type"`
	Attributes publishRequestAttributes `json:"attributes"`
}

type publishRequestAttributes struct {
	AssetPaths []string `json:"assetPaths"`
}

// PublishJobResponse is the JSON:API response from start publish/unpublish
// and from the status endpoints.
type PublishJobResponse struct {
	Data  PublishJobData `json:"data"`
	Links PublishLinks   `json:"links,omitempty"`
}

// PublishJobData holds the publish/unpublish job fields.
type PublishJobData struct {
	Type       string               `json:"type"`
	ID         string               `json:"id"`
	Attributes PublishJobAttributes `json:"attributes"`
}

// PublishJobStatusDetail holds per-state asset counts from the job.
type PublishJobStatusDetail struct {
	ItemStateSummary map[string]int `json:"itemStateSummary,omitempty"`
}

// PublishItemDetail holds the per-asset result inside a publish/unpublish job status.
type PublishItemDetail struct {
	ItemIndex        int    `json:"itemIndex"`
	ItemGUID         string `json:"itemGUID,omitempty"`
	ItemState        string `json:"itemState,omitempty"`
	ItemStatusDetail string `json:"itemStatusDetail,omitempty"`
	ItemStartDate    string `json:"itemStartDate,omitempty"`
	ItemEndDate      string `json:"itemEndDate,omitempty"`
	AssetPath        string `json:"assetPath,omitempty"`
}

// PublishJobAttributes holds the status and progress fields.
type PublishJobAttributes struct {
	JobState        string                 `json:"jobState"`
	JobStatusDetail PublishJobStatusDetail `json:"jobStatusDetail,omitempty"`
	StartedBy       string                 `json:"startedBy,omitempty"`
	StartDate       string                 `json:"startDate,omitempty"`
	EndDate         string                 `json:"endDate,omitempty"`
	TotalCount      int                    `json:"totalCount,omitempty"`
	ProcessedCount  int                    `json:"processedCount,omitempty"`
	AssetPaths      []string               `json:"assetPaths,omitempty"`
	ItemDetail      []PublishItemDetail    `json:"itemDetail,omitempty"`
}

// PublishLinks holds the self and status link URLs.
// Note: known bug ICAI-41690 - these links may use http:// instead of https://.
// Always construct status URLs manually from the job ID.
type PublishLinks struct {
	Self   string `json:"self,omitempty"`
	Status string `json:"status,omitempty"`
}

// PublishIsTerminal returns true when the jobState is a terminal state.
func PublishIsTerminal(jobState string) bool {
	switch jobState {
	case "COMPLETED", "SUCCESS", "FAILED", "ERROR", "WARNING":
		return true
	}
	return false
}

// PublishIsInProgress returns true when the jobState means still running.
func PublishIsInProgress(jobState string) bool {
	return jobState == "NOT_STARTED" || jobState == "PROCESSING" || jobState == "IN_PROGRESS"
}

// PublishMaxBatchSize is the maximum number of assets per publish/unpublish request.
const PublishMaxBatchSize = 199

// StartPublish submits one batch of CAI asset paths for publishing.
// If caiURL is empty, c.CAIURL() is used (auto-detected from login response).
func (c *Client) StartPublish(ctx context.Context, caiURL string, assetPaths []string) (*PublishJobResponse, error) {
	return c.startPublishOp(ctx, caiURL, "publish", assetPaths)
}

// StartUnpublish submits one batch of CAI asset paths for unpublishing.
func (c *Client) StartUnpublish(ctx context.Context, caiURL string, assetPaths []string) (*PublishJobResponse, error) {
	return c.startPublishOp(ctx, caiURL, "unpublish", assetPaths)
}

func (c *Client) startPublishOp(ctx context.Context, caiURL, opType string, assetPaths []string) (*PublishJobResponse, error) {
	base := caiURL
	if base == "" {
		base = c.CAIURL()
	}
	if base == "" {
		return nil, fmt.Errorf("CAI URL not configured; set caiUrl in profile config, IICS_CAI_URL env var, or --cai-url flag")
	}
	u := strings.TrimRight(base, "/") + "/active-bpel/asset/v1/" + opType
	req := publishRequest{
		Data: publishRequestData{
			Type:       opType,
			Attributes: publishRequestAttributes{AssetPaths: assetPaths},
		},
	}
	var resp PublishJobResponse
	if err := c.doCAIJSON(ctx, http.MethodPost, u, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetPublishStatus retrieves the current status of a publish job.
// Set full=true to retrieve the full job object including asset list.
// Note: do NOT use the status URL from the response (ICAI-41690 http:// bug).
func (c *Client) GetPublishStatus(ctx context.Context, caiURL, jobID string, full bool) (*PublishJobResponse, error) {
	return c.getPublishOpStatus(ctx, caiURL, "publish", jobID, full)
}

// GetUnpublishStatus retrieves the current status of an unpublish job.
func (c *Client) GetUnpublishStatus(ctx context.Context, caiURL, jobID string, full bool) (*PublishJobResponse, error) {
	return c.getPublishOpStatus(ctx, caiURL, "unpublish", jobID, full)
}

func (c *Client) getPublishOpStatus(ctx context.Context, caiURL, opType, jobID string, full bool) (*PublishJobResponse, error) {
	base := caiURL
	if base == "" {
		base = c.CAIURL()
	}
	if base == "" {
		return nil, fmt.Errorf("CAI URL not configured")
	}
	var u string
	if full {
		u = fmt.Sprintf("%s/active-bpel/asset/v1/%s/%s", strings.TrimRight(base, "/"), opType, jobID)
	} else {
		u = fmt.Sprintf("%s/active-bpel/asset/v1/%s/%s/Status", strings.TrimRight(base, "/"), opType, jobID)
	}
	var resp PublishJobResponse
	if err := c.doCAIJSON(ctx, http.MethodGet, u, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// AssetPathFromObject builds a CAI asset path from an Object.
//
// Preferred source: the Location field, which the API returns in the form
// "Explore/<path>.<TYPE>" - appending ".xml" gives the exact path the publish
// API expects. When Location is non-empty it is used directly (+ ".xml").
//
// Fallback: if Location is empty, the path is built from Path + Type as
// "Explore/<path>.<TYPE>.xml". Returns an error for non-publishable types.
func AssetPathFromObject(obj Object) (string, error) {
	if obj.Location != "" {
		return obj.Location + ".xml", nil
	}
	switch obj.Type {
	case "AI_SERVICE_CONNECTOR", "AI_CONNECTION", "PROCESS", "GUIDE", "TASKFLOW",
		"DTEMPLATE", "PROCESS_OBJECT":
		return fmt.Sprintf("Explore/%s.%s.xml", obj.Path, obj.Type), nil
	default:
		return "", fmt.Errorf("asset type %q is not publishable", obj.Type)
	}
}

// SplitIntoBatches splits a slice of asset paths into batches of at most batchSize.
func SplitIntoBatches(paths []string, batchSize int) [][]string {
	var batches [][]string
	for len(paths) > 0 {
		end := batchSize
		if end > len(paths) {
			end = len(paths)
		}
		batches = append(batches, paths[:end])
		paths = paths[end:]
	}
	return batches
}
