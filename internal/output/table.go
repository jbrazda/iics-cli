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

const (
	cellPad = 1 // spaces of padding on each side of a bordered cell
	colGap  = 2 // spaces between columns in borderless themes
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
	renderTable(f.w, rows, columns, widths, theme)
	return nil
}

// effectiveTheme resolves the theme to use, downgrading to "plain" when
// no-color is set or the output writer is not a TTY.
func effectiveTheme(w io.Writer, style TableStyle) string {
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
			l := utf8.RuneCountInString(extractField(row, col))
			if l > widths[i] {
				widths[i] = l
			}
		}
	}
	return widths
}

// padRight right-pads s to exactly width visible runes.
func padRight(s string, width int) string {
	l := utf8.RuneCountInString(s)
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

func renderTable(w io.Writer, rows []map[string]interface{}, columns []Column, widths []int, theme string) {
	switch theme {
	case "minimal":
		renderMinimal(w, rows, columns, widths)
	case "compact":
		renderCompact(w, rows, columns, widths)
	case "plain":
		renderPlain(w, rows, columns, widths)
	default:
		renderDefault(w, rows, columns, widths)
	}
}

// Predefined lipgloss header styles.
var (
	colorHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	boldHeaderStyle  = lipgloss.NewStyle().Bold(true)
)

func noStyle(s string) string     { return s }
func colorHeader(s string) string { return colorHeaderStyle.Render(s) }
func boldHeader(s string) string  { return boldHeaderStyle.Render(s) }

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

// renderDefault renders with unicode rounded borders and cyan bold headers.
func renderDefault(w io.Writer, rows []map[string]interface{}, columns []Column, widths []int) {
	b := unicodeBorders
	headers := headerStrings(columns)
	_, _ = fmt.Fprintln(w, hSep(b, widths, b.topLeft, b.topMid, b.topRight))
	_, _ = fmt.Fprintln(w, borderedRow(b, headers, widths, colorHeader))
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
		_, _ = fmt.Fprintln(w, borderedRow(b, dataCells(row, columns), widths, noStyle))
	}
	_, _ = fmt.Fprintln(w, hSep(b, widths, b.botLeft, b.botMid, b.botRight))
}

// renderMinimal renders with no box borders, cyan bold headers, and a unicode underline separator.
func renderMinimal(w io.Writer, rows []map[string]interface{}, columns []Column, widths []int) {
	headers := headerStrings(columns)

	var hdr strings.Builder
	for i, h := range headers {
		hdr.WriteString(colorHeader(padRight(h, widths[i])))
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
		cells := dataCells(row, columns)
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

// renderCompact renders with no borders, bold headers, and 2-space column gaps.
func renderCompact(w io.Writer, rows []map[string]interface{}, columns []Column, widths []int) {
	headers := headerStrings(columns)

	var hdr strings.Builder
	for i, h := range headers {
		hdr.WriteString(boldHeader(padRight(h, widths[i])))
		if i < len(widths)-1 {
			hdr.WriteString(strings.Repeat(" ", colGap))
		}
	}
	_, _ = fmt.Fprintln(w, hdr.String())

	for _, row := range rows {
		cells := dataCells(row, columns)
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
