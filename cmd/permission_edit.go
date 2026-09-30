package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/tui"
	"github.com/spf13/cobra"
)

func newPermissionEditCmd() *cobra.Command {
	var obj objectRef
	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Edit an object's permissions in an interactive grid",
		Long: `Edit an object's ACLs in an interactive grid: user groups and users are rows,
read / update / delete / execute / change permission are columns.

Keys: space toggles a permission, a toggles a row, c a column, d clears a
row (the ACL is deleted), n adds a user group or user, / filters. Enter ends
editing and shows a review of the changes (+ add, ~ update, - delete) before
anything is applied.

Without --object-id or --path, pick a project, then optionally a folder or
asset inside it. Requires a terminal and change permission on the object.`,
		Example: `  iics permission edit --path "Default/Sales" --type Folder
  iics permission edit --object-id <id>
  iics permission edit            # pick the object`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !isInteractiveTTY() {
				return fmt.Errorf("permission edit requires a terminal; use 'permission add/update/delete/set' in scripts")
			}
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			var id, label string
			if obj.id == "" && obj.path == "" {
				picked, perr := pickObject(ctx, c)
				if perr != nil || picked == nil {
					return perr
				}
				id, label = picked.ID, fmt.Sprintf("%s (%s)", picked.Path, picked.Type)
			} else if id, label, err = obj.resolve(ctx, c); err != nil {
				return err
			}

			access, err := c.CheckObjectAccess(ctx, id, "")
			if err != nil {
				return err
			}
			if !access.Permissions.ChangePermission {
				return fmt.Errorf("you do not have change permission on %s", label)
			}
			current, err := c.ListObjectACLs(ctx, id)
			if err != nil {
				return err
			}
			users, groups, err := listPrincipalNames(ctx, c)
			if err != nil {
				return err
			}
			currentUser := ""
			if _, p, _, perr := resolveProfile(); perr == nil && p != nil {
				currentUser = p.Username
			}

			plan, ok, err := tui.EditACLs(tui.ACLEditorInput{
				ObjectLabel: label,
				Current:     current,
				Users:       users,
				Groups:      groups,
				CurrentUser: currentUser,
				Out:         os.Stderr,
				Accessible:  prompter.Accessible,
				NoColor:     editorNoColor(),
			})
			if err != nil {
				return err
			}
			if !ok {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Canceled.")
				return nil
			}
			if err = applyACLPlan(ctx, c, id, plan); err != nil {
				return err
			}
			for _, l := range tui.ACLPlanLines(plan) {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), l)
			}
			return printACLs(ctx, c, id)
		},
	}
	obj.register(cmd)
	return cmd
}

// listPrincipalNames returns all user names and user group names.
func listPrincipalNames(ctx context.Context, c *client.Client) ([]string, []string, error) {
	var users []string
	opts := client.UserListOptions{Limit: 200}
	for {
		batch, err := c.ListUsers(ctx, opts)
		if err != nil {
			return nil, nil, fmt.Errorf("listing users: %w", err)
		}
		for _, u := range batch {
			users = append(users, u.UserName)
		}
		if len(batch) < opts.Limit {
			break
		}
		opts.Skip += opts.Limit
	}
	groups, err := listAllUserGroups(ctx, c)
	if err != nil {
		return nil, nil, fmt.Errorf("listing user groups: %w", err)
	}
	names := make([]string, len(groups))
	for i, g := range groups {
		names[i] = g.UserGroupName
	}
	return users, names, nil
}

// pickObject lets the user choose a project, then drill into folders, and
// pick the project, a folder, or an asset. Returns nil when canceled.
func pickObject(ctx context.Context, c *client.Client) (*client.Object, error) {
	projects, err := c.ListAllObjects(ctx, client.ObjectsListOptions{Type: "Project"}, nil)
	if err != nil {
		return nil, err
	}
	if len(projects.Objects) == 0 {
		return nil, fmt.Errorf("no projects found")
	}
	items := sortedObjects(projects.Objects)
	idx, err := tui.PickOne(prompter, "Select a project", items, func(o client.Object) string { return o.Path })
	if err != nil || idx < 0 {
		return nil, err
	}
	current := items[idx]

	for {
		children, lerr := c.ListAllObjects(ctx, client.ObjectsListOptions{Query: fmt.Sprintf("location=='%s'", current.Path)}, nil)
		if lerr != nil {
			return nil, lerr
		}
		// The location query also returns the location object itself.
		var kids []client.Object
		for _, o := range sortedObjects(children.Objects) {
			if o.ID != current.ID {
				kids = append(kids, o)
			}
		}
		if len(kids) == 0 {
			return &current, nil
		}
		options := append([]client.Object{current}, kids...)
		idx, err = tui.PickOne(prompter, "Select the object", options, func(o client.Object) string {
			if o.ID == current.ID {
				return fmt.Sprintf("(this %s) %s", strings.ToLower(o.Type), o.Path)
			}
			name := o.Path[strings.LastIndex(o.Path, "/")+1:]
			if o.Type == "Folder" {
				return name + "/"
			}
			return fmt.Sprintf("%s  [%s]", name, o.Type)
		})
		if err != nil || idx < 0 {
			return nil, err
		}
		chosen := options[idx]
		if chosen.ID == current.ID || chosen.Type != "Folder" {
			return &chosen, nil
		}
		current = chosen
	}
}

// sortedObjects orders folders first, then by path.
func sortedObjects(objs []client.Object) []client.Object {
	out := append([]client.Object(nil), objs...)
	sort.SliceStable(out, func(i, j int) bool {
		fi, fj := out[i].Type == "Folder", out[j].Type == "Folder"
		if fi != fj {
			return fi
		}
		return strings.ToLower(out[i].Path) < strings.ToLower(out[j].Path)
	})
	return out
}
