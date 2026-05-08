package dependencies

import "testing"

func TestParseSeedIDsFromInputJSON(t *testing.T) {
	in := []byte(`[{"id":"a1","path":"X","type":"PROCESS"},{"id":"a2"}]`)
	got, err := ParseSeedIDsFromInput(in)
	if err != nil {
		t.Fatalf("ParseSeedIDsFromInput() error = %v", err)
	}
	if len(got) != 2 || got[0] != "a1" || got[1] != "a2" {
		t.Fatalf("unexpected IDs: %#v", got)
	}
}

func TestParseSeedIDsFromInputCSV(t *testing.T) {
	in := []byte("ID,PATH,TYPE\nx1,A,PROCESS\nx2,B,GUIDE\n")
	got, err := ParseSeedIDsFromInput(in)
	if err != nil {
		t.Fatalf("ParseSeedIDsFromInput() error = %v", err)
	}
	if len(got) != 2 || got[0] != "x1" || got[1] != "x2" {
		t.Fatalf("unexpected IDs: %#v", got)
	}
}

func TestParseSeedIDsFromInputYAML(t *testing.T) {
	in := []byte("- id: y1\n  path: A\n  type: PROCESS\n- id: y2\n")
	got, err := ParseSeedIDsFromInput(in)
	if err != nil {
		t.Fatalf("ParseSeedIDsFromInput() error = %v", err)
	}
	if len(got) != 2 || got[0] != "y1" || got[1] != "y2" {
		t.Fatalf("unexpected IDs: %#v", got)
	}
}

func TestParseSeedIDsFromInputInvalid(t *testing.T) {
	if _, err := ParseSeedIDsFromInput([]byte("not-valid")); err == nil {
		t.Fatalf("expected error for invalid input")
	}
}
