package packaging

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestExportedObject_ObjectRefsRoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		metadata string
		want     []string
	}{
		{name: "empty metadata", metadata: "", want: nil},
		{name: "null metadata", metadata: "null", want: nil},
		{name: "no objectRefs field", metadata: `{"foo":"bar"}`, want: nil},
		{name: "objectRefs present", metadata: `{"objectRefs":["a","b"],"other":1}`, want: []string{"a", "b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := ExportedObject{ObjectGUID: "x", Metadata: json.RawMessage(tt.metadata)}
			got := o.ObjectRefs()
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ObjectRefs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExportedObject_SetObjectRefsPreservesOtherFields(t *testing.T) {
	o := ExportedObject{ObjectGUID: "x", Metadata: json.RawMessage(`{"foo":"bar","objectRefs":["old"]}`)}
	if err := o.SetObjectRefs([]string{"new1", "new2"}); err != nil {
		t.Fatalf("SetObjectRefs() error: %v", err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(o.Metadata, &m); err != nil {
		t.Fatalf("unmarshal updated metadata: %v", err)
	}
	if string(m["foo"]) != `"bar"` {
		t.Fatalf("expected foo field preserved, got %s", m["foo"])
	}
	var refs []string
	if err := json.Unmarshal(m["objectRefs"], &refs); err != nil {
		t.Fatalf("unmarshal objectRefs: %v", err)
	}
	if !reflect.DeepEqual(refs, []string{"new1", "new2"}) {
		t.Fatalf("objectRefs = %v, want [new1 new2]", refs)
	}
}

func TestExportedObject_SetObjectRefsInvalidMetadataErrors(t *testing.T) {
	o := ExportedObject{ObjectGUID: "x", Metadata: json.RawMessage(`[1,2,3]`)}
	if err := o.SetObjectRefs([]string{"a"}); err == nil {
		t.Fatal("expected error for metadata that is not a JSON object")
	}
}

func TestExportedObjectsToManifestLog(t *testing.T) {
	objects := []ExportedObject{
		{ObjectGUID: "g1", ObjectType: "PROCESS", Path: "/Explore/App"},
		{ObjectGUID: "g2", ObjectType: "Folder", Path: "/Explore"},
	}
	rows := ExportedObjectsToManifestLog(objects)
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0].ID != "g1" || rows[0].Type != "PROCESS" || rows[0].Path != "/Explore/App" {
		t.Fatalf("unexpected row 0: %+v", rows[0])
	}
	if rows[1].ID != "g2" || rows[1].Type != "Folder" || rows[1].Path != "/Explore" {
		t.Fatalf("unexpected row 1: %+v", rows[1])
	}
}
