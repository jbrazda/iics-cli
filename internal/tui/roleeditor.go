// Package tui holds interactive terminal UIs built on huh and Bubble Tea.
package tui

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/tui/privmatrix"
)

// ErrCanceled is returned when the user cancels an interactive flow.
var ErrCanceled = errors.New("canceled")

// PrivilegeEditorOptions configures EditPrivileges.
type PrivilegeEditorOptions struct {
	// Title is shown above each screen, e.g. "Role: CAI Viewer".
	Title string
	// Mode restricts the edit to adding or removing privileges.
	Mode privmatrix.Mode
	// ApplyLabel is the verb on the final confirmation, e.g. "Apply" or "Create role".
	ApplyLabel string
	NoColor    bool
}

const (
	choiceReview = "\x00review"
	choiceCancel = "\x00cancel"
)

// EditPrivileges runs the interactive privilege editor. assigned holds the
// privilege names the role has now; initial holds the starting selection
// (usually equal to assigned, or a cloned role's privileges on create).
// It returns the desired set of privilege names, sorted.
func EditPrivileges(all []client.Privilege, assigned, initial []string, opts PrivilegeEditorOptions) ([]string, error) {
	services := client.GroupPrivileges(all)
	assignedSet := toSet(assigned)
	selected := toSet(initial)
	if opts.ApplyLabel == "" {
		opts.ApplyLabel = "Apply"
	}

	if opts.Mode == privmatrix.ModeRemove {
		services = filterServices(services, assignedSet)
		if len(services) == 0 {
			return nil, fmt.Errorf("role has no privileges to remove")
		}
	}

	lastService := ""
	for {
		choice := lastService
		options := serviceOptions(services, assignedSet, selected)
		sel := huh.NewSelect[string]().
			Title(opts.Title).
			Description(summaryLine(assignedSet, selected)).
			Options(options...).
			Height(min(len(options)+2, 22)).
			Value(&choice)
		if err := runForm(huh.NewForm(huh.NewGroup(sel))); err != nil {
			return nil, err
		}

		switch choice {
		case choiceCancel:
			return nil, ErrCanceled
		case choiceReview:
			desired, ok, err := review(services, assignedSet, selected, opts)
			if err != nil {
				return nil, err
			}
			if ok {
				return desired, nil
			}
			continue
		}

		lastService = choice
		for i := range services {
			if services[i].Name != choice {
				continue
			}
			m := privmatrix.New(services[i], assignedSet, selected, privmatrix.Options{
				Title:   opts.Title,
				Mode:    opts.Mode,
				NoColor: opts.NoColor,
			})
			p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithOutput(os.Stderr))
			if _, err := p.Run(); err != nil {
				return nil, fmt.Errorf("privilege editor: %w", err)
			}
		}
	}
}

// review prints the pending changes and asks whether to apply them. ok is
// false when the user chose to keep editing.
func review(services []client.PrivilegeService, assigned, selected map[string]bool, opts PrivilegeEditorOptions) ([]string, bool, error) {
	desired := setToSorted(selected)
	add, remove := client.DiffPrivileges(setToSorted(assigned), desired)

	var b strings.Builder
	if len(add) == 0 && len(remove) == 0 {
		b.WriteString("No changes.")
	}
	serviceOf := make(map[string]string)
	for _, s := range services {
		for _, o := range s.Objects {
			for _, p := range o.Actions {
				serviceOf[p.Name] = s.Name
			}
		}
		for _, p := range s.Other {
			serviceOf[p.Name] = s.Name
		}
	}
	// Keep the review compact: huh descriptions do not scroll.
	const maxLines = 24
	lines := 0
	writeGroup := func(sign string, names []string) {
		byService := make(map[string][]string)
		for _, n := range names {
			byService[serviceOf[n]] = append(byService[serviceOf[n]], n)
		}
		keys := make([]string, 0, len(byService))
		for k := range byService {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if lines >= maxLines {
				fmt.Fprintf(&b, "  ... and %d more in %s\n", len(byService[k]), k)
				continue
			}
			fmt.Fprintf(&b, "%s\n", k)
			lines++
			for i, n := range byService[k] {
				if lines >= maxLines {
					fmt.Fprintf(&b, "  ... and %d more\n", len(byService[k])-i)
					break
				}
				fmt.Fprintf(&b, "  %s %s\n", sign, n)
				lines++
			}
		}
	}
	if len(add) > 0 {
		fmt.Fprintf(&b, "Add (%d):\n", len(add))
		writeGroup("+", add)
	}
	if len(remove) > 0 {
		fmt.Fprintf(&b, "Remove (%d):\n", len(remove))
		writeGroup("-", remove)
	}

	options := []huh.Option[string]{huh.NewOption("Back to editing", "back")}
	if len(desired) == 0 {
		b.WriteString("\nA role must keep at least one privilege.")
	} else if len(add) > 0 || len(remove) > 0 || opts.ApplyLabel != "Apply" {
		options = append([]huh.Option[string]{huh.NewOption(opts.ApplyLabel, "apply")}, options...)
	}
	options = append(options, huh.NewOption("Cancel", "cancel"))

	choice := options[0].Value
	sel := huh.NewSelect[string]().
		Title(opts.Title + " - review").
		Description(strings.TrimRight(b.String(), "\n")).
		Options(options...).
		Value(&choice)
	if err := runForm(huh.NewForm(huh.NewGroup(sel))); err != nil {
		return nil, false, err
	}
	switch choice {
	case "apply":
		return desired, true, nil
	case "cancel":
		return nil, false, ErrCanceled
	}
	return nil, false, nil
}

func serviceOptions(services []client.PrivilegeService, assigned, selected map[string]bool) []huh.Option[string] {
	width := 0
	for _, s := range services {
		if len(s.Name) > width {
			width = len(s.Name)
		}
	}
	opts := make([]huh.Option[string], 0, len(services)+2)
	for _, s := range services {
		total, have, add, remove := 0, 0, 0, 0
		count := func(p client.Privilege) {
			total++
			if selected[p.Name] {
				have++
			}
			if selected[p.Name] && !assigned[p.Name] {
				add++
			}
			if !selected[p.Name] && assigned[p.Name] {
				remove++
			}
		}
		for _, o := range s.Objects {
			for _, p := range o.Actions {
				count(p)
			}
		}
		for _, p := range s.Other {
			count(p)
		}
		label := fmt.Sprintf("%-*s  %4d privileges  %4d selected", width, s.Name, total, have)
		if add > 0 || remove > 0 {
			label += fmt.Sprintf("  +%d -%d", add, remove)
		}
		opts = append(opts, huh.NewOption(label, s.Name))
	}
	opts = append(opts,
		huh.NewOption("> Review and apply", choiceReview),
		huh.NewOption("x Cancel", choiceCancel),
	)
	return opts
}

func summaryLine(assigned, selected map[string]bool) string {
	add, remove := client.DiffPrivileges(setToSorted(assigned), setToSorted(selected))
	return fmt.Sprintf("Select a service (/ to filter). %d selected, pending +%d -%d.",
		countTrue(selected), len(add), len(remove))
}

// filterServices keeps only services (and their privileges) that have at
// least one assigned privilege.
func filterServices(services []client.PrivilegeService, assigned map[string]bool) []client.PrivilegeService {
	var out []client.PrivilegeService
	for _, s := range services {
		keep := false
		for _, o := range s.Objects {
			for _, p := range o.Actions {
				if assigned[p.Name] {
					keep = true
				}
			}
		}
		for _, p := range s.Other {
			if assigned[p.Name] {
				keep = true
			}
		}
		if keep {
			out = append(out, s)
		}
	}
	return out
}

func runForm(f *huh.Form) error {
	err := f.WithOutput(os.Stderr).Run()
	if errors.Is(err, huh.ErrUserAborted) {
		return ErrCanceled
	}
	return err
}

func toSet(names []string) map[string]bool {
	s := make(map[string]bool, len(names))
	for _, n := range names {
		s[n] = true
	}
	return s
}

func setToSorted(s map[string]bool) []string {
	out := make([]string, 0, len(s))
	for n, ok := range s {
		if ok {
			out = append(out, n)
		}
	}
	sort.Strings(out)
	return out
}

func countTrue(s map[string]bool) int {
	n := 0
	for _, ok := range s {
		if ok {
			n++
		}
	}
	return n
}
