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

	add := func(id, reason string) {
		if id == "" {
			return
		}
		if selected[id] {
			warnings = append(warnings, fmt.Sprintf("duplicate manifest selection ignored for %s (%s)", id, reason))
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
					add(id, "container expansion")
				}
				continue
			}
			fullNormalized := client.NormalizeLocationPath(full)
			if fullNormalized == containerNormalized || strings.HasPrefix(fullNormalized, containerNormalized+"/") {
				add(id, "container expansion")
			}
		}
	}

	for idx, e := range entries {
		entryLabel := fmt.Sprintf("entry %d", idx+1)
		if e.ID != "" {
			obj, ok := byID[e.ID]
			if !ok {
				return nil, nil, fmt.Errorf("manifest %s: id %q not found in source exportMetadata.v2.json", entryLabel, e.ID)
			}
			add(obj.ObjectGUID, "id")
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
			matches := byPathType[pathTypeKey(path, typ)]
			if len(matches) == 0 {
				return nil, nil, fmt.Errorf("manifest %s: no exported object for %q.%s", entryLabel, path, typ)
			}
			for _, m := range matches {
				add(m.ObjectGUID, "path+type")
			}
			if typ == "Project" || typ == "Folder" {
				expandContainer(path)
			} else {
				hasExplicitObjectRef = true
			}
			continue
		}

		matches := allByPath[path]
		if len(matches) == 0 {
			return nil, nil, fmt.Errorf("manifest %s: no exported object for path %q", entryLabel, path)
		}
		containerOnly := true
		for _, m := range matches {
			add(m.ObjectGUID, "path")
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
				add(id, "SYS implied by container-only selection")
			}
		}
	}

	sort.Strings(warnings)
	return selected, warnings, nil
}
