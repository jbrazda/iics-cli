package client

import (
	"context"
	"encoding/xml"
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
	req := userToCreateRequest(user)
	var resp User
	if err := c.doJSON(ctx, http.MethodPost, BaseAPIPathV3+"/users", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// updateUserV2Request is the XML body for updating a user via the V2 API.
// V2 uses different field names than V3 (timezone vs timeZoneId, emails vs email)
// and requires application/xml content type.
type updateUserV2Request struct {
	XMLName             xml.Name `xml:"user" json:"-"`
	OrgID               string   `xml:"orgId"`
	Name                string   `xml:"name"`
	FirstName           string   `xml:"firstName"`
	LastName            string   `xml:"lastName"`
	Title               string   `xml:"title,omitempty"`
	Phone               string   `xml:"phone,omitempty"`
	Emails              string   `xml:"emails,omitempty"`
	Description         string   `xml:"description,omitempty"`
	Timezone            string   `xml:"timezone,omitempty"`
	ForceChangePassword bool     `xml:"forceChangePassword,omitempty"`
}

type groupNamesRequest struct {
	Groups []string `json:"groups"`
}

type roleNamesRequest struct {
	Roles []string `json:"roles"`
}

// UpdateUser updates a user's scalar properties via the V2 API, and syncs
// group and role membership via the V3 addGroups/removeGroups/addRoles/removeRoles
// endpoints. It fetches the current user state first to compute diffs.
func (c *Client) UpdateUser(ctx context.Context, id string, user *User) (*User, error) {
	// Fetch current state to know orgId, userName, and existing groups/roles.
	current, err := c.GetUser(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching current user: %w", err)
	}

	// Update scalar properties via V2.
	v2req := &updateUserV2Request{
		OrgID:               current.OrgID,
		Name:                current.UserName,
		FirstName:           user.FirstName,
		LastName:            user.LastName,
		Title:               user.Title,
		Phone:               user.Phone,
		Emails:              user.Email,
		Description:         user.Description,
		Timezone:            user.TimeZoneID,
		ForceChangePassword: user.ForcePasswordChange,
	}
	if err := c.doXML(ctx, http.MethodPost, fmt.Sprintf("%s/user/%s", BaseAPIPathV2, id), v2req, nil); err != nil {
		return nil, fmt.Errorf("updating user properties: %w", err)
	}

	// Sync groups via V3 addGroups/removeGroups.
	currentGroups := make(map[string]bool, len(current.Groups))
	for _, g := range current.Groups {
		currentGroups[g.UserGroupName] = true
	}
	desiredGroups := make(map[string]bool, len(user.Groups))
	for _, g := range user.Groups {
		desiredGroups[g.UserGroupName] = true
	}
	var groupsToAdd, groupsToRemove []string
	for n := range desiredGroups {
		if !currentGroups[n] {
			groupsToAdd = append(groupsToAdd, n)
		}
	}
	for n := range currentGroups {
		if !desiredGroups[n] {
			groupsToRemove = append(groupsToRemove, n)
		}
	}
	if len(groupsToAdd) > 0 {
		if err := c.doJSON(ctx, http.MethodPut,
			fmt.Sprintf("%s/users/%s/addGroups", BaseAPIPathV3, id),
			groupNamesRequest{Groups: groupsToAdd}, nil); err != nil {
			return nil, fmt.Errorf("adding groups: %w", err)
		}
	}
	if len(groupsToRemove) > 0 {
		if err := c.doJSON(ctx, http.MethodPut,
			fmt.Sprintf("%s/users/%s/removeGroups", BaseAPIPathV3, id),
			groupNamesRequest{Groups: groupsToRemove}, nil); err != nil {
			return nil, fmt.Errorf("removing groups: %w", err)
		}
	}

	// Sync roles via V3 addRoles/removeRoles.
	currentRoles := make(map[string]bool, len(current.Roles))
	for _, r := range current.Roles {
		currentRoles[r.RoleName] = true
	}
	desiredRoles := make(map[string]bool, len(user.Roles))
	for _, r := range user.Roles {
		desiredRoles[r.RoleName] = true
	}
	var rolesToAdd, rolesToRemove []string
	for n := range desiredRoles {
		if !currentRoles[n] {
			rolesToAdd = append(rolesToAdd, n)
		}
	}
	for n := range currentRoles {
		if !desiredRoles[n] {
			rolesToRemove = append(rolesToRemove, n)
		}
	}
	if len(rolesToAdd) > 0 {
		if err := c.doJSON(ctx, http.MethodPut,
			fmt.Sprintf("%s/users/%s/addRoles", BaseAPIPathV3, id),
			roleNamesRequest{Roles: rolesToAdd}, nil); err != nil {
			return nil, fmt.Errorf("adding roles: %w", err)
		}
	}
	if len(rolesToRemove) > 0 {
		if err := c.doJSON(ctx, http.MethodPut,
			fmt.Sprintf("%s/users/%s/removeRoles", BaseAPIPathV3, id),
			roleNamesRequest{Roles: rolesToRemove}, nil); err != nil {
			return nil, fmt.Errorf("removing roles: %w", err)
		}
	}

	return c.GetUser(ctx, id)
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
