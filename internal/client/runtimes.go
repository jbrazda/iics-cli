package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
)

// RuntimeEnvironment represents an IICS runtime environment.
type RuntimeEnvironment struct {
	ID          string `json:"id,omitempty"`
	OrgID       string `json:"orgId,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	CreateTime  string `json:"createTime,omitempty"`
	UpdateTime  string `json:"updateTime,omitempty"`
	CreatedBy   string `json:"createdBy,omitempty"`
	UpdatedBy   string `json:"updatedBy,omitempty"`
	Status      string `json:"status,omitempty"`
	Type        string `json:"type,omitempty"`
}

// RuntimeListOptions holds query parameters for listing runtimes.
type RuntimeListOptions struct {
	Limit int
	Skip  int
}

// ListRuntimeEnvironments retrieves runtime environments.
func (c *Client) ListRuntimeEnvironments(ctx context.Context, opts RuntimeListOptions) ([]RuntimeEnvironment, error) {
	query := make(map[string]string)
	if opts.Limit > 0 {
		query["limit"] = strconv.Itoa(opts.Limit)
	}
	if opts.Skip > 0 {
		query["skip"] = strconv.Itoa(opts.Skip)
	}

	var resp []RuntimeEnvironment
	if err := c.doJSONWithQuery(ctx, http.MethodGet, "public/core/v3/runtimeEnvironments", query, nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetRuntimeEnvironment retrieves a single runtime environment by ID.
func (c *Client) GetRuntimeEnvironment(ctx context.Context, id string) (*RuntimeEnvironment, error) {
	var resp RuntimeEnvironment
	if err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("public/core/v3/runtimeEnvironments/%s", id), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreateRuntimeEnvironment creates a new runtime environment.
func (c *Client) CreateRuntimeEnvironment(ctx context.Context, rt *RuntimeEnvironment) (*RuntimeEnvironment, error) {
	var resp RuntimeEnvironment
	if err := c.doJSON(ctx, http.MethodPost, "public/core/v3/runtimeEnvironments", rt, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateRuntimeEnvironment updates an existing runtime environment.
func (c *Client) UpdateRuntimeEnvironment(ctx context.Context, id string, rt *RuntimeEnvironment) (*RuntimeEnvironment, error) {
	var resp RuntimeEnvironment
	if err := c.doJSON(ctx, http.MethodPut, fmt.Sprintf("public/core/v3/runtimeEnvironments/%s", id), rt, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
