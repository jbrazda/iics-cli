package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/config"
	"github.com/spf13/cobra"
)

// agentServiceFlags is the shared flag set for agent start / stop / restart.
type agentServiceFlags struct {
	id, name, hostname string
	service            string
	interactive        bool
	blocking           bool
	pollInterval       int
	maxWaitTime        int
}

func (f *agentServiceFlags) bind(cmd *cobra.Command, past string) {
	cmd.Flags().StringVar(&f.id, "id", "", "agent ID")
	cmd.Flags().StringVar(&f.name, "name", "", "agent name")
	cmd.Flags().StringVar(&f.hostname, "hostname", "", "agent host name")
	cmd.Flags().StringVar(&f.service, "service", "", "service display name (required unless --interactive)")
	cmd.Flags().BoolVarP(&f.interactive, "interactive", "i", false, "select the agent and service interactively")
	cmd.Flags().BoolVar(&f.blocking, "blocking", false, fmt.Sprintf("poll the service status until it has %s", past))
	cmd.Flags().IntVar(&f.pollInterval, "poll-interval", 10, "seconds between status polls (with --blocking)")
	cmd.Flags().IntVar(&f.maxWaitTime, "max-wait-time", 300, "maximum seconds to wait (with --blocking)")
	cmd.MarkFlagsMutuallyExclusive("id", "name", "hostname")
}

func titleVerb(v string) string {
	if v == "" {
		return v
	}
	return strings.ToUpper(v[:1]) + v[1:]
}

// resolveServiceTarget resolves the agent and service for an agent service
// command. canceled is true when the operator aborted an interactive prompt.
func resolveServiceTarget(ctx context.Context, c *client.Client, f *agentServiceFlags, verb string) (agent *client.Agent, svc string, canceled bool, err error) {
	switch {
	case f.id != "" || f.name != "" || f.hostname != "":
		agent, err = resolveAgent(ctx, c, f.id, f.name, f.hostname, "")
	case f.interactive:
		agent, err = pickAgent(ctx, c)
		if err == nil && agent == nil {
			return nil, "", true, nil
		}
	default:
		return nil, "", false, fmt.Errorf("one of --id, --name, --hostname, or --interactive is required")
	}
	if err != nil {
		return nil, "", false, err
	}
	if agent.FederatedID == "" {
		return nil, "", false, fmt.Errorf("agent %q has no federated ID; cannot control its services", agent.Name)
	}

	svc = f.service
	if svc == "" {
		if !f.interactive {
			return nil, "", false, fmt.Errorf("--service is required")
		}
		if verb == "start" {
			svc, err = promptStartService(ctx, c, agent.ID)
		} else {
			svc, err = pickAgentService(ctx, c, agent.ID, verb)
		}
		if err != nil {
			return nil, "", false, err
		}
		if svc == "" {
			return nil, "", true, nil
		}
	}
	return agent, svc, false, nil
}

func newAgentStartCmd() *cobra.Command {
	return newAgentServiceCmd("start", "Start an agent service", client.AgentServiceStart, "started")
}

func newAgentStopCmd() *cobra.Command {
	return newAgentServiceCmd("stop", "Stop an agent service", client.AgentServiceStop, "stopped")
}

func newAgentServiceCmd(use, short string, action client.AgentServiceAction, past string) *cobra.Command {
	var f agentServiceFlags
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Example: fmt.Sprintf(`  iics agent %s --id <agent-id> --service "Data Integration Server"
  iics agent %s --hostname devinfacld01 --service "Process Server" --blocking
  iics agent %s -i`, use, use, use),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()
			w := cmd.OutOrStdout()

			agent, svc, canceled, err := resolveServiceTarget(ctx, c, &f, use)
			if err != nil {
				return err
			}
			if canceled {
				_, _ = fmt.Fprintln(w, "Canceled.")
				return nil
			}
			label := fmt.Sprintf("%s (%s)", agent.Name, agent.AgentHost)

			if f.interactive {
				ok, cerr := promptYesNo(fmt.Sprintf("%s service %q on agent %s", titleVerb(use), svc, label), true)
				if cerr != nil {
					return cerr
				}
				if !ok {
					_, _ = fmt.Fprintln(w, "Canceled.")
					return nil
				}
			}

			doBlock := f.blocking
			if f.interactive && !cmd.Flags().Changed("blocking") {
				doBlock, err = promptYesNo(fmt.Sprintf("Wait for service %q to %s", svc, use), true)
				if err != nil {
					return err
				}
			}

			if err := c.SetAgentServiceState(ctx, agent.FederatedID, svc, action); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(w, "Service %q %s on agent %s\n", svc, past, agent.ID)

			if doBlock {
				deadline := time.Now().Add(time.Duration(f.maxWaitTime) * time.Second)
				return waitForAgentService(ctx, c, agent.ID, svc, use, past, f.pollInterval, deadline, w)
			}
			return nil
		},
	}
	f.bind(cmd, past)
	return cmd
}

func newAgentRestartCmd() *cobra.Command {
	var f agentServiceFlags
	cmd := &cobra.Command{
		Use:   "restart",
		Short: "Restart an agent service (stop, wait, start)",
		Long: `Restart a service on a Secure Agent.

The service is stopped and the command always waits for it to leave the agent's
service listing, then it is started again. With --blocking (or after the
interactive prompt) it also waits for the service to come back up (RUNNING with
subState 0). --max-wait-time bounds the whole operation.`,
		Example: `  iics agent restart --id <agent-id> --service "Data Integration Server"
  iics agent restart -i`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()
			w := cmd.OutOrStdout()

			agent, svc, canceled, err := resolveServiceTarget(ctx, c, &f, "restart")
			if err != nil {
				return err
			}
			if canceled {
				_, _ = fmt.Fprintln(w, "Canceled.")
				return nil
			}
			label := fmt.Sprintf("%s (%s)", agent.Name, agent.AgentHost)

			if f.interactive {
				ok, cerr := promptYesNo(fmt.Sprintf("Restart service %q on agent %s", svc, label), true)
				if cerr != nil {
					return cerr
				}
				if !ok {
					_, _ = fmt.Fprintln(w, "Canceled.")
					return nil
				}
			}

			waitStart := f.blocking
			if f.interactive && !cmd.Flags().Changed("blocking") {
				waitStart, err = promptYesNo(fmt.Sprintf("Wait for service %q to start", svc), true)
				if err != nil {
					return err
				}
			}

			deadline := time.Now().Add(time.Duration(f.maxWaitTime) * time.Second)

			if err := c.SetAgentServiceState(ctx, agent.FederatedID, svc, client.AgentServiceStop); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(w, "Service %q stop initiated on agent %s\n", svc, agent.ID)
			if err := waitForAgentService(ctx, c, agent.ID, svc, "stop", "stopped", f.pollInterval, deadline, w); err != nil {
				return err
			}

			if err := c.SetAgentServiceState(ctx, agent.FederatedID, svc, client.AgentServiceStart); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(w, "Service %q started on agent %s\n", svc, agent.ID)
			if waitStart {
				return waitForAgentService(ctx, c, agent.ID, svc, "start", "started", f.pollInterval, deadline, w)
			}
			return nil
		},
	}
	f.bind(cmd, "restarted")
	return cmd
}

// pickAgent lists the secure agents and lets the operator choose one.
// Returns nil when the operator cancels.
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

// pickAgentService lets the operator choose one of the agent's running service
// engines. verb is used in the prompt label. Returns "" on cancel.
func pickAgentService(ctx context.Context, c *client.Client, agentID, verb string) (string, error) {
	details, err := c.GetAgentDetails(ctx, agentID, false)
	if err != nil {
		return "", err
	}
	if len(details.AgentEngines) == 0 {
		return "", fmt.Errorf("no running services found on agent %s", agentID)
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

// promptStartService asks for the display name of the service to start. Stopped
// services are not returned by any API, so this is a free-text prompt; the
// currently-running engines are shown for reference. Returns "" on cancel.
func promptStartService(ctx context.Context, c *client.Client, agentID string) (string, error) {
	details, err := c.GetAgentDetails(ctx, agentID, false)
	if err != nil {
		return "", err
	}
	if len(details.AgentEngines) > 0 {
		running := make([]string, len(details.AgentEngines))
		for i, e := range details.AgentEngines {
			running[i] = e.AgentEngineStatus.AppDisplayName
		}
		_, _ = fmt.Fprintf(os.Stderr, "(already running: %s)\n", strings.Join(running, ", "))
	}
	return promptText("Service to start (display name)", "")
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

func servicePending(status string) bool {
	switch strings.ToUpper(status) {
	case "NEED_RUNNING", "NEED_STOP", "DEPLOYING", "STARTING", "STOPPING":
		return true
	}
	return false
}

// serviceReached reports whether the service has reached the target state for
// verb ("start"/"stop"). For "start" the engine must be RUNNING with subState
// "0" (operational). Returns an error if any matching engine is in ERROR.
func serviceReached(verb string, states []client.AgentEngineStatus) (bool, error) {
	for _, s := range states {
		if strings.EqualFold(s.Status, "ERROR") {
			return false, fmt.Errorf("service is in ERROR state")
		}
	}
	if verb == "start" {
		operational := false
		for _, s := range states {
			if servicePending(s.Status) {
				return false, nil
			}
			if strings.EqualFold(s.Status, "RUNNING") {
				// subState "0" means the engine is fully operational; "-1"
				// (and other values) mean it is still initializing.
				if s.SubState == "0" {
					operational = true
				} else {
					return false, nil
				}
			}
		}
		return operational, nil
	}
	// stop: done when no matching engine remains running or transitioning.
	for _, s := range states {
		if strings.EqualFold(s.Status, "RUNNING") || servicePending(s.Status) {
			return false, nil
		}
	}
	return true, nil
}

// waitForAgentService polls the agent details until the service reaches the
// target state for verb, printing each poll, or until deadline passes.
func waitForAgentService(ctx context.Context, c *client.Client, agentID, svc, verb, past string, pollSec int, deadline time.Time, w io.Writer) error {
	if pollSec < 1 {
		pollSec = 1
	}
	poll := time.Duration(pollSec) * time.Second
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
			_, _ = fmt.Fprintf(w, "%s%s: %s (desired %s, subState %s)\n", ts(), svc, s.Status, s.DesiredStatus, s.SubState)
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
			return fmt.Errorf("timed out waiting for service %q to %s", svc, verb)
		}
	}
}
