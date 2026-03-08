package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

const defaultPageSize = 200

// Object represents an IICS asset.
type Object struct {
	ID          string   `json:"id"`
	Path        string   `json:"path"`
	Type        string   `json:"type"`
	Description string   `json:"description"`
	UpdatedBy   string   `json:"updatedBy"`
	UpdateTime  string   `json:"updateTime"`
	Tags        []string `json:"tags,omitempty"`
}

// ObjectsListOptions holds query parameters for listing objects.
type ObjectsListOptions struct {
	Query string
	Type  string
	Tag   string
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
	if opts.Tag != "" {
		filters = append(filters, fmt.Sprintf("tag=='%s'", opts.Tag))
	}
	if len(filters) > 0 {
		query["q"] = strings.Join(filters, " and ")
	}

	if opts.Limit > 0 {
		query["limit"] = strconv.Itoa(opts.Limit)
	}
	if opts.Skip > 0 {
		query["skip"] = strconv.Itoa(opts.Skip)
	}

	var resp ObjectsListResponse
	if err := c.doJSONWithQuery(ctx, http.MethodGet, "public/core/v3/objects", query, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListAllObjects retrieves all assets matching the filter options by paging
// through results with the default page size (200) until exhausted.
// opts.Limit and opts.Skip are ignored; use ListObjects for single-page control.
func (c *Client) ListAllObjects(ctx context.Context, opts ObjectsListOptions) (*ObjectsListResponse, error) {
	all := &ObjectsListResponse{}
	skip := 0

	for {
		page, err := c.ListObjects(ctx, ObjectsListOptions{
			Query: opts.Query,
			Type:  opts.Type,
			Tag:   opts.Tag,
			Limit: defaultPageSize,
			Skip:  skip,
		})
		if err != nil {
			return nil, err
		}

		all.Objects = append(all.Objects, page.Objects...)
		all.Count = len(all.Objects)

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
	path := fmt.Sprintf("public/core/v3/objects/%s/references", objectID)
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
