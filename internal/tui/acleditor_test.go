package tui

import (
	"reflect"
	"strings"
	"testing"

	"github.com/jbrazda/iics-cli/internal/client"
)

func TestDesiredACLsAndPlan(t *testing.T) {
	dev := client.ACLPrincipal{Type: client.PrincipalGroup, Name: "Developer"}
	ops := client.ACLPrincipal{Type: client.PrincipalGroup, Name: "Ops"}
	ann := client.ACLPrincipal{Type: client.PrincipalUser, Name: "ann@x.com"}
	current := []client.ObjectACL{
		{ID: "a1", Principal: dev, Permissions: client.ACLPermissions{Read: true}},
		{ID: "a2", Principal: ops, Permissions: client.ACLPermissions{Read: true, Execute: true}},
	}
	rows := []aclRow{{principal: dev}, {principal: ops}, {principal: ann, added: true}}

	selected := map[string]bool{}
	permsToSet(selected, dev, client.ACLPermissions{Read: true, Update: true}) // update
	permsToSet(selected, ops, client.ACLPermissions{})                         // cleared -> delete
	permsToSet(selected, ann, client.ACLPermissions{Read: true})               // new -> create

	desired := desiredACLs(rows, selected)
	if len(desired) != 2 {
		t.Fatalf("principals without permissions must be dropped, got %+v", desired)
	}
	got := ACLPlanLines(client.DiffACLs(current, desired, true))
	want := []string{
		"+ user ann@x.com: read",
		"~ group Developer: read,update",
		"- group Ops",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("plan lines = %v, want %v", got, want)
	}
}

func TestACLGridRowsOrder(t *testing.T) {
	rows := aclGridRows([]aclRow{
		{principal: client.ACLPrincipal{Type: client.PrincipalUser, Name: "zed@x.com"}},
		{principal: client.ACLPrincipal{Type: client.PrincipalGroup, Name: "ops"}, added: true},
		{principal: client.ACLPrincipal{Type: client.PrincipalGroup, Name: "Admins"}},
	})
	var labels []string
	for _, r := range rows {
		labels = append(labels, r.Label+" "+r.Tag)
	}
	want := []string{"Admins group", "ops group (new)", "zed@x.com user"}
	if !reflect.DeepEqual(labels, want) {
		t.Errorf("rows = %v, want %v", labels, want)
	}
	if len(rows[0].Cells) != 5 {
		t.Errorf("each row needs 5 permission cells, got %d", len(rows[0].Cells))
	}
}

func TestSelfLockoutWarning(t *testing.T) {
	me := client.ACLPrincipal{Type: client.PrincipalUser, Name: "Admin@X.com"}
	current := []client.ObjectACL{{Principal: me, Permissions: client.ACLPermissions{Read: true, ChangePermission: true}}}
	kept := []client.ObjectACL{{Principal: me, Permissions: client.ACLPermissions{ChangePermission: true}}}
	dropped := []client.ObjectACL{{Principal: me, Permissions: client.ACLPermissions{Read: true}}}

	if w := selfLockoutWarning("admin@x.com", current, kept); w != "" {
		t.Errorf("no warning expected, got %q", w)
	}
	if w := selfLockoutWarning("admin@x.com", current, dropped); !strings.Contains(w, "your own user") {
		t.Errorf("expected warning, got %q", w)
	}
	if w := selfLockoutWarning("admin@x.com", current, nil); w == "" {
		t.Error("deleting the own ACL should warn")
	}
	if w := selfLockoutWarning("", current, nil); w != "" {
		t.Error("unknown current user should not warn")
	}
}
