package dependencies

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/jbrazda/iics-cli/internal/client"
)

func TestTraverseByIDs_ResolvesTransitiveGraph(t *testing.T) {
	// Graph:
	// A -> B (ID)
	// B -> C (path+type lookup)
	// C -> D (ID)
	refsByID := map[string][]client.ObjectReference{
		"A": {
			{ID: "B", AppContextID: "B", Path: "Proj/B", Type: "PROCESS"},
		},
		"B": {
			{Path: "Proj/C", Type: "AI_CONNECTION"},
		},
		"C": {
			{ID: "D", AppContextID: "D", Path: "Proj/D", Type: "AI_SERVICE_CONNECTOR"},
		},
		"D": {},
	}

	lookupByID := map[string]client.LookupResult{
		"A": {ID: "A", Path: "Proj/A", Type: "PROCESS"},
		"B": {ID: "B", Path: "Proj/B", Type: "PROCESS"},
		"C": {ID: "C", Path: "Proj/C", Type: "AI_CONNECTION"},
		"D": {ID: "D", Path: "Proj/D", Type: "AI_SERVICE_CONNECTOR"},
	}
	lookupByPathType := map[string]client.LookupResult{
		"Proj/C.AI_CONNECTION": lookupByID["C"],
	}

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(client.LoginResponse{
				Products: []client.Product{{Name: "Data Integration", BaseAPIURL: srv.URL}},
				UserInfo: client.UserInfo{SessionID: "sid"},
			})
			return

		case "/public/core/v3/lookup":
			var req client.LookupRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode lookup request: %v", err)
			}
			out := client.LookupResponse{}
			for _, obj := range req.Objects {
				if obj.ID != "" {
					if v, ok := lookupByID[obj.ID]; ok {
						out.Objects = append(out.Objects, v)
					}
					continue
				}
				key := obj.Path + "." + obj.Type
				if v, ok := lookupByPathType[key]; ok {
					out.Objects = append(out.Objects, v)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(out)
			return

		case "/public/core/v3/objects/A/references":
			_ = json.NewEncoder(w).Encode(client.ObjectDependenciesResponse{
				ID:         "A",
				Count:      len(refsByID["A"]),
				References: refsByID["A"],
			})
			return
		case "/public/core/v3/objects/B/references":
			_ = json.NewEncoder(w).Encode(client.ObjectDependenciesResponse{
				ID:         "B",
				Count:      len(refsByID["B"]),
				References: refsByID["B"],
			})
			return
		case "/public/core/v3/objects/C/references":
			_ = json.NewEncoder(w).Encode(client.ObjectDependenciesResponse{
				ID:         "C",
				Count:      len(refsByID["C"]),
				References: refsByID["C"],
			})
			return
		case "/public/core/v3/objects/D/references":
			_ = json.NewEncoder(w).Encode(client.ObjectDependenciesResponse{
				ID:         "D",
				Count:      0,
				References: refsByID["D"],
			})
			return
		}

		http.NotFound(w, r)
	}))
	defer srv.Close()

	c := client.NewClient(srv.URL+"/login", "user", "pass")
	graph, err := TraverseByIDs(context.Background(), c, []string{"A"}, "uses", 0, 0)
	if err != nil {
		t.Fatalf("TraverseByIDs() error: %v", err)
	}

	if len(graph.Nodes) != 4 {
		t.Fatalf("expected 4 nodes, got %d", len(graph.Nodes))
	}
	for _, id := range []string{"A", "B", "C", "D"} {
		if _, ok := graph.Nodes[id]; !ok {
			t.Fatalf("missing node %s", id)
		}
	}

	edges := make([]string, 0, len(graph.Edges))
	for _, e := range graph.Edges {
		edges = append(edges, e.FromID+"->"+e.ToID)
	}
	sort.Strings(edges)
	want := []string{"A->B", "B->C", "C->D"}
	if len(edges) != len(want) {
		t.Fatalf("edges count=%d want=%d edges=%v", len(edges), len(want), edges)
	}
	for i := range want {
		if edges[i] != want[i] {
			t.Fatalf("edge[%d]=%s want %s (all=%v)", i, edges[i], want[i], edges)
		}
	}
}
