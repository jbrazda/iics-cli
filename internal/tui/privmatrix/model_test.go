package privmatrix

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jbrazda/iics-cli/internal/client"
)

func testService() client.PrivilegeService {
	return client.GroupPrivileges([]client.Privilege{
		{ID: "1", Name: "view.data.transfer.task", Service: "DI"},
		{ID: "2", Name: "create.data.transfer.task", Service: "DI"},
		{ID: "3", Name: "delete.data.transfer.task", Service: "DI"},
		{ID: "4", Name: "view.mapping", Service: "DI"},
		{ID: "5", Name: "update.mapping", Service: "DI"},
		{ID: "6", Name: "edc.iics.discovery", Service: "DI", Description: "EDC discovery"},
	})[0]
}

func key(s string) tea.KeyMsg {
	switch s {
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "space":
		return tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func press(m *Model, keys ...string) {
	for _, k := range keys {
		m.Update(key(k))
	}
}

func TestToggleCell(t *testing.T) {
	assigned := map[string]bool{"view.data.transfer.task": true}
	selected := map[string]bool{"view.data.transfer.task": true}
	m := New(testService(), assigned, selected, Options{Title: "R", NoColor: true})

	// Row 0 is data.transfer.task; column 1 is create.
	press(m, "right", "space")
	if !selected["create.data.transfer.task"] {
		t.Error("expected create.data.transfer.task selected")
	}
	// Column 0 (view) is assigned; toggling marks it for removal.
	press(m, "left", "space")
	if selected["view.data.transfer.task"] {
		t.Error("expected view.data.transfer.task deselected")
	}
	view := m.View()
	if !strings.Contains(view, "[+]") || !strings.Contains(view, "[-]") {
		t.Errorf("view should show pending add and remove marks:\n%s", view)
	}
}

func TestRowAndColumnToggle(t *testing.T) {
	selected := map[string]bool{}
	m := New(testService(), map[string]bool{}, selected, Options{NoColor: true})

	press(m, "a")
	for _, n := range []string{"view.data.transfer.task", "create.data.transfer.task", "delete.data.transfer.task"} {
		if !selected[n] {
			t.Errorf("row toggle should select %s", n)
		}
	}
	press(m, "a")
	if selected["view.data.transfer.task"] {
		t.Error("second row toggle should clear the row")
	}

	press(m, "c") // column 0: view on both objects
	if !selected["view.data.transfer.task"] || !selected["view.mapping"] {
		t.Errorf("column toggle should select all views: %v", selected)
	}
}

func TestOtherRowsSkipHeader(t *testing.T) {
	selected := map[string]bool{}
	m := New(testService(), map[string]bool{}, selected, Options{NoColor: true})

	press(m, "right", "right", "down", "down")
	if m.rows[m.cursorRow].kind != rowOther {
		t.Fatalf("cursor should skip the header and land on an Other row, got kind %v", m.rows[m.cursorRow].kind)
	}
	if m.cursorCol != 0 {
		t.Errorf("Other rows have one column, cursorCol=%d", m.cursorCol)
	}
	press(m, "space")
	if !selected["edc.iics.discovery"] {
		t.Error("expected edc.iics.discovery selected")
	}
}

func TestFilter(t *testing.T) {
	m := New(testService(), map[string]bool{}, map[string]bool{}, Options{NoColor: true})
	press(m, "/", "m", "a", "p", "enter")
	if len(m.rows) != 1 || m.rows[0].object.Key != "mapping" {
		t.Fatalf("filter should leave only mapping, got %d rows", len(m.rows))
	}
	press(m, "/", "e", "d", "c")
	if len(m.rows) != 0 {
		// "mapedc" matches nothing
		t.Fatalf("expected no rows, got %d", len(m.rows))
	}
	press(m, "esc", "ctrl+u")
	m.filter.SetValue("discovery")
	m.rebuildRows()
	if len(m.rows) != 2 || m.rows[1].other.Name != "edc.iics.discovery" {
		t.Errorf("description filter should match the Other row, got %+v", m.rows)
	}
}

func TestAddModeLocksAssigned(t *testing.T) {
	assigned := map[string]bool{"view.data.transfer.task": true}
	selected := map[string]bool{"view.data.transfer.task": true}
	m := New(testService(), assigned, selected, Options{Mode: ModeAdd, NoColor: true})
	press(m, "space")
	if !selected["view.data.transfer.task"] {
		t.Error("assigned privilege must stay selected in add mode")
	}
	press(m, "a")
	if !selected["create.data.transfer.task"] || !selected["view.data.transfer.task"] {
		t.Errorf("row toggle in add mode should add unassigned only: %v", selected)
	}
}

func TestRemoveModeShowsOnlyAssigned(t *testing.T) {
	assigned := map[string]bool{"update.mapping": true}
	selected := map[string]bool{"update.mapping": true}
	m := New(testService(), assigned, selected, Options{Mode: ModeRemove, NoColor: true})
	if len(m.rows) != 1 || m.rows[0].object.Key != "mapping" {
		t.Fatalf("remove mode should show only objects with assigned privileges, got %d rows", len(m.rows))
	}
	press(m, "a")
	if selected["update.mapping"] {
		t.Error("row toggle in remove mode should deselect the assigned privilege")
	}
	if selected["view.mapping"] {
		t.Error("unassigned privilege must not be selectable in remove mode")
	}
}

func TestQuitKeys(t *testing.T) {
	m := New(testService(), map[string]bool{}, map[string]bool{}, Options{NoColor: true})
	_, cmd := m.Update(key("enter"))
	if !m.Done() || cmd == nil {
		t.Error("enter should finish the grid")
	}
}
