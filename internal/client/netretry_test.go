package client

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"
)

// dropFirst returns a handler that closes the connection without a response
// (the client sees EOF) for the first n requests, then answers with body.
func dropFirst(t *testing.T, n int32, calls *atomic.Int32, body string) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) <= n {
			conn, _, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Fatalf("hijack: %v", err)
			}
			_ = conn.Close()
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	})
}

func TestLookupRetriesDroppedConnection(t *testing.T) {
	waits := noSleep(t)
	var calls atomic.Int32
	c := newTestClient(dropFirst(t, 2, &calls, `{"objects":[{"id":"1","path":"P/A","type":"TASKFLOW"}]}`))
	resp, err := c.Lookup(context.Background(), []LookupObject{{Path: "P/A", Type: "TASKFLOW"}})
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	if len(resp.Objects) != 1 {
		t.Errorf("objects = %d, want 1", len(resp.Objects))
	}
	if n := calls.Load(); n != 3 {
		t.Errorf("calls = %d, want 3", n)
	}
	if len(*waits) != 2 {
		t.Errorf("waits = %v, want 2", *waits)
	}
}

func TestLookupGivesUpAfterMaxNetworkRetries(t *testing.T) {
	noSleep(t)
	var calls atomic.Int32
	c := newTestClient(dropFirst(t, 100, &calls, `{}`))
	if _, err := c.Lookup(context.Background(), []LookupObject{{Path: "P/A", Type: "TASKFLOW"}}); err == nil {
		t.Fatal("Lookup() error = nil, want connection error")
	}
	if n := calls.Load(); n != maxNetworkRetries+1 {
		t.Errorf("calls = %d, want %d", n, maxNetworkRetries+1)
	}
}

func TestGetRetriesDroppedConnection(t *testing.T) {
	noSleep(t)
	var calls atomic.Int32
	c := newTestClient(dropFirst(t, 1, &calls, `{"name":"x"}`))
	var out map[string]any
	if err := c.doJSON(context.Background(), http.MethodGet, BaseAPIPathV3+"/anything", nil, &out); err != nil {
		t.Fatalf("doJSON(GET) error = %v", err)
	}
	if n := calls.Load(); n != 2 {
		t.Errorf("calls = %d, want 2", n)
	}
}

// A POST that changes data is not resent: the server may have processed it.
func TestWritePostNotRetriedOnDroppedConnection(t *testing.T) {
	noSleep(t)
	var calls atomic.Int32
	c := newTestClient(dropFirst(t, 1, &calls, `{}`))
	if err := c.doJSON(context.Background(), http.MethodPost, BaseAPIPathV3+"/publish", map[string]string{"a": "b"}, nil); err == nil {
		t.Fatal("doJSON(POST) error = nil, want connection error")
	}
	if n := calls.Load(); n != 1 {
		t.Errorf("calls = %d, want 1 (no resend)", n)
	}
}
