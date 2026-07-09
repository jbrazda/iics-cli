package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
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
	if err := c.ensureSession(ctx); err != nil {
		return nil, err
	}
	base, err := c.publishBaseURL(caiURL, assetPaths)
	if err != nil {
		return nil, err
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
	c.setPublishJobBase(resp.Data.ID, base)
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
	if err := c.ensureSession(ctx); err != nil {
		return nil, err
	}
	base := caiURL
	if base == "" {
		base = c.getPublishJobBase(jobID)
	}
	if base == "" {
		base = c.CAIURL()
	}
	if base == "" {
		base = trimSaaSPath(c.BaseAPIURL())
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

// publishBaseURL selects the correct host for a publish/unpublish request.
//
// Per Informatica KB "HOW-TO: Publish a task flow using Rest API in CDI",
// TaskFlow publishing must go to <baseApiUrl>/active-bpel/asset/v1/publish
// with the "/saas" path segment stripped from baseApiUrl - NOT the CAI host.
// Other publishable asset types (Process, Guide, Connector, etc.) use the
// CAI-specific URL as originally designed in CR-0006.
func (c *Client) publishBaseURL(caiURL string, assetPaths []string) (string, error) {
	if hasOnlyTaskflows(assetPaths) {
		base := trimSaaSPath(c.BaseAPIURL())
		if base == "" {
			return "", fmt.Errorf("base API URL not configured; run 'iics login' to discover baseApiUrl or provide profile baseApiUrl")
		}
		return base, nil
	}
	base := caiURL
	if base == "" {
		base = c.CAIURL()
	}
	if base == "" {
		return "", fmt.Errorf("CAI URL not configured; set caiUrl in profile config, IICS_CAI_URL env var, or --cai-url flag")
	}
	return base, nil
}

func hasOnlyTaskflows(assetPaths []string) bool {
	if len(assetPaths) == 0 {
		return false
	}
	for _, p := range assetPaths {
		if !isTaskflowPath(p) {
			return false
		}
	}
	return true
}

// isTaskflowPath reports whether the given asset path is a TASKFLOW asset.
func isTaskflowPath(p string) bool {
	return strings.HasSuffix(strings.ToUpper(strings.TrimSpace(p)), ".TASKFLOW.XML")
}

// AssetBatchKind identifies which backend a publish/unpublish batch must be
// routed to. CAI assets (connections, connectors, processes, guides, process
// objects) and TaskFlow assets publish through different endpoints and must
// never be combined in the same request batch - doing so causes the runtime
// to return AvrEntryNotFoundFault for the TaskFlow entries.
type AssetBatchKind string

const (
	// AssetBatchKindCAI marks a batch of non-TaskFlow CAI assets, routed to the CAI URL.
	AssetBatchKindCAI AssetBatchKind = "CAI"
	// AssetBatchKindTaskflow marks a batch of TaskFlow assets, routed to the base API URL.
	AssetBatchKindTaskflow AssetBatchKind = "TASKFLOW"
)

// AssetBatch is one homogeneous, size-limited group of asset paths ready to
// submit as a single publish/unpublish request.
type AssetBatch struct {
	Kind  AssetBatchKind
	Paths []string
}

// PartitionAssetsByKind splits assetPaths into CAI assets and TaskFlow assets,
// preserving the relative order within each group.
func PartitionAssetsByKind(assetPaths []string) (caiPaths, taskflowPaths []string) {
	for _, p := range assetPaths {
		if isTaskflowPath(p) {
			taskflowPaths = append(taskflowPaths, p)
		} else {
			caiPaths = append(caiPaths, p)
		}
	}
	return caiPaths, taskflowPaths
}

// SplitPublishBatches partitions assetPaths into CAI and TaskFlow groups, then
// splits each group into batches of at most batchSize so every batch is
// homogeneous and can be routed to the correct backend by publishBaseURL.
//
// Group order (which group's batches are returned first) follows the kind of
// the first entry in assetPaths, so that callers who pre-sort assets into
// dependency order (e.g. CAI assets before TaskFlows for publish, or
// TaskFlows before CAI assets for unpublish) get their intended cross-group
// ordering preserved. Relative order within each group is preserved as-is.
func SplitPublishBatches(assetPaths []string, batchSize int) []AssetBatch {
	caiPaths, taskflowPaths := PartitionAssetsByKind(assetPaths)

	var batches []AssetBatch
	appendCAI := func() {
		for _, b := range SplitIntoBatches(caiPaths, batchSize) {
			batches = append(batches, AssetBatch{Kind: AssetBatchKindCAI, Paths: b})
		}
	}
	appendTaskflow := func() {
		for _, b := range SplitIntoBatches(taskflowPaths, batchSize) {
			batches = append(batches, AssetBatch{Kind: AssetBatchKindTaskflow, Paths: b})
		}
	}

	if len(assetPaths) > 0 && isTaskflowPath(assetPaths[0]) {
		appendTaskflow()
		appendCAI()
	} else {
		appendCAI()
		appendTaskflow()
	}
	return batches
}

// trimSaaSPath strips the path (e.g. "/saas") from a baseApiUrl, leaving just
// scheme+host, per the KB instruction "make sure the /saas is removed from
// the baseApiUrl".
func trimSaaSPath(base string) string {
	if base == "" {
		return ""
	}
	u, err := url.Parse(base)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	u.Path = ""
	u.RawPath = ""
	u.RawQuery = ""
	u.Fragment = ""
	return strings.TrimRight(u.String(), "/")
}

func (c *Client) setPublishJobBase(jobID, base string) {
	if jobID == "" || base == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.publishJobBase[jobID] = base
}

func (c *Client) getPublishJobBase(jobID string) string {
	if jobID == "" {
		return ""
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.publishJobBase[jobID]
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
