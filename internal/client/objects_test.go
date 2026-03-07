package client

import (
	"context"
	"encoding/json"
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
