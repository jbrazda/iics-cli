package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
)

// visibleLen returns the number of visible runes in s, ignoring ANSI escape sequences.
func visibleLen(s string) int {
	return utf8.RuneCountInString(stripANSIText(s))
}

const (
	cellPad       = 1 // spaces of padding on each side of a bordered cell
	colGap        = 2 // spaces between columns in borderless themes (minimal, gh)
	compactColGap = 1 // spaces between columns in compact theme
)

type tableFormatter struct {
	w     io.Writer
	style TableStyle
}

func newTableFormatter(w io.Writer, style TableStyle) *tableFormatter {
	return &tableFormatter{w: w, style: style}
}

func (f *tableFormatter) Format(data interface{}, columns []Column) error {
	if len(columns) == 0 {
		enc := json.NewEncoder(f.w)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	}

	rows, err := extractRows(data)
	if err != nil {
		return err
	}

	if len(rows) == 0 {
		_, _ = fmt.Fprintln(f.w, "No results found.")
		return nil
	}

	theme := effectiveTheme(f.w, f.style)
	widths := computeColWidths(rows, columns)
	renderTable(f.w, rows, columns, widths, theme, f.style)

	// Row count footer
	n := len(rows)
	if theme == "markdown" {
		if n == 1 {
			_, _ = fmt.Fprintln(f.w, "<!-- 1 row -->")
		} else {
			_, _ = fmt.Fprintf(f.w, "<!-- %d rows -->\n", n)
		}
	} else {
		if n == 1 {
			_, _ = fmt.Fprintln(f.w, "1 row")
		} else {
			_, _ = fmt.Fprintf(f.w, "%d rows\n", n)
		}
	}

	return nil
}

// effectiveTheme resolves the theme to use, downgrading to "plain" when
// no-color is set or the output writer is not a TTY.
// The "markdown" and "gh" themes are always colorless and bypass the TTY check.
func effectiveTheme(w io.Writer, style TableStyle) string {
	if style.Theme == "markdown" || style.Theme == "gh" {
		return style.Theme
	}
	if style.NoColor || !isTerminal(w) {
		return "plain"
	}
	if style.Theme == "" {
		return "default"
	}
	return style.Theme
}

func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
}

// computeColWidths returns the minimum display width for each column.
// Width = max(header_len, max_cell_len), at least Column.Width if set.
func computeColWidths(rows []map[string]interface{}, columns []Column) []int {
	widths := make([]int, len(columns))
	for i, col := range columns {
		w := utf8.RuneCountInString(col.Header)
		if col.Width > w {
			w = col.Width
		}
		widths[i] = w
	}
	for _, row := range rows {
		for i, col := range columns {
			l := visibleLen(extractField(row, col))
			if l > widths[i] {
				widths[i] = l
			}
		}
	}
	return widths
}

// padRight right-pads s to exactly width visible runes.
func padRight(s string, width int) string {
	l := visibleLen(s)
	if l >= width {
		return s
	}
	return s + strings.Repeat(" ", width-l)
}

// --- Border character sets ---

type borderSet struct {
	topLeft, topMid, topRight string
	midLeft, midMid, midRight string
	botLeft, botMid, botRight string
	hLine, vLine              string
}

var (
	unicodeBorders = borderSet{
		topLeft: "╭", topMid: "┬", topRight: "╮",
		midLeft: "├", midMid: "┼", midRight: "┤",
		botLeft: "╰", botMid: "┴", botRight: "╯",
		hLine: "─", vLine: "│",
	}
	asciiBorders = borderSet{
		topLeft: "+", topMid: "+", topRight: "+",
		midLeft: "+", midMid: "+", midRight: "+",
		botLeft: "+", botMid: "+", botRight: "+",
		hLine: "-", vLine: "|",
	}
)

// hSep builds a horizontal separator line for bordered themes.
func hSep(b borderSet, widths []int, left, mid, right string) string {
	var sb strings.Builder
	sb.WriteString(left)
	for i, w := range widths {
		sb.WriteString(strings.Repeat(b.hLine, w+2*cellPad))
		if i < len(widths)-1 {
			sb.WriteString(mid)
		}
	}
	sb.WriteString(right)
	return sb.String()
}

// borderedRow renders one row with vertical border separators.
// styleFn is applied to each cell's content (after padding to column width).
func borderedRow(b borderSet, cells []string, widths []int, styleFn func(string) string) string {
	var sb strings.Builder
	sb.WriteString(b.vLine)
	for i, cell := range cells {
		sb.WriteString(strings.Repeat(" ", cellPad))
		sb.WriteString(styleFn(padRight(cell, widths[i])))
		sb.WriteString(strings.Repeat(" ", cellPad))
		sb.WriteString(b.vLine)
	}
	return sb.String()
}

// makeHeaderStyle returns a lipgloss.Style for header cells.
// color is a lipgloss color string. If empty, only Bold is applied (no color).
func makeHeaderStyle(color string) lipgloss.Style {
	s := lipgloss.NewStyle().Bold(true)
	if color != "" {
		s = s.Foreground(lipgloss.Color(color))
	}
	return s
}

func noStyle(s string) string { return s }

func headerStrings(columns []Column) []string {
	h := make([]string, len(columns))
	for i, col := range columns {
		h[i] = col.Header
	}
	return h
}

func dataCells(row map[string]interface{}, columns []Column) []string {
	cells := make([]string, len(columns))
	for i, col := range columns {
		cells[i] = extractField(row, col)
	}
	return cells
}

func sanitizedDataCells(row map[string]interface{}, columns []Column) []string {
	cells := make([]string, len(columns))
	for i, col := range columns {
		cells[i] = stripANSIText(extractField(row, col))
	}
	return cells
}

func renderTable(w io.Writer, rows []map[string]interface{}, columns []Column, widths []int, theme string, style TableStyle) {
	switch theme {
	case "minimal":
		renderMinimal(w, rows, columns, widths, style)
	case "compact":
		renderCompact(w, rows, columns, widths, style)
	case "plain":
		renderPlain(w, rows, columns, widths)
	case "markdown":
		renderMarkdown(w, rows, columns, widths)
	case "gh":
		renderGH(w, rows, columns, widths)
	default:
		renderDefault(w, rows, columns, widths, style)
	}
}

// renderDefault renders with unicode rounded borders and configurable bold headers.
// Defaults to cyan ("6") when no HeaderColor is set.
func renderDefault(w io.Writer, rows []map[string]interface{}, columns []Column, widths []int, style TableStyle) {
	b := unicodeBorders
	color := style.HeaderColor
	if color == "" {
		color = "6" // built-in default: cyan
	}
	hStyle := makeHeaderStyle(color)
	headerFn := func(s string) string { return hStyle.Render(s) }

	headers := headerStrings(columns)
	_, _ = fmt.Fprintln(w, hSep(b, widths, b.topLeft, b.topMid, b.topRight))
	_, _ = fmt.Fprintln(w, borderedRow(b, headers, widths, headerFn))
	_, _ = fmt.Fprintln(w, hSep(b, widths, b.midLeft, b.midMid, b.midRight))
	for _, row := range rows {
		_, _ = fmt.Fprintln(w, borderedRow(b, dataCells(row, columns), widths, noStyle))
	}
	_, _ = fmt.Fprintln(w, hSep(b, widths, b.botLeft, b.botMid, b.botRight))
}

// renderPlain renders with ASCII borders and no color. Used for non-TTY and --no-color.
func renderPlain(w io.Writer, rows []map[string]interface{}, columns []Column, widths []int) {
	b := asciiBorders
	headers := headerStrings(columns)
	_, _ = fmt.Fprintln(w, hSep(b, widths, b.topLeft, b.topMid, b.topRight))
	_, _ = fmt.Fprintln(w, borderedRow(b, headers, widths, noStyle))
	_, _ = fmt.Fprintln(w, hSep(b, widths, b.midLeft, b.midMid, b.midRight))
	for _, row := range rows {
		_, _ = fmt.Fprintln(w, borderedRow(b, sanitizedDataCells(row, columns), widths, noStyle))
	}
	_, _ = fmt.Fprintln(w, hSep(b, widths, b.botLeft, b.botMid, b.botRight))
}

// renderMinimal renders with no box borders, configurable bold headers, and a unicode underline separator.
// Defaults to cyan ("6") when no HeaderColor is set.
func renderMinimal(w io.Writer, rows []map[string]interface{}, columns []Column, widths []int, style TableStyle) {
	color := style.HeaderColor
	if color == "" {
		color = "6" // built-in default: cyan
	}
	hStyle := makeHeaderStyle(color)
	headerFn := func(s string) string { return hStyle.Render(s) }

	headers := headerStrings(columns)

	var hdr strings.Builder
	for i, h := range headers {
		hdr.WriteString(headerFn(padRight(h, widths[i])))
		if i < len(widths)-1 {
			hdr.WriteString(strings.Repeat(" ", colGap))
		}
	}
	_, _ = fmt.Fprintln(w, hdr.String())

	var ul strings.Builder
	for i, wd := range widths {
		ul.WriteString(strings.Repeat("─", wd))
		if i < len(widths)-1 {
			ul.WriteString(strings.Repeat(" ", colGap))
		}
	}
	_, _ = fmt.Fprintln(w, ul.String())

	for _, row := range rows {
		cells := sanitizedDataCells(row, columns)
		var rb strings.Builder
		for i, cell := range cells {
			rb.WriteString(padRight(cell, widths[i]))
			if i < len(widths)-1 {
				rb.WriteString(strings.Repeat(" ", colGap))
			}
		}
		_, _ = fmt.Fprintln(w, rb.String())
	}
}

// renderCompact renders with no borders, gray bold headers (default), and 1-space column gaps.
// Defaults to gray ("244") when no HeaderColor is set.
func renderCompact(w io.Writer, rows []map[string]interface{}, columns []Column, widths []int, style TableStyle) {
	color := style.HeaderColor
	if color == "" {
		color = "244" // built-in default for compact: gray
	}
	hStyle := makeHeaderStyle(color)
	headerFn := func(s string) string { return hStyle.Render(s) }

	headers := headerStrings(columns)
	var hdr strings.Builder
	for i, h := range headers {
		hdr.WriteString(headerFn(padRight(h, widths[i])))
		if i < len(widths)-1 {
			hdr.WriteString(strings.Repeat(" ", compactColGap))
		}
	}
	_, _ = fmt.Fprintln(w, hdr.String())

	for _, row := range rows {
		cells := sanitizedDataCells(row, columns)
		var rb strings.Builder
		for i, cell := range cells {
			rb.WriteString(padRight(cell, widths[i]))
			if i < len(widths)-1 {
				rb.WriteString(strings.Repeat(" ", compactColGap))
			}
		}
		_, _ = fmt.Fprintln(w, rb.String())
	}
}

// renderMarkdown renders a GitHub-Flavored Markdown table.
// No ANSI color is used regardless of TTY or style settings.
// Column widths are padded for alignment in raw source.
func renderMarkdown(w io.Writer, rows []map[string]interface{}, columns []Column, widths []int) {
	headers := headerStrings(columns)

	// Header row
	var hdr strings.Builder
	hdr.WriteString("| ")
	for i, h := range headers {
		hdr.WriteString(padRight(h, widths[i]))
		if i < len(widths)-1 {
			hdr.WriteString(" | ")
		}
	}
	hdr.WriteString(" |")
	_, _ = fmt.Fprintln(w, hdr.String())

	// Separator row
	var sep strings.Builder
	sep.WriteString("| ")
	for i, wd := range widths {
		sep.WriteString(strings.Repeat("-", wd))
		if i < len(widths)-1 {
			sep.WriteString(" | ")
		}
	}
	sep.WriteString(" |")
	_, _ = fmt.Fprintln(w, sep.String())

	// Data rows
	for _, row := range rows {
		cells := sanitizedDataCells(row, columns)
		var rb strings.Builder
		rb.WriteString("| ")
		for i, cell := range cells {
			rb.WriteString(padRight(cell, widths[i]))
			if i < len(widths)-1 {
				rb.WriteString(" | ")
			}
		}
		rb.WriteString(" |")
		_, _ = fmt.Fprintln(w, rb.String())
	}
}

// renderGH renders a plain space-padded table with no borders or decorations,
// matching the GitHub CLI (gh) table output style.
// No ANSI color is used regardless of TTY or style settings.
func renderGH(w io.Writer, rows []map[string]interface{}, columns []Column, widths []int) {
	headers := headerStrings(columns)

	var hdr strings.Builder
	for i, h := range headers {
		hdr.WriteString(padRight(h, widths[i]))
		if i < len(widths)-1 {
			hdr.WriteString(strings.Repeat(" ", colGap))
		}
	}
	_, _ = fmt.Fprintln(w, hdr.String())

	for _, row := range rows {
		cells := sanitizedDataCells(row, columns)
		var rb strings.Builder
		for i, cell := range cells {
			rb.WriteString(padRight(cell, widths[i]))
			if i < len(widths)-1 {
				rb.WriteString(strings.Repeat(" ", colGap))
			}
		}
		_, _ = fmt.Fprintln(w, rb.String())
	}
}

// extractRows converts the data to a slice of maps for processing.
func extractRows(data interface{}) ([]map[string]interface{}, error) {
	// If data is already []map[string]interface{}, use directly
	if rows, ok := data.([]map[string]interface{}); ok {
		return rows, nil
	}

	// Marshal and unmarshal through JSON for a uniform representation
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshaling data: %w", err)
	}

	// Try as array first
	var rows []map[string]interface{}
	if err := json.Unmarshal(jsonData, &rows); err == nil {
		return rows, nil
	}

	// Try as single object
	var single map[string]interface{}
	if err := json.Unmarshal(jsonData, &single); err == nil {
		return []map[string]interface{}{single}, nil
	}

	// Handle nil/empty
	v := reflect.ValueOf(data)
	if !v.IsValid() || (v.Kind() == reflect.Ptr && v.IsNil()) {
		return nil, nil
	}
	if v.Kind() == reflect.Slice && v.Len() == 0 {
		return nil, nil
	}

	return nil, fmt.Errorf("unsupported data type for table output: %T", data)
}

// extractField gets a field value from a row map using a Column definition.
func extractField(row map[string]interface{}, col Column) string {
	if col.Func != nil {
		return col.Func(row)
	}

	// Support nested fields with dot notation
	parts := strings.Split(col.Field, ".")
	var val interface{} = row

	for _, part := range parts {
		m, ok := val.(map[string]interface{})
		if !ok {
			return ""
		}
		val, ok = m[part]
		if !ok {
			return ""
		}
	}

	if val == nil {
		return ""
	}

	switch v := val.(type) {
	case string:
		return v
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%.2f", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", v)
	}
}
