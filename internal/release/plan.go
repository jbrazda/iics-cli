package release

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/config"
	"github.com/jbrazda/iics-cli/internal/dependencies"
	"gopkg.in/yaml.v3"
)

type Asset struct {
	ID         string
	Path       string
	Type       string
	Location   string
	Dependency string
	Status     string
	Warning    string
}

var publishableTypes = map[string]bool{
	"AI_SERVICE_CONNECTOR": true,
	"AI_CONNECTION":        true,
	"PROCESS":              true,
	"GUIDE":                true,
	"TASKFLOW":             true,
}

var publishTypeRank = map[string]int{
	"AI_SERVICE_CONNECTOR": 0,
	"AI_CONNECTION":        1,
	"PROCESS":              2,
	"GUIDE":                3,
	"TASKFLOW":             4,
}

var connectorAssetTypes = map[string]bool{
	"AI_SERVICE_CONNECTOR": true,
}

var connectionAssetTypes = map[string]bool{
	"AI_CONNECTION": true,
	"Connection":    true,
}

const targetProfileMapEnv = "IICS_TARGET_PROFILE_MAP"

type TargetResolutionOptions struct {
	TargetProfileMap string
	Verbose          bool
	Debug            bool
}

type AssetValidation struct {
	Status  string
	Warning string
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
		path := client.NormalizeLocationPath(n.Path)
		typ := n.Type
		if path == "" || typ == "" {
			if so, ok := seedObj[id]; ok {
				path = client.NormalizeLocationPath(so.Path)
				typ = so.Type
			}
		}
		if path == "" || typ == "" {
			continue
		}
		loc := client.BuildLocation(path, typ)
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
		path := client.NormalizeLocationPath(o.Path)
		loc := client.BuildLocation(path, o.Type)
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

func ApplyPolicies(assets []Asset, includeConnectors, includeConnections bool, excludePatterns []*regexp.Regexp) []Asset {
	assets = ExcludeAssetsByLocation(assets, excludePatterns)
	filtered := make([]Asset, 0, len(assets))
	for _, a := range assets {
		if !includeConnectors && isConnectorType(a.Type) {
			continue
		} else if !includeConnections && isConnectionType(a.Type) {
			continue
		}
		filtered = append(filtered, a)
	}
	return filtered
}

// ExcludeAssetsByLocation drops assets whose Location matches any of patterns.
func ExcludeAssetsByLocation(assets []Asset, patterns []*regexp.Regexp) []Asset {
	if len(patterns) == 0 {
		return assets
	}
	out := make([]Asset, 0, len(assets))
	for _, a := range assets {
		excluded := false
		for _, re := range patterns {
			if re.MatchString(a.Location) {
				excluded = true
				break
			}
		}
		if !excluded {
			out = append(out, a)
		}
	}
	return out
}

func ConnectorAssets(assets []Asset) []Asset {
	out := make([]Asset, 0)
	for _, a := range assets {
		if isConnectorOrConnectionType(a.Type) {
			out = append(out, a)
		}
	}
	return out
}

func ShouldWriteConnectorPackage(includeConnectors, includeConnections bool) bool {
	return includeConnectors || includeConnections
}

func ConnectorPackageAssets(assets []Asset) []Asset {
	out := ConnectorAssets(assets)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Type != out[j].Type {
			return out[i].Type < out[j].Type
		}
		return out[i].Path < out[j].Path
	})
	return out
}

func isConnectorType(assetType string) bool {
	return connectorAssetTypes[assetType]
}

func isConnectionType(assetType string) bool {
	return connectionAssetTypes[assetType]
}

func isConnectorOrConnectionType(assetType string) bool {
	return isConnectorType(assetType) || isConnectionType(assetType)
}

func PublishAssets(assets []Asset) []Asset {
	out := make([]Asset, 0)
	for _, a := range assets {
		if publishableTypes[a.Type] {
			out = append(out, a)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return publishTypeRankForAsset(out[i].Type) < publishTypeRankForAsset(out[j].Type)
	})
	return out
}

func publishTypeRankForAsset(assetType string) int {
	if rank, ok := publishTypeRank[assetType]; ok {
		return rank
	}
	return 99
}

func FilterMissingTransitiveForTarget(ctx context.Context, targetProfileName string, assets []Asset, opts TargetResolutionOptions) ([]Asset, error) {
	tc, _, err := resolveTargetClient(targetProfileName, opts)
	if err != nil {
		return nil, err
	}
	validations := validateAssetsExistInTarget(ctx, tc, assets)
	return FilterMissingTransitiveWithValidations(assets, validations), nil
}

// FilterMissingTransitiveWithValidations applies the missing-transitive policy
// using a precomputed validation matrix (asset Location -> AssetValidation),
// avoiding a network round trip when the matrix was already computed
// elsewhere (e.g. by ValidateAssetsForTargets for the status table).
func FilterMissingTransitiveWithValidations(assets []Asset, validations map[string]AssetValidation) []Asset {
	missingByLocation := make(map[string]bool, len(assets))
	for _, a := range assets {
		if a.Dependency != "transitive" {
			continue
		}
		v, ok := validations[a.Location]
		missingByLocation[a.Location] = !ok || v.Status != "found"
	}
	return ApplyMissingTransitivePolicy(assets, missingByLocation)
}

func ValidateAssetsForTarget(ctx context.Context, targetProfileName string, assets []Asset, opts TargetResolutionOptions) ([]AssetValidation, error) {
	tc, _, err := resolveTargetClient(targetProfileName, opts)
	if err != nil {
		return nil, err
	}
	validations := validateAssetsExistInTarget(ctx, tc, assets)
	out := make([]AssetValidation, len(assets))
	for i, a := range assets {
		if v, ok := validations[a.Location]; ok {
			out[i] = v
		} else {
			out[i] = AssetValidation{Status: "unknown"}
		}
	}
	return out, nil
}

// ValidateAssetsForTargets computes an existence validation matrix for every
// (target, asset) pair, batching each target's non-Connection assets into a
// single Lookup call rather than one call per asset. The result (keyed by
// target, then by asset Location) can be reused across the status table
// render and BuildPlan's package/publish annotation instead of re-querying
// the same assets twice.
func ValidateAssetsForTargets(ctx context.Context, targets []string, assets []Asset, opts TargetResolutionOptions) (map[string]map[string]AssetValidation, error) {
	out := make(map[string]map[string]AssetValidation, len(targets))
	for _, target := range targets {
		tc, _, err := resolveTargetClient(target, opts)
		if err != nil {
			return nil, fmt.Errorf("profile %q: %w", target, err)
		}
		out[target] = validateAssetsExistInTarget(ctx, tc, assets)
	}
	return out, nil
}

// AnnotateAssetsWithTargetValidation returns a copy of assets with Status/Warning
// populated for a specific target environment.
func AnnotateAssetsWithTargetValidation(ctx context.Context, targetProfileName string, assets []Asset, opts TargetResolutionOptions) ([]Asset, error) {
	validations, err := ValidateAssetsForTarget(ctx, targetProfileName, assets, opts)
	if err != nil {
		return nil, err
	}
	out := make([]Asset, len(assets))
	copy(out, assets)
	for i := range out {
		if i < len(validations) {
			out[i].Status = validations[i].Status
			out[i].Warning = validations[i].Warning
		}
	}
	return out, nil
}

// AnnotateAssetsWithValidations returns a copy of assets with Status/Warning
// populated from a precomputed validation matrix (asset Location ->
// AssetValidation), avoiding a network round trip.
func AnnotateAssetsWithValidations(assets []Asset, validations map[string]AssetValidation) []Asset {
	out := make([]Asset, len(assets))
	copy(out, assets)
	for i := range out {
		if v, ok := validations[out[i].Location]; ok {
			out[i].Status = v.Status
			out[i].Warning = v.Warning
		}
	}
	return out
}

func statusFieldForTarget(target string) string {
	return fmt.Sprintf("status (%s)", strings.ToLower(strings.TrimSpace(target)))
}

// EnsureCurrentTargetStatusField ensures the target-specific status field exists in
// package fields when writing per-environment package files.
func EnsureCurrentTargetStatusField(fields []string, target string) []string {
	if strings.TrimSpace(target) == "" {
		return fields
	}
	targetStatusField := statusFieldForTarget(target)
	for _, field := range fields {
		if strings.EqualFold(strings.TrimSpace(field), targetStatusField) {
			return fields
		}
	}
	out := make([]string, 0, len(fields)+1)
	out = append(out, fields...)
	out = append(out, targetStatusField)
	return out
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

func resolveProfileNameForTarget(cfg *config.Config, target string, profileMap map[string]string) (string, bool, error) {
	target = strings.TrimSpace(strings.ToUpper(target))
	if target == "" {
		return "", false, fmt.Errorf("target profile is required")
	}
	if mapped, ok := mappedProfileForTarget(target, profileMap); ok {
		name, found := findProfileNameCaseInsensitive(cfg, mapped)
		if !found {
			return "", true, fmt.Errorf("target profile %q not found in config for target %q", mapped, target)
		}
		return name, true, nil
	}
	if name, found := findProfileNameCaseInsensitive(cfg, target); found {
		return name, false, nil
	}
	return "", false, nil
}

func resolveEffectiveTargetProfileMap(flagMap string) (map[string]string, error) {
	raw := strings.TrimSpace(flagMap)
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv(targetProfileMapEnv))
	}
	if raw == "" {
		return map[string]string{}, nil
	}
	return parseTargetProfileMap(raw)
}

func parseTargetProfileMap(raw string) (map[string]string, error) {
	out := make(map[string]string)
	parts := strings.Split(raw, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid target profile map entry %q; expected TARGET=profile", part)
		}
		target := strings.ToUpper(strings.TrimSpace(kv[0]))
		profile := strings.TrimSpace(kv[1])
		if target == "" || profile == "" {
			return nil, fmt.Errorf("invalid target profile map entry %q; expected TARGET=profile", part)
		}
		out[target] = profile
	}
	return out, nil
}

func mappedProfileForTarget(target string, profileMap map[string]string) (string, bool) {
	for _, key := range targetKeyCandidates(target) {
		if profile, ok := profileMap[key]; ok {
			return profile, true
		}
	}
	return "", false
}

func findProfileNameCaseInsensitive(cfg *config.Config, requested string) (string, bool) {
	requested = strings.TrimSpace(requested)
	if requested == "" {
		return "", false
	}
	if _, ok := cfg.Profiles[requested]; ok {
		return requested, true
	}
	lowerReq := strings.ToLower(requested)
	for name := range cfg.Profiles {
		if strings.ToLower(name) == lowerReq {
			return name, true
		}
	}
	return "", false
}

func targetKeyCandidates(target string) []string {
	upper := strings.ToUpper(strings.TrimSpace(target))
	if upper == "" {
		return nil
	}
	candidates := []string{upper}
	switch upper {
	case "STG":
		candidates = append(candidates, "STAGE")
	case "STAGE":
		candidates = append(candidates, "STG")
	}
	return candidates
}

func resolveEnvTargetProfile(target string) (*config.Profile, bool, error) {
	var partialKeys []string
	for _, key := range targetKeyCandidates(target) {
		userKey := "IICS_USER_" + key
		pwdKey := "IICS_PWD_" + key
		user := strings.TrimSpace(os.Getenv(userKey))
		pwd := strings.TrimSpace(os.Getenv(pwdKey))
		if user == "" && pwd == "" {
			continue
		}
		if user == "" || pwd == "" {
			partialKeys = append(partialKeys, key)
			continue
		}
		profile, err := buildEnvProfile(key, user, pwd)
		if err != nil {
			return nil, false, err
		}
		return profile, true, nil
	}
	if len(partialKeys) > 0 {
		return nil, false, fmt.Errorf("incomplete CI target env credentials for %s; set both IICS_USER_<TARGET> and IICS_PWD_<TARGET>", strings.Join(partialKeys, ","))
	}

	// Fallback to global target overrides when provided.
	globalUser := strings.TrimSpace(os.Getenv("IICS_TARGET_USERNAME"))
	globalPwd := strings.TrimSpace(os.Getenv("IICS_TARGET_PASSWORD"))
	if globalUser != "" || globalPwd != "" {
		if globalUser == "" || globalPwd == "" {
			return nil, false, fmt.Errorf("incomplete global target credentials; set both IICS_TARGET_USERNAME and IICS_TARGET_PASSWORD")
		}
		profile, err := buildEnvProfile(strings.ToUpper(strings.TrimSpace(target)), globalUser, globalPwd)
		if err != nil {
			return nil, false, err
		}
		return profile, true, nil
	}
	return nil, false, nil
}

func buildEnvProfile(targetKey, username, password string) (*config.Profile, error) {
	loginURL := strings.TrimSpace(os.Getenv("IICS_LOGIN_URL_" + targetKey))
	region := strings.TrimSpace(os.Getenv("IICS_REGION_" + targetKey))
	if loginURL == "" {
		loginURL = strings.TrimSpace(os.Getenv("IICS_TARGET_LOGIN_URL"))
	}
	if region == "" {
		region = strings.TrimSpace(os.Getenv("IICS_TARGET_REGION"))
	}
	if loginURL == "" && region == "" {
		return nil, fmt.Errorf("missing login URL/region for target %q; set IICS_LOGIN_URL_%s or IICS_REGION_%s (or IICS_TARGET_LOGIN_URL/IICS_TARGET_REGION)", targetKey, targetKey, targetKey)
	}
	return &config.Profile{
		Name:     "env-" + strings.ToLower(targetKey),
		Username: username,
		Password: password,
		LoginURL: loginURL,
		Region:   region,
	}, nil
}

func resolveTargetClient(targetProfileName string, opts TargetResolutionOptions) (*client.Client, string, error) {
	cfg, err := config.Load("")
	if err != nil {
		return nil, "", fmt.Errorf("loading config for target %q: %w", targetProfileName, err)
	}
	profileMap, err := resolveEffectiveTargetProfileMap(opts.TargetProfileMap)
	if err != nil {
		return nil, "", err
	}
	resolvedProfileName, explicitlyMapped, err := resolveProfileNameForTarget(cfg, targetProfileName, profileMap)
	if err != nil {
		return nil, "", err
	}

	var profile *config.Profile
	if resolvedProfileName != "" {
		profile, err = cfg.ResolveTargetProfile(resolvedProfileName)
		if err != nil {
			return nil, "", err
		}
	} else {
		if explicitlyMapped {
			return nil, "", fmt.Errorf("target profile resolution failed for target %q", targetProfileName)
		}
		envProfile, found, envErr := resolveEnvTargetProfile(targetProfileName)
		if envErr != nil {
			return nil, "", envErr
		}
		if !found {
			return nil, "", fmt.Errorf(
				`target profile %q not found in config and no CI env credentials found for target %q; set profile, --target-profile-map/%s, or env vars IICS_USER_%s and IICS_PWD_%s`,
				targetProfileName, targetProfileName, targetProfileMapEnv, strings.ToUpper(targetProfileName), strings.ToUpper(targetProfileName),
			)
		}
		profile = envProfile
		resolvedProfileName = "env:" + strings.ToUpper(targetProfileName)
	}

	loginURL, err := profile.GetLoginURL()
	if err != nil {
		return nil, "", fmt.Errorf("resolving target login URL for profile %q: %w", resolvedProfileName, err)
	}
	clientOptions := []client.ClientOption{
		client.WithVerbose(opts.Verbose),
		client.WithDebug(opts.Debug),
	}
	return client.NewClient(loginURL, profile.Username, profile.Password, clientOptions...), resolvedProfileName, nil
}

// validateAssetsExistInTarget checks existence of every asset against tc,
// batching everything except Connection-typed assets (which have no bulk
// lookup API) into a single Lookup call instead of one round trip per asset.
// A lookup failure is recorded as an "unknown" status against the affected
// assets rather than aborting the whole pass, matching the previous
// per-asset error handling.
func validateAssetsExistInTarget(ctx context.Context, tc *client.Client, assets []Asset) map[string]AssetValidation {
	out := make(map[string]AssetValidation, len(assets))

	var lookupAssets []Asset
	var lookupObjs []client.LookupObject
	for _, a := range assets {
		if a.Type == "Connection" {
			out[a.Location] = validateConnectionExists(ctx, tc, a)
			continue
		}
		lookupAssets = append(lookupAssets, a)
		lookupObjs = append(lookupObjs, client.LookupObject{Path: a.Path, Type: a.Type})
	}

	if len(lookupObjs) == 0 {
		return out
	}

	resp, err := tc.Lookup(ctx, lookupObjs)
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) &&
			bytes.Contains(apiErr.ResponseBody, []byte(`"V3API_LookupError_012"`)) {
			for _, a := range lookupAssets {
				out[a.Location] = AssetValidation{Status: "missing"}
			}
			return out
		}
		warning := err.Error()
		for _, a := range lookupAssets {
			out[a.Location] = AssetValidation{Status: "unknown", Warning: warning}
		}
		return out
	}

	found := make(map[string]bool, len(resp.Objects))
	for _, obj := range resp.Objects {
		found[client.BuildLookupMatchKey(obj.Path, obj.Type)] = true
	}
	for _, a := range lookupAssets {
		if found[client.BuildLookupMatchKey(a.Path, a.Type)] {
			out[a.Location] = AssetValidation{Status: "found"}
		} else {
			out[a.Location] = AssetValidation{Status: "missing"}
		}
	}
	return out
}

func validateConnectionExists(ctx context.Context, tc *client.Client, a Asset) AssetValidation {
	name := a.Path
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		name = name[idx+1:]
	}
	_, err := tc.GetConnectionByName(ctx, name)
	if err == nil {
		return AssetValidation{Status: "found"}
	}
	if isMissingConnectionError(err) {
		return AssetValidation{Status: "missing"}
	}
	return AssetValidation{Status: "unknown", Warning: err.Error()}
}

func isMissingConnectionError(err error) bool {
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	if apiErr.IsNotFound() {
		return true
	}
	if apiErr.StatusCode != http.StatusForbidden || !strings.EqualFold(strings.TrimSpace(apiErr.Code), "APP_13436") {
		return false
	}
	msg := strings.ToLower(strings.TrimSpace(apiErr.Message))
	if strings.Contains(msg, "no object named connection exists") {
		return true
	}
	body := strings.ToLower(string(apiErr.ResponseBody))
	return strings.Contains(body, "no object named connection exists")
}

func SortAssetsByLocation(assets []Asset) []Asset {
	out := make([]Asset, len(assets))
	copy(out, assets)
	sort.SliceStable(out, func(i, j int) bool {
		left := strings.ToLower(strings.TrimSpace(out[i].Location))
		right := strings.ToLower(strings.TrimSpace(out[j].Location))
		if left != right {
			return left < right
		}
		if out[i].Type != out[j].Type {
			return out[i].Type < out[j].Type
		}
		return out[i].Path < out[j].Path
	})
	return out
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
			row = append(row, assetFieldValue(a, field))
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

func WriteAssetsJSON(path string, assets []Asset, fields []string) error {
	if len(fields) == 0 {
		fields = []string{"location"}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}
	rows := make([]orderedAssetRow, 0, len(assets))
	for _, a := range assets {
		rows = append(rows, newOrderedAssetRow(a, fields))
	}
	data, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return fmt.Errorf("serializing assets json: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("writing json %s: %w", path, err)
	}
	return nil
}

func WriteAssetsYAML(path string, assets []Asset, fields []string) error {
	if len(fields) == 0 {
		fields = []string{"location"}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}
	rows := make([]orderedAssetRow, 0, len(assets))
	for _, a := range assets {
		rows = append(rows, newOrderedAssetRow(a, fields))
	}
	data, err := yaml.Marshal(rows)
	if err != nil {
		return fmt.Errorf("serializing assets yaml: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing yaml %s: %w", path, err)
	}
	return nil
}

func assetFieldValue(a Asset, field string) string {
	f := strings.ToLower(strings.TrimSpace(field))
	switch f {
	case "location":
		return a.Location
	case "dependency":
		return a.Dependency
	case "id":
		return a.ID
	case "type":
		return a.Type
	case "path":
		return a.Path
	case "status":
		return a.Status
	case "warning":
		return a.Warning
	default:
		if strings.HasPrefix(f, "status (") && strings.HasSuffix(f, ")") {
			return a.Status
		}
		if strings.HasPrefix(f, "warning (") && strings.HasSuffix(f, ")") {
			return a.Warning
		}
		return ""
	}
}

type orderedAssetRow struct {
	fields []string
	values []string
}

func newOrderedAssetRow(a Asset, fields []string) orderedAssetRow {
	normalized := make([]string, len(fields))
	values := make([]string, len(fields))
	for i, field := range fields {
		key := strings.ToLower(strings.TrimSpace(field))
		normalized[i] = key
		values[i] = assetFieldValue(a, key)
	}
	return orderedAssetRow{
		fields: normalized,
		values: values,
	}
}

func (r orderedAssetRow) MarshalJSON() ([]byte, error) {
	var sb strings.Builder
	sb.WriteByte('{')
	for i := range r.fields {
		if i > 0 {
			sb.WriteByte(',')
		}
		key, err := json.Marshal(r.fields[i])
		if err != nil {
			return nil, err
		}
		value, err := json.Marshal(r.values[i])
		if err != nil {
			return nil, err
		}
		sb.Write(key)
		sb.WriteByte(':')
		sb.Write(value)
	}
	sb.WriteByte('}')
	return []byte(sb.String()), nil
}

func (r orderedAssetRow) MarshalYAML() (interface{}, error) {
	node := &yaml.Node{Kind: yaml.MappingNode}
	for i := range r.fields {
		node.Content = append(node.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: r.fields[i]},
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: r.values[i]},
		)
	}
	return node, nil
}
