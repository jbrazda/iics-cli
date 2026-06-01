package dependencies

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/jbrazda/iics-cli/internal/client"
)

func TestResolveSeedAssets_ExpandsContainerAndClassifiesDependencies(t *testing.T) {
	lookupByID := map[string]client.LookupResult{
		"folder": {ID: "folder", Path: "ZZ_TEST_CLI/Processes", Type: "Folder"},
		"nested": {ID: "nested", Path: "ZZ_TEST_CLI/Processes/Nested", Type: "Folder"},
		"proc1":  {ID: "proc1", Path: "ZZ_TEST_CLI/Processes/P1", Type: "PROCESS"},
		"proc2":  {ID: "proc2", Path: "ZZ_TEST_CLI/Processes/Nested/P2", Type: "PROCESS"},
		"conn":   {ID: "conn", Path: "Shared/ConnA", Type: "AI_CONNECTION"},
	}
	lookupByPathType := map[string]client.LookupResult{
		"ZZ_TEST_CLI/Processes.Folder": lookupByID["folder"],
	}
	refsByID := map[string][]client.ObjectReference{
		"folder": {},
		"nested": {},
		"proc1": {
			{ID: "conn", AppContextID: "conn", Path: "Shared/ConnA", Type: "AI_CONNECTION"},
		},
		"proc2": {},
		"conn":  {},
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
		case "/public/core/v3/objects":
			skip, _ := strconv.Atoi(r.URL.Query().Get("skip"))
			limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
			if limit == 0 {
				limit = 200
			}
			location := queryLocation(t, r.URL.Query().Get("q"))
			objectsByLocation := map[string][]client.Object{
				"ZZ_TEST_CLI/Processes": {
					{ID: "nested", Path: "ZZ_TEST_CLI/Processes/Nested", Type: "Folder"},
					{ID: "proc1", Path: "ZZ_TEST_CLI/Processes/P1", Type: "PROCESS"},
				},
				"ZZ_TEST_CLI/Processes/Nested": {
					{ID: "proc2", Path: "ZZ_TEST_CLI/Processes/Nested/P2", Type: "PROCESS"},
				},
			}
			objects := objectsByLocation[location]
			var page []client.Object
			if skip < len(objects) {
				end := skip + limit
				if end > len(objects) {
					end = len(objects)
				}
				page = objects[skip:end]
			} else {
				page = []client.Object{}
			}
			_ = json.NewEncoder(w).Encode(client.ObjectsListResponse{
				Count:   len(page),
				Objects: page,
			})
			return
		}

		if strings.HasPrefix(r.URL.Path, "/public/core/v3/objects/") && strings.HasSuffix(r.URL.Path, "/references") {
			id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/public/core/v3/objects/"), "/references")
			refs := refsByID[id]
			_ = json.NewEncoder(w).Encode(client.ObjectDependenciesResponse{
				ID:         id,
				Count:      len(refs),
				References: refs,
			})
			return
		}

		http.NotFound(w, r)
	}))
	defer srv.Close()

	c := client.NewClient(srv.URL+"/login", "user", "pass")
	entries := []client.ArtifactEntry{{Path: "ZZ_TEST_CLI/Processes", Type: "Folder"}}
	assets, stats, err := ResolveSeedAssets(context.Background(), c, entries, "uses", 0, 0)
	if err != nil {
		t.Fatalf("ResolveSeedAssets() error: %v", err)
	}

	if stats.ContainerRoots != 1 {
		t.Fatalf("container roots = %d, want 1", stats.ContainerRoots)
	}
	if stats.ExpandedExplicitObjects != 3 {
		t.Fatalf("expanded explicit objects = %d, want 3", stats.ExpandedExplicitObjects)
	}

	byLocation := make(map[string]ResolvedSeedAsset, len(assets))
	for _, a := range assets {
		byLocation[a.Location] = a
	}

	for _, location := range []string{
		"Explore/ZZ_TEST_CLI/Processes.Folder",
		"Explore/ZZ_TEST_CLI/Processes/Nested.Folder",
		"Explore/ZZ_TEST_CLI/Processes/P1.PROCESS",
		"Explore/ZZ_TEST_CLI/Processes/Nested/P2.PROCESS",
	} {
		row, ok := byLocation[location]
		if !ok {
			t.Fatalf("expected explicit location %s", location)
		}
		if row.Dependency != "explicit" {
			t.Fatalf("%s dependency = %s, want explicit", location, row.Dependency)
		}
	}
	transitiveLocation := "Explore/Shared/ConnA.AI_CONNECTION"
	if byLocation[transitiveLocation].Dependency != "transitive" {
		t.Fatalf("%s dependency = %s, want transitive", transitiveLocation, byLocation[transitiveLocation].Dependency)
	}
}

func queryLocation(t *testing.T, rawQuery string) string {
	t.Helper()
	const prefix = "location=='"
	const suffix = "'"
	if !strings.HasPrefix(rawQuery, prefix) || !strings.HasSuffix(rawQuery, suffix) {
		t.Fatalf("unexpected object query: %q", rawQuery)
	}
	return strings.TrimSuffix(strings.TrimPrefix(rawQuery, prefix), suffix)
}
