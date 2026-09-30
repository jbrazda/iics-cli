package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// createUserRequest is the POST body for creating a user.
// Field names and types match the IICS v3 API exactly.
type createUserRequest struct {
	Name                string   `json:"name"`
	FirstName           string   `json:"firstName,omitempty"`
	LastName            string   `json:"lastName,omitempty"`
	Email               string   `json:"email,omitempty"`
	Phone               string   `json:"phone,omitempty"`
	Title               string   `json:"title,omitempty"`
	Description         string   `json:"description,omitempty"`
	Authentication      int      `json:"authentication"`
	AliasName           string   `json:"aliasName,omitempty"`
	ForcePasswordChange bool     `json:"forcePasswordChange,omitempty"`
	Roles               []string `json:"roles,omitempty"`
	Groups              []string `json:"groups,omitempty"`
}

// userToCreateRequest converts a User (as populated by the wizard or file parser) into
// the shape the API expects for POST /users.
func userToCreateRequest(u *User) *createUserRequest {
	req := &createUserRequest{
		Name:                u.UserName,
		FirstName:           u.FirstName,
		LastName:            u.LastName,
		Email:               u.Email,
		Phone:               u.Phone,
		Title:               u.Title,
		Description:         u.Description,
		ForcePasswordChange: u.ForcePasswordChange,
	}
	if strings.EqualFold(u.Authentication, "SSO") {
		req.Authentication = 1 // 0 = Native (default)
		req.AliasName = u.AliasName
	}
	for _, g := range u.Groups {
		req.Groups = append(req.Groups, g.ID)
	}
	for _, r := range u.Roles {
		req.Roles = append(req.Roles, r.ID)
	}
	return req
}

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
	AliasName           string         `json:"aliasName,omitempty"`
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
	// Query is the q filter, e.g. userName==jdoe@acme.com. The API supports
	// exact matches on userId and userName only.
	Query string
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
	if opts.Query != "" {
		query["q"] = opts.Query
	}

	var resp []User
	if err := c.doJSONWithQuery(ctx, http.MethodGet, BaseAPIPathV3+"/users", query, nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetUser retrieves a single user by ID with the q=userId== filter.
// The IICS v3 API does not support GET /users/{id}; only DELETE is allowed on that path.
func (c *Client) GetUser(ctx context.Context, id string) (*User, error) {
	users, err := c.ListUsers(ctx, UserListOptions{Query: "userId==" + id, Limit: 1})
	if err != nil {
		return nil, err
	}
	for i := range users {
		if users[i].ID == id {
			return &users[i], nil
		}
	}
	return nil, fmt.Errorf("user %q not found", id)
}

// GetUserByName finds a user by exact userName (case-insensitive) with the
// q=userName== filter.
func (c *Client) GetUserByName(ctx context.Context, userName string) (*User, error) {
	u, err := c.FindUserByName(ctx, userName)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, fmt.Errorf("user %q not found", userName)
	}
	return u, nil
}

// FindUserByName is GetUserByName but returns nil without an error when no
// user has that name.
func (c *Client) FindUserByName(ctx context.Context, userName string) (*User, error) {
	users, err := c.ListUsers(ctx, UserListOptions{Query: "userName==" + userName, Limit: 1})
	if err != nil {
		return nil, err
	}
	for i := range users {
		if strings.EqualFold(users[i].UserName, userName) {
			return &users[i], nil
		}
	}
	return nil, nil
}

// SearchUsers returns users whose user name, first name, last name, full
// name or email contains text (case-insensitive). The API only filters
// exact matches, so the users are listed page by page and filtered here; an
// organization has at most 1000 users, groups and roles combined.
func (c *Client) SearchUsers(ctx context.Context, text string) ([]User, error) {
	lower := strings.ToLower(strings.TrimSpace(text))
	opts := UserListOptions{Limit: 200}
	var matches []User
	for {
		users, err := c.ListUsers(ctx, opts)
		if err != nil {
			return nil, err
		}
		for _, u := range users {
			hay := strings.ToLower(strings.Join([]string{u.UserName, u.FirstName, u.LastName, u.FirstName + " " + u.LastName, u.Email}, "\x1f"))
			if strings.Contains(hay, lower) {
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
	req := userToCreateRequest(user)
	var resp User
	if err := c.doJSON(ctx, http.MethodPost, BaseAPIPathV3+"/users", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

type groupNamesRequest struct {
	Groups []string `json:"groups"`
}

type roleNamesRequest struct {
	Roles []string `json:"roles"`
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
