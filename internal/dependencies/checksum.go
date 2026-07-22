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
	for _, c := range ObjectChecksumCandidates(path, objectName, objectType) {
		if entries[c] {
			return true
		}
	}
	return false
}

// ObjectChecksumCandidates returns the candidate package-relative object file
// paths for an exported object. Mass Ingestion asset types (e.g. MI_TASK,
// MI_FILE_LISTENER, MI_SERVICE_CONNECTOR) serialize as "<Name>.<TYPE>.dat"
// rather than .xml/.json, so .dat is included as a generic candidate
// alongside .xml/.zip/.json.
func ObjectChecksumCandidates(path, objectName, objectType string) []string {
	if objectName == "" || objectType == "" {
		return nil
	}
	basePath := strings.TrimPrefix(path, "/")
	basePath = strings.TrimSuffix(basePath, "/")

	base := objectName + "." + objectType
	if basePath != "" {
		base = basePath + "/" + base
	}
	return []string{
		base + ".xml",
		base + ".zip",
		base + ".json",
		base + ".dat",
	}
}
