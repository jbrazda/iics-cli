package release

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadExcludePatterns(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "exclude.txt")
	if err := os.WriteFile(f, []byte("# comment\n^Explore/SYS/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	patterns, err := LoadExcludePatterns(f)
	if err != nil {
		t.Fatalf("LoadExcludePatterns() error = %v", err)
	}
	if len(patterns) != 1 {
		t.Fatalf("patterns = %d, want 1", len(patterns))
	}
}

func TestApplyPolicies(t *testing.T) {
	assets := []Asset{
		{Location: "Explore/A.PROCESS", Type: "PROCESS"},
		{Location: "Explore/B.AI_CONNECTION", Type: "AI_CONNECTION"},
	}
	got := ApplyPolicies(assets, false, false, nil)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
}

func TestApplyMissingTransitivePolicy(t *testing.T) {
	assets := []Asset{
		{Location: "Explore/A.PROCESS", Type: "PROCESS", Dependency: "explicit"},
		{Location: "Explore/B.PROCESS", Type: "PROCESS", Dependency: "transitive"},
		{Location: "Explore/C.GUIDE", Type: "GUIDE", Dependency: "transitive"},
	}
	missing := map[string]bool{
		"Explore/B.PROCESS": true,
		"Explore/C.GUIDE":   false,
	}
	got := ApplyMissingTransitivePolicy(assets, missing)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Location != "Explore/A.PROCESS" || got[1].Location != "Explore/B.PROCESS" {
		t.Fatalf("unexpected assets: %#v", got)
	}
}
