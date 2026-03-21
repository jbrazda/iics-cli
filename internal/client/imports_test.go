package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"strings"
	"testing"
)

func TestUploadImportPackage(t *testing.T) {
	const fakeZip = "PK\x03\x04fake-zip-data"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/public/core/v3/import/package" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		ct := r.Header.Get("Content-Type")
		mt, _, err := mime.ParseMediaType(ct)
		if err != nil || mt != "multipart/form-data" {
			t.Errorf("expected multipart/form-data Content-Type, got %s", ct)
		}

		err = r.ParseMultipartForm(1 << 20)
		if err != nil {
			t.Fatalf("parsing multipart form: %v", err)
		}
		file, _, err := r.FormFile("package")
		if err != nil {
			t.Fatalf("getting form file: %v", err)
		}
		defer file.Close()
		data, _ := io.ReadAll(file)
		if string(data) != fakeZip {
			t.Errorf("unexpected file content: %q", data)
		}

		resp := ImportUploadResponse{
			JobID:         "upload-job-1",
			JobStatus:     JobStatus{State: "READY"},
			ChecksumValid: true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	c := newTestClient(handler)
	resp, err := c.UploadImportPackage(context.Background(), "test.zip", strings.NewReader(fakeZip))
	if err != nil {
		t.Fatalf("UploadImportPackage() error: %v", err)
	}
	if resp.JobID != "upload-job-1" {
		t.Errorf("expected job ID upload-job-1, got %s", resp.JobID)
	}
	if !resp.ChecksumValid {
		t.Error("expected ChecksumValid=true")
	}
}

func TestStartImport(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/import/job123") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var req ImportStartRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Name != "test-import" {
			t.Errorf("expected name 'test-import', got %s", req.Name)
		}
		if req.ImportSpecification.DefaultConflictResolution != "OVERWRITE" {
			t.Errorf("expected OVERWRITE, got %s", req.ImportSpecification.DefaultConflictResolution)
		}

		resp := ImportJob{
			ID:     "job123",
			Name:   "test-import",
			Status: JobStatus{State: "IN_PROGRESS"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	c := newTestClient(handler)
	req := &ImportStartRequest{
		Name: "test-import",
		ImportSpecification: ImportSpecification{
			DefaultConflictResolution: "OVERWRITE",
		},
	}
	job, err := c.StartImport(context.Background(), "job123", req)
	if err != nil {
		t.Fatalf("StartImport() error: %v", err)
	}
	if job.ID != "job123" {
		t.Errorf("expected ID job123, got %s", job.ID)
	}
	if job.Status.State != "IN_PROGRESS" {
		t.Errorf("expected IN_PROGRESS, got %s", job.Status.State)
	}
}

func TestGetImportStatus(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("expand") != "objects" {
			t.Errorf("expected expand=objects, got %q", r.URL.Query().Get("expand"))
		}

		resp := ImportJob{
			ID:   "job123",
			Name: "test-import",
			Status: JobStatus{
				State: "SUCCESSFUL",
			},
			Objects: []ImportJobObject{
				{
					SourceObject: ImportObjectRef{ID: "obj1", Name: "MyMapping", Path: "/Mappings", Type: "MTT"},
					TargetObject: ImportObjectRef{Name: "MyMapping", Path: "/Mappings", Type: "MTT"},
					Status:       JobStatus{State: "SUCCESSFUL", Message: "Overwrite existing object"},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	c := newTestClient(handler)
	job, err := c.GetImportStatus(context.Background(), "job123", true)
	if err != nil {
		t.Fatalf("GetImportStatus() error: %v", err)
	}
	if job.Status.State != "SUCCESSFUL" {
		t.Errorf("expected SUCCESSFUL, got %s", job.Status.State)
	}
	if len(job.Objects) != 1 {
		t.Fatalf("expected 1 object, got %d", len(job.Objects))
	}
	if job.Objects[0].SourceObject.Name != "MyMapping" {
		t.Errorf("expected source object name MyMapping, got %s", job.Objects[0].SourceObject.Name)
	}
}

func TestFlattenImportObjects(t *testing.T) {
	objects := []ImportJobObject{
		{
			SourceObject: ImportObjectRef{
				ID:          "0KfTorzrNwihXfV38FliA2",
				Name:        "TestServiceConnection1",
				Path:        "/ZZ_TEST_CLI/Connections",
				Type:        "AI_CONNECTION",
				Description: "Test connection",
			},
			TargetObject: ImportObjectRef{
				Name: "TestServiceConnection1",
				Path: "/ZZ_TEST_CLI/Connections",
				Type: "AI_CONNECTION",
			},
			Status: JobStatus{State: "SUCCESSFUL", Message: "Overwrite existing object"},
		},
		{
			SourceObject: ImportObjectRef{
				ID:   "0pXCTRhtjrcdwOCIS5P7S5",
				Name: "m_Test_Git",
				Path: "/ZZ_TEST_CLI/Mappings",
				Type: "DTEMPLATE",
			},
			TargetObject: ImportObjectRef{
				Name: "m_Test_Git_renamed",
				Path: "/ZZ_TEST_CLI/Mappings",
				Type: "DTEMPLATE",
			},
			Status: JobStatus{State: "SUCCESSFUL", Message: "Overwrite existing object"},
		},
	}

	flat := FlattenImportObjects(objects)

	if len(flat) != 2 {
		t.Fatalf("expected 2 flat objects, got %d", len(flat))
	}

	f0 := flat[0]
	if f0.SourceID != "0KfTorzrNwihXfV38FliA2" {
		t.Errorf("expected sourceId 0KfTorzrNwihXfV38FliA2, got %s", f0.SourceID)
	}
	if f0.SourceName != "TestServiceConnection1" {
		t.Errorf("expected sourceName TestServiceConnection1, got %s", f0.SourceName)
	}
	if f0.SourcePath != "/ZZ_TEST_CLI/Connections" {
		t.Errorf("expected sourcePath /ZZ_TEST_CLI/Connections, got %s", f0.SourcePath)
	}
	if f0.SourceType != "AI_CONNECTION" {
		t.Errorf("expected sourceType AI_CONNECTION, got %s", f0.SourceType)
	}
	if f0.SourceDescription != "Test connection" {
		t.Errorf("expected sourceDescription 'Test connection', got %s", f0.SourceDescription)
	}
	if f0.TargetName != "TestServiceConnection1" {
		t.Errorf("expected targetName TestServiceConnection1, got %s", f0.TargetName)
	}
	if f0.State != "SUCCESSFUL" {
		t.Errorf("expected state SUCCESSFUL, got %s", f0.State)
	}
	if f0.Message != "Overwrite existing object" {
		t.Errorf("expected message 'Overwrite existing object', got %s", f0.Message)
	}

	// second object has different source/target names
	f1 := flat[1]
	if f1.SourceName != "m_Test_Git" {
		t.Errorf("expected sourceName m_Test_Git, got %s", f1.SourceName)
	}
	if f1.TargetName != "m_Test_Git_renamed" {
		t.Errorf("expected targetName m_Test_Git_renamed, got %s", f1.TargetName)
	}
}

func TestGetImportStatus_NoExpand(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("expand") != "" {
			t.Errorf("expected no expand param, got %q", r.URL.Query().Get("expand"))
		}
		resp := ImportJob{
			ID:     "job456",
			Name:   "test-import",
			Status: JobStatus{State: "IN_PROGRESS"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	c := newTestClient(handler)
	job, err := c.GetImportStatus(context.Background(), "job456", false)
	if err != nil {
		t.Fatalf("GetImportStatus() error: %v", err)
	}
	if job.Status.State != "IN_PROGRESS" {
		t.Errorf("expected IN_PROGRESS, got %s", job.Status.State)
	}
}

func TestDownloadImportLog(t *testing.T) {
	const logContent = "Import started.\nObject MyMapping: SUCCESSFUL\nImport completed.\n"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/public/core/v3/import/job123/log" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(logContent))
	})

	c := newTestClient(handler)
	var buf bytes.Buffer
	if err := c.DownloadImportLog(context.Background(), "job123", &buf); err != nil {
		t.Fatalf("DownloadImportLog() error: %v", err)
	}
	if buf.String() != logContent {
		t.Errorf("log content mismatch: got %q", buf.String())
	}
}
