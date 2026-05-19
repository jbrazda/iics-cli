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
- [x] Connectors
- [x] Connections
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
	if !opts.IncludeConnections {
		t.Fatalf("IncludeConnections should be true")
	}
}

func TestParseDeploymentOptionsMarkdownTagWithBackticksAndComment(t *testing.T) {
	md := `
## Deployment Options
- [ ] Full Deployment
- [x] Selective - Tag-Based
Tag: ` + "`" + `sample_tag` + "`" + ` <!-- enter single-word tag here -->
- [x] TST
- [x] QA
`
	opts, err := ParseDeploymentOptionsMarkdown(md)
	if err != nil {
		t.Fatalf("ParseDeploymentOptionsMarkdown() error = %v", err)
	}
	if opts.Mode != ModeTagBased {
		t.Fatalf("mode = %q, want %q", opts.Mode, ModeTagBased)
	}
	if opts.Tag != "sample_tag" {
		t.Fatalf("tag = %q, want sample_tag", opts.Tag)
	}
}

func TestParseDeploymentOptionsMarkdownSeparateConnectorCheckboxes(t *testing.T) {
	md := `
## Deployment Options
- [ ] Full Deployment
- [x] Selective - Tag-Based
Tag: sprint_21
- [x] TST
- [x] Connectors
- [ ] Connections
`
	opts, err := ParseDeploymentOptionsMarkdown(md)
	if err != nil {
		t.Fatalf("ParseDeploymentOptionsMarkdown() error = %v", err)
	}
	if !opts.IncludeConnectors {
		t.Fatalf("IncludeConnectors should be true")
	}
	if opts.IncludeConnections {
		t.Fatalf("IncludeConnections should be false")
	}
}
