package release

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/jbrazda/iics-cli/internal/client"
)

// newBuildPlanTestServer returns an httptest server that fakes v3 login and
// lookup responses for BuildPlan's target-resolution calls (FilterMissingTransitiveForTarget,
// AnnotateAssetsWithTargetValidation). An asset is reported as already existing in the
// target when its path contains "Existing"; every other asset is reported missing.
// The returned counter tracks how many /lookup requests were made, so tests
// can assert assets are validated in as few batched calls as possible.
func newBuildPlanTestServer(t *testing.T) (*httptest.Server, *int) {
	t.Helper()
	lookupCalls := 0
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
		lookupCalls++
		var req client.LookupRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		var resp client.LookupResponse
		for _, o := range req.Objects {
			if strings.Contains(o.Path, "Existing") {
				resp.Objects = append(resp.Objects, client.LookupResult{ID: "id-" + o.Path, Path: o.Path, Type: o.Type})
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, &lookupCalls
}

// setEnvTargetCredentials points target-profile resolution at the given
// test server via the CI env-var fallback (IICS_USER_<TARGET>/IICS_PWD_<TARGET>/
// IICS_LOGIN_URL_<TARGET>), isolated from any real ~/.iics/config.yaml via HOME.
func setEnvTargetCredentials(t *testing.T, target, loginURL string) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	key := strings.ToUpper(target)
	t.Setenv("IICS_USER_"+key, "user")
	t.Setenv("IICS_PWD_"+key, "pass")
	t.Setenv("IICS_LOGIN_URL_"+key, loginURL)
}

func readCSVRows(t *testing.T, path string) [][]string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	rows := make([][]string, len(lines))
	for i, l := range lines {
		rows[i] = strings.Split(l, ",")
	}
	return rows
}

func TestBuildPlanFullMode(t *testing.T) {
	srv, lookupCalls := newBuildPlanTestServer(t)
	setEnvTargetCredentials(t, "QA", srv.URL+"/login")

	assets := []Asset{
		{Location: "Explore/A.PROCESS", Path: "Explore/A", Type: "PROCESS", Dependency: "explicit"},
		{Location: "Explore/Missing.GUIDE", Path: "Explore/Missing", Type: "GUIDE", Dependency: "transitive"},
		{Location: "Explore/Existing.TASKFLOW", Path: "Explore/Existing", Type: "TASKFLOW", Dependency: "transitive"},
	}

	outputRoot := t.TempDir()
	var generated []string
	result, err := BuildPlan(context.Background(), PlanOptions{
		Assets:                  assets,
		ConnectorSourceAssets:   assets,
		Targets:                 []string{"QA"},
		FilterMissingTransitive: true,
		OutputRoot:              outputRoot,
		PackageFileBaseName:     "full_build.package",
		PlanExt:                 "csv",
		WriteAssets:             WriteAssetsCSV,
		PackageFields:           []string{"location", "type", "path", "dependency"},
		PublishFields:           []string{"location", "type", "path", "dependency"},
		OnTargetFilesGenerated: func(env, packageFile, publishFile string, publishAssetCount int) {
			generated = append(generated, env, packageFile, publishFile)
			if publishAssetCount != 2 {
				t.Errorf("publishAssetCount = %d, want 2", publishAssetCount)
			}
		},
		CompletedLabel: "full mode",
	})
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}

	if result.FilesWritten != 2 {
		t.Fatalf("FilesWritten = %d, want 2", result.FilesWritten)
	}
	if result.ConnectorsFile != "" {
		t.Fatalf("ConnectorsFile = %q, want empty (no include-connectors requested)", result.ConnectorsFile)
	}
	if len(generated) != 3 {
		t.Fatalf("OnTargetFilesGenerated not invoked as expected: %#v", generated)
	}

	// Existing.TASKFLOW is transitive and already exists in QA, so it is
	// filtered out; only the explicit PROCESS and the still-missing transitive
	// GUIDE remain in the package/publish output.
	packageAssets := result.AssetsByTarget["QA"]
	if len(packageAssets) != 2 {
		t.Fatalf("AssetsByTarget[QA] len = %d, want 2: %#v", len(packageAssets), packageAssets)
	}
	byLocation := map[string]ManifestLogAsset{}
	for _, a := range packageAssets {
		byLocation[a.Location] = a
	}
	if got := byLocation["Explore/A.PROCESS"].Status; got != "missing" {
		t.Errorf("Explore/A.PROCESS status = %q, want missing", got)
	}
	if got := byLocation["Explore/Missing.GUIDE"].Status; got != "missing" {
		t.Errorf("Explore/Missing.GUIDE status = %q, want missing", got)
	}
	if _, ok := byLocation["Explore/Existing.TASKFLOW"]; ok {
		t.Fatalf("Explore/Existing.TASKFLOW should have been filtered out: %#v", packageAssets)
	}

	publishAssets := result.PublishByTarget["QA"]
	if len(publishAssets) != 2 {
		t.Fatalf("PublishByTarget[QA] len = %d, want 2: %#v", len(publishAssets), publishAssets)
	}

	packagePath := filepath.Join(outputRoot, "qa", "full_build.package.csv")
	rows := readCSVRows(t, packagePath)
	if len(rows) != 3 { // header + 2 rows
		t.Fatalf("package csv rows = %d, want 3: %#v", len(rows), rows)
	}
	if rows[0][0] != "LOCATION" {
		t.Fatalf("unexpected header: %#v", rows[0])
	}

	publishPath := filepath.Join(outputRoot, "qa", "publish_assets.csv")
	if _, statErr := os.Stat(publishPath); statErr != nil {
		t.Fatalf("expected publish file at %s: %v", publishPath, statErr)
	}

	// The missing-transitive filter and the package annotation each validate
	// existence for the same 3 assets; both must batch all their assets into
	// a single /lookup call rather than one call per asset.
	if *lookupCalls != 2 {
		t.Errorf("lookupCalls = %d, want 2 (one batched call per validation pass)", *lookupCalls)
	}
}

func TestBuildPlanSelectiveModeWithConnectors(t *testing.T) {
	srv, _ := newBuildPlanTestServer(t)
	setEnvTargetCredentials(t, "TST", srv.URL+"/login")

	allFiltered := []Asset{
		{Location: "Explore/A.PROCESS", Path: "Explore/A", Type: "PROCESS", Dependency: "explicit"},
	}
	connectorSourceAssets := []Asset{
		{Location: "Explore/A.PROCESS", Path: "Explore/A", Type: "PROCESS", Dependency: "explicit"},
		{Location: "Explore/Conn.AI_CONNECTION", Path: "Explore/Conn", Type: "AI_CONNECTION", Dependency: "explicit"},
	}

	outputRoot := t.TempDir()
	var processed []string
	result, err := BuildPlan(context.Background(), PlanOptions{
		Assets:                  allFiltered,
		ConnectorSourceAssets:   connectorSourceAssets,
		Targets:                 []string{"TST"},
		FilterMissingTransitive: false,
		OutputRoot:              outputRoot,
		PackageFileBaseName:     "tag_build.package",
		PlanExt:                 "csv",
		WriteAssets:             WriteAssetsCSV,
		PackageFields:           []string{"location", "type", "path", "dependency"},
		PublishFields:           []string{"location", "type", "path", "dependency"},
		IncludeConnectors:       true,
		OnTargetProcessing: func(env string) {
			processed = append(processed, env)
		},
		CompletedLabel: "selective mode",
	})
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}

	if len(processed) != 1 || processed[0] != "TST" {
		t.Fatalf("OnTargetProcessing calls = %#v, want [TST]", processed)
	}
	// 2 files per target + 1 connectors package file.
	if result.FilesWritten != 3 {
		t.Fatalf("FilesWritten = %d, want 3", result.FilesWritten)
	}
	wantConnectorsFile := filepath.Join(outputRoot, "connectors.package.csv")
	if result.ConnectorsFile != wantConnectorsFile {
		t.Fatalf("ConnectorsFile = %q, want %q", result.ConnectorsFile, wantConnectorsFile)
	}

	packagePath := filepath.Join(outputRoot, "tst", "tag_build.package.csv")
	if _, statErr := os.Stat(packagePath); statErr != nil {
		t.Fatalf("expected package file at %s: %v", packagePath, statErr)
	}

	connectorRows := readCSVRows(t, wantConnectorsFile)
	if len(connectorRows) != 2 { // header + 1 connector asset
		t.Fatalf("connector csv rows = %d, want 2: %#v", len(connectorRows), connectorRows)
	}
	if connectorRows[1][0] != "Explore/Conn.AI_CONNECTION" {
		t.Fatalf("unexpected connector row: %#v", connectorRows[1])
	}
}

func TestBuildPlanReusesPrecomputedValidations(t *testing.T) {
	// No profile or CI env credentials configured for "QA" at all: if
	// BuildPlan resolved a target client despite a precomputed validation
	// matrix being supplied, resolveTargetClient would fail immediately.
	// Succeeding here proves the missing-transitive filter and package
	// annotation both reused the matrix instead of validating over the
	// network.
	t.Setenv("HOME", t.TempDir())

	assets := []Asset{
		{Location: "Explore/A.PROCESS", Path: "Explore/A", Type: "PROCESS", Dependency: "explicit"},
		{Location: "Explore/Missing.GUIDE", Path: "Explore/Missing", Type: "GUIDE", Dependency: "transitive"},
		{Location: "Explore/Existing.TASKFLOW", Path: "Explore/Existing", Type: "TASKFLOW", Dependency: "transitive"},
	}
	validations := map[string]map[string]AssetValidation{
		"QA": {
			"Explore/A.PROCESS":         {Status: "missing"},
			"Explore/Missing.GUIDE":     {Status: "missing"},
			"Explore/Existing.TASKFLOW": {Status: "found"},
		},
	}

	outputRoot := t.TempDir()
	result, err := BuildPlan(context.Background(), PlanOptions{
		Assets:                  assets,
		ConnectorSourceAssets:   assets,
		Targets:                 []string{"QA"},
		FilterMissingTransitive: true,
		Validations:             validations,
		OutputRoot:              outputRoot,
		PackageFileBaseName:     "full_build.package",
		PlanExt:                 "csv",
		WriteAssets:             WriteAssetsCSV,
		PackageFields:           []string{"location", "type", "path", "dependency"},
		PublishFields:           []string{"location", "type", "path", "dependency"},
		CompletedLabel:          "full mode",
	})
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}

	// Existing.TASKFLOW is transitive and marked "found" in the precomputed
	// matrix, so it is filtered out; the explicit PROCESS and the still-
	// missing transitive GUIDE remain, stamped with the matrix's statuses.
	packageAssets := result.AssetsByTarget["QA"]
	if len(packageAssets) != 2 {
		t.Fatalf("AssetsByTarget[QA] len = %d, want 2: %#v", len(packageAssets), packageAssets)
	}
	byLocation := map[string]ManifestLogAsset{}
	for _, a := range packageAssets {
		byLocation[a.Location] = a
	}
	if got := byLocation["Explore/A.PROCESS"].Status; got != "missing" {
		t.Errorf("Explore/A.PROCESS status = %q, want missing", got)
	}
	if _, ok := byLocation["Explore/Existing.TASKFLOW"]; ok {
		t.Fatalf("Explore/Existing.TASKFLOW should have been filtered out via precomputed validations: %#v", packageAssets)
	}
}

func TestBuildPlanPublishIncludeFoundTransitive(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	assets := []Asset{
		{Location: "Explore/A.PROCESS", Path: "Explore/A", Type: "PROCESS", Dependency: "explicit"},
		{Location: "Explore/Missing.GUIDE", Path: "Explore/Missing", Type: "GUIDE", Dependency: "transitive"},
		{Location: "Explore/Existing.TASKFLOW", Path: "Explore/Existing", Type: "TASKFLOW", Dependency: "transitive"},
	}
	validations := map[string]map[string]AssetValidation{
		"QA": {
			"Explore/A.PROCESS":         {Status: "missing"},
			"Explore/Missing.GUIDE":     {Status: "missing"},
			"Explore/Existing.TASKFLOW": {Status: "found"},
		},
	}

	runPlan := func(includeFoundTransitive bool) PlanResult {
		outputRoot := t.TempDir()
		result, err := BuildPlan(context.Background(), PlanOptions{
			Assets:                        assets,
			ConnectorSourceAssets:         assets,
			Targets:                       []string{"QA"},
			FilterMissingTransitive:       true,
			Validations:                   validations,
			OutputRoot:                    outputRoot,
			PackageFileBaseName:           "full_build.package",
			PlanExt:                       "csv",
			WriteAssets:                   WriteAssetsCSV,
			PackageFields:                 []string{"location", "type", "path", "dependency"},
			PublishFields:                 []string{"location", "type", "path", "dependency"},
			PublishIncludeFoundTransitive: includeFoundTransitive,
			CompletedLabel:                "full mode",
		})
		if err != nil {
			t.Fatalf("BuildPlan() error = %v", err)
		}
		return result
	}

	withoutFlag := runPlan(false)
	packageAssets := withoutFlag.AssetsByTarget["QA"]
	if len(packageAssets) != 2 {
		t.Fatalf("AssetsByTarget[QA] len = %d, want 2: %#v", len(packageAssets), packageAssets)
	}
	publishAssets := withoutFlag.PublishByTarget["QA"]
	for _, a := range publishAssets {
		if a.Location == "Explore/Existing.TASKFLOW" {
			t.Fatalf("Existing.TASKFLOW should not be in publish output when PublishIncludeFoundTransitive is false: %#v", publishAssets)
		}
	}

	withFlag := runPlan(true)
	packageAssetsWithFlag := withFlag.AssetsByTarget["QA"]
	if len(packageAssetsWithFlag) != len(packageAssets) {
		t.Fatalf("AssetsByTarget[QA] changed when PublishIncludeFoundTransitive set: got %d, want %d", len(packageAssetsWithFlag), len(packageAssets))
	}
	publishAssetsWithFlag := withFlag.PublishByTarget["QA"]
	found := false
	for _, a := range publishAssetsWithFlag {
		if a.Location == "Explore/Existing.TASKFLOW" {
			found = true
		}
	}
	if !found {
		t.Fatalf("Existing.TASKFLOW should be in publish output when PublishIncludeFoundTransitive is true: %#v", publishAssetsWithFlag)
	}
}

func TestBuildPlanPublishExcludePatterns(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	assets := []Asset{
		{Location: "Explore/A.PROCESS", Path: "Explore/A", Type: "PROCESS", Dependency: "explicit"},
		{Location: "Explore/Secret.GUIDE", Path: "Explore/Secret", Type: "GUIDE", Dependency: "explicit"},
	}
	validations := map[string]map[string]AssetValidation{
		"QA": {
			"Explore/A.PROCESS":    {Status: "missing"},
			"Explore/Secret.GUIDE": {Status: "missing"},
		},
	}
	excludePattern := regexp.MustCompile(`Secret`)

	outputRoot := t.TempDir()
	result, err := BuildPlan(context.Background(), PlanOptions{
		Assets:                  assets,
		ConnectorSourceAssets:   assets,
		Targets:                 []string{"QA"},
		FilterMissingTransitive: true,
		Validations:             validations,
		OutputRoot:              outputRoot,
		PackageFileBaseName:     "full_build.package",
		PlanExt:                 "csv",
		WriteAssets:             WriteAssetsCSV,
		PackageFields:           []string{"location", "type", "path", "dependency"},
		PublishFields:           []string{"location", "type", "path", "dependency"},
		PublishExcludePatterns:  []*regexp.Regexp{excludePattern},
		CompletedLabel:          "full mode",
	})
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}

	packageAssets := result.AssetsByTarget["QA"]
	if len(packageAssets) != 2 {
		t.Fatalf("AssetsByTarget[QA] len = %d, want 2 (package unaffected by publish excludes): %#v", len(packageAssets), packageAssets)
	}

	publishAssets := result.PublishByTarget["QA"]
	if len(publishAssets) != 1 {
		t.Fatalf("PublishByTarget[QA] len = %d, want 1: %#v", len(publishAssets), publishAssets)
	}
	if publishAssets[0].Location != "Explore/A.PROCESS" {
		t.Fatalf("PublishByTarget[QA][0].Location = %q, want Explore/A.PROCESS", publishAssets[0].Location)
	}
}

func TestBuildPlanPublishExcludeAppliesWithIncludeFoundTransitive(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	assets := []Asset{
		{Location: "Explore/A.PROCESS", Path: "Explore/A", Type: "PROCESS", Dependency: "explicit"},
		{Location: "Explore/Existing.TASKFLOW", Path: "Explore/Existing", Type: "TASKFLOW", Dependency: "transitive"},
	}
	validations := map[string]map[string]AssetValidation{
		"QA": {
			"Explore/A.PROCESS":         {Status: "missing"},
			"Explore/Existing.TASKFLOW": {Status: "found"},
		},
	}
	excludePattern := regexp.MustCompile(`Existing`)

	outputRoot := t.TempDir()
	result, err := BuildPlan(context.Background(), PlanOptions{
		Assets:                        assets,
		ConnectorSourceAssets:         assets,
		Targets:                       []string{"QA"},
		FilterMissingTransitive:       true,
		Validations:                   validations,
		OutputRoot:                    outputRoot,
		PackageFileBaseName:           "full_build.package",
		PlanExt:                       "csv",
		WriteAssets:                   WriteAssetsCSV,
		PackageFields:                 []string{"location", "type", "path", "dependency"},
		PublishFields:                 []string{"location", "type", "path", "dependency"},
		PublishIncludeFoundTransitive: true,
		PublishExcludePatterns:        []*regexp.Regexp{excludePattern},
		CompletedLabel:                "full mode",
	})
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}

	publishAssets := result.PublishByTarget["QA"]
	for _, a := range publishAssets {
		if a.Location == "Explore/Existing.TASKFLOW" {
			t.Fatalf("Existing.TASKFLOW should still be excluded from publish despite PublishIncludeFoundTransitive: %#v", publishAssets)
		}
	}
}

func TestBuildPlanPropagatesTargetResolutionError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	// No IICS_USER_*/IICS_PWD_* env vars and no config profile for "UNKNOWNENV":
	// resolveTargetClient must fail, and BuildPlan must surface that error.
	_, err := BuildPlan(context.Background(), PlanOptions{
		Assets:              []Asset{{Location: "Explore/A.PROCESS", Path: "Explore/A", Type: "PROCESS", Dependency: "explicit"}},
		Targets:             []string{"UNKNOWNENV"},
		OutputRoot:          t.TempDir(),
		PackageFileBaseName: "tag_build.package",
		PlanExt:             "csv",
		WriteAssets:         WriteAssetsCSV,
		PackageFields:       []string{"location"},
		PublishFields:       []string{"location"},
	})
	if err == nil {
		t.Fatal("expected error for unresolvable target profile")
	}
}
