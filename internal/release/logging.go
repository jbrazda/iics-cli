package release

import (
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"
)

type AssetTypeCount struct {
	Type  string
	Count int
}

func AssetCountsByType(assets []Asset) []AssetTypeCount {
	if len(assets) == 0 {
		return nil
	}
	counts := make(map[string]int)
	for _, a := range assets {
		counts[a.Type]++
	}
	out := make([]AssetTypeCount, 0, len(counts))
	for typ, count := range counts {
		out = append(out, AssetTypeCount{Type: typ, Count: count})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Type < out[j].Type
	})
	return out
}

func FormatAssetTypeCounts(counts []AssetTypeCount) string {
	if len(counts) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(counts))
	for _, c := range counts {
		parts = append(parts, fmt.Sprintf("%s=%d", c.Type, c.Count))
	}
	return strings.Join(parts, ", ")
}

func FormatDependencyTable(assets []Asset) string {
	if len(assets) == 0 {
		return "(no assets)"
	}
	rows := make([]Asset, len(assets))
	copy(rows, assets)
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Type != rows[j].Type {
			return rows[i].Type < rows[j].Type
		}
		if rows[i].Path != rows[j].Path {
			return rows[i].Path < rows[j].Path
		}
		return rows[i].Location < rows[j].Location
	})

	var sb strings.Builder
	tw := tabwriter.NewWriter(&sb, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "LOCATION\tDEPENDENCY\tTYPE\tPATH")
	for _, a := range rows {
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", a.Location, a.Dependency, a.Type, a.Path)
	}
	_ = tw.Flush()
	return strings.TrimSuffix(sb.String(), "\n")
}
