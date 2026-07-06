package dependencies

import "testing"

func TestIncludeCDISysRefsFromObjectRefs(t *testing.T) {
	nodes := []CDIObjectRefNode{
		{ID: "mtt", Type: "MTT", ObjectRefs: []string{"conn", "agent", "other"}},
		{ID: "conn", Type: "Connection"},
		{ID: "agent", Type: "AgentGroup"},
		{ID: "other", Type: "PROCESS"},
	}
	selected := map[string]bool{"mtt": true}

	added := IncludeCDISysRefsFromObjectRefs(nodes, selected)
	if added != 2 {
		t.Fatalf("expected 2 added refs, got %d", added)
	}
	if !selected["conn"] || !selected["agent"] {
		t.Fatalf("expected connection and agent refs to be selected: %#v", selected)
	}
	if selected["other"] {
		t.Fatalf("did not expect non-sys ref selected")
	}
}

func TestIncludeCDISysRefsFromObjectRefs_NonCDIType(t *testing.T) {
	nodes := []CDIObjectRefNode{
		{ID: "proc", Type: "PROCESS", ObjectRefs: []string{"conn"}},
		{ID: "conn", Type: "Connection"},
	}
	selected := map[string]bool{"proc": true}

	added := IncludeCDISysRefsFromObjectRefs(nodes, selected)
	if added != 0 {
		t.Fatalf("expected no additions, got %d", added)
	}
	if selected["conn"] {
		t.Fatalf("did not expect connection selected for non-CDI carrier")
	}
}
