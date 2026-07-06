package dependencies

var cdiCarrierTypes = map[string]bool{
	"DTEMPLATE":     true,
	"MAPPLET":       true,
	"MTT":           true,
	"DSS":           true,
	"DRS":           true,
	"DMASK":         true,
	"FWCONFIG":      true,
	"VISIOTEMPLATE": true,
	"PCS":           true,
	"CustomSource":  true,
	"WORKFLOW":      true,
	"TASKFLOW":      true,
}

var cdiSystemDependencyTypes = map[string]bool{
	"Connection": true,
	"AgentGroup": true,
}

// CDIObjectRefNode represents an exported object with dependency refs needed to
// retain required CDI SYS dependencies in selective package assembly.
type CDIObjectRefNode struct {
	ID         string
	Type       string
	ObjectRefs []string
}

// IncludeCDISysRefsFromObjectRefs adds Connection/AgentGroup refs from selected
// CDI carrier objects into selectedIDs. It ignores unknown refs.
func IncludeCDISysRefsFromObjectRefs(nodes []CDIObjectRefNode, selectedIDs map[string]bool) int {
	if len(nodes) == 0 || len(selectedIDs) == 0 {
		return 0
	}
	byID := make(map[string]CDIObjectRefNode, len(nodes))
	for _, node := range nodes {
		if node.ID == "" {
			continue
		}
		byID[node.ID] = node
	}
	added := 0
	for id := range selectedIDs {
		node, ok := byID[id]
		if !ok || !cdiCarrierTypes[node.Type] {
			continue
		}
		for _, refID := range node.ObjectRefs {
			ref, exists := byID[refID]
			if !exists || !cdiSystemDependencyTypes[ref.Type] {
				continue
			}
			if selectedIDs[refID] {
				continue
			}
			selectedIDs[refID] = true
			added++
		}
	}
	return added
}
