package client

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestListUserGroups(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/public/core/v3/userGroups" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		groups := []UserGroup{
			{
				ID:            "g1",
				UserGroupName: "Admin Group",
				Roles: []UserRole{
					{ID: "r1", RoleName: "Admin"},
				},
				Users: []UserGroupMember{
					{ID: "u1", UserName: "user1@example.com"},
					{ID: "u2", UserName: "user2@example.com"},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(groups)
	})

	c := newTestClient(handler)
	groups, err := c.ListUserGroups(context.Background(), UserGroupListOptions{})
	if err != nil {
		t.Fatalf("ListUserGroups() error: %v", err)
	}
	if len(groups) != 1 {
		t.Errorf("expected 1 group, got %d", len(groups))
	}
	if groups[0].ID != "g1" {
		t.Errorf("expected ID g1, got %s", groups[0].ID)
	}
	if groups[0].UserGroupName != "Admin Group" {
		t.Errorf("expected UserGroupName 'Admin Group', got %s", groups[0].UserGroupName)
	}
	if len(groups[0].Roles) != 1 {
		t.Errorf("expected 1 role, got %d", len(groups[0].Roles))
	}
	if groups[0].Roles[0].RoleName != "Admin" {
		t.Errorf("expected role Admin, got %s", groups[0].Roles[0].RoleName)
	}
	if groups[0].CountMembers != 2 {
		t.Errorf("expected CountMembers 2, got %d", groups[0].CountMembers)
	}
	if groups[0].CountRoles != 1 {
		t.Errorf("expected CountRoles 1, got %d", groups[0].CountRoles)
	}
}

func TestListUserGroupsQuery(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("q") != `userGroupName=="Administrator"` {
			t.Errorf("expected q param, got %s", r.URL.Query().Get("q"))
		}
		groups := []UserGroup{{ID: "g1", UserGroupName: "Administrator"}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(groups)
	})

	c := newTestClient(handler)
	groups, err := c.ListUserGroups(context.Background(), UserGroupListOptions{
		Query: `userGroupName=="Administrator"`,
	})
	if err != nil {
		t.Fatalf("ListUserGroups() error: %v", err)
	}
	if len(groups) != 1 {
		t.Errorf("expected 1 group, got %d", len(groups))
	}
	if groups[0].UserGroupName != "Administrator" {
		t.Errorf("expected UserGroupName 'Administrator', got %s", groups[0].UserGroupName)
	}
}

func TestCreateUserGroupRequestShape(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/public/core/v3/userGroups" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["name"] != "Data Eng" {
			t.Errorf("expected name 'Data Eng', got %v", body["name"])
		}
		roles, _ := body["roles"].([]interface{})
		if len(roles) != 2 || roles[0] != "r1" || roles[1] != "r2" {
			t.Errorf("expected roles [r1 r2], got %v", body["roles"])
		}
		if _, hasUGN := body["userGroupName"]; hasUGN {
			t.Errorf("request must not send userGroupName: %v", body)
		}
		_ = json.NewEncoder(w).Encode(UserGroup{ID: "g9", UserGroupName: "Data Eng"})
	})
	c := newTestClient(handler)
	g := &UserGroup{
		Name:  "Data Eng",
		Roles: []UserRole{{ID: "r1"}, {ID: "r2"}},
	}
	out, err := c.CreateUserGroup(context.Background(), g)
	if err != nil {
		t.Fatalf("CreateUserGroup() error: %v", err)
	}
	if out.ID != "g9" {
		t.Errorf("expected g9, got %s", out.ID)
	}
}

func TestCreateUserGroupRequiresRole(t *testing.T) {
	c := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("should not reach the server")
	}))
	if _, err := c.CreateUserGroup(context.Background(), &UserGroup{Name: "X"}); err == nil {
		t.Fatal("expected an error when no role is provided")
	}
}

func TestGetUserGroupByName(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]UserGroup{
			{ID: "g1", UserGroupName: "Alpha"},
			{ID: "g2", UserGroupName: "Beta"},
		})
	})
	c := newTestClient(handler)
	g, err := c.GetUserGroupByName(context.Background(), "Beta")
	if err != nil {
		t.Fatalf("GetUserGroupByName() error: %v", err)
	}
	if g.ID != "g2" {
		t.Errorf("expected g2, got %s", g.ID)
	}
	if _, err := c.GetUserGroupByName(context.Background(), "Missing"); err == nil {
		t.Error("expected not-found error")
	}
}

func TestGetUserGroupScansList(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/public/core/v3/userGroups" {
			t.Errorf("unexpected path %s (v3 has no get-by-id)", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode([]UserGroup{
			{ID: "g1", UserGroupName: "Alpha"},
			{ID: "g2", UserGroupName: "Beta"},
		})
	})
	c := newTestClient(handler)
	g, err := c.GetUserGroup(context.Background(), "g2")
	if err != nil {
		t.Fatalf("GetUserGroup() error: %v", err)
	}
	if g.UserGroupName != "Beta" {
		t.Errorf("expected Beta, got %s", g.UserGroupName)
	}
}

func TestUpdateUserGroupDiffsRoles(t *testing.T) {
	var addBody, removeBody map[string][]string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/public/core/v3/userGroups" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode([]UserGroup{{
				ID: "g1", UserGroupName: "Team",
				Roles: []UserRole{{ID: "r1", RoleName: "Keep"}, {ID: "r2", RoleName: "Drop"}},
			}})
		case r.URL.Path == "/public/core/v3/userGroups/g1/addRoles":
			_ = json.NewDecoder(r.Body).Decode(&addBody)
		case r.URL.Path == "/public/core/v3/userGroups/g1/removeRoles":
			_ = json.NewDecoder(r.Body).Decode(&removeBody)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	})
	c := newTestClient(handler)
	_, err := c.UpdateUserGroup(context.Background(), "g1", &UserGroup{
		Roles: []UserRole{{RoleName: "Keep"}, {RoleName: "Add"}},
	})
	if err != nil {
		t.Fatalf("UpdateUserGroup() error: %v", err)
	}
	if got := addBody["roles"]; len(got) != 1 || got[0] != "Add" {
		t.Errorf("addRoles = %v, want [Add]", got)
	}
	if got := removeBody["roles"]; len(got) != 1 || got[0] != "Drop" {
		t.Errorf("removeRoles = %v, want [Drop]", got)
	}
}

func TestUpdateUserGroupRejectsRename(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]UserGroup{{ID: "g1", UserGroupName: "Old", Roles: []UserRole{{RoleName: "R"}}}})
	})
	c := newTestClient(handler)
	_, err := c.UpdateUserGroup(context.Background(), "g1", &UserGroup{Name: "New"})
	if err == nil {
		t.Fatal("expected rename to be rejected")
	}
}
