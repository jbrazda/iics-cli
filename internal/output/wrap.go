package output

import (
	"strings"
	"unicode/utf8"
)

// WrapCell hard-wraps s onto lines of at most width visible columns, breaking on
// spaces where possible and hard-splitting tokens longer than width. Newlines
// already present in s are preserved as line breaks. The result is suitable for
// a table cell rendered by the bordered themes, which display embedded newlines
// as stacked physical lines.
func WrapCell(s string, width int) string {
	if width <= 0 {
		return s
	}
	var lines []string
	for _, para := range strings.Split(s, "\n") {
		lines = append(lines, wrapParagraph(para, width)...)
	}
	return strings.Join(lines, "\n")
}

func wrapParagraph(s string, width int) []string {
	if s == "" {
		return []string{""}
	}
	var lines []string
	var cur strings.Builder
	curLen := 0
	flush := func() {
		lines = append(lines, cur.String())
		cur.Reset()
		curLen = 0
	}
	for _, word := range strings.Fields(s) {
		wordLen := visibleLen(word)
		for wordLen > width {
			if cur.Len() > 0 {
				flush()
			}
			// A word wider than the column can't be split without cutting
			// through its ANSI escape bytes, so fall back to its plain text
			// for the overflow line (styling is dropped, same trade-off
			// TruncateCell makes on a truncated line).
			r := []rune(stripANSIText(word))
			lines = append(lines, string(r[:width]))
			word = string(r[width:])
			wordLen = len(r) - width
		}
		switch {
		case cur.Len() == 0:
			cur.WriteString(word)
			curLen = wordLen
		case curLen+1+wordLen <= width:
			cur.WriteByte(' ')
			cur.WriteString(word)
			curLen += 1 + wordLen
		default:
			flush()
			cur.WriteString(word)
			curLen = wordLen
		}
	}
	if cur.Len() > 0 || len(lines) == 0 {
		flush()
	}
	return lines
}

const ellipsis = "…"

// TruncateCell right-truncates s to at most width visible columns, appending
// an ellipsis when content is cut. Any embedded newlines are truncated line
// by line. ANSI styling on a truncated line is dropped along with the cut
// text; width <= 0 disables truncation.
func TruncateCell(s string, width int) string {
	return truncateCell(s, width, false)
}

// TruncateCellLeft left-truncates s to at most width visible columns,
// keeping the tail - useful for paths and URLs where the end (filename,
// last segment) matters more than the start.
func TruncateCellLeft(s string, width int) string {
	return truncateCell(s, width, true)
}

func truncateCell(s string, width int, fromLeft bool) string {
	if width <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = truncateLine(line, width, fromLeft)
	}
	return strings.Join(lines, "\n")
}

func truncateLine(s string, width int, fromLeft bool) string {
	plain := stripANSIText(s)
	if utf8.RuneCountInString(plain) <= width {
		return s
	}
	if width <= utf8.RuneCountInString(ellipsis) {
		return ellipsis
	}
	r := []rune(plain)
	keep := width - utf8.RuneCountInString(ellipsis)
	if fromLeft {
		return ellipsis + string(r[len(r)-keep:])
	}
	return string(r[:keep]) + ellipsis
}
