package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLogin(t *testing.T) {
	loginResp := LoginResponse{
		Products: []Product{
			{Name: "Integration Cloud", BaseAPIURL: "https://usw3.dm-us.informaticacloud.com/saas"},
		},
		UserInfo: UserInfo{
			SessionID: "test-session-id",
			ID:        "user123",
			Name:      "testuser",
			OrgID:     "org123",
			OrgName:   "Test Org",
			Status:    "Active",
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decoding login request: %v", err)
		}
		if req.Username != "testuser" {
			t.Errorf("expected username testuser, got %s", req.Username)
		}
		if req.Password != "testpass" {
			t.Errorf("expected password testpass, got %s", req.Password)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(loginResp)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "testuser", "testpass")
	resp, err := c.Login(context.Background())
	if err != nil {
		t.Fatalf("Login() error: %v", err)
	}

	if resp.UserInfo.SessionID != "test-session-id" {
		t.Errorf("expected session ID test-session-id, got %s", resp.UserInfo.SessionID)
	}
	if resp.UserInfo.OrgName != "Test Org" {
		t.Errorf("expected org name Test Org, got %s", resp.UserInfo.OrgName)
	}
	if c.SessionID() != "test-session-id" {
		t.Errorf("client session ID not set, got %s", c.SessionID())
	}
	if c.BaseAPIURL() != "https://usw3.dm-us.informaticacloud.com/saas" {
		t.Errorf("client base URL not set, got %s", c.BaseAPIURL())
	}
}

func TestLoginFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-Id", "login-req-123")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{
			"code":    "AUTH_FAILED",
			"message": "Invalid credentials",
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "baduser", "badpass")
	_, err := c.Login(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid credentials")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", apiErr.StatusCode)
	}
	if apiErr.Code != "AUTH_FAILED" {
		t.Errorf("expected code AUTH_FAILED, got %s", apiErr.Code)
	}
	if apiErr.ResponseHeaders == nil {
		t.Fatal("expected ResponseHeaders to be set")
	}
	if apiErr.ResponseHeaders.Get("X-Request-Id") != "login-req-123" {
		t.Errorf("expected X-Request-Id header, got %s", apiErr.ResponseHeaders.Get("X-Request-Id"))
	}
	if len(apiErr.ResponseBody) == 0 {
		t.Error("expected ResponseBody to be set")
	}
}
