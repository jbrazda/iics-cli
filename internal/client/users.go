package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
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

// GetUser retrieves a single user by ID by scanning the users list.
// The IICS v3 API does not support GET /users/{id}; only DELETE is allowed on that path.
func (c *Client) GetUser(ctx context.Context, id string) (*User, error) {
	opts := UserListOptions{Limit: 200}
	for {
		users, err := c.ListUsers(ctx, opts)
		if err != nil {
			return nil, err
		}
		for i := range users {
			if users[i].ID == id {
				return &users[i], nil
			}
		}
		if len(users) < opts.Limit {
			break
		}
		opts.Skip += opts.Limit
	}
	return nil, fmt.Errorf("user %q not found", id)
}

// GetUserByName finds a user by exact userName match.
func (c *Client) GetUserByName(ctx context.Context, userName string) (*User, error) {
	opts := UserListOptions{Limit: 200}
	for {
		users, err := c.ListUsers(ctx, opts)
		if err != nil {
			return nil, err
		}
		for i := range users {
			if users[i].UserName == userName {
				return &users[i], nil
			}
		}
		if len(users) < opts.Limit {
			break
		}
		opts.Skip += opts.Limit
	}
	return nil, fmt.Errorf("user %q not found", userName)
}

// SearchUsers returns all users whose userName contains the given substring (case-insensitive).
func (c *Client) SearchUsers(ctx context.Context, substring string) ([]User, error) {
	lower := strings.ToLower(substring)
	opts := UserListOptions{Limit: 200}
	var matches []User
	for {
		users, err := c.ListUsers(ctx, opts)
		if err != nil {
			return nil, err
		}
		for _, u := range users {
			if strings.Contains(strings.ToLower(u.UserName), lower) {
				matches = append(matches, u)
			}
		}
		if len(users) < opts.Limit {
			break
		}
		opts.Skip += opts.Limit
	}
	return matches, nil
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

// ChangePasswordRequest is the request body for the ChangePassword endpoint.
// Set OldPassword when changing your own password.
// Set UserID when an administrator is changing another user's password.
type ChangePasswordRequest struct {
	NewPassword string `json:"newPassword"`
	OldPassword string `json:"oldPassword,omitempty"`
	UserID      string `json:"userId,omitempty"`
}

// ResetPasswordRequest is the request body for the ResetPassword endpoint.
type ResetPasswordRequest struct {
	UserID         string `json:"userId"`
	SecurityAnswer string `json:"securityAnswer"`
	NewPassword    string `json:"newPassword"`
}

// ChangePassword changes a user password.
func (c *Client) ChangePassword(ctx context.Context, req *ChangePasswordRequest) error {
	return c.doJSON(ctx, http.MethodPost, BaseAPIPathV3+"/Users/ChangePassword", req, nil)
}

// ResetPassword resets a user password using the user's security answer.
func (c *Client) ResetPassword(ctx context.Context, req *ResetPasswordRequest) error {
	return c.doJSON(ctx, http.MethodPost, BaseAPIPathV3+"/Users/ResetPassword", req, nil)
}
