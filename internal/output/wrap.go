package output

import "strings"

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
	flush := func() {
		lines = append(lines, cur.String())
		cur.Reset()
	}
	for _, word := range strings.Fields(s) {
		for len(word) > width {
			if cur.Len() > 0 {
				flush()
			}
			lines = append(lines, word[:width])
			word = word[width:]
		}
		switch {
		case cur.Len() == 0:
			cur.WriteString(word)
		case cur.Len()+1+len(word) <= width:
			cur.WriteByte(' ')
			cur.WriteString(word)
		default:
			flush()
			cur.WriteString(word)
		}
	}
	if cur.Len() > 0 || len(lines) == 0 {
		flush()
	}
	return lines
}
