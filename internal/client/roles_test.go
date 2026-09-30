package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestGetRoleByName(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/public/core/v3/roles" {
			t.Errorf("expected /public/core/v3/roles, got %s", r.URL.Path)
		}
		if q := r.URL.Query().Get("q"); q != `roleName=="Business Manager"` {
			t.Errorf("unexpected q: %s", q)
		}
		if e := r.URL.Query().Get("expand"); e != "privileges" {
			t.Errorf("expected expand=privileges, got %q", e)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{
			"id": "7EjAMAHsiOTcg8v29z0Gsl",
			"orgId": "52ZSTB0IDK6dXxaEQLUaQu",
			"roleName": "Business Manager",
			"systemRole": true,
			"status": "Disabled",
			"privileges": [
				{"id": "5Cgp0GcsmRejyxIgV4eXy1", "name": "view.ai.console", "description": "View application integration console", "service": "ApplicationIntegration", "status": "Disabled"}
			]
		}]`))
	})
	c := newTestClient(handler)

	role, err := c.GetRole(context.Background(), RoleGetOptions{Name: "Business Manager", ExpandPrivileges: true})
	if err != nil {
		t.Fatalf("GetRole() error: %v", err)
	}
	if role.ID != "7EjAMAHsiOTcg8v29z0Gsl" || role.RoleName != "Business Manager" || !role.SystemRole {
		t.Errorf("unexpected role: %+v", role)
	}
	if len(role.Privileges) != 1 {
		t.Fatalf("expected 1 privilege, got %d", len(role.Privileges))
	}
	p := role.Privileges[0]
	if p.Name != "view.ai.console" || p.Service != "ApplicationIntegration" || p.Status != "Disabled" {
		t.Errorf("unexpected privilege: %+v", p)
	}
}

func TestGetRoleByID(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if q := r.URL.Query().Get("q"); q != `roleId=="abc123"` {
			t.Errorf("unexpected q: %s", q)
		}
		if r.URL.Query().Has("expand") {
			t.Errorf("expand should not be set")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id": "abc123", "roleName": "Designer"}]`))
	})
	c := newTestClient(handler)

	role, err := c.GetRole(context.Background(), RoleGetOptions{ID: "abc123"})
	if err != nil {
		t.Fatalf("GetRole() error: %v", err)
	}
	if role.RoleName != "Designer" {
		t.Errorf("expected Designer, got %s", role.RoleName)
	}
}

func TestGetRoleNotFound(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	})
	c := newTestClient(handler)

	_, err := c.GetRole(context.Background(), RoleGetOptions{Name: "Missing"})
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected not found error, got %v", err)
	}
}

func TestGetRoleRequiresSelector(t *testing.T) {
	c := newTestClient(http.NotFoundHandler())
	if _, err := c.GetRole(context.Background(), RoleGetOptions{}); err == nil {
		t.Error("expected error when neither ID nor name is set")
	}
	if _, err := c.GetRole(context.Background(), RoleGetOptions{ID: "a", Name: "b"}); err == nil {
		t.Error("expected error when both ID and name are set")
	}
}

func TestCreateRole(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/public/core/v3/roles" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var got map[string]interface{}
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("invalid body: %v", err)
		}
		if got["name"] != "CAIviewer" || got["description"] != "View CAI" {
			t.Errorf("unexpected body: %s", body)
		}
		privs, _ := got["privileges"].([]interface{})
		if len(privs) != 2 || privs[0] != "aQwUdcM8RcQewA1yWphZ4F" {
			t.Errorf("unexpected privileges: %v", got["privileges"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id": "8j2MPlr8ubZgteIOwleSCk", "roleName": "CAIviewer", "systemRole": false,
			"privileges": [{"id": "aQwUdcM8RcQewA1yWphZ4F", "name": "view.ai.assets", "description": "View assets"}]}`))
	})
	c := newTestClient(handler)

	role, err := c.CreateRole(context.Background(), &CreateRoleRequest{
		Name:        "CAIviewer",
		Description: "View CAI",
		Privileges:  []string{"aQwUdcM8RcQewA1yWphZ4F", "0nTOXl8dzEwlSFoM0cO8gI"},
	})
	if err != nil {
		t.Fatalf("CreateRole() error: %v", err)
	}
	if role.ID != "8j2MPlr8ubZgteIOwleSCk" || len(role.Privileges) != 1 {
		t.Errorf("unexpected role: %+v", role)
	}
}

func TestAddRolePrivileges(t *testing.T) {
	tests := []struct {
		name     string
		ref      RoleRef
		call     func(*Client, context.Context, RoleRef, []string) error
		wantPath string
	}{
		{"add by id", RoleRef{ID: "abc123"}, (*Client).AddRolePrivileges, "/public/core/v3/roles/abc123/addPrivileges"},
		{"add by name", RoleRef{Name: "CAI Viewer"}, (*Client).AddRolePrivileges, "/public/core/v3/roles/name/CAI%20Viewer/addPrivileges"},
		{"remove by id", RoleRef{ID: "abc123"}, (*Client).RemoveRolePrivileges, "/public/core/v3/roles/abc123/removePrivileges"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPut {
					t.Errorf("expected PUT, got %s", r.Method)
				}
				if r.URL.EscapedPath() != tt.wantPath {
					t.Errorf("expected path %s, got %s", tt.wantPath, r.URL.EscapedPath())
				}
				body, _ := io.ReadAll(r.Body)
				if string(body) != `{"privileges":["create.data.transfer.task"]}` {
					t.Errorf("unexpected body: %s", body)
				}
				w.WriteHeader(http.StatusOK)
			})
			c := newTestClient(handler)
			if err := tt.call(c, context.Background(), tt.ref, []string{"create.data.transfer.task"}); err != nil {
				t.Fatalf("error: %v", err)
			}
		})
	}
}

func TestAddRolePrivilegesValidation(t *testing.T) {
	c := newTestClient(http.NotFoundHandler())
	if err := c.AddRolePrivileges(context.Background(), RoleRef{}, []string{"x"}); err == nil {
		t.Error("expected error without role reference")
	}
	if err := c.AddRolePrivileges(context.Background(), RoleRef{ID: "a"}, nil); err == nil {
		t.Error("expected error without privileges")
	}
}

func TestResolvePrivileges(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"id": "id1", "name": "view.ai.assets"},
			{"id": "id2", "name": "view.ai.designer"}
		]`))
	})
	c := newTestClient(handler)

	got, err := c.ResolvePrivileges(context.Background(), []string{"view.ai.assets", "id2", "id1"})
	if err != nil {
		t.Fatalf("ResolvePrivileges() error: %v", err)
	}
	if len(got) != 2 || got[0].ID != "id1" || got[1].Name != "view.ai.designer" {
		t.Errorf("unexpected result: %+v", got)
	}

	_, err = c.ResolvePrivileges(context.Background(), []string{"view.ai.assets", "bogus"})
	if err == nil || !strings.Contains(err.Error(), "bogus") {
		t.Errorf("expected unknown privilege error, got %v", err)
	}
}

func TestDeleteRole(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/public/core/v3/roles/abc123" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	c := newTestClient(handler)
	if err := c.DeleteRole(context.Background(), "abc123"); err != nil {
		t.Fatalf("DeleteRole() error: %v", err)
	}
}
