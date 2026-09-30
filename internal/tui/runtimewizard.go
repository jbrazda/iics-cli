package tui

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/jbrazda/iics-cli/internal/client"
)

// RuntimeWizardInput configures RunRuntimeWizard.
type RuntimeWizardInput struct {
	// Env holds the starting values (e.g. from --from-file) and receives the
	// result.
	Env *client.RuntimeEnvironment
	// Agents are the Secure Agents that can be added (not assigned to
	// another environment).
	Agents []client.Agent
	// ExistingNames are the names of existing environments, rejected.
	ExistingNames []string

	Out        io.Writer
	Accessible bool
}

// RunRuntimeWizard collects a new runtime environment's name, description,
// sharing and member agents, then shows a review. It returns false if the
// user canceled.
func RunRuntimeWizard(in RuntimeWizardInput) (bool, error) {
	env := in.Env
	items := agentItems(in.Agents, env.Agents)
	agentIDs := make([]string, 0, len(env.Agents))
	for _, a := range env.Agents {
		if a.ID != "" {
			agentIDs = append(agentIDs, a.ID)
		}
	}

	for {
		fields := []*huh.Group{
			huh.NewGroup(
				huh.NewInput().Title("Environment name").Value(&env.Name).
					Validate(requiredUnique("environment name", in.ExistingNames)),
				huh.NewInput().Title("Description (optional)").Value(&env.Description),
				huh.NewConfirm().Title("Shared environment?").Value(&env.IsShared),
			).Title("Runtime environment"),
		}
		if len(items) > 0 {
			opts := make([]huh.Option[string], len(items))
			for i, it := range items {
				opts[i] = huh.NewOption(it.label, it.id)
			}
			fields = append(fields, huh.NewGroup(
				huh.NewMultiSelect[string]().
					Title("Secure Agents (optional)").
					Description("Unassigned agents. space toggle, / filter, ctrl+a all").
					Options(opts...).
					Filterable(true).
					Height(min(len(opts)+3, 14)).
					Value(&agentIDs),
			).Title("Agents"))
		} else {
			fields = append(fields, huh.NewGroup(
				huh.NewNote().Title("Agents").Description("No unassigned Secure Agents are available. Agents can be added later."),
			))
		}
		err := huh.NewForm(fields...).WithOutput(in.Out).WithAccessible(in.Accessible).Run()
		if errors.Is(err, huh.ErrUserAborted) {
			return false, nil
		}
		if err != nil {
			return false, err
		}

		env.Name = strings.TrimSpace(env.Name)
		env.Description = strings.TrimSpace(env.Description)
		want := toSet(agentIDs)
		env.Agents = nil
		var names []string
		for _, it := range items {
			if want[it.id] {
				env.Agents = append(env.Agents, client.RuntimeEnvironmentAgent{ID: it.id})
				names = append(names, it.name)
			}
		}

		lines := []string{
			"Name:        " + env.Name,
			"Description: " + orDash(env.Description),
			fmt.Sprintf("Shared:      %t", env.IsShared),
			"Agents:      " + orDash(strings.Join(names, ", ")),
		}
		choice, err := confirmReview(in.Out, in.Accessible, "Create runtime environment "+env.Name, lines, "Create environment", true)
		if err != nil {
			return false, err
		}
		switch choice {
		case reviewApply:
			return true, nil
		case reviewCancel:
			return false, nil
		}
	}
}

type agentItem struct {
	id, name, label string
}

// agentItems builds checklist entries from the available agents plus any
// seeded agents not in that list (e.g. from --from-file), sorted by name.
func agentItems(available []client.Agent, seeded []client.RuntimeEnvironmentAgent) []agentItem {
	seen := make(map[string]bool)
	var items []agentItem
	for _, a := range available {
		if a.ID == "" || seen[a.ID] {
			continue
		}
		seen[a.ID] = true
		status := "inactive"
		if a.Active {
			status = "active"
		}
		label := a.Name
		if a.AgentHost != "" {
			label += " (" + a.AgentHost + ")"
		}
		items = append(items, agentItem{id: a.ID, name: a.Name, label: label + " - " + status})
	}
	for _, a := range seeded {
		if a.ID == "" || seen[a.ID] {
			continue
		}
		seen[a.ID] = true
		name := a.Name
		if name == "" {
			name = a.ID
		}
		items = append(items, agentItem{id: a.ID, name: name, label: name + " (from file)"})
	}
	sort.SliceStable(items, func(i, j int) bool {
		return strings.ToLower(items[i].name) < strings.ToLower(items[j].name)
	})
	return items
}
