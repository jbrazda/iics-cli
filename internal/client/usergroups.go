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

// UserGroup represents an IICS user group. The response uses "userGroupName";
// "name" is accepted as an input alias (it is the field the v3 create/update
// request expects).
type UserGroup struct {
	ID            string            `json:"id,omitempty"`
	OrgID         string            `json:"orgId,omitempty"`
	UserGroupName string            `json:"userGroupName,omitempty"`
	Name          string            `json:"name,omitempty"`
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

// ResolvedName returns the group name, preferring the response field
// (userGroupName) and falling back to the request alias (name).
func (g *UserGroup) ResolvedName() string {
	if g.UserGroupName != "" {
		return g.UserGroupName
	}
	return g.Name
}

// userGroupRequest is the body the v3 create/update endpoints expect: a name,
// an optional description, and arrays of role and user IDs. roles must be
// non-empty.
type userGroupRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Roles       []string `json:"roles"`
	Users       []string `json:"users,omitempty"`
}

func toUserGroupRequest(g *UserGroup) userGroupRequest {
	req := userGroupRequest{Name: g.ResolvedName(), Description: g.Description, Roles: []string{}}
	for _, r := range g.Roles {
		if r.ID != "" {
			req.Roles = append(req.Roles, r.ID)
		}
	}
	for _, u := range g.Users {
		if u.ID != "" {
			req.Users = append(req.Users, u.ID)
		}
	}
	return req
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

// GetUserGroup retrieves a single user group by ID. The v3 API has no
// get-by-id endpoint, so the paginated list is scanned.
func (c *Client) GetUserGroup(ctx context.Context, id string) (*UserGroup, error) {
	opts := UserGroupListOptions{Limit: 200}
	for {
		groups, err := c.ListUserGroups(ctx, opts)
		if err != nil {
			return nil, err
		}
		for i := range groups {
			if groups[i].ID == id {
				return &groups[i], nil
			}
		}
		if len(groups) < opts.Limit {
			return nil, fmt.Errorf("user group %q not found", id)
		}
		opts.Skip += opts.Limit
	}
}

// GetUserGroupByName retrieves a single user group by its exact name, scanning
// the paginated list (the v3 API has no by-name endpoint).
func (c *Client) GetUserGroupByName(ctx context.Context, name string) (*UserGroup, error) {
	opts := UserGroupListOptions{Limit: 200}
	for {
		groups, err := c.ListUserGroups(ctx, opts)
		if err != nil {
			return nil, err
		}
		for i := range groups {
			if groups[i].UserGroupName == name {
				return &groups[i], nil
			}
		}
		if len(groups) < opts.Limit {
			return nil, fmt.Errorf("user group %q not found", name)
		}
		opts.Skip += opts.Limit
	}
}

// CreateUserGroup creates a new user group. The group must have a name and at
// least one role.
func (c *Client) CreateUserGroup(ctx context.Context, group *UserGroup) (*UserGroup, error) {
	req := toUserGroupRequest(group)
	if req.Name == "" {
		return nil, fmt.Errorf("user group name is required")
	}
	if len(req.Roles) == 0 {
		return nil, fmt.Errorf("user group %q requires at least one role", req.Name)
	}
	var resp UserGroup
	if err := c.doJSON(ctx, http.MethodPost, BaseAPIPathV3+"/userGroups", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// modifyUserGroupMembers calls one of the addRoles / removeRoles / addUsers /
// removeUsers PUT sub-endpoints. Roles and users are identified by name.
func (c *Client) modifyUserGroupMembers(ctx context.Context, id, action, key string, names []string) error {
	if len(names) == 0 {
		return nil
	}
	body := map[string][]string{key: names}
	return c.doJSON(ctx, http.MethodPut, fmt.Sprintf("%s/userGroups/%s/%s", BaseAPIPathV3, id, action), body, nil)
}

// AddUserGroupRoles adds roles (by role name) to a user group.
func (c *Client) AddUserGroupRoles(ctx context.Context, id string, roleNames []string) error {
	return c.modifyUserGroupMembers(ctx, id, "addRoles", "roles", roleNames)
}

// RemoveUserGroupRoles removes roles (by role name) from a user group.
func (c *Client) RemoveUserGroupRoles(ctx context.Context, id string, roleNames []string) error {
	return c.modifyUserGroupMembers(ctx, id, "removeRoles", "roles", roleNames)
}

// AddUserGroupUsers adds users (by user name) to a user group.
func (c *Client) AddUserGroupUsers(ctx context.Context, id string, userNames []string) error {
	return c.modifyUserGroupMembers(ctx, id, "addUsers", "users", userNames)
}

// RemoveUserGroupUsers removes users (by user name) from a user group.
func (c *Client) RemoveUserGroupUsers(ctx context.Context, id string, userNames []string) error {
	return c.modifyUserGroupMembers(ctx, id, "removeUsers", "users", userNames)
}

// UpdateUserGroup applies the differences between desired and the group's current
// state. The v3 API only supports changing role and user membership (via the
// add/remove sub-endpoints), so a name or description change is rejected. When
// desired.Roles or desired.Users is nil that dimension is left untouched. The
// refreshed group is returned.
func (c *Client) UpdateUserGroup(ctx context.Context, id string, desired *UserGroup) (*UserGroup, error) {
	current, err := c.GetUserGroup(ctx, id)
	if err != nil {
		return nil, err
	}
	if n := desired.ResolvedName(); n != "" && n != current.UserGroupName {
		return nil, fmt.Errorf("the IICS API cannot rename a user group (%q -> %q)", current.UserGroupName, n)
	}
	if desired.Description != "" && desired.Description != current.Description {
		return nil, fmt.Errorf("the IICS API cannot change a user group's description")
	}

	if desired.Roles != nil {
		want := make(map[string]bool)
		for _, r := range desired.Roles {
			if r.RoleName != "" {
				want[r.RoleName] = true
			}
		}
		if len(want) == 0 {
			return nil, fmt.Errorf("a user group must keep at least one role")
		}
		have := make(map[string]bool)
		for _, r := range current.Roles {
			have[r.RoleName] = true
		}
		var add, remove []string
		for name := range want {
			if !have[name] {
				add = append(add, name)
			}
		}
		for name := range have {
			if !want[name] {
				remove = append(remove, name)
			}
		}
		if err := c.AddUserGroupRoles(ctx, id, add); err != nil {
			return nil, err
		}
		if err := c.RemoveUserGroupRoles(ctx, id, remove); err != nil {
			return nil, err
		}
	}

	if desired.Users != nil {
		want := make(map[string]bool)
		for _, u := range desired.Users {
			if u.UserName != "" {
				want[u.UserName] = true
			}
		}
		have := make(map[string]bool)
		for _, u := range current.Users {
			have[u.UserName] = true
		}
		var add, remove []string
		for name := range want {
			if !have[name] {
				add = append(add, name)
			}
		}
		for name := range have {
			if !want[name] {
				remove = append(remove, name)
			}
		}
		if err := c.AddUserGroupUsers(ctx, id, add); err != nil {
			return nil, err
		}
		if err := c.RemoveUserGroupUsers(ctx, id, remove); err != nil {
			return nil, err
		}
	}

	return c.GetUserGroup(ctx, id)
}

// DeleteUserGroup deletes a user group by ID.
func (c *Client) DeleteUserGroup(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, fmt.Sprintf("%s/userGroups/%s", BaseAPIPathV3, id), nil, nil)
}
