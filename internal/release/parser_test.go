package release

import "testing"

func TestParseDeploymentOptionsMarkdown(t *testing.T) {
	md := `
## Deployment Options
- [ ] Full Deployment
- [x] Selective - Tag-Based
Tag: sprint_20
- [x] TST
- [ ] QA
- [x] STG
- [x] Connectors and Connections
`
	opts, err := ParseDeploymentOptionsMarkdown(md)
	if err != nil {
		t.Fatalf("ParseDeploymentOptionsMarkdown() error = %v", err)
	}
	if opts.Mode != ModeTagBased {
		t.Fatalf("mode = %q, want %q", opts.Mode, ModeTagBased)
	}
	if opts.Tag != "sprint_20" {
		t.Fatalf("tag = %q, want sprint_20", opts.Tag)
	}
	if !opts.IncludeConnectors {
		t.Fatalf("IncludeConnectors should be true")
	}
}
