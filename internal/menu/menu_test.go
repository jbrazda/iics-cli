package menu

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jbrazda/iics-cli/internal/tui"
)

func env(vars map[string]string) func(string) string {
	return func(k string) string { return vars[k] }
}

func TestShouldOpen(t *testing.T) {
	off := false
	on := true
	tests := []struct {
		name string
		in   OpenInput
		want bool
	}{
		{"terminal", OpenInput{StdinTTY: true, StdoutTTY: true}, true},
		{"stdout piped", OpenInput{StdinTTY: true}, false},
		{"stdin piped", OpenInput{StdoutTTY: true}, false},
		{"CI", OpenInput{StdinTTY: true, StdoutTTY: true, Env: env(map[string]string{"CI": "true"})}, false},
		{"opt-out env", OpenInput{StdinTTY: true, StdoutTTY: true, Env: env(map[string]string{"IICS_NO_MENU": "1"})}, false},
		{"opt-out env false", OpenInput{StdinTTY: true, StdoutTTY: true, Env: env(map[string]string{"IICS_NO_MENU": "0"})}, true},
		{"config off", OpenInput{StdinTTY: true, StdoutTTY: true, ConfigMenu: &off}, false},
		{"config on", OpenInput{StdinTTY: true, StdoutTTY: true, ConfigMenu: &on}, true},
	}
	for _, tt := range tests {
		if got := ShouldOpen(tt.in); got != tt.want {
			t.Errorf("%s: ShouldOpen = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestCommandArgsAndLine(t *testing.T) {
	e := Entry{Args: []string{"role", "edit"}}
	if got := e.CommandArgs("dev", ""); !reflect.DeepEqual(got, []string{"role", "edit", "--profile", "dev"}) {
		t.Errorf("CommandArgs = %v", got)
	}
	add := Entry{Args: []string{"profile", "add"}, AskArg: "Profile name", NoProfileFlag: true}
	if got := add.CommandArgs("dev", "qa eu"); !reflect.DeepEqual(got, []string{"profile", "add", "qa eu"}) {
		t.Errorf("CommandArgs = %v", got)
	}
	edit := Entry{Args: []string{"profile", "edit"}, ActiveProfileArg: true, NoProfileFlag: true}
	if got := edit.CommandArgs("dev", ""); !reflect.DeepEqual(got, []string{"profile", "edit", "dev"}) {
		t.Errorf("CommandArgs = %v", got)
	}
	if got := CommandLine("iics", []string{"profile", "add", "qa eu", "it's"}); got != `iics profile add 'qa eu' 'it'\''s'` {
		t.Errorf("CommandLine = %s", got)
	}
}

func TestDefaultEntriesAreRunnable(t *testing.T) {
	seen := map[string]bool{}
	for _, e := range DefaultEntries() {
		if e.Group == "" || e.Label == "" || e.Description == "" {
			t.Errorf("entry missing text: %+v", e)
		}
		if e.Action == "" && len(e.Args) == 0 {
			t.Errorf("entry %q has neither args nor action", e.Label)
		}
		if seen[e.Label] {
			t.Errorf("duplicate label %q", e.Label)
		}
		seen[e.Label] = true
	}
}

func key(s string) tea.KeyMsg {
	switch s {
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func press(m *Model, keys ...string) {
	for _, k := range keys {
		m.Update(key(k))
	}
}

var testEntries = []Entry{
	{Group: "Session", Label: "Switch profile", Description: "d", Action: ActionSwitchProfile, SessionOnly: true},
	{Group: "Session", Label: "Add profile", Description: "d", Args: []string{"profile", "add"}, AskArg: "Profile name", NoProfileFlag: true, SessionOnly: true, Writes: true},
	{Group: "Roles", Label: "List roles", Description: "d", Args: []string{"role", "list"}},
	{Group: "Roles", Label: "Edit role privileges", Description: "privilege grid", Args: []string{"role", "edit"}, Writes: true},
}

func TestModelNavigationAndRun(t *testing.T) {
	m := NewModel(testEntries, Header{Profile: "dev", Production: true}, true, "", true)
	view := m.View()
	for _, want := range []string{"PRODUCTION", "profile dev", "Session", "Roles", "Edit role privileges ✎", "$ iics --profile dev"} {
		if want == "$ iics --profile dev" {
			continue // first entry is an action without a command line
		}
		if !strings.Contains(view, want) {
			t.Errorf("view missing %q:\n%s", want, view)
		}
	}
	// Cursor starts on the first entry (headers are skipped).
	press(m, "down", "down", "down") // Add profile, List roles, Edit role privileges
	press(m, "enter")
	if !m.Done() || m.Result().Kind != ResultRun || m.Result().Entry.Label != "Edit role privileges" {
		t.Fatalf("expected run of Edit role privileges, got %+v", m.Result())
	}
	if !strings.Contains(m.View(), "$ iics role edit --profile dev") {
		t.Errorf("equivalent command not shown:\n%s", m.View())
	}
}

func TestModelFilterLastAndKeys(t *testing.T) {
	m := NewModel(testEntries, Header{Profile: "dev"}, true, "List roles", true)
	press(m, "enter")
	if m.Result().Entry.Label != "List roles" {
		t.Errorf("cursor should start on the last entry, got %+v", m.Result())
	}

	m = NewModel(testEntries, Header{Profile: "dev"}, true, "", true)
	press(m, "/", "g", "r", "i", "d", "enter", "enter")
	if m.Result().Entry.Label != "Edit role privileges" {
		t.Errorf("filter on description should select Edit role privileges, got %+v", m.Result())
	}

	m = NewModel(testEntries, Header{}, true, "", true)
	press(m, "p")
	if m.Result().Kind != ResultSwitchProfile {
		t.Errorf("p should switch profile, got %+v", m.Result())
	}

	m = NewModel(testEntries, Header{}, true, "", true)
	press(m, "q")
	if m.Result().Kind != ResultQuit {
		t.Errorf("q should quit, got %+v", m.Result())
	}
}

func TestModelDisabledWithoutProfile(t *testing.T) {
	m := NewModel(testEntries, Header{Notice: "No profile configured"}, false, "List roles", true)
	press(m, "enter")
	if m.Done() {
		t.Fatal("entries needing a profile must not run without one")
	}
	if !strings.Contains(m.View(), "Add a profile first") || !strings.Contains(m.View(), "No profile configured") {
		t.Errorf("expected notice and message:\n%s", m.View())
	}
}

// fakeRunner records command lines and returns scripted exit codes.
type fakeRunner struct {
	calls [][]string
	codes []int
}

func (f *fakeRunner) Run(args []string) int {
	f.calls = append(f.calls, args)
	if len(f.codes) == 0 {
		return 0
	}
	c := f.codes[0]
	f.codes = f.codes[1:]
	return c
}

type loopHarness struct {
	deps    Deps
	runner  *fakeRunner
	out     *bytes.Buffer
	results []Result
	states  map[string]State
	def     string
}

func newHarness(input string, results []Result, states map[string]State) *loopHarness {
	h := &loopHarness{runner: &fakeRunner{}, out: &bytes.Buffer{}, results: results, states: states}
	h.deps = Deps{
		Runner:   h.runner,
		Prompter: &tui.Prompter{In: strings.NewReader(input), Out: h.out},
		Out:      h.out,
		Binary:   "iics",
		NoColor:  true,
		Entries:  testEntries,
		LoadState: func(p string) (State, error) {
			st, ok := h.states[p]
			if !ok {
				st = h.states[""]
			}
			st.Header.Profile = p
			return st, nil
		},
		SetDefault: func(p string) error { h.def = p; return nil },
		ShowMenu: func(m *Model) (Result, error) {
			if len(h.results) == 0 {
				return Result{Kind: ResultQuit}, nil
			}
			r := h.results[0]
			h.results = h.results[1:]
			return r, nil
		},
		Pause: func() (bool, error) { return false, nil },
	}
	return h
}

var twoProfiles = []ProfileInfo{{Name: "dev"}, {Name: "prd", Production: true}}

func TestLoopRunsAndReturns(t *testing.T) {
	st := State{HasProfile: true, Profiles: twoProfiles}
	h := newHarness("", []Result{
		{Kind: ResultRun, Entry: testEntries[2]},
		{Kind: ResultRun, Entry: testEntries[3]},
	}, map[string]State{"dev": st})
	h.runner.codes = []int{0, 3}
	if err := Run(h.deps, "dev"); err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"role", "list", "--profile", "dev"}, {"role", "edit", "--profile", "dev"}}
	if !reflect.DeepEqual(h.runner.calls, want) {
		t.Errorf("calls = %v, want %v", h.runner.calls, want)
	}
	out := h.out.String()
	for _, s := range []string{"$ iics role list --profile dev", "✓ done", "✗ failed (exit 3)"} {
		if !strings.Contains(out, s) {
			t.Errorf("output missing %q:\n%s", s, out)
		}
	}
}

func TestLoopProductionConfirmation(t *testing.T) {
	prod := State{HasProfile: true, Profiles: twoProfiles, Header: Header{Production: true}}
	// Declined write, then a read (never asks), then an accepted write.
	h := newHarness("n\ny\n", []Result{
		{Kind: ResultRun, Entry: testEntries[3]},
		{Kind: ResultRun, Entry: testEntries[2]},
		{Kind: ResultRun, Entry: testEntries[3]},
	}, map[string]State{"prd": prod})
	if err := Run(h.deps, "prd"); err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"role", "list", "--profile", "prd"}, {"role", "edit", "--profile", "prd"}}
	if !reflect.DeepEqual(h.runner.calls, want) {
		t.Errorf("calls = %v, want %v", h.runner.calls, want)
	}
	if n := strings.Count(h.out.String(), "against PRODUCTION"); n != 2 {
		t.Errorf("expected 2 production confirmations, got %d:\n%s", n, h.out.String())
	}
}

func TestLoopSwitchProfileAndSetDefault(t *testing.T) {
	states := map[string]State{
		"dev": {HasProfile: true, Profiles: twoProfiles},
		"prd": {HasProfile: true, Profiles: twoProfiles, Header: Header{Production: true}},
	}
	setDefault := Entry{Group: "Session", Label: "Set as default profile", Description: "d", Action: ActionSetDefault, Writes: true}
	// Switch to profile 2 (prd), run a read, set default (confirm y on prd).
	h := newHarness("2\ny\n", []Result{
		{Kind: ResultSwitchProfile},
		{Kind: ResultRun, Entry: testEntries[2]},
		{Kind: ResultRun, Entry: setDefault},
	}, states)
	if err := Run(h.deps, "dev"); err != nil {
		t.Fatal(err)
	}
	if len(h.runner.calls) != 1 || h.runner.calls[0][3] != "prd" {
		t.Errorf("command should run with the switched profile, got %v", h.runner.calls)
	}
	if h.def != "prd" {
		t.Errorf("default profile = %q, want prd", h.def)
	}
}

func TestLoopFirstRunAddsProfile(t *testing.T) {
	states := map[string]State{
		"":    {},
		"qa":  {HasProfile: true, Profiles: []ProfileInfo{{Name: "qa"}}},
		"dev": {},
	}
	h := newHarness("qa\n", nil, states)
	if err := Run(h.deps, "dev"); err != nil {
		t.Fatal(err)
	}
	if len(h.runner.calls) != 1 || !reflect.DeepEqual(h.runner.calls[0], []string{"profile", "add", "qa"}) {
		t.Errorf("first run should add a profile, got %v", h.runner.calls)
	}
}

func TestLoopQuitFromPause(t *testing.T) {
	h := newHarness("", []Result{{Kind: ResultRun, Entry: testEntries[2]}, {Kind: ResultRun, Entry: testEntries[2]}},
		map[string]State{"dev": {HasProfile: true}})
	h.deps.Pause = func() (bool, error) { return true, nil }
	if err := Run(h.deps, "dev"); err != nil {
		t.Fatal(err)
	}
	if len(h.runner.calls) != 1 {
		t.Errorf("q at the pause should quit after one command, got %d calls", len(h.runner.calls))
	}
}

func TestLineMenu(t *testing.T) {
	// Numbered menu: without a profile only session entries (1, 2) are listed.
	h := newHarness("2\nqa\n0\n", nil, map[string]State{"dev": {Profiles: twoProfiles}, "qa": {HasProfile: true}})
	h.deps.Line = true
	if err := Run(h.deps, "dev"); err != nil {
		t.Fatal(err)
	}
	if len(h.runner.calls) != 1 || !reflect.DeepEqual(h.runner.calls[0], []string{"profile", "add", "qa"}) {
		t.Errorf("calls = %v", h.runner.calls)
	}
	first, _, _ := strings.Cut(h.out.String(), "$ iics profile add")
	if strings.Contains(first, "List roles") {
		t.Errorf("entries needing a profile must be hidden without one:\n%s", first)
	}
	if !strings.Contains(h.out.String()[len(first):], "List roles") {
		t.Error("after adding a profile all entries should be listed")
	}
}
