package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/output"
	"github.com/jbrazda/iics-cli/internal/tui/privmatrix"
	"github.com/spf13/cobra"
)

func newRoleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "role",
		Short: "Manage roles",
	}
	cmd.AddCommand(newRoleListCmd())
	cmd.AddCommand(newRoleGetCmd())
	cmd.AddCommand(newRoleCreateCmd())
	cmd.AddCommand(newRoleEditCmd())
	cmd.AddCommand(newRolePrivilegesCmd("add-privileges", "Add privileges to a custom role", "Added"))
	cmd.AddCommand(newRolePrivilegesCmd("remove-privileges", "Remove privileges from a custom role", "Removed"))
	cmd.AddCommand(newRoleDeleteCmd())
	return cmd
}

func newRoleListCmd() *cobra.Command {
	var opts client.RoleListOptions
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List roles",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			roles, err := c.ListRoles(context.Background(), opts)
			if err != nil {
				return err
			}
			f, err := getFormatter()
			if err != nil {
				return err
			}
			columns := []output.Column{
				{Header: "ID", Field: "id", Width: 24},
				{Header: "NAME", Field: "roleName", Width: 30},
				{Header: "SYSTEM", Field: "systemRole", Width: 8},
				{Header: "DESCRIPTION", Field: "description"},
			}
			return f.Format(roles, columns)
		},
	}
	cmd.Flags().IntVar(&opts.Limit, "limit", 200, "max results")
	cmd.Flags().IntVar(&opts.Skip, "skip", 0, "number of results to skip")
	return cmd
}

func newRoleGetCmd() *cobra.Command {
	var opts client.RoleGetOptions
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get role details (prompts for a role when --id and --name are omitted)",
		Example: `  iics role get --id <role-id>
  iics role get --name "Business Manager"
  iics role get --name "Business Manager" --privileges
  iics role get --privileges    # pick a role interactively`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()
			if opts.ID == "" && opts.Name == "" {
				picked, perr := pickRole(ctx, c, "view")
				if perr != nil {
					return perr
				}
				opts.ID = picked.ID
			}
			role, err := c.GetRole(ctx, opts)
			if err != nil {
				return err
			}
			sortRolePrivileges(role.Privileges)
			if outputFmt == "csv" {
				f, ferr := getFormatter()
				if ferr != nil {
					return ferr
				}
				if opts.ExpandPrivileges {
					// Plain status column: CSV keeps ANSI codes on a TTY.
					cols := append([]output.Column(nil), rolePrivilegeColumns...)
					cols[2].Func = nil
					return f.Format(role.Privileges, cols)
				}
				return f.Format(role, roleDetailColumns)
			}
			if outputFmt != "" && outputFmt != "table" {
				f, ferr := getFormatter()
				if ferr != nil {
					return ferr
				}
				return f.Format(role, nil)
			}
			w := cmd.OutOrStdout()
			if err = printRoleDetails(w, role); err != nil {
				return err
			}
			if !opts.ExpandPrivileges && !cmd.Flags().Changed("privileges") && isInteractiveTTY() {
				if !promptShowPrivileges() {
					return nil
				}
				opts.ID, opts.Name, opts.ExpandPrivileges = role.ID, "", true
				if role, err = c.GetRole(ctx, opts); err != nil {
					return err
				}
				sortRolePrivileges(role.Privileges)
			}
			if !opts.ExpandPrivileges {
				return nil
			}
			return printRolePrivileges(w, role)
		},
	}
	cmd.Flags().StringVar(&opts.ID, "id", "", "role ID (omit with --name to pick interactively)")
	cmd.Flags().StringVar(&opts.Name, "name", "", "role name")
	cmd.Flags().BoolVar(&opts.ExpandPrivileges, "privileges", false, "include privileges assigned to the role (table output on a terminal prompts when omitted)")
	cmd.MarkFlagsMutuallyExclusive("id", "name")
	return cmd
}

// printRoleDetails renders a role as a vertical detail table.
func printRoleDetails(w io.Writer, r *client.Role) error {
	cfg, _ := loadConfig()
	tf := output.New(output.FormatTable, w, resolveTableStyle(cfg))

	_, _ = fmt.Fprintf(w, "Role: %s\n\n", r.RoleName)
	rows := []output.KVRow{
		output.KV("id", r.ID),
		output.KV("orgId", r.OrgID),
		output.KV("displayName", r.DisplayName),
		output.KV("description", r.Description),
		output.KV("systemRole", strconv.FormatBool(r.SystemRole)),
		output.KV("status", r.Status),
		output.KV("createdBy", r.CreatedBy),
		output.KV("createTime", r.CreateTime),
		output.KV("updatedBy", r.UpdatedBy),
		output.KV("updateTime", r.UpdateTime),
	}
	return tf.Format(rows, output.KVCols)
}

// printRolePrivileges renders the privileges assigned to a role.
func printRolePrivileges(w io.Writer, r *client.Role) error {
	cfg, _ := loadConfig()
	tf := output.New(output.FormatTable, w, resolveTableStyle(cfg))

	_, _ = fmt.Fprintf(w, "\nPrivileges (%d):\n", len(r.Privileges))
	if len(r.Privileges) == 0 {
		return nil
	}
	return tf.Format(r.Privileges, rolePrivilegeColumns)
}

var rolePrivilegeColumns = []output.Column{
	{Header: "NAME", Field: "name", Width: 36},
	{Header: "SERVICE", Field: "service", Width: 24},
	{Header: "STATUS", Field: "status", Width: 10, Func: privilegeStatusFunc},
	{Header: "ID", Field: "id", Width: 24},
	{Header: "DESCRIPTION", Field: "description"},
}

var roleDetailColumns = []output.Column{
	{Header: "ID", Field: "id"},
	{Header: "NAME", Field: "roleName"},
	{Header: "DISPLAY NAME", Field: "displayName"},
	{Header: "SYSTEM", Field: "systemRole"},
	{Header: "STATUS", Field: "status"},
	{Header: "DESCRIPTION", Field: "description"},
	{Header: "UPDATED BY", Field: "updatedBy"},
	{Header: "UPDATED", Field: "updateTime"},
}

func newRoleCreateCmd() *cobra.Command {
	var (
		fromFile   string
		fromRole   string
		req        client.CreateRoleRequest
		privileges []string
	)
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a custom role",
		Long: `Create a custom role.

On a terminal, when --name or privileges are missing (no --privilege,
--from-role or --from-file), an interactive wizard asks for the name,
description and an optional role to copy privileges from, then opens the
privilege editor.`,
		Example: `  iics role create                         # interactive wizard
  iics role create --name "CAI Viewer" --description "View CAI assets" \
    --privilege view.ai.designer --privilege view.ai.assets
  iics role create --name "Designer Copy" --from-role Designer
  iics role create --from-file cai-viewer-role.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			var body client.CreateRoleRequest
			if fromFile != "" {
				data, err := os.ReadFile(fromFile)
				if err != nil {
					return fmt.Errorf("reading file: %w", err)
				}
				if err := json.Unmarshal(data, &body); err != nil {
					return fmt.Errorf("parsing JSON: %w", err)
				}
			}
			if req.Name != "" {
				body.Name = req.Name
			}
			if req.Description != "" {
				body.Description = req.Description
			}
			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			wizard := fromFile == "" && isInteractiveTTY() &&
				(body.Name == "" || (len(privileges) == 0 && fromRole == ""))
			if wizard {
				ok, werr := runRoleCreateWizard(ctx, c, &body, fromRole, privileges)
				if werr != nil {
					return werr
				}
				if !ok {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Canceled.")
					return nil
				}
			} else {
				refs := append(body.Privileges, privileges...)
				if fromRole != "" {
					src, ferr := c.FindRole(ctx, fromRole, true)
					if ferr != nil {
						return ferr
					}
					for _, p := range src.Privileges {
						refs = append(refs, p.ID)
					}
					if body.Description == "" {
						body.Description = src.Description
					}
				}
				if body.Name == "" {
					return fmt.Errorf("--name is required")
				}
				if len(refs) == 0 {
					return fmt.Errorf("at least one --privilege or --from-role is required")
				}
				resolved, rerr := c.ResolvePrivileges(ctx, refs)
				if rerr != nil {
					return rerr
				}
				body.Privileges = make([]string, len(resolved))
				for i, p := range resolved {
					body.Privileges[i] = p.ID
				}
			}

			created, err := c.CreateRole(ctx, &body)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Role created: %s (ID: %s, %d privileges)\n", created.RoleName, created.ID, len(body.Privileges))
			return nil
		},
	}
	cmd.Flags().StringVar(&req.Name, "name", "", "role name (prompted on a terminal when omitted)")
	cmd.Flags().StringVar(&req.Description, "description", "", "role description")
	cmd.Flags().StringSliceVar(&privileges, "privilege", nil, "privilege name or ID to assign (repeatable or comma-separated)")
	cmd.Flags().StringVar(&fromRole, "from-role", "", "copy privileges (and description) from an existing role, by name or ID")
	cmd.Flags().StringVar(&fromFile, "from-file", "", "JSON file with role definition (name, description, privileges)")
	return cmd
}

// newRoleEditCmd opens the interactive privilege editor for a custom role.
func newRoleEditCmd() *cobra.Command {
	var ref client.RoleRef
	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Interactively add and remove privileges of a custom role",
		Long: `Open the interactive privilege editor for a custom role. Privileges are
grouped by service, then by object with one column per action (view, create,
update, delete, execute, change permission). Changes are applied after a
review step. Requires a terminal.`,
		Example: `  iics role edit                      # pick a role from a list
  iics role edit --name "CAI Viewer"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			return editRolePrivilegesInteractive(context.Background(), c, cmd.OutOrStdout(), ref, privmatrix.ModeEdit)
		},
	}
	cmd.Flags().StringVar(&ref.ID, "id", "", "role ID")
	cmd.Flags().StringVar(&ref.Name, "name", "", "role name")
	cmd.MarkFlagsMutuallyExclusive("id", "name")
	return cmd
}

// newRolePrivilegesCmd builds add-privileges / remove-privileges. The API has
// no general role update endpoint; privileges are the only mutable part.
func newRolePrivilegesCmd(use, short, verb string) *cobra.Command {
	var (
		ref        client.RoleRef
		privileges []string
	)
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Long: fmt.Sprintf(`%s.

Without --privilege on a terminal, the interactive privilege editor opens
(and a role picker when --id and --name are omitted).`, short),
		Example: fmt.Sprintf(`  iics role %[1]s                     # interactive editor
  iics role %[1]s --name "CAI Viewer" --privilege view.ai.console
  iics role %[1]s --id <role-id> --privilege create.data.transfer.task,delete.data.transfer.task`, use),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(privileges) == 0 && isInteractiveTTY() {
				c, err := getClient(cmd)
				if err != nil {
					return err
				}
				mode := privmatrix.ModeAdd
				if use == "remove-privileges" {
					mode = privmatrix.ModeRemove
				}
				return editRolePrivilegesInteractive(context.Background(), c, cmd.OutOrStdout(), ref, mode)
			}
			if ref.ID == "" && ref.Name == "" {
				return fmt.Errorf("--id or --name is required")
			}
			if len(privileges) == 0 {
				return fmt.Errorf("at least one --privilege is required")
			}
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			resolved, err := c.ResolvePrivileges(context.Background(), privileges)
			if err != nil {
				return err
			}
			names := make([]string, len(resolved))
			for i, p := range resolved {
				names[i] = p.Name
			}
			if use == "add-privileges" {
				err = c.AddRolePrivileges(context.Background(), ref, names)
			} else {
				err = c.RemoveRolePrivileges(context.Background(), ref, names)
			}
			if err != nil {
				return err
			}
			target := ref.Name
			if target == "" {
				target = ref.ID
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s %d privilege(s) for role %s: %s\n", verb, len(names), target, strings.Join(names, ", "))
			return nil
		},
	}
	cmd.Flags().StringVar(&ref.ID, "id", "", "role ID")
	cmd.Flags().StringVar(&ref.Name, "name", "", "role name")
	cmd.Flags().StringSliceVar(&privileges, "privilege", nil, "privilege name or ID (repeatable or comma-separated)")
	cmd.MarkFlagsMutuallyExclusive("id", "name")
	return cmd
}

func newRoleDeleteCmd() *cobra.Command {
	var (
		id   string
		name string
		yes  bool
	)
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a custom role",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" && name == "" {
				return fmt.Errorf("--id or --name is required")
			}
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			label := id
			if name != "" {
				role, err := c.GetRole(context.Background(), client.RoleGetOptions{Name: name})
				if err != nil {
					return err
				}
				id = role.ID
				label = fmt.Sprintf("%s (ID: %s)", role.RoleName, role.ID)
			}
			if !yes {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Are you sure you want to delete role %s? [y/N]: ", label)
				var confirm string
				_, _ = fmt.Scanln(&confirm)
				if confirm != "y" && confirm != "Y" {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Canceled.")
					return nil
				}
			}
			if err := c.DeleteRole(context.Background(), id); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Role deleted: %s\n", label)
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "role ID")
	cmd.Flags().StringVar(&name, "name", "", "role name")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompt")
	cmd.MarkFlagsMutuallyExclusive("id", "name")
	return cmd
}
