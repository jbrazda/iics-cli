package dependencies

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/jbrazda/iics-cli/internal/client"
)

// ResolvedSeedAsset is one resolved dependency asset with explicit/transitive class.
type ResolvedSeedAsset struct {
	ID         string
	Path       string
	Type       string
	Location   string
	Dependency string
}

// SeedResolutionStats captures key counters from dependency seed resolution.
type SeedResolutionStats struct {
	InputEntries            int
	ResolvedSeedIDs         int
	ContainerRoots          int
	ExpandedExplicitObjects int
	ResolvedAssets          int
	ExplicitAssets          int
	TransitiveAssets        int
}

type resolvedSeedObject struct {
	ID   string
	Path string
	Type string
}

// ResolveSeedAssets resolves explicit and transitive dependency assets from seed entries.
//
// Entries can include ID, LOCATION, or PATH+TYPE. Project and Folder entries are expanded
// recursively so all contained objects are treated as explicit.
func ResolveSeedAssets(
	ctx context.Context,
	c *client.Client,
	entries []client.ArtifactEntry,
	refType string,
	limit, skip int,
) ([]ResolvedSeedAsset, SeedResolutionStats, error) {
	stats := SeedResolutionStats{InputEntries: len(entries)}
	if c == nil {
		return nil, stats, fmt.Errorf("client is required")
	}
	if len(entries) == 0 {
		return nil, stats, fmt.Errorf("at least one seed entry is required")
	}

	normalized := normalizeSeedEntries(entries)
	explicitByID, containerRoots, err := resolveExplicitSeedObjects(ctx, c, normalized)
	if err != nil {
		return nil, stats, err
	}
	stats.ResolvedSeedIDs = len(explicitByID)
	stats.ContainerRoots = len(containerRoots)

	if len(containerRoots) > 0 {
		added, expandErr := expandContainerObjects(ctx, c, explicitByID, containerRoots)
		if expandErr != nil {
			return nil, stats, expandErr
		}
		stats.ExpandedExplicitObjects = added
	}

	seedIDs := make([]string, 0, len(explicitByID))
	for id, meta := range explicitByID {
		if meta.Type == "Project" || meta.Type == "Folder" {
			continue
		}
		seedIDs = append(seedIDs, id)
	}
	assetsByLocation := make(map[string]ResolvedSeedAsset, len(explicitByID))
	if len(seedIDs) > 0 {
		sort.Strings(seedIDs)

		graph, err := TraverseByIDs(ctx, c, seedIDs, refType, limit, skip)
		if err != nil {
			return nil, stats, err
		}

		for id, n := range graph.Nodes {
			seedMeta, hasSeedMeta := explicitByID[id]
			path := client.NormalizeLocationPath(strings.TrimSpace(n.Path))
			typ := strings.TrimSpace(n.Type)
			if path == "" && hasSeedMeta {
				path = seedMeta.Path
			}
			if typ == "" && hasSeedMeta {
				typ = seedMeta.Type
			}
			if path == "" || typ == "" {
				continue
			}
			location := client.BuildLocation(path, typ)
			dependencyClass := "transitive"
			if _, explicit := explicitByID[id]; explicit {
				dependencyClass = "explicit"
			}
			assetsByLocation[location] = ResolvedSeedAsset{
				ID:         id,
				Path:       path,
				Type:       typ,
				Location:   location,
				Dependency: dependencyClass,
			}
		}
	}

	// Ensure explicit seeds are present even when traversal node metadata is incomplete.
	for id, meta := range explicitByID {
		if meta.Path == "" || meta.Type == "" {
			continue
		}
		location := client.BuildLocation(meta.Path, meta.Type)
		assetsByLocation[location] = ResolvedSeedAsset{
			ID:         id,
			Path:       meta.Path,
			Type:       meta.Type,
			Location:   location,
			Dependency: "explicit",
		}
	}

	assets := make([]ResolvedSeedAsset, 0, len(assetsByLocation))
	for _, a := range assetsByLocation {
		assets = append(assets, a)
		if a.Dependency == "explicit" {
			stats.ExplicitAssets++
		} else {
			stats.TransitiveAssets++
		}
	}
	sort.Slice(assets, func(i, j int) bool {
		if assets[i].Type != assets[j].Type {
			return assets[i].Type < assets[j].Type
		}
		return assets[i].Path < assets[j].Path
	})
	stats.ResolvedAssets = len(assets)
	return assets, stats, nil
}

func normalizeSeedEntries(entries []client.ArtifactEntry) []client.ArtifactEntry {
	out := make([]client.ArtifactEntry, len(entries))
	for i, entry := range entries {
		out[i] = client.ArtifactEntry{
			ID:   strings.TrimSpace(entry.ID),
			Path: client.NormalizeLocationPath(strings.TrimSpace(entry.Path)),
			Type: strings.TrimSpace(entry.Type),
		}
	}
	return out
}

func resolveExplicitSeedObjects(
	ctx context.Context,
	c *client.Client,
	entries []client.ArtifactEntry,
) (map[string]resolvedSeedObject, map[string]bool, error) {
	enriched := make([]client.ArtifactEntry, len(entries))
	copy(enriched, entries)

	pathLookup := make([]client.LookupObject, 0, len(entries))
	for i, entry := range enriched {
		if entry.ID != "" {
			continue
		}
		if entry.Path == "" {
			return nil, nil, fmt.Errorf("entry %d has no id or path", i+1)
		}
		if entry.Type == "" {
			return nil, nil, fmt.Errorf("entry %d path %q requires TYPE when ID is not provided", i+1, entry.Path)
		}
		pathLookup = append(pathLookup, client.LookupObject{
			Path: entry.Path,
			Type: entry.Type,
		})
	}

	if len(pathLookup) > 0 {
		resp, err := c.Lookup(ctx, pathLookup)
		if err != nil {
			return nil, nil, fmt.Errorf("resolving seed IDs by path+type: %w", err)
		}
		enriched = client.ReconcileArtifactEntriesWithLookup(enriched, resp.Objects)
	}

	idLookup := make([]client.LookupObject, 0, len(enriched))
	for i, entry := range enriched {
		if strings.TrimSpace(entry.ID) == "" {
			return nil, nil, fmt.Errorf("could not resolve ID for entry %d (path=%s, type=%s)", i+1, entry.Path, entry.Type)
		}
		idLookup = append(idLookup, client.LookupObject{ID: entry.ID})
	}
	metaByID := make(map[string]resolvedSeedObject, len(enriched))
	if len(idLookup) > 0 {
		resp, err := c.Lookup(ctx, idLookup)
		if err != nil {
			return nil, nil, fmt.Errorf("resolving seed object metadata: %w", err)
		}
		for _, obj := range resp.Objects {
			id := strings.TrimSpace(obj.ID)
			path := client.NormalizeLocationPath(strings.TrimSpace(obj.Path))
			typ := strings.TrimSpace(obj.Type)
			if id == "" || path == "" || typ == "" {
				continue
			}
			metaByID[id] = resolvedSeedObject{
				ID:   id,
				Path: path,
				Type: typ,
			}
		}
	}

	explicitByID := make(map[string]resolvedSeedObject, len(enriched))
	containerRoots := make(map[string]bool)
	for _, entry := range enriched {
		id := strings.TrimSpace(entry.ID)
		if id == "" {
			continue
		}
		meta := resolvedSeedObject{
			ID:   id,
			Path: entry.Path,
			Type: entry.Type,
		}
		if lookedUp, ok := metaByID[id]; ok {
			meta = lookedUp
		} else {
			meta.Path = client.NormalizeLocationPath(strings.TrimSpace(meta.Path))
			meta.Type = strings.TrimSpace(meta.Type)
		}
		explicitByID[id] = meta
		if (meta.Type == "Project" || meta.Type == "Folder") && meta.Path != "" {
			containerRoots[meta.Path] = true
		}
	}

	return explicitByID, containerRoots, nil
}

func expandContainerObjects(
	ctx context.Context,
	c *client.Client,
	explicitByID map[string]resolvedSeedObject,
	containerRoots map[string]bool,
) (int, error) {
	added := 0
	for root := range containerRoots {
		resp, err := c.ListAllObjects(ctx, client.ObjectsListOptions{
			Query: fmt.Sprintf("location=='%s'", root),
		}, nil)
		if err != nil {
			return 0, fmt.Errorf("listing objects for container %q expansion: %w", root, err)
		}
		for _, obj := range resp.Objects {
			id := strings.TrimSpace(obj.ID)
			path := client.NormalizeLocationPath(strings.TrimSpace(obj.Path))
			typ := strings.TrimSpace(obj.Type)
			if id == "" || path == "" || typ == "" {
				continue
			}
			if !pathWithinAnyContainer(path, map[string]bool{root: true}) {
				continue
			}
			if _, exists := explicitByID[id]; exists {
				continue
			}
			explicitByID[id] = resolvedSeedObject{
				ID:   id,
				Path: path,
				Type: typ,
			}
			added++
		}
	}
	return added, nil
}

func pathWithinAnyContainer(path string, roots map[string]bool) bool {
	for root := range roots {
		if path == root || strings.HasPrefix(path, root+"/") {
			return true
		}
	}
	return false
}
