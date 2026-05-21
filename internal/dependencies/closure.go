package dependencies

// RefClosureNode represents one package object and its in-package references.
type RefClosureNode struct {
	ID   string
	Refs []string
}

// CloneIDSet returns a shallow copy of an ID membership set.
func CloneIDSet(in map[string]bool) map[string]bool {
	out := make(map[string]bool, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// IncludeReferencedClosure expands selectedIDs to include all in-package objects
// reachable through refs from the currently selected set. It mutates selectedIDs
// and returns the count of newly added IDs.
func IncludeReferencedClosure(nodes []RefClosureNode, selectedIDs map[string]bool) int {
	if len(nodes) == 0 || len(selectedIDs) == 0 {
		return 0
	}

	refsByID := make(map[string][]string, len(nodes))
	for _, n := range nodes {
		if n.ID == "" {
			continue
		}
		refsByID[n.ID] = n.Refs
	}

	queue := make([]string, 0, len(selectedIDs))
	for id := range selectedIDs {
		if id != "" {
			queue = append(queue, id)
		}
	}

	added := 0
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		for _, ref := range refsByID[id] {
			if ref == "" {
				continue
			}
			if _, exists := refsByID[ref]; !exists {
				continue
			}
			if selectedIDs[ref] {
				continue
			}
			selectedIDs[ref] = true
			added++
			queue = append(queue, ref)
		}
	}
	return added
}

// AddedIDsAfterClosure returns the set of IDs added by closure expansion,
// without mutating the input selectedIDs set.
func AddedIDsAfterClosure(nodes []RefClosureNode, selectedIDs map[string]bool) map[string]bool {
	expanded := CloneIDSet(selectedIDs)
	_ = IncludeReferencedClosure(nodes, expanded)
	added := make(map[string]bool)
	for id := range expanded {
		if !selectedIDs[id] {
			added[id] = true
		}
	}
	return added
}

// CountSetIntersection counts how many IDs appear in both sets.
func CountSetIntersection(a, b map[string]bool) int {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	if len(a) > len(b) {
		a, b = b, a
	}
	count := 0
	for id := range a {
		if b[id] {
			count++
		}
	}
	return count
}
