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
// batch when any item in it is not found. Like the real API, the message
// lists every missing object and reports DTEMPLATE as MAPPING.
func writeLookupNotFound(w http.ResponseWriter, missing []client.LookupObject) {
	items := make([]string, len(missing))
	for i, o := range missing {
		typ := o.Type
		if typ == "DTEMPLATE" {
			typ = "MAPPING"
		}
		items[i] = fmt.Sprintf("[type: %s, location: %s]", typ, client.NormalizeLocationPath(o.Path))
	}
	msg := "Cannot find the following objects: [" + strings.Join(items, ", ") + "]"
	body, _ := json.Marshal(map[string]any{"error": map[string]any{
		"code": "V3API_LookupError_012", "message": msg, "debugMessage": msg, "requestId": "r1",
	}})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_, _ = w.Write(body)
}

// lookupFake simulates the target org. Paths in missing do not exist; every
// other path exists. Connections named in missingConnections do not exist.
// When failStatus is set, /lookup answers with that status instead. When
// noMissingList is set, the 012 message does not list the missing objects.
type lookupFake struct {
	missing            map[string]bool
	noMissingList      bool
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
			// Retry-After: 0 keeps the client's 429 retries instant.
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(f.failStatus)
			_, _ = w.Write([]byte(`{"error":{"code":"INTERNAL","message":"boom"}}`))
			return
		}
		var req client.LookupRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		var resp client.LookupResponse
		var notFound []client.LookupObject
		for _, o := range req.Objects {
			if f.missing[o.Path+"."+o.Type] || f.missing[o.Path] {
				notFound = append(notFound, o)
				continue
			}
			resp.Objects = append(resp.Objects, client.LookupResult{ID: "id-" + o.Path, Path: o.Path, Type: o.Type})
		}
		if len(notFound) > 0 {
			if f.noMissingList {
				notFound = nil
			}
			writeLookupNotFound(w, notFound)
			return
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
	// The 012 message names the missing asset; the rest is looked up again.
	if n := f.lookupCalls.Load(); n != 2 {
		t.Errorf("lookupCalls = %d, want 2", n)
	}
}

// Without a usable missing-object list in the 012 message, the batch is
// bisected until the missing asset is isolated.
func TestValidateAssetsBisectsWithoutMissingList(t *testing.T) {
	assets := makeAssets(50, "transitive")
	missing := assets[37]
	f := &lookupFake{missing: map[string]bool{missing.Path: true}, noMissingList: true}
	got := validateAssetsExistInTarget(context.Background(), f.client(t), "QA", assets)
	if c := statusCounts(got); c["found"] != 49 || c["missing"] != 1 {
		t.Fatalf("counts = %v, want 49 found, 1 missing", c)
	}
	if got[missing.Location].Status != "missing" {
		t.Errorf("%s status = %q, want missing", missing.Location, got[missing.Location].Status)
	}
	if n := f.lookupCalls.Load(); n > 13 {
		t.Errorf("lookupCalls = %d, want <= 13", n)
	}
}

// Two assets with the same path but different types cannot be told apart by
// the 012 message, so they are resolved by bisection instead.
func TestValidateAssetsSamePathDifferentTypes(t *testing.T) {
	assets := []Asset{
		{Location: "Explore/P/Load.DTEMPLATE", Path: "Explore/P/Load", Type: "DTEMPLATE"},
		{Location: "Explore/P/Load.TASKFLOW", Path: "Explore/P/Load", Type: "TASKFLOW"},
		{Location: "Explore/P/Other.PROCESS", Path: "Explore/P/Other", Type: "PROCESS"},
	}
	f := &lookupFake{missing: map[string]bool{"Explore/P/Load.TASKFLOW": true}}
	got := validateAssetsExistInTarget(context.Background(), f.client(t), "QA", assets)
	want := map[string]string{
		"Explore/P/Load.DTEMPLATE": "found",
		"Explore/P/Load.TASKFLOW":  "missing",
		"Explore/P/Other.PROCESS":  "found",
	}
	for loc, w := range want {
		if got[loc].Status != w {
			t.Errorf("%s status = %q, want %q", loc, got[loc].Status, w)
		}
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

func TestValidateAssetsRateLimitedIsMissing(t *testing.T) {
	f := &lookupFake{failStatus: http.StatusTooManyRequests}
	assets := append(makeAssets(3, "transitive"), Asset{
		Location: "Explore/Proj/Root.TASKFLOW", Path: "Explore/Proj/Root", Type: "TASKFLOW", Dependency: "explicit",
	})
	got := validateAssetsExistInTarget(context.Background(), f.client(t), "QA", assets)
	for _, a := range assets {
		v := got[a.Location]
		if v.Status != "missing" || !strings.HasPrefix(v.Warning, "APIRateLimitError") {
			t.Errorf("%s = %+v, want missing with APIRateLimitError warning", a.Location, v)
		}
	}
	// Rate-limited transitive assets stay in the package.
	if kept := FilterMissingTransitiveWithValidations(assets, got); len(kept) != len(assets) {
		t.Errorf("kept %d assets, want %d", len(kept), len(assets))
	}
}

func TestValidateConnectionRateLimitedIsMissing(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		resp := client.LoginResponse{
			Products: []client.Product{{Name: "Data Integration", BaseAPIURL: "http://" + r.Host}},
			UserInfo: client.UserInfo{SessionID: "s", ID: "u1"},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})
	mux.HandleFunc("/api/v2/connection/name/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	tc := client.NewClient(srv.URL+"/login", "user", "pass")

	a := Asset{Location: "SYS/Conn.Connection", Path: "SYS/Conn", Type: "Connection"}
	v := validateAssetsExistInTarget(context.Background(), tc, "QA", []Asset{a})[a.Location]
	if v.Status != "missing" || !strings.HasPrefix(v.Warning, "APIRateLimitError") {
		t.Errorf("connection = %+v, want missing with APIRateLimitError warning", v)
	}
}
