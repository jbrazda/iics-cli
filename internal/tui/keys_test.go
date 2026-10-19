package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

func TestParseKey(t *testing.T) {
	tests := []struct {
		in     string
		key    Key
		r      rune
		wantOK bool
		used   int
	}{
		{"\r", KeyEnter, 0, true, 1},
		{"\n", KeyEnter, 0, true, 1},
		{"\x1b", KeyEsc, 0, true, 1},
		{"\x1b[A", 0, 0, false, 3},     // arrow up
		{"\x1b[1;5Ay", 0, 0, false, 6}, // ctrl+arrow, then y
		{"\x1bOP", 0, 0, false, 3},     // F1
		{"\x1bx", 0, 0, false, 2},      // alt+x
		{"\x03", KeyCancel, 0, true, 1},
		{"\x04", KeyCancel, 0, true, 1},
		{"y", KeyRune, 'y', true, 1},
		{"Q", KeyRune, 'Q', true, 1},
		{"é", KeyRune, 'é', true, 2},
		{"\x01", 0, 0, false, 1},
		{"", 0, 0, false, 0},
	}
	for _, tt := range tests {
		k, r, ok, used := parseKey([]byte(tt.in))
		if ok != tt.wantOK || used != tt.used || (ok && (k != tt.key || r != tt.r)) {
			t.Errorf("parseKey(%q) = (%v, %q, %v, %d), want (%v, %q, %v, %d)", tt.in, k, r, ok, used, tt.key, tt.r, tt.wantOK, tt.used)
		}
	}
}

func keyMsg(s string) tea.KeyMsg {
	switch s {
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEscape}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func isCancel(msg tea.Msg) bool {
	k, ok := msg.(tea.KeyMsg)
	return ok && k.Type == tea.KeyCtrlC
}

func newSelectForm() *huh.Form {
	v := 0
	f := huh.NewForm(huh.NewGroup(huh.NewSelect[int]().
		Options(huh.NewOption("alpha", 0), huh.NewOption("beta", 1)).
		Value(&v)))
	f.Init()
	return f
}

func TestCancelKeySelect(t *testing.T) {
	f := newSelectForm()
	if !isCancel(cancelKey(f, keyMsg("esc"), true)) {
		t.Error("esc on a select should cancel")
	}
	if !isCancel(cancelKey(f, keyMsg("q"), true)) {
		t.Error("q on a select should cancel")
	}
	if isCancel(cancelKey(f, keyMsg("j"), true)) {
		t.Error("other keys must pass through")
	}

	// Start filtering: esc and q belong to the filter now.
	_, _ = f.Update(keyMsg("/"))
	if isCancel(cancelKey(f, keyMsg("esc"), true)) {
		t.Error("esc while filtering should close the filter, not cancel")
	}
	if isCancel(cancelKey(f, keyMsg("q"), true)) {
		t.Error("q while filtering should be typed into the filter")
	}
}

func TestCancelKeyInput(t *testing.T) {
	v := ""
	f := huh.NewForm(huh.NewGroup(huh.NewInput().Value(&v)))
	f.Init()
	if !isCancel(cancelKey(f, keyMsg("esc"), true)) {
		t.Error("esc on an input should cancel")
	}
	if isCancel(cancelKey(f, keyMsg("q"), true)) {
		t.Error("q on an input must be typed")
	}
}

func TestCancelKeyConfirm(t *testing.T) {
	v := false
	f := huh.NewForm(huh.NewGroup(huh.NewConfirm().Value(&v)))
	f.Init()
	if !isCancel(cancelKey(f, keyMsg("q"), true)) {
		t.Error("q on a confirm should cancel")
	}
}

func TestCancelKeyWizardIgnoresQ(t *testing.T) {
	f := newSelectForm()
	if isCancel(cancelKey(f, keyMsg("q"), false)) {
		t.Error("q must not cancel a wizard")
	}
	if !isCancel(cancelKey(f, keyMsg("esc"), false)) {
		t.Error("esc should cancel a wizard")
	}
}
