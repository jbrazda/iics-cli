package client

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

const frsUsersJSON = `[{"id":"u1","userName":"jdoe@example.com"}]`

func TestListProjectInfoFRS(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/frs/api/v1/Projects":
			if r.Header.Get("IDS-SESSION-ID") != "test-session" {
				t.Errorf("expected IDS-SESSION-ID header on FRS request")
			}
			if r.URL.Query().Get("$top") == "" {
				t.Errorf("expected $top query parameter")
			}
			_, _ = w.Write([]byte(`{"value":[{"id":"p1","name":"Finance","description":"d","createdBy":"u1","lastUpdatedBy":"u2","createdTime":"2025-01-01T00:00:00Z","lastUpdatedTime":"2025-02-01T00:00:00Z","documentType":"Project","parentInfo":[{"parentId":"s","parentName":"REG","parentType":"Space"}]}]}`))
		case "/public/core/v3/users":
			_, _ = w.Write([]byte(frsUsersJSON))
		default:
			t.Errorf("unexpected request %s", r.URL.Path)
		}
	})

	c := newTestClient(handler)
	res, err := c.ListProjectInfo(context.Background())
	if err != nil {
		t.Fatalf("ListProjectInfo() error: %v", err)
	}
	if res.FRSError != nil {
		t.Fatalf("unexpected FRS fallback: %v", res.FRSError)
	}
	got := res.Items[0]
	if got.Path != "Finance" || got.CreatedTime != "2025-01-01T00:00:00Z" || got.UpdateTime != "2025-02-01T00:00:00Z" {
		t.Errorf("unexpected project: %+v", got)
	}
	if got.CreatedBy != "jdoe@example.com" {
		t.Errorf("expected createdBy resolved to user name, got %q", got.CreatedBy)
	}
	if got.UpdatedBy != "u2" {
		t.Errorf("expected unknown user ID kept, got %q", got.UpdatedBy)
	}
}

func TestListFolderInfoFRS(t *testing.T) {
	folders := `{"value":[
		{"id":"f1","name":"Staging","documentType":"Folder","parentInfo":[{"parentType":"Space","parentName":"REG"},{"parentType":"Project","parentName":"Finance"}]},
		{"id":"f2","name":"Other","documentType":"Folder","parentInfo":[{"parentType":"Project","parentName":"Sales"}]}]}`
	tests := []struct {
		projectID, projectName, wantPath string
		wantIDs                          []string
	}{
		{"", "", "/frs/api/v1/Folders", []string{"f1", "f2"}},
		{"", "finance", "/frs/api/v1/Folders", []string{"f1"}},
		{"p1", "", "/frs/api/v1/Projects('p1')/Folders", []string{"f1", "f2"}},
	}
	for _, tt := range tests {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if r.URL.Path == "/public/core/v3/users" {
				_, _ = w.Write([]byte(frsUsersJSON))
				return
			}
			if r.URL.Path != tt.wantPath {
				t.Errorf("path = %s, want %s", r.URL.Path, tt.wantPath)
			}
			_, _ = w.Write([]byte(folders))
		})
		c := newTestClient(handler)
		res, err := c.ListFolderInfo(context.Background(), tt.projectID, tt.projectName)
		if err != nil {
			t.Fatalf("ListFolderInfo(%q, %q) error: %v", tt.projectID, tt.projectName, err)
		}
		var ids []string
		for _, it := range res.Items {
			ids = append(ids, it.ID)
		}
		if strings.Join(ids, ",") != strings.Join(tt.wantIDs, ",") {
			t.Errorf("ListFolderInfo(%q, %q) ids = %v, want %v", tt.projectID, tt.projectName, ids, tt.wantIDs)
		}
		if res.Items[0].Path != "Finance/Staging" {
			t.Errorf("expected full path Finance/Staging, got %q", res.Items[0].Path)
		}
	}
}

func TestListFolderInfoFallback(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasPrefix(r.URL.Path, "/frs/"):
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":{"code":"FRS_404","message":"gone"}}`))
		case r.URL.Path == "/public/core/v3/objects":
			if q := r.URL.Query().Get("q"); q != "location=='Finance'" {
				t.Errorf("unexpected objects query %q", q)
			}
			_, _ = w.Write([]byte(`{"count":2,"objects":[
				{"id":"p1","path":"Finance","type":"Project"},
				{"id":"f1","path":"Finance/Staging","type":"Folder","description":"d","updatedBy":"jdoe","updateTime":"2025-02-01T00:00:00Z"}]}`))
		default:
			t.Errorf("unexpected request %s", r.URL.Path)
		}
	})

	c := newTestClient(handler)
	res, err := c.ListFolderInfo(context.Background(), "", "Finance")
	if err != nil {
		t.Fatalf("ListFolderInfo() error: %v", err)
	}
	if res.FRSError == nil {
		t.Fatal("expected FRSError to be set on fallback")
	}
	if len(res.Items) != 1 {
		t.Fatalf("expected only the folder, got %+v", res.Items)
	}
	got := res.Items[0]
	if got.Name != "Staging" || got.UpdatedBy != "jdoe" || got.CreatedTime != "" {
		t.Errorf("unexpected fallback folder: %+v", got)
	}
}

func TestFilterContainersByPath(t *testing.T) {
	items := []ContainerInfo{{Path: "Finance/Staging"}, {Path: "Sales/AccountSearch_v1"}, {Path: "Sales/Orders"}}

	got, err := FilterContainersByPath(items, "sales/acc.*_V1$")
	if err != nil {
		t.Fatalf("FilterContainersByPath() error: %v", err)
	}
	if len(got) != 1 || got[0].Path != "Sales/AccountSearch_v1" {
		t.Errorf("unexpected match: %+v", got)
	}

	all, _ := FilterContainersByPath(items, "")
	if len(all) != 3 {
		t.Errorf("empty pattern should keep all items, got %d", len(all))
	}

	if _, err := FilterContainersByPath(items, "["); err == nil {
		t.Error("expected error for invalid pattern")
	}
}
