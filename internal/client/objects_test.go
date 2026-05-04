package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestClient(handler http.Handler) *Client {
	srv := httptest.NewServer(handler)
	c := NewClient(srv.URL+"/login", "user", "pass")
	c.SetSession("test-session", srv.URL)
	return c
}

func TestListObjects(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("INFA-SESSION-ID") != "test-session" {
			t.Errorf("expected session header, got %s", r.Header.Get("INFA-SESSION-ID"))
		}

		// Check query params
		q := r.URL.Query().Get("q")
		if q != "type=='MTT'" {
			t.Errorf("expected query type=='MTT', got %s", q)
		}

		resp := ObjectsListResponse{
			Count: 2,
			Objects: []Object{
				{ID: "obj1", Path: "Default/Task1", Type: "MTT", UpdatedBy: "admin"},
				{ID: "obj2", Path: "Default/Task2", Type: "MTT", UpdatedBy: "admin"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	c := newTestClient(handler)
	resp, err := c.ListObjects(context.Background(), ObjectsListOptions{Type: "MTT"})
	if err != nil {
		t.Fatalf("ListObjects() error: %v", err)
	}

	if resp.Count != 2 {
		t.Errorf("expected count 2, got %d", resp.Count)
	}
	if len(resp.Objects) != 2 {
		t.Errorf("expected 2 objects, got %d", len(resp.Objects))
	}
	if resp.Objects[0].ID != "obj1" {
		t.Errorf("expected first object ID obj1, got %s", resp.Objects[0].ID)
	}
}

func TestListAllObjects(t *testing.T) {
	// Simulate 3 pages: two full pages of 200, then a short page of 50.
	// Total expected: 450 objects.
	pageSize := defaultPageSize

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		skip := 0
		if s := r.URL.Query().Get("skip"); s != "" {
			fmt.Sscanf(s, "%d", &skip)
		}
		limit := pageSize
		if l := r.URL.Query().Get("limit"); l != "" {
			fmt.Sscanf(l, "%d", &limit)
		}

		// Build a page of objects
		count := limit // full page
		if skip >= pageSize*2 {
			count = 50 // last (short) page
		}

		objects := make([]Object, count)
		for i := range objects {
			objects[i] = Object{
				ID:   fmt.Sprintf("obj-%d", skip+i),
				Type: "MTT",
				Path: fmt.Sprintf("Default/Task%d", skip+i),
			}
		}
		resp := ObjectsListResponse{Count: count, Objects: objects}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	c := newTestClient(handler)
	resp, err := c.ListAllObjects(context.Background(), ObjectsListOptions{Type: "MTT"}, nil)
	if err != nil {
		t.Fatalf("ListAllObjects() error: %v", err)
	}

	want := pageSize*2 + 50 // 450
	if len(resp.Objects) != want {
		t.Errorf("expected %d objects, got %d", want, len(resp.Objects))
	}
	if resp.Count != want {
		t.Errorf("expected Count=%d, got %d", want, resp.Count)
	}
}

func TestListAllObjectsSinglePage(t *testing.T) {
	// When the API returns fewer than defaultPageSize, no second request is made.
	calls := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		resp := ObjectsListResponse{
			Count:   3,
			Objects: []Object{{ID: "a"}, {ID: "b"}, {ID: "c"}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	c := newTestClient(handler)
	resp, err := c.ListAllObjects(context.Background(), ObjectsListOptions{}, nil)
	if err != nil {
		t.Fatalf("ListAllObjects() error: %v", err)
	}
	if len(resp.Objects) != 3 {
		t.Errorf("expected 3 objects, got %d", len(resp.Objects))
	}
	if calls != 1 {
		t.Errorf("expected 1 API call for short result set, got %d", calls)
	}
}

func TestListObjectsAutoLogin(t *testing.T) {
	callCount := 0

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		// First call: return 401 to trigger re-login
		if callCount == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Second call: login endpoint
		if callCount == 2 {
			resp := LoginResponse{
				Products: []Product{{BaseAPIURL: r.URL.String()}},
				UserInfo: UserInfo{SessionID: "new-session"},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		// Third call: actual objects request with new session
		resp := ObjectsListResponse{
			Count:   1,
			Objects: []Object{{ID: "obj1", Type: "MTT"}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	c := newTestClient(handler)
	resp, err := c.ListObjects(context.Background(), ObjectsListOptions{})
	if err != nil {
		t.Fatalf("ListObjects() error: %v", err)
	}

	if resp.Count != 1 {
		t.Errorf("expected count 1, got %d", resp.Count)
	}
	if callCount < 3 {
		t.Errorf("expected at least 3 calls (401 + login + retry), got %d", callCount)
	}
}

func TestGetObjectDependencies(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/public/core/v3/objects/obj1/references" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("refType") != "uses" {
			t.Errorf("expected refType=uses, got %s", r.URL.Query().Get("refType"))
		}
		resp := ObjectDependenciesResponse{
			ID:    "obj1",
			Count: 1,
			References: []ObjectReference{
				{AppContextID: "dep1", Path: "Sales/Mapping1", Type: "MTT"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	c := newTestClient(handler)
	resp, err := c.GetObjectDependencies(context.Background(), "obj1", "uses", 50, 0)
	if err != nil {
		t.Fatalf("GetObjectDependencies() error: %v", err)
	}
	if len(resp.References) != 1 {
		t.Fatalf("expected 1 References entry, got %d", len(resp.References))
	}
	got := resp.References[0]
	if got.AppContextID != "dep1" {
		t.Errorf("expected AppContextID dep1, got %s", got.AppContextID)
	}
	wantLocation := "Explore/Sales/Mapping1.MTT"
	if got.Location != wantLocation {
		t.Errorf("expected Location %q, got %q", wantLocation, got.Location)
	}
}

func TestGetAllObjectDependencies(t *testing.T) {
	// Simulate 2 pages: first full (50 items), second short (10 items). Total: 60.
	calls := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		skip := 0
		fmt.Sscanf(r.URL.Query().Get("skip"), "%d", &skip)

		count := depPageSize
		if skip >= depPageSize {
			count = 10
		}
		refs := make([]ObjectReference, count)
		for i := range refs {
			refs[i] = ObjectReference{
				AppContextID: fmt.Sprintf("dep-%d", skip+i),
				Path:         fmt.Sprintf("Sales/Obj%d", skip+i),
				Type:         "MTT",
			}
		}
		resp := ObjectDependenciesResponse{ID: "obj1", Count: count, References: refs}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	c := newTestClient(handler)
	resp, err := c.GetAllObjectDependencies(context.Background(), "obj1", "uses")
	if err != nil {
		t.Fatalf("GetAllObjectDependencies() error: %v", err)
	}
	want := depPageSize + 10
	if len(resp.References) != want {
		t.Errorf("expected %d References entries, got %d", want, len(resp.References))
	}
	if calls != 2 {
		t.Errorf("expected 2 API calls, got %d", calls)
	}
	// Verify Location is populated on paginated results
	for _, ref := range resp.References {
		wantLoc := "Explore/" + ref.Path + "." + ref.Type
		if ref.Location != wantLoc {
			t.Errorf("expected Location %q, got %q", wantLoc, ref.Location)
		}
	}
}
