package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
)

// UserGroupMember is a user reference as returned within a UserGroup object.
type UserGroupMember struct {
	ID          string `json:"id,omitempty"`
	UserName    string `json:"userName,omitempty"`
	Description string `json:"description,omitempty"`
}

// UserGroup represents an IICS user group.
type UserGroup struct {
	ID            string            `json:"id,omitempty"`
	OrgID         string            `json:"orgId,omitempty"`
	UserGroupName string            `json:"userGroupName"`
	Description   string            `json:"description,omitempty"`
	CreateTime    string            `json:"createTime,omitempty"`
	UpdateTime    string            `json:"updateTime,omitempty"`
	CreatedBy     string            `json:"createdBy,omitempty"`
	UpdatedBy     string            `json:"updatedBy,omitempty"`
	Roles         []UserRole        `json:"roles,omitempty"`
	Users         []UserGroupMember `json:"users,omitempty"`
	CountMembers  int               `json:"countMembers,omitempty"`
	CountRoles    int               `json:"countRoles,omitempty"`
}

// UserGroupListOptions holds query parameters for listing user groups.
type UserGroupListOptions struct {
	Limit int
	Skip  int
	Query string
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
	if opts.Query != "" {
		query["q"] = opts.Query
	}

	var resp []UserGroup
	if err := c.doJSONWithQuery(ctx, http.MethodGet, BaseAPIPathV3+"/userGroups", query, nil, &resp); err != nil {
		return nil, err
	}
	for i := range resp {
		resp[i].CountMembers = len(resp[i].Users)
		resp[i].CountRoles = len(resp[i].Roles)
	}
	return resp, nil
}

// GetUserGroup retrieves a single user group by ID.
func (c *Client) GetUserGroup(ctx context.Context, id string) (*UserGroup, error) {
	var resp UserGroup
	if err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("%s/userGroups/%s", BaseAPIPathV3, id), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreateUserGroup creates a new user group.
func (c *Client) CreateUserGroup(ctx context.Context, group *UserGroup) (*UserGroup, error) {
	var resp UserGroup
	if err := c.doJSON(ctx, http.MethodPost, BaseAPIPathV3+"/userGroups", group, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateUserGroup updates an existing user group.
func (c *Client) UpdateUserGroup(ctx context.Context, id string, group *UserGroup) (*UserGroup, error) {
	var resp UserGroup
	if err := c.doJSON(ctx, http.MethodPut, fmt.Sprintf("%s/userGroups/%s", BaseAPIPathV3, id), group, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteUserGroup deletes a user group by ID.
func (c *Client) DeleteUserGroup(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, fmt.Sprintf("%s/userGroups/%s", BaseAPIPathV3, id), nil, nil)
}
