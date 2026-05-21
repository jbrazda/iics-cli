package dependencies

import (
	"fmt"
	"strings"

	"github.com/jbrazda/iics-cli/internal/client"
)

// ParseSeedEntriesFromInput parses object seed entries from JSON, CSV, YAML, or TXT.
// Supported rows can include ID, LOCATION, or PATH+TYPE fields.
func ParseSeedEntriesFromInput(data []byte) ([]client.ArtifactEntry, error) {
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, fmt.Errorf("stdin is empty")
	}
	format := client.DetectArtifactsFormat(data)
	entries, err := client.ParseArtifactsReader(strings.NewReader(string(data)), format)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("no object seed entries found in input")
	}
	return entries, nil
}

// ParseSeedIDsFromInput is retained for compatibility with ID-only call sites.
func ParseSeedIDsFromInput(data []byte) ([]string, error) {
	entries, err := ParseSeedEntriesFromInput(data)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		if strings.TrimSpace(entry.ID) != "" {
			ids = append(ids, strings.TrimSpace(entry.ID))
		}
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("no IDs found in input")
	}
	return ids, nil
}
