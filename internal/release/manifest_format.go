package release

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type ManifestFormat string

const (
	ManifestFormatYAML       ManifestFormat = "yaml"
	ManifestFormatJSON       ManifestFormat = "json"
	ManifestFormatProperties ManifestFormat = "properties"
)

const manifestPropertiesPrefix = "iics.release.manifest"

func ParseManifestFormat(raw string) (ManifestFormat, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "yaml", "yml":
		return ManifestFormatYAML, nil
	case "json":
		return ManifestFormatJSON, nil
	case "properties", "props":
		return ManifestFormatProperties, nil
	default:
		return "", fmt.Errorf("unknown manifest format %q; valid formats: yaml, json, properties", raw)
	}
}

func DetectManifestFormat(sourcePath string, data []byte) ManifestFormat {
	switch strings.ToLower(strings.TrimSpace(filepath.Ext(sourcePath))) {
	case ".yaml", ".yml":
		return ManifestFormatYAML
	case ".json":
		return ManifestFormatJSON
	case ".properties", ".props":
		return ManifestFormatProperties
	}

	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return ManifestFormatYAML
	}
	if trimmed[0] == '{' || trimmed[0] == '[' {
		return ManifestFormatJSON
	}
	if hasManifestPropertiesPrefix(trimmed) {
		return ManifestFormatProperties
	}
	return ManifestFormatYAML
}

func ManifestFileExtension(format ManifestFormat) string {
	switch format {
	case ManifestFormatJSON:
		return "json"
	case ManifestFormatProperties:
		return "properties"
	default:
		return "yaml"
	}
}

func MarshalManifest(m Manifest, format ManifestFormat) ([]byte, error) {
	switch format {
	case ManifestFormatJSON:
		data, err := json.MarshalIndent(m, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("serializing manifest json: %w", err)
		}
		return append(data, '\n'), nil
	case ManifestFormatProperties:
		return marshalManifestProperties(m), nil
	default:
		data, err := yaml.Marshal(m)
		if err != nil {
			return nil, fmt.Errorf("serializing manifest yaml: %w", err)
		}
		return data, nil
	}
}

func ParseManifestAutoWithPolicy(sourcePath string, data []byte, policy TargetPolicy) (Manifest, ManifestFormat, error) {
	format := DetectManifestFormat(sourcePath, data)
	m, err := ParseManifestWithPolicy(data, format, policy)
	if err != nil {
		return Manifest{}, "", err
	}
	return m, format, nil
}

func ParseManifestWithPolicy(data []byte, format ManifestFormat, policy TargetPolicy) (Manifest, error) {
	var m Manifest
	switch format {
	case ManifestFormatJSON:
		if err := json.Unmarshal(data, &m); err != nil {
			return Manifest{}, fmt.Errorf("parsing manifest json: %w", err)
		}
	case ManifestFormatProperties:
		var err error
		m, err = parseManifestProperties(data)
		if err != nil {
			return Manifest{}, err
		}
	default:
		if err := yaml.Unmarshal(data, &m); err != nil {
			return Manifest{}, fmt.Errorf("parsing manifest yaml: %w", err)
		}
	}
	if err := ValidateManifestWithPolicy(&m, policy); err != nil {
		return Manifest{}, err
	}
	return m, nil
}

func hasManifestPropertiesPrefix(data []byte) bool {
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}
		return strings.HasPrefix(line, manifestPropertiesPrefix+".")
	}
	return false
}

func marshalManifestProperties(m Manifest) []byte {
	lines := make([]string, 0, 10)
	lines = append(lines, fmt.Sprintf("%s.schemaVersion=%s", manifestPropertiesPrefix, m.SchemaVersion))
	lines = append(lines, fmt.Sprintf("%s.generatedAt=%s", manifestPropertiesPrefix, m.GeneratedAt))
	if m.Source != "" {
		lines = append(lines, fmt.Sprintf("%s.source=%s", manifestPropertiesPrefix, m.Source))
	}
	lines = append(lines, fmt.Sprintf("%s.options.mode=%s", manifestPropertiesPrefix, m.Options.Mode))
	if m.Options.Tag != "" {
		lines = append(lines, fmt.Sprintf("%s.options.tag=%s", manifestPropertiesPrefix, m.Options.Tag))
	}
	lines = append(lines, fmt.Sprintf("%s.options.targets=%s", manifestPropertiesPrefix, strings.Join(m.Options.Targets, ",")))
	lines = append(lines, fmt.Sprintf("%s.options.includeConnectors=%t", manifestPropertiesPrefix, m.Options.IncludeConnectors))
	lines = append(lines, fmt.Sprintf("%s.options.includeConnections=%t", manifestPropertiesPrefix, m.Options.IncludeConnections))
	lines = append(lines, fmt.Sprintf("%s.options.connectorsOnly=%t", manifestPropertiesPrefix, m.Options.ConnectorsOnly))
	if m.Options.ExcludeFile != "" {
		lines = append(lines, fmt.Sprintf("%s.options.excludeFile=%s", manifestPropertiesPrefix, m.Options.ExcludeFile))
	}
	return []byte(strings.Join(lines, "\n") + "\n")
}

func parseManifestProperties(data []byte) (Manifest, error) {
	props := make(map[string]string)
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}
		sep := strings.IndexAny(line, "=:")
		if sep < 0 {
			continue
		}
		key := strings.TrimSpace(line[:sep])
		value := strings.TrimSpace(line[sep+1:])
		props[key] = value
	}

	get := func(key string) string {
		return strings.TrimSpace(props[manifestPropertiesPrefix+"."+key])
	}

	m := Manifest{
		SchemaVersion: get("schemaVersion"),
		GeneratedAt:   get("generatedAt"),
		Source:        get("source"),
		Options: Options{
			Mode:        DeployMode(get("options.mode")),
			Tag:         get("options.tag"),
			ExcludeFile: get("options.excludeFile"),
		},
	}

	targetsRaw := get("options.targets")
	if targetsRaw != "" {
		parts := strings.Split(targetsRaw, ",")
		m.Options.Targets = make([]string, 0, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				m.Options.Targets = append(m.Options.Targets, part)
			}
		}
	}

	if raw := get("options.includeConnectors"); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return Manifest{}, fmt.Errorf("parsing manifest properties includeConnectors: %w", err)
		}
		m.Options.IncludeConnectors = v
	}
	if raw := get("options.includeConnections"); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return Manifest{}, fmt.Errorf("parsing manifest properties includeConnections: %w", err)
		}
		m.Options.IncludeConnections = v
	}
	if raw := get("options.connectorsOnly"); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return Manifest{}, fmt.Errorf("parsing manifest properties connectorsOnly: %w", err)
		}
		m.Options.ConnectorsOnly = v
	}

	return m, nil
}
