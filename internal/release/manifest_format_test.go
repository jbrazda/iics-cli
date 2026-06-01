package release

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestManifestFormatRoundTripProperties(t *testing.T) {
	policy, err := ResolveTargetPolicy("TST,QA,PROD")
	if err != nil {
		t.Fatalf("ResolveTargetPolicy() error = %v", err)
	}
	in := Manifest{
		SchemaVersion: "v1",
		GeneratedAt:   "2026-05-19T10:00:00Z",
		Source:        "ci-run-123",
		Options: Options{
			Mode:               ModeTagBased,
			Tag:                "sprint-42",
			Targets:            []string{"qa", "tst"},
			IncludeConnectors:  true,
			IncludeConnections: true,
			ExcludeFile:        "conf/excludes.txt",
		},
	}

	data, err := MarshalManifest(in, ManifestFormatProperties)
	if err != nil {
		t.Fatalf("MarshalManifest() error = %v", err)
	}
	if !strings.Contains(string(data), "iics.release.manifest.options.mode=tag-based") {
		t.Fatalf("properties output missing expected prefix key:\n%s", string(data))
	}
	if !strings.Contains(string(data), "iics.release.manifest.options.includeConnections=true") {
		t.Fatalf("properties output missing includeConnections key:\n%s", string(data))
	}
	if strings.Contains(string(data), "iics.release.manifest.options.connectorsOnly") {
		t.Fatalf("properties output should not include connectorsOnly key:\n%s", string(data))
	}

	out, err := ParseManifestWithPolicy(data, ManifestFormatProperties, policy)
	if err != nil {
		t.Fatalf("ParseManifestWithPolicy() error = %v", err)
	}
	if out.SchemaVersion != in.SchemaVersion || out.Source != in.Source {
		t.Fatalf("round-trip manifest mismatch: got=%+v want=%+v", out, in)
	}
	if out.Options.Mode != ModeTagBased || out.Options.Tag != "sprint-42" {
		t.Fatalf("round-trip options mismatch: got=%+v", out.Options)
	}
	if !out.Options.IncludeConnectors || !out.Options.IncludeConnections {
		t.Fatalf("include flags mismatch after round-trip: %#v", out.Options)
	}
	if len(out.Options.Targets) != 2 || out.Options.Targets[0] != "qa" || out.Options.Targets[1] != "tst" {
		t.Fatalf("normalized targets mismatch: %#v", out.Options.Targets)
	}
}

func TestDetectManifestFormat(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		data   string
		expect ManifestFormat
	}{
		{name: "extension yaml", path: "release_manifest.yaml", data: "{}", expect: ManifestFormatYAML},
		{name: "extension json", path: "release_manifest.json", data: "mode: full", expect: ManifestFormatJSON},
		{name: "extension properties", path: "release_manifest.properties", data: "mode: full", expect: ManifestFormatProperties},
		{name: "stdin json", path: "-", data: `{"schemaVersion":"v1"}`, expect: ManifestFormatJSON},
		{name: "stdin properties", path: "-", data: "iics.release.manifest.schemaVersion=v1", expect: ManifestFormatProperties},
		{name: "stdin yaml fallback", path: "-", data: "schemaVersion: v1", expect: ManifestFormatYAML},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectManifestFormat(tt.path, []byte(tt.data))
			if got != tt.expect {
				t.Fatalf("DetectManifestFormat() = %s, want %s", got, tt.expect)
			}
		})
	}
}

func TestWriteManifestFilesWithFormat(t *testing.T) {
	dir := t.TempDir()
	manifest := Manifest{
		SchemaVersion: "v1",
		GeneratedAt:   "2026-05-19T10:00:00Z",
		Options: Options{
			Mode:    ModeFullDeployment,
			Targets: []string{"qa", "tst"},
		},
	}

	manifestPath, mdPath, err := WriteManifestFilesWithFormat(dir, manifest, ManifestFormatJSON)
	if err != nil {
		t.Fatalf("WriteManifestFilesWithFormat() error = %v", err)
	}
	if filepath.Ext(manifestPath) != ".json" {
		t.Fatalf("manifest path ext = %s, want .json", filepath.Ext(manifestPath))
	}
	if filepath.Ext(mdPath) != ".md" {
		t.Fatalf("md path ext = %s, want .md", filepath.Ext(mdPath))
	}
}
