package dependencies

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"github.com/jbrazda/iics-cli/internal/client"
)

type BuildManifestParseOptions struct {
	ExcludeFoundTransitive bool
	TargetStatus           string
}

type BuildManifestStats struct {
	TotalRows                int
	IncludedRows             int
	ExcludedTransitiveFound  int
	SelectedStatusColumnName string
}

func canonicalHeader(h string) string {
	return strings.ToLower(strings.TrimSpace(h))
}

func extractStatusTarget(h string) string {
	c := canonicalHeader(h)
	if !strings.HasPrefix(c, "status (") || !strings.HasSuffix(c, ")") {
		return ""
	}
	return strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(c, "status ("), ")"))
}

func ParseBuildManifestCSV(data []byte, opts BuildManifestParseOptions) ([]client.ArtifactEntry, BuildManifestStats, error) {
	stats := BuildManifestStats{}
	r := csv.NewReader(strings.NewReader(string(data)))
	r.FieldsPerRecord = -1

	headers, err := r.Read()
	if err != nil {
		if err == io.EOF {
			return nil, stats, nil
		}
		return nil, stats, fmt.Errorf("reading csv headers: %w", err)
	}

	idx := map[string]int{
		"id":         -1,
		"path":       -1,
		"type":       -1,
		"location":   -1,
		"dependency": -1,
	}
	statusCols := make(map[string]int)
	statusNames := make(map[string]string)
	for i, h := range headers {
		ch := canonicalHeader(h)
		if _, ok := idx[ch]; ok {
			idx[ch] = i
		}
		if target := extractStatusTarget(h); target != "" {
			statusCols[target] = i
			statusNames[target] = strings.TrimSpace(h)
		}
	}

	statusIdx := -1
	if opts.ExcludeFoundTransitive {
		if idx["dependency"] < 0 {
			return nil, stats, fmt.Errorf("manifest csv missing DEPENDENCY column required by --exclude-found-transitive")
		}
		target := strings.ToLower(strings.TrimSpace(opts.TargetStatus))
		if target != "" {
			i, ok := statusCols[target]
			if !ok {
				return nil, stats, fmt.Errorf("status column for target %q not found; expected STATUS (%s)", opts.TargetStatus, opts.TargetStatus)
			}
			statusIdx = i
			stats.SelectedStatusColumnName = statusNames[target]
		} else {
			if len(statusCols) == 0 {
				return nil, stats, fmt.Errorf("manifest csv has no STATUS (<target>) column required by --exclude-found-transitive")
			}
			if len(statusCols) > 1 {
				return nil, stats, fmt.Errorf("manifest csv has multiple STATUS columns; use --status-target")
			}
			for _, i := range statusCols {
				statusIdx = i
			}
			for _, name := range statusNames {
				stats.SelectedStatusColumnName = name
			}
		}
	}

	val := func(row []string, i int) string {
		if i < 0 || i >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[i])
	}

	entries := make([]client.ArtifactEntry, 0)
	for {
		row, readErr := r.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, stats, fmt.Errorf("reading csv row: %w", readErr)
		}
		stats.TotalRows++

		if opts.ExcludeFoundTransitive {
			dep := strings.ToLower(val(row, idx["dependency"]))
			status := strings.ToLower(val(row, statusIdx))
			if dep == "transitive" && status == "found" {
				stats.ExcludedTransitiveFound++
				continue
			}
		}

		id := val(row, idx["id"])
		path := val(row, idx["path"])
		typ := val(row, idx["type"])
		location := val(row, idx["location"])

		if id != "" {
			entries = append(entries, client.ArtifactEntry{ID: id, Type: typ})
			stats.IncludedRows++
			continue
		}
		if location != "" {
			p, t, parseErr := client.ParseLocationString(location)
			if parseErr != nil {
				return nil, stats, fmt.Errorf("parsing location %q: %w", location, parseErr)
			}
			if typ == "" {
				typ = t
			}
			entries = append(entries, client.ArtifactEntry{Path: p, Type: typ})
			stats.IncludedRows++
			continue
		}
		if path != "" {
			entries = append(entries, client.ArtifactEntry{Path: path, Type: typ})
			stats.IncludedRows++
			continue
		}
		return nil, stats, fmt.Errorf("manifest row %d has no id, location, or path", stats.TotalRows)
	}

	return entries, stats, nil
}
