package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
)

// Role represents an IICS role.
type Role struct {
	ID          string   `json:"id,omitempty"`
	OrgID       string   `json:"orgId,omitempty"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	CreateTime  string   `json:"createTime,omitempty"`
	UpdateTime  string   `json:"updateTime,omitempty"`
	CreatedBy   string   `json:"createdBy,omitempty"`
	UpdatedBy   string   `json:"updatedBy,omitempty"`
	SystemRole  bool     `json:"systemRole,omitempty"`
	Privileges  []string `json:"privileges,omitempty"`
}

// RoleListOptions holds query parameters for listing roles.
type RoleListOptions struct {
	Limit int
	Skip  int
}

// ListRoles retrieves roles.
func (c *Client) ListRoles(ctx context.Context, opts RoleListOptions) ([]Role, error) {
	query := make(map[string]string)
	if opts.Limit > 0 {
		query["limit"] = strconv.Itoa(opts.Limit)
	}
	if opts.Skip > 0 {
		query["skip"] = strconv.Itoa(opts.Skip)
	}

	var resp []Role
	if err := c.doJSONWithQuery(ctx, http.MethodGet, "roles", query, nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetRole retrieves a single role by ID.
func (c *Client) GetRole(ctx context.Context, id string) (*Role, error) {
	var resp Role
	if err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("roles/%s", id), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreateRole creates a new role.
func (c *Client) CreateRole(ctx context.Context, role *Role) (*Role, error) {
	var resp Role
	if err := c.doJSON(ctx, http.MethodPost, "roles", role, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateRole updates an existing role.
func (c *Client) UpdateRole(ctx context.Context, id string, role *Role) (*Role, error) {
	var resp Role
	if err := c.doJSON(ctx, http.MethodPut, fmt.Sprintf("roles/%s", id), role, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteRole deletes a role by ID.
func (c *Client) DeleteRole(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, fmt.Sprintf("roles/%s", id), nil, nil)
}
