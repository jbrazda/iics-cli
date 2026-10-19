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
		if r.URL.Query().Has("type") {
			t.Errorf("type is filtered client-side and should not be sent")
		}

		conns := []Connection{
			{ID: "c1", Name: "My Conn", Type: "TOOLKIT"},
			{ID: "c2", Name: "Other Conn", Type: "TOOLKIT"},
			{ID: "c3", Name: "File Conn", Type: "CSVFile"},
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
		if r.URL.Path != "/api/v2/connection/conn123" {
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
		if r.URL.Path != "/api/v2/connection/conn123" {
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
		if r.URL.Path != "/api/v2/connection" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["@type"] != "connection" {
			t.Errorf("expected @type connection, got %v", body["@type"])
		}
		if body["name"] != "New Conn" {
			t.Errorf("expected name 'New Conn', got %v", body["name"])
		}
		if body["customAttr"] != "kept" {
			t.Errorf("expected connector-specific attribute to be sent, got %v", body["customAttr"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"@type":"connection","id":"new123","name":"New Conn","type":"JDBC","port":1433}`))
	})

	c := newTestClient(handler)
	conn, err := c.CreateConnection(context.Background(), map[string]interface{}{"name": "New Conn", "type": "JDBC", "customAttr": "kept"})
	if err != nil {
		t.Fatalf("CreateConnection() error: %v", err)
	}
	if conn.ID != "new123" {
		t.Errorf("expected ID new123, got %s", conn.ID)
	}
	if conn.Port != 1433 {
		t.Errorf("expected port 1433, got %d", conn.Port)
	}
}

func TestUpdateConnection(t *testing.T) {
	for _, partial := range []bool{false, true} {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}
			if r.URL.Path != "/api/v2/connection/conn123" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			mode := r.Header.Get("Update-Mode")
			if partial && mode != "PARTIAL" {
				t.Errorf("expected Update-Mode PARTIAL, got %q", mode)
			}
			if !partial && mode != "" {
				t.Errorf("expected no Update-Mode header, got %q", mode)
			}
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["@type"] != "connection" {
				t.Errorf("expected @type connection, got %v", body["@type"])
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"conn123","name":"C","type":"JDBC"}`))
		})

		c := newTestClient(handler)
		conn, err := c.UpdateConnection(context.Background(), "conn123", map[string]interface{}{"description": "d"}, partial)
		if err != nil {
			t.Fatalf("UpdateConnection(partial=%v) error: %v", partial, err)
		}
		if conn.ID != "conn123" {
			t.Errorf("expected ID conn123, got %s", conn.ID)
		}
	}
}

func TestListConnectionsClientSideFilter(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"id":"1","name":"Sales Prod","type":"Salesforce"},
			{"id":"2","name":"Sales Dev","type":"Salesforce"},
			{"id":"3","name":"Sales DB","type":"SqlServer2019"}]`))
	})

	c := newTestClient(handler)
	conns, err := c.ListConnections(context.Background(), ConnectionListOptions{Type: "salesforce", Name: "prod"})
	if err != nil {
		t.Fatalf("ListConnections() error: %v", err)
	}
	if len(conns) != 1 || conns[0].ID != "1" {
		t.Errorf("expected only connection 1, got %+v", conns)
	}
}

func TestTestConnection(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v2/connection/test/conn123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("icSessionId") != "test-session" {
			t.Errorf("expected V2 session header")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"@type":"success","description":"The test for this connection was successful."}`))
	})

	c := newTestClient(handler)
	res, err := c.TestConnection(context.Background(), "conn123")
	if err != nil {
		t.Fatalf("TestConnection() error: %v", err)
	}
	if res.AtType != "success" || res.Description == "" {
		t.Errorf("unexpected result: %+v", res)
	}
}

func TestSearchConnections(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/connection/search" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("uiType") != "Salesforce" || q.Get("runtimeEnvironmentId") != "rt1" {
			t.Errorf("unexpected query: %s", r.URL.RawQuery)
		}
		if q.Has("agentId") {
			t.Errorf("agentId should not be sent when empty")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":"1","name":"SF","type":"Salesforce"}]`))
	})

	c := newTestClient(handler)
	conns, err := c.SearchConnections(context.Background(), ConnectionSearchOptions{UIType: "Salesforce", RuntimeEnvironmentID: "rt1"})
	if err != nil {
		t.Fatalf("SearchConnections() error: %v", err)
	}
	if len(conns) != 1 {
		t.Errorf("expected 1 connection, got %d", len(conns))
	}

	if _, err := c.SearchConnections(context.Background(), ConnectionSearchOptions{}); err == nil {
		t.Error("expected error when type is missing")
	}
}

func TestListConnectionObjects(t *testing.T) {
	tests := []struct {
		opts     ConnectionObjectsOptions
		wantPath string
	}{
		{ConnectionObjectsOptions{ID: "c1"}, "/api/v2/connection/source/c1"},
		{ConnectionObjectsOptions{ID: "c1", Target: true}, "/api/v2/connection/target/c1"},
		{ConnectionObjectsOptions{Name: "My Conn", SearchPattern: "Acc"}, "/api/v2/connection/source/name/My%20Conn"},
	}
	for _, tt := range tests {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.EscapedPath() != tt.wantPath {
				t.Errorf("path = %s, want %s", r.URL.EscapedPath(), tt.wantPath)
			}
			if got := r.URL.Query().Get("searchPattern"); got != tt.opts.SearchPattern {
				t.Errorf("searchPattern = %q, want %q", got, tt.opts.SearchPattern)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"@type":"connListItem","id":"0","name":"Account"}]`))
		})
		c := newTestClient(handler)
		items, err := c.ListConnectionObjects(context.Background(), tt.opts)
		if err != nil {
			t.Fatalf("ListConnectionObjects(%+v) error: %v", tt.opts, err)
		}
		if len(items) != 1 || items[0].Name != "Account" {
			t.Errorf("unexpected items: %+v", items)
		}
	}

	c := newTestClient(http.NotFoundHandler())
	if _, err := c.ListConnectionObjects(context.Background(), ConnectionObjectsOptions{}); err == nil {
		t.Error("expected error without ID or name")
	}
}

func TestGetConnectionMetadata(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/connection/target/c1/metadata" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"@type":"runtimeAttribute","name":"UPDATE","dataType":"STRING","defaultValue":"None","label":"","mandatory":false,"maxLength":0,"sessionVarAllowed":false,"uniqueName":"UPDATE","possibleValues":["Update as Update","None"]}]`))
	})

	c := newTestClient(handler)
	attrs, err := c.GetConnectionMetadata(context.Background(), "c1", true)
	if err != nil {
		t.Fatalf("GetConnectionMetadata() error: %v", err)
	}
	if len(attrs) != 1 || attrs[0].Name != "UPDATE" || len(attrs[0].PossibleValues) != 2 {
		t.Errorf("unexpected attributes: %+v", attrs)
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
