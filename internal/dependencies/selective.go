package dependencies

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jbrazda/iics-cli/internal/client"
)

// ExportedObjectRef is the minimal object metadata needed for selective packaging.
type ExportedObjectRef struct {
	ObjectGUID string
	ObjectName string
	ObjectType string
	Path       string
}

func fullObjectPath(path, objectName string) string {
	base := strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(path), "/"), "/")
	if base == "" {
		return objectName
	}
	if objectName == "" {
		return base
	}
	return base + "/" + objectName
}

func pathTypeKey(path, objectType string) string {
	base := strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(path), "/"), "/")
	return base + "\x1f" + strings.TrimSpace(objectType)
}

func hasLocationRoot(path string) bool {
	return strings.HasPrefix(path, "Explore/") || strings.HasPrefix(path, "SYS/")
}

func parentPaths(fullPath string) []string {
	parts := strings.Split(strings.Trim(fullPath, "/"), "/")
	if len(parts) <= 1 {
		return nil
	}
	out := make([]string, 0, len(parts)-1)
	for i := len(parts) - 1; i >= 1; i-- {
		out = append(out, strings.Join(parts[:i], "/"))
	}
	return out
}

// IncludeParentContainers adds parent Project/Folder objects inferred from
// already-selected object paths. It mutates selectedIDs and returns the number
// of newly added parent container IDs.
func IncludeParentContainers(objects []ExportedObjectRef, selectedIDs map[string]bool) int {
	if len(selectedIDs) == 0 || len(objects) == 0 {
		return 0
	}

	byID := make(map[string]ExportedObjectRef, len(objects))
	containerIDsByPath := make(map[string][]string)
	for _, o := range objects {
		if o.ObjectGUID == "" || o.ObjectName == "" || o.ObjectType == "" {
			continue
		}
		byID[o.ObjectGUID] = o
		if o.ObjectType == "Project" || o.ObjectType == "Folder" {
			p := fullObjectPath(o.Path, o.ObjectName)
			containerIDsByPath[p] = append(containerIDsByPath[p], o.ObjectGUID)
		}
	}

	required := make(map[string]bool)
	for id := range selectedIDs {
		o, ok := byID[id]
		if !ok {
			continue
		}
		for _, parent := range parentPaths(fullObjectPath(o.Path, o.ObjectName)) {
			required[parent] = true
		}
	}

	added := 0
	for path := range required {
		for _, id := range containerIDsByPath[path] {
			if !selectedIDs[id] {
				selectedIDs[id] = true
				added++
			}
		}
	}
	return added
}

// SelectExportedObjects resolves manifest entries to exported objects.
// It supports direct object references and container expansion for Project/Folder.
func SelectExportedObjects(entries []client.ArtifactEntry, objects []ExportedObjectRef) (map[string]bool, []string, error) {
	byID := make(map[string]ExportedObjectRef, len(objects))
	byPathType := make(map[string][]ExportedObjectRef, len(objects))
	allByPath := make(map[string][]ExportedObjectRef, len(objects))
	fullPathByID := make(map[string]string, len(objects))
	locationByID := make(map[string]string, len(objects))

	for _, o := range objects {
		if o.ObjectGUID == "" || o.ObjectName == "" || o.ObjectType == "" {
			continue
		}
		byID[o.ObjectGUID] = o
		fp := fullObjectPath(o.Path, o.ObjectName)
		fullPathByID[o.ObjectGUID] = fp
		locationByID[o.ObjectGUID] = client.BuildLocation(fp, o.ObjectType)
		canonicalPath := strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(fp), "/"), "/")
		normalizedPath := client.NormalizeLocationPath(canonicalPath)
		byPathType[pathTypeKey(canonicalPath, o.ObjectType)] = append(byPathType[pathTypeKey(canonicalPath, o.ObjectType)], o)
		allByPath[canonicalPath] = append(allByPath[canonicalPath], o)
		if normalizedPath != canonicalPath {
			byPathType[pathTypeKey(normalizedPath, o.ObjectType)] = append(byPathType[pathTypeKey(normalizedPath, o.ObjectType)], o)
			allByPath[normalizedPath] = append(allByPath[normalizedPath], o)
		}
	}

	selected := make(map[string]bool)
	warnings := make([]string, 0)
	hasExplicitObjectRef := false
	seenManifestIDs := make(map[string]bool)
	seenManifestPathTypes := make(map[string]bool)
	seenManifestPaths := make(map[string]bool)

	add := func(id, reason string, duplicateManifestSelector bool) {
		if id == "" {
			return
		}
		if selected[id] {
			if duplicateManifestSelector {
				warnings = append(warnings, fmt.Sprintf("duplicate manifest selection ignored for %s (%s)", id, reason))
			}
			return
		}
		selected[id] = true
	}

	expandContainer := func(containerPath string) {
		containerPath = strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(containerPath), "/"), "/")
		containerNormalized := client.NormalizeLocationPath(containerPath)
		containerHasRoot := hasLocationRoot(containerPath)
		for id, full := range fullPathByID {
			if containerHasRoot {
				if full == containerPath || strings.HasPrefix(full, containerPath+"/") {
					add(id, "container expansion", false)
				}
				continue
			}
			fullNormalized := client.NormalizeLocationPath(full)
			if fullNormalized == containerNormalized || strings.HasPrefix(fullNormalized, containerNormalized+"/") {
				add(id, "container expansion", false)
			}
		}
	}

	for idx, e := range entries {
		entryLabel := fmt.Sprintf("entry %d", idx+1)
		if e.ID != "" {
			duplicateManifestSelector := seenManifestIDs[e.ID]
			seenManifestIDs[e.ID] = true
			obj, ok := byID[e.ID]
			if !ok {
				return nil, nil, fmt.Errorf("manifest %s: id %q not found in source exportMetadata.v2.json", entryLabel, e.ID)
			}
			add(obj.ObjectGUID, "id", duplicateManifestSelector)
			if obj.ObjectType == "Project" || obj.ObjectType == "Folder" {
				expandContainer(fullPathByID[obj.ObjectGUID])
			} else {
				hasExplicitObjectRef = true
			}
			continue
		}

		path := strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(e.Path), "/"), "/")
		if path == "" {
			return nil, nil, fmt.Errorf("manifest %s: missing path", entryLabel)
		}
		typ := strings.TrimSpace(e.Type)

		if typ != "" {
			pathType := pathTypeKey(path, typ)
			duplicateManifestSelector := seenManifestPathTypes[pathType]
			seenManifestPathTypes[pathType] = true
			matches := byPathType[pathType]
			if len(matches) == 0 {
				return nil, nil, fmt.Errorf("manifest %s: no exported object for %q.%s", entryLabel, path, typ)
			}
			for _, m := range matches {
				add(m.ObjectGUID, "path+type", duplicateManifestSelector)
			}
			if typ == "Project" || typ == "Folder" {
				expandContainer(path)
			} else {
				hasExplicitObjectRef = true
			}
			continue
		}

		matches := allByPath[path]
		duplicateManifestSelector := seenManifestPaths[path]
		seenManifestPaths[path] = true
		if len(matches) == 0 {
			return nil, nil, fmt.Errorf("manifest %s: no exported object for path %q", entryLabel, path)
		}
		containerOnly := true
		for _, m := range matches {
			add(m.ObjectGUID, "path", duplicateManifestSelector)
			if m.ObjectType == "Project" || m.ObjectType == "Folder" {
				expandContainer(path)
			} else {
				containerOnly = false
				hasExplicitObjectRef = true
			}
		}
		if containerOnly {
			expandContainer(path)
		}
	}

	// If manifest references only Project/Folder containers, include SYS-rooted assets.
	if !hasExplicitObjectRef {
		for id, loc := range locationByID {
			if strings.HasPrefix(loc, "SYS/") {
				add(id, "SYS implied by container-only selection", false)
			}
		}
	}

	sort.Strings(warnings)
	return selected, warnings, nil
}
