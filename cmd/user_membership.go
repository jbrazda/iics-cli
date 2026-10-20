package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/output"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// membershipKind describes one of the two user membership types.
type membershipKind struct {
	use, noun, plural string
	// longNote is appended to the command's long help.
	longNote string
	// available returns every entry that can be assigned.
	available func(ctx context.Context, c *client.Client) ([]client.MemberName, error)
	// current returns the assignment names (MemberName.Name form) of the user.
	current func(u *client.User) []string
	// inherited, when set, returns the names the user already has through
	// user groups; requested additions of these are skipped.
	inherited func(ctx context.Context, c *client.Client, u *client.User) ([]string, error)
	// check, when set, validates a planned change before it is applied.
	check  func(u *client.User, current, add, remove []string) error
	add    func(c *client.Client, ctx context.Context, userID string, names []string) error
	remove func(c *client.Client, ctx context.Context, userID string, names []string) error
}

var userRoleMembership = membershipKind{
	use: "update-roles", noun: "role", plural: "roles",
	longNote: `

Roles the user already has through a user group are not assigned directly
(reported as skipped). IICS requires a user to keep at least one directly
assigned role, so a change that would remove the last one fails before any
change is made; add a replacement role in the same command.`,
	// The role assignment endpoints match the role's display name.
	available: func(ctx context.Context, c *client.Client) ([]client.MemberName, error) {
		roles, err := c.ListRoles(ctx, client.RoleListOptions{})
		if err != nil {
			return nil, err
		}
		names := make([]client.MemberName, len(roles))
		for i, r := range roles {
			names[i] = client.MemberName{Name: client.RoleMemberName(r.RoleName, r.DisplayName), Aliases: []string{r.RoleName}}
		}
		return names, nil
	},
	current: func(u *client.User) []string {
		names := make([]string, len(u.Roles))
		for i, r := range u.Roles {
			names[i] = client.RoleMemberName(r.RoleName, r.DisplayName)
		}
		return names
	},
	inherited: func(ctx context.Context, c *client.Client, u *client.User) ([]string, error) {
		if len(u.Groups) == 0 {
			return nil, nil
		}
		groups, err := listAllUserGroups(ctx, c)
		if err != nil {
			return nil, err
		}
		ids := make([]string, len(u.Groups))
		for i, g := range u.Groups {
			ids[i] = g.ID
		}
		var names []string
		for _, r := range client.InheritedRoles(groups, ids) {
			names = append(names, client.RoleMemberName(r.RoleName, r.DisplayName))
		}
		return names, nil
	},
	// IICS rejects removing a user's last direct role.
	check: func(u *client.User, current, add, remove []string) error {
		return client.CheckKeepsDirectRole(u.UserName, current, add, remove)
	},
	add:    (*client.Client).AddUserRoles,
	remove: (*client.Client).RemoveUserRoles,
}

var userGroupMembership = membershipKind{
	use: "update-groups", noun: "user group", plural: "user groups",
	available: func(ctx context.Context, c *client.Client) ([]client.MemberName, error) {
		groups, err := listAllUserGroups(ctx, c)
		if err != nil {
			return nil, err
		}
		names := make([]client.MemberName, len(groups))
		for i, g := range groups {
			names[i] = client.MemberName{Name: g.UserGroupName}
		}
		return names, nil
	},
	current: func(u *client.User) []string {
		names := make([]string, len(u.Groups))
		for i, g := range u.Groups {
			names[i] = g.UserGroupName
		}
		return names
	},
	add:    (*client.Client).AddUserGroups,
	remove: (*client.Client).RemoveUserGroups,
}

// newUserMembershipCmd builds user update-roles / update-groups.
func newUserMembershipCmd(k membershipKind) *cobra.Command {
	var (
		id, userName, csvFields string
		req                     client.MembershipRequest
	)
	cmd := &cobra.Command{
		Use:   k.use,
		Short: fmt.Sprintf("Add, remove or replace a user's %s", k.plural),
		Long: fmt.Sprintf(`Add, remove or replace the %[1]s assigned to a user.

--add and --remove can be combined; --replace sets the exact list (an empty
value removes all %[1]s) and cannot be combined with them. Names match
case-insensitively and duplicates are ignored. Unknown names, or a name in
both --add and --remove, fail before any change is made. Additions are
applied before removals. The resulting user is printed in the --output format.%[2]s`, k.plural, k.longNote),
		Example: fmt.Sprintf(`  iics user %[1]s --username jdoe@example.com --add "Designer,Monitor"
  iics user %[1]s --id <user-id> --add Designer --remove Monitor
  iics user %[1]s --username jdoe@example.com --replace "Designer" -o json`, k.use),
		RunE: func(cmd *cobra.Command, args []string) error {
			req.ReplaceSet = cmd.Flags().Changed("replace")
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			user, err := resolveUser(ctx, c, id, userName)
			if err != nil {
				return err
			}
			available, err := k.available(ctx, c)
			if err != nil {
				return err
			}
			if k.inherited != nil {
				if req.Inherited, err = k.inherited(ctx, c, user); err != nil {
					return err
				}
			}
			current := k.current(user)
			plan, err := client.PlanMembership(k.noun, available, current, req)
			if err != nil {
				return err
			}
			if k.check != nil {
				if err = k.check(user, current, plan.Add, plan.Remove); err != nil {
					return err
				}
			}

			w := cmd.ErrOrStderr()
			reportSkipped(w, "Already assigned", plan.AlreadyAssigned)
			reportSkipped(w, "Not assigned", plan.NotAssigned)
			reportSkipped(w, "Inherited from user group", plan.Inherited)
			if len(plan.Add) == 0 && len(plan.Remove) == 0 {
				_, _ = fmt.Fprintf(w, "No %s changes for %s.\n", k.noun, user.UserName)
			}
			if len(plan.Add) > 0 {
				if err = k.add(c, ctx, user.ID, plan.Add); err != nil {
					return err
				}
				_, _ = fmt.Fprintf(w, "Added %s(s): %s\n", k.noun, strings.Join(plan.Add, ", "))
			}
			if len(plan.Remove) > 0 {
				if err = k.remove(c, ctx, user.ID, plan.Remove); err != nil {
					return err
				}
				_, _ = fmt.Fprintf(w, "Removed %s(s): %s\n", k.noun, strings.Join(plan.Remove, ", "))
			}

			updated, err := c.GetUser(ctx, user.ID)
			if err != nil {
				return err
			}
			return printUser(updated, csvFields)
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "user ID")
	cmd.Flags().StringVar(&userName, "username", "", "user name (exact match)")
	cmd.Flags().StringSliceVar(&req.Add, "add", nil, fmt.Sprintf("%s to assign (comma-separated or repeated)", k.plural))
	cmd.Flags().StringSliceVar(&req.Remove, "remove", nil, fmt.Sprintf("%s to unassign (comma-separated or repeated)", k.plural))
	cmd.Flags().StringSliceVar(&req.Replace, "replace", nil, fmt.Sprintf("exact list of %s to keep; all others are removed", k.plural))
	cmd.Flags().StringVar(&csvFields, "fields", defaultCSVFields, "comma-separated fields for CSV output")
	cmd.MarkFlagsOneRequired("id", "username")
	cmd.MarkFlagsMutuallyExclusive("id", "username")
	cmd.MarkFlagsMutuallyExclusive("replace", "add")
	cmd.MarkFlagsMutuallyExclusive("replace", "remove")
	// Accept --uid / --uname as aliases for --id / --username.
	cmd.Flags().SetNormalizeFunc(func(_ *pflag.FlagSet, name string) pflag.NormalizedName {
		switch name {
		case "uid":
			name = "id"
		case "uname":
			name = "username"
		}
		return pflag.NormalizedName(name)
	})
	return cmd
}

// applyUserMembership applies the user group and role differences between
// before and after through the add/remove assignment endpoints (additions
// first). Other user fields are not changed.
func applyUserMembership(ctx context.Context, c *client.Client, w io.Writer, before, after *client.User) error {
	for _, k := range []membershipKind{userGroupMembership, userRoleMembership} {
		add, remove := client.DiffPrivileges(k.current(before), k.current(after))
		if k.check != nil {
			if err := k.check(before, k.current(before), add, remove); err != nil {
				return err
			}
		}
		if len(add) > 0 {
			if err := k.add(c, ctx, before.ID, add); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(w, "Added %s(s): %s\n", k.noun, strings.Join(add, ", "))
		}
		if len(remove) > 0 {
			if err := k.remove(c, ctx, before.ID, remove); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(w, "Removed %s(s): %s\n", k.noun, strings.Join(remove, ", "))
		}
	}
	return nil
}

func reportSkipped(w io.Writer, label string, names []string) {
	if len(names) > 0 {
		_, _ = fmt.Fprintf(w, "%s (skipped): %s\n", label, strings.Join(names, ", "))
	}
}

// printUser writes a user in the --output format: sections for table, the
// selected fields for CSV, the full object for JSON and YAML.
func printUser(user *client.User, csvFields string) error {
	f, err := output.ParseFormat(outputFmt)
	if err != nil {
		return err
	}
	cfg, _ := loadConfig()
	style := resolveTableStyle(cfg)
	switch f {
	case output.FormatCSV:
		return output.New(output.FormatCSV, os.Stdout, style).Format([]*client.User{user}, buildUserCSVColumns(csvFields))
	case output.FormatTable:
		return printUserSections(os.Stdout, user, style)
	default:
		return output.New(f, os.Stdout, style).Format(user, nil)
	}
}
