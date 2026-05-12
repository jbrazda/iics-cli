package dependencies

// ObjectRefsNode represents one object and its metadata objectRefs list.
type ObjectRefsNode struct {
	ID         string
	ObjectRefs []string
}

// PruneDanglingObjectRefs removes objectRefs that do not exist in the selected
// object ID set. It returns pruned refs by object ID and the total number of
// removed dangling refs.
func PruneDanglingObjectRefs(nodes []ObjectRefsNode) (map[string][]string, int) {
	selected := make(map[string]bool, len(nodes))
	for _, n := range nodes {
		if n.ID != "" {
			selected[n.ID] = true
		}
	}

	out := make(map[string][]string, len(nodes))
	removed := 0
	for _, n := range nodes {
		seen := make(map[string]bool)
		pruned := make([]string, 0, len(n.ObjectRefs))
		for _, ref := range n.ObjectRefs {
			if !selected[ref] {
				removed++
				continue
			}
			if seen[ref] {
				continue
			}
			seen[ref] = true
			pruned = append(pruned, ref)
		}
		out[n.ID] = pruned
	}
	return out, removed
}

// CountDanglingObjectRefs counts refs that do not point to an object present in
// the selected ID set.
func CountDanglingObjectRefs(nodes []ObjectRefsNode) int {
	selected := make(map[string]bool, len(nodes))
	for _, n := range nodes {
		if n.ID != "" {
			selected[n.ID] = true
		}
	}
	count := 0
	for _, n := range nodes {
		for _, ref := range n.ObjectRefs {
			if !selected[ref] {
				count++
			}
		}
	}
	return count
}
