package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
)

// minColumnWidth is the floor a column is shrunk to before it is dropped
// instead (CR-0038 D5).
const minColumnWidth = 8

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
	termWidth := adaptWidth(f.w, f.style, theme)

	var dropped []string
	if termWidth <= 0 {
		widths := computeColWidths(rows, columns)
		renderTable(f.w, rows, columns, widths, theme, f.style)
	} else {
		kept, widths, drop, vertical := planColumns(rows, columns, termWidth, theme)
		dropped = drop
		if vertical {
			renderVertical(f.w, rows, kept)
		} else {
			kept, widths = reorderWrappedLast(kept, widths)
			shrunk := applyShrink(kept, widths)
			renderTable(f.w, rows, shrunk, widths, theme, f.style)
		}
	}

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

	if len(dropped) > 0 {
		_, _ = fmt.Fprintf(os.Stderr, "# %d column(s) hidden (%s) - use --wide or -o json\n",
			len(dropped), strings.Join(dropped, ", "))
	}

	return nil
}

// adaptWidth returns the terminal width to adapt the table to, or 0 to
// disable adaptation entirely. Adaptation is disabled when the style says so
// (SkipAdapt, e.g. --wide or config style.responsiveTables: false), when the
// theme is markdown/gh (always colorless, script-friendly output), or when
// the writer is not a TTY - matching effectiveTheme's TTY check (CR-0038 D2).
func adaptWidth(w io.Writer, style TableStyle, theme string) int {
	if style.SkipAdapt || theme == "markdown" || theme == "gh" {
		return 0
	}
	if !isTerminal(w) {
		return 0
	}
	if style.Width > 0 {
		return style.Width
	}
	return 0
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

// planColumns fits columns to termWidth, applying the CR-0038 tier strategy:
// drop priority-5 columns first, then shrink (wrap/truncate) priority-4 and
// priority-3 columns, then drop priority-4 if that still isn't enough, then
// truncate priority-2. If priority-1 columns alone still overflow, useVertical
// is true and the caller should fall back to a vertical layout instead.
// Columns with Priority == 0 get a default Priority and Shrink inferred from
// Field/Width (see resolveDefaults).
func planColumns(rows []map[string]interface{}, columns []Column, termWidth int, theme string) (kept []Column, widths []int, dropped []string, useVertical bool) {
	kept = resolveDefaults(columns)
	widths = cappedWidths(rows, kept)

	fits := func() bool { return sumInts(widths)+layoutOverhead(theme, len(kept)) <= termWidth }
	if fits() {
		return kept, widths, nil, false
	}

	kept, widths, dropped = dropTier(kept, widths, dropped, theme, termWidth, 5, fits)
	if fits() {
		return kept, widths, dropped, false
	}

	shrinkTier(kept, widths, theme, termWidth, 4)
	if fits() {
		return kept, widths, dropped, false
	}
	shrinkTier(kept, widths, theme, termWidth, 3)
	if fits() {
		return kept, widths, dropped, false
	}

	kept, widths, dropped = dropTier(kept, widths, dropped, theme, termWidth, 4, fits)
	if fits() {
		return kept, widths, dropped, false
	}

	shrinkTier(kept, widths, theme, termWidth, 2)
	if fits() {
		return kept, widths, dropped, false
	}

	return kept, widths, dropped, true
}

// resolveDefaults returns a copy of columns with Priority/Shrink filled in
// for any column that left Priority unset (CR-0038 D4):
//
//  1. an opaque 24-char ID column (Field == "id" or ending in Id/ID, Width
//     == 24) -> priority 5, ShrinkNever
//  2. no Width floor set -> priority 4, ShrinkWrap
//  3. a short Width (<= 16) -> priority 2, ShrinkNever
//  4. the first column, if not already caught above -> priority 1
//  5. everything else -> priority 3, ShrinkTruncate
func resolveDefaults(columns []Column) []Column {
	out := make([]Column, len(columns))
	for i, c := range columns {
		if c.Priority != 0 {
			out[i] = c
			continue
		}
		switch {
		case looksLikeIDColumn(c):
			c.Priority, c.Shrink = 5, ShrinkNever
		case c.Width == 0:
			c.Priority, c.Shrink = 4, ShrinkWrap
		case c.Width > 0 && c.Width <= 16:
			c.Priority, c.Shrink = 2, ShrinkNever
		case i == 0:
			c.Priority = 1
		default:
			c.Priority, c.Shrink = 3, ShrinkTruncate
		}
		out[i] = c
	}
	return out
}

func looksLikeIDColumn(c Column) bool {
	if c.Width != 24 {
		return false
	}
	return c.Field == "id" || strings.HasSuffix(c.Field, "Id") || strings.HasSuffix(c.Field, "ID")
}

// cappedWidths returns each column's natural width, capped by MaxWidth.
func cappedWidths(rows []map[string]interface{}, columns []Column) []int {
	widths := computeColWidths(rows, columns)
	for i, c := range columns {
		if c.MaxWidth > 0 && widths[i] > c.MaxWidth {
			widths[i] = c.MaxWidth
		}
	}
	return widths
}

// layoutOverhead approximates the non-content characters (borders, padding,
// gaps) a theme adds around n columns, for width-fitting purposes.
func layoutOverhead(theme string, n int) int {
	if n == 0 {
		return 0
	}
	switch theme {
	case "minimal", "gh":
		return (n - 1) * colGap
	case "compact":
		return (n - 1) * compactColGap
	case "markdown":
		return (n+1)*3 - 1
	default: // default, plain: bordered
		return n*(2*cellPad+1) + 1
	}
}

func sumInts(xs []int) int {
	s := 0
	for _, x := range xs {
		s += x
	}
	return s
}

// dropTier removes columns in the given priority tier, widest first, until
// fits() is true or no more columns in that tier remain.
func dropTier(kept []Column, widths []int, dropped []string, theme string, termWidth, tier int, fits func() bool) ([]Column, []int, []string) {
	for !fits() {
		idx := widestInTier(kept, widths, tier, false)
		if idx < 0 {
			return kept, widths, dropped
		}
		dropped = append(dropped, kept[idx].Header)
		kept = append(append([]Column{}, kept[:idx]...), kept[idx+1:]...)
		widths = append(append([]int{}, widths[:idx]...), widths[idx+1:]...)
	}
	return kept, widths, dropped
}

// shrinkTier narrows columns in the given priority tier (skipping
// ShrinkNever columns) one visible column at a time, widest first, down to
// minColumnWidth, until the layout fits within termWidth.
func shrinkTier(kept []Column, widths []int, theme string, termWidth, tier int) {
	budget := termWidth - layoutOverhead(theme, len(kept))
	for sumInts(widths) > budget {
		idx := widestInTier(kept, widths, tier, true)
		if idx < 0 || widths[idx] <= minColumnWidth {
			return
		}
		widths[idx]--
	}
}

// widestInTier returns the index of the widest column in the given priority
// tier, or -1 if none qualify. When shrinkableOnly is true, ShrinkNever
// columns are excluded.
func widestInTier(kept []Column, widths []int, tier int, shrinkableOnly bool) int {
	best := -1
	for i, c := range kept {
		if c.Priority != tier {
			continue
		}
		if shrinkableOnly && c.Shrink == ShrinkNever {
			continue
		}
		if best < 0 || widths[i] > widths[best] {
			best = i
		}
	}
	return best
}

// reorderWrappedLast moves any kept column with Shrink == ShrinkWrap to the
// end of the display order, widest first among those, so a wrapped cell's
// extra physical lines never disrupt the alignment of columns to its right
// (CR-0038 D4 display-order rule). Column order for json/yaml/csv output is
// unaffected - this only reorders what is passed to the table renderer.
func reorderWrappedLast(kept []Column, widths []int) ([]Column, []int) {
	type pair struct {
		col   Column
		width int
	}
	var front, wrapped []pair
	for i, c := range kept {
		p := pair{c, widths[i]}
		if c.Shrink == ShrinkWrap {
			wrapped = append(wrapped, p)
		} else {
			front = append(front, p)
		}
	}
	if len(wrapped) == 0 {
		return kept, widths
	}
	sort.SliceStable(wrapped, func(i, j int) bool { return wrapped[i].width > wrapped[j].width })
	out := append(front, wrapped...)
	newCols := make([]Column, len(out))
	newWidths := make([]int, len(out))
	for i, p := range out {
		newCols[i] = p.col
		newWidths[i] = p.width
	}
	return newCols, newWidths
}

// applyShrink returns columns whose Func wraps or truncates the underlying
// value to fit the planned width, per each column's Shrink mode.
// ShrinkNever columns are left untouched (planColumns never shrinks them
// below their natural width).
func applyShrink(kept []Column, widths []int) []Column {
	out := make([]Column, len(kept))
	for i, c := range kept {
		orig := c
		width := widths[i]
		shrink := c.Shrink
		c.Func = func(row interface{}) string {
			m, _ := row.(map[string]interface{})
			raw := extractField(m, orig)
			switch shrink {
			case ShrinkWrap:
				return WrapCell(raw, width)
			case ShrinkTruncateLeft:
				return TruncateCellLeft(raw, width)
			case ShrinkNever:
				return raw
			default:
				return TruncateCell(raw, width)
			}
		}
		out[i] = c
	}
	return out
}

// renderVertical prints one PROPERTY: VALUE block per row - the fallback
// when even the priority-1 columns don't fit the terminal width side by
// side (CR-0038 D1).
func renderVertical(w io.Writer, rows []map[string]interface{}, columns []Column) {
	labelWidth := 0
	for _, c := range columns {
		if l := utf8.RuneCountInString(c.Header); l > labelWidth {
			labelWidth = l
		}
	}
	for i, row := range rows {
		if i > 0 {
			_, _ = fmt.Fprintln(w)
		}
		for _, c := range columns {
			_, _ = fmt.Fprintf(w, "%s  %s\n", padRight(c.Header, labelWidth), extractField(row, c))
		}
	}
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
			l := maxLineLen(extractField(row, col))
			if l > widths[i] {
				widths[i] = l
			}
		}
	}
	return widths
}

// maxLineLen returns the visible length of the longest physical line in s
// (cells may contain embedded newlines for multi-line rendering).
func maxLineLen(s string) int {
	if !strings.Contains(s, "\n") {
		return visibleLen(s)
	}
	max := 0
	for _, line := range strings.Split(s, "\n") {
		if l := visibleLen(line); l > max {
			max = l
		}
	}
	return max
}

// splitCellLines expands a logical row whose cells may contain embedded newlines
// into the physical lines needed to render it, padding shorter cells with "".
func splitCellLines(cells []string) [][]string {
	maxLines := 1
	parts := make([][]string, len(cells))
	for i, c := range cells {
		parts[i] = strings.Split(c, "\n")
		if len(parts[i]) > maxLines {
			maxLines = len(parts[i])
		}
	}
	lines := make([][]string, maxLines)
	for l := 0; l < maxLines; l++ {
		row := make([]string, len(cells))
		for i := range cells {
			if l < len(parts[i]) {
				row[i] = parts[i][l]
			}
		}
		lines[l] = row
	}
	return lines
}

// flattenCells replaces embedded newlines with spaces for renderers that cannot
// display multi-line cells.
func flattenCells(cells []string) []string {
	out := make([]string, len(cells))
	for i, c := range cells {
		out[i] = strings.ReplaceAll(c, "\n", " ")
	}
	return out
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
		for _, line := range splitCellLines(dataCells(row, columns)) {
			_, _ = fmt.Fprintln(w, borderedRow(b, line, widths, noStyle))
		}
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
		for _, line := range splitCellLines(sanitizedDataCells(row, columns)) {
			_, _ = fmt.Fprintln(w, borderedRow(b, line, widths, noStyle))
		}
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
		lines := splitCellLines(sanitizedDataCells(row, columns))
		for _, line := range lines {
			var rb strings.Builder
			for i, cell := range line {
				rb.WriteString(padRight(cell, widths[i]))
				if i < len(widths)-1 {
					rb.WriteString(strings.Repeat(" ", colGap))
				}
			}
			_, _ = fmt.Fprintln(w, rb.String())
		}
		if len(lines) > 1 {
			_, _ = fmt.Fprintln(w)
		}
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
		lines := splitCellLines(sanitizedDataCells(row, columns))
		for _, line := range lines {
			var rb strings.Builder
			for i, cell := range line {
				rb.WriteString(padRight(cell, widths[i]))
				if i < len(widths)-1 {
					rb.WriteString(strings.Repeat(" ", compactColGap))
				}
			}
			_, _ = fmt.Fprintln(w, rb.String())
		}
		if len(lines) > 1 {
			_, _ = fmt.Fprintln(w)
		}
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
		cells := flattenCells(sanitizedDataCells(row, columns))
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
		cells := flattenCells(sanitizedDataCells(row, columns))
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
	if !v.IsValid() || (v.Kind() == reflect.Pointer && v.IsNil()) {
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
