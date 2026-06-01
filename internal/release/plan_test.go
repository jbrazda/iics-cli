package release

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/jbrazda/iics-cli/internal/config"
	"gopkg.in/yaml.v3"
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
		{Location: "Explore/C.AI_SERVICE_CONNECTOR", Type: "AI_SERVICE_CONNECTOR"},
		{Location: "Explore/D.Connection", Type: "Connection"},
	}
	got := ApplyPolicies(assets, false, false, nil)
	if len(got) != 1 || got[0].Type != "PROCESS" {
		t.Fatalf("unexpected filtered assets: %#v", got)
	}
}

func TestApplyPoliciesIncludeFlags(t *testing.T) {
	assets := []Asset{
		{Location: "Explore/A.PROCESS", Type: "PROCESS"},
		{Location: "Explore/B.AI_CONNECTION", Type: "AI_CONNECTION"},
		{Location: "Explore/C.AI_SERVICE_CONNECTOR", Type: "AI_SERVICE_CONNECTOR"},
		{Location: "Explore/D.Connection", Type: "Connection"},
	}

	onlyConnectors := ApplyPolicies(assets, true, false, nil)
	if len(onlyConnectors) != 2 {
		t.Fatalf("onlyConnectors len = %d, want 2", len(onlyConnectors))
	}
	for _, a := range onlyConnectors {
		if a.Type != "PROCESS" && a.Type != "AI_SERVICE_CONNECTOR" {
			t.Fatalf("onlyConnectors contains unexpected type %q", a.Type)
		}
	}

	onlyConnections := ApplyPolicies(assets, false, true, nil)
	if len(onlyConnections) != 3 {
		t.Fatalf("onlyConnections len = %d, want 3", len(onlyConnections))
	}
	for _, a := range onlyConnections {
		if a.Type == "AI_SERVICE_CONNECTOR" {
			t.Fatalf("onlyConnections should not include connector type: %#v", onlyConnections)
		}
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

func TestWriteAssetsJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tag_build.package.json")
	assets := []Asset{
		{Location: "Explore/A.PROCESS", Dependency: "explicit", Type: "PROCESS", Path: "A"},
		{Location: "Explore/B.GUIDE", Dependency: "transitive", Type: "GUIDE", Path: "B"},
	}
	if err := WriteAssetsJSON(path, assets, []string{"location", "dependency", "type", "path"}); err != nil {
		t.Fatalf("WriteAssetsJSON() error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var rows []map[string]string
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows len = %d, want 2", len(rows))
	}
	if rows[0]["location"] != "Explore/A.PROCESS" || rows[1]["dependency"] != "transitive" {
		t.Fatalf("unexpected rows: %#v", rows)
	}
}

func TestWriteAssetsYAMLEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "publish_assets.yaml")
	if err := WriteAssetsYAML(path, nil, []string{"location", "dependency"}); err != nil {
		t.Fatalf("WriteAssetsYAML() error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if strings.TrimSpace(string(data)) != "[]" {
		t.Fatalf("expected empty yaml list, got %q", string(data))
	}
}

func TestWriteAssetsYAMLFieldSelection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "publish_assets.yaml")
	assets := []Asset{
		{Location: "Explore/A.PROCESS", Dependency: "explicit", Type: "PROCESS", Path: "A", ID: "id-1"},
	}
	if err := WriteAssetsYAML(path, assets, []string{"location", "id"}); err != nil {
		t.Fatalf("WriteAssetsYAML() error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var rows []map[string]string
	if err := yaml.Unmarshal(data, &rows); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows len = %d, want 1", len(rows))
	}
	if rows[0]["location"] != "Explore/A.PROCESS" || rows[0]["id"] != "id-1" {
		t.Fatalf("unexpected row: %#v", rows[0])
	}
	if _, ok := rows[0]["type"]; ok {
		t.Fatalf("unexpected type key in selected fields output: %#v", rows[0])
	}
}

func TestEnsureCurrentTargetStatusField(t *testing.T) {
	fields := []string{"location", "type", "path", "dependency"}
	got := EnsureCurrentTargetStatusField(fields, "QA")
	if len(got) != 5 {
		t.Fatalf("len = %d, want 5", len(got))
	}
	if got[4] != "status (qa)" {
		t.Fatalf("last field = %q, want %q", got[4], "status (qa)")
	}

	got2 := EnsureCurrentTargetStatusField(got, "qa")
	if len(got2) != 5 {
		t.Fatalf("len after duplicate ensure = %d, want 5", len(got2))
	}
}

func TestWriteAssetsCSV_DefaultPackageFieldOrderWithTargetStatus(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "full_build.package.csv")
	assets := []Asset{
		{
			Location:   "Explore/A.PROCESS",
			Dependency: "transitive",
			Type:       "PROCESS",
			Path:       "A",
			Status:     "found",
		},
	}
	fields := EnsureCurrentTargetStatusField([]string{"location", "type", "path", "dependency"}, "QA")
	if err := WriteAssetsCSV(path, assets, fields); err != nil {
		t.Fatalf("WriteAssetsCSV() error = %v", err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = f.Close() }()
	r := csv.NewReader(f)
	headers, err := r.Read()
	if err != nil {
		t.Fatalf("Read headers error = %v", err)
	}
	wantHeaders := []string{"LOCATION", "TYPE", "PATH", "DEPENDENCY", "STATUS (QA)"}
	if !reflect.DeepEqual(headers, wantHeaders) {
		t.Fatalf("unexpected headers: %#v", headers)
	}
	row, err := r.Read()
	if err != nil {
		t.Fatalf("Read row error = %v", err)
	}
	wantRow := []string{"Explore/A.PROCESS", "PROCESS", "A", "transitive", "found"}
	if !reflect.DeepEqual(row, wantRow) {
		t.Fatalf("unexpected row: %#v", row)
	}
}

func TestWriteAssetsCSV_DefaultPublishFieldOrder(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "publish_assets.csv")
	assets := []Asset{
		{
			Location:   "Explore/A.PROCESS",
			Dependency: "explicit",
			Type:       "PROCESS",
			Path:       "A",
		},
	}
	fields := []string{"location", "type", "path", "dependency"}
	if err := WriteAssetsCSV(path, assets, fields); err != nil {
		t.Fatalf("WriteAssetsCSV() error = %v", err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = f.Close() }()
	r := csv.NewReader(f)
	headers, err := r.Read()
	if err != nil {
		t.Fatalf("Read headers error = %v", err)
	}
	wantHeaders := []string{"LOCATION", "TYPE", "PATH", "DEPENDENCY"}
	if !reflect.DeepEqual(headers, wantHeaders) {
		t.Fatalf("unexpected headers: %#v", headers)
	}
	row, err := r.Read()
	if err != nil {
		t.Fatalf("Read row error = %v", err)
	}
	wantRow := []string{"Explore/A.PROCESS", "PROCESS", "A", "explicit"}
	if !reflect.DeepEqual(row, wantRow) {
		t.Fatalf("unexpected row: %#v", row)
	}
}
