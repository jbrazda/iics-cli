package client

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestUpdateFolder(t *testing.T) {
	tests := []struct {
		folderID, projectID, projectName, folderName string
		wantPath                                     string
	}{
		{"f1", "", "", "", "/public/core/v3/folders/f1"},
		{"f1", "p1", "", "", "/public/core/v3/folders/f1"},
		{"", "", "Orders", "Northeast Orders", "/public/core/v3/projects/name/Orders/folders/name/Northeast%20Orders"},
	}
	for _, tt := range tests {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPatch {
				t.Errorf("expected PATCH, got %s", r.Method)
			}
			if r.URL.EscapedPath() != tt.wantPath {
				t.Errorf("path = %s, want %s", r.URL.EscapedPath(), tt.wantPath)
			}
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["name"] != "Northeast Requests" {
				t.Errorf("unexpected body: %v", body)
			}
			w.WriteHeader(http.StatusNoContent)
		})

		c := newTestClient(handler)
		if _, err := c.UpdateFolder(context.Background(), tt.folderID, "Northeast Requests", "", tt.projectID, tt.projectName, tt.folderName); err != nil {
			t.Fatalf("UpdateFolder(%+v) error: %v", tt, err)
		}
	}
}

func TestUpdateFolderValidation(t *testing.T) {
	c := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
	}))
	cases := []struct{ folderID, name, projectID, projectName, folderName string }{
		{"", "n", "", "", ""},          // no identifier
		{"f1", "", "", "", ""},         // nothing to update
		{"f1", "n", "", "Orders", "F"}, // ID mixed with names
		{"", "n", "p1", "Orders", "F"}, // project ID mixed with names
		{"", "n", "", "", "F"},         // folder name without project name
		{"", "n", "", "Orders", ""},    // project name without folder name
		{"", "n", "p1", "", ""},        // project ID without folder ID
	}
	for _, tc := range cases {
		if _, err := c.UpdateFolder(context.Background(), tc.folderID, tc.name, "", tc.projectID, tc.projectName, tc.folderName); err == nil {
			t.Errorf("expected error for %+v", tc)
		}
	}
}

func TestCreateFolderEscapesProjectName(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if want := "/public/core/v3/projects/name/Finance%20ETL/folders"; r.URL.EscapedPath() != want {
			t.Errorf("path = %s, want %s", r.URL.EscapedPath(), want)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"f1","name":"Staging"}`))
	})

	c := newTestClient(handler)
	f, err := c.CreateFolder(context.Background(), "Staging", "", "", "Finance ETL")
	if err != nil {
		t.Fatalf("CreateFolder() error: %v", err)
	}
	if f.ID != "f1" {
		t.Errorf("expected ID f1, got %s", f.ID)
	}
}
