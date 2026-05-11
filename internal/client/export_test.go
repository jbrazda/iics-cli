package client

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestCreateExport(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var req ExportRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Name != "test-export" {
			t.Errorf("expected name 'test-export', got %s", req.Name)
		}
		if len(req.Objects) != 2 {
			t.Fatalf("expected 2 objects, got %d", len(req.Objects))
		}
		if !req.Objects[0].IncludeDependencies {
			t.Error("expected IncludeDependencies=true")
		}

		resp := ExportJob{
			ID:   "job123",
			Name: "test-export",
			Status: JobStatus{
				State: "IN_PROGRESS",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	c := newTestClient(handler)
	job, err := c.CreateExport(context.Background(), &ExportRequest{
		Name: "test-export",
		Objects: []ExportObject{
			{ID: "obj1", IncludeDependencies: true},
			{ID: "obj2", IncludeDependencies: true},
		},
	})
	if err != nil {
		t.Fatalf("CreateExport() error: %v", err)
	}
	if job.ID != "job123" {
		t.Errorf("expected job ID job123, got %s", job.ID)
	}
	if job.Status.State != "IN_PROGRESS" {
		t.Errorf("expected state IN_PROGRESS, got %s", job.Status.State)
	}
}

func TestStartExport(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Query().Get("includeTagInformation") != "true" {
			t.Errorf("expected includeTagInformation=true query param")
		}

		var req ExportRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Name != "tagged-export" {
			t.Errorf("expected name 'tagged-export', got %s", req.Name)
		}
		if len(req.Objects) != 1 || !req.Objects[0].IncludeDependencies {
			t.Error("expected object-level IncludeDependencies=true")
		}

		resp := ExportJob{
			ID:   "job456",
			Name: "tagged-export",
			Status: JobStatus{
				State: "QUEUED",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	c := newTestClient(handler)
	job, err := c.StartExport(context.Background(), &ExportRequest{
		Name:    "tagged-export",
		Objects: []ExportObject{{ID: "obj1", IncludeDependencies: true}},
	}, ExportCreateOptions{IncludeTags: true})
	if err != nil {
		t.Fatalf("StartExport() error: %v", err)
	}
	if job.ID != "job456" {
		t.Errorf("expected job ID job456, got %s", job.ID)
	}
}

func TestGetExportStatus(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("expand") != "objects" {
			t.Errorf("expected expand=objects, got %s", r.URL.Query().Get("expand"))
		}

		resp := ExportJob{
			ID:   "job123",
			Name: "test-export",
			Status: JobStatus{
				State: "SUCCESSFUL",
			},
			Objects: []JobObject{
				{ID: "obj1", Name: "Mapping1", Status: JobStatus{State: "SUCCESSFUL"}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	c := newTestClient(handler)
	job, err := c.GetExportStatus(context.Background(), "job123", true)
	if err != nil {
		t.Fatalf("GetExportStatus() error: %v", err)
	}
	if job.Status.State != "SUCCESSFUL" {
		t.Errorf("expected state SUCCESSFUL, got %s", job.Status.State)
	}
	if len(job.Objects) != 1 {
		t.Fatalf("expected 1 object, got %d", len(job.Objects))
	}
}

func TestDownloadExportPackage(t *testing.T) {
	zipContent := []byte("PK\x03\x04fake-zip-data")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/public/core/v3/export/job123/package" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/zip")
		w.Write(zipContent)
	})

	c := newTestClient(handler)
	var buf bytes.Buffer
	err := c.DownloadExportPackage(context.Background(), "job123", &buf)
	if err != nil {
		t.Fatalf("DownloadExportPackage() error: %v", err)
	}
	if !bytes.Equal(buf.Bytes(), zipContent) {
		t.Errorf("downloaded content mismatch")
	}
}

func TestDownloadExportLog(t *testing.T) {
	logContent := []byte("export log line 1\nexport log line 2\n")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/public/core/v3/export/job123/log" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Write(logContent)
	})

	c := newTestClient(handler)
	var buf bytes.Buffer
	err := c.DownloadExportLog(context.Background(), "job123", &buf)
	if err != nil {
		t.Fatalf("DownloadExportLog() error: %v", err)
	}
	if !bytes.Equal(buf.Bytes(), logContent) {
		t.Errorf("log content mismatch")
	}
}

func TestParseLocationString(t *testing.T) {
	tests := []struct {
		location    string
		wantPath    string
		wantType    string
		wantErrSubs string
	}{
		{
			location: "Explore/ProjectName/MyFolderName/MyProcess.PROCESS",
			wantPath: "ProjectName/MyFolderName/MyProcess",
			wantType: "PROCESS",
		},
		{
			location: "Explore/ProjectName/MyConnection.AI_CONNECTION",
			wantPath: "ProjectName/MyConnection",
			wantType: "AI_CONNECTION",
		},
		{
			location: "Explore/MyProjectName.Project",
			wantPath: "MyProjectName",
			wantType: "Project",
		},
		{
			location:    "InvalidNoType",
			wantErrSubs: "expected Explore/path.TYPE or SYS/path.TYPE format",
		},
		{
			// Without "Explore/" prefix still works
			location: "SomeProject/Asset.DTEMPLATE",
			wantPath: "SomeProject/Asset",
			wantType: "DTEMPLATE",
		},
		{
			location: "SYS/Agents/Group1.AgentGroup",
			wantPath: "Agents/Group1",
			wantType: "AgentGroup",
		},
	}

	for _, tc := range tests {
		path, assetType, err := ParseLocationString(tc.location)
		if tc.wantErrSubs != "" {
			if err == nil || !strings.Contains(err.Error(), tc.wantErrSubs) {
				t.Errorf("ParseLocationString(%q): want error containing %q, got %v", tc.location, tc.wantErrSubs, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseLocationString(%q): unexpected error: %v", tc.location, err)
			continue
		}
		if path != tc.wantPath {
			t.Errorf("ParseLocationString(%q): path = %q, want %q", tc.location, path, tc.wantPath)
		}
		if assetType != tc.wantType {
			t.Errorf("ParseLocationString(%q): type = %q, want %q", tc.location, assetType, tc.wantType)
		}
	}
}

func TestParseArtifactsTXT(t *testing.T) {
	input := `Explore/ProjectName/MyFolderName/MyProcess.PROCESS
Explore/ProjectName/MyConnection.AI_CONNECTION

Explore/MyProjectName.Project
`
	entries, err := ParseArtifactsReader(strings.NewReader(input), "txt")
	if err != nil {
		t.Fatalf("ParseArtifactsReader(txt) error: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if entries[0].Path != "ProjectName/MyFolderName/MyProcess" || entries[0].Type != "PROCESS" {
		t.Errorf("entry 0: got path=%q type=%q", entries[0].Path, entries[0].Type)
	}
	if entries[1].Path != "ProjectName/MyConnection" || entries[1].Type != "AI_CONNECTION" {
		t.Errorf("entry 1: got path=%q type=%q", entries[1].Path, entries[1].Type)
	}
	if entries[2].Path != "MyProjectName" || entries[2].Type != "Project" {
		t.Errorf("entry 2: got path=%q type=%q", entries[2].Path, entries[2].Type)
	}
}

func TestParseArtifactsJSON(t *testing.T) {
	// Test objects list format
	input := `{"count": 2, "objects": [
		{"id": "abc123", "type": "PROCESS"},
		{"location": "Explore/MyProject/MyMapping.DTEMPLATE"}
	]}`
	entries, err := ParseArtifactsReader(strings.NewReader(input), "json")
	if err != nil {
		t.Fatalf("ParseArtifactsReader(json) error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].ID != "abc123" || entries[0].Type != "PROCESS" {
		t.Errorf("entry 0: got id=%q type=%q", entries[0].ID, entries[0].Type)
	}
	if entries[1].Path != "MyProject/MyMapping" || entries[1].Type != "DTEMPLATE" {
		t.Errorf("entry 1: got path=%q type=%q", entries[1].Path, entries[1].Type)
	}

	// Test plain array format
	input2 := `[{"path": "MyProject/Conn", "type": "AI_CONNECTION"}]`
	entries2, err := ParseArtifactsReader(strings.NewReader(input2), "json")
	if err != nil {
		t.Fatalf("ParseArtifactsReader(json array) error: %v", err)
	}
	if len(entries2) != 1 || entries2[0].Path != "MyProject/Conn" || entries2[0].Type != "AI_CONNECTION" {
		t.Errorf("array format: got %+v", entries2)
	}
}

func TestParseArtifactsYAML(t *testing.T) {
	input := `objects:
  - id: xyz789
    type: MTT
  - path: MyProject/MyProcess
    type: PROCESS
`
	entries, err := ParseArtifactsReader(strings.NewReader(input), "yaml")
	if err != nil {
		t.Fatalf("ParseArtifactsReader(yaml) error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].ID != "xyz789" || entries[0].Type != "MTT" {
		t.Errorf("entry 0: got id=%q type=%q", entries[0].ID, entries[0].Type)
	}
	if entries[1].Path != "MyProject/MyProcess" || entries[1].Type != "PROCESS" {
		t.Errorf("entry 1: got path=%q type=%q", entries[1].Path, entries[1].Type)
	}
}

func TestParseArtifactsCSV(t *testing.T) {
	input := `ID,PATH,TYPE,DESCRIPTION,UPDATED BY,UPDATE TIME
abc123,MyProject/Task1,MTT,A task,admin,2024-01-01
,MyProject/Conn,AI_CONNECTION,,,
`
	entries, err := ParseArtifactsReader(strings.NewReader(input), "csv")
	if err != nil {
		t.Fatalf("ParseArtifactsReader(csv) error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].ID != "abc123" || entries[0].Type != "MTT" {
		t.Errorf("entry 0: got id=%q type=%q", entries[0].ID, entries[0].Type)
	}
	if entries[1].Path != "MyProject/Conn" || entries[1].Type != "AI_CONNECTION" {
		t.Errorf("entry 1: got path=%q type=%q", entries[1].Path, entries[1].Type)
	}
}

func TestParseArtifactsJSONPathOnly(t *testing.T) {
	// Entries with only path (no type, no id) should be accepted and require lookup.
	input := `[{"path": "MyProject/MyMapping"}, {"path": "MyProject/MyTask", "type": "MTT"}]`
	entries, err := ParseArtifactsReader(strings.NewReader(input), "json")
	if err != nil {
		t.Fatalf("ParseArtifactsReader(json path-only) error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Path != "MyProject/MyMapping" || entries[0].Type != "" || entries[0].ID != "" {
		t.Errorf("entry 0: got path=%q type=%q id=%q", entries[0].Path, entries[0].Type, entries[0].ID)
	}
	if entries[1].Path != "MyProject/MyTask" || entries[1].Type != "MTT" {
		t.Errorf("entry 1: got path=%q type=%q", entries[1].Path, entries[1].Type)
	}
}

func TestParseArtifactsCSVPathOnly(t *testing.T) {
	// CSV rows with only a path column (no type, no id) should be accepted and require lookup.
	input := `ID,PATH,TYPE
,MyProject/Task1,
abc123,MyProject/Task2,MTT
`
	entries, err := ParseArtifactsReader(strings.NewReader(input), "csv")
	if err != nil {
		t.Fatalf("ParseArtifactsReader(csv path-only) error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Path != "MyProject/Task1" || entries[0].Type != "" || entries[0].ID != "" {
		t.Errorf("entry 0: got path=%q type=%q id=%q", entries[0].Path, entries[0].Type, entries[0].ID)
	}
	if entries[1].ID != "abc123" || entries[1].Type != "MTT" {
		t.Errorf("entry 1: got id=%q type=%q", entries[1].ID, entries[1].Type)
	}
}

func TestParseArtifactsCSVWithLocation(t *testing.T) {
	input := `ID,LOCATION,TYPE
,Explore/MyProject/MyTask.MTT,
abc456,,PROCESS
`
	entries, err := ParseArtifactsReader(strings.NewReader(input), "csv")
	if err != nil {
		t.Fatalf("ParseArtifactsReader(csv with location) error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Path != "MyProject/MyTask" || entries[0].Type != "MTT" {
		t.Errorf("entry 0: got path=%q type=%q", entries[0].Path, entries[0].Type)
	}
	if entries[1].ID != "abc456" {
		t.Errorf("entry 1: got id=%q", entries[1].ID)
	}
}

func TestReconcileArtifactEntriesWithLookup_ReorderedAndPartial(t *testing.T) {
	entries := []ArtifactEntry{
		{Path: "Proj/A", Type: "MTT"},
		{Path: "Proj/B", Type: "MTT"},
		{Path: "Proj/C", Type: "MTT"},
	}
	results := []LookupResult{
		{ID: "id-b", Path: "Proj/B", Type: "MTT"},
		{ID: "id-c", Path: "Proj/C", Type: "MTT"},
	}

	got := ReconcileArtifactEntriesWithLookup(entries, results)

	if got[0].ID != "" {
		t.Fatalf("entry 0 ID = %q, want unresolved empty ID", got[0].ID)
	}
	if got[1].ID != "id-b" {
		t.Fatalf("entry 1 ID = %q, want %q", got[1].ID, "id-b")
	}
	if got[2].ID != "id-c" {
		t.Fatalf("entry 2 ID = %q, want %q", got[2].ID, "id-c")
	}
}

func TestReconcileArtifactEntriesWithLookup_DuplicateKeys(t *testing.T) {
	entries := []ArtifactEntry{
		{Path: "Proj/Shared", Type: "MTT"},
		{Path: "Proj/Shared", Type: "MTT"},
	}
	results := []LookupResult{
		{ID: "id-1", Path: "Proj/Shared", Type: "MTT"},
		{ID: "id-2", Path: "Proj/Shared", Type: "MTT"},
	}

	got := ReconcileArtifactEntriesWithLookup(entries, results)

	if got[0].ID != "id-1" {
		t.Fatalf("entry 0 ID = %q, want %q", got[0].ID, "id-1")
	}
	if got[1].ID != "id-2" {
		t.Fatalf("entry 1 ID = %q, want %q", got[1].ID, "id-2")
	}
}

func TestReconcileArtifactEntriesWithLookup_PathOnlyFallback(t *testing.T) {
	entries := []ArtifactEntry{
		{Path: "Explore/Proj/NoType"},
	}
	results := []LookupResult{
		{ID: "id-x", Path: "Proj/NoType", Type: "DSS"},
	}

	got := ReconcileArtifactEntriesWithLookup(entries, results)

	if got[0].ID != "id-x" {
		t.Fatalf("entry ID = %q, want %q", got[0].ID, "id-x")
	}
	if got[0].Type != "DSS" {
		t.Fatalf("entry type = %q, want %q", got[0].Type, "DSS")
	}
}
