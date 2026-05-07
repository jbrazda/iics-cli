package release

import (
	"strings"
	"testing"
)

func TestAssetCountsByTypeSorted(t *testing.T) {
	assets := []Asset{
		{Type: "PROCESS"},
		{Type: "AI_CONNECTION"},
		{Type: "PROCESS"},
		{Type: "GUIDE"},
	}
	counts := AssetCountsByType(assets)
	if len(counts) != 3 {
		t.Fatalf("len = %d, want 3", len(counts))
	}
	if counts[0].Type != "AI_CONNECTION" || counts[0].Count != 1 {
		t.Fatalf("unexpected first count: %#v", counts[0])
	}
	if counts[1].Type != "GUIDE" || counts[1].Count != 1 {
		t.Fatalf("unexpected second count: %#v", counts[1])
	}
	if counts[2].Type != "PROCESS" || counts[2].Count != 2 {
		t.Fatalf("unexpected third count: %#v", counts[2])
	}
}

func TestFormatAssetTypeCounts(t *testing.T) {
	got := FormatAssetTypeCounts([]AssetTypeCount{
		{Type: "GUIDE", Count: 2},
		{Type: "PROCESS", Count: 5},
	})
	if got != "GUIDE=2, PROCESS=5" {
		t.Fatalf("format = %q, want %q", got, "GUIDE=2, PROCESS=5")
	}
	if FormatAssetTypeCounts(nil) != "none" {
		t.Fatalf("empty format mismatch")
	}
}

func TestFormatDependencyTableIncludesAllRows(t *testing.T) {
	assets := []Asset{
		{
			Location:   "Explore/Z.PROCESS",
			Dependency: "explicit",
			Type:       "PROCESS",
			Path:       "Z",
		},
		{
			Location:   "Explore/A.GUIDE",
			Dependency: "transitive",
			Type:       "GUIDE",
			Path:       "A",
		},
	}
	table := FormatDependencyTable(assets)
	if !strings.Contains(table, "LOCATION") {
		t.Fatalf("missing header: %q", table)
	}
	if !strings.Contains(table, "Explore/Z.PROCESS") {
		t.Fatalf("missing explicit row: %q", table)
	}
	if !strings.Contains(table, "Explore/A.GUIDE") {
		t.Fatalf("missing transitive row: %q", table)
	}
}
