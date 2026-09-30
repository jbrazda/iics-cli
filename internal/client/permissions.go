package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

// Principal types accepted by the object permissions API.
const (
	PrincipalUser  = "USER"
	PrincipalGroup = "GROUP"
)

// ACLPrincipal identifies the user or user group an ACL applies to.
type ACLPrincipal struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

// ACLPermissions are the permissions an ACL grants.
type ACLPermissions struct {
	Read             bool `json:"read"`
	Update           bool `json:"update"`
	Delete           bool `json:"delete"`
	Execute          bool `json:"execute"`
	ChangePermission bool `json:"changePermission"`
}

// ObjectACL is an access control list entry for one principal on an object.
type ObjectACL struct {
	ID          string         `json:"id,omitempty"`
	Principal   ACLPrincipal   `json:"principal"`
	Permissions ACLPermissions `json:"permissions"`
}

// ObjectAccess is the current user's access to an object (checkAccess).
type ObjectAccess struct {
	Permissions struct {
		Create           bool `json:"create"`
		Read             bool `json:"read"`
		Update           bool `json:"update"`
		Delete           bool `json:"delete"`
		Execute          bool `json:"execute"`
		ChangePermission bool `json:"changePermission"`
	} `json:"permissions"`
}

// aclRequest is the POST/PUT body; the ACL ID is only part of the URI.
type aclRequest struct {
	Principal   ACLPrincipal   `json:"principal"`
	Permissions ACLPermissions `json:"permissions"`
}

func permissionsPath(objectID string, parts ...string) string {
	p := fmt.Sprintf("%s/objects/%s/permissions", BaseAPIPathV3, url.PathEscape(objectID))
	for _, part := range parts {
		p += "/" + url.PathEscape(part)
	}
	return p
}

// ListObjectACLs returns all ACLs defined on an object.
func (c *Client) ListObjectACLs(ctx context.Context, objectID string) ([]ObjectACL, error) {
	var resp []ObjectACL
	if err := c.doJSON(ctx, http.MethodGet, permissionsPath(objectID), nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetObjectACL returns one ACL of an object.
func (c *Client) GetObjectACL(ctx context.Context, objectID, aclID string) (*ObjectACL, error) {
	var resp ObjectACL
	if err := c.doJSON(ctx, http.MethodGet, permissionsPath(objectID, aclID), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreateObjectACL creates an ACL for one principal and returns it with its ID.
func (c *Client) CreateObjectACL(ctx context.Context, objectID string, acl ObjectACL) (*ObjectACL, error) {
	var raw json.RawMessage
	body := aclRequest{Principal: acl.Principal, Permissions: acl.Permissions}
	if err := c.doJSON(ctx, http.MethodPost, permissionsPath(objectID), body, &raw); err != nil {
		return nil, err
	}
	// The documented response is an array with the created ACL; accept a
	// single object too.
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return &acl, nil
	}
	if raw[0] == '[' {
		var list []ObjectACL
		if err := json.Unmarshal(raw, &list); err != nil {
			return nil, fmt.Errorf("parsing response: %w", err)
		}
		if len(list) == 0 {
			return &acl, nil
		}
		return &list[0], nil
	}
	var one ObjectACL
	if err := json.Unmarshal(raw, &one); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	return &one, nil
}

// UpdateObjectACL replaces the permissions of an existing ACL.
func (c *Client) UpdateObjectACL(ctx context.Context, objectID, aclID string, acl ObjectACL) error {
	body := aclRequest{Principal: acl.Principal, Permissions: acl.Permissions}
	return c.doJSON(ctx, http.MethodPut, permissionsPath(objectID, aclID), body, nil)
}

// DeleteObjectACL deletes one ACL of an object.
func (c *Client) DeleteObjectACL(ctx context.Context, objectID, aclID string) error {
	return c.doJSON(ctx, http.MethodDelete, permissionsPath(objectID, aclID), nil, nil)
}

// DeleteAllObjectACLs deletes every ACL of an object.
func (c *Client) DeleteAllObjectACLs(ctx context.Context, objectID string) error {
	return c.doJSON(ctx, http.MethodDelete, permissionsPath(objectID), nil, nil)
}

// CheckObjectAccess returns the current user's access to an object. With
// assetType set, Create reports whether that asset type can be created in
// the object (a project or folder).
func (c *Client) CheckObjectAccess(ctx context.Context, objectID, assetType string) (*ObjectAccess, error) {
	query := map[string]string{}
	if assetType != "" {
		query["type"] = assetType
	}
	var resp ObjectAccess
	if err := c.doJSONWithQuery(ctx, http.MethodGet, permissionsPath(objectID, "checkAccess"), query, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// PrincipalKey identifies a principal regardless of name case.
func PrincipalKey(p ACLPrincipal) string {
	return strings.ToUpper(p.Type) + "\x1f" + strings.ToLower(p.Name)
}

// ACLPlan lists the ACL changes needed to reach a desired state.
type ACLPlan struct {
	Create []ObjectACL
	// Update entries carry the existing ACL ID and the new permissions.
	Update []ObjectACL
	// Delete entries are existing ACLs (with IDs).
	Delete []ObjectACL
}

// Empty reports whether the plan has no changes.
func (p ACLPlan) Empty() bool {
	return len(p.Create) == 0 && len(p.Update) == 0 && len(p.Delete) == 0
}

// DiffACLs compares current ACLs with desired ones, matched by principal
// type and case-insensitive name. Principals only in desired are created,
// those with different permissions are updated. Principals only in current
// are deleted when prune is true and left alone otherwise. Results are
// sorted by principal.
func DiffACLs(current, desired []ObjectACL, prune bool) ACLPlan {
	var plan ACLPlan
	cur := make(map[string]ObjectACL, len(current))
	for _, a := range current {
		cur[PrincipalKey(a.Principal)] = a
	}
	want := make(map[string]bool, len(desired))
	for _, d := range desired {
		k := PrincipalKey(d.Principal)
		if want[k] {
			continue
		}
		want[k] = true
		existing, ok := cur[k]
		switch {
		case !ok:
			plan.Create = append(plan.Create, ObjectACL{Principal: d.Principal, Permissions: d.Permissions})
		case existing.Permissions != d.Permissions:
			plan.Update = append(plan.Update, ObjectACL{ID: existing.ID, Principal: existing.Principal, Permissions: d.Permissions})
		}
	}
	if prune {
		for _, a := range current {
			if !want[PrincipalKey(a.Principal)] {
				plan.Delete = append(plan.Delete, a)
			}
		}
	}
	for _, s := range [][]ObjectACL{plan.Create, plan.Update, plan.Delete} {
		sort.Slice(s, func(i, j int) bool { return PrincipalKey(s[i].Principal) < PrincipalKey(s[j].Principal) })
	}
	return plan
}

// ParseACLPermissions parses a comma-separated permission list such as
// "read,execute". "all" grants every permission and "none" none. Names are
// case-insensitive; "changePermission", "change-permission" and "perm" are
// accepted for the change permission right.
func ParseACLPermissions(list []string) (ACLPermissions, error) {
	var p ACLPermissions
	for _, item := range list {
		for _, name := range strings.Split(item, ",") {
			switch strings.ToLower(strings.TrimSpace(name)) {
			case "":
			case "read":
				p.Read = true
			case "update":
				p.Update = true
			case "delete":
				p.Delete = true
			case "execute":
				p.Execute = true
			case "changepermission", "change-permission", "perm":
				p.ChangePermission = true
			case "all":
				p = ACLPermissions{Read: true, Update: true, Delete: true, Execute: true, ChangePermission: true}
			case "none":
				p = ACLPermissions{}
			default:
				return p, fmt.Errorf("unknown permission %q (use read, update, delete, execute, changePermission, all, none)", strings.TrimSpace(name))
			}
		}
	}
	return p, nil
}

// String lists the granted permissions, e.g. "read,execute", or "none".
func (p ACLPermissions) String() string {
	var out []string
	for _, f := range []struct {
		on   bool
		name string
	}{{p.Read, "read"}, {p.Update, "update"}, {p.Delete, "delete"}, {p.Execute, "execute"}, {p.ChangePermission, "changePermission"}} {
		if f.on {
			out = append(out, f.name)
		}
	}
	if len(out) == 0 {
		return "none"
	}
	return strings.Join(out, ",")
}
