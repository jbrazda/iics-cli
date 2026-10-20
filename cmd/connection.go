package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/output"
	"github.com/spf13/cobra"
)

func newConnectionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "connection",
		Aliases: []string{"conn"},
		Short:   "Manage connections",
	}

	cmd.AddCommand(newConnectionListCmd())
	cmd.AddCommand(newConnectionGetCmd())
	cmd.AddCommand(newConnectionCreateCmd())
	cmd.AddCommand(newConnectionUpdateCmd())
	cmd.AddCommand(newConnectionDeleteCmd())
	cmd.AddCommand(newConnectionTestCmd())
	cmd.AddCommand(newConnectionSearchCmd())
	cmd.AddCommand(newConnectionObjectsCmd())
	cmd.AddCommand(newConnectionMetadataCmd())
	return cmd
}

func newConnectionListCmd() *cobra.Command {
	var opts client.ConnectionListOptions

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List connections",
		Example: `  iics connection list
  iics connection list --type TOOLKIT
  iics connection list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			conns, err := c.ListConnections(context.Background(), opts)
			if err != nil {
				return err
			}

			f, err := getFormatter()
			if err != nil {
				return err
			}

			columns := []output.Column{
				{Header: "ID", Field: "id", Width: 24, Priority: 5, Shrink: output.ShrinkNever},
				{Header: "NAME", Field: "name", Width: 30, Priority: 1},
				{Header: "TYPE", Field: "type", Width: 15, Priority: 2, Shrink: output.ShrinkNever},
				{Header: "UPDATED BY", Field: "updatedBy", Width: 20, Priority: 3, Shrink: output.ShrinkTruncate},
				{Header: "UPDATED", Field: "updateTime", Width: 20, Priority: 2, Shrink: output.ShrinkNever},
			}

			return f.Format(conns, columns)
		},
	}

	cmd.Flags().StringVar(&opts.Type, "type", "", "filter by connection type (exact, case-insensitive)")
	cmd.Flags().StringVar(&opts.Name, "name", "", "filter by name substring (case-insensitive)")
	cmd.Flags().IntVar(&opts.Limit, "limit", 200, "max results")
	cmd.Flags().IntVar(&opts.Skip, "skip", 0, "number of results to skip")

	return cmd
}

func newConnectionGetCmd() *cobra.Command {
	var id, name string

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get connection details",
		Example: `  iics connection get --id <connection-id>
  iics connection get --name "Salesforce Prod" --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" && name == "" {
				return fmt.Errorf("--id or --name is required")
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			conn, err := getConnectionByIDOrName(context.Background(), c, id, name)
			if err != nil {
				return err
			}

			f, err := getFormatter()
			if err != nil {
				return err
			}

			columns := []output.Column{
				{Header: "ID", Field: "id", Width: 24},
				{Header: "NAME", Field: "name", Width: 30},
				{Header: "TYPE", Field: "type", Width: 15},
				{Header: "HOST", Field: "host", Width: 20},
				{Header: "RUNTIME ENV", Field: "runtimeEnvironmentId", Width: 24},
				{Header: "UPDATED", Field: "updateTime", Width: 20},
				{Header: "DESCRIPTION", Field: "description"},
			}

			return f.Format(conn, columns)
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "connection ID")
	cmd.Flags().StringVar(&name, "name", "", "connection name")
	cmd.MarkFlagsMutuallyExclusive("id", "name")
	return cmd
}

// getConnectionByIDOrName fetches a connection by ID, or by name when id is empty.
func getConnectionByIDOrName(ctx context.Context, c *client.Client, id, name string) (*client.Connection, error) {
	if id != "" {
		return c.GetConnection(ctx, id)
	}
	return c.GetConnectionByName(ctx, name)
}

// resolveConnectionID returns id, or looks up the ID of the named connection.
func resolveConnectionID(ctx context.Context, c *client.Client, id, name string) (string, error) {
	if id != "" {
		return id, nil
	}
	if name == "" {
		return "", fmt.Errorf("--id or --name is required")
	}
	conn, err := c.GetConnectionByName(ctx, name)
	if err != nil {
		return "", err
	}
	return conn.ID, nil
}

// readConnectionDef reads a JSON connection definition as a generic map so
// connector-specific attributes are sent unchanged.
func readConnectionDef(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file %s: %w", path, err)
	}
	var def map[string]interface{}
	if err := json.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("parsing connection JSON: %w", err)
	}
	return def, nil
}

func newConnectionCreateCmd() *cobra.Command {
	var fromFile string

	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a connection",
		Example: `  iics connection create --from-file connection.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if fromFile == "" {
				return fmt.Errorf("--from-file is required")
			}

			def, err := readConnectionDef(fromFile)
			if err != nil {
				return err
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			created, err := c.CreateConnection(context.Background(), def)
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Connection created: %s (ID: %s)\n", created.Name, created.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&fromFile, "from-file", "", "JSON file with connection definition (required)")
	return cmd
}

func newConnectionUpdateCmd() *cobra.Command {
	var (
		id       string
		fromFile string
		partial  bool
	)

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a connection",
		Long: `Update a connection from a JSON file.

Without --partial the definition replaces the connection: every attribute
must be present. With --partial only the attributes in the file change.
"connection get" returns the password masked, so do not send its output back
as a full update; use --partial and leave the password out.`,
		Example: `  iics connection update --id <connection-id> --from-file connection.json
  iics connection update --id <connection-id> --partial --from-file changes.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" {
				return fmt.Errorf("--id is required")
			}
			if fromFile == "" {
				return fmt.Errorf("--from-file is required")
			}

			def, err := readConnectionDef(fromFile)
			if err != nil {
				return err
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			updated, err := c.UpdateConnection(context.Background(), id, def, partial)
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Connection updated: %s (ID: %s)\n", updated.Name, updated.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "connection ID (required)")
	cmd.Flags().StringVar(&fromFile, "from-file", "", "JSON file with connection updates (required)")
	cmd.Flags().BoolVar(&partial, "partial", false, "change only the attributes in the file (Update-Mode: PARTIAL)")
	return cmd
}

func newConnectionDeleteCmd() *cobra.Command {
	var (
		id  string
		yes bool
	)

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a connection",
		Example: `  iics connection delete --id <connection-id>
  iics connection delete --id <connection-id> --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" {
				return fmt.Errorf("--id is required")
			}

			if !yes && !confirmAction(cmd, fmt.Sprintf("Are you sure you want to delete connection %s?", id)) {
				return nil
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			if err := c.DeleteConnection(context.Background(), id); err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Connection deleted: %s\n", id)
			return nil
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "connection ID (required)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompt")
	return cmd
}

func newConnectionTestCmd() *cobra.Command {
	var id, name string

	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test a connection",
		Example: `  iics connection test --id <connection-id>
  iics connection test --name "Salesforce Prod"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			connID, err := resolveConnectionID(ctx, c, id, name)
			if err != nil {
				return err
			}

			res, err := c.TestConnection(ctx, connID)
			if err != nil {
				return err
			}

			if outputFmt != "" && outputFmt != "table" {
				f, ferr := getFormatter()
				if ferr != nil {
					return ferr
				}
				return f.Format(res, nil)
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), res.Description)
			return nil
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "connection ID")
	cmd.Flags().StringVar(&name, "name", "", "connection name")
	cmd.MarkFlagsMutuallyExclusive("id", "name")
	return cmd
}

func newConnectionSearchCmd() *cobra.Command {
	var opts client.ConnectionSearchOptions

	cmd := &cobra.Command{
		Use:   "search",
		Short: "List connections of a type available to a runtime environment or agent",
		Example: `  iics connection search --type Salesforce --runtime-id <runtime-environment-id>
  iics connection search --type TOOLKIT --agent-id <agent-id> --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.UIType == "" {
				return fmt.Errorf("--type is required")
			}
			if opts.RuntimeEnvironmentID == "" && opts.AgentID == "" {
				return fmt.Errorf("--runtime-id or --agent-id is required")
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			conns, err := c.SearchConnections(context.Background(), opts)
			if err != nil {
				return err
			}

			f, err := getFormatter()
			if err != nil {
				return err
			}

			columns := []output.Column{
				{Header: "ID", Field: "id", Width: 24, Priority: 5, Shrink: output.ShrinkNever},
				{Header: "NAME", Field: "name", Width: 30, Priority: 1},
				{Header: "TYPE", Field: "type", Width: 15, Priority: 2, Shrink: output.ShrinkNever},
				{Header: "RUNTIME ENV", Field: "runtimeEnvironmentId", Width: 24, Priority: 4, Shrink: output.ShrinkNever},
				{Header: "UPDATED", Field: "updateTime", Width: 20, Priority: 3, Shrink: output.ShrinkNever},
			}

			return f.Format(conns, columns)
		},
	}

	cmd.Flags().StringVar(&opts.UIType, "type", "", "connection type, e.g. Salesforce, CSVFile, TOOLKIT (required)")
	cmd.Flags().StringVar(&opts.RuntimeEnvironmentID, "runtime-id", "", "runtime environment ID (takes precedence over --agent-id)")
	cmd.Flags().StringVar(&opts.AgentID, "agent-id", "", "Secure Agent ID")
	return cmd
}

func newConnectionObjectsCmd() *cobra.Command {
	var opts client.ConnectionObjectsOptions

	cmd := &cobra.Command{
		Use:   "objects",
		Short: "List the source or target objects of a connection",
		Long: `List the source objects (or target objects with --target) available
through a connection, such as tables or Salesforce objects. The API returns
at most 200 objects; use --search-pattern to narrow the list.`,
		Example: `  iics connection objects --id <connection-id>
  iics connection objects --name "Salesforce Prod" --target --search-pattern Account`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.ID == "" && opts.Name == "" {
				return fmt.Errorf("--id or --name is required")
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			items, err := c.ListConnectionObjects(context.Background(), opts)
			if err != nil {
				return err
			}

			f, err := getFormatter()
			if err != nil {
				return err
			}

			columns := []output.Column{
				{Header: "ID", Field: "id", Width: 6},
				{Header: "NAME", Field: "name"},
			}

			return f.Format(items, columns)
		},
	}

	cmd.Flags().StringVar(&opts.ID, "id", "", "connection ID")
	cmd.Flags().StringVar(&opts.Name, "name", "", "connection name")
	cmd.Flags().BoolVar(&opts.Target, "target", false, "list target objects instead of source objects")
	cmd.Flags().StringVar(&opts.SearchPattern, "search-pattern", "", "case-insensitive substring filter on object names")
	cmd.MarkFlagsMutuallyExclusive("id", "name")
	return cmd
}

func newConnectionMetadataCmd() *cobra.Command {
	var (
		id, name string
		target   bool
	)

	cmd := &cobra.Command{
		Use:   "metadata",
		Short: "Show the source or target runtime attributes of a connection",
		Example: `  iics connection metadata --id <connection-id>
  iics connection metadata --name "Salesforce Prod" --target`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			connID, err := resolveConnectionID(ctx, c, id, name)
			if err != nil {
				return err
			}

			attrs, err := c.GetConnectionMetadata(ctx, connID, target)
			if err != nil {
				return err
			}

			f, err := getFormatter()
			if err != nil {
				return err
			}

			columns := []output.Column{
				{Header: "NAME", Field: "name", Width: 30, Priority: 1},
				{Header: "DATA TYPE", Field: "dataType", Width: 10, Priority: 2, Shrink: output.ShrinkNever},
				{Header: "MANDATORY", Field: "mandatory", Width: 9, Priority: 3, Shrink: output.ShrinkNever},
				{Header: "DEFAULT", Field: "defaultValue", Width: 15, Priority: 2},
				{Header: "VALUES", Field: "possibleValues", MaxWidth: 60, Priority: 4, Shrink: output.ShrinkWrap, Func: joinedListFunc("possibleValues")},
			}

			return f.Format(attrs, columns)
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "connection ID")
	cmd.Flags().StringVar(&name, "name", "", "connection name")
	cmd.Flags().BoolVar(&target, "target", false, "show target attributes instead of source attributes")
	cmd.MarkFlagsMutuallyExclusive("id", "name")
	return cmd
}

// joinedListFunc renders a list field of a table row as comma-separated text.
func joinedListFunc(field string) func(row interface{}) string {
	return func(row interface{}) string {
		m, _ := row.(map[string]interface{})
		items, _ := m[field].([]interface{})
		parts := make([]string, 0, len(items))
		for _, it := range items {
			parts = append(parts, fmt.Sprint(it))
		}
		return strings.Join(parts, ", ")
	}
}
