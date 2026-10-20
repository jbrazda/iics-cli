package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/output"
	"github.com/jbrazda/iics-cli/internal/tui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const agentConfigsExperimental = "Experimental: uses the undocumented Administrator form endpoint /runtime/app/agent.do."

func newAgentConfigsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "configs",
		Short: "Show and change agent-level Secure Agent service properties (experimental)",
		Long: `Show and change Secure Agent service properties of a single agent, such as
the Process Server cluster or database settings, for any services the agent
runs.

` + agentConfigsExperimental + ` The public REST API only
configures properties for a whole Secure Agent group (see "iics environment
configs").`,
	}
	cmd.AddCommand(newAgentConfigsGetCmd())
	cmd.AddCommand(newAgentConfigsSetCmd())
	cmd.AddCommand(newAgentConfigsEditCmd())
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
	// Long names (up to 76 characters on Process Server) wrap inside the
	// cell; ShrinkWrap would move the column to the end.
	{Header: "NAME", Field: "name", Width: 24, Priority: 1, Shrink: output.ShrinkNever, Func: func(v interface{}) string {
		row, _ := v.(map[string]interface{})
		s, _ := row["name"].(string)
		return output.WrapCell(s, 36)
	}},
	{Header: "VALUE", Field: "value", Width: 30, Priority: 1, Func: wrapConfigField("value")},
	{Header: "DEFAULT", Field: "defaultValue", Width: 30, Priority: 3, Func: wrapConfigField("defaultValue")},
	{Header: "OVERRIDDEN", Field: "overridden", Width: 10, Priority: 2, Shrink: output.ShrinkNever, Func: func(v interface{}) string {
		row, _ := v.(map[string]interface{})
		if b, _ := row["custom"].(bool); b {
			return "custom"
		}
		if b, _ := row["overridden"].(bool); b {
			return "yes"
		}
		return ""
	}},
}

// agentConfigDiffCols is the table layout for the changes `agent configs set`
// makes.
var agentConfigDiffCols = []output.Column{
	{Header: "PROPERTY", Field: "property", Width: 30, Priority: 1, Shrink: output.ShrinkNever},
	{Header: "CURRENT", Field: "oldValue", Width: 30, Priority: 1, Func: diffValueFunc("oldValue", "oldOverridden")},
	{Header: "NEW", Field: "newValue", Width: 30, Priority: 1, Func: diffValueFunc("newValue", "newOverridden")},
}

// diffValueFunc shows a value, "(default)" when it is not overridden, and
// hides sensitive values.
func diffValueFunc(valueField, overField string) func(v interface{}) string {
	return func(v interface{}) string {
		row, _ := v.(map[string]interface{})
		s, _ := row[valueField].(string)
		change, _ := row["change"].(string)
		switch {
		case change == "add" && valueField == "oldValue":
			return "(new custom property)"
		case change == "remove" && valueField == "newValue":
			return "(removed)"
		}
		if sens, _ := row["sensitive"].(bool); sens {
			s = "***"
		}
		s = output.WrapCell(s, 44)
		if custom, _ := row["custom"].(bool); custom {
			return s
		}
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
		export                  bool
	)
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Show an agent's service properties",
		Long:  "Show an agent's service properties with their defaults and whether they are overridden.\n\n" + agentConfigsExperimental,
		Example: `  iics agent configs get --name CAI-A03
  iics agent configs get --name CAI-A03 --service process-engine --overridden
  iics agent configs get --hostname myhost --service "Process Server" --type db
  iics agent configs get --name CDI-G01-A01 --service "Data Integration Server" --type TOMCAT_JRE -o json
  iics agent configs get --name CAI-A03 --service process-engine --overridden --export > pe.yaml`,
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
			if export {
				return writeAgentConfigExport(cmd, props)
			}
			if outputFmt != "" && outputFmt != "table" {
				f, err := getFormatter()
				if err != nil {
					return err
				}
				return f.Format(props, agentPropertyCols)
			}
			return printAgentConfigTables(cmd, form, props)
		},
	}
	addAgentSelectorFlags(cmd, &id, &fid, &name, &hostname)
	cmd.Flags().StringVar(&flt.Service, "service", "", `service name, for example process-engine, "Process Server", or agent for the agent's own properties`)
	cmd.Flags().StringVar(&flt.Type, "type", "", "property type, for example cluster, db or AGENTCORE_LOG4J/DEBUG")
	cmd.Flags().StringVar(&flt.Name, "search", "", "regular expression on the property name (case-insensitive)")
	cmd.Flags().BoolVar(&flt.Overridden, "overridden", false, "show only properties that differ from the defaults")
	cmd.Flags().BoolVar(&flt.IncludeHidden, "include-hidden", false, "also show the agent's own Agent Core, Agent Manager and Tomcat properties, which the UI hides and which cannot be changed")
	cmd.Flags().BoolVar(&export, "export", false, "write the properties as a change file for 'agent configs set --from-file' (YAML, or JSON with -o json)")
	return cmd
}

// printAgentConfigTables prints props as one table per service, like
// `agent details --full`: a "Service: <display name> (<service>)" header,
// then TYPE, NAME, VALUE, DEFAULT and OVERRIDDEN columns.
func printAgentConfigTables(cmd *cobra.Command, form *client.AgentConfigForm, props []client.AgentConfigProperty) error {
	w := cmd.OutOrStdout()
	_, _ = fmt.Fprintf(w, "Agent: %s (%s)\n", form.AgentName, form.AgentID)
	if len(props) == 0 {
		_, _ = fmt.Fprintln(w, "\nNo properties match.")
		return nil
	}
	cfg, _ := loadConfig()
	tf := output.New(output.FormatTable, w, resolveTableStyle(cfg))
	for start := 0; start < len(props); {
		end := start + 1
		for end < len(props) && props[end].Service == props[start].Service {
			end++
		}
		p := props[start]
		_, _ = fmt.Fprintf(w, "\nService: %s (%s)\n\n", p.ServiceName, p.Service)
		if err := tf.Format(props[start:end], agentPropertyCols[1:]); err != nil {
			return err
		}
		start = end
	}
	return nil
}

// writeAgentConfigExport writes props in the --from-file change format.
func writeAgentConfigExport(cmd *cobra.Command, props []client.AgentConfigProperty) error {
	w := cmd.OutOrStdout()
	if outputFmt == "json" {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(client.AgentConfigMap(props))
	}
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	if err := enc.Encode(client.AgentConfigDocument(props)); err != nil {
		return err
	}
	return enc.Close()
}

// parseAgentConfigChanges builds the change list from --set, --reset and
// --from-file. A --set value starting with @ is read from that file.
func parseAgentConfigChanges(sets, adds, resets, removes []string, fromFile string) ([]client.AgentConfigChange, error) {
	var changes []client.AgentConfigChange
	if fromFile != "" {
		data, err := os.ReadFile(fromFile)
		if err != nil {
			return nil, fmt.Errorf("reading file: %w", err)
		}
		if changes, err = client.ParseAgentConfigChanges(data); err != nil {
			return nil, fmt.Errorf("%s: %w", fromFile, err)
		}
	}
	parseValue := func(flag, s string, add bool) error {
		prop, value, ok := strings.Cut(s, "=")
		if !ok {
			return fmt.Errorf("invalid --%s %q: expected service:type:name=value", flag, s)
		}
		if file, isFile := strings.CutPrefix(value, "@"); isFile {
			data, err := os.ReadFile(file)
			if err != nil {
				return fmt.Errorf("reading value for %s: %w", prop, err)
			}
			value = string(data)
		}
		changes = append(changes, client.AgentConfigChange{Property: prop, Value: value, Add: add})
		return nil
	}
	for _, s := range sets {
		if err := parseValue("set", s, false); err != nil {
			return nil, err
		}
	}
	for _, s := range adds {
		if err := parseValue("add", s, true); err != nil {
			return nil, err
		}
	}
	for _, r := range resets {
		changes = append(changes, client.AgentConfigChange{Property: r, Reset: true})
	}
	for _, r := range removes {
		changes = append(changes, client.AgentConfigChange{Property: r, Remove: true})
	}
	return changes, nil
}

// checkGroupConfigs returns an error when the agent's group has group-level
// service properties, which replace the agent's own settings.
func checkGroupConfigs(ctx context.Context, c *client.Client, agent *client.Agent) error {
	if agent.GroupID == "" {
		return nil
	}
	groupCfg, err := c.GetAgentGroupConfigs(ctx, agent.GroupID)
	if err != nil {
		return fmt.Errorf("checking group-level properties: %w", err)
	}
	if len(groupCfg) > 0 {
		return fmt.Errorf("agent %s is in group %s, which has group-level properties for %v; "+
			"remove them first (iics environment configs) or use --force", agent.Name, agent.GroupID, sortedKeys(groupCfg))
	}
	return nil
}

// saveAgentConfigDiffs saves the form, verifies the changes and prints the
// restart hint.
func saveAgentConfigDiffs(ctx context.Context, cmd *cobra.Command, c *client.Client, agent *client.Agent, form *client.AgentConfigForm, diffs []client.AgentConfigDiff) error {
	if err := c.SaveAgentConfigForm(ctx, form); err != nil {
		return err
	}
	if err := c.VerifyAgentConfigDiffs(ctx, agent.ID, diffs); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Updated %d propert%s on agent %s.\n", len(diffs), map[bool]string{true: "y", false: "ies"}[len(diffs) == 1], agent.Name)
	printRestartHint(cmd, agent, diffs)
	return nil
}

// printRestartHint lists the services whose properties changed. Some, such as
// Process Server, only mark themselves as needing a restart and keep running
// with the old values until restarted.
func printRestartHint(cmd *cobra.Command, agent *client.Agent, diffs []client.AgentConfigDiff) {
	var services []string
	seen := map[string]bool{}
	for _, d := range diffs {
		if d.ServiceName != "" && !seen[d.ServiceName] {
			seen[d.ServiceName] = true
			services = append(services, d.ServiceName)
		}
	}
	if len(services) == 0 {
		return
	}
	w := cmd.ErrOrStderr()
	_, _ = fmt.Fprintf(w, "\nChanged services: %s. Services such as Process Server use the new values only after a restart:\n", strings.Join(services, ", "))
	for _, s := range services {
		_, _ = fmt.Fprintf(w, "  iics agent restart --id %s --service %q --blocking\n", agent.ID, s)
	}
}

func newAgentConfigsSetCmd() *cobra.Command {
	var (
		id, fid, name, hostname string
		sets, adds              []string
		resets, removes         []string
		addMissing, resetAll    bool
		resetService            string
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

Changes come from --set, --reset and --from-file. The file is JSON or YAML,
either a nested map (a null value resets the property), as written by
"agent configs get --export":

  process-engine:
    cluster:
      primary-node: "true"
    jvm:
      max-heap: "'4G'"
      min-heap: null

or a list: [{"property": "service:type:name", "value": "...", "reset": false}].
Only the listed properties change.

The changes are shown first and applied after confirmation (or with --yes).
The agent's whole property form is submitted, as the Administrator UI does,
and read back to verify. Some services, such as Process Server, keep running
with the old values and are only marked as needing a restart; restart them
with "iics agent restart".

Agents in a Secure Agent group with group-level properties get those settings
from the group; the command refuses to change them unless --force is given.

` + agentConfigsExperimental,
		Example: `  iics agent configs set --name CAI-A03 --set "process-engine:cluster:primary-node=false" --dry-run
  iics agent configs set --name CAI-A03 --set "process-engine:jvm:max-heap='4G'"
  iics agent configs set --name CAI-A03 --set "process-engine:db:connection-properties=@db-props.yaml"
  iics agent configs set --name CAI-A03 --reset process-engine:jvm:max-heap --yes
  iics agent configs set --name CAI-A03 --reset-all --service "Process Server" --dry-run
  iics agent configs set --name CAI-A03 --from-file changes.yaml
  iics agent configs get --name CAI-A01 --service process-engine --overridden --export > pe.yaml
  iics agent configs set --name CAI-A02 --from-file pe.yaml --dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" && fid == "" && name == "" && hostname == "" {
				return fmt.Errorf("one of --id, --fid, --name, or --hostname is required")
			}
			changes, err := parseAgentConfigChanges(sets, adds, resets, removes, fromFile)
			if err != nil {
				return err
			}
			if len(changes) == 0 && !resetAll {
				return fmt.Errorf("nothing to change: use --set, --add, --reset, --reset-all, --remove or --from-file")
			}
			if resetService != "" && !resetAll {
				return fmt.Errorf("--service requires --reset-all")
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
			if !force {
				if err = checkGroupConfigs(ctx, c, agent); err != nil {
					return err
				}
			}
			form, err := c.GetAgentConfigForm(ctx, agent.ID)
			if err != nil {
				return err
			}
			if resetAll {
				resets, skipped, rErr := form.ResetAllChanges(resetService)
				if rErr != nil {
					return rErr
				}
				if skipped > 0 {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Reset all: %d sensitive propert%s left unchanged; reset them one by one with --reset.\n",
						skipped, map[bool]string{true: "y", false: "ies"}[skipped == 1])
				}
				changes = append(changes, resets...)
			}
			diffs, err := form.ApplyWith(changes, client.ApplyOptions{AddMissing: addMissing})
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
			return saveAgentConfigDiffs(ctx, cmd, c, agent, form, diffs)
		},
	}
	addAgentSelectorFlags(cmd, &id, &fid, &name, &hostname)
	cmd.Flags().StringArrayVar(&sets, "set", nil, "set a property: service:type:name=value (value @file reads a file); repeatable")
	cmd.Flags().StringArrayVar(&resets, "reset", nil, "reset a property to its default (removes a custom property): service:type:name; repeatable")
	cmd.Flags().StringArrayVar(&adds, "add", nil, "add a custom property: service:type:name=value; repeatable")
	cmd.Flags().StringArrayVar(&removes, "remove", nil, "remove a custom property: service:type:name; repeatable")
	cmd.Flags().BoolVar(&resetAll, "reset-all", false, "reset every overridden standard property to its default, like the UI's Reset All (custom and sensitive properties are kept)")
	cmd.Flags().StringVar(&resetService, "service", "", "with --reset-all: only this service (name, display name or versioned key)")
	cmd.Flags().BoolVar(&addMissing, "add-missing", false, "add properties from --set or --from-file that do not exist as custom properties")
	cmd.Flags().StringVar(&fromFile, "from-file", "", "JSON or YAML change file: service/type/name map (null resets) or list of {property, value, reset}")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show the changes without applying them")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "apply without asking for confirmation")
	cmd.Flags().BoolVar(&force, "force", false, "change the agent even if its group has group-level properties")
	return cmd
}

func newAgentConfigsEditCmd() *cobra.Command {
	var (
		id, fid, name, hostname string
		force                   bool
	)
	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Edit an agent's service properties interactively",
		Long: `Edit an agent's service properties in an interactive editor.

Pick a service, then a property. One-line values open in an input field;
Process Server values in single quotes ('10G') are edited without the quotes,
which are added back. Multi-value properties such as LDAP properties, database
connection properties or additional logging open in an entry editor: edit,
add or remove entries, edit the whole value as text, or open it in $EDITOR
(ctrl+e). Any property can be reset to its default.

Review shows all pending changes before they are applied. Esc goes back one
screen; on the service list it cancels. Without an agent flag the agent is
picked from a list.

` + agentConfigsExperimental,
		Example: `  iics agent configs edit
  iics agent configs edit --name CAI-A03
  iics agent configs edit --hostname cletinfacld07 -p tst`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !isInteractiveTTY() {
				return fmt.Errorf("agent configs edit requires a terminal; use agent configs set")
			}
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()
			w := cmd.OutOrStdout()
			var agent *client.Agent
			if id == "" && fid == "" && name == "" && hostname == "" {
				agent, err = pickAgent(ctx, c)
			} else {
				agent, err = resolveAgent(ctx, c, id, name, hostname, fid)
			}
			if err != nil || agent == nil {
				return err
			}
			if !force {
				if err = checkGroupConfigs(ctx, c, agent); err != nil {
					return err
				}
			}
			form, err := c.GetAgentConfigForm(ctx, agent.ID)
			if err != nil {
				return err
			}
			props, err := form.Filter(client.AgentConfigFilter{})
			if err != nil {
				return err
			}
			changes, err := tui.EditAgentConfigs(props, tui.AgentConfigEditorOptions{
				Title:       "Agent: " + agent.Name,
				CustomTypes: form.CustomTypes(),
			})
			if errors.Is(err, tui.ErrCanceled) {
				_, _ = fmt.Fprintln(w, "Canceled.")
				return nil
			}
			if err != nil {
				return err
			}
			diffs, err := form.Apply(changes)
			if err != nil {
				return err
			}
			if len(diffs) == 0 {
				_, _ = fmt.Fprintln(w, "No changes.")
				return nil
			}
			_, _ = fmt.Fprintf(w, "Changes for agent %s (%s):\n\n", agent.Name, agent.ID)
			cfg, _ := loadConfig()
			if err := output.New(output.FormatTable, w, resolveTableStyle(cfg)).Format(diffs, agentConfigDiffCols); err != nil {
				return err
			}
			return saveAgentConfigDiffs(ctx, cmd, c, agent, form, diffs)
		},
	}
	addAgentSelectorFlags(cmd, &id, &fid, &name, &hostname)
	cmd.Flags().BoolVar(&force, "force", false, "edit the agent even if its group has group-level properties")
	return cmd
}
