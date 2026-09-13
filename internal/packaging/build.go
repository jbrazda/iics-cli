package packaging

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/dependencies"
	"github.com/jbrazda/iics-cli/internal/release"
)

// BuildOptions configures the selective-export pipeline run by BuildSelectivePackage.
type BuildOptions struct {
	// ExcludeFoundTransitive mirrors the --exclude-found-transitive flag.
	ExcludeFoundTransitive bool
	// PackageName mirrors the --name flag override.
	PackageName string
	// IncludeTags mirrors the --include-tags flag.
	IncludeTags bool
	// Target is the output ZIP path (--target); its basename (without extension) is
	// used to derive the default package name when PackageName is empty, same as today.
	Target string
}

// Result carries everything cmd/package.go needs to write output/log a "package create" run.
type Result struct {
	// Files holds the final package contents (excluding exportPackage.chksum), including
	// the regenerated exportMetadata.v2.json and ContentsofExportPackage_<name>.csv.
	Files map[string][]byte
	// ReportIncluded is the set of selected assets for the manifest log.
	ReportIncluded []release.ManifestLogAsset
	// ReportSelected is the count of selected assets.
	ReportSelected int
	// ReportExcluded is the count of transitive-found rows excluded by
	// --exclude-found-transitive (manifest rows + closure-suppressed).
	ReportExcluded int
	// Warnings are selection warnings (e.g. duplicate manifest selectors) to be printed
	// to stderr as "Warning: <text>", unconditionally (not gated by --verbose).
	Warnings []string
	// Notices are verbose-only progress lines, in emission order, so cmd/ can decide
	// whether to print them under --verbose. Each entry includes its trailing newline.
	Notices []string
}

// BuildSelectivePackage runs the full selective-export pipeline (selection -> parent
// container / referenced-closure / CDI-ref inclusion -> metadata graph -> file
// filtering -> exportMetadata.v2.json / ContentsofExportPackage CSV regeneration)
// over already-collected file contents from a source directory.
//
// manifestEntries/manifestStats must come from parsing the selection manifest
// (see cmd's readPackageSelectionManifest); the caller decides whether a selection
// manifest was supplied at all and only calls this when it was.
func BuildSelectivePackage(
	fileContents map[string][]byte,
	absSource string,
	manifestEntries []client.ArtifactEntry,
	manifestStats dependencies.BuildManifestStats,
	opts BuildOptions,
) (Result, error) {
	var result Result

	if len(manifestEntries) == 0 {
		return result, fmt.Errorf("selection manifest is empty")
	}

	meta, metaErr := ReadExportMetadata("", absSource)
	if metaErr != nil {
		return result, fmt.Errorf("reading source metadata for selective packaging: %w", metaErr)
	}

	exported := make([]dependencies.ExportedObjectRef, 0, len(meta.ExportedObjects))
	for _, o := range meta.ExportedObjects {
		exported = append(exported, dependencies.ExportedObjectRef{
			ObjectGUID: o.ObjectGUID,
			ObjectName: o.ObjectName,
			ObjectType: o.ObjectType,
			Path:       o.Path,
		})
	}
	selectedIDs, warnings, selErr := dependencies.SelectExportedObjects(manifestEntries, exported)
	if selErr != nil {
		return result, selErr
	}
	parentAdded := dependencies.IncludeParentContainers(exported, selectedIDs)
	closureNodes := make([]dependencies.RefClosureNode, 0, len(meta.ExportedObjects))
	for _, o := range meta.ExportedObjects {
		if o.ObjectGUID == "" {
			continue
		}
		closureNodes = append(closureNodes, dependencies.RefClosureNode{
			ID:   o.ObjectGUID,
			Refs: o.ObjectRefs(),
		})
	}
	excludedSelectedIDs := make(map[string]bool)
	if opts.ExcludeFoundTransitive && len(manifestStats.ExcludedEntries) > 0 {
		resolvedExcluded, _, excludedSelErr := dependencies.SelectExportedObjects(manifestStats.ExcludedEntries, exported)
		if excludedSelErr != nil {
			return result, fmt.Errorf("resolving excluded transitive-found entries: %w", excludedSelErr)
		}
		excludedSelectedIDs = resolvedExcluded
	}

	closureAdded := 0
	closureSuppressedExcluded := 0
	if !opts.ExcludeFoundTransitive {
		closureAdded = dependencies.IncludeReferencedClosure(closureNodes, selectedIDs)
	} else {
		closureAddedIDs := dependencies.AddedIDsAfterClosure(closureNodes, selectedIDs)
		closureSuppressedExcluded = len(closureAddedIDs)
		if len(excludedSelectedIDs) > 0 && len(closureAddedIDs) > 0 {
			// Keep a floor count based on all closure-suppressed additions and
			// use excluded-entry overlap to avoid undercounting when manifest rows
			// and closure additions are both present.
			overlap := dependencies.CountSetIntersection(closureAddedIDs, excludedSelectedIDs)
			if overlap > closureSuppressedExcluded {
				closureSuppressedExcluded = overlap
			}
		}
	}
	if len(selectedIDs) == 0 {
		return result, fmt.Errorf("no assets matched selection manifest")
	}

	result.Warnings = warnings

	if opts.ExcludeFoundTransitive {
		totalExcluded := manifestStats.ExcludedTransitiveFound + closureSuppressedExcluded
		result.Notices = append(result.Notices, fmt.Sprintf(
			"Selection filter: excluded %d transitive found rows using %s (manifest rows: %d, closure-suppressed: %d)\n",
			totalExcluded,
			manifestStats.SelectedStatusColumnName,
			manifestStats.ExcludedTransitiveFound,
			closureSuppressedExcluded,
		))
	}
	if parentAdded > 0 {
		result.Notices = append(result.Notices, fmt.Sprintf("Selection refinement: included %d inferred parent Project/Folder objects\n", parentAdded))
	}
	if closureAdded > 0 {
		result.Notices = append(result.Notices, fmt.Sprintf("Selection refinement: included %d in-package referenced dependencies\n", closureAdded))
	}
	cdiRefAdded := 0
	if opts.ExcludeFoundTransitive {
		refNodes := make([]dependencies.CDIObjectRefNode, 0, len(meta.ExportedObjects))
		for _, o := range meta.ExportedObjects {
			if o.ObjectGUID == "" {
				continue
			}
			refNodes = append(refNodes, dependencies.CDIObjectRefNode{
				ID:         o.ObjectGUID,
				Type:       o.ObjectType,
				ObjectRefs: o.ObjectRefs(),
			})
		}
		cdiRefAdded = dependencies.IncludeCDISysRefsFromObjectRefs(refNodes, selectedIDs)
		if cdiRefAdded > 0 {
			result.Notices = append(result.Notices, fmt.Sprintf("Selection refinement: included %d CDI Connection/AgentGroup refs from objectRefs\n", cdiRefAdded))
		}
	}

	selectedObjects := make([]ExportedObject, 0, len(selectedIDs))
	for _, o := range meta.ExportedObjects {
		if selectedIDs[o.ObjectGUID] {
			selectedObjects = append(selectedObjects, o)
		}
	}
	if len(selectedObjects) == 0 {
		return result, fmt.Errorf("selection manifest resolved no exported objects")
	}
	result.ReportIncluded = ExportedObjectsToManifestLog(selectedObjects)
	result.ReportSelected = len(selectedObjects)
	result.ReportExcluded = manifestStats.ExcludedTransitiveFound + closureSuppressedExcluded
	metadataObjects := selectedObjects
	if !opts.ExcludeFoundTransitive {
		selectedNodes := make([]dependencies.ObjectRefsNode, len(selectedObjects))
		for i, o := range selectedObjects {
			selectedNodes[i] = dependencies.ObjectRefsNode{
				ID:         o.ObjectGUID,
				ObjectRefs: o.ObjectRefs(),
			}
		}
		prunedRefsByID, prunedCount := dependencies.PruneDanglingObjectRefs(selectedNodes)
		for i := range selectedObjects {
			if setErr := selectedObjects[i].SetObjectRefs(prunedRefsByID[selectedObjects[i].ObjectGUID]); setErr != nil {
				return result, setErr
			}
		}
		postPruneNodes := make([]dependencies.ObjectRefsNode, len(selectedObjects))
		for i, o := range selectedObjects {
			postPruneNodes[i] = dependencies.ObjectRefsNode{
				ID:         o.ObjectGUID,
				ObjectRefs: o.ObjectRefs(),
			}
		}
		if dangling := dependencies.CountDanglingObjectRefs(postPruneNodes); dangling > 0 {
			return result, fmt.Errorf("selection produced %d unresolved objectRefs after pruning; cannot create import-safe package", dangling)
		}
		if prunedCount > 0 {
			result.Notices = append(result.Notices, fmt.Sprintf("Selection refinement: pruned %d dangling metadata objectRefs\n", prunedCount))
		}
	} else {
		metadataIDs := buildMetadataGraphForSelection(meta.ExportedObjects, selectedIDs, excludedSelectedIDs)
		metadataObjectList := make([]ExportedObject, 0, len(metadataIDs))
		for _, o := range meta.ExportedObjects {
			if metadataIDs[o.ObjectGUID] {
				metadataObjectList = append(metadataObjectList, o)
			}
		}
		if len(metadataObjectList) == 0 {
			return result, fmt.Errorf("selection metadata graph resolved no exported objects")
		}
		metadataNodes := make([]dependencies.ObjectRefsNode, len(metadataObjectList))
		for i, o := range metadataObjectList {
			metadataNodes[i] = dependencies.ObjectRefsNode{
				ID:         o.ObjectGUID,
				ObjectRefs: o.ObjectRefs(),
			}
		}
		prunedRefsByID, _ := dependencies.PruneDanglingObjectRefs(metadataNodes)
		for i := range metadataObjectList {
			if setErr := metadataObjectList[i].SetObjectRefs(prunedRefsByID[metadataObjectList[i].ObjectGUID]); setErr != nil {
				return result, setErr
			}
		}
		metadataObjects = metadataObjectList
	}

	filtered := filterPackageFilesForSelection(fileContents, meta.ExportedObjects, selectedIDs)
	if len(filtered) == 0 {
		return result, fmt.Errorf("no package files remained after selection filtering")
	}

	finalPackageName := opts.PackageName
	if finalPackageName == "" {
		finalPackageName = strings.TrimSuffix(filepath.Base(opts.Target), filepath.Ext(opts.Target))
	}

	meta.Name = finalPackageName
	if !opts.IncludeTags {
		meta.Tags = nil
	}
	meta.ExportedObjects = metadataObjects
	metaData, marshalErr := json.MarshalIndent(meta, "", "  ")
	if marshalErr != nil {
		return result, fmt.Errorf("serializing filtered exportMetadata.v2.json: %w", marshalErr)
	}
	filtered["exportMetadata.v2.json"] = metaData

	contentsCSV, csvErr := buildContentsOfExportPackageCSV(selectedObjects)
	if csvErr != nil {
		return result, csvErr
	}
	filtered["ContentsofExportPackage_"+finalPackageName+".csv"] = contentsCSV

	result.Files = filtered
	return result, nil
}

// filterPackageFilesForSelection drops files backing exported objects that were not
// selected, and always drops any preexisting exportPackage.chksum, exportMetadata.v2.json,
// and ContentsofExportPackage_*.csv entries (these are regenerated by the caller).
func filterPackageFilesForSelection(
	fileContents map[string][]byte,
	allObjects []ExportedObject,
	selectedIDs map[string]bool,
) map[string][]byte {
	allObjectFiles := make(map[string]bool)
	selectedObjectFiles := make(map[string]bool)
	for _, o := range allObjects {
		candidates := dependencies.ObjectChecksumCandidates(o.Path, o.ObjectName, o.ObjectType)
		for _, c := range candidates {
			if _, ok := fileContents[c]; ok {
				allObjectFiles[c] = true
				if selectedIDs[o.ObjectGUID] {
					selectedObjectFiles[c] = true
				}
			}
		}
	}

	out := make(map[string][]byte)
	for path, data := range fileContents {
		if path == "exportPackage.chksum" || path == "exportMetadata.v2.json" {
			continue
		}
		if strings.HasPrefix(path, "ContentsofExportPackage_") && strings.HasSuffix(path, ".csv") {
			continue
		}
		if allObjectFiles[path] {
			if selectedObjectFiles[path] {
				out[path] = data
			}
			continue
		}
		out[path] = data
	}
	return out
}

// buildMetadataGraphForSelection computes the set of exported-object IDs that must
// remain in the regenerated exportMetadata.v2.json for an --exclude-found-transitive
// run: the selected IDs plus their in-package objectRefs closure, minus any excluded
// (transitive-found) IDs that were not themselves explicitly selected.
func buildMetadataGraphForSelection(
	allObjects []ExportedObject,
	selectedIDs map[string]bool,
	excludedIDs map[string]bool,
) map[string]bool {
	metadataIDs := make(map[string]bool, len(selectedIDs))
	for id := range selectedIDs {
		metadataIDs[id] = true
	}
	byID := make(map[string]ExportedObject, len(allObjects))
	exportedRefs := make([]dependencies.ExportedObjectRef, 0, len(allObjects))
	for _, o := range allObjects {
		if o.ObjectGUID == "" {
			continue
		}
		byID[o.ObjectGUID] = o
		exportedRefs = append(exportedRefs, dependencies.ExportedObjectRef{
			ObjectGUID: o.ObjectGUID,
			ObjectName: o.ObjectName,
			ObjectType: o.ObjectType,
			Path:       o.Path,
		})
	}

	queue := make([]string, 0, len(metadataIDs))
	for id := range metadataIDs {
		queue = append(queue, id)
	}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		obj, ok := byID[id]
		if !ok {
			continue
		}
		for _, refID := range obj.ObjectRefs() {
			if excludedIDs[refID] && !selectedIDs[refID] {
				continue
			}
			if _, exists := byID[refID]; !exists {
				continue
			}
			if metadataIDs[refID] {
				continue
			}
			metadataIDs[refID] = true
			queue = append(queue, refID)
		}
	}

	// Include container hierarchy needed for selected/closure objects.
	_ = dependencies.IncludeParentContainers(exportedRefs, metadataIDs)
	for id := range excludedIDs {
		if selectedIDs[id] {
			continue
		}
		delete(metadataIDs, id)
	}
	return metadataIDs
}

// buildContentsOfExportPackageCSV builds the ContentsofExportPackage_<name>.csv content.
func buildContentsOfExportPackageCSV(objects []ExportedObject) ([]byte, error) {
	rows := make([][]string, 0, len(objects)+1)
	rows = append(rows, []string{"objectPath", "objectName", "objectType", "id"})

	sorted := make([]ExportedObject, len(objects))
	copy(sorted, objects)
	sort.SliceStable(sorted, func(i, j int) bool {
		a := client.NormalizeLocationPath(sorted[i].Path) + "/" + sorted[i].ObjectName + "." + sorted[i].ObjectType
		b := client.NormalizeLocationPath(sorted[j].Path) + "/" + sorted[j].ObjectName + "." + sorted[j].ObjectType
		return a < b
	})

	for _, o := range sorted {
		rows = append(rows, []string{
			o.Path,
			o.ObjectName,
			o.ObjectType,
			o.ObjectGUID,
		})
	}

	var b bytes.Buffer
	w := csv.NewWriter(&b)
	if err := w.WriteAll(rows); err != nil {
		return nil, fmt.Errorf("writing ContentsofExportPackage csv: %w", err)
	}
	return b.Bytes(), nil
}
