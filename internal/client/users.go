package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
)

// User represents an IICS user.
type User struct {
	ID             string   `json:"id,omitempty"`
	OrgID          string   `json:"orgId,omitempty"`
	Name           string   `json:"name"`
	Description    string   `json:"description,omitempty"`
	CreateTime     string   `json:"createTime,omitempty"`
	UpdateTime     string   `json:"updateTime,omitempty"`
	CreatedBy      string   `json:"createdBy,omitempty"`
	UpdatedBy      string   `json:"updatedBy,omitempty"`
	FirstName      string   `json:"firstName,omitempty"`
	LastName       string   `json:"lastName,omitempty"`
	Email          string   `json:"emails,omitempty"`
	Phone          string   `json:"phone,omitempty"`
	Title          string   `json:"title,omitempty"`
	Status         string   `json:"status,omitempty"`
	Authentication string   `json:"authentication,omitempty"`
	Timezone       string   `json:"timezone,omitempty"`
	Roles          []string `json:"roles,omitempty"`
	Groups         []string `json:"groups,omitempty"`
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
	if err := c.doJSONWithQuery(ctx, http.MethodGet, "public/core/v3/users", query, nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetUser retrieves a single user by ID.
func (c *Client) GetUser(ctx context.Context, id string) (*User, error) {
	var resp User
	if err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("public/core/v3/users/%s", id), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreateUser creates a new user.
func (c *Client) CreateUser(ctx context.Context, user *User) (*User, error) {
	var resp User
	if err := c.doJSON(ctx, http.MethodPost, "public/core/v3/users", user, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateUser updates an existing user.
func (c *Client) UpdateUser(ctx context.Context, id string, user *User) (*User, error) {
	var resp User
	if err := c.doJSON(ctx, http.MethodPut, fmt.Sprintf("public/core/v3/users/%s", id), user, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteUser deletes a user by ID.
func (c *Client) DeleteUser(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, fmt.Sprintf("public/core/v3/users/%s", id), nil, nil)
}
