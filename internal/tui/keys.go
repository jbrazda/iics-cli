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

// cancelKeysFilter makes Esc and q cancel a huh form the same way ctrl+c
// does, unless the focused field uses the key itself:
//   - Esc is left to the field while it has an enabled binding for it, such
//     as closing or clearing a list filter.
//   - q cancels only list and confirm fields, and not while a filter is
//     active; in text inputs it is typed.
func cancelKeysFilter(m tea.Model, msg tea.Msg) tea.Msg {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return msg
	}
	s := k.String()
	if s != "esc" && s != "q" {
		return msg
	}
	form, ok := m.(*huh.Form)
	if !ok {
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

// RunForm runs a single-step form with Esc and q as cancel keys (see
// cancelKeysFilter). Canceling returns ErrCanceled.
func RunForm(form *huh.Form, out io.Writer, accessible bool) error {
	err := form.
		WithProgramOptions(tea.WithReportFocus(), tea.WithFilter(cancelKeysFilter)).
		WithOutput(out).
		WithAccessible(accessible).
		Run()
	if errors.Is(err, huh.ErrUserAborted) {
		return ErrCanceled
	}
	return err
}
