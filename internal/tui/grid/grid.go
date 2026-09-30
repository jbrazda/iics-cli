// Package grid provides a Bubble Tea model for toggling a grid of boolean
// cells: rows are items (objects, principals), columns are actions or
// permissions. It renders pending changes against an original state, supports
// filtering, row and column toggles, locked cells, and scrolling.
package grid

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Cell is one toggleable value. Cells with an empty Key do not exist for
// the row and render as " . ".
type Cell struct {
	Key    string
	Locked bool
	// Detail is shown in the footer when the cell is focused.
	Detail string
}

// RowKind distinguishes item rows from section headers.
type RowKind int

const (
	// RowItem is a row of cells.
	RowItem RowKind = iota
	// RowHeader is a non-selectable section label, e.g. "-- Other --".
	RowHeader
)

// Row is one grid row.
type Row struct {
	Kind  RowKind
	Label string
	// Tag is appended to the label, e.g. "(new)".
	Tag   string
	Cells []Cell
	// Match is extra text matched by the filter besides Label and cell details.
	Match string
}

// Options configures a Model.
type Options struct {
	Title   string
	Columns []string
	// KeyHeader is the header of the label column.
	KeyHeader string
	NoColor   bool
	// Actions are extra keys that end the grid with that action, e.g.
	// {"n": "add principal"}. They are listed in the help line.
	Actions map[string]string
	// ActionOrder fixes the order of Actions in the help line.
	ActionOrder []string
	// ClearKey, when set, clears every unlocked cell of the focused row.
	ClearKey string
	// EmptyText is shown when the grid has no rows at all.
	EmptyText string
}

const (
	cellWidth   = 8
	chromeLines = 6 // title, blank, column header, scroll info, detail, help
)

// Model is the Bubble Tea model.
type Model struct {
	opts     Options
	all      []Row
	rows     []Row // visible rows
	original map[string]bool
	selected map[string]bool

	cursorRow, cursorCol int
	offset               int
	width, height        int

	filter    textinput.Model
	filtering bool
	done      bool
	action    string

	styles styles
}

type styles struct {
	title, header, dim, add, remove, keep, cursor, detail lipgloss.Style
}

func newStyles(noColor bool) styles {
	if noColor {
		plain := lipgloss.NewStyle()
		return styles{
			title: plain.Bold(true), header: plain.Bold(true), dim: plain, add: plain,
			remove: plain, keep: plain.Bold(true), cursor: plain.Reverse(true), detail: plain,
		}
	}
	return styles{
		title:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")),
		header: lipgloss.NewStyle().Bold(true),
		dim:    lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
		add:    lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true),
		remove: lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true),
		keep:   lipgloss.NewStyle().Bold(true),
		cursor: lipgloss.NewStyle().Reverse(true),
		detail: lipgloss.NewStyle().Foreground(lipgloss.Color("250")),
	}
}

// New creates a model. original holds the keys that are set now; selected
// holds the desired state and is updated in place.
func New(rows []Row, original, selected map[string]bool, opts Options) *Model {
	ti := textinput.New()
	ti.Prompt = "/"
	ti.Placeholder = "filter"
	if opts.KeyHeader == "" {
		opts.KeyHeader = "NAME"
	}
	m := &Model{
		opts: opts, all: rows, original: original, selected: selected,
		filter: ti, width: 100, height: 30, styles: newStyles(opts.NoColor),
	}
	m.refilter()
	return m
}

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd { return nil }

// Done reports whether the user left the grid.
func (m *Model) Done() bool { return m.done }

// Action returns the extra action key that ended the grid, or "".
func (m *Model) Action() string { return m.action }

// Filter returns the current filter text.
func (m *Model) Filter() string { return m.filter.Value() }

// SetFilter sets the filter text.
func (m *Model) SetFilter(v string) {
	m.filter.SetValue(v)
	m.offset = 0
	m.refilter()
}

// VisibleRows returns the rows currently shown (after filtering).
func (m *Model) VisibleRows() []Row { return m.rows }

// Cursor returns the focused row and column.
func (m *Model) Cursor() (row, col int) { return m.cursorRow, m.cursorCol }

func (m *Model) matches(r Row, q string) bool {
	if q == "" {
		return true
	}
	if strings.Contains(strings.ToLower(r.Label), q) || strings.Contains(strings.ToLower(r.Match), q) {
		return true
	}
	for _, c := range r.Cells {
		if c.Key != "" && (strings.Contains(strings.ToLower(c.Key), q) || strings.Contains(strings.ToLower(c.Detail), q)) {
			return true
		}
	}
	return false
}

// refilter rebuilds the visible rows. A header is shown when at least one
// row of its section is visible.
func (m *Model) refilter() {
	q := strings.ToLower(strings.TrimSpace(m.filter.Value()))
	m.rows = m.rows[:0]
	var pending *Row
	for i := range m.all {
		r := m.all[i]
		if r.Kind == RowHeader {
			pending = &m.all[i]
			continue
		}
		if !m.matches(r, q) {
			continue
		}
		if pending != nil {
			m.rows = append(m.rows, *pending)
			pending = nil
		}
		m.rows = append(m.rows, r)
	}
	if m.cursorRow >= len(m.rows) {
		m.cursorRow = len(m.rows) - 1
	}
	if m.cursorRow < 0 {
		m.cursorRow = 0
	}
	m.skipHeader(1)
	m.clampCol()
}

func (m *Model) cell(r, c int) (Cell, bool) {
	if r < 0 || r >= len(m.rows) || m.rows[r].Kind != RowItem {
		return Cell{}, false
	}
	cells := m.rows[r].Cells
	if c < 0 || c >= len(cells) || cells[c].Key == "" {
		return Cell{}, false
	}
	return cells[c], true
}

func (m *Model) skipHeader(dir int) {
	if len(m.rows) == 0 || m.rows[m.cursorRow].Kind != RowHeader {
		return
	}
	next := m.cursorRow + dir
	if next < 0 || next >= len(m.rows) {
		next = m.cursorRow - dir
	}
	if next >= 0 && next < len(m.rows) {
		m.cursorRow = next
	}
}

func (m *Model) clampCol() {
	if len(m.rows) == 0 {
		m.cursorCol = 0
		return
	}
	n := len(m.rows[m.cursorRow].Cells)
	if m.cursorCol >= n {
		m.cursorCol = max(n-1, 0)
	}
}

func (m *Model) toggle(c Cell) {
	if c.Locked {
		return
	}
	m.selected[c.Key] = !m.selected[c.Key]
}

// setAll sets all unlocked cells, or clears them when all are already set.
func (m *Model) setAll(cells []Cell) {
	allOn := true
	for _, c := range cells {
		if !c.Locked && !m.selected[c.Key] {
			allOn = false
			break
		}
	}
	for _, c := range cells {
		if !c.Locked {
			m.selected[c.Key] = !allOn
		}
	}
}

func (m *Model) rowCells(r int) []Cell {
	var out []Cell
	for c := range m.opts.Columns {
		if cell, ok := m.cell(r, c); ok {
			out = append(out, cell)
		}
	}
	return out
}

func (m *Model) columnCells(c int) []Cell {
	var out []Cell
	for r := range m.rows {
		// Only rows with a cell per column take part in column toggles.
		if len(m.rows[r].Cells) < len(m.opts.Columns) {
			continue
		}
		if cell, ok := m.cell(r, c); ok {
			out = append(out, cell)
		}
	}
	return out
}

// ClearRow unsets every unlocked cell of the focused row.
func (m *Model) ClearRow() {
	for _, c := range m.rowCells(m.cursorRow) {
		if !c.Locked {
			m.selected[c.Key] = false
		}
	}
}

func (m *Model) pageSize() int {
	return max(m.height-chromeLines, 3)
}

func (m *Model) moveRow(delta int) {
	if len(m.rows) == 0 {
		return
	}
	dir := 1
	if delta < 0 {
		dir = -1
	}
	m.cursorRow = min(max(m.cursorRow+delta, 0), len(m.rows)-1)
	m.skipHeader(dir)
	m.clampCol()
}

func (m *Model) ensureVisible() {
	page := m.pageSize()
	if m.cursorRow < m.offset {
		m.offset = m.cursorRow
	}
	if m.cursorRow >= m.offset+page {
		m.offset = m.cursorRow - page + 1
	}
	m.offset = max(m.offset, 0)
}

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.ensureVisible()
		return m, nil
	case tea.KeyMsg:
		if m.filtering {
			switch msg.String() {
			case "enter", "esc", "down", "up":
				m.filtering = false
				m.filter.Blur()
				return m, nil
			}
			var cmd tea.Cmd
			m.filter, cmd = m.filter.Update(msg)
			m.offset = 0
			m.refilter()
			return m, cmd
		}
		key := msg.String()
		if m.opts.ClearKey != "" && key == m.opts.ClearKey {
			m.ClearRow()
			return m, nil
		}
		if _, ok := m.opts.Actions[key]; ok {
			m.action = key
			m.done = true
			return m, tea.Quit
		}
		switch key {
		case "ctrl+c", "esc", "enter", "q":
			m.done = true
			return m, tea.Quit
		case "up", "k":
			m.moveRow(-1)
		case "down", "j":
			m.moveRow(1)
		case "pgup":
			m.moveRow(-m.pageSize())
		case "pgdown":
			m.moveRow(m.pageSize())
		case "home", "g":
			m.cursorRow = 0
			m.skipHeader(1)
			m.clampCol()
		case "end", "G":
			m.cursorRow = len(m.rows) - 1
			m.skipHeader(-1)
			m.clampCol()
		case "left", "h":
			if m.cursorCol > 0 {
				m.cursorCol--
			}
		case "right", "l":
			m.cursorCol++
			m.clampCol()
		case " ", "x":
			if c, ok := m.cell(m.cursorRow, m.cursorCol); ok {
				m.toggle(c)
			}
		case "a":
			m.setAll(m.rowCells(m.cursorRow))
		case "c":
			if len(m.rows) > 0 && len(m.rows[m.cursorRow].Cells) >= len(m.opts.Columns) {
				m.setAll(m.columnCells(m.cursorCol))
			}
		case "/":
			m.filtering = true
			m.filter.Focus()
			return m, textinput.Blink
		case "ctrl+u":
			m.SetFilter("")
		}
		m.ensureVisible()
	}
	return m, nil
}

func (m *Model) mark(c Cell) string {
	orig, want := m.original[c.Key], m.selected[c.Key]
	switch {
	case orig && want:
		if c.Locked {
			return m.styles.dim.Render("[x]")
		}
		return m.styles.keep.Render("[x]")
	case !orig && want:
		return m.styles.add.Render("[+]")
	case orig && !want:
		return m.styles.remove.Render("[-]")
	}
	if c.Locked {
		return m.styles.dim.Render("[ ]")
	}
	return "[ ]"
}

func plainMark(orig, want bool) string {
	switch {
	case orig && want:
		return "[x]"
	case !orig && want:
		return "[+]"
	case orig && !want:
		return "[-]"
	}
	return "[ ]"
}

func truncate(s string, n int) string {
	if n <= 1 {
		return ""
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

func (m *Model) keyWidth() int {
	return min(max(m.width-cellWidth*len(m.opts.Columns)-3, 16), 48)
}

func (m *Model) helpLine() string {
	parts := []string{"arrows move", "space toggle", "a row", "c column", "/ filter", "ctrl+u clear filter"}
	if m.opts.ClearKey != "" {
		parts = append(parts, m.opts.ClearKey+" clear row")
	}
	for _, k := range m.opts.ActionOrder {
		if desc, ok := m.opts.Actions[k]; ok {
			parts = append(parts, k+" "+desc)
		}
	}
	parts = append(parts, "enter/esc done")
	return strings.Join(parts, "  ")
}

// View implements tea.Model.
func (m *Model) View() string {
	var b strings.Builder
	kw := m.keyWidth()

	b.WriteString(m.styles.title.Render(m.opts.Title))
	if m.filtering || m.filter.Value() != "" {
		b.WriteString("   ")
		b.WriteString(m.filter.View())
	}
	b.WriteString("\n\n")

	var hdr strings.Builder
	_, _ = fmt.Fprintf(&hdr, "  %-*s ", kw, m.opts.KeyHeader)
	for _, h := range m.opts.Columns {
		_, _ = fmt.Fprintf(&hdr, "%-*s", cellWidth, h)
	}
	b.WriteString(m.styles.header.Render(strings.TrimRight(hdr.String(), " ")))
	b.WriteString("\n")

	if len(m.rows) == 0 {
		msg := "no matching rows"
		if len(m.all) == 0 && m.opts.EmptyText != "" {
			msg = m.opts.EmptyText
		}
		b.WriteString(m.styles.dim.Render("  " + msg))
		b.WriteString("\n")
	}
	end := min(m.offset+m.pageSize(), len(m.rows))
	for r := m.offset; r < end; r++ {
		row := m.rows[r]
		if row.Kind == RowHeader {
			b.WriteString(m.styles.dim.Render("  " + row.Label))
			b.WriteString("\n")
			continue
		}
		pointer := "  "
		if r == m.cursorRow {
			pointer = "> "
		}
		label := row.Label
		if row.Tag != "" {
			label += " " + row.Tag
		}
		b.WriteString(pointer)
		_, _ = fmt.Fprintf(&b, "%-*s ", kw, truncate(label, kw))
		for c := range row.Cells {
			var text string
			if cell, ok := m.cell(r, c); ok {
				text = m.mark(cell)
				if r == m.cursorRow && c == m.cursorCol {
					text = m.styles.cursor.Render(plainMark(m.original[cell.Key], m.selected[cell.Key]))
				}
			} else {
				text = m.styles.dim.Render(" . ")
				if r == m.cursorRow && c == m.cursorCol {
					text = m.styles.cursor.Render(" . ")
				}
			}
			b.WriteString(text)
			if c < len(row.Cells)-1 {
				b.WriteString(strings.Repeat(" ", cellWidth-3))
			}
		}
		b.WriteString("\n")
	}
	if len(m.rows) > m.pageSize() {
		b.WriteString(m.styles.dim.Render(fmt.Sprintf("  rows %d-%d of %d", m.offset+1, end, len(m.rows))))
	}
	b.WriteString("\n")

	if c, ok := m.cell(m.cursorRow, m.cursorCol); ok && c.Detail != "" {
		b.WriteString(m.styles.detail.Render(truncate(c.Detail, m.width-1)))
	}
	b.WriteString("\n")
	b.WriteString(m.styles.dim.Render(truncate(m.helpLine(), m.width-1)))
	return b.String()
}
