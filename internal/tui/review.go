package tui

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/charmbracelet/huh"
)

// Review choices returned by confirmReview.
const (
	reviewApply  = "apply"
	reviewBack   = "back"
	reviewCancel = "cancel"
)

// confirmReview shows lines under title and asks whether to apply, go back to
// editing, or cancel. When canApply is false the apply option is omitted
// (for example when nothing changed or a required value is missing).
func confirmReview(out io.Writer, accessible bool, title string, lines []string, applyLabel string, canApply bool) (string, error) {
	var options []huh.Option[string]
	if canApply {
		options = append(options, huh.NewOption(applyLabel, reviewApply))
	}
	options = append(options,
		huh.NewOption("Back to editing", reviewBack),
		huh.NewOption("Cancel", reviewCancel),
	)
	choice := options[0].Value
	sel := huh.NewSelect[string]().
		Title(title).
		Description(strings.Join(lines, "\n")).
		Options(options...).
		Value(&choice)
	err := huh.NewForm(huh.NewGroup(sel)).WithOutput(out).WithAccessible(accessible).Run()
	if errors.Is(err, huh.ErrUserAborted) {
		return reviewCancel, nil
	}
	if err != nil {
		return "", err
	}
	return choice, nil
}

// SetChanges describes membership changes between two sets of names as
// "+ <kind>: name" and "- <kind>: name" lines, additions first, each sorted.
func SetChanges(kind string, before, after []string) []string {
	cur := toSet(before)
	want := toSet(after)
	var add, remove []string
	for n := range want {
		if !cur[n] {
			add = append(add, n)
		}
	}
	for n := range cur {
		if !want[n] {
			remove = append(remove, n)
		}
	}
	sort.Strings(add)
	sort.Strings(remove)
	var out []string
	for _, n := range add {
		out = append(out, fmt.Sprintf("+ %s: %s", kind, n))
	}
	for _, n := range remove {
		out = append(out, fmt.Sprintf("- %s: %s", kind, n))
	}
	return out
}

// requiredUnique validates a non-empty name that is not in existing
// (case-insensitive).
func requiredUnique(what string, existing []string) func(string) error {
	taken := make(map[string]bool, len(existing))
	for _, n := range existing {
		taken[strings.ToLower(n)] = true
	}
	return func(v string) error {
		v = strings.TrimSpace(v)
		if v == "" {
			return fmt.Errorf("%s is required", what)
		}
		if taken[strings.ToLower(v)] {
			return fmt.Errorf("%q already exists", v)
		}
		return nil
	}
}

func atLeastOne(what string) func([]string) error {
	return func(v []string) error {
		if len(v) == 0 {
			return fmt.Errorf("select at least one %s", what)
		}
		return nil
	}
}
