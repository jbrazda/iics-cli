package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListConnections(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("type") != "TOOLKIT" {
			t.Errorf("expected type=TOOLKIT, got %s", r.URL.Query().Get("type"))
		}

		conns := []Connection{
			{ID: "c1", Name: "My Conn", Type: "TOOLKIT"},
			{ID: "c2", Name: "Other Conn", Type: "TOOLKIT"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(conns)
	})

	c := newTestClient(handler)
	conns, err := c.ListConnections(context.Background(), ConnectionListOptions{Type: "TOOLKIT"})
	if err != nil {
		t.Fatalf("ListConnections() error: %v", err)
	}

	if len(conns) != 2 {
		t.Errorf("expected 2 connections, got %d", len(conns))
	}
	if conns[0].Name != "My Conn" {
		t.Errorf("expected 'My Conn', got %s", conns[0].Name)
	}
}

func TestGetConnection(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/public/core/v3/connections/conn123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		conn := Connection{ID: "conn123", Name: "Test Connection", Type: "JDBC"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(conn)
	})

	c := newTestClient(handler)
	conn, err := c.GetConnection(context.Background(), "conn123")
	if err != nil {
		t.Fatalf("GetConnection() error: %v", err)
	}
	if conn.ID != "conn123" {
		t.Errorf("expected ID conn123, got %s", conn.ID)
	}
	if conn.Type != "JDBC" {
		t.Errorf("expected type JDBC, got %s", conn.Type)
	}
}

func TestDeleteConnection(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/public/core/v3/connections/conn123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	c := newTestClient(handler)
	err := c.DeleteConnection(context.Background(), "conn123")
	if err != nil {
		t.Fatalf("DeleteConnection() error: %v", err)
	}
}

func TestCreateConnection(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var conn Connection
		json.NewDecoder(r.Body).Decode(&conn)
		if conn.Name != "New Conn" {
			t.Errorf("expected name 'New Conn', got %s", conn.Name)
		}

		conn.ID = "new123"
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(conn)
	})

	c := newTestClient(handler)
	conn, err := c.CreateConnection(context.Background(), &Connection{Name: "New Conn", Type: "JDBC"})
	if err != nil {
		t.Fatalf("CreateConnection() error: %v", err)
	}
	if conn.ID != "new123" {
		t.Errorf("expected ID new123, got %s", conn.ID)
	}
}

func TestAPIErrorNotFound(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"code":    "NOT_FOUND",
			"message": "Connection not found",
		})
	})

	c := newTestClient(handler)
	_, err := c.GetConnection(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if !apiErr.IsNotFound() {
		t.Errorf("expected IsNotFound=true, got status %d", apiErr.StatusCode)
	}
}

func TestNewTestClientHelper(t *testing.T) {
	// Verify the newTestClient helper sets session correctly
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sid := r.Header.Get("INFA-SESSION-ID")
		if sid != "test-session" {
			t.Errorf("expected session 'test-session', got '%s'", sid)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("[]"))
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	c := NewClient(srv.URL+"/login", "user", "pass")
	c.SetSession("test-session", srv.URL)

	if c.SessionID() != "test-session" {
		t.Errorf("SessionID() = %q, want 'test-session'", c.SessionID())
	}
}
