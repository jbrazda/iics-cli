package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/config"
	"github.com/jbrazda/iics-cli/internal/tui"
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
	idx, err := tui.PickOne(prompter, fmt.Sprintf("Select a user group to %s", action), groups, func(g client.UserGroup) string {
		return fmt.Sprintf("%s (%d members, %d roles)", g.UserGroupName, g.CountMembers, g.CountRoles)
	})
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

// runGroupWizard interactively edits a user group. On create it asks for the
// name, description and roles; on update (forCreate=false) only roles can
// change - the v3 API cannot rename a group or edit its description. It
// returns false if the user canceled.
func runGroupWizard(ctx context.Context, c *client.Client, g *client.UserGroup, forCreate bool) (bool, error) {
	if !config.IsTerminal() {
		return false, fmt.Errorf("--interactive requires a terminal")
	}
	roles, err := listAllRoles(ctx, c)
	if err != nil {
		return false, err
	}
	if len(roles) == 0 {
		return false, fmt.Errorf("no roles available; a user group requires at least one role")
	}
	var existing []string
	if forCreate {
		groups, gerr := listAllUserGroups(ctx, c)
		if gerr != nil {
			return false, gerr
		}
		for _, eg := range groups {
			existing = append(existing, eg.UserGroupName)
		}
	}
	return tui.RunGroupWizard(tui.GroupWizardInput{
		Group:         g,
		Update:        !forCreate,
		Roles:         roles,
		ExistingNames: existing,
		Out:           os.Stderr,
		Accessible:    prompter.Accessible,
	})
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
