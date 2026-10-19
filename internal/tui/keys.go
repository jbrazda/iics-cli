package tui

import (
	"errors"
	"fmt"
	"io"
	"os"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// Key is a single key press read by ReadKey.
type Key int

// Keys returned by ReadKey.
const (
	KeyRune   Key = iota // a printable character
	KeyEnter             // Enter
	KeyEsc               // Esc on its own
	KeyCancel            // ctrl+c or ctrl+d
)

// parseKey reads the first key in b and returns the number of bytes it
// used. Escape sequences such as arrow and function keys are consumed and
// reported with ok false, as are other control characters.
func parseKey(b []byte) (k Key, r rune, ok bool, n int) {
	if len(b) == 0 {
		return 0, 0, false, 0
	}
	switch b[0] {
	case 0x1b:
		if len(b) == 1 {
			return KeyEsc, 0, true, 1
		}
		switch b[1] {
		case '[': // CSI: parameters, then a final byte in 0x40-0x7e
			for i := 2; i < len(b); i++ {
				if b[i] >= 0x40 && b[i] <= 0x7e {
					return 0, 0, false, i + 1
				}
			}
			return 0, 0, false, len(b)
		case 'O': // SS3: one more byte (F1-F4, keypad)
			return 0, 0, false, min(3, len(b))
		}
		return 0, 0, false, 2 // alt+key
	case 0x03, 0x04:
		return KeyCancel, 0, true, 1
	case '\r', '\n':
		return KeyEnter, 0, true, 1
	}
	r, size := utf8.DecodeRune(b)
	if r == utf8.RuneError || r < 0x20 {
		return 0, 0, false, size
	}
	return KeyRune, r, true, size
}

// ReadKey reads one key press from a terminal in raw mode, so nothing is
// echoed. Arrow keys, function keys and other control input are skipped,
// also when they arrive in the same read as the key.
func ReadKey(in *os.File) (Key, rune, error) {
	fd := int(in.Fd())
	state, err := term.MakeRaw(fd)
	if err != nil {
		return 0, 0, fmt.Errorf("reading key: %w", err)
	}
	defer func() { _ = term.Restore(fd, state) }()

	buf := make([]byte, 32)
	for {
		n, err := in.Read(buf)
		if err != nil {
			if err == io.EOF {
				return KeyCancel, 0, nil
			}
			return 0, 0, fmt.Errorf("reading key: %w", err)
		}
		// Return the first real key; anything typed after it in the same
		// read is dropped, which is fine for single-key prompts.
		for b := buf[:n]; len(b) > 0; {
			k, r, ok, used := parseKey(b)
			if ok {
				return k, r, nil
			}
			b = b[used:]
		}
	}
}

// KeyConfirm asks a yes/no question answered with a single key on a
// terminal: y or n, Enter for the default, Esc, q or ctrl+c for no. Other
// keys, including arrows, are ignored. The answer is echoed as "yes"/"no".
func KeyConfirm(in *os.File, out io.Writer, label string, defaultYes bool) (bool, error) {
	hint := "[y/N]"
	if defaultYes {
		hint = "[Y/n]"
	}
	_, _ = fmt.Fprintf(out, "%s %s: ", label, hint)
	for {
		k, r, err := ReadKey(in)
		if err != nil {
			return false, err
		}
		answer, done := false, true
		switch {
		case k == KeyEnter:
			answer = defaultYes
		case k == KeyEsc || k == KeyCancel:
		case r == 'y' || r == 'Y':
			answer = true
		case r == 'n' || r == 'N' || r == 'q' || r == 'Q':
		default:
			done = false
		}
		if done {
			word := "no"
			if answer {
				word = "yes"
			}
			_, _ = fmt.Fprintln(out, word)
			return answer, nil
		}
	}
}

// cancelKey turns Esc (and q when allowQ) into ctrl+c, which makes a huh
// form cancel, unless the focused field uses the key itself:
//   - Esc is left to the field while it has an enabled binding for it, such
//     as closing or clearing a list filter.
//   - q cancels only list and confirm fields, and not while a filter is
//     active; in text inputs it is typed.
func cancelKey(form *huh.Form, msg tea.Msg, allowQ bool) tea.Msg {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return msg
	}
	s := k.String()
	if s != "esc" && (s != "q" || !allowQ) {
		return msg
	}
	field := form.GetFocusedField()
	if field == nil {
		return msg
	}
	esc := tea.KeyMsg{Type: tea.KeyEscape}
	for _, b := range field.KeyBinds() {
		if b.Enabled() && key.Matches(esc, b) {
			// The field handles Esc (filter active); leave both keys to it.
			return msg
		}
	}
	if s == "q" {
		switch field.(type) {
		case *huh.Input, *huh.Text:
			return msg
		}
	}
	return tea.KeyMsg{Type: tea.KeyCtrlC}
}

var hintStyle = lipgloss.NewStyle().Faint(true)

// cancelModel runs a huh form, applies the cancel keys and shows a hint
// below the form's own key help.
type cancelModel struct {
	form   *huh.Form
	allowQ bool
	hint   string
}

func (m cancelModel) Init() tea.Cmd { return m.form.Init() }

func (m cancelModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	_, cmd := m.form.Update(cancelKey(m.form, msg, m.allowQ))
	if m.form.State != huh.StateNormal {
		return m, tea.Quit
	}
	return m, cmd
}

func (m cancelModel) View() string {
	if m.form.State != huh.StateNormal {
		return ""
	}
	return m.form.View() + "\n" + hintStyle.Render(m.hint)
}

// runCancelable runs form with Esc (and q when allowQ) as cancel keys and
// returns huh.ErrUserAborted when canceled. Accessible mode uses huh's line
// prompts unchanged.
func runCancelable(form *huh.Form, out io.Writer, accessible, allowQ bool) error {
	if accessible {
		return form.WithOutput(out).WithAccessible(true).Run()
	}
	hint := "esc or ctrl+c to cancel"
	if allowQ {
		hint = "esc, q (in lists) or ctrl+c to cancel"
	}
	// Leave quitting to cancelModel, which watches the form state.
	form.SubmitCmd = nil
	form.CancelCmd = nil
	m := cancelModel{form: form, allowQ: allowQ, hint: hint}
	if _, err := tea.NewProgram(m, tea.WithOutput(out), tea.WithReportFocus()).Run(); err != nil {
		return fmt.Errorf("running form: %w", err)
	}
	if form.State == huh.StateAborted {
		return huh.ErrUserAborted
	}
	return nil
}

// RunForm runs a single-step form (picker, question): Esc cancels, and q
// cancels lists and yes/no prompts. Canceling returns ErrCanceled.
func RunForm(form *huh.Form, out io.Writer, accessible bool) error {
	err := runCancelable(form, out, accessible, true)
	if errors.Is(err, huh.ErrUserAborted) {
		return ErrCanceled
	}
	return err
}

// RunWizard runs a multi-field wizard: Esc or ctrl+c cancels; q is always
// typed because wizards contain text inputs. Canceling returns
// huh.ErrUserAborted.
func RunWizard(form *huh.Form, out io.Writer, accessible bool) error {
	return runCancelable(form, out, accessible, false)
}
