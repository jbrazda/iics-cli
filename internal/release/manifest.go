package release

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type DeployMode string

const (
	ModeFullDeployment DeployMode = "full"
	ModeTagBased       DeployMode = "tag-based"
)

const (
	validDeployTargetsEnv = "IICS_VALID_DEPLOY_TARGETS"
	defaultValidTargets   = "TST,QA,STG,PROD"
)

type TargetPolicy struct {
	Allowed map[string]bool
	Ordered []string
}

type Options struct {
	Mode              DeployMode `yaml:"mode" json:"mode"`
	Tag               string     `yaml:"tag,omitempty" json:"tag,omitempty"`
	Targets           []string   `yaml:"targets" json:"targets"`
	IncludeConnectors bool       `yaml:"includeConnectors" json:"includeConnectors"`
	ConnectorsOnly    bool       `yaml:"connectorsOnly,omitempty" json:"connectorsOnly,omitempty"`
	ExcludeFile       string     `yaml:"excludeFile,omitempty" json:"excludeFile,omitempty"`
}

type Manifest struct {
	SchemaVersion string  `yaml:"schemaVersion" json:"schemaVersion"`
	GeneratedAt   string  `yaml:"generatedAt" json:"generatedAt"`
	Source        string  `yaml:"source,omitempty" json:"source,omitempty"`
	Options       Options `yaml:"options" json:"options"`
}

func DefaultOptions() Options {
	return Options{
		Mode:    ModeFullDeployment,
		Targets: []string{"TST", "QA"},
	}
}

func NewManifest(opts Options, source string) (Manifest, error) {
	policy, err := ResolveTargetPolicy("")
	if err != nil {
		return Manifest{}, err
	}
	return NewManifestWithPolicy(opts, source, policy)
}

func NewManifestWithPolicy(opts Options, source string, policy TargetPolicy) (Manifest, error) {
	if err := ValidateOptionsWithPolicy(&opts, policy); err != nil {
		return Manifest{}, err
	}
	return Manifest{
		SchemaVersion: "v1",
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Source:        source,
		Options:       opts,
	}, nil
}

func ValidateManifest(m *Manifest) error {
	policy, err := ResolveTargetPolicy("")
	if err != nil {
		return err
	}
	return ValidateManifestWithPolicy(m, policy)
}

func ValidateManifestWithPolicy(m *Manifest, policy TargetPolicy) error {
	if m.SchemaVersion == "" {
		return fmt.Errorf("schemaVersion is required")
	}
	if m.SchemaVersion != "v1" {
		return fmt.Errorf("unsupported schemaVersion %q", m.SchemaVersion)
	}
	return ValidateOptionsWithPolicy(&m.Options, policy)
}

func ValidateOptions(opts *Options) error {
	policy, err := ResolveTargetPolicy("")
	if err != nil {
		return err
	}
	return ValidateOptionsWithPolicy(opts, policy)
}

func ValidateOptionsWithPolicy(opts *Options, policy TargetPolicy) error {
	opts.Mode = DeployMode(strings.TrimSpace(strings.ToLower(string(opts.Mode))))
	if opts.Mode == "" {
		opts.Mode = ModeFullDeployment
	}
	switch opts.Mode {
	case ModeFullDeployment, ModeTagBased:
	default:
		return fmt.Errorf("invalid deploy mode %q: must be full or tag-based", opts.Mode)
	}

	opts.Tag = strings.TrimSpace(opts.Tag)
	if opts.Mode == ModeTagBased && opts.Tag == "" {
		return fmt.Errorf("tag is required when mode is tag-based")
	}
	if opts.Mode == ModeFullDeployment {
		opts.Tag = ""
	}

	if len(opts.Targets) == 0 {
		opts.Targets = defaultTargetsForPolicy(policy)
	}
	normalized := make([]string, 0, len(opts.Targets))
	seen := make(map[string]bool, len(opts.Targets))
	for _, t := range opts.Targets {
		u := strings.ToUpper(strings.TrimSpace(t))
		if u == "" {
			continue
		}
		if !policy.Allowed[u] {
			return fmt.Errorf("invalid target %q: must be one of %s", t, strings.Join(policy.Ordered, ","))
		}
		if !seen[u] {
			seen[u] = true
			normalized = append(normalized, u)
		}
	}
	if len(normalized) == 0 {
		return fmt.Errorf("at least one target is required")
	}
	sort.Strings(normalized)
	opts.Targets = normalized

	if opts.ConnectorsOnly && !opts.IncludeConnectors {
		opts.IncludeConnectors = true
	}
	return nil
}

func ParseManifestYAML(data []byte) (Manifest, error) {
	policy, err := ResolveTargetPolicy("")
	if err != nil {
		return Manifest{}, err
	}
	return ParseManifestYAMLWithPolicy(data, policy)
}

func ParseManifestYAMLWithPolicy(data []byte, policy TargetPolicy) (Manifest, error) {
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return Manifest{}, fmt.Errorf("parsing manifest: %w", err)
	}
	if err := ValidateManifestWithPolicy(&m, policy); err != nil {
		return Manifest{}, err
	}
	return m, nil
}

func ResolveTargetPolicy(validTargetsOverride string) (TargetPolicy, error) {
	raw := strings.TrimSpace(validTargetsOverride)
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv(validDeployTargetsEnv))
	}
	if raw == "" {
		raw = defaultValidTargets
	}
	ordered, err := parseValidTargets(raw)
	if err != nil {
		return TargetPolicy{}, err
	}
	allowed := make(map[string]bool, len(ordered))
	for _, t := range ordered {
		allowed[t] = true
	}
	return TargetPolicy{
		Allowed: allowed,
		Ordered: ordered,
	}, nil
}

func parseValidTargets(raw string) ([]string, error) {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]bool, len(parts))
	for _, p := range parts {
		t := strings.ToUpper(strings.TrimSpace(p))
		if t == "" {
			continue
		}
		if !isValidTargetToken(t) {
			return nil, fmt.Errorf("invalid target token %q in valid-targets list", p)
		}
		if !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("valid-targets list must contain at least one value")
	}
	return out, nil
}

func isValidTargetToken(v string) bool {
	for _, r := range v {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}

func defaultTargetsForPolicy(policy TargetPolicy) []string {
	if policy.Allowed["TST"] && policy.Allowed["QA"] {
		return []string{"TST", "QA"}
	}
	if len(policy.Ordered) >= 2 {
		return []string{policy.Ordered[0], policy.Ordered[1]}
	}
	if len(policy.Ordered) == 1 {
		return []string{policy.Ordered[0]}
	}
	return []string{"TST", "QA"}
}

func RenderManifestMarkdown(m Manifest) string {
	opts := m.Options
	var sb strings.Builder
	sb.WriteString("# Release Manifest\n\n")
	_, _ = fmt.Fprintf(&sb, "- Schema Version: `%s`\n", m.SchemaVersion)
	_, _ = fmt.Fprintf(&sb, "- Generated At (UTC): `%s`\n", m.GeneratedAt)
	if m.Source != "" {
		_, _ = fmt.Fprintf(&sb, "- Source: `%s`\n", m.Source)
	}
	_, _ = fmt.Fprintf(&sb, "- Mode: `%s`\n", opts.Mode)
	if opts.Tag != "" {
		_, _ = fmt.Fprintf(&sb, "- Tag: `%s`\n", opts.Tag)
	}
	_, _ = fmt.Fprintf(&sb, "- Targets: `%s`\n", strings.Join(opts.Targets, ", "))
	_, _ = fmt.Fprintf(&sb, "- Include Connectors: `%t`\n", opts.IncludeConnectors)
	_, _ = fmt.Fprintf(&sb, "- Connectors Only: `%t`\n", opts.ConnectorsOnly)
	if opts.ExcludeFile != "" {
		_, _ = fmt.Fprintf(&sb, "- Exclude File: `%s`\n", opts.ExcludeFile)
	}
	return sb.String()
}

func WriteManifestFiles(outputRoot string, m Manifest) (string, string, error) {
	confDir := filepath.Join(outputRoot, "conf")
	logDir := filepath.Join(outputRoot, "logs")
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		return "", "", fmt.Errorf("creating conf directory: %w", err)
	}
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return "", "", fmt.Errorf("creating logs directory: %w", err)
	}

	yamlPath := filepath.Join(confDir, "release_manifest.yaml")
	mdPath := filepath.Join(logDir, "release_manifest.md")

	data, err := yaml.Marshal(m)
	if err != nil {
		return "", "", fmt.Errorf("serializing manifest yaml: %w", err)
	}
	if err := os.WriteFile(yamlPath, data, 0o644); err != nil {
		return "", "", fmt.Errorf("writing manifest yaml: %w", err)
	}
	if err := os.WriteFile(mdPath, []byte(RenderManifestMarkdown(m)), 0o644); err != nil {
		return "", "", fmt.Errorf("writing manifest markdown: %w", err)
	}
	return yamlPath, mdPath, nil
}
