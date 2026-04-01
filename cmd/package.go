package cmd

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/config"
	"github.com/jbrazda/iics-cli/internal/output"
	"github.com/spf13/cobra"
)

func newPackageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "package",
		Short: "Work with IICS export package files",
	}
	cmd.AddCommand(newPackageExpandCmd())
	cmd.AddCommand(newPackageCreateCmd())
	cmd.AddCommand(newPackageDependenciesCmd())
	return cmd
}

// ---------------------------------------------------------------------------
// package expand
// ---------------------------------------------------------------------------

func newPackageExpandCmd() *cobra.Command {
	var (
		file      string
		target    string
		recursive bool
		clean     bool
		yes       bool
	)
	cmd := &cobra.Command{
		Use:   "expand",
		Short: "Extract an IICS export ZIP package to a directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			// 1. Validate --file
			if _, err := os.Stat(file); err != nil {
				return fmt.Errorf("file not found: %w", err)
			}

			// 2. Check target directory
			entries, readErr := os.ReadDir(target)
			if readErr == nil && len(entries) > 0 {
				if !clean {
					return fmt.Errorf("target directory is not empty; use --clean to overwrite")
				}
				if !yes {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "About to delete %d entries from %s. Continue? [y/N]: ", len(entries), target)
					var confirm string
					_, _ = fmt.Scanln(&confirm)
					if confirm != "y" && confirm != "Y" {
						_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Canceled.")
						return nil
					}
				}
				for _, e := range entries {
					if err := os.RemoveAll(filepath.Join(target, e.Name())); err != nil {
						return fmt.Errorf("cleaning target: %w", err)
					}
				}
			}

			// 3. Create target directory
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("creating target directory: %w", err)
			}

			// 4. Open ZIP
			r, err := zip.OpenReader(file)
			if err != nil {
				return fmt.Errorf("opening zip: %w", err)
			}
			defer func() { _ = r.Close() }()

			if verbose {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Contents of %s:\n", file)
				for _, f := range r.File {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", f.Name)
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout())
			}

			// 5. Extract entries
			count, err := extractZIPEntries(r.File, target, recursive)
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Expanded %d files to %s\n", count, target)

			if verbose {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nContents of %s:\n", target)
				if walkErr := filepath.Walk(target, func(path string, fi os.FileInfo, werr error) error {
					if werr != nil {
						return werr
					}
					if !fi.IsDir() {
						rel, _ := filepath.Rel(target, path)
						_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", filepath.ToSlash(rel))
					}
					return nil
				}); walkErr != nil {
					return walkErr
				}
			}

			return nil
		},
	}
	cmd.Flags().StringVarP(&file, "file", "f", "", "path to the source ZIP package (required)")
	cmd.Flags().StringVarP(&target, "target", "t", "", "destination directory (required; created if absent)")
	cmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "recursively expand nested ZIPs into <name>.zip/ folders")
	cmd.Flags().BoolVarP(&clean, "clean", "c", false, "delete target contents before expanding")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "suppress confirmation prompt for --clean")
	_ = cmd.MarkFlagRequired("file")
	_ = cmd.MarkFlagRequired("target")
	return cmd
}

// extractZIPEntries extracts zip entries into destDir.
// JSON files are pretty-printed. If recursive is true, entries whose name ends
// in .zip are expanded into a directory named <entry>.zip/ instead of written as files.
func extractZIPEntries(files []*zip.File, destDir string, recursive bool) (int, error) {
	count := 0
	for _, f := range files {
		destPath := filepath.Join(destDir, filepath.FromSlash(f.Name))

		// Guard against zip slip (path traversal)
		if !strings.HasPrefix(
			filepath.Clean(destPath)+string(os.PathSeparator),
			filepath.Clean(destDir)+string(os.PathSeparator),
		) {
			return count, fmt.Errorf("illegal file path in zip: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(destPath, 0o755); err != nil {
				return count, fmt.Errorf("mkdir %s: %w", destPath, err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return count, fmt.Errorf("mkdir parent of %s: %w", destPath, err)
		}

		rc, err := f.Open()
		if err != nil {
			return count, fmt.Errorf("opening %s: %w", f.Name, err)
		}
		data, readErr := io.ReadAll(rc)
		_ = rc.Close()
		if readErr != nil {
			return count, fmt.Errorf("reading %s: %w", f.Name, readErr)
		}

		// Recursively expand nested ZIP into a directory
		if recursive && strings.HasSuffix(strings.ToLower(f.Name), ".zip") {
			nested, nestedErr := expandNestedZIP(data, destPath)
			if nestedErr != nil {
				return count, fmt.Errorf("expanding nested zip %s: %w", f.Name, nestedErr)
			}
			count += nested
			continue
		}

		// Pretty-print JSON
		if strings.HasSuffix(strings.ToLower(f.Name), ".json") {
			if pretty, prettErr := prettyJSON(data); prettErr == nil {
				data = pretty
			}
		}

		if err := os.WriteFile(destPath, data, 0o644); err != nil {
			return count, fmt.Errorf("writing %s: %w", destPath, err)
		}
		count++
	}
	return count, nil
}

// expandNestedZIP extracts zip bytes into destDir (whose name ends in .zip).
func expandNestedZIP(data []byte, destDir string) (int, error) {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return 0, fmt.Errorf("mkdir %s: %w", destDir, err)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return 0, fmt.Errorf("reading nested zip: %w", err)
	}
	count := 0
	for _, f := range zr.File {
		destPath := filepath.Join(destDir, filepath.FromSlash(f.Name))
		if f.FileInfo().IsDir() {
			if mkErr := os.MkdirAll(destPath, 0o755); mkErr != nil {
				return count, mkErr
			}
			continue
		}
		if mkErr := os.MkdirAll(filepath.Dir(destPath), 0o755); mkErr != nil {
			return count, mkErr
		}
		rc, openErr := f.Open()
		if openErr != nil {
			return count, openErr
		}
		content, readErr := io.ReadAll(rc)
		_ = rc.Close()
		if readErr != nil {
			return count, readErr
		}
		if strings.HasSuffix(strings.ToLower(f.Name), ".json") {
			if pretty, prettErr := prettyJSON(content); prettErr == nil {
				content = pretty
			}
		}
		if writeErr := os.WriteFile(destPath, content, 0o644); writeErr != nil {
			return count, writeErr
		}
		count++
	}
	return count, nil
}

// prettyJSON re-marshals data with 2-space indentation.
func prettyJSON(data []byte) ([]byte, error) {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return json.MarshalIndent(v, "", "  ")
}

// ---------------------------------------------------------------------------
// package create
// ---------------------------------------------------------------------------

func newPackageCreateCmd() *cobra.Command {
	var (
		source string
		target string
		force  bool
	)
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an IICS export ZIP package from a directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			// 1. Validate source
			fi, err := os.Stat(source)
			if err != nil {
				return fmt.Errorf("source: %w", err)
			}
			if !fi.IsDir() {
				return fmt.Errorf("source must be a directory: %s", source)
			}

			// 2. Check target
			if _, statErr := os.Stat(target); statErr == nil && !force {
				return fmt.Errorf("output file already exists; use --force to overwrite: %s", target)
			}

			// 3. Collect file contents
			absSource, err := filepath.Abs(source)
			if err != nil {
				return fmt.Errorf("resolving source: %w", err)
			}

			fileContents := make(map[string][]byte)
			var zipDirs []string // relative paths of dirs whose name ends in .zip

			walkErr := filepath.Walk(absSource, func(path string, info os.FileInfo, werr error) error {
				if werr != nil {
					return werr
				}
				if path == absSource {
					return nil
				}
				base := filepath.Base(path)

				// Skip dot files / dot directories
				if strings.HasPrefix(base, ".") {
					if info.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}

				// Skip exportPackage.chksum (always regenerated)
				if !info.IsDir() && base == "exportPackage.chksum" {
					return nil
				}

				if info.IsDir() {
					if strings.HasSuffix(base, ".zip") {
						// Directory from recursive expand — will be re-zipped
						rel, _ := filepath.Rel(absSource, path)
						zipDirs = append(zipDirs, rel)
						return filepath.SkipDir
					}
					return nil
				}

				rel, _ := filepath.Rel(absSource, path)
				data, readErr := os.ReadFile(path)
				if readErr != nil {
					return fmt.Errorf("reading %s: %w", rel, readErr)
				}
				fileContents[filepath.ToSlash(rel)] = data
				return nil
			})
			if walkErr != nil {
				return fmt.Errorf("walking source: %w", walkErr)
			}

			// Re-zip expanded .zip directories back into nested ZIPs
			for _, rel := range zipDirs {
				data, zipErr := createNestedZIP(filepath.Join(absSource, rel))
				if zipErr != nil {
					return fmt.Errorf("creating nested zip for %s: %w", rel, zipErr)
				}
				fileContents[filepath.ToSlash(rel)] = data
			}

			// 4. Generate checksum from all collected files
			chksum := generatePackageChecksum(fileContents)

			// 5. Write output ZIP
			outFile, err := os.Create(target)
			if err != nil {
				return fmt.Errorf("creating output: %w", err)
			}
			zw := zip.NewWriter(outFile)

			sortedPaths := make([]string, 0, len(fileContents))
			for p := range fileContents {
				sortedPaths = append(sortedPaths, p)
			}
			sort.Strings(sortedPaths)

			for _, p := range sortedPaths {
				w, wErr := zw.Create(p)
				if wErr != nil {
					_ = outFile.Close()
					return fmt.Errorf("creating entry %s: %w", p, wErr)
				}
				if _, wErr = w.Write(fileContents[p]); wErr != nil {
					_ = outFile.Close()
					return fmt.Errorf("writing entry %s: %w", p, wErr)
				}
			}

			// Add checksum as the final entry
			cw, cwErr := zw.Create("exportPackage.chksum")
			if cwErr != nil {
				_ = outFile.Close()
				return fmt.Errorf("creating chksum entry: %w", cwErr)
			}
			if _, cwErr = cw.Write([]byte(chksum)); cwErr != nil {
				_ = outFile.Close()
				return fmt.Errorf("writing chksum: %w", cwErr)
			}

			if closeErr := zw.Close(); closeErr != nil {
				_ = outFile.Close()
				return fmt.Errorf("finalizing zip: %w", closeErr)
			}
			if closeErr := outFile.Close(); closeErr != nil {
				return fmt.Errorf("closing output: %w", closeErr)
			}

			if verbose {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Files in package:\n")
				for _, p := range sortedPaths {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", p)
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  exportPackage.chksum\n")
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created %s (%d files)\n", target, len(sortedPaths)+1)
			return nil
		},
	}
	cmd.Flags().StringVarP(&source, "source", "s", "", "source directory (required)")
	cmd.Flags().StringVarP(&target, "target", "t", "", "output ZIP file path (required)")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "overwrite existing output file")
	_ = cmd.MarkFlagRequired("source")
	_ = cmd.MarkFlagRequired("target")
	return cmd
}

// createNestedZIP zips the contents of dirPath into a bytes.Buffer and returns the bytes.
func createNestedZIP(dirPath string) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	if walkErr := filepath.Walk(dirPath, func(path string, fi os.FileInfo, werr error) error {
		if werr != nil {
			return werr
		}
		if path == dirPath || fi.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(dirPath, path)
		if relErr != nil {
			return relErr
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		w, wErr := zw.Create(filepath.ToSlash(rel))
		if wErr != nil {
			return wErr
		}
		_, writeErr := w.Write(data)
		return writeErr
	}); walkErr != nil {
		return nil, walkErr
	}
	if closeErr := zw.Close(); closeErr != nil {
		return nil, closeErr
	}
	return buf.Bytes(), nil
}

// generatePackageChecksum produces the exportPackage.chksum content.
// Format: sorted "path=UPPERCASE_SHA256" pairs, one per line, spaces escaped as "\ ".
func generatePackageChecksum(files map[string][]byte) string {
	entries := make([]string, 0, len(files))
	for path, data := range files {
		hash := sha256.Sum256(data)
		hexHash := fmt.Sprintf("%X", hash[:])
		escapedPath := strings.ReplaceAll(path, " ", "\\ ")
		entries = append(entries, escapedPath+"="+hexHash)
	}
	sort.Strings(entries)
	return strings.Join(entries, "\n") + "\n"
}

// ---------------------------------------------------------------------------
// package dependencies
// ---------------------------------------------------------------------------

// publishableTypes is the set of IICS asset types publishable via the CAI publish API.
var publishableTypes = map[string]bool{
	"AI_SERVICE_CONNECTOR": true,
	"AI_CONNECTION":        true,
	"PROCESS":              true,
	"GUIDE":                true,
	"TASKFLOW":             true,
}

// typePriority controls sort order for dependency output (0 = unknown/sort last).
var typePriority = map[string]int{
	"AI_CONNECTION":        1,
	"AI_SERVICE_CONNECTOR": 2,
	"PROCESS":              3,
	"GUIDE":                4,
	"TASKFLOW":             5,
}

// exportMetadata represents the exportMetadata.v2.json file in an IICS export package.
type exportMetadata struct {
	Name            string           `json:"name"`
	SourceOrgID     string           `json:"sourceOrgId"`
	SourceOrgName   string           `json:"sourceOrgName"`
	ExportedObjects []exportedObject `json:"exportedObjects"`
}

// exportedObject is one asset record in exportMetadata.v2.json.
type exportedObject struct {
	ObjectGUID string          `json:"objectGuid"`
	ObjectName string          `json:"objectName"`
	ObjectType string          `json:"objectType"`
	Path       string          `json:"path"`
	Metadata   exportedObjMeta `json:"metadata"`
}

// exportedObjMeta holds the objectRefs array for an exported object.
type exportedObjMeta struct {
	ObjectRefs []string `json:"objectRefs"`
}

// dependencyItem is one row in the dependency output.
type dependencyItem struct {
	Path         string `json:"path"`
	Type         string `json:"type"`
	Source       string `json:"source"`
	TargetStatus string `json:"targetStatus,omitempty"`
	Warning      string `json:"warning,omitempty"`
}

// dependencyEdge represents a directed dependency between two assets (for Mermaid output).
type dependencyEdge struct {
	FromKey string // normalized matchKey of the dependent asset
	ToKey   string // normalized matchKey of the dependency
}

// readExportMetadata reads exportMetadata.v2.json from a ZIP file or workspace directory.
func readExportMetadata(filePath, workspace string) (*exportMetadata, error) {
	if workspace != "" {
		data, err := os.ReadFile(filepath.Join(workspace, "exportMetadata.v2.json"))
		if err != nil {
			return nil, fmt.Errorf("reading exportMetadata.v2.json: %w", err)
		}
		var meta exportMetadata
		if err := json.Unmarshal(data, &meta); err != nil {
			return nil, fmt.Errorf("parsing exportMetadata.v2.json: %w", err)
		}
		return &meta, nil
	}
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening package file: %w", err)
	}
	defer func() { _ = r.Close() }()
	for _, f := range r.File {
		if f.Name == "exportMetadata.v2.json" {
			rc, oErr := f.Open()
			if oErr != nil {
				return nil, fmt.Errorf("opening exportMetadata.v2.json in ZIP: %w", oErr)
			}
			data, rErr := io.ReadAll(rc)
			_ = rc.Close()
			if rErr != nil {
				return nil, fmt.Errorf("reading exportMetadata.v2.json from ZIP: %w", rErr)
			}
			var meta exportMetadata
			if err := json.Unmarshal(data, &meta); err != nil {
				return nil, fmt.Errorf("parsing exportMetadata.v2.json: %w", err)
			}
			return &meta, nil
		}
	}
	return nil, fmt.Errorf("exportMetadata.v2.json not found in package")
}

// resolveDependencies performs a BFS over the package metadata to produce a flat
// dependency list and a set of directed edges for graph rendering.
//
// When c is non-nil, external GUIDs (not present in the package) are resolved via
// the source org API up to 5 BFS depth levels. When c is nil, only assets present
// in the package metadata are included.
//
// publishMode restricts the result to publishableTypes only.
// excludeRe, when non-nil, is matched against "path/name.type" to skip assets
// and stop recursion through them.
func resolveDependencies(
	ctx context.Context,
	meta *exportMetadata,
	c *client.Client,
	publishMode bool,
	excludeRe *regexp.Regexp,
) ([]dependencyItem, []dependencyEdge, error) {

	// Build pkgMap: GUID -> *exportedObject
	pkgMap := make(map[string]*exportedObject, len(meta.ExportedObjects))
	for i := range meta.ExportedObjects {
		pkgMap[meta.ExportedObjects[i].ObjectGUID] = &meta.ExportedObjects[i]
	}

	// guidToMatchKey maps every resolved GUID to its normalized matchKey.
	// matchKey format: "Project/Folder/Name.TYPE" (no leading slash, no "Explore/" prefix).
	// Export metadata paths are "/Explore/Project/Folder"; the V3 API uses "Project/Folder".
	guidToMatchKey := make(map[string]string)
	// resultMap: matchKey -> dependencyItem
	resultMap := make(map[string]dependencyItem)
	// rawEdges: (fromGUID, toGUID) to be filtered at the end
	var rawEdges [][2]string

	// apiPath strips the leading "/" and "Explore/" prefix from an export-metadata path
	// to produce the V3 API path format used by Lookup.
	// "/Explore/ZZ_TEST_CLI/Connections" -> "ZZ_TEST_CLI/Connections"
	// "/Explore" -> "" (projects live directly under Explore root)
	// "/SYS/CDI-G01" -> "SYS/CDI-G01"
	apiPath := func(p string) string {
		p = strings.TrimPrefix(p, "/")
		if p == "Explore" {
			return ""
		}
		p = strings.TrimPrefix(p, "Explore/")
		return p
	}

	// mkFromPkg builds a matchKey from a package object using the V3 API path format.
	mkFromPkg := func(obj *exportedObject) string {
		base := apiPath(obj.Path)
		if base == "" {
			return obj.ObjectName + "." + obj.ObjectType
		}
		return base + "/" + obj.ObjectName + "." + obj.ObjectType
	}
	// mkFromLookup builds a matchKey from a Lookup API result.
	// Lookup result.Path already contains the object name; strip leading "/" and "Explore/".
	mkFromLookup := func(r client.LookupResult) string {
		return apiPath(r.Path) + "." + r.Type
	}

	// Phase 1: process all package objects.
	slog.Info("resolving dependencies: phase 1 - scanning package objects",
		"total", len(meta.ExportedObjects),
		"publishMode", publishMode,
	)
	externalGUIDs := make(map[string]bool)
	for i := range meta.ExportedObjects {
		obj := &meta.ExportedObjects[i]
		mk := mkFromPkg(obj)

		if excludeRe != nil && excludeRe.MatchString(mk) {
			continue
		}
		if publishMode && !publishableTypes[obj.ObjectType] {
			continue
		}

		base := apiPath(obj.Path)
		var fullPath string
		if base == "" {
			fullPath = obj.ObjectName
		} else {
			fullPath = base + "/" + obj.ObjectName
		}
		guidToMatchKey[obj.ObjectGUID] = mk
		resultMap[mk] = dependencyItem{
			Path:   fullPath,
			Type:   obj.ObjectType,
			Source: "package",
		}

		for _, refGUID := range obj.Metadata.ObjectRefs {
			rawEdges = append(rawEdges, [2]string{obj.ObjectGUID, refGUID})
			if _, inPkg := pkgMap[refGUID]; !inPkg {
				externalGUIDs[refGUID] = true
			}
		}
	}

	slog.Info("resolving dependencies: phase 1 complete",
		"packageObjects", len(resultMap),
		"externalGUIDs", len(externalGUIDs),
	)

	// Phase 2: BFS on external GUIDs using the source org API.
	if c != nil && len(externalGUIDs) > 0 {
		slog.Info("resolving dependencies: phase 2 - resolving external GUIDs via source org API",
			"externalGUIDs", len(externalGUIDs),
		)
		visited := make(map[string]bool)

		currentQueue := make([]string, 0, len(externalGUIDs))
		for g := range externalGUIDs {
			currentQueue = append(currentQueue, g)
		}

		for depth := 0; depth < 5 && len(currentQueue) > 0; depth++ {
			// Deduplicate and skip already-visited GUIDs.
			toProcess := make([]string, 0, len(currentQueue))
			for _, g := range currentQueue {
				if !visited[g] {
					visited[g] = true
					toProcess = append(toProcess, g)
				}
			}
			currentQueue = nil
			if len(toProcess) == 0 {
				break
			}

			slog.Info("resolving dependencies: BFS depth level",
				"depth", depth,
				"guids", len(toProcess),
			)

			// Batch lookup by GUID.
			lookupObjs := make([]client.LookupObject, len(toProcess))
			for i, g := range toProcess {
				lookupObjs[i] = client.LookupObject{ID: g}
			}
			resp, err := c.Lookup(ctx, lookupObjs)
			if err != nil {
				return nil, nil, fmt.Errorf("looking up external dependencies (depth %d): %w", depth, err)
			}

			// For each resolved external object, collect its uses refs.
			type parentedRef struct {
				parentGUID string
				path       string
				refType    string
			}
			var allRefs []parentedRef

			for _, result := range resp.Objects {
				mk := mkFromLookup(result)
				if excludeRe != nil && excludeRe.MatchString(mk) {
					continue
				}
				if publishMode && !publishableTypes[result.Type] {
					continue
				}

				guidToMatchKey[result.ID] = mk
				resultMap[mk] = dependencyItem{
					Path:   apiPath(result.Path),
					Type:   result.Type,
					Source: "external",
				}

				// Fetch this object's uses dependencies for further BFS levels.
				depsResp, dErr := c.GetObjectDependencies(ctx, result.ID, "uses", 200, 0)
				if dErr == nil {
					for _, ref := range depsResp.Uses {
						allRefs = append(allRefs, parentedRef{
							parentGUID: result.ID,
							path:       ref.Path,
							refType:    ref.Type,
						})
					}
				}
				// Non-fatal: skip objects whose deps cannot be fetched.
			}

			// Batch-lookup all uses refs by path+type to get GUIDs.
			if len(allRefs) > 0 {
				ptObjs := make([]client.LookupObject, len(allRefs))
				for i, pr := range allRefs {
					ptObjs[i] = client.LookupObject{Path: pr.path, Type: pr.refType}
				}
				ptResp, ptErr := c.Lookup(ctx, ptObjs)
				if ptErr == nil {
					// Build (path.type) -> GUID map from results.
					ptToGUID := make(map[string]string, len(ptResp.Objects))
					for _, r := range ptResp.Objects {
						ptToGUID[mkFromLookup(r)] = r.ID
					}
					// Record edges and enqueue newly discovered GUIDs.
					for _, pr := range allRefs {
						refKey := apiPath(pr.path) + "." + pr.refType
						if childGUID, ok := ptToGUID[refKey]; ok {
							rawEdges = append(rawEdges, [2]string{pr.parentGUID, childGUID})
							if !visited[childGUID] {
								currentQueue = append(currentQueue, childGUID)
							}
						}
					}
				}
				// Non-fatal: if path+type lookup fails we just miss those edges.
			}
		}
	}

	// Filter and deduplicate edges: both endpoints must appear in resultMap.
	var edges []dependencyEdge
	edgeSeen := make(map[[2]string]bool)
	for _, re := range rawEdges {
		fromMK, fromOK := guidToMatchKey[re[0]]
		toMK, toOK := guidToMatchKey[re[1]]
		if !fromOK || !toOK {
			continue
		}
		if _, ok := resultMap[fromMK]; !ok {
			continue
		}
		if _, ok := resultMap[toMK]; !ok {
			continue
		}
		pair := [2]string{fromMK, toMK}
		if !edgeSeen[pair] {
			edgeSeen[pair] = true
			edges = append(edges, dependencyEdge{FromKey: fromMK, ToKey: toMK})
		}
	}

	// Build sorted items slice.
	items := make([]dependencyItem, 0, len(resultMap))
	for _, item := range resultMap {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		pi, pj := typePriority[items[i].Type], typePriority[items[j].Type]
		if pi != pj {
			if pi == 0 {
				return false
			}
			if pj == 0 {
				return true
			}
			return pi < pj
		}
		if items[i].Type != items[j].Type {
			return items[i].Type < items[j].Type
		}
		return items[i].Path < items[j].Path
	})

	return items, edges, nil
}

// validateTargetDependencies looks up each dependency individually in the target org
// and sets TargetStatus ("found" or "missing") and Warning on items in place.
//
// Per-object lookups are used instead of a single batch because:
//   - Some asset types (e.g. Connection/SAAS_CONNECTION, AgentGroup) use different
//     path and type representations in the Lookup API than in the export metadata.
//   - The Lookup API returns V3API_LookupError_012 for the entire batch when ANY item
//     is not found or has an unresolvable path, making batch results unreliable for
//     mixed-type packages.
func validateTargetDependencies(ctx context.Context, targetProfileName string, deps []dependencyItem) error {
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config for target validation: %w", err)
	}
	targetProfile, err := cfg.ResolveTargetProfile(targetProfileName)
	if err != nil {
		return err
	}
	loginURL, err := targetProfile.GetLoginURL()
	if err != nil {
		return fmt.Errorf("resolving target login URL: %w", err)
	}

	tc := client.NewClient(loginURL, targetProfile.Username, targetProfile.Password,
		client.WithDebug(debug), client.WithVerbose(verbose))

	slog.Info("validating dependencies against target org",
		"profile", targetProfileName,
		"count", len(deps),
	)

	for i := range deps {
		d := &deps[i]
		slog.Debug("looking up dependency in target org",
			"path", d.Path,
			"type", d.Type,
		)

		resp, lookupErr := tc.Lookup(ctx, []client.LookupObject{{Path: d.Path, Type: d.Type}})
		if lookupErr != nil {
			var apiErr *client.APIError
			// V3API_LookupError_012: the Lookup API nests error code under "error"."code"
			// so APIError.Code is empty; detect it by scanning the raw response body.
			if errors.As(lookupErr, &apiErr) &&
				bytes.Contains(apiErr.ResponseBody, []byte(`"V3API_LookupError_012"`)) {
				d.TargetStatus = "missing"
				d.Warning = "asset not found in target org"
				slog.Debug("dependency missing in target org", "path", d.Path, "type", d.Type)
			} else {
				// Other errors (unsupported type, network, auth) - report but continue.
				d.TargetStatus = "unknown"
				d.Warning = fmt.Sprintf("lookup error: %v", lookupErr)
				slog.Warn("dependency lookup failed", "path", d.Path, "type", d.Type, "error", lookupErr)
			}
			continue
		}

		// Any non-empty result means the object exists in the target org.
		// We intentionally do NOT match by path because some types (AgentGroup,
		// Connection) return a different path format than what we sent.
		if len(resp.Objects) > 0 {
			d.TargetStatus = "found"
			slog.Debug("dependency found in target org", "path", d.Path, "type", d.Type)
		} else {
			d.TargetStatus = "missing"
			d.Warning = "asset not found in target org"
			slog.Debug("dependency missing in target org (empty result)", "path", d.Path, "type", d.Type)
		}
	}
	return nil
}

// applyFilter returns only the dependency items whose "path.type" matches pattern.
func applyFilter(deps []dependencyItem, pattern string) ([]dependencyItem, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	filtered := deps[:0:0]
	for _, d := range deps {
		if re.MatchString(d.Path + "." + d.Type) {
			filtered = append(filtered, d)
		}
	}
	return filtered, nil
}

// renderMermaid writes a Mermaid graph TD diagram to w.
func renderMermaid(items []dependencyItem, edges []dependencyEdge, w io.Writer) error {
	// Assign short node IDs.
	nodeID := make(map[string]string, len(items))
	for i, item := range items {
		key := item.Path + "." + item.Type
		nodeID[key] = fmt.Sprintf("n%d", i)
	}

	var sb strings.Builder
	sb.WriteString("graph TD\n")

	// Emit node definitions.
	hasMissing := false
	for _, item := range items {
		key := item.Path + "." + item.Type
		id := nodeID[key]

		// Display name: last path segment.
		name := item.Path
		if idx := strings.LastIndex(item.Path, "/"); idx >= 0 {
			name = item.Path[idx+1:]
		}

		label := item.Type + ": " + name
		if item.TargetStatus == "missing" {
			hasMissing = true
			fmt.Fprintf(&sb, "    %s:::missing[\"%s\"]\n", id, label)
		} else {
			fmt.Fprintf(&sb, "    %s[\"%s\"]\n", id, label)
		}
	}

	// Emit edges.
	for _, e := range edges {
		fromID, fromOK := nodeID[e.FromKey]
		toID, toOK := nodeID[e.ToKey]
		if !fromOK || !toOK {
			continue
		}
		fmt.Fprintf(&sb, "    %s --> %s\n", fromID, toID)
	}

	// Emit classDef only when there are missing nodes.
	if hasMissing {
		sb.WriteString("    classDef missing fill:#ffcccc,stroke:#cc0000\n")
	}

	_, err := fmt.Fprint(w, sb.String())
	return err
}

func newPackageDependenciesCmd() *cobra.Command {
	var (
		file           string
		workspace      string
		publishMode    bool
		excludePattern string
		filterPattern  string
		targetProfile  string
	)

	cmd := &cobra.Command{
		Use:   "dependencies",
		Short: "Resolve and list transitive dependencies of an IICS export package",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate mutually exclusive flags.
			if file == "" && workspace == "" {
				return fmt.Errorf("one of --file or --workspace is required")
			}
			if file != "" && workspace != "" {
				return fmt.Errorf("--file and --workspace are mutually exclusive")
			}
			if publishMode && targetProfile == "" {
				return fmt.Errorf("--target-profile is required when --publish is set")
			}

			// Compile exclude regex.
			var excludeRe *regexp.Regexp
			if excludePattern != "" {
				var err error
				excludeRe, err = regexp.Compile(excludePattern)
				if err != nil {
					return fmt.Errorf("invalid --exclude pattern: %w", err)
				}
			}

			// Read package metadata.
			meta, err := readExportMetadata(file, workspace)
			if err != nil {
				return err
			}

			ctx := context.Background()

			// Try to get source client (optional: used for external dep resolution).
			// Skip getClient entirely when no credentials are configured, to avoid
			// triggering the interactive setup wizard unnecessarily.
			var srcClient *client.Client
			if _, _, _, profileErr := resolveProfile(); profileErr == nil || !isMissingCredentialsError(profileErr) {
				if c, clientErr := getClient(cmd); clientErr == nil {
					srcClient = c
				} else if verbose {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Note: could not connect to source org; external dependencies will not be resolved: %v\n", clientErr)
				}
			} else if verbose {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Note: no source profile configured; external dependencies will not be resolved\n")
			}

			// Resolve dependency graph.
			deps, edges, err := resolveDependencies(ctx, meta, srcClient, publishMode, excludeRe)
			if err != nil {
				return err
			}

			// Apply output filter (does not affect resolution).
			if filterPattern != "" {
				deps, err = applyFilter(deps, filterPattern)
				if err != nil {
					return fmt.Errorf("invalid --filter pattern: %w", err)
				}
			}

			// Validate against target org if requested.
			if targetProfile != "" {
				if valErr := validateTargetDependencies(ctx, targetProfile, deps); valErr != nil {
					return valErr
				}
			}

			// Mermaid output is handled before getFormatter() to avoid ParseFormat error.
			if outputFmt == "mermaid" {
				return renderMermaid(deps, edges, cmd.OutOrStdout())
			}

			f, err := getFormatter()
			if err != nil {
				return err
			}

			columns := []output.Column{
				{Header: "PATH", Field: "path", Width: 70},
				{Header: "TYPE", Field: "type", Width: 22},
				{Header: "SOURCE", Field: "source", Width: 10},
			}
			if targetProfile != "" {
				columns = append(columns,
					output.Column{Header: "TARGET", Field: "targetStatus", Width: 10},
					output.Column{Header: "WARNING", Field: "warning", Width: 50},
				)
			}

			return f.Format(deps, columns)
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "path to IICS export ZIP package (mutually exclusive with --workspace)")
	cmd.Flags().StringVarP(&workspace, "workspace", "w", "", "path to expanded workspace directory (mutually exclusive with --file)")
	cmd.Flags().BoolVar(&publishMode, "publish", false, "restrict output to publishable types only; requires --target-profile")
	cmd.Flags().StringVarP(&excludePattern, "exclude", "e", "", "regex matched against path/name.type to exclude assets from resolution")
	cmd.Flags().StringVar(&filterPattern, "filter", "", "regex matched against path/name.type to filter final output (does not affect resolution)")
	cmd.Flags().StringVarP(&targetProfile, "target-profile", "t", "", "profile name for target org validation (required with --publish)")

	return cmd
}
