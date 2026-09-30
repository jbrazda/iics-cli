package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

const aclListJSON = `[
	{"id": "a1", "principal": {"type": "GROUP", "name": "Developer"},
	 "permissions": {"read": true, "update": true, "delete": true, "execute": true, "changePermission": false}},
	{"id": "a2", "principal": {"type": "USER", "name": "larry@infa.com"},
	 "permissions": {"read": true, "update": false, "delete": false, "execute": false, "changePermission": false}}
]`

func TestListObjectACLs(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/public/core/v3/objects/obj1/permissions" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(aclListJSON))
	})
	acls, err := newTestClient(handler).ListObjectACLs(context.Background(), "obj1")
	if err != nil {
		t.Fatalf("ListObjectACLs() error: %v", err)
	}
	if len(acls) != 2 || acls[0].Principal.Name != "Developer" || !acls[0].Permissions.Execute || acls[1].Principal.Type != "USER" {
		t.Errorf("unexpected ACLs: %+v", acls)
	}
}

func TestCreateObjectACL(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/public/core/v3/objects/obj1/permissions" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		want := `{"principal":{"type":"USER","name":"larry@infa.com"},"permissions":{"read":true,"update":false,"delete":false,"execute":true,"changePermission":false}}`
		if string(body) != want {
			t.Errorf("body = %s\nwant  %s", body, want)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id": "new1", "principal": {"type": "USER", "name": "larry@infa.com"}, "permissions": {"read": true, "execute": true}}]`))
	})
	acl, err := newTestClient(handler).CreateObjectACL(context.Background(), "obj1", ObjectACL{
		Principal:   ACLPrincipal{Type: PrincipalUser, Name: "larry@infa.com"},
		Permissions: ACLPermissions{Read: true, Execute: true},
	})
	if err != nil {
		t.Fatalf("CreateObjectACL() error: %v", err)
	}
	if acl.ID != "new1" {
		t.Errorf("expected ID new1, got %q", acl.ID)
	}
}

func TestUpdateAndDeleteObjectACL(t *testing.T) {
	var calls []string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		if r.Method == http.MethodPut {
			var body map[string]json.RawMessage
			_ = json.NewDecoder(r.Body).Decode(&body)
			if _, hasID := body["id"]; hasID {
				t.Error("PUT body must not contain the ACL id")
			}
		}
		w.WriteHeader(http.StatusNoContent)
	})
	c := newTestClient(handler)
	ctx := context.Background()
	acl := ObjectACL{ID: "a1", Principal: ACLPrincipal{Type: PrincipalGroup, Name: "Developer"}, Permissions: ACLPermissions{Read: true}}
	if err := c.UpdateObjectACL(ctx, "obj1", "a1", acl); err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteObjectACL(ctx, "obj1", "a1"); err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteAllObjectACLs(ctx, "obj1"); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"PUT /public/core/v3/objects/obj1/permissions/a1",
		"DELETE /public/core/v3/objects/obj1/permissions/a1",
		"DELETE /public/core/v3/objects/obj1/permissions",
	}
	if !reflect.DeepEqual(calls, want) {
		t.Errorf("calls = %v, want %v", calls, want)
	}
}

func TestCheckObjectAccess(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/public/core/v3/objects/obj1/permissions/checkAccess" || r.URL.Query().Get("type") != "DTEMPLATE" {
			t.Errorf("unexpected request %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"permissions":{"create":true,"read":true,"update":true,"delete":true,"execute":false,"changePermission":true}}`))
	})
	a, err := newTestClient(handler).CheckObjectAccess(context.Background(), "obj1", "DTEMPLATE")
	if err != nil {
		t.Fatal(err)
	}
	if !a.Permissions.Create || !a.Permissions.ChangePermission || a.Permissions.Execute {
		t.Errorf("unexpected access: %+v", a.Permissions)
	}
}

func TestDiffACLs(t *testing.T) {
	var current []ObjectACL
	_ = json.Unmarshal([]byte(aclListJSON), &current)

	desired := []ObjectACL{
		{Principal: ACLPrincipal{Type: "GROUP", Name: "developer"}, Permissions: ACLPermissions{Read: true}},              // update (case-insensitive match)
		{Principal: ACLPrincipal{Type: "GROUP", Name: "Everyone"}, Permissions: ACLPermissions{Read: true}},               // create
		{Principal: ACLPrincipal{Type: "GROUP", Name: "Everyone"}, Permissions: ACLPermissions{Read: true, Update: true}}, // duplicate, ignored
	}

	plan := DiffACLs(current, desired, false)
	if len(plan.Create) != 1 || plan.Create[0].Principal.Name != "Everyone" || plan.Create[0].Permissions.Update {
		t.Errorf("create = %+v", plan.Create)
	}
	if len(plan.Update) != 1 || plan.Update[0].ID != "a1" || plan.Update[0].Permissions != (ACLPermissions{Read: true}) || plan.Update[0].Principal.Name != "Developer" {
		t.Errorf("update = %+v", plan.Update)
	}
	if len(plan.Delete) != 0 {
		t.Errorf("without prune nothing is deleted, got %+v", plan.Delete)
	}

	plan = DiffACLs(current, desired, true)
	if len(plan.Delete) != 1 || plan.Delete[0].ID != "a2" {
		t.Errorf("prune should delete a2, got %+v", plan.Delete)
	}
	if !DiffACLs(current, current, true).Empty() {
		t.Error("identical ACLs should give an empty plan")
	}
}

func TestParseACLPermissions(t *testing.T) {
	p, err := ParseACLPermissions([]string{"Read, execute", "PERM"})
	if err != nil || p != (ACLPermissions{Read: true, Execute: true, ChangePermission: true}) {
		t.Errorf("got %+v, %v", p, err)
	}
	if p, _ := ParseACLPermissions([]string{"all"}); p.String() != "read,update,delete,execute,changePermission" {
		t.Errorf("all = %s", p)
	}
	if p, _ := ParseACLPermissions([]string{"all", "none"}); p.String() != "none" {
		t.Errorf("none = %s", p)
	}
	if _, err := ParseACLPermissions([]string{"write"}); err == nil || !strings.Contains(err.Error(), "write") {
		t.Errorf("expected unknown permission error, got %v", err)
	}
}
