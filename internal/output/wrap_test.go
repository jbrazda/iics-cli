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

func TestWrapCellANSI(t *testing.T) {
	// An ANSI-colored word whose visible length fits the column must not be
	// split mid-escape-sequence just because its raw byte length exceeds
	// width (regression: "found" rendered as "\x1b[1;32mfound\x1b[0m" was
	// hard-split into "foun" / "d").
	colored := "\x1b[1;32mfound\x1b[0m"
	if got := WrapCell(colored, 12); got != colored {
		t.Errorf("WrapCell corrupted a short colored word: got %q, want %q", got, colored)
	}
	if visibleLen(colored) != len("found") {
		t.Fatalf("test fixture invariant broken: visibleLen(%q) = %d", colored, visibleLen(colored))
	}

	// A colored word whose visible length genuinely exceeds width still
	// hard-splits, without panicking or leaving stray escape bytes; styling
	// on the overflow line is dropped, matching TruncateCell's behavior.
	longColored := "\x1b[1;32mexceedinglylongstatus\x1b[0m"
	got := WrapCell(longColored, 8)
	if strings.Contains(got, "\x1b") {
		t.Errorf("hard-split of colored word left stray escape bytes: %q", got)
	}
	if !strings.Contains(got, "exceedin") {
		t.Errorf("hard-split of colored word = %q", got)
	}
}

func TestTruncateCell(t *testing.T) {
	if got := TruncateCell("hello world", 8); got != "hello w…" {
		t.Errorf("TruncateCell = %q", got)
	}
	if got := TruncateCell("short", 10); got != "short" {
		t.Errorf("TruncateCell should pass through short content: %q", got)
	}
	if got := TruncateCell("short", 0); got != "short" {
		t.Errorf("width 0 should pass through")
	}
	if got := TruncateCell("hello", 1); got != "…" {
		t.Errorf("TruncateCell width 1 = %q", got)
	}
}

func TestTruncateCellLeft(t *testing.T) {
	if got := TruncateCellLeft("/a/b/c/final_report.csv", 12); got != "…_report.csv" {
		t.Errorf("TruncateCellLeft = %q", got)
	}
	if got := TruncateCellLeft("short", 10); got != "short" {
		t.Errorf("TruncateCellLeft should pass through short content: %q", got)
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
