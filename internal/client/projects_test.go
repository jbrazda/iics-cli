package client

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestUpdateProject(t *testing.T) {
	tests := []struct {
		id, projectName string
		wantPath        string
	}{
		{"p1", "", "/public/core/v3/projects/p1"},
		{"", "Northeast Orders", "/public/core/v3/projects/name/Northeast%20Orders"},
	}
	for _, tt := range tests {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPatch {
				t.Errorf("expected PATCH, got %s", r.Method)
			}
			if r.URL.EscapedPath() != tt.wantPath {
				t.Errorf("path = %s, want %s", r.URL.EscapedPath(), tt.wantPath)
			}
			if r.Header.Get("INFA-SESSION-ID") != "test-session" {
				t.Errorf("expected V3 session header")
			}
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["description"] != "Orders from all regions" {
				t.Errorf("unexpected body: %v", body)
			}
			if _, ok := body["name"]; ok {
				t.Errorf("name should be omitted when empty: %v", body)
			}
			w.WriteHeader(http.StatusOK)
		})

		c := newTestClient(handler)
		p, err := c.UpdateProject(context.Background(), tt.id, tt.projectName, "", "Orders from all regions")
		if err != nil {
			t.Fatalf("UpdateProject(%q, %q) error: %v", tt.id, tt.projectName, err)
		}
		if p == nil || p.ID != "" {
			t.Errorf("expected empty project for empty response body, got %+v", p)
		}
	}
}

func TestUpdateProjectValidation(t *testing.T) {
	c := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
	}))
	cases := []struct{ id, projectName, name, desc string }{
		{"", "", "n", ""},
		{"p1", "P", "n", ""},
		{"p1", "", "", ""},
	}
	for _, tc := range cases {
		if _, err := c.UpdateProject(context.Background(), tc.id, tc.projectName, tc.name, tc.desc); err == nil {
			t.Errorf("expected error for %+v", tc)
		}
	}
}
