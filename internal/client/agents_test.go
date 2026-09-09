package client

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestListAgentsQueryParams(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/agent" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("icSessionId") != "test-session" {
			t.Errorf("expected v2 session header, got %q", r.Header.Get("icSessionId"))
		}
		q := r.URL.Query()
		if q.Get("basicInfo") != "true" || q.Get("includeUnassignedOnly") != "true" {
			t.Errorf("unexpected query: %s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`[{"@type":"agent","id":"a1","name":"A1","spiUrl":"https://spi","federatedId":"fed-1","serverUrl":"","createTimeUTC":"2020-01-01T00:00:00Z"}]`))
	})
	c := newTestClient(handler)
	agents, err := c.ListAgents(context.Background(), AgentListOptions{BasicInfo: true, IncludeUnassignedOnly: true})
	if err != nil {
		t.Fatalf("ListAgents() error: %v", err)
	}
	if len(agents) != 1 || agents[0].ID != "a1" {
		t.Fatalf("unexpected agents: %+v", agents)
	}
	if agents[0].SpiURL != "https://spi" || agents[0].FederatedID != "fed-1" || agents[0].CreateTimeUTC == "" || agents[0].Type != "agent" {
		t.Errorf("extended fields not parsed: %+v", agents[0])
	}
}

func TestGetAgentByName(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/agent/name/My%20Agent" && r.URL.EscapedPath() != "/api/v2/agent/name/My%20Agent" {
			t.Errorf("unexpected path: %s", r.URL.EscapedPath())
		}
		_ = json.NewEncoder(w).Encode(Agent{ID: "a9", Name: "My Agent"})
	})
	c := newTestClient(handler)
	a, err := c.GetAgentByName(context.Background(), "My Agent")
	if err != nil {
		t.Fatalf("GetAgentByName() error: %v", err)
	}
	if a.ID != "a9" {
		t.Errorf("expected a9, got %s", a.ID)
	}
}

func TestGetAgentDetailsNested(t *testing.T) {
	const body = `{
	  "@type":"agentdetails","id":"a1","name":"A1","platformAgent":true,"serverUrl":"https://x",
	  "agentConfigs":[{"@type":"engineConfig","type":"JVM","name":"opt","value":"-Xmx","defaultValue":"-Xms","customized":true}],
	  "agentEngines":[
	    {"@type":"AgentEngine",
	     "agentEngineStatus":{"appname":"process-engine","appDisplayName":"Process Server","appversion":"1.2","status":"RUNNING","desiredStatus":"RUNNING","subState":"0"},
	     "agentEngineConfigs":[{"@type":"engineConfig","type":"server","name":"host-name","value":"'HOST'","platform":"all","customized":false,"defaultValue":"'localhost'"}]}
	  ]}`
	var full bool
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/agent/details/a1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("onlyStatus"); (got == "false") != full {
			t.Errorf("onlyStatus=%q for full=%v", got, full)
		}
		_, _ = w.Write([]byte(body))
	})
	c := newTestClient(handler)

	full = false
	if _, err := c.GetAgentDetails(context.Background(), "a1", false); err != nil {
		t.Fatalf("GetAgentDetails(full=false) error: %v", err)
	}

	full = true
	d, err := c.GetAgentDetails(context.Background(), "a1", true)
	if err != nil {
		t.Fatalf("GetAgentDetails(full=true) error: %v", err)
	}
	if !d.PlatformAgent || d.ServerURL != "https://x" {
		t.Errorf("summary fields not parsed: %+v", *d)
	}
	if len(d.AgentEngines) != 1 {
		t.Fatalf("expected 1 engine, got %d", len(d.AgentEngines))
	}
	e := d.AgentEngines[0]
	if e.AgentEngineStatus.Status != "RUNNING" || e.AgentEngineStatus.AppName != "process-engine" {
		t.Errorf("engine status not parsed: %+v", e.AgentEngineStatus)
	}
	if len(e.AgentEngineConfigs) != 1 || e.AgentEngineConfigs[0].DefaultValue != "'localhost'" {
		t.Errorf("engine configs not parsed: %+v", e.AgentEngineConfigs)
	}
	if len(d.AgentConfigs) != 1 || !d.AgentConfigs[0].Customized {
		t.Errorf("agent configs not parsed: %+v", d.AgentConfigs)
	}
}

func TestSetAgentServiceState(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/public/core/v3/agent/service" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("INFA-SESSION-ID") != "test-session" {
			t.Errorf("expected v3 session header, got %q", r.Header.Get("INFA-SESSION-ID"))
		}
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["agentId"] != "fed-1" || body["serviceName"] != "Data Integration Server" || body["serviceAction"] != "stop" {
			t.Errorf("unexpected body: %+v", body)
		}
		w.WriteHeader(http.StatusOK)
	})
	c := newTestClient(handler)
	if err := c.SetAgentServiceState(context.Background(), "fed-1", "Data Integration Server", AgentServiceStop); err != nil {
		t.Fatalf("SetAgentServiceState() error: %v", err)
	}
}

func TestDeleteAgent(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/v2/agent/a1" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	})
	c := newTestClient(handler)
	if err := c.DeleteAgent(context.Background(), "a1"); err != nil {
		t.Fatalf("DeleteAgent() error: %v", err)
	}
}

func TestFindAgentByHostname(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]Agent{
			{ID: "a1", Name: "A1", AgentHost: "host01"},
			{ID: "a2", Name: "A2", AgentHost: "host02"},
		})
	})
	c := newTestClient(handler)
	a, err := c.FindAgent(context.Background(), AgentSelector{Hostname: "HOST02"})
	if err != nil {
		t.Fatalf("FindAgent() error: %v", err)
	}
	if a.ID != "a2" {
		t.Errorf("expected a2, got %s", a.ID)
	}
}

func TestFindAgentByFederatedID(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/agent" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode([]Agent{
			{ID: "a4", Name: "A4", FederatedID: "fed-4"},
			{ID: "a5", Name: "A5", FederatedID: "fed-5"},
		})
	})
	c := newTestClient(handler)
	a, err := c.FindAgent(context.Background(), AgentSelector{FederatedID: "fed-5"})
	if err != nil {
		t.Fatalf("FindAgent() error: %v", err)
	}
	if a.ID != "a5" {
		t.Errorf("expected a5, got %s", a.ID)
	}
}

func TestFindAgentByFederatedIDViaRuntimeFallback(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/agent":
			_ = json.NewEncoder(w).Encode([]Agent{{ID: "a1", Name: "A1", FederatedID: "fed-1"}})
		case "/api/v2/runtimeEnvironment":
			_ = json.NewEncoder(w).Encode([]RuntimeEnvironment{
				{ID: "e1", Name: "Env", Agents: []RuntimeEnvironmentAgent{{ID: "a5", FederatedID: "fed-5"}}},
			})
		case "/api/v2/agent/a5":
			_ = json.NewEncoder(w).Encode(Agent{ID: "a5", Name: "A5"})
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	})
	c := newTestClient(handler)
	a, err := c.FindAgent(context.Background(), AgentSelector{FederatedID: "fed-5"})
	if err != nil {
		t.Fatalf("FindAgent() error: %v", err)
	}
	if a.ID != "a5" {
		t.Errorf("expected a5, got %s", a.ID)
	}
}

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
