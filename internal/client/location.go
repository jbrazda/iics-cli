package client

import "strings"

var sysLocationTypes = map[string]bool{
	"Connection": true,
	"AgentGroup": true,
}

// NormalizeLocationPath strips leading root path markers and returns a canonical
// asset path without a leading slash, Explore/, or SYS/.
func NormalizeLocationPath(path string) string {
	normalized := strings.TrimPrefix(strings.TrimSpace(path), "/")
	normalized = strings.TrimPrefix(normalized, "Explore/")
	normalized = strings.TrimPrefix(normalized, "SYS/")
	return normalized
}

// LocationRootForType returns the root folder prefix for an asset type.
func LocationRootForType(assetType string) string {
	if sysLocationTypes[assetType] {
		return "SYS/"
	}
	return "Explore/"
}

// BuildLocation builds a normalized location string in "<root><path>.<type>" form.
func BuildLocation(path, assetType string) string {
	return LocationRootForType(assetType) + NormalizeLocationPath(path) + "." + assetType
}
