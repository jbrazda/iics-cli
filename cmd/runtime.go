package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/charmbracelet/lipgloss"
	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/output"
	"github.com/spf13/cobra"
)

// rtAttr is used to render runtime environment fields as an attribute-value table.
type rtAttr struct {
	Attribute string `json:"attribute"`
	Value     string `json:"value"`
}

var rtAttrCols = []output.Column{
	{Header: "ATTRIBUTE", Field: "attribute", Width: 20},
	{Header: "VALUE", Field: "value", Width: 60},
}

var runtimeAgentCols = []output.Column{
	{Header: "NAME", Field: "name", Width: 25},
	{Header: "HOST", Field: "agentHost", Width: 20},
	{Header: "PLATFORM", Field: "platform", Width: 10},
	{Header: "VERSION", Field: "agentVersion", Width: 10},
	{Header: "ACTIVE", Field: "active", Width: 8, Func: agentActiveFunc},
	{Header: "READY", Field: "readyToRun", Width: 8, Func: agentReadyFunc},
}

func agentActiveFunc(v interface{}) string {
	row, _ := v.(map[string]interface{})
	active, _ := row["active"].(bool)
	if active {
		if noColor {
			return "yes"
		}
		return lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true).Render("yes")
	}
	if noColor {
		return "no"
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render("no")
}

func agentReadyFunc(v interface{}) string {
	row, _ := v.(map[string]interface{})
	ready, _ := row["readyToRun"].(bool)
	if ready {
		if noColor {
			return "yes"
		}
		return lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true).Render("yes")
	}
	if noColor {
		return "no"
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render("no")
}

func agentCountFunc(v interface{}) string {
	row, ok := v.(map[string]interface{})
	if !ok {
		return "0"
	}
	agents, ok := row["agents"].([]interface{})
	if !ok {
		return "0"
	}
	return strconv.Itoa(len(agents))
}

func runtimeEnvAttrs(rt *client.RuntimeEnvironment) []rtAttr {
	shared := "false"
	if rt.IsShared {
		shared = "true"
	}
	return []rtAttr{
		{"id", rt.ID},
		{"orgId", rt.OrgID},
		{"federatedId", rt.FederatedID},
		{"isShared", shared},
		{"createdBy", rt.CreatedBy},
		{"updatedBy", rt.UpdatedBy},
		{"createTime", rt.CreateTime},
		{"updateTime", rt.UpdateTime},
	}
}

func newRuntimeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "runtime",
		Aliases: []string{"rt"},
		Short:   "Manage runtime environments",
	}
	cmd.AddCommand(newRuntimeListCmd())
	cmd.AddCommand(newRuntimeGetCmd())
	cmd.AddCommand(newRuntimeCreateCmd())
	cmd.AddCommand(newRuntimeUpdateCmd())
	return cmd
}

func newRuntimeListCmd() *cobra.Command {
	var opts client.RuntimeListOptions
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List runtime environments",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			runtimes, err := c.ListRuntimeEnvironments(context.Background(), opts)
			if err != nil {
				return err
			}
			f, err := getFormatter()
			if err != nil {
				return err
			}
			columns := []output.Column{
				{Header: "ID", Field: "id", Width: 22},
				{Header: "NAME", Field: "name", Width: 30},
				{Header: "FEDERATED ID", Field: "federatedId", Width: 24},
				{Header: "SHARED", Field: "isShared", Width: 8},
				{Header: "AGENTS", Field: "agents", Width: 7, Func: agentCountFunc},
				{Header: "UPDATED", Field: "updateTime", Width: 22},
			}
			return f.Format(runtimes, columns)
		},
	}
	cmd.Flags().IntVar(&opts.Limit, "limit", 200, "max results")
	cmd.Flags().IntVar(&opts.Skip, "skip", 0, "number of results to skip")
	return cmd
}

func newRuntimeGetCmd() *cobra.Command {
	var (
		id   string
		name string
	)
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get runtime environment details",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" && name == "" {
				return fmt.Errorf("either --id or --name is required")
			}
			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			var rt *client.RuntimeEnvironment
			if id != "" {
				rt, err = c.GetRuntimeEnvironment(context.Background(), id)
			} else {
				rt, err = c.GetRuntimeEnvironmentByName(context.Background(), name)
			}
			if err != nil {
				return err
			}

			f, err := getFormatter()
			if err != nil {
				return err
			}

			// Non-table formats: let the formatter render the full nested struct.
			if outputFmt != "" && outputFmt != "table" {
				return f.Format(rt, nil)
			}

			// Table mode: tree-style view.
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Runtime Environment: %s\n\n", rt.Name)
			if err := f.Format(runtimeEnvAttrs(rt), rtAttrCols); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nAgents (%d):\n", len(rt.Agents))
			if len(rt.Agents) > 0 {
				return f.Format(rt.Agents, runtimeAgentCols)
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "  (none)")
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "runtime environment ID")
	cmd.Flags().StringVar(&name, "name", "", "runtime environment name")
	cmd.MarkFlagsMutuallyExclusive("id", "name")
	return cmd
}

func newRuntimeCreateCmd() *cobra.Command {
	var fromFile string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a runtime environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			if fromFile == "" {
				return fmt.Errorf("--from-file is required")
			}
			data, err := os.ReadFile(fromFile)
			if err != nil {
				return fmt.Errorf("reading file: %w", err)
			}
			var rt client.RuntimeEnvironment
			err = json.Unmarshal(data, &rt)
			if err != nil {
				return fmt.Errorf("parsing JSON: %w", err)
			}
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			created, err := c.CreateRuntimeEnvironment(context.Background(), &rt)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Runtime environment created: %s (ID: %s)\n", created.Name, created.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&fromFile, "from-file", "", "JSON file (required)")
	return cmd
}

func newRuntimeUpdateCmd() *cobra.Command {
	var (
		id       string
		fromFile string
	)
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a runtime environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" {
				return fmt.Errorf("--id is required")
			}
			if fromFile == "" {
				return fmt.Errorf("--from-file is required")
			}
			data, err := os.ReadFile(fromFile)
			if err != nil {
				return fmt.Errorf("reading file: %w", err)
			}
			var rt client.RuntimeEnvironment
			err = json.Unmarshal(data, &rt)
			if err != nil {
				return fmt.Errorf("parsing JSON: %w", err)
			}
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			updated, err := c.UpdateRuntimeEnvironment(context.Background(), id, &rt)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Runtime environment updated: %s (ID: %s)\n", updated.Name, updated.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "runtime environment ID (required)")
	cmd.Flags().StringVar(&fromFile, "from-file", "", "JSON file (required)")
	return cmd
}
