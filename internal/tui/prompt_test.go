package tui

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func linePrompter(input string) (*Prompter, *bytes.Buffer) {
	out := &bytes.Buffer{}
	return &Prompter{In: strings.NewReader(input), Out: out}, out
}

func TestLineSelect(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"2\n", 1},
		{"q\n", -1},
		{"\n", -1},
		{"0\n", -1},
		{"9\n1\n", 0}, // invalid, then valid
	}
	for _, tt := range tests {
		p, _ := linePrompter(tt.input)
		got, err := p.Select("Pick", []string{"a", "b", "c"})
		if err != nil {
			t.Fatalf("Select(%q) error: %v", tt.input, err)
		}
		if got != tt.want {
			t.Errorf("Select(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestLineMultiSelect(t *testing.T) {
	opts := []string{"a", "b", "c"}
	tests := []struct {
		input    string
		defaults []int
		want     []int
	}{
		{"1,3\n", nil, []int{0, 2}},
		{" 2 , 1 \n", nil, []int{1, 0}},
		{"\n", []int{1}, []int{1}},
		{"0\n", []int{1}, nil},
		{"\n", nil, nil},
	}
	for _, tt := range tests {
		p, out := linePrompter(tt.input)
		got, err := p.MultiSelect("Pick", opts, tt.defaults)
		if err != nil {
			t.Fatalf("MultiSelect(%q) error: %v", tt.input, err)
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("MultiSelect(%q, %v) = %v, want %v", tt.input, tt.defaults, got, tt.want)
		}
		if tt.defaults != nil && !strings.Contains(out.String(), "[2]* b") {
			t.Errorf("defaults should be marked with *:\n%s", out.String())
		}
	}

	p, _ := linePrompter("1,x\n")
	if _, err := p.MultiSelect("Pick", opts, nil); err == nil {
		t.Error("expected error for invalid selection")
	}
}

func TestLineTextAndConfirm(t *testing.T) {
	p, out := linePrompter("\nAlice\n\nyes\nn\n")
	if v, _ := p.Text("Name", "Bob"); v != "Bob" {
		t.Errorf("empty Text should return default, got %q", v)
	}
	if !strings.Contains(out.String(), "Name [Bob]: ") {
		t.Errorf("default should be shown in the prompt: %q", out.String())
	}
	if v, _ := p.Text("Name", "Bob"); v != "Alice" {
		t.Errorf("Text = %q, want Alice", v)
	}
	if v, _ := p.Confirm("Ok?", true); !v {
		t.Error("empty Confirm should return default true")
	}
	if v, _ := p.Confirm("Ok?", false); !v {
		t.Error("Confirm(yes) should be true")
	}
	if v, _ := p.Confirm("Ok?", true); v {
		t.Error("Confirm(n) should be false")
	}
}

func TestLineTimezone(t *testing.T) {
	zones := []string{"America/New_York", "Europe/Paris", "Europe/Prague"}
	tests := []struct {
		input, current, want string
	}{
		{"\n", "UTC", "UTC"},
		{"0\n", "UTC", ""},
		{"york\n", "", "America/New_York"},
		{"europe\n2\n", "", "Europe/Prague"},
		{"mars\nparis\n", "", "Europe/Paris"},
	}
	for _, tt := range tests {
		p, _ := linePrompter(tt.input)
		got, err := p.Timezone(zones, tt.current)
		if err != nil {
			t.Fatalf("Timezone(%q) error: %v", tt.input, err)
		}
		if got != tt.want {
			t.Errorf("Timezone(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// Piped answers for consecutive prompts must all be read from one buffer.
func TestLineSequentialPrompts(t *testing.T) {
	p, _ := linePrompter("Data Team\nDescription here\n2,3\ny\n")
	name, _ := p.Text("Name", "")
	desc, _ := p.Text("Description", "")
	picks, _ := p.MultiSelect("Roles", []string{"a", "b", "c"}, nil)
	ok, _ := p.Confirm("Create?", false)
	if name != "Data Team" || desc != "Description here" || !reflect.DeepEqual(picks, []int{1, 2}) || !ok {
		t.Errorf("got name=%q desc=%q picks=%v ok=%v", name, desc, picks, ok)
	}
}

func TestLineEOF(t *testing.T) {
	p, _ := linePrompter("")
	if _, err := p.Text("Name", ""); err == nil {
		t.Error("expected error on EOF without input")
	}
	p, _ = linePrompter("last")
	if v, err := p.Text("Name", ""); err != nil || v != "last" {
		t.Errorf("final line without newline: got %q, %v", v, err)
	}
}

func TestLineSecret(t *testing.T) {
	p, out := linePrompter("")
	p.Password = func() (string, error) { return "s3cret", nil }
	v, err := p.Secret("Password: ")
	if err != nil || v != "s3cret" {
		t.Errorf("Secret = %q, %v", v, err)
	}
	if !strings.HasPrefix(out.String(), "Password: ") {
		t.Errorf("prompt not written: %q", out.String())
	}
}

func TestPickOne(t *testing.T) {
	type item struct{ name string }
	p, out := linePrompter("2\n")
	idx, err := PickOne(p, "Pick", []item{{"x"}, {"y"}}, func(i item) string { return "item " + i.name })
	if err != nil || idx != 1 {
		t.Errorf("PickOne = %d, %v", idx, err)
	}
	if !strings.Contains(out.String(), "[2] item y") {
		t.Errorf("labels not rendered: %s", out.String())
	}
}
