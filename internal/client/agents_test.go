package client

import (
	"bytes"
	"context"
	"net/http"
	"testing"
)

func TestGetAgentInstallerInfo(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v2/agent/installerInfo/win64" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("icSessionId") != "test-session" {
			t.Errorf("expected v2 session header, got %q", r.Header.Get("icSessionId"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"@type":"agentInstallerInfo","downloadUrl":"https://cdn.example/agent64.6403.exe","installToken":"tok123","checksumDownloadUrl":"https://cdn.example/agent64.6403_win64.sha256"}`))
	})

	c := newTestClient(handler)
	info, err := c.GetAgentInstallerInfo(context.Background(), "win64")
	if err != nil {
		t.Fatalf("GetAgentInstallerInfo() error: %v", err)
	}
	if info.DownloadURL != "https://cdn.example/agent64.6403.exe" {
		t.Errorf("unexpected downloadUrl: %s", info.DownloadURL)
	}
	if info.InstallToken != "tok123" {
		t.Errorf("unexpected installToken: %s", info.InstallToken)
	}
	if info.ChecksumDownloadURL == "" {
		t.Errorf("expected checksumDownloadUrl")
	}
}

func TestDownloadFileAndFetchText(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/binary":
			_, _ = w.Write([]byte("installer-bytes"))
		case "/checksum":
			_, _ = w.Write([]byte("abc123  agent64.exe\n"))
		default:
			http.NotFound(w, r)
		}
	})

	c := newTestClient(handler)
	base := c.BaseAPIURL()

	var buf bytes.Buffer
	n, err := c.DownloadFile(context.Background(), base+"/binary", &buf)
	if err != nil {
		t.Fatalf("DownloadFile() error: %v", err)
	}
	if n != int64(len("installer-bytes")) || buf.String() != "installer-bytes" {
		t.Errorf("unexpected download: n=%d body=%q", n, buf.String())
	}

	text, err := c.FetchText(context.Background(), base+"/checksum")
	if err != nil {
		t.Fatalf("FetchText() error: %v", err)
	}
	if text != "abc123  agent64.exe\n" {
		t.Errorf("unexpected checksum text: %q", text)
	}
}
