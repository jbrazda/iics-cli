package dependencies

import "testing"

func TestParseSeedEntriesFromInputJSON(t *testing.T) {
	in := []byte(`[{"id":"a1","path":"X","type":"PROCESS"},{"location":"Explore/ZZ/Processes.Folder"}]`)
	got, err := ParseSeedEntriesFromInput(in)
	if err != nil {
		t.Fatalf("ParseSeedEntriesFromInput() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].ID != "a1" {
		t.Fatalf("entry 0 id = %q, want a1", got[0].ID)
	}
	if got[1].Path != "ZZ/Processes" || got[1].Type != "Folder" {
		t.Fatalf("entry 1 = %#v", got[1])
	}
}

func TestParseSeedEntriesFromInputCSVPathType(t *testing.T) {
	in := []byte("LOCATION,PATH,TYPE\nExplore/ZZ/Processes.Folder,ZZ/Processes,Folder\n")
	got, err := ParseSeedEntriesFromInput(in)
	if err != nil {
		t.Fatalf("ParseSeedEntriesFromInput() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].Path != "ZZ/Processes" || got[0].Type != "Folder" {
		t.Fatalf("entry = %#v", got[0])
	}
}

func TestParseSeedIDsFromInputStillWorksForIDRows(t *testing.T) {
	in := []byte("ID,PATH,TYPE\nx1,A,PROCESS\nx2,B,GUIDE\n")
	got, err := ParseSeedIDsFromInput(in)
	if err != nil {
		t.Fatalf("ParseSeedIDsFromInput() error = %v", err)
	}
	if len(got) != 2 || got[0] != "x1" || got[1] != "x2" {
		t.Fatalf("unexpected IDs: %#v", got)
	}
}

func TestParseSeedEntriesFromInputInvalid(t *testing.T) {
	if _, err := ParseSeedEntriesFromInput([]byte("not-valid")); err == nil {
		t.Fatalf("expected error for invalid input")
	}
}
