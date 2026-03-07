package client

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
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
