package client

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
)

func TestGetAgentGroupConfigs(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v2/runtimeEnvironment/g1/configs" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("icSessionId") != "test-session" {
			t.Errorf("expected v2 session header")
		}
		_, _ = w.Write([]byte(`{"Data_Integration_Server":[{"name":"JVMOption1","value":"-Xmx2048m"}]}`))
	})
	c := newTestClient(handler)
	cfg, err := c.GetAgentGroupConfigs(context.Background(), "g1")
	if err != nil {
		t.Fatalf("GetAgentGroupConfigs() error: %v", err)
	}
	svc := cfg["Data_Integration_Server"]
	if len(svc) != 1 || svc[0]["value"] != "-Xmx2048m" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestUpdateAgentGroupConfigs(t *testing.T) {
	want := AgentGroupServiceConfig{
		"Data_Integration_Server": {{"name": "JVMOption1", "value": "-Xmx4096m"}},
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/v2/runtimeEnvironment/g1/configs" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var got AgentGroupServiceConfig
		_ = json.NewDecoder(r.Body).Decode(&got)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("body = %+v, want %+v", got, want)
		}
		_ = json.NewEncoder(w).Encode(got)
	})
	c := newTestClient(handler)
	got, err := c.UpdateAgentGroupConfigs(context.Background(), "g1", want)
	if err != nil {
		t.Fatalf("UpdateAgentGroupConfigs() error: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("response = %+v, want %+v", got, want)
	}
}
