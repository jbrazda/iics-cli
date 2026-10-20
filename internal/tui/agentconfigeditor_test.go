package tui

import (
	"strings"
	"testing"

	"github.com/jbrazda/iics-cli/internal/client"
)

const connProps = `- key: timeBetweenEvictionRunsMillis
  value: 300000
- key: testOnBorrow
  value: true
- key: url
  value: 'jdbc:postgresql://db:5432/x'
`

func TestQuoteValue(t *testing.T) {
	for _, tc := range []struct {
		in, plain string
		quoted    bool
	}{
		{"'10G'", "10G", true},
		{"''", "", true},
		{"'it''s'", "it's", true},
		{"true", "true", false},
		{"'", "'", false},
	} {
		plain, quoted := unquoteValue(tc.in)
		if plain != tc.plain || quoted != tc.quoted {
			t.Errorf("unquoteValue(%q) = %q, %v", tc.in, plain, quoted)
		}
		if back := quoteValue(plain, quoted); back != tc.in {
			t.Errorf("quoteValue(%q, %v) = %q, want %q", plain, quoted, back, tc.in)
		}
	}
}

func TestEntryListRoundTrip(t *testing.T) {
	l, ok := parseEntryList(connProps)
	if !ok || l.Len() != 3 {
		t.Fatalf("parseEntryList: ok=%v", ok)
	}
	if got := l.Label(1); got != "key=testOnBorrow  value=true" {
		t.Errorf("Label = %q", got)
	}
	if s, err := l.String(); err != nil || s != connProps {
		t.Errorf("unchanged String() = %q, %v", s, err)
	}

	l.Set(0, []string{"timeBetweenEvictionRunsMillis", "60000"})
	l.Add(l.Keys(), []string{"maxTotal", "20"})
	l.Remove(1)
	want := `- key: timeBetweenEvictionRunsMillis
  value: 60000
- key: url
  value: 'jdbc:postgresql://db:5432/x'
- key: maxTotal
  value: 20
`
	if s, err := l.String(); err != nil || s != want {
		t.Errorf("edited String() =\n%s\nwant\n%s", s, want)
	}
}

func TestEntryListQuotesWhenNeeded(t *testing.T) {
	l, _ := parseEntryList("- name: a\n  level: INFO\n")
	l.Set(0, []string{"org.apache: x", "WARN"})
	s, _ := l.String()
	if s != "- name: 'org.apache: x'\n  level: WARN\n" {
		t.Errorf("String() = %q", s)
	}
}

func TestParseEntryListRejects(t *testing.T) {
	for _, v := range []string{"", "'10G'", "true", "- a\n- b\n", "key: value\n", "- key: [1, 2]\n"} {
		if _, ok := parseEntryList(v); ok {
			t.Errorf("parseEntryList(%q) should not be a list", v)
		}
	}
}

func TestPreviewValue(t *testing.T) {
	if got := previewValue(connProps, false, 80); got != "[3 entries] key=timeBetweenEvictionRunsMillis  value=300000" {
		t.Errorf("list preview = %q", got)
	}
	if got := previewValue("secret", true, 60); got != "***" {
		t.Errorf("sensitive preview = %q", got)
	}
	if got := previewValue("", false, 60); got != "(empty)" {
		t.Errorf("empty preview = %q", got)
	}
	if got := previewValue("abcdefghij", false, 5); got != "abcd…" {
		t.Errorf("truncated preview = %q", got)
	}
}

func TestAgentConfigEditorRecord(t *testing.T) {
	e := &agentConfigEditor{
		props: []client.AgentConfigProperty{
			{Service: "process-engine", Type: "jvm", Name: "max-heap", Value: "'10G'", DefaultValue: "'1536M'", Overridden: true},
			{Service: "process-engine", Type: "cluster", Name: "enabled", Value: "false", DefaultValue: "false"},
		},
		pending: map[string]client.AgentConfigChange{},
	}
	e.record(0, client.AgentConfigChange{Value: "'4G'"})
	e.record(1, client.AgentConfigChange{Reset: true}) // already default: no change
	if len(e.pending) != 1 {
		t.Fatalf("pending = %v", e.pending)
	}
	if v, over := e.value(0); v != "'4G'" || !over {
		t.Errorf("value(0) = %q, %v", v, over)
	}
	e.record(0, client.AgentConfigChange{Reset: true})
	if v, over := e.value(0); v != "'1536M'" || over {
		t.Errorf("value(0) after reset = %q, %v", v, over)
	}
	e.record(0, client.AgentConfigChange{Value: "'10G'"}) // back to the current value
	if len(e.pending) != 0 {
		t.Errorf("pending after undo by value = %v", e.pending)
	}
	e.record(1, client.AgentConfigChange{Value: "true"})
	if ch := e.changes(); len(ch) != 1 || ch[0].Property != "process-engine:cluster:enabled" || ch[0].Value != "true" {
		t.Errorf("changes() = %+v", ch)
	}
}

func TestEntryDiff(t *testing.T) {
	newV := `- key: timeBetweenEvictionRunsMillis
  value: 300000
- key: testOnBorrow
  value: false
- key: maxTotal
  value: 20
`
	lines, ok := entryDiff(connProps, newV)
	want := []string{
		"~ key=testOnBorrow  value: true → false",
		"+ key=maxTotal  value=20",
		"- key=url  value=jdbc:postgresql://db:5432/x",
	}
	if !ok || strings.Join(lines, "\n") != strings.Join(want, "\n") {
		t.Errorf("entryDiff =\n%s", strings.Join(lines, "\n"))
	}
	if lines, ok := entryDiff("", "- name: a\n  level: INFO\n"); !ok || len(lines) != 1 || lines[0] != "+ name=a  level=INFO" {
		t.Errorf("entryDiff from empty = %v, %v", lines, ok)
	}
	if _, ok := entryDiff("'10G'", "'4G'"); ok {
		t.Error("scalars are not lists")
	}
}

func TestAgentConfigEditorCustom(t *testing.T) {
	e := &agentConfigEditor{
		props: []client.AgentConfigProperty{
			{Service: "Data_Integration_Server", Type: "TOMCAT_CFG", Name: "maxDTMProcesses", Value: "5", Custom: true},
		},
		pending: map[string]client.AgentConfigChange{},
	}
	e.record(0, client.AgentConfigChange{Value: "6"})
	if ch := e.pending["Data_Integration_Server:TOMCAT_CFG:maxDTMProcesses"]; ch.Value != "6" || ch.Add || ch.Remove {
		t.Errorf("custom value change = %+v", ch)
	}
	e.record(0, client.AgentConfigChange{Reset: true}) // reset on a custom property removes it
	if !e.removing(0) {
		t.Error("reset of a custom property should remove it")
	}

	// A property added in the editor stays an addition when its value changes.
	e.props = append(e.props, client.AgentConfigProperty{Service: "Data_Integration_Server", Type: "PMRDTM_CFG", Name: "New", Custom: true})
	e.pending["Data_Integration_Server:PMRDTM_CFG:New"] = client.AgentConfigChange{Property: "Data_Integration_Server:PMRDTM_CFG:New", Value: "a", Add: true}
	e.record(1, client.AgentConfigChange{Value: "b"})
	if ch := e.pending["Data_Integration_Server:PMRDTM_CFG:New"]; !ch.Add || ch.Value != "b" {
		t.Errorf("added property change = %+v", ch)
	}
	if got := e.changes(); len(got) != 2 || !got[0].Remove || !got[1].Add {
		t.Errorf("changes() = %+v", got)
	}
}
