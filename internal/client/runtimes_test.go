package client

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestListRuntimeEnvironments(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/runtimeEnvironment" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}

		runtimes := []RuntimeEnvironment{
			{
				ID:          "rt1",
				Name:        "Runtime One",
				FederatedID: "fed1",
				Agents: []RuntimeEnvironmentAgent{
					{ID: "a1", Name: "Agent One", Active: true, AgentVersion: "75.12"},
				},
			},
			{
				ID:          "rt2",
				Name:        "Runtime Two",
				FederatedID: "fed2",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(runtimes)
	})

	c := newTestClient(handler)
	runtimes, err := c.ListRuntimeEnvironments(context.Background(), RuntimeListOptions{})
	if err != nil {
		t.Fatalf("ListRuntimeEnvironments() error: %v", err)
	}
	if len(runtimes) != 2 {
		t.Errorf("expected 2 runtimes, got %d", len(runtimes))
	}
	if runtimes[0].Name != "Runtime One" {
		t.Errorf("expected 'Runtime One', got %s", runtimes[0].Name)
	}
	if len(runtimes[0].Agents) != 1 {
		t.Errorf("expected 1 agent on runtime[0], got %d", len(runtimes[0].Agents))
	}
	if runtimes[0].Agents[0].AgentVersion != "75.12" {
		t.Errorf("expected agent version '75.12', got %s", runtimes[0].Agents[0].AgentVersion)
	}
}

func TestGetRuntimeEnvironment(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/runtimeEnvironment/rt123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		rt := RuntimeEnvironment{
			ID:          "rt123",
			Name:        "Test Runtime",
			FederatedID: "fed123",
			Agents: []RuntimeEnvironmentAgent{
				{ID: "a1", Name: "Agent One", Active: true, Platform: "win64"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rt)
	})

	c := newTestClient(handler)
	rt, err := c.GetRuntimeEnvironment(context.Background(), "rt123")
	if err != nil {
		t.Fatalf("GetRuntimeEnvironment() error: %v", err)
	}
	if rt.ID != "rt123" {
		t.Errorf("expected ID rt123, got %s", rt.ID)
	}
	if rt.FederatedID != "fed123" {
		t.Errorf("expected federatedId fed123, got %s", rt.FederatedID)
	}
	if len(rt.Agents) != 1 {
		t.Errorf("expected 1 agent, got %d", len(rt.Agents))
	}
}

func TestGetRuntimeEnvironmentByName(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/runtimeEnvironment/name/My Runtime" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		rt := RuntimeEnvironment{
			ID:          "rt-name-lookup",
			Name:        "My Runtime",
			FederatedID: "fed-name",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rt)
	})

	c := newTestClient(handler)
	rt, err := c.GetRuntimeEnvironmentByName(context.Background(), "My Runtime")
	if err != nil {
		t.Fatalf("GetRuntimeEnvironmentByName() error: %v", err)
	}
	if rt.Name != "My Runtime" {
		t.Errorf("expected name 'My Runtime', got %s", rt.Name)
	}
	if rt.ID != "rt-name-lookup" {
		t.Errorf("expected ID rt-name-lookup, got %s", rt.ID)
	}
}

func TestCreateRuntimeEnvironment(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v2/runtimeEnvironment" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var rt RuntimeEnvironment
		json.NewDecoder(r.Body).Decode(&rt)
		rt.ID = "new-rt"
		rt.FederatedID = "fed-new"
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(rt)
	})

	c := newTestClient(handler)
	rt, err := c.CreateRuntimeEnvironment(context.Background(), &RuntimeEnvironment{Name: "New Runtime"})
	if err != nil {
		t.Fatalf("CreateRuntimeEnvironment() error: %v", err)
	}
	if rt.ID != "new-rt" {
		t.Errorf("expected ID new-rt, got %s", rt.ID)
	}
	if rt.FederatedID != "fed-new" {
		t.Errorf("expected federatedId fed-new, got %s", rt.FederatedID)
	}
}

func TestUpdateRuntimeEnvironment(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/api/v2/runtimeEnvironment/rt123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var rt RuntimeEnvironment
		json.NewDecoder(r.Body).Decode(&rt)
		rt.ID = "rt123"
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rt)
	})

	c := newTestClient(handler)
	rt, err := c.UpdateRuntimeEnvironment(context.Background(), "rt123", &RuntimeEnvironment{Name: "Updated Runtime"})
	if err != nil {
		t.Fatalf("UpdateRuntimeEnvironment() error: %v", err)
	}
	if rt.ID != "rt123" {
		t.Errorf("expected ID rt123, got %s", rt.ID)
	}
}
