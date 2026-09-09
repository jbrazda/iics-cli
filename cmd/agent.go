package cmd

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/config"
	"github.com/jbrazda/iics-cli/internal/filter"
	"github.com/jbrazda/iics-cli/internal/output"
	"github.com/spf13/cobra"
)

func newAgentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Manage secure agents",
	}
	cmd.AddCommand(newAgentListCmd())
	cmd.AddCommand(newAgentGetCmd())
	cmd.AddCommand(newAgentDetailsCmd())
	cmd.AddCommand(newAgentDeleteCmd())
	cmd.AddCommand(newAgentStartCmd())
	cmd.AddCommand(newAgentStopCmd())
	cmd.AddCommand(newAgentInstallerInfoCmd())
	cmd.AddCommand(newAgentInstallerDownloadCmd())
	return cmd
}

// agentColumnMap maps technical (JSON tag) field names to table columns for
// `agent list --fields`.
var agentColumnMap = map[string]output.Column{
	"id":               {Header: "ID", Field: "id", Width: 24},
	"orgId":            {Header: "ORG ID", Field: "orgId", Width: 12},
	"name":             {Header: "NAME", Field: "name", Width: 22},
	"description":      {Header: "DESCRIPTION", Field: "description", Width: 30},
	"agentHost":        {Header: "HOST", Field: "agentHost", Width: 20},
	"active":           {Header: "ACTIVE", Field: "active", Width: 7, Func: agentActiveFunc},
	"readyToRun":       {Header: "READY", Field: "readyToRun", Width: 7, Func: agentReadyFunc},
	"platform":         {Header: "PLATFORM", Field: "platform", Width: 9},
	"agentVersion":     {Header: "VERSION", Field: "agentVersion", Width: 9},
	"upgradeStatus":    {Header: "UPGRADE", Field: "upgradeStatus", Width: 13},
	"agentGroupId":     {Header: "GROUP ID", Field: "agentGroupId", Width: 24},
	"federatedId":      {Header: "FEDERATED ID", Field: "federatedId", Width: 24},
	"serverUrl":        {Header: "SERVER URL", Field: "serverUrl", Width: 30},
	"spiUrl":           {Header: "SPI URL", Field: "spiUrl", Width: 30},
	"proxyHost":        {Header: "PROXY HOST", Field: "proxyHost", Width: 18},
	"createdBy":        {Header: "CREATED BY", Field: "createdBy", Width: 18},
	"updatedBy":        {Header: "UPDATED BY", Field: "updatedBy", Width: 18},
	"createTime":       {Header: "CREATED", Field: "createTime", Width: 22},
	"updateTime":       {Header: "UPDATED", Field: "updateTime", Width: 22},
	"lastStatusChange": {Header: "LAST STATUS CHANGE", Field: "lastStatusChange", Width: 22},
	"lastUpgraded":     {Header: "LAST UPGRADED", Field: "lastUpgraded", Width: 22},
	"lastUpgradeCheck": {Header: "LAST UPGRADE CHECK", Field: "lastUpgradeCheck", Width: 22},
	"configUpdateTime": {Header: "CONFIG UPDATED", Field: "configUpdateTime", Width: 22},
	"createTimeUTC":    {Header: "CREATED (UTC)", Field: "createTimeUTC", Width: 22},
	"updateTimeUTC":    {Header: "UPDATED (UTC)", Field: "updateTimeUTC", Width: 22},
}

const agentListDefaultFields = "id,name,agentHost,active,readyToRun,platform,agentVersion,agentGroupId"

// agentColumnsFromFields resolves a comma-separated field list into columns,
// silently skipping unknown names. An empty string yields the default set.
func agentColumnsFromFields(fields string) []output.Column {
	if strings.TrimSpace(fields) == "" {
		fields = agentListDefaultFields
	}
	var columns []output.Column
	for _, name := range strings.Split(fields, ",") {
		if col, ok := agentColumnMap[strings.TrimSpace(name)]; ok {
			columns = append(columns, col)
		}
	}
	if len(columns) == 0 {
		columns = agentColumnsFromFields(agentListDefaultFields)
	}
	return columns
}

func agentBoolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// resolveAgent resolves a full agent from mutually exclusive selectors. Exactly
// one of id/name/host/fid must be non-empty.
func resolveAgent(ctx context.Context, c *client.Client, id, name, host, fid string) (*client.Agent, error) {
	if id != "" {
		return c.GetAgent(ctx, id)
	}
	return c.FindAgent(ctx, client.AgentSelector{Name: name, Hostname: host, FederatedID: fid})
}

// resolveAgentID resolves an agent ID from mutually exclusive selectors.
func resolveAgentID(ctx context.Context, c *client.Client, id, name, host, fid string) (string, error) {
	if id != "" {
		return id, nil
	}
	a, err := resolveAgent(ctx, c, id, name, host, fid)
	if err != nil {
		return "", err
	}
	return a.ID, nil
}

func newAgentListCmd() *cobra.Command {
	var (
		opts    client.AgentListOptions
		fields  string
		filters []string
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List secure agents",
		Example: `  iics agent list
  iics agent list --fields name,agentHost,active,agentVersion
  iics agent list --filter agentHost==devinfacld01 --filter active==true`,
		RunE: func(cmd *cobra.Command, args []string) error {
			preds, err := filter.ParseAll(filters)
			if err != nil {
				return err
			}
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			agents, err := c.ListAgents(context.Background(), opts)
			if err != nil {
				return err
			}
			f, err := getFormatter()
			if err != nil {
				return err
			}
			columns := agentColumnsFromFields(fields)
			if len(preds) > 0 {
				rows, err := filter.Apply(agents, preds)
				if err != nil {
					return err
				}
				return f.Format(rows, columns)
			}
			return f.Format(agents, columns)
		},
	}
	cmd.Flags().IntVar(&opts.Limit, "limit", 200, "max results (currently ignored by the API)")
	cmd.Flags().IntVar(&opts.Skip, "skip", 0, "number of results to skip (currently ignored by the API)")
	cmd.Flags().BoolVar(&opts.IncludeUnassignedOnly, "unassigned", false, "include only unassigned agents")
	cmd.Flags().BoolVar(&opts.BasicInfo, "basic-info", false, "include package and configuration details")
	cmd.Flags().StringVar(&fields, "fields", "", "comma-separated columns to display (technical field names)")
	cmd.Flags().StringArrayVar(&filters, "filter", nil, "client-side filter, e.g. agentHost==host01 or active!=true (repeatable, AND-ed)")
	return cmd
}

// agentToKVRows renders an agent as an ordered property/value list.
func agentToKVRows(a *client.Agent) []output.KVRow {
	rows := []output.KVRow{
		output.KV("id", a.ID),
		output.KV("orgId", a.OrgID),
		output.KV("name", a.Name),
	}
	if a.Description != "" {
		rows = append(rows, output.KV("description", a.Description))
	}
	rows = append(rows,
		output.KV("federatedId", a.FederatedID),
		output.KV("active", agentBoolStr(a.Active)),
		output.KV("readyToRun", agentBoolStr(a.ReadyToRun)),
		output.KV("platform", a.Platform),
		output.KV("agentHost", a.AgentHost),
		output.KV("agentVersion", a.AgentVersion),
		output.KV("upgradeStatus", a.UpgradeStatus),
		output.KV("lastUpgraded", a.LastUpgraded),
		output.KV("lastUpgradeCheck", a.LastUpgradeCheck),
		output.KV("lastStatusChange", a.LastStatusChange),
		output.KV("configUpdateTime", a.ConfigUpdateTime),
		output.KV("agentGroupId", a.GroupID),
	)
	if a.SpiURL != "" {
		rows = append(rows, output.KV("spiUrl", a.SpiURL))
	}
	if a.ServerURL != "" {
		rows = append(rows, output.KV("serverUrl", a.ServerURL))
	}
	if a.ProxyHost != "" {
		rows = append(rows,
			output.KV("proxyHost", a.ProxyHost),
			output.KV("proxyPort", strconv.Itoa(a.ProxyPort)),
			output.KV("proxyUser", a.ProxyUser),
		)
	}
	rows = append(rows,
		output.KV("createdBy", a.CreatedBy),
		output.KV("createTime", a.CreateTime),
		output.KV("updatedBy", a.UpdatedBy),
		output.KV("updateTime", a.UpdateTime),
	)
	return rows
}

func newAgentGetCmd() *cobra.Command {
	var id, name string
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a secure agent by ID or name",
		Example: `  iics agent get --id <agent-id>
  iics agent get --name "My Agent" --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" && name == "" {
				return fmt.Errorf("either --id or --name is required")
			}
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()
			var agent *client.Agent
			if id != "" {
				agent, err = c.GetAgent(ctx, id)
			} else {
				agent, err = c.GetAgentByName(ctx, name)
			}
			if err != nil {
				return err
			}
			f, err := getFormatter()
			if err != nil {
				return err
			}
			if outputFmt == "" || outputFmt == "table" {
				return f.Format(agentToKVRows(agent), output.KVCols)
			}
			return f.Format(agent, agentColumnsFromFields(""))
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "agent ID")
	cmd.Flags().StringVar(&name, "name", "", "agent name")
	cmd.MarkFlagsMutuallyExclusive("id", "name")
	return cmd
}

// agentDetailsToKVRows renders the agent summary section of an agent details response.
func agentDetailsToKVRows(d *client.AgentDetails) []output.KVRow {
	rows := agentToKVRows(&d.Agent)
	extra := []output.KVRow{
		output.KV("platformAgent", agentBoolStr(d.PlatformAgent)),
	}
	if d.ServerURL != "" {
		extra = append(extra, output.KV("serverUrl", d.ServerURL))
	}
	return append(rows, extra...)
}

var agentEngineConfigCols = []output.Column{
	{Header: "TYPE", Field: "type", Width: 14},
	{Header: "NAME", Field: "name", Width: 30},
	{Header: "VALUE", Field: "value", Width: 40, Func: wrapConfigField("value")},
	{Header: "DEFAULT", Field: "defaultValue", Width: 40, Func: wrapConfigField("defaultValue")},
	{Header: "CUSTOMIZED", Field: "customized", Width: 10, Func: func(v interface{}) string {
		row, _ := v.(map[string]interface{})
		if b, _ := row["customized"].(bool); b {
			return "yes"
		}
		return "no"
	}},
}

// wrapConfigField returns a column Func that hard-wraps the named field at 44
// columns so long values/defaults render as multi-line cells.
func wrapConfigField(field string) func(v interface{}) string {
	return func(v interface{}) string {
		row, _ := v.(map[string]interface{})
		s, _ := row[field].(string)
		return output.WrapCell(s, 44)
	}
}

func engineStatusKVRows(s client.AgentEngineStatus) []output.KVRow {
	return []output.KVRow{
		output.KV("status", s.Status),
		output.KV("desiredStatus", s.DesiredStatus),
		output.KV("subState", s.SubState),
		output.KV("replacePolicy", s.ReplacePolicy),
		output.KV("createTime", s.CreateTime),
		output.KV("updateTime", s.UpdateTime),
	}
}

func newAgentDetailsCmd() *cobra.Command {
	var (
		id, fid, name, hostname string
		full                    bool
	)
	cmd := &cobra.Command{
		Use:   "details",
		Short: "Get agent service engine details",
		Example: `  iics agent details --id <agent-id>
  iics agent details --hostname devinfacld01 --full
  iics agent details --name "My Agent" --output json`,
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
			details, err := c.GetAgentDetails(ctx, agentID, full)
			if err != nil {
				return err
			}
			f, err := getFormatter()
			if err != nil {
				return err
			}
			if outputFmt != "" && outputFmt != "table" {
				return f.Format(details, nil)
			}

			w := cmd.OutOrStdout()
			cfg, _ := loadConfig()
			tf := output.New(output.FormatTable, w, resolveTableStyle(cfg))

			_, _ = fmt.Fprintln(w, "Agent:")
			_, _ = fmt.Fprintln(w)
			if err := tf.Format(agentDetailsToKVRows(details), output.KVCols); err != nil {
				return err
			}

			if full && len(details.AgentConfigs) > 0 {
				_, _ = fmt.Fprintln(w)
				_, _ = fmt.Fprintln(w, "Agent Config:")
				_, _ = fmt.Fprintln(w)
				if err := tf.Format(details.AgentConfigs, agentEngineConfigCols); err != nil {
					return err
				}
			}

			for _, e := range details.AgentEngines {
				st := e.AgentEngineStatus
				_, _ = fmt.Fprintln(w)
				_, _ = fmt.Fprintf(w, "Service: %s (%s v%s)\n", st.AppDisplayName, st.AppName, st.AppVersion)
				_, _ = fmt.Fprintln(w)
				if err := tf.Format(engineStatusKVRows(st), output.KVCols); err != nil {
					return err
				}
				if full && len(e.AgentEngineConfigs) > 0 {
					_, _ = fmt.Fprintln(w)
					if err := tf.Format(e.AgentEngineConfigs, agentEngineConfigCols); err != nil {
						return err
					}
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "agent ID")
	cmd.Flags().StringVar(&fid, "fid", "", "agent federated ID")
	cmd.Flags().StringVar(&name, "name", "", "agent name")
	cmd.Flags().StringVar(&hostname, "hostname", "", "agent host name")
	cmd.Flags().BoolVar(&full, "full", false, "include agent and service configuration properties")
	cmd.MarkFlagsMutuallyExclusive("id", "fid", "name", "hostname")
	return cmd
}

func newAgentDeleteCmd() *cobra.Command {
	var (
		id, name, hostname string
		yes                bool
	)
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a secure agent",
		Example: `  iics agent delete --id <agent-id>
  iics agent delete --hostname devinfacld01 --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" && name == "" && hostname == "" {
				return fmt.Errorf("one of --id, --name, or --hostname is required")
			}
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()
			agentID, err := resolveAgentID(ctx, c, id, name, hostname, "")
			if err != nil {
				return err
			}
			if !yes {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Are you sure you want to delete agent %s? [y/N]: ", agentID)
				var confirm string
				_, _ = fmt.Scanln(&confirm)
				if confirm != "y" && confirm != "Y" {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Canceled.")
					return nil
				}
			}
			if err := c.DeleteAgent(ctx, agentID); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Agent deleted: %s\n", agentID)
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "agent ID")
	cmd.Flags().StringVar(&name, "name", "", "agent name")
	cmd.Flags().StringVar(&hostname, "hostname", "", "agent host name")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompt")
	cmd.MarkFlagsMutuallyExclusive("id", "name", "hostname")
	return cmd
}

func newAgentStartCmd() *cobra.Command {
	return newAgentServiceCmd("start", "Start an agent service", client.AgentServiceStart, "started")
}

func newAgentStopCmd() *cobra.Command {
	return newAgentServiceCmd("stop", "Stop an agent service", client.AgentServiceStop, "stopped")
}

func newAgentServiceCmd(use, short string, action client.AgentServiceAction, past string) *cobra.Command {
	var (
		id, name, hostname string
		service            string
		interactive        bool
		blocking           bool
		pollInterval       int
		maxWaitTime        int
	)
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Example: fmt.Sprintf(`  iics agent %s --id <agent-id> --service "Data Integration Server"
  iics agent %s --hostname devinfacld01 --service "Process Server"
  iics agent %s -i`, use, use, use),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			// Resolve the agent: explicit selector, or an interactive picker.
			var agent *client.Agent
			if id != "" || name != "" || hostname != "" {
				agent, err = resolveAgent(ctx, c, id, name, hostname, "")
				if err != nil {
					return err
				}
			} else if interactive {
				agent, err = pickAgent(ctx, c)
				if err != nil {
					return err
				}
				if agent == nil {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Canceled.")
					return nil
				}
			} else {
				return fmt.Errorf("one of --id, --name, --hostname, or --interactive is required")
			}
			if agent.FederatedID == "" {
				return fmt.Errorf("agent %q has no federated ID; cannot control its services", agent.Name)
			}
			agentID := agent.ID
			agentLabel := fmt.Sprintf("%s (%s)", agent.Name, agent.AgentHost)

			// Resolve the service: --service, or pick from the agent's engines.
			svc := service
			if svc == "" {
				if !interactive {
					return fmt.Errorf("--service is required")
				}
				svc, err = pickAgentService(ctx, c, agentID, use)
				if err != nil {
					return err
				}
				if svc == "" {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Canceled.")
					return nil
				}
			}

			if interactive {
				ok, cerr := promptYesNo(fmt.Sprintf("%s service %q on agent %s", strings.ToUpper(use[:1])+use[1:], svc, agentLabel), true)
				if cerr != nil {
					return cerr
				}
				if !ok {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Canceled.")
					return nil
				}
			}

			// Decide whether to wait for the service to reach its target state.
			doBlock := blocking
			if interactive && !cmd.Flags().Changed("blocking") {
				doBlock, err = promptYesNo(fmt.Sprintf("Wait for service %q to %s", svc, use), true)
				if err != nil {
					return err
				}
			}

			if err := c.SetAgentServiceState(ctx, agent.FederatedID, svc, action); err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			_, _ = fmt.Fprintf(w, "Service %q %s on agent %s\n", svc, past, agentID)

			if doBlock {
				return waitForAgentService(ctx, c, agentID, svc, use, past, pollInterval, maxWaitTime, w)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "agent ID")
	cmd.Flags().StringVar(&name, "name", "", "agent name")
	cmd.Flags().StringVar(&hostname, "hostname", "", "agent host name")
	cmd.Flags().StringVar(&service, "service", "", "service name (required unless --interactive)")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "select the agent and service interactively")
	cmd.Flags().BoolVar(&blocking, "blocking", false, fmt.Sprintf("poll the service status until it has %s", past))
	cmd.Flags().IntVar(&pollInterval, "poll-interval", 10, "seconds between status polls (with --blocking)")
	cmd.Flags().IntVar(&maxWaitTime, "max-wait-time", 300, "maximum seconds to wait (with --blocking)")
	cmd.MarkFlagsMutuallyExclusive("id", "name", "hostname")
	return cmd
}

// serviceStates returns the statuses of every engine on the agent matching the
// given service display name.
func serviceStates(d *client.AgentDetails, svc string) []client.AgentEngineStatus {
	var out []client.AgentEngineStatus
	for _, e := range d.AgentEngines {
		if e.AgentEngineStatus.AppDisplayName == svc {
			out = append(out, e.AgentEngineStatus)
		}
	}
	return out
}

// serviceReached reports whether the service has reached the target state for
// verb ("start"/"stop"). It returns an error if any matching engine is in ERROR.
func serviceReached(verb string, states []client.AgentEngineStatus) (bool, error) {
	for _, s := range states {
		if strings.EqualFold(s.Status, "ERROR") {
			return false, fmt.Errorf("service is in ERROR state")
		}
	}
	pending := func(status string) bool {
		switch strings.ToUpper(status) {
		case "NEED_RUNNING", "NEED_STOP", "DEPLOYING", "STARTING", "STOPPING":
			return true
		}
		return false
	}
	if verb == "start" {
		running := false
		for _, s := range states {
			if strings.EqualFold(s.Status, "RUNNING") {
				running = true
			}
			if pending(s.Status) {
				return false, nil
			}
		}
		return running, nil
	}
	// stop
	for _, s := range states {
		if strings.EqualFold(s.Status, "RUNNING") || pending(s.Status) {
			return false, nil
		}
	}
	return true, nil
}

// waitForAgentService polls the agent details until the service reaches the
// target state for verb, printing each poll, or until maxWaitSec elapses.
func waitForAgentService(ctx context.Context, c *client.Client, agentID, svc, verb, past string, pollSec, maxWaitSec int, w io.Writer) error {
	if pollSec < 1 {
		pollSec = 1
	}
	poll := time.Duration(pollSec) * time.Second
	deadline := time.Now().Add(time.Duration(maxWaitSec) * time.Second)
	for {
		// Wait before each poll (including the first) so the control plane has
		// time to reflect the action just initiated.
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(poll):
		}

		details, err := c.GetAgentDetails(ctx, agentID, false)
		if err != nil {
			return err
		}
		states := serviceStates(details, svc)
		if len(states) == 0 {
			_, _ = fmt.Fprintf(w, "%s%s: (not reported)\n", ts(), svc)
		}
		for _, s := range states {
			_, _ = fmt.Fprintf(w, "%s%s: %s (desired %s)\n", ts(), svc, s.Status, s.DesiredStatus)
		}

		done, rerr := serviceReached(verb, states)
		if rerr != nil {
			return fmt.Errorf("%s: %w", svc, rerr)
		}
		if done {
			_, _ = fmt.Fprintf(w, "%sService %q %s.\n", ts(), svc, past)
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out after %ds waiting for service %q to %s", maxWaitSec, svc, verb)
		}
	}
}

// pickAgent lists the secure agents and lets the operator choose one.
func pickAgent(ctx context.Context, c *client.Client) (*client.Agent, error) {
	if !config.IsTerminal() {
		return nil, fmt.Errorf("--interactive requires a terminal")
	}
	agents, err := c.ListAgents(ctx, client.AgentListOptions{})
	if err != nil {
		return nil, err
	}
	if len(agents) == 0 {
		return nil, fmt.Errorf("no agents found")
	}
	labels := make([]string, len(agents))
	for i, a := range agents {
		labels[i] = fmt.Sprintf("%s (%s) - v%s", a.Name, a.AgentHost, a.AgentVersion)
	}
	idx, err := promptSelect("Select an agent", labels)
	if err != nil {
		return nil, err
	}
	if idx < 0 {
		return nil, nil
	}
	return &agents[idx], nil
}

// pickAgentService fetches the agent's service engines and lets the operator
// choose one. verb is the action name used in the prompt ("start"/"stop").
func pickAgentService(ctx context.Context, c *client.Client, agentID, verb string) (string, error) {
	details, err := c.GetAgentDetails(ctx, agentID, false)
	if err != nil {
		return "", err
	}
	if len(details.AgentEngines) == 0 {
		return "", fmt.Errorf("no services found on agent %s", agentID)
	}
	labels := make([]string, len(details.AgentEngines))
	for i, e := range details.AgentEngines {
		st := e.AgentEngineStatus
		labels[i] = fmt.Sprintf("%s (%s)", st.AppDisplayName, st.Status)
	}
	idx, err := promptSelect(fmt.Sprintf("Select a service to %s", verb), labels)
	if err != nil {
		return "", err
	}
	if idx < 0 {
		return "", nil
	}
	return details.AgentEngines[idx].AgentEngineStatus.AppDisplayName, nil
}
