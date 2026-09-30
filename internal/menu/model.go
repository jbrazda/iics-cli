package menu

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Header describes the active profile and session shown above the menu.
type Header struct {
	Profile    string
	Org        string
	User       string
	Session    string
	Production bool
	// Notice is an extra line, e.g. "No profile configured".
	Notice string
	// Binary is the command name used for the equivalent command line.
	Binary string
}

// ResultKind is what the user chose in the menu.
type ResultKind int

const (
	// ResultQuit ends the menu.
	ResultQuit ResultKind = iota
	// ResultRun runs Result.Entry.
	ResultRun
	// ResultSwitchProfile opens the profile picker.
	ResultSwitchProfile
)

// Result is returned by the model when it finishes.
type Result struct {
	Kind  ResultKind
	Entry Entry
}

type row struct {
	header bool
	group  string
	entry  Entry
	index  int // index in entries
}

// Model is the Bubble Tea main menu.
type Model struct {
	entries []Entry
	header  Header
	// hasProfile disables non-session entries when false.
	hasProfile bool

	rows   []row
	cursor int
	offset int
	width  int
	height int

	filter    textinput.Model
	filtering bool
	showHelp  bool
	message   string

	result Result
	done   bool
	styles menuStyles
}

type menuStyles struct {
	title, prod, header, group, selected, dim, desc, cmd, warn lipgloss.Style
}

func newMenuStyles(noColor bool) menuStyles {
	if noColor {
		p := lipgloss.NewStyle()
		return menuStyles{
			title: p.Bold(true), prod: p.Bold(true).Reverse(true), header: p, group: p.Bold(true),
			selected: p.Reverse(true), dim: p, desc: p, cmd: p, warn: p.Bold(true),
		}
	}
	return menuStyles{
		title:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")),
		prod:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("1")).Padding(0, 1),
		header:   lipgloss.NewStyle().Foreground(lipgloss.Color("250")),
		group:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("4")),
		selected: lipgloss.NewStyle().Bold(true).Reverse(true),
		dim:      lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
		desc:     lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
		cmd:      lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Italic(true),
		warn:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("1")),
	}
}

// NewModel creates the menu. last is the label of the entry to focus.
func NewModel(entries []Entry, header Header, hasProfile bool, last string, noColor bool) *Model {
	ti := textinput.New()
	ti.Prompt = "/"
	ti.Placeholder = "filter"
	m := &Model{
		entries: entries, header: header, hasProfile: hasProfile,
		filter: ti, width: 100, height: 30, styles: newMenuStyles(noColor),
	}
	m.rebuild()
	for i, r := range m.rows {
		if !r.header && r.entry.Label == last {
			m.cursor = i
		}
	}
	m.skipHeader(1)
	return m
}

// Result returns the user's choice once the model is done.
func (m *Model) Result() Result { return m.result }

// Done reports whether the model finished.
func (m *Model) Done() bool { return m.done }

func (m *Model) enabled(e Entry) bool {
	return m.hasProfile || e.SessionOnly
}

func (m *Model) rebuild() {
	q := strings.ToLower(strings.TrimSpace(m.filter.Value()))
	m.rows = m.rows[:0]
	lastGroup := ""
	for i, e := range m.entries {
		if q != "" {
			hay := strings.ToLower(e.Group + " " + e.Label + " " + e.Description + " " + strings.Join(e.Args, " "))
			if !strings.Contains(hay, q) {
				continue
			}
		}
		if e.Group != lastGroup {
			m.rows = append(m.rows, row{header: true, group: e.Group})
			lastGroup = e.Group
		}
		m.rows = append(m.rows, row{group: e.Group, entry: e, index: i})
	}
	if m.cursor >= len(m.rows) {
		m.cursor = len(m.rows) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.skipHeader(1)
}

func (m *Model) skipHeader(dir int) {
	for len(m.rows) > 0 && m.rows[m.cursor].header {
		next := m.cursor + dir
		if next < 0 || next >= len(m.rows) {
			dir = -dir
			next = m.cursor + dir
			if next < 0 || next >= len(m.rows) {
				return
			}
		}
		m.cursor = next
	}
}

func (m *Model) move(delta int) {
	if len(m.rows) == 0 {
		return
	}
	dir := 1
	if delta < 0 {
		dir = -1
	}
	m.cursor = min(max(m.cursor+delta, 0), len(m.rows)-1)
	m.skipHeader(dir)
}

func (m *Model) headerLines() int {
	n := 3 // title, profile line, blank
	if m.header.Notice != "" {
		n++
	}
	return n
}

func (m *Model) pageSize() int {
	footer := 4 // blank, description, command, key line
	if m.showHelp {
		footer += 6
	}
	return max(m.height-m.headerLines()-footer, 3)
}

func (m *Model) ensureVisible() {
	page := m.pageSize()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+page {
		m.offset = m.cursor - page + 1
	}
	m.offset = max(m.offset, 0)
}

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.ensureVisible()
		return m, nil
	case tea.KeyMsg:
		m.message = ""
		if m.filtering {
			switch msg.String() {
			case "enter", "down", "up":
				m.filtering = false
				m.filter.Blur()
				return m, nil
			case "esc":
				m.filtering = false
				m.filter.Blur()
				m.filter.SetValue("")
				m.rebuild()
				return m, nil
			case "ctrl+c":
				return m.finish(Result{Kind: ResultQuit})
			}
			var cmd tea.Cmd
			m.filter, cmd = m.filter.Update(msg)
			m.offset = 0
			m.rebuild()
			return m, cmd
		}
		switch msg.String() {
		case "ctrl+c", "q":
			return m.finish(Result{Kind: ResultQuit})
		case "esc":
			if m.filter.Value() != "" {
				m.filter.SetValue("")
				m.rebuild()
			}
		case "up", "k":
			m.move(-1)
		case "down", "j":
			m.move(1)
		case "pgup":
			m.move(-m.pageSize())
		case "pgdown":
			m.move(m.pageSize())
		case "home", "g":
			m.cursor = 0
			m.skipHeader(1)
		case "end", "G":
			m.cursor = len(m.rows) - 1
			m.skipHeader(-1)
		case "/":
			m.filtering = true
			m.filter.Focus()
			return m, textinput.Blink
		case "?":
			m.showHelp = !m.showHelp
		case "p":
			return m.finish(Result{Kind: ResultSwitchProfile})
		case "enter":
			if len(m.rows) == 0 {
				return m, nil
			}
			e := m.rows[m.cursor].entry
			if !m.enabled(e) {
				m.message = "Add a profile first (Session > Add profile)."
				return m, nil
			}
			if e.Action == ActionSwitchProfile {
				return m.finish(Result{Kind: ResultSwitchProfile})
			}
			return m.finish(Result{Kind: ResultRun, Entry: e})
		}
		m.ensureVisible()
	}
	return m, nil
}

func (m *Model) finish(r Result) (tea.Model, tea.Cmd) {
	m.result = r
	m.done = true
	return m, tea.Quit
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

// View implements tea.Model.
func (m *Model) View() string {
	var b strings.Builder
	w := m.width - 1
	h := m.header

	b.WriteString(m.styles.title.Render("iics"))
	b.WriteString("  ")
	b.WriteString(m.styles.dim.Render("main menu"))
	if m.filtering || m.filter.Value() != "" {
		b.WriteString("   ")
		b.WriteString(m.filter.View())
	}
	b.WriteString("\n")

	profile := "profile " + orDash(h.Profile)
	var parts []string
	parts = append(parts, profile)
	if h.Org != "" {
		parts = append(parts, "org "+h.Org)
	}
	if h.User != "" {
		parts = append(parts, h.User)
	}
	if h.Session != "" {
		parts = append(parts, h.Session)
	}
	line := m.styles.header.Render(truncate(strings.Join(parts, " · "), w-14))
	if h.Production {
		line = m.styles.prod.Render("PRODUCTION") + " " + line
	}
	b.WriteString(line)
	b.WriteString("\n")
	if h.Notice != "" {
		b.WriteString(m.styles.warn.Render(truncate(h.Notice, w)))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	if len(m.rows) == 0 {
		b.WriteString(m.styles.dim.Render("  no matching entries"))
		b.WriteString("\n")
	}
	end := min(m.offset+m.pageSize(), len(m.rows))
	for i := m.offset; i < end; i++ {
		r := m.rows[i]
		if r.header {
			b.WriteString(m.styles.group.Render(r.group))
			b.WriteString("\n")
			continue
		}
		label := r.entry.Label
		if r.entry.Writes {
			label += " ✎"
		}
		text := "  " + truncate(label, w-4)
		switch {
		case i == m.cursor:
			b.WriteString(m.styles.selected.Render("> " + truncate(label, w-4)))
		case !m.enabled(r.entry):
			b.WriteString(m.styles.dim.Render(text))
		default:
			b.WriteString(text)
		}
		b.WriteString("\n")
	}
	if len(m.rows) > m.pageSize() {
		b.WriteString(m.styles.dim.Render(fmt.Sprintf("  %d-%d of %d", m.offset+1, end, len(m.rows))))
	}
	b.WriteString("\n")

	if len(m.rows) > 0 {
		e := m.rows[m.cursor].entry
		b.WriteString(m.styles.desc.Render(truncate(e.Description, w)))
		b.WriteString("\n")
		if e.Action == "" {
			binary := h.Binary
			if binary == "" {
				binary = "iics"
			}
			b.WriteString(m.styles.cmd.Render(truncate("$ "+CommandLine(binary, e.CommandArgs(h.Profile, "<"+strings.ToLower(e.AskArg)+">")), w)))
		}
		b.WriteString("\n")
	}
	if m.message != "" {
		b.WriteString(m.styles.warn.Render(truncate(m.message, w)))
		b.WriteString("\n")
	}
	if m.showHelp {
		for _, l := range []string{
			"↑/↓ k/j  move          pgup/pgdn  page       g/G  first/last",
			"enter    run entry     /          filter     esc  clear filter",
			"p        switch profile                      ?    hide help",
			"q        quit          ctrl+c (in a command) cancels only that command",
			"✎ marks entries that change data; on a PRODUCTION profile they ask first.",
		} {
			b.WriteString(m.styles.dim.Render(truncate(l, w)))
			b.WriteString("\n")
		}
	}
	b.WriteString(m.styles.dim.Render(truncate("enter run · / filter · p profile · ? help · q quit", w)))
	return b.String()
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
