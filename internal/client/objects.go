package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

const defaultPageSize = 200

// ObjectSourceControl holds source control metadata for an asset.
type ObjectSourceControl struct {
	CheckedOutBy     string `json:"checkedOutBy,omitempty"`
	CheckedOutTime   string `json:"checkedOutTime,omitempty"`
	Hash             string `json:"hash,omitempty"`
	LastCheckinBy    string `json:"lastCheckinBy,omitempty"`
	LastCheckinTime  string `json:"lastCheckinTime,omitempty"`
	LastPullTime     string `json:"lastPullTime,omitempty"`
	SourceControlled bool   `json:"sourceControlled,omitempty"`
}

// ObjectCustomAttributes holds Application Integration publication metadata.
type ObjectCustomAttributes struct {
	PublishedBy     string `json:"publishedBy,omitempty"`
	PublicationDate string `json:"publicationDate,omitempty"`
}

// Object represents an IICS asset.
type Object struct {
	ID               string                  `json:"id"`
	Path             string                  `json:"path"`
	Type             string                  `json:"type"`
	Description      string                  `json:"description"`
	UpdatedBy        string                  `json:"updatedBy"`
	UpdateTime       string                  `json:"updateTime"`
	Tags             []string                `json:"tags,omitempty"`
	SourceControl    *ObjectSourceControl    `json:"sourceControl,omitempty"`
	CustomAttributes *ObjectCustomAttributes `json:"customAttributes,omitempty"`
	Location         string                  `json:"location,omitempty"` // computed: "Explore/<path>.<type>"
}

// ObjectsListOptions holds query parameters for listing objects.
type ObjectsListOptions struct {
	Query string
	Type  string
	Limit int
	Skip  int
}

// ObjectsListResponse is the response from listing objects.
type ObjectsListResponse struct {
	Count   int      `json:"count"`
	Objects []Object `json:"objects"`
}

// ObjectReference is used in dependency lookups.
type ObjectReference struct {
	AppContextID string `json:"appContextId"`
	Path         string `json:"path"`
	Type         string `json:"type"`
	UpdatedBy    string `json:"updatedBy"`
	UpdateTime   string `json:"updateTime"`
}

// ObjectDependenciesResponse holds object dependency results.
type ObjectDependenciesResponse struct {
	Uses   []ObjectReference `json:"uses,omitempty"`
	UsedBy []ObjectReference `json:"usedBy,omitempty"`
}

// ListObjects retrieves organization assets with optional filtering.
func (c *Client) ListObjects(ctx context.Context, opts ObjectsListOptions) (*ObjectsListResponse, error) {
	query := make(map[string]string)

	// Build query filter
	filters := []string{}
	if opts.Query != "" {
		filters = append(filters, opts.Query)
	}
	if opts.Type != "" {
		filters = append(filters, fmt.Sprintf("type=='%s'", opts.Type))
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

	var resp ObjectsListResponse
	if err := c.doJSONWithQuery(ctx, http.MethodGet, BaseAPIPathV3+"/objects", query, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListAllObjects retrieves all assets matching the filter options by paging
// through results with the default page size (200) until exhausted.
// opts.Limit and opts.Skip are ignored; use ListObjects for single-page control.
// progressFn, if non-nil, is called after each page with the page number (1-based) and total fetched count.
func (c *Client) ListAllObjects(ctx context.Context, opts ObjectsListOptions, progressFn func(page, fetched int)) (*ObjectsListResponse, error) {
	all := &ObjectsListResponse{}
	skip := 0
	pageNum := 0

	for {
		page, err := c.ListObjects(ctx, ObjectsListOptions{
			Query: opts.Query,
			Type:  opts.Type,
			Limit: defaultPageSize,
			Skip:  skip,
		})
		if err != nil {
			return nil, err
		}

		pageNum++
		all.Objects = append(all.Objects, page.Objects...)
		all.Count = len(all.Objects)

		if progressFn != nil {
			progressFn(pageNum, all.Count)
		}

		if len(page.Objects) < defaultPageSize {
			// Last page — no more results
			break
		}
		skip += defaultPageSize
	}

	return all, nil
}

// GetObjectDependencies finds uses/usedBy references for an object.
func (c *Client) GetObjectDependencies(ctx context.Context, objectID string, refType string, limit, skip int) (*ObjectDependenciesResponse, error) {
	path := fmt.Sprintf("%s/objects/%s/references", BaseAPIPathV3, objectID)
	query := make(map[string]string)

	if refType != "" {
		query["refType"] = refType
	}
	if limit > 0 {
		query["limit"] = strconv.Itoa(limit)
	}
	if skip > 0 {
		query["skip"] = strconv.Itoa(skip)
	}

	var resp ObjectDependenciesResponse
	if err := c.doJSONWithQuery(ctx, http.MethodGet, path, query, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
