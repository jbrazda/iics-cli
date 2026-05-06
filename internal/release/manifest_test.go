package release

import "testing"

func TestValidateOptions(t *testing.T) {
	opts := Options{
		Mode:    ModeTagBased,
		Tag:     "sprint-10",
		Targets: []string{"qa", "tst", "qa"},
	}
	if err := ValidateOptions(&opts); err != nil {
		t.Fatalf("ValidateOptions() error = %v", err)
	}
	if len(opts.Targets) != 2 || opts.Targets[0] != "QA" || opts.Targets[1] != "TST" {
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
	if !p.Allowed["TST"] || !p.Allowed["QA"] || !p.Allowed["STG"] || !p.Allowed["PROD"] {
		t.Fatalf("default policy missing expected targets: %#v", p.Ordered)
	}
}

func TestResolveTargetPolicyPrecedence(t *testing.T) {
	t.Setenv("IICS_VALID_DEPLOY_TARGETS", "dev,qa")
	p, err := ResolveTargetPolicy("sit,uat")
	if err != nil {
		t.Fatalf("ResolveTargetPolicy() error = %v", err)
	}
	if len(p.Ordered) != 2 || p.Ordered[0] != "SIT" || p.Ordered[1] != "UAT" {
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
	if len(opts.Targets) != 1 || opts.Targets[0] != "DEV" {
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
