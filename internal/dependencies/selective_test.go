package dependencies

import (
	"strings"
	"testing"

	"github.com/jbrazda/iics-cli/internal/client"
)

func TestSelectExportedObjects_PathTypeAndContainerExpansion(t *testing.T) {
	objects := []ExportedObjectRef{
		{ObjectGUID: "proj", ObjectName: "App", ObjectType: "Project", Path: "/Explore"},
		{ObjectGUID: "folder", ObjectName: "Mappings", ObjectType: "Folder", Path: "/Explore/App"},
		{ObjectGUID: "map", ObjectName: "m1", ObjectType: "DTEMPLATE", Path: "/Explore/App/Mappings"},
		{ObjectGUID: "sysconn", ObjectName: "ConnA", ObjectType: "Connection", Path: "/SYS/Connections"},
	}
	entries := []client.ArtifactEntry{
		{Path: "Explore/App/Mappings", Type: "Folder"},
	}

	got, _, err := SelectExportedObjects(entries, objects)
	if err != nil {
		t.Fatalf("SelectExportedObjects() error: %v", err)
	}
	for _, id := range []string{"folder", "map", "sysconn"} {
		if !got[id] {
			t.Fatalf("expected selected id %q", id)
		}
	}
}

func TestSelectExportedObjects_ProjectIDExpandsDescendants(t *testing.T) {
	objects := []ExportedObjectRef{
		{ObjectGUID: "proj", ObjectName: "App", ObjectType: "Project", Path: "/Explore"},
		{ObjectGUID: "folder", ObjectName: "Mappings", ObjectType: "Folder", Path: "/Explore/App"},
		{ObjectGUID: "map", ObjectName: "m1", ObjectType: "DTEMPLATE", Path: "/Explore/App/Mappings"},
	}
	entries := []client.ArtifactEntry{
		{ID: "proj"},
	}

	got, _, err := SelectExportedObjects(entries, objects)
	if err != nil {
		t.Fatalf("SelectExportedObjects() error: %v", err)
	}
	for _, id := range []string{"proj", "folder", "map"} {
		if !got[id] {
			t.Fatalf("expected selected id %q", id)
		}
	}
}

func TestSelectExportedObjects_MixedDoesNotAutoIncludeSYS(t *testing.T) {
	objects := []ExportedObjectRef{
		{ObjectGUID: "folder", ObjectName: "Mappings", ObjectType: "Folder", Path: "/Explore/App"},
		{ObjectGUID: "map", ObjectName: "m1", ObjectType: "DTEMPLATE", Path: "/Explore/App/Mappings"},
		{ObjectGUID: "task", ObjectName: "t1", ObjectType: "MTT", Path: "/Explore/App/Tasks"},
		{ObjectGUID: "sysconn", ObjectName: "ConnA", ObjectType: "Connection", Path: "/SYS/Connections"},
	}
	entries := []client.ArtifactEntry{
		{Path: "Explore/App/Mappings", Type: "Folder"},
		{Path: "Explore/App/Tasks/t1", Type: "MTT"},
	}

	got, _, err := SelectExportedObjects(entries, objects)
	if err != nil {
		t.Fatalf("SelectExportedObjects() error: %v", err)
	}
	if got["sysconn"] {
		t.Fatalf("did not expect sysconn for mixed explicit selection")
	}
}

func TestSelectExportedObjects_MixedSelectorsDoNotWarn(t *testing.T) {
	objects := []ExportedObjectRef{
		{ObjectGUID: "map", ObjectName: "m1", ObjectType: "DTEMPLATE", Path: "/Explore/App/Mappings"},
	}
	entries := []client.ArtifactEntry{
		{Path: "Explore/App/Mappings/m1", Type: "DTEMPLATE"},
		{ID: "map"},
	}

	got, warnings, err := SelectExportedObjects(entries, objects)
	if err != nil {
		t.Fatalf("SelectExportedObjects() error: %v", err)
	}
	if !got["map"] {
		t.Fatalf("expected map selected")
	}
	if len(warnings) != 0 {
		t.Fatalf("did not expect warnings for mixed selectors, got %v", warnings)
	}
}

func TestSelectExportedObjects_DuplicateManifestPathTypeWarning(t *testing.T) {
	objects := []ExportedObjectRef{
		{ObjectGUID: "map", ObjectName: "m1", ObjectType: "DTEMPLATE", Path: "/Explore/App/Mappings"},
	}
	entries := []client.ArtifactEntry{
		{Path: "Explore/App/Mappings/m1", Type: "DTEMPLATE"},
		{Path: "Explore/App/Mappings/m1", Type: "DTEMPLATE"},
	}

	got, warnings, err := SelectExportedObjects(entries, objects)
	if err != nil {
		t.Fatalf("SelectExportedObjects() error: %v", err)
	}
	if !got["map"] {
		t.Fatalf("expected map selected")
	}
	if len(warnings) == 0 {
		t.Fatalf("expected duplicate warning for duplicate manifest selector")
	}
}

func TestIncludeParentContainers(t *testing.T) {
	objects := []ExportedObjectRef{
		{ObjectGUID: "proj", ObjectName: "ZZ_TEST_CLI", ObjectType: "Project", Path: "/Explore"},
		{ObjectGUID: "folder1", ObjectName: "Processes", ObjectType: "Folder", Path: "/Explore/ZZ_TEST_CLI"},
		{ObjectGUID: "folder2", ObjectName: "Nested", ObjectType: "Folder", Path: "/Explore/ZZ_TEST_CLI/Processes"},
		{ObjectGUID: "leaf", ObjectName: "MyProc", ObjectType: "PROCESS", Path: "/Explore/ZZ_TEST_CLI/Processes/Nested"},
	}
	selected := map[string]bool{"leaf": true}

	added := IncludeParentContainers(objects, selected)
	if added != 3 {
		t.Fatalf("expected 3 parent containers added, got %d", added)
	}
	for _, id := range []string{"proj", "folder1", "folder2", "leaf"} {
		if !selected[id] {
			t.Fatalf("expected selected id %q", id)
		}
	}
}

func TestSelectExportedObjects_SuppressesInferredDuplicateWarnings(t *testing.T) {
	objects := []ExportedObjectRef{
		{ObjectGUID: "proj", ObjectName: "App", ObjectType: "Project", Path: "/Explore"},
		{ObjectGUID: "folder", ObjectName: "Mappings", ObjectType: "Folder", Path: "/Explore/App"},
		{ObjectGUID: "map", ObjectName: "m1", ObjectType: "DTEMPLATE", Path: "/Explore/App/Mappings"},
		{ObjectGUID: "sysconn", ObjectName: "ConnA", ObjectType: "Connection", Path: "/SYS/Connections"},
	}
	entries := []client.ArtifactEntry{
		{ID: "proj"},
		{Path: "Explore/App/Mappings", Type: "Folder"},
	}

	_, warnings, err := SelectExportedObjects(entries, objects)
	if err != nil {
		t.Fatalf("SelectExportedObjects() error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("did not expect warnings for inferred overlap, got %v", warnings)
	}
	for _, w := range warnings {
		if strings.Contains(w, "container expansion") || strings.Contains(w, "SYS implied by container-only selection") {
			t.Fatalf("unexpected inferred duplicate warning: %q", w)
		}
	}
}
