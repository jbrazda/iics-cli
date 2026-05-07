package release

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/config"
	"github.com/jbrazda/iics-cli/internal/dependencies"
)

type Asset struct {
	ID         string
	Path       string
	Type       string
	Location   string
	Dependency string
}

var publishableTypes = map[string]bool{
	"AI_SERVICE_CONNECTOR": true,
	"AI_CONNECTION":        true,
	"PROCESS":              true,
	"GUIDE":                true,
	"TASKFLOW":             true,
}

var connectorTypes = map[string]bool{
	"AI_SERVICE_CONNECTOR": true,
	"AI_CONNECTION":        true,
	"Connection":           true,
}

func ResolveTagAssets(ctx context.Context, c *client.Client, tag string) ([]Asset, error) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return nil, fmt.Errorf("tag is required")
	}
	resp, err := c.ListAllObjects(ctx, client.ObjectsListOptions{
		Query: fmt.Sprintf("tag=='%s'", tag),
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("listing tagged objects: %w", err)
	}
	if len(resp.Objects) == 0 {
		return nil, fmt.Errorf("no objects found for tag %q", tag)
	}

	seedIDs := make([]string, 0, len(resp.Objects))
	seedSet := make(map[string]bool, len(resp.Objects))
	seedObj := make(map[string]client.Object, len(resp.Objects))
	for _, o := range resp.Objects {
		if o.ID == "" {
			continue
		}
		seedIDs = append(seedIDs, o.ID)
		seedSet[o.ID] = true
		seedObj[o.ID] = o
	}
	if len(seedIDs) == 0 {
		return nil, fmt.Errorf("tagged objects did not include resolvable IDs")
	}

	graph, err := dependencies.TraverseByIDs(ctx, c, seedIDs, "uses", 0, 0)
	if err != nil {
		return nil, fmt.Errorf("resolving transitive dependencies: %w", err)
	}

	result := make(map[string]Asset, len(graph.Nodes))
	for id, n := range graph.Nodes {
		path := strings.TrimPrefix(strings.TrimPrefix(n.Path, "/"), "Explore/")
		typ := n.Type
		if path == "" || typ == "" {
			if so, ok := seedObj[id]; ok {
				path = strings.TrimPrefix(strings.TrimPrefix(so.Path, "/"), "Explore/")
				typ = so.Type
			}
		}
		if path == "" || typ == "" {
			continue
		}
		loc := "Explore/" + path + "." + typ
		dep := "transitive"
		if seedSet[id] {
			dep = "explicit"
		}
		result[loc] = Asset{
			ID:         id,
			Path:       path,
			Type:       typ,
			Location:   loc,
			Dependency: dep,
		}
	}

	for _, o := range resp.Objects {
		if o.Path == "" || o.Type == "" {
			continue
		}
		path := strings.TrimPrefix(strings.TrimPrefix(o.Path, "/"), "Explore/")
		loc := "Explore/" + path + "." + o.Type
		if _, ok := result[loc]; ok {
			continue
		}
		result[loc] = Asset{
			ID:         o.ID,
			Path:       path,
			Type:       o.Type,
			Location:   loc,
			Dependency: "explicit",
		}
	}

	assets := make([]Asset, 0, len(result))
	for _, a := range result {
		assets = append(assets, a)
	}
	sort.Slice(assets, func(i, j int) bool {
		if assets[i].Type != assets[j].Type {
			return assets[i].Type < assets[j].Type
		}
		return assets[i].Path < assets[j].Path
	})
	return assets, nil
}

func LoadExcludePatterns(filePath string) ([]*regexp.Regexp, error) {
	if strings.TrimSpace(filePath) == "" {
		return nil, nil
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading exclude file: %w", err)
	}
	lines := strings.Split(string(data), "\n")
	patterns := make([]*regexp.Regexp, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		re, err := regexp.Compile(line)
		if err != nil {
			return nil, fmt.Errorf("invalid regex %q in %s: %w", line, filePath, err)
		}
		patterns = append(patterns, re)
	}
	return patterns, nil
}

func ApplyPolicies(assets []Asset, includeConnectors, connectorsOnly bool, excludePatterns []*regexp.Regexp) []Asset {
	filtered := make([]Asset, 0, len(assets))
	for _, a := range assets {
		excluded := false
		for _, re := range excludePatterns {
			if re.MatchString(a.Location) {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}
		if connectorsOnly {
			if !connectorTypes[a.Type] {
				continue
			}
		} else if !includeConnectors && connectorTypes[a.Type] {
			continue
		}
		filtered = append(filtered, a)
	}
	return filtered
}

func ConnectorAssets(assets []Asset) []Asset {
	out := make([]Asset, 0)
	for _, a := range assets {
		if connectorTypes[a.Type] {
			out = append(out, a)
		}
	}
	return out
}

func PublishAssets(assets []Asset) []Asset {
	out := make([]Asset, 0)
	for _, a := range assets {
		if publishableTypes[a.Type] {
			out = append(out, a)
		}
	}
	return out
}

func FilterMissingTransitiveForTarget(ctx context.Context, targetProfileName string, assets []Asset) ([]Asset, error) {
	cfg, err := config.Load("")
	if err != nil {
		return nil, fmt.Errorf("loading config for target %q: %w", targetProfileName, err)
	}
	resolvedProfileName, err := resolveTargetProfileName(cfg, targetProfileName)
	if err != nil {
		return nil, err
	}
	profile, err := cfg.ResolveTargetProfile(resolvedProfileName)
	if err != nil {
		return nil, err
	}
	loginURL, err := profile.GetLoginURL()
	if err != nil {
		return nil, fmt.Errorf("resolving target login URL for profile %q: %w", resolvedProfileName, err)
	}
	tc := client.NewClient(loginURL, profile.Username, profile.Password)

	missingByLocation := make(map[string]bool, len(assets))
	for _, a := range assets {
		if a.Dependency != "transitive" {
			continue
		}
		exists, existsErr := assetExistsInTarget(ctx, tc, a)
		if existsErr != nil {
			return nil, fmt.Errorf("checking target %q for %s: %w", targetProfileName, a.Location, existsErr)
		}
		missingByLocation[a.Location] = !exists
	}

	return ApplyMissingTransitivePolicy(assets, missingByLocation), nil
}

func ApplyMissingTransitivePolicy(assets []Asset, missingByLocation map[string]bool) []Asset {
	filtered := make([]Asset, 0, len(assets))
	for _, a := range assets {
		if a.Dependency != "transitive" {
			filtered = append(filtered, a)
			continue
		}
		if missingByLocation[a.Location] {
			filtered = append(filtered, a)
		}
	}
	return filtered
}

func resolveTargetProfileName(cfg *config.Config, requested string) (string, error) {
	req := strings.TrimSpace(requested)
	if req == "" {
		return "", fmt.Errorf("target profile is required")
	}
	if _, ok := cfg.Profiles[req]; ok {
		return req, nil
	}
	lowerReq := strings.ToLower(req)
	for name := range cfg.Profiles {
		if strings.ToLower(name) == lowerReq {
			return name, nil
		}
	}
	return "", fmt.Errorf("target profile %q not found in config", requested)
}

func assetExistsInTarget(ctx context.Context, tc *client.Client, a Asset) (bool, error) {
	if a.Type == "Connection" {
		name := a.Path
		if idx := strings.LastIndex(name, "/"); idx >= 0 {
			name = name[idx+1:]
		}
		_, err := tc.GetConnectionByName(ctx, name)
		if err == nil {
			return true, nil
		}
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.IsNotFound() {
			return false, nil
		}
		return false, err
	}

	resp, err := tc.Lookup(ctx, []client.LookupObject{{Path: a.Path, Type: a.Type}})
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) &&
			bytes.Contains(apiErr.ResponseBody, []byte(`"V3API_LookupError_012"`)) {
			return false, nil
		}
		return false, err
	}
	return len(resp.Objects) > 0, nil
}

func WriteAssetsCSV(path string, assets []Asset, fields []string) error {
	if len(fields) == 0 {
		fields = []string{"location"}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating csv %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	w := csv.NewWriter(f)
	header := make([]string, len(fields))
	for i, field := range fields {
		header[i] = strings.ToUpper(field)
	}
	if err := w.Write(header); err != nil {
		return err
	}
	for _, a := range assets {
		row := make([]string, 0, len(fields))
		for _, field := range fields {
			switch strings.ToLower(field) {
			case "location":
				row = append(row, a.Location)
			case "dependency":
				row = append(row, a.Dependency)
			case "id":
				row = append(row, a.ID)
			case "type":
				row = append(row, a.Type)
			case "path":
				row = append(row, a.Path)
			default:
				row = append(row, "")
			}
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return err
	}
	return nil
}
