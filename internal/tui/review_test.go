package tui

import (
	"reflect"
	"testing"

	"github.com/jbrazda/iics-cli/internal/client"
)

func TestSetChanges(t *testing.T) {
	got := SetChanges("Role", []string{"B", "A", "C"}, []string{"C", "D", "B", "E"})
	want := []string{"+ Role: D", "+ Role: E", "- Role: A"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("SetChanges = %v, want %v", got, want)
	}
	if got := SetChanges("Role", []string{"A"}, []string{"A"}); got != nil {
		t.Errorf("expected no changes, got %v", got)
	}
}

func TestRequiredUnique(t *testing.T) {
	v := requiredUnique("group name", []string{"Data Engineering"})
	if err := v("  "); err == nil {
		t.Error("empty name should fail")
	}
	if err := v("data engineering"); err == nil {
		t.Error("existing name (case-insensitive) should fail")
	}
	if err := v("Data Science"); err != nil {
		t.Errorf("new name should pass: %v", err)
	}
}

func TestAtLeastOne(t *testing.T) {
	v := atLeastOne("role")
	if err := v(nil); err == nil {
		t.Error("empty selection should fail")
	}
	if err := v([]string{"r1"}); err != nil {
		t.Errorf("one selection should pass: %v", err)
	}
}

func TestAgentItems(t *testing.T) {
	available := []client.Agent{
		{ID: "a2", Name: "zeta", AgentHost: "host2", Active: true},
		{ID: "a1", Name: "Alpha"},
		{ID: "a1", Name: "Alpha"}, // duplicate
	}
	seeded := []client.RuntimeEnvironmentAgent{{ID: "a2"}, {ID: "a3", Name: "Mid"}, {ID: "a4"}}
	got := agentItems(available, seeded)

	var labels []string
	for _, it := range got {
		labels = append(labels, it.label)
	}
	want := []string{"a4 (from file)", "Alpha - inactive", "Mid (from file)", "zeta (host2) - active"}
	if !reflect.DeepEqual(labels, want) {
		t.Errorf("labels = %v, want %v", labels, want)
	}
}
