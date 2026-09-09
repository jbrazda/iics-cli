package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/config"
)

// listAllUserGroups fetches every user group, following pagination.
func listAllUserGroups(ctx context.Context, c *client.Client) ([]client.UserGroup, error) {
	var all []client.UserGroup
	opts := client.UserGroupListOptions{Limit: 200}
	for {
		batch, err := c.ListUserGroups(ctx, opts)
		if err != nil {
			return nil, err
		}
		all = append(all, batch...)
		if len(batch) < opts.Limit {
			return all, nil
		}
		opts.Skip += opts.Limit
	}
}

// pickUserGroup lists the user groups and lets the operator choose one.
// action is a verb used in the prompt label (e.g. "edit", "delete").
func pickUserGroup(ctx context.Context, c *client.Client, action string) (*client.UserGroup, error) {
	if !config.IsTerminal() {
		return nil, fmt.Errorf("provide --id or --name (stdin is not a terminal)")
	}
	groups, err := listAllUserGroups(ctx, c)
	if err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return nil, fmt.Errorf("no user groups found")
	}
	labels := make([]string, len(groups))
	for i, g := range groups {
		labels[i] = fmt.Sprintf("%s (%d members, %d roles)", g.UserGroupName, g.CountMembers, g.CountRoles)
	}
	idx, err := promptSelect(fmt.Sprintf("Select a user group to %s", action), labels)
	if err != nil {
		return nil, err
	}
	if idx < 0 {
		return nil, fmt.Errorf("canceled")
	}
	return &groups[idx], nil
}

// resolveUserGroup returns the group identified by --id or --name, or prompts the
// operator to pick one when both are empty.
func resolveUserGroup(ctx context.Context, c *client.Client, id, name, action string) (*client.UserGroup, error) {
	switch {
	case id != "":
		return c.GetUserGroup(ctx, id)
	case name != "":
		return c.GetUserGroupByName(ctx, name)
	default:
		return pickUserGroup(ctx, c, action)
	}
}

// runGroupWizard interactively edits a user group. On create it prompts for
// name, description, and roles. On update (forCreate=false) only roles can
// change - the v3 API cannot rename a group or edit its description - so the
// name/description are shown for context and only the role selection is prompted.
func runGroupWizard(ctx context.Context, c *client.Client, g *client.UserGroup, forCreate bool) error {
	if !config.IsTerminal() {
		return fmt.Errorf("--interactive requires a terminal")
	}

	if forCreate {
		for {
			name, err := promptText("Group Name", g.UserGroupName)
			if err != nil {
				return err
			}
			if name != "" {
				g.UserGroupName = name
				break
			}
			_, _ = fmt.Fprintln(os.Stderr, "Group name is required.")
		}
		desc, err := promptText("Description (optional)", g.Description)
		if err != nil {
			return err
		}
		g.Description = desc
	} else {
		_, _ = fmt.Fprintf(os.Stderr, "Editing group %q", g.UserGroupName)
		if g.Description != "" {
			_, _ = fmt.Fprintf(os.Stderr, " - %s", g.Description)
		}
		_, _ = fmt.Fprintln(os.Stderr, " (name and description cannot be changed via the API)")
	}

	roles, err := listAllRoles(ctx, c)
	if err != nil {
		return err
	}
	if len(roles) == 0 {
		return fmt.Errorf("no roles available; a user group requires at least one role")
	}
	current := make(map[string]bool)
	for _, r := range g.Roles {
		current[r.ID] = true
	}
	labels := make([]string, len(roles))
	defaults := make([]int, 0)
	for i, r := range roles {
		labels[i] = fmt.Sprintf("%s - %s", r.RoleName, truncate(r.Description, 80))
		if current[r.ID] {
			defaults = append(defaults, i)
		}
	}
	for {
		picks, perr := promptMultiSelect("Roles (at least one required)", labels, defaults)
		if perr != nil {
			return perr
		}
		if len(picks) == 0 {
			_, _ = fmt.Fprintln(os.Stderr, "Select at least one role.")
			continue
		}
		g.Roles = nil
		for _, idx := range picks {
			g.Roles = append(g.Roles, client.UserRole{ID: roles[idx].ID, RoleName: roles[idx].RoleName})
		}
		return nil
	}
}

// listAllRoles fetches every role, following pagination.
func listAllRoles(ctx context.Context, c *client.Client) ([]client.Role, error) {
	var all []client.Role
	opts := client.RoleListOptions{Limit: 200}
	for {
		batch, err := c.ListRoles(ctx, opts)
		if err != nil {
			return nil, err
		}
		all = append(all, batch...)
		if len(batch) < opts.Limit {
			return all, nil
		}
		opts.Skip += opts.Limit
	}
}
