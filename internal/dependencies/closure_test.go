package dependencies

import "testing"

func TestIncludeReferencedClosure(t *testing.T) {
	nodes := []RefClosureNode{
		{ID: "task", Refs: []string{"mapping", "conn", "external"}},
		{ID: "mapping", Refs: []string{"conn"}},
		{ID: "conn", Refs: nil},
	}
	selected := map[string]bool{"task": true}

	added := IncludeReferencedClosure(nodes, selected)
	if added != 2 {
		t.Fatalf("expected 2 added nodes, got %d", added)
	}
	for _, id := range []string{"task", "mapping", "conn"} {
		if !selected[id] {
			t.Fatalf("expected selected id %q", id)
		}
	}
	if selected["external"] {
		t.Fatalf("did not expect unknown external id in selected set")
	}
}

func TestIncludeReferencedClosure_NoSelection(t *testing.T) {
	nodes := []RefClosureNode{
		{ID: "a", Refs: []string{"b"}},
		{ID: "b", Refs: nil},
	}
	selected := map[string]bool{}

	if added := IncludeReferencedClosure(nodes, selected); added != 0 {
		t.Fatalf("expected 0 additions, got %d", added)
	}
	if len(selected) != 0 {
		t.Fatalf("expected selected to remain empty")
	}
}
