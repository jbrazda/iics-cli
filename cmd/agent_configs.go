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

const agentConfigsExperimental = "Experimental: uses the undocumented Administrator form endpoint /runtime/app/agent.do."

func newAgentConfigsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "configs",
		Short: "Show and change agent-level Secure Agent service properties (experimental)",
		Long: `Show and change Secure Agent service properties of a single agent, such as
the Process Server cluster or database settings, and the agent's own
Agent Core, Agent Manager and Tomcat settings.

` + agentConfigsExperimental + ` The public REST API only
configures properties for a whole Secure Agent group (see "iics environment
configs").`,
	}
	cmd.AddCommand(newAgentConfigsGetCmd())
	cmd.AddCommand(newAgentConfigsSetCmd())
	return cmd
}

// agentPropertyCols is the table layout for `agent configs get`.
var agentPropertyCols = []output.Column{
	{Header: "SERVICE", Field: "service", Width: 14, Priority: 1, Shrink: output.ShrinkTruncate},
	{Header: "TYPE", Field: "type", Width: 12, Priority: 1, Shrink: output.ShrinkTruncate, Func: func(v interface{}) string {
		row, _ := v.(map[string]interface{})
		t, _ := row["type"].(string)
		if sub, _ := row["subType"].(string); sub != "" {
			t += "/" + sub
		}
		return t
	}},
	{Header: "NAME", Field: "name", Width: 24, Priority: 1, Shrink: output.ShrinkWrap},
	{Header: "VALUE", Field: "value", Width: 30, Priority: 1, Func: wrapConfigField("value")},
	{Header: "DEFAULT", Field: "defaultValue", Width: 30, Priority: 3, Func: wrapConfigField("defaultValue")},
	{Header: "OVERRIDDEN", Field: "overridden", Width: 10, Priority: 2, Shrink: output.ShrinkNever, Func: func(v interface{}) string {
		row, _ := v.(map[string]interface{})
		if b, _ := row["overridden"].(bool); b {
			return "yes"
		}
		return ""
	}},
}

// agentConfigDiffCols is the table layout for the changes `agent configs set`
// makes.
var agentConfigDiffCols = []output.Column{
	{Header: "PROPERTY", Field: "property", Width: 30, Priority: 1, Shrink: output.ShrinkWrap},
	{Header: "CURRENT", Field: "oldValue", Width: 30, Priority: 1, Func: diffValueFunc("oldValue", "oldOverridden")},
	{Header: "NEW", Field: "newValue", Width: 30, Priority: 1, Func: diffValueFunc("newValue", "newOverridden")},
}

// diffValueFunc shows a value, "(default)" when it is not overridden, and
// hides sensitive values.
func diffValueFunc(valueField, overField string) func(v interface{}) string {
	return func(v interface{}) string {
		row, _ := v.(map[string]interface{})
		s, _ := row[valueField].(string)
		if sens, _ := row["sensitive"].(bool); sens {
			s = "***"
		}
		s = output.WrapCell(s, 44)
		if over, _ := row[overField].(bool); !over {
			s += " (default)"
		}
		return s
	}
}

func addAgentSelectorFlags(cmd *cobra.Command, id, fid, name, hostname *string) {
	cmd.Flags().StringVar(id, "id", "", "agent ID")
	cmd.Flags().StringVar(fid, "fid", "", "agent federated ID")
	cmd.Flags().StringVar(name, "name", "", "agent name")
	cmd.Flags().StringVar(hostname, "hostname", "", "agent host name")
	cmd.MarkFlagsMutuallyExclusive("id", "fid", "name", "hostname")
}

func newAgentConfigsGetCmd() *cobra.Command {
	var (
		id, fid, name, hostname string
		flt                     client.AgentConfigFilter
	)
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Show an agent's service properties",
		Long:  "Show an agent's service properties with their defaults and whether they are overridden.\n\n" + agentConfigsExperimental,
		Example: `  iics agent configs get --name CAI-A03
  iics agent configs get --name CAI-A03 --service process-engine --overridden
  iics agent configs get --hostname myhost --service "Process Server" --type db
  iics agent configs get --id <agent-id> --service agent --type TOMCAT_JRE -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" && fid == "" && name == "" && hostname == "" {
				return fmt.Errorf("one of --id, --fid, --name, or --hostname is required")
			}
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()
			agentID, err := resolveAgentID(ctx, c, id, name, hostname, fid)
			if err != nil {
				return err
			}
			form, err := c.GetAgentConfigForm(ctx, agentID)
			if err != nil {
				return err
			}
			props, err := form.Filter(flt)
			if err != nil {
				return err
			}
			f, err := getFormatter()
			if err != nil {
				return err
			}
			return f.Format(props, agentPropertyCols)
		},
	}
	addAgentSelectorFlags(cmd, &id, &fid, &name, &hostname)
	cmd.Flags().StringVar(&flt.Service, "service", "", `service name, for example process-engine, "Process Server", or agent for the agent's own properties`)
	cmd.Flags().StringVar(&flt.Type, "type", "", "property type, for example cluster, db or AGENTCORE_LOG4J/DEBUG")
	cmd.Flags().StringVar(&flt.Name, "search", "", "regular expression on the property name (case-insensitive)")
	cmd.Flags().BoolVar(&flt.Overridden, "overridden", false, "show only properties that differ from the defaults")
	return cmd
}

// parseAgentConfigChanges builds the change list from --set, --reset and
// --from-file. A --set value starting with @ is read from that file.
func parseAgentConfigChanges(sets, resets []string, fromFile string) ([]client.AgentConfigChange, error) {
	var changes []client.AgentConfigChange
	if fromFile != "" {
		data, err := os.ReadFile(fromFile)
		if err != nil {
			return nil, fmt.Errorf("reading file: %w", err)
		}
		if err := json.Unmarshal(data, &changes); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", fromFile, err)
		}
	}
	for _, s := range sets {
		prop, value, ok := strings.Cut(s, "=")
		if !ok {
			return nil, fmt.Errorf("invalid --set %q: expected service:type:name=value", s)
		}
		if file, isFile := strings.CutPrefix(value, "@"); isFile {
			data, err := os.ReadFile(file)
			if err != nil {
				return nil, fmt.Errorf("reading value for %s: %w", prop, err)
			}
			value = string(data)
		}
		changes = append(changes, client.AgentConfigChange{Property: prop, Value: value})
	}
	for _, r := range resets {
		changes = append(changes, client.AgentConfigChange{Property: r, Reset: true})
	}
	return changes, nil
}

func newAgentConfigsSetCmd() *cobra.Command {
	var (
		id, fid, name, hostname string
		sets, resets            []string
		fromFile                string
		dryRun, yes, force      bool
	)
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Change an agent's service properties",
		Long: `Change or reset service properties of one agent.

Properties are named service:type:name, as shown by "iics agent configs get".
Use type/subType (for example AGENTCORE_LOG4J/DEBUG) when a name exists for
both subtypes. Values are stored as typed, including quotes: Process Server
values such as '1024M' keep their single quotes.

The changes are shown first and applied after confirmation (or with --yes).
The agent's whole property form is submitted, as the Administrator UI does,
and read back to verify. The agent applies changed properties itself, which
may restart the affected service.

Agents in a Secure Agent group with group-level properties get those settings
from the group; the command refuses to change them unless --force is given.

` + agentConfigsExperimental,
		Example: `  iics agent configs set --name CAI-A03 --set "process-engine:cluster:primary-node=false" --dry-run
  iics agent configs set --name CAI-A03 --set "process-engine:jvm:max-heap='4G'"
  iics agent configs set --name CAI-A03 --set "process-engine:db:connection-properties=@db-props.yaml"
  iics agent configs set --name CAI-A03 --reset process-engine:jvm:max-heap --yes
  iics agent configs set --name CAI-A03 --from-file changes.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" && fid == "" && name == "" && hostname == "" {
				return fmt.Errorf("one of --id, --fid, --name, or --hostname is required")
			}
			changes, err := parseAgentConfigChanges(sets, resets, fromFile)
			if err != nil {
				return err
			}
			if len(changes) == 0 {
				return fmt.Errorf("nothing to change: use --set, --reset or --from-file")
			}
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()
			agent, err := resolveAgent(ctx, c, id, name, hostname, fid)
			if err != nil {
				return err
			}
			if agent.GroupID != "" && !force {
				groupCfg, gErr := c.GetAgentGroupConfigs(ctx, agent.GroupID)
				if gErr != nil {
					return fmt.Errorf("checking group-level properties: %w", gErr)
				}
				if len(groupCfg) > 0 {
					return fmt.Errorf("agent %s is in group %s, which has group-level properties for %v; "+
						"remove them first (iics environment configs) or use --force", agent.Name, agent.GroupID, sortedKeys(groupCfg))
				}
			}
			form, err := c.GetAgentConfigForm(ctx, agent.ID)
			if err != nil {
				return err
			}
			diffs, err := form.Apply(changes)
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			if len(diffs) == 0 {
				_, _ = fmt.Fprintln(w, "Nothing to change: the properties already have these values.")
				return nil
			}
			if outputFmt == "" || outputFmt == "table" {
				_, _ = fmt.Fprintf(w, "Changes for agent %s (%s):\n\n", agent.Name, agent.ID)
			}
			f, err := getFormatter()
			if err != nil {
				return err
			}
			if err := f.Format(diffs, agentConfigDiffCols); err != nil {
				return err
			}
			if dryRun {
				return nil
			}
			if !yes && !confirmAction(cmd, fmt.Sprintf("Apply %d change(s) to agent %s?", len(diffs), agent.Name)) {
				return nil
			}
			if err := c.SaveAgentConfigForm(ctx, form); err != nil {
				return err
			}
			if err := c.VerifyAgentConfigDiffs(ctx, agent.ID, diffs); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(w, "Updated %d propert%s on agent %s.\n", len(diffs), map[bool]string{true: "y", false: "ies"}[len(diffs) == 1], agent.Name)
			return nil
		},
	}
	addAgentSelectorFlags(cmd, &id, &fid, &name, &hostname)
	cmd.Flags().StringArrayVar(&sets, "set", nil, "set a property: service:type:name=value (value @file reads a file); repeatable")
	cmd.Flags().StringArrayVar(&resets, "reset", nil, "reset a property to its default: service:type:name; repeatable")
	cmd.Flags().StringVar(&fromFile, "from-file", "", `JSON file with [{"property":"service:type:name","value":"...","reset":false}]`)
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show the changes without applying them")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "apply without asking for confirmation")
	cmd.Flags().BoolVar(&force, "force", false, "change the agent even if its group has group-level properties")
	return cmd
}
