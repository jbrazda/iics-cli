package output

import (
	"bytes"
	"strings"
	"testing"
)

var tableTestColumns = []Column{
	{Header: "ID", Field: "id", Width: 4},
	{Header: "NAME", Field: "name", Width: 8},
	{Header: "STATUS", Field: "status", Width: 6},
}

var tableTestData = []map[string]interface{}{
	{"id": "a1", "name": "dev-org", "status": "active"},
	{"id": "a2", "name": "prod-org", "status": "active"},
	{"id": "a3", "name": "qa-org", "status": "paused"},
}

var tableTestDataOne = []map[string]interface{}{
	{"id": "a1", "name": "dev-org", "status": "active"},
}

// TestMarkdownTheme verifies the markdown theme produces GFM-format output with
// pipe delimiters, a separator row, and an HTML comment row count footer.
func TestMarkdownTheme(t *testing.T) {
	var buf bytes.Buffer
	f := New(FormatTable, &buf, TableStyle{Theme: "markdown"})
	if err := f.Format(tableTestData, tableTestColumns); err != nil {
		t.Fatalf("Format() error: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "| ID") {
		t.Errorf("markdown output should start rows with '|', got:\n%s", out)
	}
	if !strings.Contains(out, "| ---") {
		t.Errorf("markdown output should contain separator row '| ---', got:\n%s", out)
	}
	if !strings.Contains(out, "<!-- 3 rows -->") {
		t.Errorf("markdown output should contain '<!-- 3 rows -->', got:\n%s", out)
	}
	if strings.Contains(out, "\x1b[") {
		t.Errorf("markdown output should not contain ANSI codes, got:\n%s", out)
	}
}

// TestMarkdownThemeNonTTY verifies that effectiveTheme returns "markdown" even
// for a non-TTY writer (bytes.Buffer), bypassing the TTY downgrade to "plain".
func TestMarkdownThemeNonTTY(t *testing.T) {
	var buf bytes.Buffer
	got := effectiveTheme(&buf, TableStyle{Theme: "markdown"})
	if got != "markdown" {
		t.Errorf("effectiveTheme() = %q, want %q", got, "markdown")
	}
}

// TestGHTheme verifies the gh theme produces plain space-padded output with no
// pipe delimiters, no separator row, no ANSI codes, and a plain row count footer.
func TestGHTheme(t *testing.T) {
	var buf bytes.Buffer
	f := New(FormatTable, &buf, TableStyle{Theme: "gh"})
	if err := f.Format(tableTestData, tableTestColumns); err != nil {
		t.Fatalf("Format() error: %v", err)
	}
	out := buf.String()

	if strings.Contains(out, "|") {
		t.Errorf("gh output should not contain pipe chars, got:\n%s", out)
	}
	if strings.Contains(out, "---") {
		t.Errorf("gh output should not contain separator row, got:\n%s", out)
	}
	if strings.Contains(out, "\x1b[") {
		t.Errorf("gh output should not contain ANSI codes, got:\n%s", out)
	}
	if !strings.Contains(out, "3 rows") {
		t.Errorf("gh output should contain '3 rows' footer, got:\n%s", out)
	}
	if !strings.Contains(out, "dev-org") {
		t.Errorf("gh output should contain cell data, got:\n%s", out)
	}
}

// TestGHThemeNonTTY verifies that effectiveTheme returns "gh" even for a
// non-TTY writer, bypassing the TTY downgrade to "plain".
func TestGHThemeNonTTY(t *testing.T) {
	var buf bytes.Buffer
	got := effectiveTheme(&buf, TableStyle{Theme: "gh"})
	if got != "gh" {
		t.Errorf("effectiveTheme() = %q, want %q", got, "gh")
	}
}

// TestRowFooterPlural verifies that 3 rows appends "3 rows" on a separate line.
func TestRowFooterPlural(t *testing.T) {
	var buf bytes.Buffer
	f := New(FormatTable, &buf, TableStyle{Theme: "plain"})
	if err := f.Format(tableTestData, tableTestColumns); err != nil {
		t.Fatalf("Format() error: %v", err)
	}
	out := buf.String()

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	last := lines[len(lines)-1]
	if last != "3 rows" {
		t.Errorf("last line = %q, want %q", last, "3 rows")
	}
}

// TestRowFooterSingular verifies that 1 row appends "1 row" (singular).
func TestRowFooterSingular(t *testing.T) {
	var buf bytes.Buffer
	f := New(FormatTable, &buf, TableStyle{Theme: "plain"})
	if err := f.Format(tableTestDataOne, tableTestColumns); err != nil {
		t.Fatalf("Format() error: %v", err)
	}
	out := buf.String()

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	last := lines[len(lines)-1]
	if last != "1 row" {
		t.Errorf("last line = %q, want %q", last, "1 row")
	}
}

// TestRowFooterMarkdown verifies that markdown theme appends "<!-- 3 rows -->" footer.
func TestRowFooterMarkdown(t *testing.T) {
	var buf bytes.Buffer
	f := New(FormatTable, &buf, TableStyle{Theme: "markdown"})
	if err := f.Format(tableTestData, tableTestColumns); err != nil {
		t.Fatalf("Format() error: %v", err)
	}
	out := buf.String()

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	last := lines[len(lines)-1]
	if last != "<!-- 3 rows -->" {
		t.Errorf("last line = %q, want %q", last, "<!-- 3 rows -->")
	}
}

// TestMarkdownFooterSingular verifies that 1 row in markdown appends "<!-- 1 row -->".
func TestMarkdownFooterSingular(t *testing.T) {
	var buf bytes.Buffer
	f := New(FormatTable, &buf, TableStyle{Theme: "markdown"})
	if err := f.Format(tableTestDataOne, tableTestColumns); err != nil {
		t.Fatalf("Format() error: %v", err)
	}
	out := buf.String()

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	last := lines[len(lines)-1]
	if last != "<!-- 1 row -->" {
		t.Errorf("last line = %q, want %q", last, "<!-- 1 row -->")
	}
}

// TestCompactGap verifies the compact theme uses 1-space column gaps.
// We write two adjacent columns and check that exactly one space separates them
// (no ANSI codes between the padded cell value and the next column).
func TestCompactGap(t *testing.T) {
	var buf bytes.Buffer
	// Use plain equivalent: compact is colorless on non-TTY, but effectiveTheme
	// would downgrade to plain. Force by using a real compact render via renderCompact.
	cols := []Column{
		{Header: "A", Field: "a", Width: 1},
		{Header: "B", Field: "b", Width: 1},
	}
	data := []map[string]interface{}{{"a": "x", "b": "y"}}
	widths := computeColWidths(data, cols)
	renderCompact(&buf, data, cols, widths, TableStyle{})
	out := buf.String()
	// Strip ANSI from header line
	stripped := ansiEscape.ReplaceAllString(out, "")
	lines := strings.Split(strings.TrimRight(stripped, "\n"), "\n")
	// Data row (second line) should be "x y" with exactly one space
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got:\n%s", out)
	}
	dataLine := lines[1]
	if dataLine != "x y" {
		t.Errorf("compact data row = %q, want %q (1-space gap)", dataLine, "x y")
	}
}

// TestHeaderColorPropagation verifies that TableStyle.HeaderColor is applied
// without panicking and that the header text still appears in output.
func TestHeaderColorPropagation(t *testing.T) {
	var buf bytes.Buffer
	// Use renderDefault directly (non-TTY would downgrade to plain otherwise)
	cols := []Column{{Header: "NAME", Field: "name", Width: 4}}
	data := []map[string]interface{}{{"name": "foo"}}
	widths := computeColWidths(data, cols)
	// Should not panic with color "244"
	renderDefault(&buf, data, cols, widths, TableStyle{HeaderColor: "244"})
	out := buf.String()
	if !strings.Contains(out, "NAME") {
		t.Errorf("output should contain header text 'NAME', got:\n%s", out)
	}
}
