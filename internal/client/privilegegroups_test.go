package client

import (
	"reflect"
	"testing"
)

func TestGroupPrivileges(t *testing.T) {
	all := []Privilege{
		{ID: "1", Name: "create.data.transfer.task", Service: "DI"},
		{ID: "2", Name: "view.data.transfer.task", Service: "DI"},
		{ID: "3", Name: "delete.mapping", Service: "DI"},
		{ID: "4", Name: "edc.iics.discovery", Service: "DI"},
		{ID: "5", Name: "feature.mcp.SuperAdmin", Service: "MetadataControlPlane"},
		{ID: "6", Name: "changeperm.test_case", Service: ""},
		{ID: "7", Name: "view.", Service: "DI"},
		{ID: "8", Name: "PROFILE.viewResults", Service: "Profile"},
	}

	got := GroupPrivileges(all)

	names := make([]string, len(got))
	for i, s := range got {
		names[i] = s.Name
	}
	wantNames := []string{NoServiceGroup, "DI", "MetadataControlPlane", "Profile"}
	if !reflect.DeepEqual(names, wantNames) {
		t.Fatalf("services = %v, want %v", names, wantNames)
	}

	di := got[1]
	if len(di.Objects) != 2 || di.Objects[0].Key != "data.transfer.task" || di.Objects[1].Key != "mapping" {
		t.Fatalf("DI objects = %+v", di.Objects)
	}
	dtt := di.Objects[0].Actions
	if dtt[ActionCreate].ID != "1" || dtt[ActionView].ID != "2" || len(dtt) != 2 {
		t.Errorf("data.transfer.task actions = %+v", dtt)
	}
	if len(di.Other) != 2 || di.Other[0].Name != "edc.iics.discovery" || di.Other[1].Name != "view." {
		t.Errorf("DI other = %+v", di.Other)
	}
	if di.Count() != 5 {
		t.Errorf("DI count = %d, want 5", di.Count())
	}

	if got[0].Objects[0].Key != "test_case" || got[0].Objects[0].Actions[ActionChangePerm].ID != "6" {
		t.Errorf("no-service group = %+v", got[0])
	}
	if len(got[2].Other) != 1 || len(got[3].Other) != 1 {
		t.Errorf("unrecognized names should go to Other: %+v %+v", got[2], got[3])
	}
}

func TestDiffPrivileges(t *testing.T) {
	add, remove := DiffPrivileges(
		[]string{"view.a", "create.a", "delete.a"},
		[]string{"view.a", "update.a", "execute.a"},
	)
	if !reflect.DeepEqual(add, []string{"execute.a", "update.a"}) {
		t.Errorf("add = %v", add)
	}
	if !reflect.DeepEqual(remove, []string{"create.a", "delete.a"}) {
		t.Errorf("remove = %v", remove)
	}

	add, remove = DiffPrivileges([]string{"x"}, []string{"x"})
	if add != nil || remove != nil {
		t.Errorf("expected no changes, got add=%v remove=%v", add, remove)
	}
}
