package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/output"
	"github.com/spf13/cobra"
)

func newPermissionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "permission",
		Aliases: []string{"perm"},
		Short:   "Manage object permissions (ACLs)",
		Long: `Manage object permissions. Each access control list (ACL) entry grants one
user or user group read, update, delete, execute and change-permission rights
on one object (project, folder or asset).

Select the object with --object-id, or with --path and --type.`,
	}
	cmd.AddCommand(newPermissionGetCmd())
	cmd.AddCommand(newPermissionAddCmd())
	cmd.AddCommand(newPermissionUpdateCmd())
	cmd.AddCommand(newPermissionDeleteCmd())
	cmd.AddCommand(newPermissionSetCmd())
	cmd.AddCommand(newPermissionCheckCmd())
	return cmd
}

// objectRef holds the flags that select an object.
type objectRef struct {
	id, path, typ string
}

func (o *objectRef) register(cmd *cobra.Command) {
	cmd.Flags().StringVar(&o.id, "object-id", "", "object ID")
	cmd.Flags().StringVar(&o.path, "path", "", `object path, e.g. "Default/Sales/m_load_orders" (with --type)`)
	cmd.Flags().StringVar(&o.typ, "type", "", "object type for --path, e.g. Project, Folder, DTEMPLATE, MTT, TASKFLOW")
	cmd.MarkFlagsMutuallyExclusive("object-id", "path")
	cmd.MarkFlagsRequiredTogether("path", "type")
}

// resolve returns the object ID and a display label.
func (o *objectRef) resolve(ctx context.Context, c *client.Client) (string, string, error) {
	if o.id != "" {
		return o.id, o.id, nil
	}
	if o.path == "" {
		return "", "", fmt.Errorf("--object-id or --path with --type is required")
	}
	resp, err := c.Lookup(ctx, []client.LookupObject{{Path: o.path, Type: o.typ}})
	if err != nil {
		return "", "", err
	}
	if len(resp.Objects) == 0 || resp.Objects[0].ID == "" {
		return "", "", fmt.Errorf("object %q of type %s not found", o.path, o.typ)
	}
	r := resp.Objects[0]
	return r.ID, fmt.Sprintf("%s (%s)", r.Path, r.Type), nil
}

// principalFlags hold --user / --group.
type principalFlags struct {
	user, group string
}

func (p *principalFlags) register(cmd *cobra.Command) {
	cmd.Flags().StringVar(&p.user, "user", "", "user name")
	cmd.Flags().StringVar(&p.group, "group", "", "user group name")
	cmd.MarkFlagsMutuallyExclusive("user", "group")
}

func (p *principalFlags) principal() (client.ACLPrincipal, bool) {
	switch {
	case p.user != "":
		return client.ACLPrincipal{Type: client.PrincipalUser, Name: p.user}, true
	case p.group != "":
		return client.ACLPrincipal{Type: client.PrincipalGroup, Name: p.group}, true
	}
	return client.ACLPrincipal{}, false
}

var aclColumns = []output.Column{
	{Header: "TYPE", Field: "principal.type", Width: 6},
	{Header: "NAME", Field: "principal.name", Width: 36},
	{Header: "READ", Field: "permissions.read", Width: 5},
	{Header: "UPDATE", Field: "permissions.update", Width: 6},
	{Header: "DELETE", Field: "permissions.delete", Width: 6},
	{Header: "EXECUTE", Field: "permissions.execute", Width: 7},
	{Header: "CHANGE PERM", Field: "permissions.changePermission", Width: 11},
	{Header: "ACL ID", Field: "id", Width: 24},
}

// printACLs lists an object's ACLs in the --output format.
func printACLs(ctx context.Context, c *client.Client, objectID string) error {
	acls, err := c.ListObjectACLs(ctx, objectID)
	if err != nil {
		return err
	}
	f, err := getFormatter()
	if err != nil {
		return err
	}
	if acls == nil {
		acls = []client.ObjectACL{}
	}
	return f.Format(acls, aclColumns)
}

func findACL(acls []client.ObjectACL, p client.ACLPrincipal) (client.ObjectACL, bool) {
	key := client.PrincipalKey(p)
	for _, a := range acls {
		if client.PrincipalKey(a.Principal) == key {
			return a, true
		}
	}
	return client.ObjectACL{}, false
}

func newPermissionGetCmd() *cobra.Command {
	var obj objectRef
	cmd := &cobra.Command{
		Use:   "get",
		Short: "List the ACLs of an object",
		Example: `  iics permission get --object-id <object-id>
  iics permission get --path "Default/Sales" --type Folder -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()
			id, _, err := obj.resolve(ctx, c)
			if err != nil {
				return err
			}
			return printACLs(ctx, c, id)
		},
	}
	obj.register(cmd)
	return cmd
}

func newPermissionAddCmd() *cobra.Command {
	var (
		obj   objectRef
		who   principalFlags
		grant []string
	)
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Grant permissions to a user or user group",
		Long: `Create an ACL for a user or user group on an object. Fails if the principal
already has an ACL on the object; use "permission update" to change it.`,
		Example: `  iics permission add --object-id <id> --group "Data Engineering" --grant read,execute
  iics permission add --path "Default/Sales" --type Folder --user jdoe@example.com --grant all`,
		RunE: func(cmd *cobra.Command, args []string) error {
			p, ok := who.principal()
			if !ok {
				return fmt.Errorf("--user or --group is required")
			}
			perms, err := client.ParseACLPermissions(grant)
			if err != nil {
				return err
			}
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()
			id, label, err := obj.resolve(ctx, c)
			if err != nil {
				return err
			}
			acls, err := c.ListObjectACLs(ctx, id)
			if err != nil {
				return err
			}
			if existing, found := findACL(acls, p); found {
				return fmt.Errorf("%s %q already has an ACL (%s) on %s; use 'permission update'", strings.ToLower(p.Type), existing.Principal.Name, existing.ID, label)
			}
			if _, err = c.CreateObjectACL(ctx, id, client.ObjectACL{Principal: p, Permissions: perms}); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Granted %s to %s %s on %s\n", perms, strings.ToLower(p.Type), p.Name, label)
			return printACLs(ctx, c, id)
		},
	}
	obj.register(cmd)
	who.register(cmd)
	cmd.Flags().StringSliceVar(&grant, "grant", nil, "permissions: read, update, delete, execute, changePermission, all (comma-separated)")
	_ = cmd.MarkFlagRequired("grant")
	return cmd
}

func newPermissionUpdateCmd() *cobra.Command {
	var (
		obj   objectRef
		who   principalFlags
		aclID string
		grant []string
	)
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Set the permissions of an existing ACL",
		Long: `Replace the permissions of an existing ACL, selected by --acl-id or by
--user / --group. --grant is the complete new permission set.`,
		Example: `  iics permission update --object-id <id> --group "Data Engineering" --grant read
  iics permission update --object-id <id> --acl-id <acl-id> --grant read,update,execute`,
		RunE: func(cmd *cobra.Command, args []string) error {
			p, byPrincipal := who.principal()
			if !byPrincipal && aclID == "" {
				return fmt.Errorf("--acl-id, --user or --group is required")
			}
			perms, err := client.ParseACLPermissions(grant)
			if err != nil {
				return err
			}
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()
			id, label, err := obj.resolve(ctx, c)
			if err != nil {
				return err
			}
			acls, err := c.ListObjectACLs(ctx, id)
			if err != nil {
				return err
			}
			var target client.ObjectACL
			found := false
			if byPrincipal {
				target, found = findACL(acls, p)
			} else {
				for _, a := range acls {
					if a.ID == aclID {
						target, found = a, true
					}
				}
			}
			if !found {
				return fmt.Errorf("no matching ACL on %s; use 'permission add' to create one", label)
			}
			target.Permissions = perms
			if err = c.UpdateObjectACL(ctx, id, target.ID, target); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Set %s %s on %s to %s\n", strings.ToLower(target.Principal.Type), target.Principal.Name, label, perms)
			return printACLs(ctx, c, id)
		},
	}
	obj.register(cmd)
	who.register(cmd)
	cmd.Flags().StringVar(&aclID, "acl-id", "", "ACL ID")
	cmd.Flags().StringSliceVar(&grant, "grant", nil, "complete permission set: read, update, delete, execute, changePermission, all, none")
	_ = cmd.MarkFlagRequired("grant")
	cmd.MarkFlagsMutuallyExclusive("acl-id", "user", "group")
	return cmd
}

func newPermissionDeleteCmd() *cobra.Command {
	var (
		obj   objectRef
		who   principalFlags
		aclID string
		all   bool
		yes   bool
	)
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete an ACL, or all ACLs of an object",
		Example: `  iics permission delete --object-id <id> --group "Data Engineering"
  iics permission delete --object-id <id> --acl-id <acl-id>
  iics permission delete --object-id <id> --all --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			p, byPrincipal := who.principal()
			if !byPrincipal && aclID == "" && !all {
				return fmt.Errorf("--acl-id, --user, --group or --all is required")
			}
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()
			id, label, err := obj.resolve(ctx, c)
			if err != nil {
				return err
			}
			if all {
				if !yes && !confirmAction(cmd, fmt.Sprintf("Delete ALL permissions on %s?", label)) {
					return nil
				}
				if err = c.DeleteAllObjectACLs(ctx, id); err != nil {
					return err
				}
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Deleted all ACLs on %s\n", label)
				return printACLs(ctx, c, id)
			}
			target := client.ObjectACL{ID: aclID}
			if byPrincipal {
				acls, lerr := c.ListObjectACLs(ctx, id)
				if lerr != nil {
					return lerr
				}
				var found bool
				if target, found = findACL(acls, p); !found {
					return fmt.Errorf("%s %q has no ACL on %s", strings.ToLower(p.Type), p.Name, label)
				}
			}
			desc := target.ID
			if target.Principal.Name != "" {
				desc = fmt.Sprintf("%s %s", strings.ToLower(target.Principal.Type), target.Principal.Name)
			}
			if !yes && !confirmAction(cmd, fmt.Sprintf("Delete the ACL of %s on %s?", desc, label)) {
				return nil
			}
			if err = c.DeleteObjectACL(ctx, id, target.ID); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Deleted ACL of %s on %s\n", desc, label)
			return printACLs(ctx, c, id)
		},
	}
	obj.register(cmd)
	who.register(cmd)
	cmd.Flags().StringVar(&aclID, "acl-id", "", "ACL ID")
	cmd.Flags().BoolVar(&all, "all", false, "delete every ACL of the object")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompt")
	cmd.MarkFlagsMutuallyExclusive("acl-id", "user", "group", "all")
	return cmd
}

func newPermissionSetCmd() *cobra.Command {
	var (
		obj      objectRef
		fromFile string
		prune    bool
		dryRun   bool
		yes      bool
	)
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Apply ACLs from a JSON file",
		Long: `Make an object's ACLs match a JSON file: principals in the file that have no
ACL are added, and principals whose permissions differ are updated. With
--prune, ACLs of principals not in the file are deleted (asks for
confirmation unless --yes). --dry-run shows the changes without applying them.

The file is a JSON array in the same shape as "permission get -o json";
"id" values are ignored:

  [
    {"principal": {"type": "GROUP", "name": "Data Engineering"},
     "permissions": {"read": true, "update": true, "delete": false,
                     "execute": true, "changePermission": false}}
  ]`,
		Example: `  iics permission set --object-id <id> --from-file acls.json --dry-run
  iics permission set --path "Default/Sales" --type Folder --from-file acls.json --prune`,
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := readInputFile(fromFile)
			if err != nil {
				return err
			}
			var desired []client.ObjectACL
			if err = json.Unmarshal(data, &desired); err != nil {
				return fmt.Errorf("parsing JSON (expected an array of ACLs): %w", err)
			}
			for i, d := range desired {
				t := strings.ToUpper(d.Principal.Type)
				if (t != client.PrincipalUser && t != client.PrincipalGroup) || strings.TrimSpace(d.Principal.Name) == "" {
					return fmt.Errorf("entry %d: principal needs type USER or GROUP and a name", i+1)
				}
				desired[i].Principal.Type = t
			}
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()
			id, label, err := obj.resolve(ctx, c)
			if err != nil {
				return err
			}
			current, err := c.ListObjectACLs(ctx, id)
			if err != nil {
				return err
			}
			plan := client.DiffACLs(current, desired, prune)
			w := cmd.ErrOrStderr()
			writeACLPlan(w, plan)
			if plan.Empty() {
				_, _ = fmt.Fprintf(w, "No changes for %s.\n", label)
				return printACLs(ctx, c, id)
			}
			if dryRun {
				_, _ = fmt.Fprintln(w, "Dry run: no changes applied.")
				return nil
			}
			if len(plan.Delete) > 0 && !yes && !confirmAction(cmd, fmt.Sprintf("Delete %d ACL(s) on %s?", len(plan.Delete), label)) {
				return nil
			}
			if err = applyACLPlan(ctx, c, id, plan); err != nil {
				return err
			}
			return printACLs(ctx, c, id)
		},
	}
	obj.register(cmd)
	cmd.Flags().StringVar(&fromFile, "from-file", "", "JSON file with the desired ACLs; - for stdin (required)")
	cmd.Flags().BoolVar(&prune, "prune", false, "delete ACLs of principals not in the file")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show the changes without applying them")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompt for --prune deletions")
	_ = cmd.MarkFlagRequired("from-file")
	return cmd
}

func newPermissionCheckCmd() *cobra.Command {
	var (
		obj       objectRef
		assetType string
	)
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Show your own access to an object",
		Long: `Show the current user's access to an object. With --asset-type on a project
or folder, CREATE reports whether that asset type can be created there.`,
		Example: `  iics permission check --object-id <id>
  iics permission check --path "Default" --type Project --asset-type DTEMPLATE`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()
			id, _, err := obj.resolve(ctx, c)
			if err != nil {
				return err
			}
			access, err := c.CheckObjectAccess(ctx, id, assetType)
			if err != nil {
				return err
			}
			f, err := getFormatter()
			if err != nil {
				return err
			}
			return f.Format(access, []output.Column{
				{Header: "CREATE", Field: "permissions.create"},
				{Header: "READ", Field: "permissions.read"},
				{Header: "UPDATE", Field: "permissions.update"},
				{Header: "DELETE", Field: "permissions.delete"},
				{Header: "EXECUTE", Field: "permissions.execute"},
				{Header: "CHANGE PERM", Field: "permissions.changePermission"},
			})
		},
	}
	obj.register(cmd)
	cmd.Flags().StringVar(&assetType, "asset-type", "", "asset type to check create access for, e.g. DTEMPLATE")
	return cmd
}

// writeACLPlan describes an ACLPlan, one line per change.
func writeACLPlan(w io.Writer, plan client.ACLPlan) {
	for _, a := range plan.Create {
		_, _ = fmt.Fprintf(w, "+ %s %s: %s\n", strings.ToLower(a.Principal.Type), a.Principal.Name, a.Permissions)
	}
	for _, a := range plan.Update {
		_, _ = fmt.Fprintf(w, "~ %s %s: %s\n", strings.ToLower(a.Principal.Type), a.Principal.Name, a.Permissions)
	}
	for _, a := range plan.Delete {
		_, _ = fmt.Fprintf(w, "- %s %s\n", strings.ToLower(a.Principal.Type), a.Principal.Name)
	}
}

// applyACLPlan creates, then updates, then deletes ACLs. It stops at the
// first error and reports how many changes were applied.
func applyACLPlan(ctx context.Context, c *client.Client, objectID string, plan client.ACLPlan) error {
	done := 0
	total := len(plan.Create) + len(plan.Update) + len(plan.Delete)
	fail := func(action string, a client.ObjectACL, err error) error {
		return fmt.Errorf("%s ACL of %s %s failed after %d of %d change(s): %w",
			action, strings.ToLower(a.Principal.Type), a.Principal.Name, done, total, err)
	}
	for _, a := range plan.Create {
		if _, err := c.CreateObjectACL(ctx, objectID, a); err != nil {
			return fail("creating", a, err)
		}
		done++
	}
	for _, a := range plan.Update {
		if err := c.UpdateObjectACL(ctx, objectID, a.ID, a); err != nil {
			return fail("updating", a, err)
		}
		done++
	}
	for _, a := range plan.Delete {
		if err := c.DeleteObjectACL(ctx, objectID, a.ID); err != nil {
			return fail("deleting", a, err)
		}
		done++
	}
	return nil
}

// readInputFile reads a file, or stdin for "-".
func readInputFile(path string) ([]byte, error) {
	var (
		data []byte
		err  error
	)
	if path == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}
	return data, nil
}
