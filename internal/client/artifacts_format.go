package client

import "strings"

// DetectArtifactsFormat sniffs input bytes to decide txt/json/csv/yaml.
func DetectArtifactsFormat(data []byte) string {
	preview := strings.TrimSpace(string(data))
	if len(preview) == 0 {
		return "txt"
	}
	if preview[0] == '{' || preview[0] == '[' {
		return "json"
	}
	if strings.HasPrefix(preview, "---") ||
		strings.HasPrefix(preview, "- ") ||
		strings.HasPrefix(preview, "objects:") ||
		strings.HasPrefix(preview, "id:") ||
		strings.HasPrefix(preview, "path:") ||
		strings.HasPrefix(preview, "location:") {
		return "yaml"
	}

	firstLine := preview
	if idx := strings.IndexByte(preview, '\n'); idx >= 0 {
		firstLine = preview[:idx]
	}
	firstLine = strings.TrimSpace(strings.TrimSuffix(firstLine, "\r"))
	normalizedHeader := strings.ToLower(firstLine)
	if normalizedHeader == "id" || normalizedHeader == "path" || normalizedHeader == "type" || normalizedHeader == "location" {
		return "csv"
	}
	if strings.Contains(firstLine, ",") {
		return "csv"
	}
	return "txt"
}
