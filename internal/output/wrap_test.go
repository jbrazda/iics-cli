package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestWrapCell(t *testing.T) {
	got := WrapCell("the quick brown fox jumps", 10)
	want := "the quick\nbrown fox\njumps"
	if got != want {
		t.Errorf("WrapCell = %q, want %q", got, want)
	}
	if WrapCell("short", 0) != "short" {
		t.Errorf("width 0 should pass through")
	}
	// long token is hard-split
	if got := WrapCell("abcdefghij", 4); got != "abcd\nefgh\nij" {
		t.Errorf("hard-split = %q", got)
	}
	// existing newlines preserved
	if got := WrapCell("a\nb", 10); got != "a\nb" {
		t.Errorf("newline preserved = %q", got)
	}
}

func TestTableMultiLineCell(t *testing.T) {
	var buf bytes.Buffer
	f := New(FormatTable, &buf, TableStyle{NoColor: true})
	rows := []KVRow{
		{Property: "value", Value: "line-one\nline-two"},
	}
	if err := f.Format(rows, KVCols); err != nil {
		t.Fatalf("Format: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "line-one") || !strings.Contains(out, "line-two") {
		t.Fatalf("multi-line cell not rendered:\n%s", out)
	}
	// both physical lines are present as separate table rows
	if strings.Count(out, "|") < 8 {
		t.Errorf("expected stacked rows, got:\n%s", out)
	}
}
