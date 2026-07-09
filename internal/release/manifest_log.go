package release

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

const DefaultManifestLogPath = "target/iics/import/logs/release_manifest.md"

func ResolveManifestLogPath(enabled bool, value string) (bool, string) {
	if !enabled {
		return false, ""
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return true, DefaultManifestLogPath
	}
	return true, value
}

func AppendManifestLog(path, markdown string) error {
	if strings.TrimSpace(path) == "" || strings.TrimSpace(markdown) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating log directory: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening log file: %w", err)
	}
	defer func() { _ = f.Close() }()
	if _, err := f.WriteString(ensureSectionSpacing(markdown)); err != nil {
		return fmt.Errorf("writing log file: %w", err)
	}
	return nil
}

func ensureSectionSpacing(s string) string {
	s = strings.TrimRight(s, "\n")
	if s == "" {
		return ""
	}
	return "\n\n" + s + "\n"
}

func MarkdownTable(headers []string, rows [][]string, rightAligned map[int]bool) string {
	var b strings.Builder
	escapedHeaders := make([]string, len(headers))
	widths := make([]int, len(headers))
	for i, header := range headers {
		escapedHeaders[i] = EscapeMarkdownCell(header)
		widths[i] = maxInt(markdownCellWidth(escapedHeaders[i]), minMarkdownSeparatorWidth(rightAligned[i]))
	}

	escapedRows := make([][]string, 0, len(rows))
	for _, row := range rows {
		escapedRow := make([]string, len(headers))
		for i := range headers {
			if i < len(row) {
				escapedRow[i] = EscapeMarkdownCell(row[i])
			} else {
				escapedRow[i] = EscapeMarkdownCell("")
			}
			widths[i] = maxInt(widths[i], markdownCellWidth(escapedRow[i]))
		}
		escapedRows = append(escapedRows, escapedRow)
	}

	writeAlignedMarkdownRow(&b, escapedHeaders, widths, nil)
	separators := make([]string, len(headers))
	for i := range headers {
		if rightAligned[i] {
			separators[i] = strings.Repeat("-", widths[i]-1) + ":"
		} else {
			separators[i] = strings.Repeat("-", widths[i])
		}
	}
	writeAlignedMarkdownRow(&b, separators, widths, nil)
	for _, row := range escapedRows {
		writeAlignedMarkdownRow(&b, row, widths, rightAligned)
	}
	return b.String()
}

func writeAlignedMarkdownRow(b *strings.Builder, cells []string, widths []int, rightAligned map[int]bool) {
	b.WriteString("|")
	for i, cell := range cells {
		width := widths[i]
		if rightAligned[i] {
			cell = padMarkdownCellLeft(cell, width)
		} else {
			cell = padMarkdownCellRight(cell, width)
		}
		b.WriteString(" ")
		b.WriteString(cell)
		b.WriteString(" |")
	}
	b.WriteString("\n")
}

func minMarkdownSeparatorWidth(rightAligned bool) int {
	if rightAligned {
		return 4
	}
	return 3
}

func markdownCellWidth(s string) int {
	return utf8.RuneCountInString(s)
}

func padMarkdownCellLeft(s string, width int) string {
	if pad := width - markdownCellWidth(s); pad > 0 {
		return strings.Repeat(" ", pad) + s
	}
	return s
}

func padMarkdownCellRight(s string, width int) string {
	if pad := width - markdownCellWidth(s); pad > 0 {
		return s + strings.Repeat(" ", pad)
	}
	return s
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func EscapeMarkdownCell(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return " "
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = strings.ReplaceAll(s, "\n", "<br>")
	s = strings.ReplaceAll(s, "|", `\|`)
	return s
}

func FencedBlock(language, body string) string {
	fence := "```"
	longest := 0
	current := 0
	for _, r := range body {
		if r == '`' {
			current++
			if current > longest {
				longest = current
			}
			continue
		}
		current = 0
	}
	if longest >= len(fence) {
		fence = strings.Repeat("`", longest+1)
	}
	if language != "" {
		return fence + language + "\n" + body + "\n" + fence + "\n"
	}
	return fence + "\n" + body + "\n" + fence + "\n"
}

type ManifestLogAsset struct {
	ID         string
	Location   string
	Type       string
	Path       string
	Dependency string
	Status     string
}

type ReleasePlanLog struct {
	SchemaVersion      string
	GeneratedAt        time.Time
	Source             string
	Mode               DeployMode
	Tag                string
	Targets            []string
	IncludeConnectors  bool
	IncludeConnections bool
	AssetsByTarget     map[string][]ManifestLogAsset
	PublishByTarget    map[string][]ManifestLogAsset
}

func RenderReleasePlanLog(r ReleasePlanLog) string {
	var b strings.Builder
	generatedAt := r.GeneratedAt.UTC().Format(time.RFC3339)
	if r.SchemaVersion == "" {
		r.SchemaVersion = "v1"
	}
	tag := r.Tag
	if tag == "" {
		tag = "(none)"
	}
	b.WriteString("# Release Manifest\n\n")
	_, _ = fmt.Fprintf(&b, "- Schema Version: `%s`\n", r.SchemaVersion)
	_, _ = fmt.Fprintf(&b, "- Generated At (UTC): `%s`\n", generatedAt)
	_, _ = fmt.Fprintf(&b, "- Source: `%s`\n", r.Source)
	_, _ = fmt.Fprintf(&b, "- Mode: `%s`\n", r.Mode)
	_, _ = fmt.Fprintf(&b, "- Tag: `%s`\n", tag)
	_, _ = fmt.Fprintf(&b, "- Targets: `%s`\n", strings.Join(r.Targets, ", "))
	_, _ = fmt.Fprintf(&b, "- Include Connectors: `%t`\n", r.IncludeConnectors)
	_, _ = fmt.Fprintf(&b, "- Include Connections: `%t`\n", r.IncludeConnections)
	b.WriteString("\n## Package Content per Target\n\n")
	b.WriteString(renderTypeCountsByTarget(r.Targets, r.AssetsByTarget))
	b.WriteString("\n## Publishable Assets per Target\n\n")
	b.WriteString(renderTypeCountsByTarget(r.Targets, r.PublishByTarget))
	return b.String()
}

func renderTypeCountsByTarget(targets []string, assetsByTarget map[string][]ManifestLogAsset) string {
	typeSet := map[string]bool{}
	counts := map[string]map[string]int{}
	for _, target := range targets {
		counts[target] = map[string]int{}
		for _, asset := range assetsByTarget[target] {
			typ := strings.TrimSpace(asset.Type)
			if typ == "" {
				typ = "(unknown)"
			}
			typeSet[typ] = true
			counts[target][typ]++
		}
	}
	types := make([]string, 0, len(typeSet))
	for typ := range typeSet {
		types = append(types, typ)
	}
	sort.Strings(types)
	headers := []string{"TYPE"}
	right := map[int]bool{}
	for i, target := range targets {
		headers = append(headers, "COUNT ("+strings.ToUpper(target)+")")
		right[i+1] = true
	}
	rows := make([][]string, 0, len(types))
	for _, typ := range types {
		row := []string{typ}
		for _, target := range targets {
			row = append(row, fmt.Sprintf("%d", counts[target][typ]))
		}
		rows = append(rows, row)
	}
	if len(rows) == 0 {
		rows = append(rows, append([]string{"(none)"}, zeroCountCells(len(targets))...))
	}
	return MarkdownTable(headers, rows, right)
}

func zeroCountCells(n int) []string {
	cells := make([]string, n)
	for i := range cells {
		cells[i] = "0"
	}
	return cells
}

type PackageCreateLog struct {
	PackagePath             string
	Source                  string
	SelectionManifest       string
	PackageName             string
	FileCount               int
	AssetsSelected          int
	ParentContainersAdded   int
	TransitiveDepsIncluded  int
	TransitiveFoundExcluded int
	DanglingRefsPruned      int
	Warnings                []string
	IncludedAssets          []ManifestLogAsset
}

func RenderPackageCreateLog(r PackageCreateLog) string {
	var b strings.Builder
	b.WriteString("## Package Build Report\n\n")
	b.WriteString(MarkdownTable([]string{"FIELD", "VALUE"}, [][]string{
		{"Package", r.PackagePath},
		{"Source", r.Source},
		{"Selection Manifest", valueOrNone(r.SelectionManifest)},
		{"Package Name", valueOrNone(r.PackageName)},
		{"Files", fmt.Sprintf("%d", r.FileCount)},
	}, nil))
	b.WriteString("\n### Selection Summary\n\n")
	b.WriteString(MarkdownTable([]string{"FIELD", "VALUE"}, [][]string{
		{"Assets selected", fmt.Sprintf("%d", r.AssetsSelected)},
		{"Parent containers added", fmt.Sprintf("%d", r.ParentContainersAdded)},
		{"Transitive deps included", fmt.Sprintf("%d", r.TransitiveDepsIncluded)},
		{"Transitive found excluded", fmt.Sprintf("%d", r.TransitiveFoundExcluded)},
		{"Dangling refs pruned", fmt.Sprintf("%d", r.DanglingRefsPruned)},
	}, map[int]bool{1: true}))
	if len(r.Warnings) > 0 {
		b.WriteString("\n### Warnings\n\n")
		rows := make([][]string, 0, len(r.Warnings))
		for _, warning := range r.Warnings {
			rows = append(rows, []string{warning})
		}
		b.WriteString(MarkdownTable([]string{"WARNING"}, rows, nil))
	}
	if len(r.IncludedAssets) > 0 {
		b.WriteString("\n### Included Assets\n\n")
		b.WriteString(renderAssetsTable(r.IncludedAssets, []string{"PATH", "TYPE", "ID"}))
	}
	return b.String()
}

type ExportRunLog struct {
	JobID     string
	Name      string
	State     string
	Message   string
	ExportZIP string
	ExportLog string
	Objects   []ManifestLogAsset
}

func RenderExportRunLog(r ExportRunLog) string {
	var b strings.Builder
	b.WriteString("## Backup and Rollback Plan\n\n### Export Summary\n\n")
	b.WriteString(MarkdownTable([]string{"FIELD", "VALUE"}, [][]string{
		{"Job ID", r.JobID},
		{"Name", r.Name},
		{"State", r.State},
		{"Message", valueOrNone(r.Message)},
		{"Export ZIP", r.ExportZIP},
		{"Export Log", valueOrNone(r.ExportLog)},
	}, nil))
	if len(r.Objects) > 0 {
		b.WriteString("\n### Exported Objects\n\n")
		b.WriteString(renderAssetsTable(r.Objects, []string{"ID", "PATH", "TYPE", "STATUS"}))
	}
	return b.String()
}

type ImportRunLog struct {
	JobID      string
	Name       string
	State      string
	Message    string
	StartTime  string
	EndTime    string
	Objects    []ImportLogObject
	LogContent string
}

type ImportLogObject struct {
	SourceID   string
	SourcePath string
	SourceName string
	TargetName string
	SourceType string
	State      string
	Message    string
}

func RenderImportRunLog(r ImportRunLog) string {
	var b strings.Builder
	b.WriteString("## Import Report\n\n### Import Summary\n\n")
	b.WriteString(MarkdownTable([]string{"FIELD", "VALUE"}, [][]string{
		{"Job ID", r.JobID},
		{"Name", r.Name},
		{"State", r.State},
		{"Message", valueOrNone(r.Message)},
		{"Start Date", r.StartTime},
		{"End Date", r.EndTime},
		{"Total", fmt.Sprintf("%d", len(r.Objects))},
		{"Errors", fmt.Sprintf("%d", importErrorCount(r.Objects))},
	}, map[int]bool{1: true}))
	if len(r.Objects) > 0 {
		rows := make([][]string, 0, len(r.Objects))
		for _, object := range r.Objects {
			rows = append(rows, []string{object.SourceID, object.SourcePath, object.SourceName, object.TargetName, object.SourceType, object.State, object.Message})
		}
		b.WriteString("\n### Imported Objects\n\n")
		b.WriteString(MarkdownTable([]string{"SOURCE ID", "SOURCE PATH", "SOURCE NAME", "TARGET NAME", "SOURCE TYPE", "STATE", "MESSAGE"}, rows, nil))
	}
	if strings.TrimSpace(r.LogContent) != "" {
		b.WriteString("\n### Import Log\n\n")
		b.WriteString(FencedBlock("txt", r.LogContent))
	}
	return b.String()
}

func importErrorCount(objects []ImportLogObject) int {
	count := 0
	for _, object := range objects {
		if object.State != "" && object.State != "SUCCESS" {
			count++
		}
	}
	return count
}

type PublishRunLog struct {
	Batches []PublishBatchLog
}

type PublishBatchLog struct {
	Batch     int
	Group     string // asset group this batch belongs to: CAI or TASKFLOW
	JobID     string
	State     string
	StartDate string
	EndDate   string
	Duration  string
	Total     int
	Published int
	Errors    int
	Items     []PublishItemLog
}

type PublishItemLog struct {
	Batch     int
	Index     int
	GUID      string
	AssetPath string
	State     string
	StartDate string
	EndDate   string
	Duration  string
	Detail    string
}

func RenderPublishRunLog(r PublishRunLog) string {
	var b strings.Builder
	b.WriteString("## Publish Report\n\n### Publish Summary\n\n")
	rows := make([][]string, 0, len(r.Batches)+1)
	var total, published, errors int
	for _, batch := range r.Batches {
		rows = append(rows, []string{fmt.Sprintf("%d", batch.Batch), batch.Group, batch.JobID, batch.State, fmt.Sprintf("%d", batch.Total), fmt.Sprintf("%d", batch.Published), fmt.Sprintf("%d", batch.Errors), batch.StartDate, batch.EndDate, batch.Duration})
		total += batch.Total
		published += batch.Published
		errors += batch.Errors
	}
	if len(r.Batches) > 1 {
		rows = append(rows, []string{"TOTAL", "", "", publishOverallState(errors), fmt.Sprintf("%d", total), fmt.Sprintf("%d", published), fmt.Sprintf("%d", errors), "", "", ""})
	}
	b.WriteString(MarkdownTable([]string{"BATCH", "GROUP", "JOB ID", "STATE", "TOTAL", "PUBLISHED", "ERRORS", "START DATE", "END DATE", "DURATION"}, rows, map[int]bool{4: true, 5: true, 6: true}))
	var items []PublishItemLog
	var failed []PublishItemLog
	for _, batch := range r.Batches {
		items = append(items, batch.Items...)
		for _, item := range batch.Items {
			if item.State != "" && item.State != "SUCCESS" {
				failed = append(failed, item)
			}
		}
	}
	if len(items) > 0 {
		itemRows := make([][]string, 0, len(items))
		for _, item := range items {
			itemRows = append(itemRows, []string{fmt.Sprintf("%d", item.Batch), fmt.Sprintf("%d", item.Index), item.GUID, item.AssetPath, item.State, item.StartDate, item.EndDate, item.Duration})
		}
		b.WriteString("\n### Publish Items\n\n")
		b.WriteString(MarkdownTable([]string{"BATCH", "INDEX", "GUID", "ASSET PATH", "STATE", "START DATE", "END DATE", "DURATION"}, itemRows, map[int]bool{0: true, 1: true}))
	}
	if len(failed) > 0 {
		errorRows := make([][]string, 0, len(failed))
		for _, item := range failed {
			errorRows = append(errorRows, []string{item.GUID, item.AssetPath, item.State, item.Detail})
		}
		b.WriteString("\n## Publish Errors\n\n")
		b.WriteString(MarkdownTable([]string{"GUID", "ASSET PATH", "STATE", "DETAIL"}, errorRows, nil))
	}
	return b.String()
}

func publishOverallState(errors int) string {
	if errors > 0 {
		return "ERROR"
	}
	return "SUCCESS"
}

func renderAssetsTable(assets []ManifestLogAsset, headers []string) string {
	rows := make([][]string, 0, len(assets))
	for _, asset := range assets {
		row := make([]string, 0, len(headers))
		for _, header := range headers {
			switch header {
			case "LOCATION":
				row = append(row, asset.Location)
			case "PATH":
				row = append(row, asset.Path)
			case "TYPE":
				row = append(row, asset.Type)
			case "ID":
				row = append(row, asset.ID)
			case "STATUS":
				row = append(row, asset.Status)
			case "DEPENDENCY":
				row = append(row, asset.Dependency)
			}
		}
		rows = append(rows, row)
	}
	return MarkdownTable(headers, rows, nil)
}

func valueOrNone(s string) string {
	if strings.TrimSpace(s) == "" {
		return "(none)"
	}
	return s
}
