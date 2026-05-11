package dependencies

import "testing"

func TestParseBuildManifestCSV_ExcludeTransitiveFound(t *testing.T) {
	input := `LOCATION,DEPENDENCY,STATUS (qa),ID,TYPE,PATH
Explore/A/one.MTT,transitive,found,id1,MTT,A/one
Explore/A/two.MTT,transitive,missing,id2,MTT,A/two
Explore/A/three.MTT,explicit,found,id3,MTT,A/three
`
	entries, stats, err := ParseBuildManifestCSV([]byte(input), BuildManifestParseOptions{
		ExcludeFoundTransitive: true,
		TargetStatus:           "qa",
	})
	if err != nil {
		t.Fatalf("ParseBuildManifestCSV() error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if stats.ExcludedTransitiveFound != 1 {
		t.Fatalf("expected 1 excluded transitive-found row, got %d", stats.ExcludedTransitiveFound)
	}
	if stats.SelectedStatusColumnName != "STATUS (qa)" {
		t.Fatalf("expected selected status column STATUS (qa), got %q", stats.SelectedStatusColumnName)
	}
}

func TestParseBuildManifestCSV_RequiresStatusTargetWhenAmbiguous(t *testing.T) {
	input := `LOCATION,DEPENDENCY,STATUS (qa),STATUS (stg)
Explore/A/one.MTT,transitive,found,missing
`
	_, _, err := ParseBuildManifestCSV([]byte(input), BuildManifestParseOptions{
		ExcludeFoundTransitive: true,
	})
	if err == nil {
		t.Fatalf("expected ambiguous status columns error")
	}
}

func TestParseBuildManifestCSV_CsvWithoutFilterColumnsStillParses(t *testing.T) {
	input := `ID,PATH,TYPE
abc123,Project/Task1,MTT
`
	entries, stats, err := ParseBuildManifestCSV([]byte(input), BuildManifestParseOptions{})
	if err != nil {
		t.Fatalf("ParseBuildManifestCSV() error: %v", err)
	}
	if len(entries) != 1 || entries[0].ID != "abc123" {
		t.Fatalf("unexpected parsed entries: %+v", entries)
	}
	if stats.TotalRows != 1 || stats.IncludedRows != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}
