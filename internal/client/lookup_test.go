package client

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestLookupByID(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var req LookupRequest
		json.NewDecoder(r.Body).Decode(&req)

		if len(req.Objects) != 1 {
			t.Fatalf("expected 1 lookup object, got %d", len(req.Objects))
		}
		if req.Objects[0].ID != "abc123" {
			t.Errorf("expected ID abc123, got %s", req.Objects[0].ID)
		}

		resp := LookupResponse{
			Objects: []LookupResult{
				{ID: "abc123", Path: "Default/MyMapping", Type: "DTEMPLATE", Description: "Test mapping"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	c := newTestClient(handler)
	resp, err := c.Lookup(context.Background(), []LookupObject{{ID: "abc123"}})
	if err != nil {
		t.Fatalf("Lookup() error: %v", err)
	}

	if len(resp.Objects) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Objects))
	}
	if resp.Objects[0].Path != "Default/MyMapping" {
		t.Errorf("expected path 'Default/MyMapping', got %s", resp.Objects[0].Path)
	}
}

func TestLookupByPathAndType(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req LookupRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Objects[0].Path != "Default/Sync Task1" {
			t.Errorf("expected path 'Default/Sync Task1', got %s", req.Objects[0].Path)
		}
		if req.Objects[0].Type != "DSS" {
			t.Errorf("expected type DSS, got %s", req.Objects[0].Type)
		}

		resp := LookupResponse{
			Objects: []LookupResult{
				{ID: "xyz789", Path: "Default/Sync Task1", Type: "DSS"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	c := newTestClient(handler)
	resp, err := c.Lookup(context.Background(), []LookupObject{
		{Path: "Default/Sync Task1", Type: "DSS"},
	})
	if err != nil {
		t.Fatalf("Lookup() error: %v", err)
	}
	if resp.Objects[0].ID != "xyz789" {
		t.Errorf("expected ID xyz789, got %s", resp.Objects[0].ID)
	}
}
