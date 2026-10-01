package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

// AddUserRoles assigns roles (by name) to a user.
func (c *Client) AddUserRoles(ctx context.Context, userID string, roles []string) error {
	return c.putUserMembership(ctx, userID, "addRoles", roleNamesRequest{Roles: roles})
}

// RemoveUserRoles removes role assignments (by name) from a user.
func (c *Client) RemoveUserRoles(ctx context.Context, userID string, roles []string) error {
	return c.putUserMembership(ctx, userID, "removeRoles", roleNamesRequest{Roles: roles})
}

// AddUserGroups assigns user groups (by name) to a user.
func (c *Client) AddUserGroups(ctx context.Context, userID string, groups []string) error {
	return c.putUserMembership(ctx, userID, "addGroups", groupNamesRequest{Groups: groups})
}

// RemoveUserGroups removes user group assignments (by name) from a user.
func (c *Client) RemoveUserGroups(ctx context.Context, userID string, groups []string) error {
	return c.putUserMembership(ctx, userID, "removeGroups", groupNamesRequest{Groups: groups})
}

func (c *Client) putUserMembership(ctx context.Context, userID, action string, body interface{}) error {
	path := fmt.Sprintf("%s/users/%s/%s", BaseAPIPathV3, url.PathEscape(userID), action)
	return c.doJSON(ctx, http.MethodPut, path, body, nil)
}

// MembershipPlan is the result of PlanMembership.
type MembershipPlan struct {
	// Add and Remove are the names to send to the API.
	Add    []string
	Remove []string
	// AlreadyAssigned and NotAssigned are requested changes that are no-ops.
	AlreadyAssigned []string
	NotAssigned     []string
	// Inherited are requested additions skipped because a user group of the
	// user already grants them.
	Inherited []string
}

// MembershipRequest describes requested role or user group changes.
type MembershipRequest struct {
	Add     []string
	Remove  []string
	Replace []string
	// ReplaceSet is true when Replace was given (an empty Replace removes all).
	ReplaceSet bool
	// Inherited lists names (MemberName.Name form) the user already has
	// through user groups; they are not added.
	Inherited []string
}

// MemberName is an assignable role or user group. Name is the value the
// assignment endpoints expect; Aliases are other names users may type (for
// roles the endpoints match the display name, so the role name is an alias).
type MemberName struct {
	Name    string
	Aliases []string
}

// RoleMemberName returns the assignment name for a role: its display name,
// or the role name when no display name is set.
func RoleMemberName(roleName, displayName string) string {
	if displayName != "" {
		return displayName
	}
	return roleName
}

// PlanMembership validates a membership change and computes the names to add
// and remove. kind is used in error messages ("role", "user group").
// available lists every assignable entry in the organization and current the
// names (MemberName.Name form) assigned now. Requested names match a Name or
// alias case-insensitively and are returned as Name; duplicates are ignored.
// Unknown names, Replace combined with Add/Remove, and a name in both Add and
// Remove are errors.
func PlanMembership(kind string, available []MemberName, current []string, req MembershipRequest) (MembershipPlan, error) {
	var plan MembershipPlan
	if req.ReplaceSet && (len(req.Add) > 0 || len(req.Remove) > 0) {
		return plan, fmt.Errorf("--replace cannot be combined with --add or --remove")
	}
	if !req.ReplaceSet && len(req.Add) == 0 && len(req.Remove) == 0 {
		return plan, fmt.Errorf("provide --add, --remove or --replace")
	}

	canonical := make(map[string]string, len(available)*2)
	for _, m := range available {
		for _, a := range m.Aliases {
			if a != "" {
				canonical[strings.ToLower(a)] = m.Name
			}
		}
	}
	// Names take precedence over aliases of other entries.
	for _, m := range available {
		canonical[strings.ToLower(m.Name)] = m.Name
	}
	resolve := func(names []string) ([]string, error) {
		seen := make(map[string]bool)
		var out, unknown []string
		for _, n := range names {
			n = strings.TrimSpace(n)
			if n == "" {
				continue
			}
			name, ok := canonical[strings.ToLower(n)]
			if !ok {
				unknown = append(unknown, n)
				continue
			}
			if !seen[name] {
				seen[name] = true
				out = append(out, name)
			}
		}
		if len(unknown) > 0 {
			return nil, fmt.Errorf("unknown %s(s): %s", kind, strings.Join(unknown, ", "))
		}
		return out, nil
	}

	add, err := resolve(req.Add)
	if err != nil {
		return plan, err
	}
	remove, err := resolve(req.Remove)
	if err != nil {
		return plan, err
	}
	replace, err := resolve(req.Replace)
	if err != nil {
		return plan, err
	}

	cur := make(map[string]bool, len(current))
	for _, n := range current {
		cur[n] = true
	}
	inherited := make(map[string]bool, len(req.Inherited))
	for _, n := range req.Inherited {
		inherited[n] = true
	}

	if req.ReplaceSet {
		want := make(map[string]bool, len(replace))
		for _, n := range replace {
			want[n] = true
			switch {
			case cur[n]:
			case inherited[n]:
				plan.Inherited = append(plan.Inherited, n)
			default:
				plan.Add = append(plan.Add, n)
			}
		}
		for _, n := range current {
			if !want[n] {
				plan.Remove = append(plan.Remove, n)
			}
		}
	} else {
		addSet := make(map[string]bool, len(add))
		for _, n := range add {
			addSet[n] = true
		}
		var conflict []string
		for _, n := range remove {
			if addSet[n] {
				conflict = append(conflict, n)
			}
		}
		if len(conflict) > 0 {
			return plan, fmt.Errorf("%s(s) in both --add and --remove: %s", kind, strings.Join(conflict, ", "))
		}
		for _, n := range add {
			switch {
			case cur[n]:
				plan.AlreadyAssigned = append(plan.AlreadyAssigned, n)
			case inherited[n]:
				plan.Inherited = append(plan.Inherited, n)
			default:
				plan.Add = append(plan.Add, n)
			}
		}
		for _, n := range remove {
			if cur[n] {
				plan.Remove = append(plan.Remove, n)
			} else {
				plan.NotAssigned = append(plan.NotAssigned, n)
			}
		}
	}

	for _, s := range [][]string{plan.Add, plan.Remove, plan.AlreadyAssigned, plan.NotAssigned, plan.Inherited} {
		sort.Strings(s)
	}
	return plan, nil
}

// InheritedRoles returns the roles granted by the user groups with the given
// IDs, keyed by role ID.
func InheritedRoles(groups []UserGroup, groupIDs []string) map[string]UserRole {
	want := make(map[string]bool, len(groupIDs))
	for _, id := range groupIDs {
		want[id] = true
	}
	out := make(map[string]UserRole)
	for _, g := range groups {
		if !want[g.ID] {
			continue
		}
		for _, r := range g.Roles {
			if _, ok := out[r.ID]; !ok {
				out[r.ID] = r
			}
		}
	}
	return out
}

// CheckKeepsDirectRole returns an error when removing roles (after adding
// roles) would leave a user with no directly assigned role. IICS rejects that
// with V3API_IDSError_044 ("The group must have at least one role") even when
// the user's groups grant roles. current, add and remove are role names as
// planned by PlanMembership (add not assigned, remove assigned).
func CheckKeepsDirectRole(userName string, current, add, remove []string) error {
	if len(remove) == 0 || len(current)+len(add)-len(remove) > 0 {
		return nil
	}
	return fmt.Errorf("cannot remove every role assigned directly to %s: IICS requires a user to keep at least one direct role (roles from user groups do not count); add a replacement role in the same change or keep one of: %s",
		userName, strings.Join(remove, ", "))
}
