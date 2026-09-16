package packaging

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/dependencies"
)

// writeSourceMetadata writes exportMetadata.v2.json into dir so ReadExportMetadata("", dir)
// (called internally by BuildSelectivePackage) can read it back.
func writeSourceMetadata(t *testing.T, dir string, meta ExportMetadata) {
	t.Helper()
	data, err := json.Marshal(meta)
	if err != nil {
		t.Fatalf("marshal fixture metadata: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "exportMetadata.v2.json"), data, 0o644); err != nil {
		t.Fatalf("writing fixture exportMetadata.v2.json: %v", err)
	}
}

func rawRefs(t *testing.T, refs ...string) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(map[string][]string{"objectRefs": refs})
	if err != nil {
		t.Fatalf("marshal refs: %v", err)
	}
	return data
}

// baseFixtureObjects returns a small package graph:
//
//	proj  (Project /Explore)          -> App
//	folder(Folder  /Explore/App)      -> Mappings
//	obj1  (DTEMPLATE .../Mappings)    -> m1, refs conn1
//	conn1 (Connection /SYS/Connections) -> ConnA
//	obj2  (DTEMPLATE .../Mappings)    -> m2, unrelated, no refs
func baseFixtureObjects(t *testing.T) []ExportedObject {
	t.Helper()
	return []ExportedObject{
		{ObjectGUID: "proj", ObjectName: "App", ObjectType: "Project", Path: "/Explore"},
		{ObjectGUID: "folder", ObjectName: "Mappings", ObjectType: "Folder", Path: "/Explore/App"},
		{ObjectGUID: "obj1", ObjectName: "m1", ObjectType: "DTEMPLATE", Path: "/Explore/App/Mappings", Metadata: rawRefs(t, "conn1")},
		{ObjectGUID: "conn1", ObjectName: "ConnA", ObjectType: "Connection", Path: "/SYS/Connections"},
		{ObjectGUID: "obj2", ObjectName: "m2", ObjectType: "DTEMPLATE", Path: "/Explore/App/Mappings"},
	}
}

func candidateFile(path, name, typ string) string {
	return strings.TrimPrefix(strings.TrimSuffix(path, "/"), "/") + "/" + name + "." + typ + ".xml"
}

func baseFixtureFiles() map[string][]byte {
	return map[string][]byte{
		candidateFile("/Explore/App/Mappings", "m1", "DTEMPLATE"): []byte("obj1-data"),
		candidateFile("/SYS/Connections", "ConnA", "Connection"):  []byte("conn1-data"),
		candidateFile("/Explore/App/Mappings", "m2", "DTEMPLATE"): []byte("obj2-data"),
		"readme.txt": []byte("not an object file"),
	}
}

func TestBuildSelectivePackage_DefaultClosureAndParents(t *testing.T) {
	dir := t.TempDir()
	writeSourceMetadata(t, dir, ExportMetadata{Name: "src", ExportedObjects: baseFixtureObjects(t)})
	fileContents := baseFixtureFiles()

	entries := []client.ArtifactEntry{{Path: "Explore/App/Mappings/m1", Type: "DTEMPLATE"}}

	result, err := BuildSelectivePackage(fileContents, dir, entries, dependencies.BuildManifestStats{}, BuildOptions{
		Target: "/out/mypkg.zip",
	})
	if err != nil {
		t.Fatalf("BuildSelectivePackage() error: %v", err)
	}

	if result.ReportSelected != 4 {
		t.Fatalf("ReportSelected = %d, want 4 (obj1, folder, proj, conn1)", result.ReportSelected)
	}
	if result.ReportExcluded != 0 {
		t.Fatalf("ReportExcluded = %d, want 0", result.ReportExcluded)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", result.Warnings)
	}

	// obj1 and conn1 files retained, obj2's file dropped, unrelated file untouched.
	if _, ok := result.Files[candidateFile("/Explore/App/Mappings", "m1", "DTEMPLATE")]; !ok {
		t.Fatal("expected obj1 file to be retained")
	}
	if _, ok := result.Files[candidateFile("/SYS/Connections", "ConnA", "Connection")]; !ok {
		t.Fatal("expected conn1 file to be retained (closure)")
	}
	if _, ok := result.Files[candidateFile("/Explore/App/Mappings", "m2", "DTEMPLATE")]; ok {
		t.Fatal("expected obj2 file to be dropped (not selected)")
	}
	if _, ok := result.Files["readme.txt"]; !ok {
		t.Fatal("expected unrelated file to be retained untouched")
	}

	// exportMetadata.v2.json regenerated with the default name derived from --target basename.
	metaBytes, ok := result.Files["exportMetadata.v2.json"]
	if !ok {
		t.Fatal("expected regenerated exportMetadata.v2.json")
	}
	var meta ExportMetadata
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		t.Fatalf("unmarshal regenerated metadata: %v", err)
	}
	if meta.Name != "mypkg" {
		t.Fatalf("meta.Name = %q, want %q (derived from --target)", meta.Name, "mypkg")
	}
	if len(meta.ExportedObjects) != 4 {
		t.Fatalf("regenerated metadata has %d objects, want 4", len(meta.ExportedObjects))
	}

	// ContentsofExportPackage CSV regenerated using the derived name.
	if _, ok := result.Files["ContentsofExportPackage_mypkg.csv"]; !ok {
		t.Fatal("expected ContentsofExportPackage_mypkg.csv to be generated")
	}

	// Notices: parent-container inclusion (folder+proj) then closure inclusion (conn1).
	if len(result.Notices) != 2 {
		t.Fatalf("Notices = %v, want 2 entries", result.Notices)
	}
	if !strings.Contains(result.Notices[0], "included 2 inferred parent Project/Folder objects") {
		t.Fatalf("Notices[0] = %q, want parent-container notice", result.Notices[0])
	}
	if !strings.Contains(result.Notices[1], "included 1 in-package referenced dependencies") {
		t.Fatalf("Notices[1] = %q, want closure notice", result.Notices[1])
	}
}

func TestBuildSelectivePackage_ExcludeFoundTransitiveKeepsCDIRefs(t *testing.T) {
	dir := t.TempDir()
	writeSourceMetadata(t, dir, ExportMetadata{Name: "src", ExportedObjects: baseFixtureObjects(t)})
	fileContents := baseFixtureFiles()

	entries := []client.ArtifactEntry{{Path: "Explore/App/Mappings/m1", Type: "DTEMPLATE"}}
	stats := dependencies.BuildManifestStats{
		ExcludedTransitiveFound:  3,
		SelectedStatusColumnName: "STATUS (qa)",
	}

	result, err := BuildSelectivePackage(fileContents, dir, entries, stats, BuildOptions{
		ExcludeFoundTransitive: true,
		PackageName:            "explicit-name",
		Target:                 "/out/mypkg.zip",
	})
	if err != nil {
		t.Fatalf("BuildSelectivePackage() error: %v", err)
	}

	// conn1 is re-included via CDI ref rescue even though ordinary closure is suppressed.
	if result.ReportSelected != 4 {
		t.Fatalf("ReportSelected = %d, want 4", result.ReportSelected)
	}
	// closureSuppressedExcluded should count the 1 suppressed closure addition (conn1).
	if result.ReportExcluded != 4 {
		t.Fatalf("ReportExcluded = %d, want 4 (3 manifest-excluded + 1 closure-suppressed)", result.ReportExcluded)
	}

	if _, ok := result.Files[candidateFile("/SYS/Connections", "ConnA", "Connection")]; !ok {
		t.Fatal("expected conn1 file to be retained via CDI ref rescue")
	}

	// PackageName override is honored over the --target-derived default.
	if _, ok := result.Files["ContentsofExportPackage_explicit-name.csv"]; !ok {
		t.Fatal("expected ContentsofExportPackage_explicit-name.csv")
	}

	joined := strings.Join(result.Notices, "|")
	if !strings.Contains(joined, "Selection filter: excluded 4 transitive found rows using STATUS (qa) (manifest rows: 3, closure-suppressed: 1)") {
		t.Fatalf("expected exclude-transitive notice, got: %v", result.Notices)
	}
	if !strings.Contains(joined, "included 1 CDI Connection/AgentGroup refs from objectRefs") {
		t.Fatalf("expected CDI ref notice, got: %v", result.Notices)
	}
	if strings.Contains(joined, "in-package referenced dependencies") {
		t.Fatalf("did not expect ordinary closure notice when --exclude-found-transitive suppresses it, got: %v", result.Notices)
	}
}

func TestBuildSelectivePackage_EmptyManifestErrors(t *testing.T) {
	dir := t.TempDir()
	_, err := BuildSelectivePackage(map[string][]byte{}, dir, nil, dependencies.BuildManifestStats{}, BuildOptions{})
	if err == nil || !strings.Contains(err.Error(), "selection manifest is empty") {
		t.Fatalf("expected 'selection manifest is empty' error, got: %v", err)
	}
}

func TestBuildSelectivePackage_NoFilesRemainAfterFiltering(t *testing.T) {
	dir := t.TempDir()
	writeSourceMetadata(t, dir, ExportMetadata{Name: "src", ExportedObjects: baseFixtureObjects(t)})

	// Only obj2's file exists on disk, and the manifest selects obj1 only, so
	// filtering drops obj2's file and every synthetic package file is regenerated
	// from scratch, leaving nothing behind for filterPackageFilesForSelection to keep.
	fileContents := map[string][]byte{
		candidateFile("/Explore/App/Mappings", "m2", "DTEMPLATE"): []byte("obj2-data"),
		"exportPackage.chksum":          []byte("stale"),
		"exportMetadata.v2.json":        []byte("stale"),
		"ContentsofExportPackage_x.csv": []byte("stale"),
	}
	entries := []client.ArtifactEntry{{Path: "Explore/App/Mappings/m1", Type: "DTEMPLATE"}}

	_, err := BuildSelectivePackage(fileContents, dir, entries, dependencies.BuildManifestStats{}, BuildOptions{Target: "/out/p.zip"})
	if err == nil || !strings.Contains(err.Error(), "no package files remained after selection filtering") {
		t.Fatalf("expected 'no package files remained' error, got: %v", err)
	}
}

func TestBuildSelectivePackage_DuplicateManifestSelectorWarns(t *testing.T) {
	dir := t.TempDir()
	writeSourceMetadata(t, dir, ExportMetadata{Name: "src", ExportedObjects: baseFixtureObjects(t)})
	fileContents := baseFixtureFiles()

	entries := []client.ArtifactEntry{
		{Path: "Explore/App/Mappings/m1", Type: "DTEMPLATE"},
		{Path: "Explore/App/Mappings/m1", Type: "DTEMPLATE"},
	}

	result, err := BuildSelectivePackage(fileContents, dir, entries, dependencies.BuildManifestStats{}, BuildOptions{Target: "/out/p.zip"})
	if err != nil {
		t.Fatalf("BuildSelectivePackage() error: %v", err)
	}
	if len(result.Warnings) == 0 {
		t.Fatal("expected a duplicate-manifest-selector warning")
	}
}

func TestFilterPackageFilesForSelection(t *testing.T) {
	objects := baseFixtureObjects(t)
	selected := map[string]bool{"obj1": true}
	files := baseFixtureFiles()

	out := filterPackageFilesForSelection(files, objects, selected)

	if _, ok := out[candidateFile("/Explore/App/Mappings", "m1", "DTEMPLATE")]; !ok {
		t.Fatal("expected selected object's file retained")
	}
	if _, ok := out[candidateFile("/Explore/App/Mappings", "m2", "DTEMPLATE")]; ok {
		t.Fatal("expected unselected object's file dropped")
	}
	if _, ok := out["readme.txt"]; !ok {
		t.Fatal("expected non-object file retained")
	}
}

func TestFilterPackageFilesForSelection_AlwaysDropsRegeneratedEntries(t *testing.T) {
	files := map[string][]byte{
		"exportPackage.chksum":          []byte("x"),
		"exportMetadata.v2.json":        []byte("x"),
		"ContentsofExportPackage_a.csv": []byte("x"),
		"keep.txt":                      []byte("x"),
	}
	out := filterPackageFilesForSelection(files, nil, nil)
	if len(out) != 1 {
		t.Fatalf("expected only keep.txt to remain, got %v", out)
	}
	if _, ok := out["keep.txt"]; !ok {
		t.Fatal("expected keep.txt retained")
	}
}

func TestBuildContentsOfExportPackageCSV(t *testing.T) {
	objects := []ExportedObject{
		{ObjectGUID: "b", ObjectName: "zeta", ObjectType: "PROCESS", Path: "/Explore/App"},
		{ObjectGUID: "a", ObjectName: "alpha", ObjectType: "PROCESS", Path: "/Explore/App"},
	}
	data, err := buildContentsOfExportPackageCSV(objects)
	if err != nil {
		t.Fatalf("buildContentsOfExportPackageCSV() error: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\r\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected header + 2 rows, got %d lines: %v", len(lines), lines)
	}
	if lines[0] != "objectPath,objectName,objectType,id" {
		t.Fatalf("unexpected header: %q", lines[0])
	}
	// alpha sorts before zeta.
	if !strings.Contains(lines[1], "alpha") {
		t.Fatalf("expected alpha row first, got: %v", lines)
	}
	if !strings.Contains(lines[2], "zeta") {
		t.Fatalf("expected zeta row second, got: %v", lines)
	}
}

func TestBuildMetadataGraphForSelection(t *testing.T) {
	objects := baseFixtureObjects(t)
	selected := map[string]bool{"obj1": true}

	ids := buildMetadataGraphForSelection(objects, selected, nil)

	for _, want := range []string{"obj1", "conn1"} {
		if !ids[want] {
			t.Fatalf("expected %q included in metadata graph, got %v", want, ids)
		}
	}
	if ids["obj2"] {
		t.Fatal("did not expect unrelated obj2 in metadata graph")
	}
}
