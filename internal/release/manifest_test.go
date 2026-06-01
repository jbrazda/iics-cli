package release

import (
	"strings"
	"testing"
)

func TestValidateOptions(t *testing.T) {
	opts := Options{
		Mode:    ModeTagBased,
		Tag:     "sprint-10",
		Targets: []string{"qa", "tst", "qa"},
	}
	if err := ValidateOptions(&opts); err != nil {
		t.Fatalf("ValidateOptions() error = %v", err)
	}
	if len(opts.Targets) != 2 || opts.Targets[0] != "qa" || opts.Targets[1] != "tst" {
		t.Fatalf("normalized targets unexpected: %#v", opts.Targets)
	}
}

func TestValidateOptionsTagRequired(t *testing.T) {
	opts := Options{
		Mode:    ModeTagBased,
		Targets: []string{"TST"},
	}
	if err := ValidateOptions(&opts); err == nil {
		t.Fatalf("expected error for missing tag in tag-based mode")
	}
}

func TestResolveTargetPolicyDefaults(t *testing.T) {
	t.Setenv("IICS_VALID_DEPLOY_TARGETS", "")
	p, err := ResolveTargetPolicy("")
	if err != nil {
		t.Fatalf("ResolveTargetPolicy() error = %v", err)
	}
	if !p.Allowed["tst"] || !p.Allowed["qa"] || !p.Allowed["stg"] || !p.Allowed["prod"] {
		t.Fatalf("default policy missing expected targets: %#v", p.Ordered)
	}
}

func TestResolveTargetPolicyPrecedence(t *testing.T) {
	t.Setenv("IICS_VALID_DEPLOY_TARGETS", "dev,qa")
	p, err := ResolveTargetPolicy("sit,uat")
	if err != nil {
		t.Fatalf("ResolveTargetPolicy() error = %v", err)
	}
	if len(p.Ordered) != 2 || p.Ordered[0] != "sit" || p.Ordered[1] != "uat" {
		t.Fatalf("policy precedence unexpected: %#v", p.Ordered)
	}
}

func TestValidateOptionsWithPolicy(t *testing.T) {
	policy, err := ResolveTargetPolicy("DEV,QA")
	if err != nil {
		t.Fatalf("ResolveTargetPolicy() error = %v", err)
	}
	opts := Options{
		Mode:    ModeTagBased,
		Tag:     "sprint-1",
		Targets: []string{"dev"},
	}
	if err := ValidateOptionsWithPolicy(&opts, policy); err != nil {
		t.Fatalf("ValidateOptionsWithPolicy() error = %v", err)
	}
	if len(opts.Targets) != 1 || opts.Targets[0] != "dev" {
		t.Fatalf("unexpected normalized targets: %#v", opts.Targets)
	}
}

func TestValidateOptionsWithPolicyRejectsUnknownTarget(t *testing.T) {
	policy, err := ResolveTargetPolicy("DEV,QA")
	if err != nil {
		t.Fatalf("ResolveTargetPolicy() error = %v", err)
	}
	opts := Options{
		Mode:    ModeTagBased,
		Tag:     "sprint-2",
		Targets: []string{"prod"},
	}
	if err := ValidateOptionsWithPolicy(&opts, policy); err == nil {
		t.Fatalf("expected validation error for unknown target")
	}
}

func TestRenderManifestMarkdownOmitsConnectorsOnly(t *testing.T) {
	md := RenderManifestMarkdown(Manifest{
		SchemaVersion: "v1",
		GeneratedAt:   "2026-05-19T10:00:00Z",
		Options: Options{
			Mode:               ModeTagBased,
			Tag:                "sprint-1",
			Targets:            []string{"QA", "TST"},
			IncludeConnectors:  true,
			IncludeConnections: true,
		},
	})
	if strings.Contains(md, "Connectors Only:") {
		t.Fatalf("markdown should not include Connectors Only:\n%s", md)
	}
}
