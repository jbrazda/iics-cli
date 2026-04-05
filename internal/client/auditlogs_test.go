package client

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestListAuditLogs(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v2/auditlog" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("batchSize") != "" {
			t.Errorf("expected no batchSize param when Limit=0, got %s", r.URL.Query().Get("batchSize"))
		}
		entries := []AuditLog{
			{
				ID:           "al1",
				Username:     "admin@example.com",
				Category:     "AUTH",
				Event:        "LOGIN",
				EntryTimeUTC: "2025-01-15T10:00:00.000Z",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(entries)
	})

	c := newTestClient(handler)
	logs, err := c.ListAuditLogs(context.Background(), AuditLogListOptions{})
	if err != nil {
		t.Fatalf("ListAuditLogs() error: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(logs))
	}
	if logs[0].ID != "al1" {
		t.Errorf("expected ID al1, got %s", logs[0].ID)
	}
	if logs[0].Category != "AUTH" {
		t.Errorf("expected category AUTH, got %s", logs[0].Category)
	}
}

func TestListAuditLogsPaginated(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("batchSize") != "50" {
			t.Errorf("expected batchSize=50, got %s", r.URL.Query().Get("batchSize"))
		}
		if r.URL.Query().Get("batchId") != "1" {
			t.Errorf("expected batchId=1, got %s", r.URL.Query().Get("batchId"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]AuditLog{})
	})

	c := newTestClient(handler)
	_, err := c.ListAuditLogs(context.Background(), AuditLogListOptions{Limit: 50, Skip: 1})
	if err != nil {
		t.Fatalf("ListAuditLogs() error: %v", err)
	}
}
