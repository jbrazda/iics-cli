// Package privmatrix provides a Bubble Tea model for toggling the privileges
// of one service, laid out as a grid of objects (rows) by actions (columns).
package privmatrix

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jbrazda/iics-cli/internal/client"
)

// Mode restricts which cells can be toggled.
type Mode int

const (
	// ModeEdit allows adding and removing privileges.
	ModeEdit Mode = iota
	// ModeAdd only allows adding; assigned privileges are locked.
	ModeAdd
	// ModeRemove only shows assigned privileges and only allows removing them.
	ModeRemove
)

// Options configures the model.
type Options struct {
	Title   string
	Mode    Mode
	NoColor bool
}

// rowKind distinguishes grid rows.
type rowKind int

const (
	rowObject rowKind = iota
	rowHeader
	rowOther
)

type row struct {
	kind   rowKind
	object *client.PrivilegeObject
	other  *client.Privilege
}

var columnHeaders = []string{"VIEW", "CREATE", "UPDATE", "DELETE", "EXEC", "PERM"}

const (
	cellWidth   = 8
	chromeLines = 6 // title, blank, column header, blank, detail, help
)

// Model is the Bubble Tea model for one service's privilege grid.
type Model struct {
	svc      client.PrivilegeService
	assigned map[string]bool
	selected map[string]bool
	opts     Options

	rows      []row
	cursorRow int
	cursorCol int
	offset    int
	width     int
	height    int

	filter    textinput.Model
	filtering bool
	done      bool

	styles styles
}

type styles struct {
	title, header, dim, add, remove, keep, cursor, detail lipgloss.Style
}

func newStyles(noColor bool) styles {
	if noColor {
		plain := lipgloss.NewStyle()
		return styles{
			title:  plain.Bold(true),
			header: plain.Bold(true),
			dim:    plain,
			add:    plain,
			remove: plain,
			keep:   plain.Bold(true),
			cursor: plain.Reverse(true),
			detail: plain,
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

// New creates a model. assigned holds the privilege names the role has now;
// selected holds the desired state and is updated in place.
func New(svc client.PrivilegeService, assigned, selected map[string]bool, opts Options) *Model {
	ti := textinput.New()
	ti.Prompt = "/"
	ti.Placeholder = "filter"
	m := &Model{
		svc:      svc,
		assigned: assigned,
		selected: selected,
		opts:     opts,
		filter:   ti,
		width:    100,
		height:   30,
		styles:   newStyles(opts.NoColor),
	}
	m.rebuildRows()
	return m
}

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd { return nil }

// Done reports whether the user left the grid.
func (m *Model) Done() bool { return m.done }

// visible reports whether a privilege is shown in the current mode.
func (m *Model) visible(p client.Privilege) bool {
	return m.opts.Mode != ModeRemove || m.assigned[p.Name]
}

// locked reports whether a privilege cannot be toggled in the current mode.
func (m *Model) locked(p client.Privilege) bool {
	switch m.opts.Mode {
	case ModeAdd:
		return m.assigned[p.Name]
	case ModeRemove:
		return !m.assigned[p.Name]
	}
	return false
}

func (m *Model) matches(p client.Privilege, q string) bool {
	return strings.Contains(strings.ToLower(p.Name), q) || strings.Contains(strings.ToLower(p.Description), q)
}

func (m *Model) rebuildRows() {
	q := strings.ToLower(strings.TrimSpace(m.filter.Value()))
	m.rows = m.rows[:0]
	for i := range m.svc.Objects {
		obj := &m.svc.Objects[i]
		show := false
		for _, p := range obj.Actions {
			if m.visible(p) && (q == "" || strings.Contains(strings.ToLower(obj.Key), q) || m.matches(p, q)) {
				show = true
				break
			}
		}
		if show {
			m.rows = append(m.rows, row{kind: rowObject, object: obj})
		}
	}
	var others []row
	for i := range m.svc.Other {
		p := &m.svc.Other[i]
		if m.visible(*p) && (q == "" || m.matches(*p, q)) {
			others = append(others, row{kind: rowOther, other: p})
		}
	}
	if len(others) > 0 {
		m.rows = append(m.rows, row{kind: rowHeader})
		m.rows = append(m.rows, others...)
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

// cell returns the privilege at a grid position, if any.
func (m *Model) cell(r, c int) (client.Privilege, bool) {
	if r < 0 || r >= len(m.rows) {
		return client.Privilege{}, false
	}
	switch m.rows[r].kind {
	case rowObject:
		if c < 0 || c >= len(client.PrivilegeActions) {
			return client.Privilege{}, false
		}
		p, ok := m.rows[r].object.Actions[client.PrivilegeActions[c]]
		if !ok || !m.visible(p) {
			return client.Privilege{}, false
		}
		return p, true
	case rowOther:
		if c != 0 {
			return client.Privilege{}, false
		}
		return *m.rows[r].other, true
	}
	return client.Privilege{}, false
}

func (m *Model) skipHeader(dir int) {
	if len(m.rows) == 0 {
		return
	}
	if m.rows[m.cursorRow].kind == rowHeader {
		next := m.cursorRow + dir
		if next < 0 || next >= len(m.rows) {
			next = m.cursorRow - dir
		}
		if next >= 0 && next < len(m.rows) {
			m.cursorRow = next
		}
	}
}

func (m *Model) clampCol() {
	if len(m.rows) == 0 {
		m.cursorCol = 0
		return
	}
	if m.rows[m.cursorRow].kind == rowOther {
		m.cursorCol = 0
		return
	}
	if m.cursorCol >= len(client.PrivilegeActions) {
		m.cursorCol = len(client.PrivilegeActions) - 1
	}
}

func (m *Model) toggle(p client.Privilege) {
	if m.locked(p) {
		return
	}
	m.selected[p.Name] = !m.selected[p.Name]
}

// setAll selects all toggleable privileges in ps, or clears them when every
// toggleable one is already selected.
func (m *Model) setAll(ps []client.Privilege) {
	allOn := true
	for _, p := range ps {
		if !m.locked(p) && !m.selected[p.Name] {
			allOn = false
			break
		}
	}
	for _, p := range ps {
		if !m.locked(p) {
			m.selected[p.Name] = !allOn
		}
	}
}

func (m *Model) rowPrivileges(r int) []client.Privilege {
	var ps []client.Privilege
	for c := range client.PrivilegeActions {
		if p, ok := m.cell(r, c); ok {
			ps = append(ps, p)
		}
	}
	if len(ps) == 0 {
		if p, ok := m.cell(r, 0); ok {
			ps = append(ps, p)
		}
	}
	return ps
}

func (m *Model) columnPrivileges(c int) []client.Privilege {
	var ps []client.Privilege
	for r, rw := range m.rows {
		if rw.kind != rowObject {
			continue
		}
		if p, ok := m.cell(r, c); ok {
			ps = append(ps, p)
		}
	}
	return ps
}

func (m *Model) pageSize() int {
	n := m.height - chromeLines
	if n < 3 {
		n = 3
	}
	return n
}

func (m *Model) moveRow(delta int) {
	if len(m.rows) == 0 {
		return
	}
	dir := 1
	if delta < 0 {
		dir = -1
	}
	m.cursorRow += delta
	if m.cursorRow < 0 {
		m.cursorRow = 0
	}
	if m.cursorRow >= len(m.rows) {
		m.cursorRow = len(m.rows) - 1
	}
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
	if m.offset < 0 {
		m.offset = 0
	}
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
			m.rebuildRows()
			return m, cmd
		}
		switch msg.String() {
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
			if p, ok := m.cell(m.cursorRow, m.cursorCol); ok {
				m.toggle(p)
			}
		case "a":
			m.setAll(m.rowPrivileges(m.cursorRow))
		case "c":
			if len(m.rows) > 0 && m.rows[m.cursorRow].kind == rowObject {
				m.setAll(m.columnPrivileges(m.cursorCol))
			}
		case "/":
			m.filtering = true
			m.filter.Focus()
			return m, textinput.Blink
		case "ctrl+u":
			m.filter.SetValue("")
			m.rebuildRows()
		}
		m.ensureVisible()
	}
	return m, nil
}

func (m *Model) mark(p client.Privilege) string {
	orig, want := m.assigned[p.Name], m.selected[p.Name]
	switch {
	case orig && want:
		if m.locked(p) {
			return m.styles.dim.Render("[x]")
		}
		return m.styles.keep.Render("[x]")
	case !orig && want:
		return m.styles.add.Render("[+]")
	case orig && !want:
		return m.styles.remove.Render("[-]")
	}
	if m.locked(p) {
		return m.styles.dim.Render("[ ]")
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
	w := m.width - cellWidth*len(columnHeaders) - 3
	if w > 48 {
		w = 48
	}
	if w < 16 {
		w = 16
	}
	return w
}

// View implements tea.Model.
func (m *Model) View() string {
	var b strings.Builder
	kw := m.keyWidth()

	title := fmt.Sprintf("%s > %s (%d)", m.opts.Title, m.svc.Name, m.svc.Count())
	b.WriteString(m.styles.title.Render(title))
	if m.filtering || m.filter.Value() != "" {
		b.WriteString("   " + m.filter.View())
	}
	b.WriteString("\n\n")

	hdr := fmt.Sprintf("  %-*s ", kw, "OBJECT")
	for _, h := range columnHeaders {
		hdr += fmt.Sprintf("%-*s", cellWidth, h)
	}
	b.WriteString(m.styles.header.Render(strings.TrimRight(hdr, " ")) + "\n")

	if len(m.rows) == 0 {
		b.WriteString(m.styles.dim.Render("  no matching privileges") + "\n")
	}
	end := m.offset + m.pageSize()
	if end > len(m.rows) {
		end = len(m.rows)
	}
	for r := m.offset; r < end; r++ {
		rw := m.rows[r]
		pointer := "  "
		if r == m.cursorRow {
			pointer = "> "
		}
		switch rw.kind {
		case rowHeader:
			b.WriteString(m.styles.dim.Render("  -- Other --") + "\n")
		case rowObject:
			b.WriteString(pointer + fmt.Sprintf("%-*s ", kw, truncate(rw.object.Key, kw)))
			for c := range client.PrivilegeActions {
				cellText := " . "
				if p, ok := m.cell(r, c); ok {
					cellText = m.mark(p)
				} else {
					cellText = m.styles.dim.Render(cellText)
				}
				if r == m.cursorRow && c == m.cursorCol {
					cellText = m.styles.cursor.Render(stripStyle(cellText))
				}
				b.WriteString(cellText + strings.Repeat(" ", cellWidth-3))
			}
			b.WriteString("\n")
		case rowOther:
			cellText := m.mark(*rw.other)
			if r == m.cursorRow {
				cellText = m.styles.cursor.Render(stripStyle(cellText))
			}
			b.WriteString(pointer + fmt.Sprintf("%-*s ", kw, truncate(rw.other.Name, kw)) + cellText + "\n")
		}
	}
	if len(m.rows) > m.pageSize() {
		b.WriteString(m.styles.dim.Render(fmt.Sprintf("  rows %d-%d of %d", m.offset+1, end, len(m.rows))) + "\n")
	} else {
		b.WriteString("\n")
	}

	if p, ok := m.cell(m.cursorRow, m.cursorCol); ok {
		detail := p.Name
		if p.Description != "" {
			detail += " - " + p.Description
		}
		if p.Status != "" {
			detail += " (" + p.Status + ")"
		}
		b.WriteString(m.styles.detail.Render(truncate(detail, m.width-1)) + "\n")
	} else {
		b.WriteString("\n")
	}
	help := "arrows move  space toggle  a row  c column  / filter  ctrl+u clear filter  enter/esc back"
	b.WriteString(m.styles.dim.Render(truncate(help, m.width-1)))
	return b.String()
}

// stripStyle returns the three-character mark without ANSI styling so the
// cursor style can be applied cleanly.
func stripStyle(s string) string {
	for _, mk := range []string{"[x]", "[+]", "[-]", "[ ]", " . "} {
		if strings.Contains(s, mk) {
			return mk
		}
	}
	return s
}
