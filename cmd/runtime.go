package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/charmbracelet/lipgloss"
	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/config"
	"github.com/jbrazda/iics-cli/internal/filter"
	"github.com/jbrazda/iics-cli/internal/output"
	"github.com/spf13/cobra"
)

var runtimeAgentCols = []output.Column{
	{Header: "NAME", Field: "name", Width: 20},
	{Header: "HOST", Field: "agentHost", Width: 18},
	{Header: "PLATFORM", Field: "platform", Width: 9},
	{Header: "VERSION", Field: "agentVersion", Width: 9},
	{Header: "ACTIVE", Field: "active", Width: 7, Func: agentActiveFunc},
	{Header: "READY", Field: "readyToRun", Width: 7, Func: agentReadyFunc},
	{Header: "UPGRADE", Field: "upgradeStatus", Width: 13},
	{Header: "FEDERATED ID", Field: "federatedId", Width: 24},
	{Header: "GROUP ID", Field: "agentGroupId", Width: 24},
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

func runtimeEnvAttrs(rt *client.RuntimeEnvironment) []output.KVRow {
	shared := "false"
	if rt.IsShared {
		shared = "true"
	}
	return []output.KVRow{
		output.KV("id", rt.ID),
		output.KV("orgId", rt.OrgID),
		output.KV("orgUUID", rt.OrgUUID),
		output.KV("federatedId", rt.FederatedID),
		output.KV("isShared", shared),
		output.KV("createdBy", rt.CreatedBy),
		output.KV("updatedBy", rt.UpdatedBy),
		output.KV("createTime", rt.CreateTime),
		output.KV("updateTime", rt.UpdateTime),
		output.KV("createTimeUTC", rt.CreateTimeUTC),
		output.KV("updateTimeUTC", rt.UpdateTimeUTC),
	}
}

func serverlessConfigAttrs(s *client.ServerlessConfig) []output.KVRow {
	return []output.KVRow{
		output.KV("platform", s.Platform),
		output.KV("applicationType", s.ApplicationType),
		output.KV("status", s.Status),
		output.KV("statusMessage", s.StatusMessage),
		output.KV("maxComputeUnits", strconv.Itoa(s.MaxComputeUnits)),
		output.KV("executionTimeout", strconv.Itoa(s.ExecutionTimeout)),
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
	cmd.AddCommand(newRuntimeConfigsCmd())
	return cmd
}

func newRuntimeListCmd() *cobra.Command {
	var (
		opts    client.RuntimeListOptions
		filters []string
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List runtime environments",
		Example: `  iics runtime list
  iics runtime list --filter isShared==true`,
		RunE: func(cmd *cobra.Command, args []string) error {
			preds, err := filter.ParseAll(filters)
			if err != nil {
				return err
			}
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
			if len(preds) > 0 {
				rows, err := filter.Apply(runtimes, preds)
				if err != nil {
					return err
				}
				return f.Format(rows, columns)
			}
			return f.Format(runtimes, columns)
		},
	}
	cmd.Flags().IntVar(&opts.Limit, "limit", 200, "max results")
	cmd.Flags().IntVar(&opts.Skip, "skip", 0, "number of results to skip")
	cmd.Flags().StringArrayVar(&filters, "filter", nil, "client-side filter, e.g. isShared==true (repeatable, AND-ed)")
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
			if err := f.Format(runtimeEnvAttrs(rt), output.KVCols); err != nil {
				return err
			}
			if s := rt.ServerlessConfig; s != nil && (s.Platform != "" || s.Status != "" || s.ApplicationType != "") {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\nServerless Config:")
				if err := f.Format(serverlessConfigAttrs(rt.ServerlessConfig), output.KVCols); err != nil {
					return err
				}
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
	var (
		fromFile    string
		interactive bool
	)
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a runtime environment",
		Long: `Create a runtime environment.

Provide --from-file with a JSON definition, or run interactively (--interactive/-i,
or omit --from-file on a terminal) to be prompted for the name, shared flag, and
member agents.`,
		Example: `  iics runtime create --from-file my-runtime.json
  iics runtime create
  iics runtime create -i --from-file seed.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			var rt client.RuntimeEnvironment
			if fromFile != "" {
				data, err := os.ReadFile(fromFile)
				if err != nil {
					return fmt.Errorf("reading file: %w", err)
				}
				if err := json.Unmarshal(data, &rt); err != nil {
					return fmt.Errorf("parsing JSON: %w", err)
				}
			}

			useWizard := interactive || (fromFile == "" && config.IsTerminal())
			if !useWizard && fromFile == "" {
				return fmt.Errorf("--from-file or --interactive is required")
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			if useWizard {
				if werr := runRuntimeCreateWizard(ctx, c, &rt); werr != nil {
					return werr
				}
			}

			created, err := c.CreateRuntimeEnvironment(ctx, &rt)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Runtime environment created: %s (ID: %s)\n", created.Name, created.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&fromFile, "from-file", "", "JSON file with the runtime environment definition")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "create interactively (prompt for name, shared flag, and agents)")
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
