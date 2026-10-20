package client

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestUserMembershipRequests(t *testing.T) {
	tests := []struct {
		name     string
		call     func(*Client) error
		wantPath string
		wantBody string
	}{
		{"add roles", func(c *Client) error {
			return c.AddUserRoles(context.Background(), "u1", []string{"Admin", "Business Manager"})
		},
			"/public/core/v3/users/u1/addRoles", `{"roles":["Admin","Business Manager"]}`},
		{"remove roles", func(c *Client) error { return c.RemoveUserRoles(context.Background(), "u1", []string{"Designer"}) },
			"/public/core/v3/users/u1/removeRoles", `{"roles":["Designer"]}`},
		{"add groups", func(c *Client) error { return c.AddUserGroups(context.Background(), "u1", []string{"MDM Admin"}) },
			"/public/core/v3/users/u1/addGroups", `{"groups":["MDM Admin"]}`},
		{"remove groups", func(c *Client) error { return c.RemoveUserGroups(context.Background(), "u1", []string{"MDM Admin"}) },
			"/public/core/v3/users/u1/removeGroups", `{"groups":["MDM Admin"]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPut || r.URL.Path != tt.wantPath {
					t.Errorf("got %s %s, want PUT %s", r.Method, r.URL.Path, tt.wantPath)
				}
				body, _ := io.ReadAll(r.Body)
				if string(body) != tt.wantBody {
					t.Errorf("body = %s, want %s", body, tt.wantBody)
				}
				w.WriteHeader(http.StatusOK)
			})
			if err := tt.call(newTestClient(handler)); err != nil {
				t.Fatalf("error: %v", err)
			}
		})
	}
}

func TestPlanMembership(t *testing.T) {
	available := []MemberName{
		{Name: "Admin"},
		{Name: "Designer"},
		{Name: "Monitor"},
		{Name: "Application Integration Business Manager", Aliases: []string{"Business Manager"}},
	}
	current := []string{"Designer", "Monitor"}

	tests := []struct {
		name    string
		req     MembershipRequest
		want    MembershipPlan
		wantErr string
	}{
		{
			name: "add and remove with case-insensitive names and duplicates",
			req:  MembershipRequest{Add: []string{"admin", "Admin", " designer "}, Remove: []string{"monitor", "business manager"}},
			want: MembershipPlan{Add: []string{"Admin"}, Remove: []string{"Monitor"}, AlreadyAssigned: []string{"Designer"}, NotAssigned: []string{"Application Integration Business Manager"}},
		},
		{
			name: "alias resolves to display name",
			req:  MembershipRequest{Add: []string{"business manager", "Application Integration Business Manager"}},
			want: MembershipPlan{Add: []string{"Application Integration Business Manager"}},
		},
		{
			name: "replace",
			req:  MembershipRequest{Replace: []string{"Admin", "Designer"}, ReplaceSet: true},
			want: MembershipPlan{Add: []string{"Admin"}, Remove: []string{"Monitor"}},
		},
		{
			name: "replace with empty list removes all",
			req:  MembershipRequest{ReplaceSet: true},
			want: MembershipPlan{Remove: []string{"Designer", "Monitor"}},
		},
		{name: "unknown", req: MembershipRequest{Add: []string{"Admin", "Nope", "Zip"}}, wantErr: "unknown role(s): Nope, Zip"},
		{name: "conflict", req: MembershipRequest{Add: []string{"Admin"}, Remove: []string{"ADMIN"}}, wantErr: "in both --add and --remove: Admin"},
		{name: "replace with add", req: MembershipRequest{Add: []string{"Admin"}, ReplaceSet: true}, wantErr: "cannot be combined"},
		{name: "nothing requested", req: MembershipRequest{}, wantErr: "provide --add"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PlanMembership("role", available, current, tt.req)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("plan = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestRoleMemberName(t *testing.T) {
	if got := RoleMemberName("Data Preview", "Data Integration Data Previewer"); got != "Data Integration Data Previewer" {
		t.Errorf("got %q", got)
	}
	if got := RoleMemberName("Designer", ""); got != "Designer" {
		t.Errorf("got %q", got)
	}
}

func TestPlanMembershipInherited(t *testing.T) {
	avail := []MemberName{{Name: "Designer"}, {Name: "Monitor"}, {Name: "Admin"}}
	req := MembershipRequest{Add: []string{"designer", "Monitor"}, Inherited: []string{"Designer", "Admin"}}
	plan, err := PlanMembership("role", avail, nil, req)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(plan.Add, []string{"Monitor"}) || !reflect.DeepEqual(plan.Inherited, []string{"Designer"}) {
		t.Errorf("add=%v inherited=%v", plan.Add, plan.Inherited)
	}

	// A direct role that is also inherited stays assigned under --replace.
	req = MembershipRequest{Replace: []string{"Admin", "Designer"}, ReplaceSet: true, Inherited: []string{"Admin", "Designer"}}
	plan, err = PlanMembership("role", avail, []string{"Admin", "Monitor"}, req)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Add) != 0 || !reflect.DeepEqual(plan.Remove, []string{"Monitor"}) || !reflect.DeepEqual(plan.Inherited, []string{"Designer"}) {
		t.Errorf("add=%v remove=%v inherited=%v", plan.Add, plan.Remove, plan.Inherited)
	}
}

func TestInheritedRoles(t *testing.T) {
	groups := []UserGroup{
		{ID: "g1", Roles: []UserRole{{ID: "r1", RoleName: "Designer"}, {ID: "r2", RoleName: "Monitor"}}},
		{ID: "g2", Roles: []UserRole{{ID: "r2", RoleName: "Monitor"}, {ID: "r3", RoleName: "Admin"}}},
	}
	got := InheritedRoles(groups, []string{"g1"})
	if len(got) != 2 || got["r1"].RoleName != "Designer" || got["r2"].RoleName != "Monitor" {
		t.Errorf("InheritedRoles(g1) = %v", got)
	}
	if got := InheritedRoles(groups, nil); len(got) != 0 {
		t.Errorf("InheritedRoles(nil) = %v", got)
	}
}

func TestCheckKeepsDirectRole(t *testing.T) {
	tests := []struct {
		name                 string
		current, add, remove []string
		wantErr              bool
	}{
		{"remove last", []string{"A"}, nil, []string{"A"}, true},
		{"remove all", []string{"A", "B"}, nil, []string{"A", "B"}, true},
		{"swap", []string{"A"}, []string{"B"}, []string{"A"}, false},
		{"keep one", []string{"A", "B"}, nil, []string{"A"}, false},
		{"no removal", nil, nil, nil, false},
	}
	for _, tt := range tests {
		err := CheckKeepsDirectRole("u@x.com", tt.current, tt.add, tt.remove)
		if (err != nil) != tt.wantErr {
			t.Errorf("%s: err = %v, wantErr %v", tt.name, err, tt.wantErr)
		}
	}
}
