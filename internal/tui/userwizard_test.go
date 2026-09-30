package tui

import (
	"reflect"
	"strings"
	"testing"

	"github.com/jbrazda/iics-cli/internal/client"
)

func TestUserChanges(t *testing.T) {
	before := &client.User{
		FirstName: "Ann", LastName: "Lee", Email: "ann@x.com", TimeZoneID: "UTC",
		Groups: []client.UserGroupRef{{ID: "g1", UserGroupName: "Dev"}, {ID: "g2", UserGroupName: "Ops"}},
		Roles:  []client.UserRole{{ID: "r1", RoleName: "Designer"}},
	}
	after := cloneUser(before)
	after.LastName = "Smith"
	after.TimeZoneID = ""
	after.ForcePasswordChange = true
	after.Groups = []client.UserGroupRef{{ID: "g2", UserGroupName: "Ops"}, {ID: "g3", UserGroupName: "QA"}}
	after.Roles = append(after.Roles, client.UserRole{ID: "r2", RoleName: "Monitor"})

	got := UserChanges(before, after)
	want := []string{
		"Last name: Lee -> Smith",
		"Time zone: UTC -> -",
		"Force password change: false -> true",
		"+ Group: QA",
		"- Group: Dev",
		"+ Role: Monitor",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("UserChanges =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if n := UserChanges(before, cloneUser(before)); len(n) != 0 {
		t.Errorf("expected no changes, got %v", n)
	}
}

func TestUserSummary(t *testing.T) {
	u := &client.User{
		UserName: "ann.lee@x.com", FirstName: "Ann", LastName: "Lee", Email: "ann@x.com",
		Authentication: "SSO", AliasName: "alee",
		Roles: []client.UserRole{{RoleName: "Designer"}},
	}
	s := strings.Join(UserSummary(u), "\n")
	for _, want := range []string{"User name:      ann.lee@x.com", "Name:           Ann Lee", "SSO alias:      alee", "Groups:         -", "Roles:          Designer"} {
		if !strings.Contains(s, want) {
			t.Errorf("summary missing %q:\n%s", want, s)
		}
	}
	if strings.Contains(s, "Phone") {
		t.Errorf("empty optional fields should be omitted:\n%s", s)
	}
}

func TestSuggestUserName(t *testing.T) {
	if got := suggestUserName(" Ann ", "Lee", "x.com"); got != "ann.lee@x.com" {
		t.Errorf("got %q", got)
	}
	if got := suggestUserName("Ann", "", "x.com"); got != "" {
		t.Errorf("missing last name should give no suggestion, got %q", got)
	}
	if got := suggestUserName("Ann", "Lee", ""); got != "" {
		t.Errorf("missing domain should give no suggestion, got %q", got)
	}
}

func TestValidEmail(t *testing.T) {
	for _, ok := range []string{"a@b.com", " a.b+c@d.org "} {
		if err := validEmail(ok); err != nil {
			t.Errorf("validEmail(%q) = %v", ok, err)
		}
	}
	for _, bad := range []string{"", "   ", "not-an-email", "a@"} {
		if err := validEmail(bad); err == nil {
			t.Errorf("validEmail(%q) should fail", bad)
		}
	}
}

func TestOptionLabel(t *testing.T) {
	if got := optionLabel("Designer", ""); got != "Designer" {
		t.Errorf("got %q", got)
	}
	long := optionLabel("Admin", strings.Repeat("word ", 30))
	if len([]rune(long)) != len("Admin - ")+50 || !strings.HasSuffix(long, "…") {
		t.Errorf("long description should be cut to 50 runes: %q", long)
	}
	if got := optionLabel("X", "a\n  b"); got != "X - a b" {
		t.Errorf("whitespace should be collapsed, got %q", got)
	}
}

func TestMembershipByID(t *testing.T) {
	groups := []client.UserGroup{{ID: "g1", UserGroupName: "Dev"}, {ID: "g2", UserGroupName: "Ops"}}
	got := groupsByID(groups, []string{"g2"})
	if len(got) != 1 || got[0].UserGroupName != "Ops" {
		t.Errorf("groupsByID = %+v", got)
	}
	roles := []client.Role{{ID: "r1", RoleName: "Designer"}}
	if r := rolesByID(roles, nil); r != nil {
		t.Errorf("rolesByID(nil) = %+v", r)
	}
}
