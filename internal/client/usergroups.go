package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
)

// UserGroup represents an IICS user group.
type UserGroup struct {
	ID          string   `json:"id,omitempty"`
	OrgID       string   `json:"orgId,omitempty"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	CreateTime  string   `json:"createTime,omitempty"`
	UpdateTime  string   `json:"updateTime,omitempty"`
	CreatedBy   string   `json:"createdBy,omitempty"`
	UpdatedBy   string   `json:"updatedBy,omitempty"`
	Roles       []string `json:"roles,omitempty"`
}

// UserGroupListOptions holds query parameters for listing user groups.
type UserGroupListOptions struct {
	Limit int
	Skip  int
}

// ListUserGroups retrieves user groups.
func (c *Client) ListUserGroups(ctx context.Context, opts UserGroupListOptions) ([]UserGroup, error) {
	query := make(map[string]string)
	if opts.Limit > 0 {
		query["limit"] = strconv.Itoa(opts.Limit)
	}
	if opts.Skip > 0 {
		query["skip"] = strconv.Itoa(opts.Skip)
	}

	var resp []UserGroup
	if err := c.doJSONWithQuery(ctx, http.MethodGet, "public/core/v3/userGroups", query, nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetUserGroup retrieves a single user group by ID.
func (c *Client) GetUserGroup(ctx context.Context, id string) (*UserGroup, error) {
	var resp UserGroup
	if err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("public/core/v3/userGroups/%s", id), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreateUserGroup creates a new user group.
func (c *Client) CreateUserGroup(ctx context.Context, group *UserGroup) (*UserGroup, error) {
	var resp UserGroup
	if err := c.doJSON(ctx, http.MethodPost, "public/core/v3/userGroups", group, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateUserGroup updates an existing user group.
func (c *Client) UpdateUserGroup(ctx context.Context, id string, group *UserGroup) (*UserGroup, error) {
	var resp UserGroup
	if err := c.doJSON(ctx, http.MethodPut, fmt.Sprintf("public/core/v3/userGroups/%s", id), group, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteUserGroup deletes a user group by ID.
func (c *Client) DeleteUserGroup(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, fmt.Sprintf("public/core/v3/userGroups/%s", id), nil, nil)
}
