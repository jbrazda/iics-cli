package menu

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jbrazda/iics-cli/internal/tui"
)

// Runner runs a command line and returns its exit code.
type Runner interface {
	Run(args []string) int
}

// ExecRunner runs the CLI binary as a subprocess attached to the terminal.
// While the child runs, the menu ignores interrupts so Ctrl+C only reaches
// the child.
type ExecRunner struct {
	Binary string
}

// Run implements Runner.
func (r ExecRunner) Run(args []string) int {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	defer signal.Stop(sig)

	c := exec.Command(r.Binary, args...)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	err := c.Run()
	var exitErr *exec.ExitError
	switch {
	case err == nil:
		return 0
	case errors.As(err, &exitErr):
		return exitErr.ExitCode()
	default:
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return -1
	}
}

// ProfileInfo describes a configured profile for the profile picker.
type ProfileInfo struct {
	Name       string
	Region     string
	Production bool
	Default    bool
}

// State is what the menu shows for the active profile.
type State struct {
	Profiles   []ProfileInfo
	Header     Header
	HasProfile bool
}

// Deps are the menu's collaborators; tests replace them with fakes.
type Deps struct {
	Runner   Runner
	Prompter *tui.Prompter
	Out      io.Writer
	Binary   string
	NoColor  bool
	// Line selects the numbered, line-based menu (accessible / small terminal).
	Line bool
	// LoadState reads config and session information for a profile.
	LoadState func(profile string) (State, error)
	// SetDefault makes a profile the config default.
	SetDefault func(profile string) error
	// ShowMenu displays the full-screen menu and returns the choice.
	ShowMenu func(m *Model) (Result, error)
	// Pause waits after a command; it returns true when the user quits.
	Pause func() (bool, error)
	// ClearScreen clears the terminal before the menu is shown again.
	ClearScreen func()
	// Entries overrides DefaultEntries (tests).
	Entries []Entry
}

// DefaultShowMenu runs the model as a full-screen Bubble Tea program.
func DefaultShowMenu(m *Model) (Result, error) {
	if _, err := tea.NewProgram(m, tea.WithAltScreen(), tea.WithOutput(os.Stderr)).Run(); err != nil {
		return Result{}, err
	}
	return m.Result(), nil
}

// DefaultPause asks the user to press Enter (menu) or q (quit).
func DefaultPause(out io.Writer) func() (bool, error) {
	return func() (bool, error) {
		_, _ = fmt.Fprint(out, "Press Enter to return to the menu, q to quit: ")
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil && line == "" {
			return true, nil
		}
		return strings.EqualFold(strings.TrimSpace(line), "q"), nil
	}
}

// DefaultClearScreen clears the terminal.
func DefaultClearScreen(out io.Writer) func() {
	return func() { _, _ = fmt.Fprint(out, "\x1b[H\x1b[2J") }
}

// Run shows the menu for startProfile until the user quits.
func Run(d Deps, startProfile string) error {
	entries := d.Entries
	if entries == nil {
		entries = DefaultEntries()
	}
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	ok := lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
	bad := lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
	if d.NoColor {
		dim, ok, bad = lipgloss.NewStyle(), lipgloss.NewStyle(), lipgloss.NewStyle()
	}

	profile := startProfile
	st, err := d.LoadState(profile)
	if err != nil {
		return err
	}

	// First run: set up a profile before showing the menu.
	if !st.HasProfile && len(st.Profiles) == 0 {
		_, _ = fmt.Fprintln(d.Out, "No profile is configured yet. Let's add one.")
		name, perr := d.Prompter.Text("Profile name", "default")
		if perr != nil {
			return perr
		}
		if code := d.run(dim, []string{"profile", "add", name}); code == 0 {
			profile = name
		}
		if d.ClearScreen != nil {
			d.ClearScreen()
		}
	}

	last := ""
	for {
		st, err = d.LoadState(profile)
		if err != nil {
			return err
		}
		if !st.HasProfile && len(st.Profiles) == 0 {
			st.Header.Notice = "No profile configured - choose Session > Add profile."
		} else if !st.HasProfile {
			st.Header.Notice = fmt.Sprintf("Profile %q is not configured - switch profile (p) or add it.", profile)
		}
		st.Header.Binary = d.Binary

		res, err := d.choose(entries, st, last)
		if err != nil {
			return err
		}
		if res.Kind == ResultQuit {
			return nil
		}
		// Start every entry on a clean screen: the menu runs in the alternate
		// screen, so the normal screen still shows the shell before iics or
		// prompts from entries that returned to the menu without a pause.
		if d.ClearScreen != nil {
			d.ClearScreen()
		}
		switch res.Kind {
		case ResultSwitchProfile:
			if picked, perr := d.pickProfile("Switch profile", st.Profiles, profile); perr != nil {
				return perr
			} else if picked != "" {
				profile = picked
			}
			last = "Switch profile"
			continue
		}

		e := res.Entry
		last = e.Label
		if e.Writes && st.Header.Production {
			yes, cerr := d.Prompter.Confirm(fmt.Sprintf("Run %q against PRODUCTION (%s)?", e.Label, profile), false)
			if cerr != nil {
				return cerr
			}
			if !yes {
				continue
			}
		}

		var code int
		switch e.Action {
		case ActionSetDefault:
			if serr := d.SetDefault(profile); serr != nil {
				_, _ = fmt.Fprintf(d.Out, "Error: %v\n", serr)
				code = 1
			} else {
				_, _ = fmt.Fprintf(d.Out, "Default profile set to %q.\n", profile)
			}
		default:
			asked := ""
			if e.AskArg != "" {
				asked, err = d.Prompter.Text(e.AskArg, "")
				if err != nil {
					return err
				}
				if asked == "" {
					continue
				}
			}
			if e.PickProfile != "" {
				asked, err = d.pickProfile(e.PickProfile, st.Profiles, profile)
				if err != nil {
					return err
				}
				if asked == "" {
					continue
				}
			}
			code = d.run(dim, e.CommandArgs(profile, asked))
			if code == 0 && e.AskArg != "" && !st.HasProfile {
				profile = asked // first profile just added
			}
			if e.PickProfile != "" && asked == profile {
				// The active profile may be gone: fall back to another one.
				after, lerr := d.LoadState(profile)
				if lerr != nil {
					return lerr
				}
				profile = remainingProfile(after.Profiles, profile)
			}
		}

		if code == 0 {
			_, _ = fmt.Fprintln(d.Out, ok.Render("✓ done"))
		} else {
			_, _ = fmt.Fprintln(d.Out, bad.Render(fmt.Sprintf("✗ failed (exit %d)", code)))
		}
		quit, perr := d.Pause()
		if perr != nil {
			return perr
		}
		if quit {
			return nil
		}
		if d.ClearScreen != nil {
			d.ClearScreen()
		}
	}
}

func (d Deps) run(dim lipgloss.Style, args []string) int {
	_, _ = fmt.Fprintln(d.Out, dim.Render("$ "+CommandLine(d.Binary, args)))
	return d.Runner.Run(args)
}

func (d Deps) choose(entries []Entry, st State, last string) (Result, error) {
	if !d.Line {
		return d.ShowMenu(NewModel(entries, st.Header, st.HasProfile, last, d.NoColor))
	}
	return lineMenu(d, entries, st)
}

// lineMenu is the numbered, line-based menu.
func lineMenu(d Deps, entries []Entry, st State) (Result, error) {
	h := st.Header
	head := "iics main menu - profile " + orDash(h.Profile)
	if h.Production {
		head += " [PRODUCTION]"
	}
	if h.Org != "" {
		head += " - org " + h.Org
	}
	if h.Session != "" {
		head += " - " + h.Session
	}
	_, _ = fmt.Fprintln(d.Out, head)
	if h.Notice != "" {
		_, _ = fmt.Fprintln(d.Out, h.Notice)
	}
	var usable []Entry
	var labels []string
	for _, e := range entries {
		if !st.HasProfile && !e.SessionOnly {
			continue
		}
		label := e.Group + " > " + e.Label
		if e.Writes {
			label += " (changes data)"
		}
		usable = append(usable, e)
		labels = append(labels, label)
	}
	idx, err := d.Prompter.Select("Choose an entry (0 or q quits)", labels)
	if err != nil {
		return Result{}, err
	}
	if idx < 0 {
		return Result{Kind: ResultQuit}, nil
	}
	e := usable[idx]
	if e.Action == ActionSwitchProfile {
		return Result{Kind: ResultSwitchProfile}, nil
	}
	return Result{Kind: ResultRun, Entry: e}, nil
}

func (d Deps) pickProfile(title string, profiles []ProfileInfo, current string) (string, error) {
	if len(profiles) == 0 {
		_, _ = fmt.Fprintln(d.Out, "No profiles configured.")
		return "", nil
	}
	idx, err := tui.PickOne(d.Prompter, title, profiles, func(p ProfileInfo) string {
		label := p.Name
		if p.Region != "" {
			label += "  " + p.Region
		}
		if p.Production {
			label += "  PRODUCTION"
		}
		if p.Default {
			label += "  (default)"
		}
		if p.Name == current {
			label += "  (active)"
		}
		return label
	})
	if err != nil || idx < 0 {
		return "", err
	}
	return profiles[idx].Name, nil
}

// remainingProfile returns current while it is still configured, otherwise
// the default profile, the first profile, or "" when none is left.
func remainingProfile(profiles []ProfileInfo, current string) string {
	next := ""
	for i, p := range profiles {
		if p.Name == current {
			return current
		}
		if p.Default || i == 0 {
			next = p.Name
		}
	}
	return next
}
