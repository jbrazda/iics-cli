package release

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jbrazda/iics-cli/internal/config"
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

func TestPublishAssetsPublishOrder(t *testing.T) {
	input := []Asset{
		{Location: "Explore/A.PROCESS", Type: "PROCESS"},
		{Location: "Explore/B.AI_CONNECTION", Type: "AI_CONNECTION"},
		{Location: "Explore/C.GUIDE", Type: "GUIDE"},
		{Location: "Explore/D.AI_SERVICE_CONNECTOR", Type: "AI_SERVICE_CONNECTOR"},
		{Location: "Explore/E.TASKFLOW", Type: "TASKFLOW"},
	}
	got := PublishAssets(input)
	if len(got) != 5 {
		t.Fatalf("len = %d, want 5", len(got))
	}
	wantTypes := []string{"AI_SERVICE_CONNECTOR", "AI_CONNECTION", "PROCESS", "GUIDE", "TASKFLOW"}
	for i, want := range wantTypes {
		if got[i].Type != want {
			t.Fatalf("type[%d] = %q, want %q", i, got[i].Type, want)
		}
	}
}

func TestPublishAssetsStableWithinType(t *testing.T) {
	input := []Asset{
		{Location: "Explore/Conn1.AI_CONNECTION", Type: "AI_CONNECTION"},
		{Location: "Explore/Proc.PROCESS", Type: "PROCESS"},
		{Location: "Explore/Conn2.AI_CONNECTION", Type: "AI_CONNECTION"},
	}
	got := PublishAssets(input)
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	if got[0].Location != "Explore/Conn1.AI_CONNECTION" || got[1].Location != "Explore/Conn2.AI_CONNECTION" {
		t.Fatalf("AI_CONNECTION relative order changed: %#v", got)
	}
}

func TestParseTargetProfileMap(t *testing.T) {
	m, err := parseTargetProfileMap("TST=tst-prof,qa=qa-prof")
	if err != nil {
		t.Fatalf("parseTargetProfileMap() error = %v", err)
	}
	if m["TST"] != "tst-prof" || m["QA"] != "qa-prof" {
		t.Fatalf("unexpected map: %#v", m)
	}
}

func TestResolveProfileNameForTargetExplicitMissing(t *testing.T) {
	cfg := &config.Config{
		Profiles: map[string]*config.Profile{
			"tst-prof": {Username: "u", Password: "p", Region: "USE4"},
		},
	}
	_, explicit, err := resolveProfileNameForTarget(cfg, "QA", map[string]string{"QA": "qa-prof"})
	if err == nil {
		t.Fatalf("expected error for missing explicitly mapped profile")
	}
	if !explicit {
		t.Fatalf("expected explicit mapping marker")
	}
}

func TestResolveProfileNameForTargetImplicitCaseInsensitive(t *testing.T) {
	cfg := &config.Config{
		Profiles: map[string]*config.Profile{
			"qa": {Username: "u", Password: "p", Region: "USE4"},
		},
	}
	name, explicit, err := resolveProfileNameForTarget(cfg, "QA", nil)
	if err != nil {
		t.Fatalf("resolveProfileNameForTarget() error = %v", err)
	}
	if explicit {
		t.Fatalf("expected implicit resolution")
	}
	if name != "qa" {
		t.Fatalf("name = %q, want qa", name)
	}
}
