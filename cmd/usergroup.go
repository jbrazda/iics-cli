package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/config"
	"github.com/jbrazda/iics-cli/internal/output"
	"github.com/spf13/cobra"
)

// allUsergroupColumns defines all available columns for usergroup output.
var allUsergroupColumns = map[string]output.Column{
	"id":            {Header: "ID", Field: "id", Width: 24},
	"userGroupName": {Header: "NAME", Field: "userGroupName", Width: 30},
	"description":   {Header: "DESCRIPTION", Field: "description", Width: 30},
	"updatedBy":     {Header: "UPDATED BY", Field: "updatedBy", Width: 30},
	"updateTime":    {Header: "UPDATED", Field: "updateTime", Width: 24},
	"createdBy":     {Header: "CREATED BY", Field: "createdBy", Width: 30},
	"createTime":    {Header: "CREATED", Field: "createTime", Width: 24},
	"countMembers":  {Header: "MEMBERS", Field: "countMembers", Width: 8},
	"countRoles":    {Header: "ROLES", Field: "countRoles", Width: 8},
}

func columnsFromFields(fields string) []output.Column {
	var columns []output.Column
	for _, name := range strings.Split(fields, ",") {
		name = strings.TrimSpace(name)
		if col, ok := allUsergroupColumns[name]; ok {
			columns = append(columns, col)
		}
	}
	return columns
}

func newUsergroupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "group",
		Aliases: []string{"usergroup", "ug"},
		Short:   "Manage user groups",
	}

	cmd.AddCommand(newUsergroupListCmd())
	cmd.AddCommand(newUsergroupGetCmd())
	cmd.AddCommand(newUsergroupCreateCmd())
	cmd.AddCommand(newUsergroupUpdateCmd())
	cmd.AddCommand(newUsergroupDeleteCmd())
	return cmd
}

const (
	usergroupTableFields = "id,userGroupName,updatedBy,updateTime,countMembers,countRoles"
	usergroupCSVFields   = "id,userGroupName,updatedBy,updateTime,description,countMembers,countRoles"
)

func newUsergroupListCmd() *cobra.Command {
	var opts client.UserGroupListOptions
	var fields string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List user groups",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			groups, err := c.ListUserGroups(context.Background(), opts)
			if err != nil {
				return err
			}
			f, err := getFormatter()
			if err != nil {
				return err
			}
			if fields == "" {
				if outputFmt == "csv" {
					fields = usergroupCSVFields
				} else {
					fields = usergroupTableFields
				}
			}
			return f.Format(groups, columnsFromFields(fields))
		},
	}

	cmd.Flags().IntVar(&opts.Limit, "limit", 200, "max results")
	cmd.Flags().IntVar(&opts.Skip, "skip", 0, "number of results to skip")
	cmd.Flags().StringVarP(&opts.Query, "query", "q", "", `filter query (e.g. userGroupName=="Administrator")`)
	cmd.Flags().StringVar(&fields, "fields", "", "comma-separated list of fields to display (default varies by output format)")
	return cmd
}

func newUsergroupGetCmd() *cobra.Command {
	var id, name string
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get user group details",
		Example: `  iics group get --id <group-id>
  iics group get --name "Data Engineering"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			group, err := resolveUserGroup(context.Background(), c, id, name, "view")
			if err != nil {
				return err
			}
			f, err := getFormatter()
			if err != nil {
				return err
			}
			columns := []output.Column{
				{Header: "ID", Field: "id", Width: 24},
				{Header: "NAME", Field: "userGroupName", Width: 30},
				{Header: "DESCRIPTION", Field: "description"},
			}
			return f.Format(group, columns)
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "user group ID")
	cmd.Flags().StringVar(&name, "name", "", "user group name")
	cmd.MarkFlagsMutuallyExclusive("id", "name")
	return cmd
}

// groupOpResult is one row of a bulk create/update summary.
type groupOpResult struct {
	Name   string `json:"name"`
	ID     string `json:"id"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

var groupOpResultCols = []output.Column{
	{Header: "NAME", Field: "name", Width: 30},
	{Header: "ID", Field: "id", Width: 24},
	{Header: "STATUS", Field: "status", Width: 10},
	{Header: "DETAIL", Field: "detail"},
}

// groupInput is the parsed result of readGroupInput.
type groupInput struct {
	groups  []client.UserGroup
	present bool // an input source (file or piped stdin) was found
	isArray bool // the JSON was an array (bulk)
}

// readGroupInput returns the group payload(s) from --from-file or piped stdin.
func readGroupInput(fromFile string) (groupInput, error) {
	var data []byte
	switch {
	case fromFile != "":
		b, err := os.ReadFile(fromFile)
		if err != nil {
			return groupInput{}, fmt.Errorf("reading file: %w", err)
		}
		data = b
	case hasPipedStdin():
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return groupInput{}, fmt.Errorf("reading stdin: %w", err)
		}
		data = b
	default:
		return groupInput{}, nil
	}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return groupInput{}, nil
	}
	if trimmed[0] == '[' {
		var groups []client.UserGroup
		if err := json.Unmarshal(data, &groups); err != nil {
			return groupInput{}, fmt.Errorf("parsing JSON array: %w", err)
		}
		return groupInput{groups: groups, present: true, isArray: true}, nil
	}
	var g client.UserGroup
	if err := json.Unmarshal(data, &g); err != nil {
		return groupInput{}, fmt.Errorf("parsing JSON: %w", err)
	}
	return groupInput{groups: []client.UserGroup{g}, present: true}, nil
}

// printGroupOpResults renders the bulk summary and returns an error when any row failed.
func printGroupOpResults(w io.Writer, results []groupOpResult) error {
	cfg, _ := loadConfig()
	tf := output.New(output.FormatTable, w, resolveTableStyle(cfg))
	_ = tf.Format(results, groupOpResultCols)
	failed := 0
	for _, r := range results {
		if r.Status == "error" {
			failed++
		}
	}
	if failed > 0 {
		return fmt.Errorf("%d of %d group(s) failed", failed, len(results))
	}
	return nil
}

func newUsergroupCreateCmd() *cobra.Command {
	var (
		fromFile    string
		interactive bool
	)
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create one or more user groups",
		Long: `Create user groups.

Provide a JSON definition via --from-file or piped stdin (a single object or an
array for bulk creation), or run interactively (--interactive/-i, or omit all
input on a terminal) to be prompted for the name, description, and roles.`,
		Example: `  iics group create --from-file group.json
  cat groups.json | iics group create
  iics group create -i`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			in, err := readGroupInput(fromFile)
			if err != nil {
				return err
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			if !in.present {
				if !interactive && !config.IsTerminal() {
					return fmt.Errorf("provide --from-file, pipe JSON to stdin, or use --interactive")
				}
				var g client.UserGroup
				if werr := runGroupWizard(ctx, c, &g, true); werr != nil {
					return werr
				}
				in.groups = []client.UserGroup{g}
			}

			if !in.isArray {
				created, err := c.CreateUserGroup(ctx, &in.groups[0])
				if err != nil {
					return err
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "User group created: %s (ID: %s)\n", created.UserGroupName, created.ID)
				return nil
			}

			results := make([]groupOpResult, 0, len(in.groups))
			for i := range in.groups {
				g := in.groups[i]
				created, cerr := c.CreateUserGroup(ctx, &g)
				res := groupOpResult{Name: g.ResolvedName(), Status: "created"}
				if cerr != nil {
					res.Status = "error"
					res.Detail = cerr.Error()
				} else {
					res.ID = created.ID
				}
				results = append(results, res)
			}
			return printGroupOpResults(cmd.OutOrStdout(), results)
		},
	}
	cmd.Flags().StringVar(&fromFile, "from-file", "", "JSON file (object or array) with group definition(s); omit to read piped stdin")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "create interactively")
	return cmd
}

func newUsergroupUpdateCmd() *cobra.Command {
	var (
		id          string
		name        string
		fromFile    string
		interactive bool
	)
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update one or more user groups",
		Long: `Update user groups.

Single update: identify the group with --id or --name (or pick from a list when
neither is given on a terminal), then supply changes via --from-file / stdin or
--interactive.

Bulk update: pipe or pass a JSON array; each element is matched to an existing
group by its "id", or by "userGroupName" when no id is present.`,
		Example: `  iics group update --id <group-id> --from-file changes.json
  iics group update --name "Data Engineering" -i
  cat groups.json | iics group update`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			in, err := readGroupInput(fromFile)
			if err != nil {
				return err
			}
			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			if in.isArray {
				return runBulkGroupUpdate(ctx, cmd, c, in.groups)
			}

			// Single update: resolve the target group.
			target, err := resolveUserGroup(ctx, c, id, name, "edit")
			if err != nil {
				return err
			}

			switch {
			case in.present:
				applyGroupPatch(target, &in.groups[0])
			case interactive || config.IsTerminal():
				if werr := runGroupWizard(ctx, c, target, false); werr != nil {
					return werr
				}
			default:
				return fmt.Errorf("provide --from-file, pipe JSON to stdin, or use --interactive")
			}

			updated, err := c.UpdateUserGroup(ctx, target.ID, target)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "User group updated: %s (ID: %s)\n", updated.UserGroupName, updated.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "user group ID")
	cmd.Flags().StringVar(&name, "name", "", "user group name")
	cmd.Flags().StringVar(&fromFile, "from-file", "", "JSON file (object or array) with updates; omit to read piped stdin")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "edit interactively")
	cmd.MarkFlagsMutuallyExclusive("id", "name")
	return cmd
}

// applyGroupPatch overlays non-zero fields from patch onto target.
func applyGroupPatch(target, patch *client.UserGroup) {
	if n := patch.ResolvedName(); n != "" {
		target.UserGroupName = n
	}
	if patch.Description != "" {
		target.Description = patch.Description
	}
	if patch.Roles != nil {
		target.Roles = patch.Roles
	}
	if patch.Users != nil {
		target.Users = patch.Users
	}
}

func runBulkGroupUpdate(ctx context.Context, cmd *cobra.Command, c *client.Client, groups []client.UserGroup) error {
	results := make([]groupOpResult, 0, len(groups))
	for i := range groups {
		patch := groups[i]
		res := groupOpResult{Name: patch.ResolvedName(), ID: patch.ID, Status: "updated"}

		var existing *client.UserGroup
		var lerr error
		switch {
		case patch.ID != "":
			existing, lerr = c.GetUserGroup(ctx, patch.ID)
		case patch.ResolvedName() != "":
			existing, lerr = c.GetUserGroupByName(ctx, patch.ResolvedName())
		default:
			res.Status = "error"
			res.Detail = "element has neither id/name nor userGroupName"
			results = append(results, res)
			continue
		}
		if lerr != nil {
			res.Status = "error"
			res.Detail = lerr.Error()
			results = append(results, res)
			continue
		}

		applyGroupPatch(existing, &patch)
		updated, uerr := c.UpdateUserGroup(ctx, existing.ID, existing)
		if uerr != nil {
			res.Status = "error"
			res.Detail = uerr.Error()
		} else {
			res.ID = updated.ID
			res.Name = updated.UserGroupName
		}
		results = append(results, res)
	}
	return printGroupOpResults(cmd.OutOrStdout(), results)
}

func newUsergroupDeleteCmd() *cobra.Command {
	var (
		id   string
		name string
		yes  bool
	)
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a user group",
		Example: `  iics group delete --id <group-id>
  iics group delete --name "Data Engineering" --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()
			group, err := resolveUserGroup(ctx, c, id, name, "delete")
			if err != nil {
				return err
			}
			if !yes {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Are you sure you want to delete user group %s (%s)? [y/N]: ", group.UserGroupName, group.ID)
				var confirm string
				_, _ = fmt.Scanln(&confirm)
				if confirm != "y" && confirm != "Y" {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Canceled.")
					return nil
				}
			}
			if err := c.DeleteUserGroup(ctx, group.ID); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "User group deleted: %s (%s)\n", group.UserGroupName, group.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "user group ID")
	cmd.Flags().StringVar(&name, "name", "", "user group name")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompt")
	cmd.MarkFlagsMutuallyExclusive("id", "name")
	return cmd
}
