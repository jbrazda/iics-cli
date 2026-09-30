package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/config"
	"github.com/jbrazda/iics-cli/internal/tui"
)

// runRuntimeCreateWizard interactively collects the fields for a new runtime
// environment. rt is used both as the seed (from --from-file) and the result.
// It returns false if the user canceled.
func runRuntimeCreateWizard(ctx context.Context, c *client.Client, rt *client.RuntimeEnvironment) (bool, error) {
	if !config.IsTerminal() {
		return false, fmt.Errorf("--interactive requires a terminal; use --from-file")
	}
	agents, err := c.ListAgents(ctx, client.AgentListOptions{IncludeUnassignedOnly: true})
	if err != nil {
		return false, err
	}
	envs, err := c.ListRuntimeEnvironments(ctx, client.RuntimeListOptions{})
	if err != nil {
		return false, err
	}
	names := make([]string, len(envs))
	for i, e := range envs {
		names[i] = e.Name
	}
	ok, err := tui.RunRuntimeWizard(tui.RuntimeWizardInput{
		Env:           rt,
		Agents:        agents,
		ExistingNames: names,
		Out:           os.Stderr,
		Accessible:    prompter.Accessible,
	})
	if err != nil || !ok {
		return false, err
	}
	rt.Agents = trimAgentRefs(rt.Agents)
	return true, nil
}

// trimAgentRefs reduces each agent to the fields the create request needs.
func trimAgentRefs(agents []client.RuntimeEnvironmentAgent) []client.RuntimeEnvironmentAgent {
	if len(agents) == 0 {
		return nil
	}
	out := make([]client.RuntimeEnvironmentAgent, len(agents))
	for i, a := range agents {
		out[i] = client.RuntimeEnvironmentAgent{ID: a.ID}
	}
	return out
}
