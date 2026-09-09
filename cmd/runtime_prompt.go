package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/config"
)

// runRuntimeCreateWizard interactively collects the fields for a new runtime
// environment. rt is used both as the seed (from --from-file) and the result.
func runRuntimeCreateWizard(ctx context.Context, c *client.Client, rt *client.RuntimeEnvironment) error {
	if !config.IsTerminal() {
		return fmt.Errorf("--interactive requires a terminal; use --from-file")
	}

	// Name (required).
	for {
		name, err := promptText("Environment Name", rt.Name)
		if err != nil {
			return err
		}
		if name != "" {
			rt.Name = name
			break
		}
		_, _ = fmt.Fprintln(os.Stderr, "Environment name is required.")
	}

	shared, err := promptYesNo("Is shared", rt.IsShared)
	if err != nil {
		return err
	}
	rt.IsShared = shared

	// Working set of member agents, keyed by ID (deduped). Seeded from rt.Agents.
	selected := make([]client.RuntimeEnvironmentAgent, 0, len(rt.Agents))
	inSet := make(map[string]bool)
	for _, a := range rt.Agents {
		if a.ID != "" && !inSet[a.ID] {
			selected = append(selected, a)
			inSet[a.ID] = true
		}
	}

	for {
		choice, err := promptSelect("Manage agents", []string{"Add agents", "Remove agents", "Done"})
		if err != nil {
			return err
		}
		switch choice {
		case 0: // Add
			agents, err := c.ListAgents(ctx, client.AgentListOptions{IncludeUnassignedOnly: true})
			if err != nil {
				return err
			}
			var avail []client.Agent
			for _, a := range agents {
				if !inSet[a.ID] {
					avail = append(avail, a)
				}
			}
			if len(avail) == 0 {
				_, _ = fmt.Fprintln(os.Stderr, "No unassigned agents available.")
				continue
			}
			labels := make([]string, len(avail))
			for i, a := range avail {
				labels[i] = fmt.Sprintf("%s (%s)", a.Name, a.AgentHost)
			}
			picks, err := promptMultiSelect("Unassigned agents to add", labels, nil)
			if err != nil {
				return err
			}
			for _, idx := range picks {
				a := avail[idx]
				selected = append(selected, client.RuntimeEnvironmentAgent{ID: a.ID, Name: a.Name})
				inSet[a.ID] = true
			}
		case 1: // Remove
			if len(selected) == 0 {
				_, _ = fmt.Fprintln(os.Stderr, "No agents selected.")
				continue
			}
			labels := make([]string, len(selected))
			for i, a := range selected {
				labels[i] = fmt.Sprintf("%s (%s)", a.Name, a.ID)
			}
			picks, err := promptMultiSelect("Agents to remove", labels, nil)
			if err != nil {
				return err
			}
			drop := make(map[int]bool, len(picks))
			for _, idx := range picks {
				drop[idx] = true
			}
			kept := selected[:0]
			for i, a := range selected {
				if drop[i] {
					delete(inSet, a.ID)
					continue
				}
				kept = append(kept, a)
			}
			selected = kept
		default: // 2 Done, or Cancel (-1)
			rt.Agents = trimAgentRefs(selected)
			_, _ = fmt.Fprintf(os.Stderr, "%d agent(s) selected.\n", len(rt.Agents))
			return nil
		}
		_, _ = fmt.Fprintf(os.Stderr, "%d agent(s) currently selected.\n", len(selected))
	}
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
