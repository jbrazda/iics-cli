package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newCAITestClient creates a test client with caiURL set to the test server URL.
func newCAITestClient(handler http.Handler) (*Client, *httptest.Server) {
	srv := httptest.NewServer(handler)
	c := NewClient(srv.URL+"/login", "user", "pass", WithCAIURL(srv.URL))
	c.SetSession("test-session", srv.URL)
	return c, srv
}

func TestStartPublish(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/active-bpel/asset/v1/publish" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/vnd.api+json" {
			t.Errorf("expected Content-Type application/vnd.api+json, got %s", r.Header.Get("Content-Type"))
		}

		var body publishRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}
		if body.Data.Type != "publish" {
			t.Errorf("expected data.type publish, got %s", body.Data.Type)
		}
		if len(body.Data.Attributes.AssetPaths) != 2 {
			t.Errorf("expected 2 asset paths, got %d", len(body.Data.Attributes.AssetPaths))
		}
		if body.Data.Attributes.AssetPaths[0] != "Explore/Default/MyProcess.PROCESS.xml" {
			t.Errorf("unexpected asset path: %s", body.Data.Attributes.AssetPaths[0])
		}

		resp := PublishJobResponse{
			Data: PublishJobData{
				Type: "publish",
				ID:   "job-abc-123",
				Attributes: PublishJobAttributes{
					JobState:   "NOT_STARTED",
					TotalCount: 2,
				},
			},
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(resp)
	})

	c, srv := newCAITestClient(handler)
	defer srv.Close()

	paths := []string{
		"Explore/Default/MyProcess.PROCESS.xml",
		"Explore/Default/MyConn.AI_CONNECTION.xml",
	}
	result, err := c.StartPublish(context.Background(), "", paths)
	if err != nil {
		t.Fatalf("StartPublish() error: %v", err)
	}
	if result.Data.ID != "job-abc-123" {
		t.Errorf("expected job ID job-abc-123, got %s", result.Data.ID)
	}
	if result.Data.Attributes.JobState != "NOT_STARTED" {
		t.Errorf("expected jobState NOT_STARTED, got %s", result.Data.Attributes.JobState)
	}
}

func TestStartUnpublish(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/active-bpel/asset/v1/unpublish" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body publishRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}
		if body.Data.Type != "unpublish" {
			t.Errorf("expected data.type unpublish, got %s", body.Data.Type)
		}

		resp := PublishJobResponse{
			Data: PublishJobData{
				Type: "unpublish",
				ID:   "job-unpub-456",
				Attributes: PublishJobAttributes{
					JobState:   "NOT_STARTED",
					TotalCount: 1,
				},
			},
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(resp)
	})

	c, srv := newCAITestClient(handler)
	defer srv.Close()

	result, err := c.StartUnpublish(context.Background(), "", []string{"Explore/Default/MyProcess.PROCESS.xml"})
	if err != nil {
		t.Fatalf("StartUnpublish() error: %v", err)
	}
	if result.Data.ID != "job-unpub-456" {
		t.Errorf("expected job ID job-unpub-456, got %s", result.Data.ID)
	}
	if result.Data.Type != "unpublish" {
		t.Errorf("expected type unpublish, got %s", result.Data.Type)
	}
}

func TestGetPublishStatus(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		// full=false: should include /Status suffix
		if r.URL.Path != "/active-bpel/asset/v1/publish/job-abc-123/Status" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		resp := PublishJobResponse{
			Data: PublishJobData{
				Type: "publish",
				ID:   "job-abc-123",
				Attributes: PublishJobAttributes{
					JobState:       "PROCESSING",
					TotalCount:     5,
					ProcessedCount: 2,
				},
			},
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	c, srv := newCAITestClient(handler)
	defer srv.Close()

	result, err := c.GetPublishStatus(context.Background(), "", "job-abc-123", false)
	if err != nil {
		t.Fatalf("GetPublishStatus() error: %v", err)
	}
	if result.Data.Attributes.JobState != "PROCESSING" {
		t.Errorf("expected jobState PROCESSING, got %s", result.Data.Attributes.JobState)
	}
	if result.Data.Attributes.TotalCount != 5 {
		t.Errorf("expected totalCount 5, got %d", result.Data.Attributes.TotalCount)
	}
}

func TestGetPublishStatusFull(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// full=true: should NOT include /Status suffix
		if r.URL.Path != "/active-bpel/asset/v1/publish/job-abc-123" {
			t.Errorf("unexpected path for full=true: %s", r.URL.Path)
		}

		resp := PublishJobResponse{
			Data: PublishJobData{
				Type: "publish",
				ID:   "job-abc-123",
				Attributes: PublishJobAttributes{
					JobState:   "COMPLETED",
					TotalCount: 5,
					AssetPaths: []string{"Explore/Default/MyProcess.PROCESS.xml"},
				},
			},
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	c, srv := newCAITestClient(handler)
	defer srv.Close()

	result, err := c.GetPublishStatus(context.Background(), "", "job-abc-123", true)
	if err != nil {
		t.Fatalf("GetPublishStatus(full=true) error: %v", err)
	}
	if result.Data.Attributes.JobState != "COMPLETED" {
		t.Errorf("expected COMPLETED, got %s", result.Data.Attributes.JobState)
	}
	if len(result.Data.Attributes.AssetPaths) != 1 {
		t.Errorf("expected 1 asset path, got %d", len(result.Data.Attributes.AssetPaths))
	}
}

func TestGetUnpublishStatus(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/active-bpel/asset/v1/unpublish/job-unpub-456/Status" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		resp := PublishJobResponse{
			Data: PublishJobData{
				Type: "unpublish",
				ID:   "job-unpub-456",
				Attributes: PublishJobAttributes{
					JobState: "COMPLETED",
				},
			},
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	c, srv := newCAITestClient(handler)
	defer srv.Close()

	result, err := c.GetUnpublishStatus(context.Background(), "", "job-unpub-456", false)
	if err != nil {
		t.Fatalf("GetUnpublishStatus() error: %v", err)
	}
	if result.Data.Attributes.JobState != "COMPLETED" {
		t.Errorf("expected COMPLETED, got %s", result.Data.Attributes.JobState)
	}
}

func TestPublishIsTerminal(t *testing.T) {
	cases := []struct {
		state    string
		terminal bool
	}{
		{"COMPLETED", true},
		{"FAILED", true},
		{"ERROR", true},
		{"NOT_STARTED", false},
		{"PROCESSING", false},
		{"", false},
	}
	for _, tc := range cases {
		got := PublishIsTerminal(tc.state)
		if got != tc.terminal {
			t.Errorf("PublishIsTerminal(%q) = %v, want %v", tc.state, got, tc.terminal)
		}
	}
}

func TestPublishIsInProgress(t *testing.T) {
	cases := []struct {
		state      string
		inProgress bool
	}{
		{"NOT_STARTED", true},
		{"PROCESSING", true},
		{"COMPLETED", false},
		{"FAILED", false},
		{"ERROR", false},
		{"", false},
	}
	for _, tc := range cases {
		got := PublishIsInProgress(tc.state)
		if got != tc.inProgress {
			t.Errorf("PublishIsInProgress(%q) = %v, want %v", tc.state, got, tc.inProgress)
		}
	}
}

func TestAssetPathFromObject(t *testing.T) {
	cases := []struct {
		obj     Object
		want    string
		wantErr bool
	}{
		{Object{Path: "Default/MyProcess", Type: "PROCESS"}, "Explore/Default/MyProcess.PROCESS.xml", false},
		{Object{Path: "Default/MyConn", Type: "AI_CONNECTION"}, "Explore/Default/MyConn.AI_CONNECTION.xml", false},
		{Object{Path: "Default/MySvc", Type: "AI_SERVICE_CONNECTOR"}, "Explore/Default/MySvc.AI_SERVICE_CONNECTOR.xml", false},
		{Object{Path: "Default/MyMap", Type: "DTEMPLATE"}, "Explore/Default/MyMap.DTEMPLATE.xml", false},
		{Object{Path: "Default/MyGuide", Type: "GUIDE"}, "Explore/Default/MyGuide.GUIDE.xml", false},
		{Object{Path: "Default/MyPO", Type: "PROCESS_OBJECT"}, "Explore/Default/MyPO.PROCESS_OBJECT.xml", false},
		{Object{Path: "Default/MyTask", Type: "MTT"}, "", true},
	}
	for _, tc := range cases {
		got, err := AssetPathFromObject(tc.obj)
		if tc.wantErr {
			if err == nil {
				t.Errorf("AssetPathFromObject(%v) expected error, got nil", tc.obj)
			}
		} else {
			if err != nil {
				t.Errorf("AssetPathFromObject(%v) unexpected error: %v", tc.obj, err)
			}
			if got != tc.want {
				t.Errorf("AssetPathFromObject(%v) = %q, want %q", tc.obj, got, tc.want)
			}
		}
	}
}

func TestSplitIntoBatches(t *testing.T) {
	paths := make([]string, 5)
	for i := range paths {
		paths[i] = "path"
	}

	batches := SplitIntoBatches(paths, 2)
	if len(batches) != 3 {
		t.Errorf("expected 3 batches, got %d", len(batches))
	}
	if len(batches[0]) != 2 {
		t.Errorf("expected batch 0 size 2, got %d", len(batches[0]))
	}
	if len(batches[2]) != 1 {
		t.Errorf("expected batch 2 size 1, got %d", len(batches[2]))
	}

	// Empty input
	empty := SplitIntoBatches(nil, 10)
	if len(empty) != 0 {
		t.Errorf("expected empty batches, got %d", len(empty))
	}
}
