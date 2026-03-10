package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
)

// UserRole is a role reference as returned within a User object.
type UserRole struct {
	ID                 string `json:"id,omitempty"`
	RoleName           string `json:"roleName,omitempty"`
	Description        string `json:"description,omitempty"`
	DisplayName        string `json:"displayName,omitempty"`
	DisplayDescription string `json:"displayDescription,omitempty"`
}

// UserGroupRef is a group reference as returned within a User object.
type UserGroupRef struct {
	ID            string `json:"id,omitempty"`
	UserGroupName string `json:"userGroupName,omitempty"`
	Description   string `json:"description,omitempty"`
}

// User represents an IICS user.
type User struct {
	ID                  string         `json:"id,omitempty"`
	OrgID               string         `json:"orgId,omitempty"`
	UserName            string         `json:"userName,omitempty"`
	FirstName           string         `json:"firstName,omitempty"`
	LastName            string         `json:"lastName,omitempty"`
	Description         string         `json:"description,omitempty"`
	CreateTime          string         `json:"createTime,omitempty"`
	UpdateTime          string         `json:"updateTime,omitempty"`
	CreatedBy           string         `json:"createdBy,omitempty"`
	UpdatedBy           string         `json:"updatedBy,omitempty"`
	Email               string         `json:"email,omitempty"`
	Phone               string         `json:"phone,omitempty"`
	Title               string         `json:"title,omitempty"`
	State               string         `json:"state,omitempty"`
	Authentication      string         `json:"authentication,omitempty"`
	TimeZoneID          string         `json:"timeZoneId,omitempty"`
	ForcePasswordChange bool           `json:"forcePasswordChange,omitempty"`
	LastLoginTime       string         `json:"lastLoginTime,omitempty"`
	LastLoginMode       string         `json:"lastLoginMode,omitempty"`
	MaxLoginAttempts    string         `json:"maxLoginAttempts,omitempty"`
	Roles               []UserRole     `json:"roles,omitempty"`
	Groups              []UserGroupRef `json:"groups,omitempty"`
}

// UserListOptions holds query parameters for listing users.
type UserListOptions struct {
	Limit int
	Skip  int
}

// ListUsers retrieves users.
func (c *Client) ListUsers(ctx context.Context, opts UserListOptions) ([]User, error) {
	query := make(map[string]string)
	if opts.Limit > 0 {
		query["limit"] = strconv.Itoa(opts.Limit)
	}
	if opts.Skip > 0 {
		query["skip"] = strconv.Itoa(opts.Skip)
	}

	var resp []User
	if err := c.doJSONWithQuery(ctx, http.MethodGet, BaseAPIPathV3+"/users", query, nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetUser retrieves a single user by ID.
func (c *Client) GetUser(ctx context.Context, id string) (*User, error) {
	var resp User
	if err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("%s/users/%s", BaseAPIPathV3, id), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreateUser creates a new user.
func (c *Client) CreateUser(ctx context.Context, user *User) (*User, error) {
	var resp User
	if err := c.doJSON(ctx, http.MethodPost, BaseAPIPathV3+"/users", user, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateUser updates an existing user.
func (c *Client) UpdateUser(ctx context.Context, id string, user *User) (*User, error) {
	var resp User
	if err := c.doJSON(ctx, http.MethodPut, fmt.Sprintf("%s/users/%s", BaseAPIPathV3, id), user, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteUser deletes a user by ID.
func (c *Client) DeleteUser(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, fmt.Sprintf("%s/users/%s", BaseAPIPathV3, id), nil, nil)
}
