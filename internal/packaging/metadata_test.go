package packaging

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestReadExportMetadata_Workspace(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "exportMetadata.v2.json"), `{"name":"pkg","exportedObjects":[{"objectGuid":"g1","objectName":"m1","objectType":"DTEMPLATE","path":"/Explore/App"}]}`)

	meta, err := ReadExportMetadata("", dir)
	if err != nil {
		t.Fatalf("ReadExportMetadata() error: %v", err)
	}
	if meta.Name != "pkg" || len(meta.ExportedObjects) != 1 || meta.ExportedObjects[0].ObjectGUID != "g1" {
		t.Fatalf("unexpected metadata: %+v", meta)
	}
}

func TestReadExportMetadata_WorkspaceMissingFile(t *testing.T) {
	dir := t.TempDir()
	if _, err := ReadExportMetadata("", dir); err == nil {
		t.Fatal("expected error when exportMetadata.v2.json is missing")
	}
}

func TestReadExportMetadata_ZIP(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "pkg.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create("exportMetadata.v2.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, werr := w.Write([]byte(`{"name":"zippkg","exportedObjects":[]}`)); werr != nil {
		t.Fatal(werr)
	}
	if cerr := zw.Close(); cerr != nil {
		t.Fatal(cerr)
	}
	if cerr := f.Close(); cerr != nil {
		t.Fatal(cerr)
	}

	meta, err := ReadExportMetadata(zipPath, "")
	if err != nil {
		t.Fatalf("ReadExportMetadata() error: %v", err)
	}
	if meta.Name != "zippkg" {
		t.Fatalf("unexpected metadata: %+v", meta)
	}
}

func TestReadExportMetadata_ZIPMissingEntry(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "pkg.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	if _, err := zw.Create("other.txt"); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	if _, err := ReadExportMetadata(zipPath, ""); err == nil {
		t.Fatal("expected error when exportMetadata.v2.json entry is missing from ZIP")
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}
