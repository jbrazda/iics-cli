package dependencies

import "testing"

func TestPruneDanglingObjectRefs(t *testing.T) {
	nodes := []ObjectRefsNode{
		{ID: "A", ObjectRefs: []string{"B", "X", "B"}},
		{ID: "B", ObjectRefs: []string{}},
		{ID: "C", ObjectRefs: []string{"A", "Y"}},
	}

	pruned, removed := PruneDanglingObjectRefs(nodes)
	if removed != 2 {
		t.Fatalf("expected 2 removed refs, got %d", removed)
	}
	if got := len(pruned["A"]); got != 1 || pruned["A"][0] != "B" {
		t.Fatalf("unexpected pruned refs for A: %+v", pruned["A"])
	}
	if got := len(pruned["C"]); got != 1 || pruned["C"][0] != "A" {
		t.Fatalf("unexpected pruned refs for C: %+v", pruned["C"])
	}
}

func TestCountDanglingObjectRefs(t *testing.T) {
	nodes := []ObjectRefsNode{
		{ID: "A", ObjectRefs: []string{"B", "X"}},
		{ID: "B", ObjectRefs: []string{"A"}},
	}
	if got := CountDanglingObjectRefs(nodes); got != 1 {
		t.Fatalf("expected 1 dangling ref, got %d", got)
	}
}
