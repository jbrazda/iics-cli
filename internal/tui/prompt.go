package tui

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"golang.org/x/term"
)

// Prompter asks the user for input. On a terminal it renders huh forms
// (arrow keys, "/" filtering, checkboxes). Otherwise it falls back to
// line-based prompts whose input format scripts may rely on, e.g.
// "1,3,5" for a multi-select.
type Prompter struct {
	In  io.Reader
	Out io.Writer
	// Interactive selects huh forms; false selects line-based prompts.
	Interactive bool
	// Accessible runs huh forms in accessible (screen reader) mode.
	Accessible bool
	// Password reads a secret without echo on the line path. Nil uses
	// golang.org/x/term on os.Stdin.
	Password func() (string, error)

	reader *bufio.Reader
}

// Default returns a Prompter on stdin/stderr. Forms are used when both are
// terminals; IICS_ACCESSIBLE=1 enables accessible mode.
func Default() *Prompter {
	accessible, _ := strconv.ParseBool(os.Getenv("IICS_ACCESSIBLE"))
	return &Prompter{
		In:          os.Stdin,
		Out:         os.Stderr,
		Interactive: term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stderr.Fd())),
		Accessible:  accessible,
	}
}

// line returns the shared buffered reader. One reader per Prompter keeps
// piped answers for later prompts from being lost in a discarded buffer.
func (p *Prompter) line() (string, error) {
	if p.reader == nil {
		p.reader = bufio.NewReader(p.In)
	}
	s, err := p.reader.ReadString('\n')
	if err != nil && (s == "" || !errors.Is(err, io.EOF)) {
		return "", err
	}
	return strings.TrimSpace(s), nil
}

func (p *Prompter) run(fields ...huh.Field) error {
	err := huh.NewForm(huh.NewGroup(fields...)).
		WithOutput(p.Out).
		WithAccessible(p.Accessible).
		Run()
	if errors.Is(err, huh.ErrUserAborted) {
		return ErrCanceled
	}
	return err
}

func listHeight(n int) int {
	return min(n+2, 20)
}

// Text prompts for a line of text. An empty answer returns def.
func (p *Prompter) Text(label, def string) (string, error) {
	if p.Interactive {
		v := def
		if err := p.run(huh.NewInput().Title(label).Value(&v)); err != nil {
			return "", err
		}
		v = strings.TrimSpace(v)
		if v == "" {
			return def, nil
		}
		return v, nil
	}
	if def != "" {
		_, _ = fmt.Fprintf(p.Out, "%s [%s]: ", label, def)
	} else {
		_, _ = fmt.Fprintf(p.Out, "%s: ", label)
	}
	s, err := p.line()
	if err != nil {
		return "", fmt.Errorf("reading input: %w", err)
	}
	if s == "" {
		return def, nil
	}
	return s, nil
}

// Secret prompts for a password without echoing it.
func (p *Prompter) Secret(label string) (string, error) {
	if p.Interactive {
		var v string
		label = strings.TrimSuffix(strings.TrimSpace(label), ":")
		if err := p.run(huh.NewInput().Title(label).EchoMode(huh.EchoModePassword).Value(&v)); err != nil {
			return "", err
		}
		return v, nil
	}
	_, _ = fmt.Fprint(p.Out, label)
	read := p.Password
	if read == nil {
		read = func() (string, error) {
			pw, err := term.ReadPassword(int(os.Stdin.Fd()))
			return string(pw), err
		}
	}
	pw, err := read()
	_, _ = fmt.Fprintln(p.Out)
	if err != nil {
		return "", fmt.Errorf("reading password: %w", err)
	}
	return pw, nil
}

// Select presents a menu and returns the 0-based index of the chosen option,
// or -1 if the user cancels.
func (p *Prompter) Select(label string, options []string) (int, error) {
	if p.Interactive {
		idx := 0
		opts := make([]huh.Option[int], len(options))
		for i, o := range options {
			opts[i] = huh.NewOption(o, i)
		}
		sel := huh.NewSelect[int]().
			Title(label).
			Description("/ to filter, esc to cancel").
			Options(opts...).
			Height(listHeight(len(options)) + 1).
			Value(&idx)
		if err := p.run(sel); err != nil {
			if errors.Is(err, ErrCanceled) {
				return -1, nil
			}
			return -1, err
		}
		return idx, nil
	}

	_, _ = fmt.Fprintf(p.Out, "%s:\n", label)
	for i, opt := range options {
		_, _ = fmt.Fprintf(p.Out, "  [%d] %s\n", i+1, opt)
	}
	_, _ = fmt.Fprintln(p.Out, "  [0] Cancel")
	for {
		_, _ = fmt.Fprint(p.Out, "Selection (q to cancel): ")
		s, err := p.line()
		if err != nil {
			return -1, fmt.Errorf("reading selection: %w", err)
		}
		if s == "0" || s == "" || strings.EqualFold(s, "q") {
			return -1, nil
		}
		n, convErr := strconv.Atoi(s)
		if convErr != nil || n < 1 || n > len(options) {
			_, _ = fmt.Fprintf(p.Out, "Enter a number between 0 and %d, or 'q' to cancel.\n", len(options))
			continue
		}
		return n - 1, nil
	}
}

// MultiSelect presents a checklist with defaults pre-selected and returns
// the 0-based indices of the selected options (nil for none). Canceling
// returns ErrCanceled.
func (p *Prompter) MultiSelect(label string, options []string, defaults []int) ([]int, error) {
	defaultSet := make(map[int]bool, len(defaults))
	for _, d := range defaults {
		defaultSet[d] = true
	}

	if p.Interactive {
		opts := make([]huh.Option[int], len(options))
		for i, o := range options {
			opts[i] = huh.NewOption(o, i).Selected(defaultSet[i])
		}
		var picked []int
		ms := huh.NewMultiSelect[int]().
			Title(label).
			Description("space to toggle, / to filter, ctrl+a all, enter to confirm").
			Options(opts...).
			Filterable(true).
			Height(listHeight(len(options)) + 1).
			Value(&picked)
		if err := p.run(ms); err != nil {
			return nil, err
		}
		if len(picked) == 0 {
			return nil, nil
		}
		return picked, nil
	}

	_, _ = fmt.Fprintf(p.Out, "%s (comma-separated numbers, 0 = none):\n", label)
	for i, opt := range options {
		mark := " "
		if defaultSet[i] {
			mark = "*"
		}
		_, _ = fmt.Fprintf(p.Out, "  [%d]%s %s\n", i+1, mark, opt)
	}
	if len(defaults) > 0 {
		nums := make([]string, len(defaults))
		for i, d := range defaults {
			nums[i] = strconv.Itoa(d + 1)
		}
		_, _ = fmt.Fprintf(p.Out, "Selections [%s]: ", strings.Join(nums, ","))
	} else {
		_, _ = fmt.Fprint(p.Out, "Selections: ")
	}

	s, err := p.line()
	if err != nil {
		return nil, fmt.Errorf("reading selections: %w", err)
	}
	if s == "" && len(defaults) > 0 {
		return defaults, nil
	}
	if s == "0" || s == "" {
		return nil, nil
	}

	parts := strings.Split(s, ",")
	selected := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		n, convErr := strconv.Atoi(part)
		if convErr != nil || n < 0 || n > len(options) {
			return nil, fmt.Errorf("invalid selection %q; enter numbers between 0 and %d", part, len(options))
		}
		if n == 0 {
			return nil, nil
		}
		selected = append(selected, n-1)
	}
	return selected, nil
}

// Confirm asks a yes/no question. defaultYes is returned for an empty answer.
func (p *Prompter) Confirm(label string, defaultYes bool) (bool, error) {
	if p.Interactive {
		v := defaultYes
		if err := p.run(huh.NewConfirm().Title(label).Affirmative("Yes").Negative("No").Value(&v)); err != nil {
			return false, err
		}
		return v, nil
	}
	hint := "[Y/n]"
	if !defaultYes {
		hint = "[y/N]"
	}
	_, _ = fmt.Fprintf(p.Out, "%s %s: ", label, hint)
	s, err := p.line()
	if err != nil {
		return false, fmt.Errorf("reading input: %w", err)
	}
	s = strings.ToLower(s)
	if s == "" {
		return defaultYes, nil
	}
	return s == "y" || s == "yes", nil
}

// Timezone picks a time zone from zones. current is pre-selected; choosing
// "(none)" returns "".
func (p *Prompter) Timezone(zones []string, current string) (string, error) {
	if p.Interactive {
		v := current
		opts := make([]huh.Option[string], 0, len(zones)+1)
		opts = append(opts, huh.NewOption("(none)", ""))
		for _, z := range zones {
			opts = append(opts, huh.NewOption(z, z))
		}
		sel := huh.NewSelect[string]().
			Title("Time Zone").
			Description("/ to filter, e.g. New_York or Europe").
			Options(opts...).
			Height(12).
			Value(&v)
		if err := p.run(sel); err != nil {
			return "", err
		}
		return v, nil
	}

	for {
		if current != "" {
			_, _ = fmt.Fprintf(p.Out, "Time Zone ID [%s] (type to search, Enter to keep, 0 to clear): ", current)
		} else {
			_, _ = fmt.Fprint(p.Out, "Time Zone ID (type to search, e.g. New_York, Europe, or 0 to skip): ")
		}
		s, err := p.line()
		if err != nil {
			return "", fmt.Errorf("reading input: %w", err)
		}
		if s == "" {
			return current, nil
		}
		if s == "0" {
			return "", nil
		}

		lower := strings.ToLower(s)
		var matches []string
		for _, tz := range zones {
			if strings.Contains(strings.ToLower(tz), lower) {
				matches = append(matches, tz)
			}
		}
		if len(matches) == 0 {
			_, _ = fmt.Fprintf(p.Out, "No matching timezone for %q. Try again.\n", s)
			continue
		}
		if len(matches) == 1 {
			_, _ = fmt.Fprintf(p.Out, "Selected: %s\n", matches[0])
			return matches[0], nil
		}

		_, _ = fmt.Fprintf(p.Out, "Matches for %q:\n", s)
		for i, tz := range matches {
			_, _ = fmt.Fprintf(p.Out, "  [%d] %s\n", i+1, tz)
		}
		_, _ = fmt.Fprint(p.Out, "Selection (0 to search again): ")
		sel, selErr := p.line()
		if selErr != nil {
			return "", fmt.Errorf("reading selection: %w", selErr)
		}
		if sel == "0" || sel == "" {
			continue
		}
		n, convErr := strconv.Atoi(sel)
		if convErr != nil || n < 1 || n > len(matches) {
			_, _ = fmt.Fprintln(p.Out, "Invalid selection. Try again.")
			continue
		}
		return matches[n-1], nil
	}
}

// PickOne lets the user pick one item, labeled by labelFn. It returns the
// index, or -1 if the user cancels.
func PickOne[T any](p *Prompter, label string, items []T, labelFn func(T) string) (int, error) {
	labels := make([]string, len(items))
	for i, it := range items {
		labels[i] = labelFn(it)
	}
	return p.Select(label, labels)
}
