package grid

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func runes(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }

func testRows() []Row {
	return []Row{
		{Label: "Developer", Tag: "group", Cells: []Cell{{Key: "dev|read"}, {Key: "dev|update"}, {Key: "dev|delete", Locked: true}}},
		{Label: "Ops", Cells: []Cell{{Key: "ops|read"}, {}, {Key: "ops|delete"}}},
		{Kind: RowHeader, Label: "-- Other --"},
		{Label: "misc", Match: "extra words", Cells: []Cell{{Key: "misc|x", Detail: "misc detail"}}},
	}
}

func TestClearKeyAndActions(t *testing.T) {
	original := map[string]bool{"dev|read": true, "dev|update": true, "dev|delete": true}
	selected := map[string]bool{"dev|read": true, "dev|update": true, "dev|delete": true}
	m := New(testRows(), original, selected, Options{
		Columns: []string{"R", "U", "D"}, NoColor: true, ClearKey: "d",
		Actions: map[string]string{"n": "add"}, ActionOrder: []string{"n"},
	})

	m.Update(runes("d"))
	if selected["dev|read"] || selected["dev|update"] {
		t.Errorf("clear should unset unlocked cells: %v", selected)
	}
	if !selected["dev|delete"] {
		t.Error("clear must keep locked cells")
	}
	if m.Done() {
		t.Error("clear must not end the grid")
	}
	view := m.View()
	if !strings.Contains(view, "Developer group") || !strings.Contains(view, "[-]") || !strings.Contains(view, "d clear row") || !strings.Contains(view, "n add") {
		t.Errorf("unexpected view:\n%s", view)
	}

	_, cmd := m.Update(runes("n"))
	if !m.Done() || m.Action() != "n" || cmd == nil {
		t.Errorf("action key should end the grid with action n, got done=%v action=%q", m.Done(), m.Action())
	}
}

func TestMissingCellsAndColumnToggle(t *testing.T) {
	selected := map[string]bool{}
	m := New(testRows(), map[string]bool{}, selected, Options{Columns: []string{"R", "U", "D"}, NoColor: true})

	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m.Update(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
	if len(selected) != 0 {
		t.Errorf("a missing cell must not toggle: %v", selected)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m.Update(runes("c"))
	if !selected["dev|read"] || !selected["ops|read"] || selected["misc|x"] {
		t.Errorf("column toggle should cover full rows only: %v", selected)
	}
}

func TestFilterMatchesLabelMatchAndDetail(t *testing.T) {
	m := New(testRows(), map[string]bool{}, map[string]bool{}, Options{Columns: []string{"R", "U", "D"}, NoColor: true})
	for _, q := range []string{"extra", "misc detail"} {
		m.SetFilter(q)
		rows := m.VisibleRows()
		if len(rows) != 2 || rows[0].Kind != RowHeader || rows[1].Label != "misc" {
			t.Errorf("filter %q: got %+v", q, rows)
		}
	}
	m.SetFilter("ops")
	if rows := m.VisibleRows(); len(rows) != 1 || rows[0].Label != "Ops" {
		t.Errorf("filter ops: got %+v", rows)
	}
}
