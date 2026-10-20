package tui

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/config"
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

func TestWizardSuggestions(t *testing.T) {
	in := UserWizardInput{ProfileName: "dev", Patterns: config.NewUserPatterns(nil, &config.Profile{Username: "admin@acme.com"})}
	u := &client.User{FirstName: " Ann ", LastName: "Lee"}
	if got := in.suggestions(in.Patterns.UserNamePattern, u); len(got) != 1 || got[0] != "ann.lee.dev@acme.com" {
		t.Errorf("user name suggestion = %v", got)
	}
	if got, _ := in.suggest(in.Patterns.EmailPattern, u); got != "ann.lee@acme.com" {
		t.Errorf("email suggestion = %q", got)
	}
	if hint := in.suggestionHint(in.Patterns.EmailPattern, u); !strings.Contains(hint, "→") {
		t.Errorf("hint should mention the complete key: %q", hint)
	}

	missing := &client.User{FirstName: "Ann"}
	if got := in.suggestions(in.Patterns.EmailPattern, missing); got != nil {
		t.Errorf("no suggestion expected without a last name, got %v", got)
	}

	bad := UserWizardInput{ProfileName: "dev", Patterns: config.NewUserConfig{UserNamePattern: "{nick}@{domain}", Domain: "acme.com"}}
	if hint := bad.suggestionHint(bad.Patterns.UserNamePattern, u); !strings.Contains(hint, "unknown placeholder {nick}") {
		t.Errorf("unknown placeholder should be reported, got %q", hint)
	}
}

func TestCheckNewUserName(t *testing.T) {
	calls := 0
	in := UserWizardInput{existsCache: map[string]string{}, UserExists: func(n string) (string, error) {
		calls++
		switch n {
		case "taken@acme.com", "Taken@acme.com":
			return "u7", nil
		case "broken@acme.com":
			return "", errors.New("network down")
		}
		return "", nil
	}}
	if err := in.checkNewUserName("taken@acme.com"); err == nil || !strings.Contains(err.Error(), `user "taken@acme.com" already exists (ID u7)`) {
		t.Errorf("expected exists error, got %v", err)
	}
	_ = in.checkNewUserName("Taken@acme.com") // cached (case-insensitive)
	if calls != 1 {
		t.Errorf("lookups should be cached per name, got %d calls", calls)
	}
	if err := in.checkNewUserName("free@acme.com"); err != nil {
		t.Errorf("free name should pass, got %v", err)
	}
	if err := in.checkNewUserName("broken@acme.com"); err != nil {
		t.Errorf("lookup errors must not block, got %v", err)
	}
	upd := in
	upd.Update = true
	if err := upd.checkNewUserName("taken@acme.com"); err != nil {
		t.Errorf("no check on update, got %v", err)
	}
}

func TestSuggestionKeyMap(t *testing.T) {
	keys := suggestionKeyMap().Input.AcceptSuggestion.Keys()
	if len(keys) != 2 || keys[0] != "right" || keys[1] != "ctrl+e" {
		t.Errorf("accept keys = %v", keys)
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

func TestRoleOptionsHideInherited(t *testing.T) {
	roles := []client.Role{{ID: "r1", RoleName: "Designer"}, {ID: "r2", RoleName: "Monitor"}, {ID: "r3", RoleName: "Admin"}}
	inherited := map[string]client.UserRole{"r1": {ID: "r1", RoleName: "Designer"}, "r2": {ID: "r2", RoleName: "Monitor"}}
	direct := map[string]bool{"r2": true}

	var got []string
	for _, o := range roleOptions(roles, inherited, direct) {
		got = append(got, o.Value)
	}
	if !reflect.DeepEqual(got, []string{"r2", "r3"}) {
		t.Errorf("roleOptions = %v, want [r2 r3]", got)
	}
	if ids := dropInherited([]string{"r1", "r2", "r3"}, inherited, direct); !reflect.DeepEqual(ids, []string{"r2", "r3"}) {
		t.Errorf("dropInherited = %v, want [r2 r3]", ids)
	}
	if d := inheritedDescription(inherited); !strings.Contains(d, "not listed): Designer, Monitor") {
		t.Errorf("description = %q", d)
	}
}
