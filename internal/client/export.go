package client

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ExportObject specifies an object to include in an export.
type ExportObject struct {
	ID                  string `json:"id"`
	IncludeDependencies bool   `json:"includeDependencies,omitempty"`
}

// ExportRequest is the request body for creating an export job.
type ExportRequest struct {
	Name    string         `json:"name"`
	Objects []ExportObject `json:"objects"`
}

// ExportCreateOptions holds optional query parameters for creating an export job.
type ExportCreateOptions struct {
	IncludeTags bool
}

// JobStatus represents the status of an async job.
type JobStatus struct {
	State   string `json:"state"`
	Message string `json:"message,omitempty"`
}

// JobObject represents an object within a job.
type JobObject struct {
	ID     string    `json:"id"`
	Name   string    `json:"name"`
	Type   string    `json:"type"`
	Status JobStatus `json:"status"`
}

// ExportJob represents an export job.
type ExportJob struct {
	ID         string      `json:"id"`
	CreateTime string      `json:"createTime,omitempty"`
	Name       string      `json:"name"`
	Status     JobStatus   `json:"status"`
	StartTime  string      `json:"startTime,omitempty"`
	EndTime    string      `json:"endTime,omitempty"`
	Objects    []JobObject `json:"objects,omitempty"`
}

// ArtifactEntry represents a single artifact parsed from an input file.
// Either ID is set (use directly) or Path+Type are set (requires lookup).
type ArtifactEntry struct {
	ID   string
	Path string
	Type string
}

// CreateExport starts an export job.
func (c *Client) CreateExport(ctx context.Context, req *ExportRequest) (*ExportJob, error) {
	var resp ExportJob
	if err := c.doJSON(ctx, http.MethodPost, fmt.Sprintf("%s/export", BaseAPIPathV3), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// StartExport starts an export job with optional query parameters (e.g. includeTagInformation).
func (c *Client) StartExport(ctx context.Context, req *ExportRequest, opts ExportCreateOptions) (*ExportJob, error) {
	query := make(map[string]string)
	if opts.IncludeTags {
		query["includeTagInformation"] = "true"
	}
	var resp ExportJob
	if err := c.doJSONWithQuery(ctx, http.MethodPost, fmt.Sprintf("%s/export", BaseAPIPathV3), query, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetExportStatus retrieves the status of an export job.
func (c *Client) GetExportStatus(ctx context.Context, jobID string, expand bool) (*ExportJob, error) {
	query := make(map[string]string)
	if expand {
		query["expand"] = "objects"
	}

	var resp ExportJob
	if err := c.doJSONWithQuery(ctx, http.MethodGet, fmt.Sprintf("%s/export/%s", BaseAPIPathV3, jobID), query, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DownloadExportPackage downloads the export ZIP package.
func (c *Client) DownloadExportPackage(ctx context.Context, jobID string, dest io.Writer) error {
	path := fmt.Sprintf("%s/export/%s/package", BaseAPIPathV3, jobID)
	body, err := c.doRaw(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	defer func() { _ = body.Close() }()

	if _, err := io.Copy(dest, body); err != nil {
		return fmt.Errorf("downloading export package: %w", err)
	}
	return nil
}

// DownloadExportLog downloads the export job log.
func (c *Client) DownloadExportLog(ctx context.Context, jobID string, dest io.Writer) error {
	path := fmt.Sprintf("%s/export/%s/log", BaseAPIPathV3, jobID)
	body, err := c.doRaw(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	defer func() { _ = body.Close() }()

	if _, err := io.Copy(dest, body); err != nil {
		return fmt.Errorf("downloading export log: %w", err)
	}
	return nil
}

// ParseLocationString parses a location like "Explore/path/to/Asset.TYPE"
// into its path component (without "Explore/" prefix) and type component.
func ParseLocationString(location string) (path, assetType string, err error) {
	loc := strings.TrimPrefix(location, "Explore/")
	dotIdx := strings.LastIndex(loc, ".")
	if dotIdx < 0 {
		return "", "", fmt.Errorf("invalid location %q: expected Explore/path.TYPE format", location)
	}
	return loc[:dotIdx], loc[dotIdx+1:], nil
}

// ParseArtifactsFile reads an artifact list from a file.
// Format is auto-detected from the file extension: .txt, .json, .yaml/.yml, .csv.
func ParseArtifactsFile(filePath string) ([]ArtifactEntry, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening artifacts file: %w", err)
	}
	defer func() { _ = f.Close() }()

	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filePath), "."))
	return ParseArtifactsReader(f, ext)
}

// artifactObjectRaw is used for flexible JSON/YAML artifact parsing.
type artifactObjectRaw struct {
	ID       string `json:"id" yaml:"id"`
	Path     string `json:"path" yaml:"path"`
	Type     string `json:"type" yaml:"type"`
	Location string `json:"location" yaml:"location"`
}

// artifactListRaw is the objects list format {"objects": [...]} or {"count": N, "objects": [...]}.
type artifactListRaw struct {
	Objects []artifactObjectRaw `json:"objects" yaml:"objects"`
}

// ParseArtifactsReader reads artifact entries from r using the given format hint.
// format may be "txt", "json", "yaml", "yml", or "csv".
// If format is empty or unrecognized, TXT (one location per line) is assumed.
func ParseArtifactsReader(r io.Reader, format string) ([]ArtifactEntry, error) {
	switch format {
	case "json":
		return parseArtifactsJSON(r)
	case "yaml", "yml":
		return parseArtifactsYAML(r)
	case "csv":
		return parseArtifactsCSV(r)
	default:
		return parseArtifactsTXT(r)
	}
}

func parseArtifactsTXT(r io.Reader) ([]ArtifactEntry, error) {
	var entries []ArtifactEntry
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		path, assetType, err := ParseLocationString(line)
		if err != nil {
			return nil, fmt.Errorf("parsing location %q: %w", line, err)
		}
		entries = append(entries, ArtifactEntry{Path: path, Type: assetType})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading artifacts: %w", err)
	}
	return entries, nil
}

func parseArtifactsFromRawObjects(objs []artifactObjectRaw) ([]ArtifactEntry, error) {
	entries := make([]ArtifactEntry, 0, len(objs))
	for _, o := range objs {
		if o.ID != "" {
			entries = append(entries, ArtifactEntry{ID: o.ID, Type: o.Type})
			continue
		}
		if o.Location != "" {
			path, assetType, err := ParseLocationString(o.Location)
			if err != nil {
				return nil, fmt.Errorf("parsing location %q: %w", o.Location, err)
			}
			entries = append(entries, ArtifactEntry{Path: path, Type: assetType})
			continue
		}
		if o.Path != "" && o.Type != "" {
			entries = append(entries, ArtifactEntry{Path: o.Path, Type: o.Type})
			continue
		}
		return nil, fmt.Errorf("artifact entry has no id, location, or path+type: %+v", o)
	}
	return entries, nil
}

func parseArtifactsJSON(r io.Reader) ([]ArtifactEntry, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading JSON artifacts: %w", err)
	}

	// Try objects list format {"objects": [...]} first.
	var list artifactListRaw
	if err := json.Unmarshal(data, &list); err == nil && list.Objects != nil {
		return parseArtifactsFromRawObjects(list.Objects)
	}

	// Try plain array [...].
	var arr []artifactObjectRaw
	if err := json.Unmarshal(data, &arr); err != nil {
		return nil, fmt.Errorf("parsing JSON artifacts: %w", err)
	}
	return parseArtifactsFromRawObjects(arr)
}

func parseArtifactsYAML(r io.Reader) ([]ArtifactEntry, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading YAML artifacts: %w", err)
	}

	// Try objects list format first.
	var list artifactListRaw
	if err := yaml.Unmarshal(data, &list); err == nil && list.Objects != nil {
		return parseArtifactsFromRawObjects(list.Objects)
	}

	// Try plain array.
	var arr []artifactObjectRaw
	if err := yaml.Unmarshal(data, &arr); err != nil {
		return nil, fmt.Errorf("parsing YAML artifacts: %w", err)
	}
	return parseArtifactsFromRawObjects(arr)
}

func parseArtifactsCSV(r io.Reader) ([]ArtifactEntry, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true

	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("reading CSV header: %w", err)
	}

	// Normalize headers to lowercase for flexible matching.
	colIndex := make(map[string]int)
	for i, h := range headers {
		colIndex[strings.ToLower(strings.TrimSpace(h))] = i
	}
	idIdx, hasID := colIndex["id"]
	pathIdx, hasPath := colIndex["path"]
	typeIdx, hasType := colIndex["type"]
	locIdx, hasLoc := colIndex["location"]

	var entries []ArtifactEntry
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading CSV row: %w", err)
		}

		var entry ArtifactEntry
		if hasID && idIdx < len(row) {
			entry.ID = strings.TrimSpace(row[idIdx])
		}
		if hasType && typeIdx < len(row) {
			entry.Type = strings.TrimSpace(row[typeIdx])
		}
		if hasPath && pathIdx < len(row) {
			entry.Path = strings.TrimSpace(row[pathIdx])
		}
		// If no ID, try resolving from location column.
		if entry.ID == "" && hasLoc && locIdx < len(row) {
			loc := strings.TrimSpace(row[locIdx])
			if loc != "" {
				path, assetType, err := ParseLocationString(loc)
				if err != nil {
					return nil, fmt.Errorf("parsing location %q: %w", loc, err)
				}
				entry.Path = path
				entry.Type = assetType
			}
		}

		if entry.ID == "" && (entry.Path == "" || entry.Type == "") {
			continue // skip incomplete rows
		}
		entries = append(entries, entry)
	}
	return entries, nil
}
