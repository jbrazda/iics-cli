package dependencies

import (
	"context"
	"fmt"

	"github.com/jbrazda/iics-cli/internal/client"
)

// Node represents one resolved IICS object in the dependency graph.
type Node struct {
	ID   string
	Path string
	Type string
}

// Edge is a directed dependency relation: FromID depends on ToID.
type Edge struct {
	FromID string
	ToID   string
}

// Graph is the resolved transitive dependency graph.
type Graph struct {
	Nodes map[string]Node
	Edges []Edge
}

// TraverseByIDs resolves full transitive dependencies for seed object IDs.
// When limit is 0, it auto-paginates with GetAllObjectDependencies.
func TraverseByIDs(
	ctx context.Context,
	c *client.Client,
	seedIDs []string,
	refType string,
	limit, skip int,
) (*Graph, error) {
	if c == nil {
		return nil, fmt.Errorf("client is required")
	}

	queue := make([]string, 0, len(seedIDs))
	for _, id := range seedIDs {
		if id != "" {
			queue = append(queue, id)
		}
	}

	visited := make(map[string]bool, len(queue))
	edgeSeen := make(map[[2]string]bool)
	nodes := make(map[string]Node, len(queue))
	edges := make([]Edge, 0)

	for len(queue) > 0 {
		objectID := queue[0]
		queue = queue[1:]
		if objectID == "" || visited[objectID] {
			continue
		}
		visited[objectID] = true

		var resp *client.ObjectDependenciesResponse
		var err error
		if limit == 0 {
			resp, err = c.GetAllObjectDependencies(ctx, objectID, refType)
		} else {
			resp, err = c.GetObjectDependencies(ctx, objectID, refType, limit, skip)
		}
		if err != nil {
			return nil, fmt.Errorf("fetching dependencies for %s: %w", objectID, err)
		}

		lookupObjs := make([]client.LookupObject, 0)
		lookupSeen := make(map[string]bool)

		for _, ref := range resp.References {
			childID := ref.ID
			if childID == "" {
				childID = ref.AppContextID
			}
			if childID != "" {
				pair := [2]string{objectID, childID}
				if !edgeSeen[pair] {
					edgeSeen[pair] = true
					edges = append(edges, Edge{FromID: objectID, ToID: childID})
				}
				if !visited[childID] {
					queue = append(queue, childID)
				}
				continue
			}

			if ref.Path == "" || ref.Type == "" {
				continue
			}
			lookupKey := ref.Path + "." + ref.Type
			if lookupSeen[lookupKey] {
				continue
			}
			lookupSeen[lookupKey] = true
			lookupObjs = append(lookupObjs, client.LookupObject{
				Path: ref.Path,
				Type: ref.Type,
			})
		}

		if len(lookupObjs) == 0 {
			continue
		}

		lookupResp, lookupErr := c.Lookup(ctx, lookupObjs)
		if lookupErr != nil {
			return nil, fmt.Errorf("resolving dependency IDs for %s: %w", objectID, lookupErr)
		}
		for _, obj := range lookupResp.Objects {
			if obj.ID == "" {
				continue
			}
			pair := [2]string{objectID, obj.ID}
			if !edgeSeen[pair] {
				edgeSeen[pair] = true
				edges = append(edges, Edge{FromID: objectID, ToID: obj.ID})
			}
			if !visited[obj.ID] {
				queue = append(queue, obj.ID)
			}
		}
	}

	allIDs := make([]string, 0, len(visited))
	for id := range visited {
		allIDs = append(allIDs, id)
	}
	if len(allIDs) == 0 {
		return &Graph{Nodes: nodes, Edges: edges}, nil
	}

	lookupByID := make([]client.LookupObject, len(allIDs))
	for i, id := range allIDs {
		lookupByID[i] = client.LookupObject{ID: id}
	}
	lookupResp, err := c.Lookup(ctx, lookupByID)
	if err != nil {
		return nil, fmt.Errorf("resolving visited nodes: %w", err)
	}
	for _, obj := range lookupResp.Objects {
		if obj.ID == "" {
			continue
		}
		nodes[obj.ID] = Node{
			ID:   obj.ID,
			Path: obj.Path,
			Type: obj.Type,
		}
	}

	for id := range visited {
		if _, ok := nodes[id]; !ok {
			nodes[id] = Node{ID: id}
		}
	}

	return &Graph{
		Nodes: nodes,
		Edges: edges,
	}, nil
}
