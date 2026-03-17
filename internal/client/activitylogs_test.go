package client

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestListActivityLogs(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/activity/activityLog" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("taskId") != "task123" {
			t.Errorf("expected taskId=task123, got %s", r.URL.Query().Get("taskId"))
		}

		entries := []ActivityLogEntry{
			{ID: "log1", ObjectName: "MyTask", State: 1, RunID: 42, TotalSuccessRows: 39},
			{ID: "log2", ObjectName: "MyTask", State: 3, RunID: 43},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entries)
	})

	c := newTestClient(handler)
	logs, err := c.ListActivityLogs(context.Background(), ActivityLogListOptions{TaskID: "task123"})
	if err != nil {
		t.Fatalf("ListActivityLogs() error: %v", err)
	}
	if len(logs) != 2 {
		t.Errorf("expected 2 entries, got %d", len(logs))
	}
	if logs[0].ID != "log1" {
		t.Errorf("expected ID log1, got %s", logs[0].ID)
	}
	if logs[0].TotalSuccessRows != 39 {
		t.Errorf("expected TotalSuccessRows 39, got %d", logs[0].TotalSuccessRows)
	}
	if logs[1].State != 3 {
		t.Errorf("expected state 3, got %d", logs[1].State)
	}
}

func TestListActivityLogsWithRowLimit(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("rowLimit") != "50" {
			t.Errorf("expected rowLimit=50, got %s", r.URL.Query().Get("rowLimit"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
	})

	c := newTestClient(handler)
	_, err := c.ListActivityLogs(context.Background(), ActivityLogListOptions{RowLimit: 50})
	if err != nil {
		t.Fatalf("ListActivityLogs() error: %v", err)
	}
}

func TestGetActivityLog(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/activity/activityLog/log999" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		entry := ActivityLogEntry{
			ID:         "log999",
			ObjectName: "SomeTask",
			State:      1,
			RunID:      100,
			Entries: []ActivityLogEntry{
				{ID: "child1", ObjectName: "SomeTask", State: 1, RunID: 100},
			},
			TransformationEntries: []TransformationEntry{
				{ID: "tx1", TxName: "Source1", TxType: "SOURCE", SuccessRows: 39, FailedRows: 0},
			},
			LogEntryItemAttrs: map[string]string{
				"ERROR_CODE": "0",
				"Session Log File Name": "s_test.log",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entry)
	})

	c := newTestClient(handler)
	entry, err := c.GetActivityLog(context.Background(), "log999")
	if err != nil {
		t.Fatalf("GetActivityLog() error: %v", err)
	}
	if entry.ID != "log999" {
		t.Errorf("expected ID log999, got %s", entry.ID)
	}
	if entry.ObjectName != "SomeTask" {
		t.Errorf("expected ObjectName SomeTask, got %s", entry.ObjectName)
	}
	if len(entry.Entries) != 1 {
		t.Errorf("expected 1 child entry, got %d", len(entry.Entries))
	}
	if len(entry.TransformationEntries) != 1 {
		t.Errorf("expected 1 transformation entry, got %d", len(entry.TransformationEntries))
	}
	if entry.TransformationEntries[0].TxName != "Source1" {
		t.Errorf("expected TxName Source1, got %s", entry.TransformationEntries[0].TxName)
	}
	if len(entry.LogEntryItemAttrs) != 2 {
		t.Errorf("expected 2 item attrs, got %d", len(entry.LogEntryItemAttrs))
	}
}
