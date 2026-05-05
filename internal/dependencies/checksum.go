package dependencies

import "strings"

// ParseChecksumEntries parses exportPackage.chksum content and returns a set of
// package-relative file paths present in the checksum list.
func ParseChecksumEntries(data string) map[string]bool {
	out := make(map[string]bool)
	lines := strings.Split(data, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		path := line
		if idx := strings.Index(line, "="); idx >= 0 {
			path = line[:idx]
		}
		path = strings.TrimSpace(path)
		path = strings.ReplaceAll(path, `\ `, " ")
		if path != "" {
			out[path] = true
		}
	}
	return out
}

// IsObjectChecksumBacked reports whether an export metadata object has a
// corresponding object definition file in exportPackage.chksum.
func IsObjectChecksumBacked(path, objectName, objectType string, entries map[string]bool) bool {
	if len(entries) == 0 || objectName == "" || objectType == "" {
		return false
	}
	basePath := strings.TrimPrefix(path, "/")
	basePath = strings.TrimSuffix(basePath, "/")

	base := objectName + "." + objectType
	if basePath != "" {
		base = basePath + "/" + base
	}

	candidates := []string{
		base + ".xml",
		base + ".zip",
		base + ".json",
	}
	for _, c := range candidates {
		if entries[c] {
			return true
		}
	}
	return false
}
