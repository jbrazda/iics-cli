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
