package output

import (
	"strings"
	"testing"
)

func idCol() Column    { return Column{Header: "ID", Field: "id", Width: 24} }
func nameCol() Column  { return Column{Header: "NAME", Field: "name", Width: 30} }
func stateCol() Column { return Column{Header: "STATE", Field: "state", Width: 10} }
func descCol() Column  { return Column{Header: "DESCRIPTION", Field: "description"} }

func wideRows() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"id": "abcd1234abcd1234abcd1234", "name": "my-agent-one", "state": "RUNNING",
			"description": "This is a fairly long free-text description field that would normally blow out the table width on a narrow terminal",
		},
	}
}

func TestResolveDefaults(t *testing.T) {
	cols := []Column{idCol(), nameCol(), stateCol(), descCol()}
	resolved := resolveDefaults(cols)

	if resolved[0].Priority != 5 || resolved[0].Shrink != ShrinkNever {
		t.Errorf("id column: got priority=%d shrink=%v, want 5/ShrinkNever", resolved[0].Priority, resolved[0].Shrink)
	}
	// name (index 1) isn't the first column and doesn't match the id/width
	// signals, so it falls through to the general P3 default - only a
	// literal first column gets the P1 "essential" rule (D4).
	if resolved[1].Priority != 3 || resolved[1].Shrink != ShrinkTruncate {
		t.Errorf("name column: got priority=%d shrink=%v, want 3/ShrinkTruncate", resolved[1].Priority, resolved[1].Shrink)
	}
	if resolved[2].Priority != 2 || resolved[2].Shrink != ShrinkNever {
		t.Errorf("state column (width 10): got priority=%d shrink=%v, want 2/ShrinkNever", resolved[2].Priority, resolved[2].Shrink)
	}
	if resolved[3].Priority != 4 || resolved[3].Shrink != ShrinkWrap {
		t.Errorf("description column (no width): got priority=%d shrink=%v, want 4/ShrinkWrap", resolved[3].Priority, resolved[3].Shrink)
	}

	// When the literal first column isn't an ID/short/unbounded column, it
	// gets the P1 "essential" default.
	firstIsName := resolveDefaults([]Column{nameCol(), stateCol()})
	if firstIsName[0].Priority != 1 {
		t.Errorf("first column (name, no id/short/width-0 signal): got priority=%d, want 1", firstIsName[0].Priority)
	}

	// Explicit Priority is left untouched, including its Shrink default.
	explicit := []Column{{Header: "X", Field: "x", Priority: 3}}
	if got := resolveDefaults(explicit)[0]; got.Priority != 3 || got.Shrink != ShrinkTruncate {
		t.Errorf("explicit priority column: got priority=%d shrink=%v, want 3/ShrinkTruncate", got.Priority, got.Shrink)
	}
}

func TestPlanColumnsDropsIDFirst(t *testing.T) {
	cols := []Column{idCol(), nameCol(), stateCol()}
	rows := []map[string]interface{}{{"id": "abcd1234abcd1234abcd1234", "name": "my-agent-one", "state": "RUNNING"}}

	kept, _, dropped, vertical := planColumns(rows, cols, 30, "compact")
	if vertical {
		t.Fatalf("expected non-vertical layout, natural minus id column fits in 30")
	}
	if len(dropped) != 1 || dropped[0] != "ID" {
		t.Fatalf("expected ID to be dropped first, got dropped=%v", dropped)
	}
	for _, c := range kept {
		if c.Header == "ID" {
			t.Errorf("ID column should have been dropped from kept: %v", kept)
		}
	}
}

func TestPlanColumnsFallsBackToVertical(t *testing.T) {
	cols := []Column{idCol(), nameCol(), stateCol(), descCol()}
	rows := wideRows()

	// A terminal too narrow even for the essential (priority-1) NAME column
	// alongside anything else forces the vertical fallback.
	_, _, _, vertical := planColumns(rows, cols, 15, "compact")
	if !vertical {
		t.Fatalf("expected vertical fallback for a very narrow terminal")
	}
}

func TestPlanColumnsNoAdaptationWhenItFits(t *testing.T) {
	cols := []Column{idCol(), nameCol(), stateCol()}
	rows := []map[string]interface{}{{"id": "abcd1234abcd1234abcd1234", "name": "n", "state": "RUNNING"}}

	kept, _, dropped, vertical := planColumns(rows, cols, 200, "default")
	if vertical || len(dropped) != 0 || len(kept) != len(cols) {
		t.Fatalf("wide terminal should need no adaptation: kept=%v dropped=%v vertical=%v", kept, dropped, vertical)
	}
}

func TestReorderWrappedLast(t *testing.T) {
	cols := []Column{
		{Header: "DESC", Shrink: ShrinkWrap},
		{Header: "NAME", Shrink: ShrinkTruncate},
		{Header: "MSG", Shrink: ShrinkWrap},
	}
	widths := []int{10, 20, 30}
	kept, w := reorderWrappedLast(cols, widths)

	if kept[0].Header != "NAME" {
		t.Fatalf("non-wrap column should lead: got %v", headersOf(kept))
	}
	// Both wrap columns trail, widest (MSG, width 30) before DESC (width 10).
	if kept[1].Header != "MSG" || kept[2].Header != "DESC" {
		t.Fatalf("wrap columns should trail widest-first: got %v", headersOf(kept))
	}
	if len(w) != 3 || w[0] != 20 || w[1] != 30 || w[2] != 10 {
		t.Fatalf("widths should follow the same reordering: %v", w)
	}
}

func headersOf(cols []Column) []string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = c.Header
	}
	return out
}

func TestApplyShrinkTruncatesAndWraps(t *testing.T) {
	cols := []Column{
		{Header: "NAME", Field: "name", Shrink: ShrinkTruncate},
		{Header: "DESC", Field: "description", Shrink: ShrinkWrap},
	}
	widths := []int{8, 10}
	shrunk := applyShrink(cols, widths)

	row := map[string]interface{}{"name": "a-very-long-agent-name", "description": "one two three four five"}
	name := extractField(row, shrunk[0])
	if !strings.HasSuffix(name, "…") || utf8len(name) > 8 {
		t.Errorf("name should be right-truncated to 8: %q", name)
	}
	desc := extractField(row, shrunk[1])
	if !strings.Contains(desc, "\n") {
		t.Errorf("description should be wrapped onto multiple lines: %q", desc)
	}
}

func utf8len(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}

func TestRenderVertical(t *testing.T) {
	var sb strings.Builder
	cols := []Column{nameCol(), stateCol()}
	rows := []map[string]interface{}{
		{"name": "agent-one", "state": "RUNNING"},
		{"name": "agent-two", "state": "STOPPED"},
	}
	renderVertical(&sb, rows, cols)
	out := sb.String()
	if !strings.Contains(out, "NAME") || !strings.Contains(out, "agent-one") || !strings.Contains(out, "agent-two") {
		t.Fatalf("vertical output missing expected content:\n%s", out)
	}
	// A blank line separates the two record blocks.
	if !strings.Contains(out, "STOPPED\n\n") && !strings.Contains(out, "RUNNING\n\n") {
		t.Errorf("expected a blank line between records:\n%s", out)
	}
}
