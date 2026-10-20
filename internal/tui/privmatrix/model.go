// Package privmatrix adapts one service's privileges to the grid model:
// objects are rows, actions (view, create, update, delete, execute, change
// permission) are columns, and privileges that do not follow the
// "<action>.<object>" pattern are listed under "-- Other --".
package privmatrix

import (
	"fmt"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/tui/grid"
)

// Mode restricts which cells can be toggled.
type Mode int

const (
	// ModeEdit allows adding and removing privileges.
	ModeEdit Mode = iota
	// ModeAdd only allows adding; assigned privileges are locked.
	ModeAdd
	// ModeRemove only shows assigned privileges and only allows removing them.
	ModeRemove
)

// Options configures the model.
type Options struct {
	Title   string
	Mode    Mode
	NoColor bool
}

// Model is the grid model used for a service's privileges.
type Model = grid.Model

var columnHeaders = []string{"VIEW", "CREATE", "UPDATE", "DELETE", "EXEC", "PERM"}

// New creates a model for one service. assigned holds the privilege names
// the role has now; selected holds the desired state and is updated in place.
func New(svc client.PrivilegeService, assigned, selected map[string]bool, opts Options) *Model {
	visible := func(p client.Privilege) bool {
		return opts.Mode != ModeRemove || assigned[p.Name]
	}
	locked := func(p client.Privilege) bool {
		switch opts.Mode {
		case ModeAdd:
			return assigned[p.Name]
		case ModeRemove:
			return !assigned[p.Name]
		}
		return false
	}
	cellFor := func(p client.Privilege) grid.Cell {
		return grid.Cell{Key: p.Name, Locked: locked(p), Detail: detail(p)}
	}

	var rows []grid.Row
	for _, obj := range svc.Objects {
		cells := make([]grid.Cell, len(client.PrivilegeActions))
		hasCell := false
		for i, a := range client.PrivilegeActions {
			if p, ok := obj.Actions[a]; ok && visible(p) {
				cells[i] = cellFor(p)
				hasCell = true
			}
		}
		if hasCell {
			rows = append(rows, grid.Row{Label: obj.Key, Cells: cells})
		}
	}
	var others []grid.Row
	for _, p := range svc.Other {
		if visible(p) {
			others = append(others, grid.Row{Label: p.Name, Cells: []grid.Cell{cellFor(p)}})
		}
	}
	if len(others) > 0 {
		rows = append(rows, grid.Row{Kind: grid.RowHeader, Label: "-- Other --"})
		rows = append(rows, others...)
	}

	return grid.New(rows, assigned, selected, grid.Options{
		Title:     fmt.Sprintf("%s > %s (%d)", opts.Title, svc.Name, svc.Count()),
		Columns:   columnHeaders,
		KeyHeader: "OBJECT",
		NoColor:   opts.NoColor,
	})
}

func detail(p client.Privilege) string {
	d := p.Name
	if p.Description != "" {
		d += " - " + p.Description
	}
	if p.Status != "" {
		d += " (" + p.Status + ")"
	}
	return d
}
