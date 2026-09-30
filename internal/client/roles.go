package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Role represents an IICS role.
type Role struct {
	ID                 string          `json:"id,omitempty"`
	OrgID              string          `json:"orgId,omitempty"`
	RoleName           string          `json:"roleName"`
	Description        string          `json:"description,omitempty"`
	DisplayName        string          `json:"displayName,omitempty"`
	DisplayDescription string          `json:"displayDescription,omitempty"`
	Status             string          `json:"status,omitempty"`
	CreateTime         string          `json:"createTime,omitempty"`
	UpdateTime         string          `json:"updateTime,omitempty"`
	CreatedBy          string          `json:"createdBy,omitempty"`
	UpdatedBy          string          `json:"updatedBy,omitempty"`
	SystemRole         bool            `json:"systemRole,omitempty"`
	Privileges         []RolePrivilege `json:"privileges,omitempty"`
}

// RolePrivilege is a privilege assigned to a role, returned only when the
// request includes expand=privileges.
type RolePrivilege struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Service     string `json:"service,omitempty"`
	Status      string `json:"status,omitempty"`
}

// RoleGetOptions selects a single role by ID or name. Exactly one of ID or
// Name must be set.
type RoleGetOptions struct {
	ID               string
	Name             string
	ExpandPrivileges bool
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
	if err := c.doJSONWithQuery(ctx, http.MethodGet, BaseAPIPathV3+"/roles", query, nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetRole retrieves a single role by ID or name. The v3 API has no
// /roles/{id} endpoint, so the role is looked up with a q filter on roleId
// or roleName.
func (c *Client) GetRole(ctx context.Context, opts RoleGetOptions) (*Role, error) {
	query := make(map[string]string)
	var key string
	switch {
	case opts.ID != "" && opts.Name != "":
		return nil, fmt.Errorf("specify either role ID or role name, not both")
	case opts.ID != "":
		key = opts.ID
		query["q"] = fmt.Sprintf("roleId==%q", opts.ID)
	case opts.Name != "":
		key = opts.Name
		query["q"] = fmt.Sprintf("roleName==%q", opts.Name)
	default:
		return nil, fmt.Errorf("role ID or role name is required")
	}
	if opts.ExpandPrivileges {
		query["expand"] = "privileges"
	}

	var resp []Role
	if err := c.doJSONWithQuery(ctx, http.MethodGet, BaseAPIPathV3+"/roles", query, nil, &resp); err != nil {
		return nil, err
	}
	if len(resp) == 0 {
		return nil, fmt.Errorf("role %q not found", key)
	}
	return &resp[0], nil
}

// CreateRoleRequest is the POST body for creating a custom role. Privileges
// holds privilege IDs; the API requires at least one.
type CreateRoleRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Privileges  []string `json:"privileges"`
}

// CreateRole creates a new custom role.
func (c *Client) CreateRole(ctx context.Context, req *CreateRoleRequest) (*Role, error) {
	var resp Role
	if err := c.doJSON(ctx, http.MethodPost, BaseAPIPathV3+"/roles", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// RoleRef identifies a role by ID or name. Exactly one field must be set.
type RoleRef struct {
	ID   string
	Name string
}

// rolePrivilegesRequest is the PUT body for addPrivileges / removePrivileges.
type rolePrivilegesRequest struct {
	Privileges []string `json:"privileges"`
}

// AddRolePrivileges adds privileges (by name) to a custom role.
func (c *Client) AddRolePrivileges(ctx context.Context, role RoleRef, privileges []string) error {
	return c.updateRolePrivileges(ctx, role, "addPrivileges", privileges)
}

// RemoveRolePrivileges removes privileges (by name) from a custom role. A
// role must keep at least one privilege.
func (c *Client) RemoveRolePrivileges(ctx context.Context, role RoleRef, privileges []string) error {
	return c.updateRolePrivileges(ctx, role, "removePrivileges", privileges)
}

func (c *Client) updateRolePrivileges(ctx context.Context, role RoleRef, action string, privileges []string) error {
	var path string
	switch {
	case role.ID != "" && role.Name != "":
		return fmt.Errorf("specify either role ID or role name, not both")
	case role.ID != "":
		path = fmt.Sprintf("%s/roles/%s/%s", BaseAPIPathV3, url.PathEscape(role.ID), action)
	case role.Name != "":
		path = fmt.Sprintf("%s/roles/name/%s/%s", BaseAPIPathV3, url.PathEscape(role.Name), action)
	default:
		return fmt.Errorf("role ID or role name is required")
	}
	if len(privileges) == 0 {
		return fmt.Errorf("at least one privilege is required")
	}
	return c.doJSON(ctx, http.MethodPut, path, &rolePrivilegesRequest{Privileges: privileges}, nil)
}

// ResolvePrivileges maps each reference (privilege ID or name) to a known
// privilege, dropping duplicates. Unknown references are reported together in
// one error.
func (c *Client) ResolvePrivileges(ctx context.Context, refs []string) ([]Privilege, error) {
	all, err := c.ListPrivileges(ctx)
	if err != nil {
		return nil, err
	}
	byKey := make(map[string]Privilege, len(all)*2)
	for _, p := range all {
		byKey[p.ID] = p
		byKey[p.Name] = p
	}
	var resolved []Privilege
	var unknown []string
	seen := make(map[string]bool, len(refs))
	for _, ref := range refs {
		if p, ok := byKey[ref]; ok {
			if !seen[p.ID] {
				seen[p.ID] = true
				resolved = append(resolved, p)
			}
		} else {
			unknown = append(unknown, ref)
		}
	}
	if len(unknown) > 0 {
		return nil, fmt.Errorf("unknown privilege(s): %s", strings.Join(unknown, ", "))
	}
	return resolved, nil
}

// DeleteRole deletes a role by ID.
func (c *Client) DeleteRole(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, fmt.Sprintf("%s/roles/%s", BaseAPIPathV3, url.PathEscape(id)), nil, nil)
}
