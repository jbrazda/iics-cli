package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/output"
	"github.com/spf13/cobra"
)

func newRuntimeConfigsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "configs",
		Short: "Manage Secure Agent group service properties",
	}
	cmd.AddCommand(newRuntimeConfigsGetCmd())
	cmd.AddCommand(newRuntimeConfigsSetCmd())
	return cmd
}

// resolveRuntimeID returns a runtime environment ID from --id or --name.
func resolveRuntimeID(ctx context.Context, c *client.Client, id, name string) (string, error) {
	if id != "" {
		return id, nil
	}
	rt, err := c.GetRuntimeEnvironmentByName(ctx, name)
	if err != nil {
		return "", err
	}
	return rt.ID, nil
}

func newRuntimeConfigsGetCmd() *cobra.Command {
	var id, name, service string
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Show Secure Agent group service property overrides",
		Example: `  iics environment configs get --id <groupId>
  iics environment configs get --name "My Group" --service Data_Integration_Server`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" && name == "" {
				return fmt.Errorf("either --id or --name is required")
			}
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()
			groupID, err := resolveRuntimeID(ctx, c, id, name)
			if err != nil {
				return err
			}
			cfg, err := c.GetAgentGroupConfigs(ctx, groupID)
			if err != nil {
				return err
			}
			if service != "" {
				cfg = client.AgentGroupServiceConfig{service: cfg[service]}
			}

			if outputFmt != "" && outputFmt != "table" {
				f, err := getFormatter()
				if err != nil {
					return err
				}
				return f.Format(cfg, nil)
			}

			w := cmd.OutOrStdout()
			if len(cfg) == 0 {
				_, _ = fmt.Fprintln(w, "No service property overrides.")
				return nil
			}
			cfgOut, _ := loadConfig()
			tf := output.New(output.FormatTable, w, resolveTableStyle(cfgOut))
			for _, svc := range sortedKeys(cfg) {
				_, _ = fmt.Fprintf(w, "Service: %s\n\n", svc)
				for i, setting := range cfg[svc] {
					if i > 0 {
						_, _ = fmt.Fprintln(w)
					}
					if err := tf.Format(settingKVRows(setting), output.KVCols); err != nil {
						return err
					}
				}
				_, _ = fmt.Fprintln(w)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "Secure Agent group (runtime environment) ID")
	cmd.Flags().StringVar(&name, "name", "", "Secure Agent group (runtime environment) name")
	cmd.Flags().StringVar(&service, "service", "", "show only this service's properties")
	cmd.MarkFlagsMutuallyExclusive("id", "name")
	return cmd
}

func newRuntimeConfigsSetCmd() *cobra.Command {
	var id, name, fromFile string
	var yes bool
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Replace Secure Agent group service property overrides from a JSON file",
		Example: `  iics environment configs set --id <groupId> --from-file props.json
  iics environment configs set --name "My Group" --from-file props.json --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" && name == "" {
				return fmt.Errorf("either --id or --name is required")
			}
			if fromFile == "" {
				return fmt.Errorf("--from-file is required")
			}
			data, err := os.ReadFile(fromFile)
			if err != nil {
				return fmt.Errorf("reading file: %w", err)
			}
			var cfg client.AgentGroupServiceConfig
			if parseErr := json.Unmarshal(data, &cfg); parseErr != nil {
				return fmt.Errorf("parsing JSON: %w", parseErr)
			}
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()
			groupID, err := resolveRuntimeID(ctx, c, id, name)
			if err != nil {
				return err
			}
			if !yes {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Replace service property overrides for group %s? [y/N]: ", groupID)
				var confirm string
				_, _ = fmt.Scanln(&confirm)
				if confirm != "y" && confirm != "Y" {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Canceled.")
					return nil
				}
			}
			if _, err := c.UpdateAgentGroupConfigs(ctx, groupID, cfg); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Updated service properties for group %s: %v\n", groupID, sortedKeys(cfg))
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "Secure Agent group (runtime environment) ID")
	cmd.Flags().StringVar(&name, "name", "", "Secure Agent group (runtime environment) name")
	cmd.Flags().StringVar(&fromFile, "from-file", "", "JSON file with the {\"Service\":[{...}]} overrides (required)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompt")
	cmd.MarkFlagsMutuallyExclusive("id", "name")
	return cmd
}

func sortedKeys(cfg client.AgentGroupServiceConfig) []string {
	keys := make([]string, 0, len(cfg))
	for k := range cfg {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func settingKVRows(setting map[string]interface{}) []output.KVRow {
	keys := make([]string, 0, len(setting))
	for k := range setting {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	rows := make([]output.KVRow, 0, len(keys))
	for _, k := range keys {
		rows = append(rows, output.KV(k, fmt.Sprintf("%v", setting[k])))
	}
	return rows
}
