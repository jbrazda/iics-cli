package release

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/jbrazda/iics-cli/internal/client"
)

// writeLookupNotFound writes the error the V3 Lookup API returns for a whole
// batch when any item in it is not found.
func writeLookupNotFound(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_, _ = w.Write([]byte(`{"error":{"code":"V3API_LookupError_012","message":"Object not found","requestId":"r1"}}`))
}

// lookupFake simulates the target org. Paths in missing do not exist; every
// other path exists. Connections named in missingConnections do not exist.
// When failStatus is set, /lookup answers with that status instead.
type lookupFake struct {
	missing            map[string]bool
	missingConnections map[string]bool
	failStatus         int
	lookupCalls        atomic.Int32
}

func (f *lookupFake) client(t *testing.T) *client.Client {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		resp := client.LoginResponse{
			Products: []client.Product{{Name: "Data Integration", BaseAPIURL: "http://" + r.Host}},
			UserInfo: client.UserInfo{SessionID: "test-session", ID: "u1"},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
	mux.HandleFunc("/public/core/v3/lookup", func(w http.ResponseWriter, r *http.Request) {
		f.lookupCalls.Add(1)
		if f.failStatus != 0 {
			w.WriteHeader(f.failStatus)
			_, _ = w.Write([]byte(`{"error":{"code":"INTERNAL","message":"boom"}}`))
			return
		}
		var req client.LookupRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		var resp client.LookupResponse
		for _, o := range req.Objects {
			if f.missing[o.Path] {
				writeLookupNotFound(w)
				return
			}
			resp.Objects = append(resp.Objects, client.LookupResult{ID: "id-" + o.Path, Path: o.Path, Type: o.Type})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
	mux.HandleFunc("/api/v2/connection/name/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/api/v2/connection/name/")
		w.Header().Set("Content-Type", "application/json")
		if f.missingConnections[name] {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"code":"NOT_FOUND","description":"not found"}`))
			return
		}
		_, _ = fmt.Fprintf(w, `{"@type":"connection","id":"c-%s","name":%q}`, name, name)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return client.NewClient(srv.URL+"/login", "user", "pass")
}

func makeAssets(n int, dependency string) []Asset {
	assets := make([]Asset, n)
	for i := range assets {
		path := fmt.Sprintf("Explore/Proj/M%03d", i)
		assets[i] = Asset{Location: path + ".DTEMPLATE", Path: path, Type: "DTEMPLATE", Dependency: dependency}
	}
	return assets
}

func statusCounts(v map[string]AssetValidation) map[string]int {
	counts := map[string]int{}
	for _, val := range v {
		counts[val.Status]++
	}
	return counts
}

func TestValidateAssetsAllExist(t *testing.T) {
	f := &lookupFake{}
	assets := makeAssets(50, "explicit")
	got := validateAssetsExistInTarget(context.Background(), f.client(t), "QA", assets)
	if c := statusCounts(got); c["found"] != 50 {
		t.Fatalf("counts = %v, want 50 found", c)
	}
	if n := f.lookupCalls.Load(); n != 1 {
		t.Errorf("lookupCalls = %d, want 1", n)
	}
}

// Regression: one missing asset made the whole batch fail with
// V3API_LookupError_012, and every asset was reported missing.
func TestValidateAssetsOneMissingAmongFifty(t *testing.T) {
	assets := makeAssets(50, "transitive")
	missing := assets[37]
	f := &lookupFake{missing: map[string]bool{missing.Path: true}}
	got := validateAssetsExistInTarget(context.Background(), f.client(t), "QA", assets)

	for _, a := range assets {
		want := "found"
		if a.Location == missing.Location {
			want = "missing"
		}
		if got[a.Location].Status != want {
			t.Errorf("%s status = %q, want %q", a.Location, got[a.Location].Status, want)
		}
	}
	// Bisection: 1 call per level on the failing side plus its sibling.
	if n := f.lookupCalls.Load(); n > 13 {
		t.Errorf("lookupCalls = %d, want <= 13", n)
	}
}

func TestValidateAssetsAllMissing(t *testing.T) {
	assets := makeAssets(20, "explicit")
	missing := map[string]bool{}
	for _, a := range assets {
		missing[a.Path] = true
	}
	f := &lookupFake{missing: missing}
	got := validateAssetsExistInTarget(context.Background(), f.client(t), "QA", assets)
	if c := statusCounts(got); c["missing"] != 20 {
		t.Fatalf("counts = %v, want 20 missing", c)
	}
}

func TestValidateAssetsMixedTypesWithConnection(t *testing.T) {
	assets := []Asset{
		{Location: "Explore/P/BusinessService_InfaLogin.BSERVICE", Path: "Explore/P/BusinessService_InfaLogin", Type: "BSERVICE"},
		{Location: "Explore/P/dic_cities.DICTIONARY", Path: "Explore/P/dic_cities", Type: "DICTIONARY"},
		{Location: "Explore/P/DataProfiling.Folder", Path: "Explore/P/DataProfiling", Type: "Folder"},
		{Location: "Explore/P/m_load.DTEMPLATE", Path: "Explore/P/m_load", Type: "DTEMPLATE"},
		{Location: "SYS/Conn_OK.Connection", Path: "SYS/Conn_OK", Type: "Connection"},
		{Location: "SYS/Conn_Gone.Connection", Path: "SYS/Conn_Gone", Type: "Connection"},
	}
	f := &lookupFake{
		missing:            map[string]bool{"Explore/P/DataProfiling": true},
		missingConnections: map[string]bool{"Conn_Gone": true},
	}
	got := validateAssetsExistInTarget(context.Background(), f.client(t), "TST", assets)
	want := map[string]string{
		"Explore/P/BusinessService_InfaLogin.BSERVICE": "found",
		"Explore/P/dic_cities.DICTIONARY":              "found",
		"Explore/P/DataProfiling.Folder":               "missing",
		"Explore/P/m_load.DTEMPLATE":                   "found",
		"SYS/Conn_OK.Connection":                       "found",
		"SYS/Conn_Gone.Connection":                     "missing",
	}
	for loc, w := range want {
		if got[loc].Status != w {
			t.Errorf("%s status = %q, want %q", loc, got[loc].Status, w)
		}
	}
}

func TestValidateAssetsServerErrorIsUnknown(t *testing.T) {
	f := &lookupFake{failStatus: http.StatusInternalServerError}
	assets := makeAssets(5, "explicit")
	got := validateAssetsExistInTarget(context.Background(), f.client(t), "QA", assets)
	for _, a := range assets {
		v := got[a.Location]
		if v.Status != "unknown" || v.Warning == "" {
			t.Errorf("%s = %+v, want unknown with warning", a.Location, v)
		}
	}
}

func TestValidateAssetsChunksLargePlans(t *testing.T) {
	f := &lookupFake{}
	assets := makeAssets(lookupChunkSize*2+5, "explicit")
	got := validateAssetsExistInTarget(context.Background(), f.client(t), "QA", assets)
	if c := statusCounts(got); c["found"] != len(assets) {
		t.Fatalf("counts = %v, want %d found", c, len(assets))
	}
	if n := f.lookupCalls.Load(); n != 3 {
		t.Errorf("lookupCalls = %d, want 3 (one per chunk)", n)
	}
}

func TestFilterMissingTransitiveKeepsOnlyMissing(t *testing.T) {
	explicit := makeAssets(1, "explicit")
	explicit[0].Path, explicit[0].Location = "Explore/Proj/Root", "Explore/Proj/Root.TASKFLOW"
	transitive := makeAssets(50, "transitive")
	missing := transitive[12]
	assets := append(explicit, transitive...)

	f := &lookupFake{missing: map[string]bool{missing.Path: true}}
	validations := validateAssetsExistInTarget(context.Background(), f.client(t), "QA", assets)
	kept := FilterMissingTransitiveWithValidations(assets, validations)

	if len(kept) != 2 {
		t.Fatalf("kept %d assets, want 2 (explicit + missing transitive): %#v", len(kept), kept)
	}
	locs := map[string]bool{kept[0].Location: true, kept[1].Location: true}
	if !locs[explicit[0].Location] || !locs[missing.Location] {
		t.Errorf("kept = %v, want %s and %s", locs, explicit[0].Location, missing.Location)
	}
}
