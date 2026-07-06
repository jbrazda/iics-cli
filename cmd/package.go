package cmd

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
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
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/config"
	"github.com/jbrazda/iics-cli/internal/dependencies"
	"github.com/jbrazda/iics-cli/internal/output"
	"github.com/jbrazda/iics-cli/internal/release"
	"github.com/spf13/cobra"
)

func targetStatusFunc(v interface{}) string {
	row, _ := v.(map[string]interface{})
	status, _ := row["status"].(string)
	if noColor {
		return status
	}
	switch status {
	case "found":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true).Render(status)
	case "missing":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Render(status)
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render(status)
	}
}

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

		// Strip volatile server timestamp from XML assets
		if strings.HasSuffix(strings.ToLower(f.Name), ".xml") {
			data = stripServerTimestamp(data)
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
		if strings.HasSuffix(strings.ToLower(f.Name), ".xml") {
			content = stripServerTimestamp(content)
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

// serverTimestampRE matches the CurrentServerDateTime element with any namespace prefix,
// including surrounding whitespace and the line ending, so ReplaceAll leaves no blank line.
// The namespace prefix (\w+:) is not fixed - Informatica may bind it to a different prefix.
var serverTimestampRE = regexp.MustCompile(
	`(?m)^[ \t]*<\w+:CurrentServerDateTime>[^<]*</\w+:CurrentServerDateTime>[ \t]*\r?\n?`,
)

// stripServerTimestamp removes the server-injected CurrentServerDateTime element from XML
// asset files. This element changes on every export regardless of whether the asset design
// changed, producing noisy git diffs.
func stripServerTimestamp(data []byte) []byte {
	return serverTimestampRE.ReplaceAll(data, nil)
}

// ---------------------------------------------------------------------------
// package create
// ---------------------------------------------------------------------------

func newPackageCreateCmd() *cobra.Command {
	var (
		source                  string
		target                  string
		force                   bool
		manifestFile            string
		packageName             string
		includeTags             bool
		excludeFoundTransitive  bool
		excludeTargetStatusName string
		logFile                 string
	)
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an IICS export ZIP package from a directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			if excludeTargetStatusName != "" && !excludeFoundTransitive {
				return fmt.Errorf("--status-target requires --exclude-found-transitive")
			}
			// 1. Validate source
			fi, err := os.Stat(source)
			if err != nil {
				return fmt.Errorf("source: %w", err)
			}
			if !fi.IsDir() {
				return fmt.Errorf("source must be a directory: %s", source)
			}
			logEnabled, logPath := manifestLogPath(cmd, logFile)

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

			manifestEntries, manifestStats, hasSelectionManifest, err := readPackageSelectionManifest(
				manifestFile,
				excludeFoundTransitive,
				excludeTargetStatusName,
			)
			if err != nil {
				return err
			}
			var reportIncluded []release.ManifestLogAsset
			var reportSelectedCount int
			var reportExcludedCount int
			if hasSelectionManifest {
				if len(manifestEntries) == 0 {
					return fmt.Errorf("selection manifest is empty")
				}
				meta, metaErr := readExportMetadata("", absSource)
				if metaErr != nil {
					return fmt.Errorf("reading source metadata for selective packaging: %w", metaErr)
				}

				exported := make([]dependencies.ExportedObjectRef, 0, len(meta.ExportedObjects))
				for _, o := range meta.ExportedObjects {
					exported = append(exported, dependencies.ExportedObjectRef{
						ObjectGUID: o.ObjectGUID,
						ObjectName: o.ObjectName,
						ObjectType: o.ObjectType,
						Path:       o.Path,
					})
				}
				selectedIDs, warnings, selErr := dependencies.SelectExportedObjects(manifestEntries, exported)
				if selErr != nil {
					return selErr
				}
				parentAdded := dependencies.IncludeParentContainers(exported, selectedIDs)
				closureNodes := make([]dependencies.RefClosureNode, 0, len(meta.ExportedObjects))
				for _, o := range meta.ExportedObjects {
					if o.ObjectGUID == "" {
						continue
					}
					closureNodes = append(closureNodes, dependencies.RefClosureNode{
						ID:   o.ObjectGUID,
						Refs: o.objectRefs(),
					})
				}
				excludedSelectedIDs := make(map[string]bool)
				if excludeFoundTransitive && len(manifestStats.ExcludedEntries) > 0 {
					resolvedExcluded, _, excludedSelErr := dependencies.SelectExportedObjects(manifestStats.ExcludedEntries, exported)
					if excludedSelErr != nil {
						return fmt.Errorf("resolving excluded transitive-found entries: %w", excludedSelErr)
					}
					excludedSelectedIDs = resolvedExcluded
				}

				closureAdded := 0
				closureSuppressedExcluded := 0
				if !excludeFoundTransitive {
					closureAdded = dependencies.IncludeReferencedClosure(closureNodes, selectedIDs)
				} else {
					closureAddedIDs := dependencies.AddedIDsAfterClosure(closureNodes, selectedIDs)
					closureSuppressedExcluded = len(closureAddedIDs)
					if len(excludedSelectedIDs) > 0 && len(closureAddedIDs) > 0 {
						// Keep a floor count based on all closure-suppressed additions and
						// use excluded-entry overlap to avoid undercounting when manifest rows
						// and closure additions are both present.
						overlap := dependencies.CountSetIntersection(closureAddedIDs, excludedSelectedIDs)
						if overlap > closureSuppressedExcluded {
							closureSuppressedExcluded = overlap
						}
					}
				}
				if len(selectedIDs) == 0 {
					return fmt.Errorf("no assets matched selection manifest")
				}
				for _, w := range warnings {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: %s\n", w)
				}
				if verbose && excludeFoundTransitive {
					totalExcluded := manifestStats.ExcludedTransitiveFound + closureSuppressedExcluded
					_, _ = fmt.Fprintf(
						cmd.OutOrStdout(),
						"Selection filter: excluded %d transitive found rows using %s (manifest rows: %d, closure-suppressed: %d)\n",
						totalExcluded,
						manifestStats.SelectedStatusColumnName,
						manifestStats.ExcludedTransitiveFound,
						closureSuppressedExcluded,
					)
				}
				if verbose && parentAdded > 0 {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Selection refinement: included %d inferred parent Project/Folder objects\n", parentAdded)
				}
				if verbose && closureAdded > 0 {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Selection refinement: included %d in-package referenced dependencies\n", closureAdded)
				}
				cdiRefAdded := 0
				if excludeFoundTransitive {
					refNodes := make([]dependencies.CDIObjectRefNode, 0, len(meta.ExportedObjects))
					for _, o := range meta.ExportedObjects {
						if o.ObjectGUID == "" {
							continue
						}
						refNodes = append(refNodes, dependencies.CDIObjectRefNode{
							ID:         o.ObjectGUID,
							Type:       o.ObjectType,
							ObjectRefs: o.objectRefs(),
						})
					}
					cdiRefAdded = dependencies.IncludeCDISysRefsFromObjectRefs(refNodes, selectedIDs)
					if verbose && cdiRefAdded > 0 {
						_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Selection refinement: included %d CDI Connection/AgentGroup refs from objectRefs\n", cdiRefAdded)
					}
				}

				selectedObjects := make([]exportedObject, 0, len(selectedIDs))
				for _, o := range meta.ExportedObjects {
					if selectedIDs[o.ObjectGUID] {
						selectedObjects = append(selectedObjects, o)
					}
				}
				if len(selectedObjects) == 0 {
					return fmt.Errorf("selection manifest resolved no exported objects")
				}
				reportIncluded = exportedObjectsToManifestLog(selectedObjects)
				reportSelectedCount = len(selectedObjects)
				reportExcludedCount = manifestStats.ExcludedTransitiveFound + closureSuppressedExcluded
				metadataObjects := selectedObjects
				if !excludeFoundTransitive {
					selectedNodes := make([]dependencies.ObjectRefsNode, len(selectedObjects))
					for i, o := range selectedObjects {
						selectedNodes[i] = dependencies.ObjectRefsNode{
							ID:         o.ObjectGUID,
							ObjectRefs: o.objectRefs(),
						}
					}
					prunedRefsByID, prunedCount := dependencies.PruneDanglingObjectRefs(selectedNodes)
					for i := range selectedObjects {
						if setErr := selectedObjects[i].setObjectRefs(prunedRefsByID[selectedObjects[i].ObjectGUID]); setErr != nil {
							return setErr
						}
					}
					postPruneNodes := make([]dependencies.ObjectRefsNode, len(selectedObjects))
					for i, o := range selectedObjects {
						postPruneNodes[i] = dependencies.ObjectRefsNode{
							ID:         o.ObjectGUID,
							ObjectRefs: o.objectRefs(),
						}
					}
					if dangling := dependencies.CountDanglingObjectRefs(postPruneNodes); dangling > 0 {
						return fmt.Errorf("selection produced %d unresolved objectRefs after pruning; cannot create import-safe package", dangling)
					}
					if verbose && prunedCount > 0 {
						_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Selection refinement: pruned %d dangling metadata objectRefs\n", prunedCount)
					}
				} else {
					metadataIDs := buildMetadataGraphForSelection(meta.ExportedObjects, selectedIDs, excludedSelectedIDs)
					metadataObjectList := make([]exportedObject, 0, len(metadataIDs))
					for _, o := range meta.ExportedObjects {
						if metadataIDs[o.ObjectGUID] {
							metadataObjectList = append(metadataObjectList, o)
						}
					}
					if len(metadataObjectList) == 0 {
						return fmt.Errorf("selection metadata graph resolved no exported objects")
					}
					metadataNodes := make([]dependencies.ObjectRefsNode, len(metadataObjectList))
					for i, o := range metadataObjectList {
						metadataNodes[i] = dependencies.ObjectRefsNode{
							ID:         o.ObjectGUID,
							ObjectRefs: o.objectRefs(),
						}
					}
					prunedRefsByID, _ := dependencies.PruneDanglingObjectRefs(metadataNodes)
					for i := range metadataObjectList {
						if setErr := metadataObjectList[i].setObjectRefs(prunedRefsByID[metadataObjectList[i].ObjectGUID]); setErr != nil {
							return setErr
						}
					}
					metadataObjects = metadataObjectList
				}

				filtered := filterPackageFilesForSelection(fileContents, meta.ExportedObjects, selectedIDs)
				if len(filtered) == 0 {
					return fmt.Errorf("no package files remained after selection filtering")
				}

				finalPackageName := packageName
				if finalPackageName == "" {
					finalPackageName = strings.TrimSuffix(filepath.Base(target), filepath.Ext(target))
				}

				meta.Name = finalPackageName
				if !includeTags {
					meta.Tags = nil
				}
				meta.ExportedObjects = metadataObjects
				metaData, marshalErr := json.MarshalIndent(meta, "", "  ")
				if marshalErr != nil {
					return fmt.Errorf("serializing filtered exportMetadata.v2.json: %w", marshalErr)
				}
				filtered["exportMetadata.v2.json"] = metaData

				contentsCSV, csvErr := buildContentsOfExportPackageCSV(selectedObjects)
				if csvErr != nil {
					return csvErr
				}
				filtered["ContentsofExportPackage_"+finalPackageName+".csv"] = contentsCSV
				fileContents = filtered
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
			if logEnabled {
				appendManifestLogWarning(cmd, logPath, release.RenderPackageCreateLog(release.PackageCreateLog{
					PackagePath:             target,
					Source:                  source,
					SelectionManifest:       manifestFile,
					PackageName:             packageName,
					FileCount:               len(sortedPaths) + 1,
					AssetsSelected:          reportSelectedCount,
					TransitiveFoundExcluded: reportExcludedCount,
					IncludedAssets:          reportIncluded,
				}))
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&source, "source", "s", "", "source directory (required)")
	cmd.Flags().StringVarP(&target, "target", "t", "", "output ZIP file path (required)")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "overwrite existing output file")
	cmd.Flags().StringVarP(&manifestFile, "manifest-file", "m", "", "inclusion manifest file (.txt/.csv/.json/.yaml); omit to read from stdin when piped")
	cmd.Flags().StringVarP(&packageName, "name", "n", "", "package name override used for ContentsofExportPackage_<name>.csv")
	cmd.Flags().BoolVar(&includeTags, "include-tags", false, "include root tags in regenerated exportMetadata.v2.json")
	cmd.Flags().BoolVar(&excludeFoundTransitive, "exclude-found-transitive", false, "exclude manifest rows where dependency is transitive and status is found in target")
	cmd.Flags().StringVar(&excludeTargetStatusName, "status-target", "", "target key for STATUS (<target>) column (for example: qa); required when multiple STATUS columns exist")
	bindManifestLogFlag(cmd, &logFile)
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

func hasPipedStdin() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice == 0
}

func readPackageSelectionManifest(
	manifestFile string,
	excludeFoundTransitive bool,
	statusTarget string,
) ([]client.ArtifactEntry, dependencies.BuildManifestStats, bool, error) {
	stats := dependencies.BuildManifestStats{}
	parseByFormat := func(data []byte, format, sourceLabel string) ([]client.ArtifactEntry, dependencies.BuildManifestStats, error) {
		if format == "csv" {
			return dependencies.ParseBuildManifestCSV(data, dependencies.BuildManifestParseOptions{
				ExcludeFoundTransitive: excludeFoundTransitive,
				TargetStatus:           statusTarget,
			})
		}
		if excludeFoundTransitive {
			return nil, stats, fmt.Errorf("--exclude-found-transitive currently supports csv manifests; got %s from %s", format, sourceLabel)
		}
		entries, err := client.ParseArtifactsReader(bytes.NewReader(data), format)
		if err != nil {
			return nil, stats, err
		}
		return entries, stats, nil
	}

	if manifestFile != "" {
		data, err := os.ReadFile(manifestFile)
		if err != nil {
			return nil, stats, false, fmt.Errorf("reading --manifest-file: %w", err)
		}
		format := strings.ToLower(strings.TrimPrefix(filepath.Ext(manifestFile), "."))
		if format == "yml" {
			format = "yaml"
		}
		if format == "" {
			format = client.DetectArtifactsFormat(data)
		}
		entries, parseStats, parseErr := parseByFormat(data, format, "--manifest-file")
		if parseErr != nil {
			return nil, stats, false, fmt.Errorf("parsing --manifest-file: %w", parseErr)
		}
		return entries, parseStats, true, nil
	}
	if !hasPipedStdin() {
		return nil, stats, false, nil
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, stats, false, fmt.Errorf("reading stdin: %w", err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, stats, false, nil
	}
	format := detectDataFormat(data)
	entries, parseStats, parseErr := parseByFormat(data, format, "stdin")
	if parseErr != nil {
		return nil, stats, false, fmt.Errorf("parsing selection manifest from stdin (%s): %w", format, parseErr)
	}
	return entries, parseStats, true, nil
}

func filterPackageFilesForSelection(
	fileContents map[string][]byte,
	allObjects []exportedObject,
	selectedIDs map[string]bool,
) map[string][]byte {
	allObjectFiles := make(map[string]bool)
	selectedObjectFiles := make(map[string]bool)
	for _, o := range allObjects {
		candidates := dependencies.ObjectChecksumCandidates(o.Path, o.ObjectName, o.ObjectType)
		for _, c := range candidates {
			if _, ok := fileContents[c]; ok {
				allObjectFiles[c] = true
				if selectedIDs[o.ObjectGUID] {
					selectedObjectFiles[c] = true
				}
			}
		}
	}

	out := make(map[string][]byte)
	for path, data := range fileContents {
		if path == "exportPackage.chksum" || path == "exportMetadata.v2.json" {
			continue
		}
		if strings.HasPrefix(path, "ContentsofExportPackage_") && strings.HasSuffix(path, ".csv") {
			continue
		}
		if allObjectFiles[path] {
			if selectedObjectFiles[path] {
				out[path] = data
			}
			continue
		}
		out[path] = data
	}
	return out
}

func buildMetadataGraphForSelection(
	allObjects []exportedObject,
	selectedIDs map[string]bool,
	excludedIDs map[string]bool,
) map[string]bool {
	metadataIDs := make(map[string]bool, len(selectedIDs))
	for id := range selectedIDs {
		metadataIDs[id] = true
	}
	byID := make(map[string]exportedObject, len(allObjects))
	exportedRefs := make([]dependencies.ExportedObjectRef, 0, len(allObjects))
	for _, o := range allObjects {
		if o.ObjectGUID == "" {
			continue
		}
		byID[o.ObjectGUID] = o
		exportedRefs = append(exportedRefs, dependencies.ExportedObjectRef{
			ObjectGUID: o.ObjectGUID,
			ObjectName: o.ObjectName,
			ObjectType: o.ObjectType,
			Path:       o.Path,
		})
	}

	queue := make([]string, 0, len(metadataIDs))
	for id := range metadataIDs {
		queue = append(queue, id)
	}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		obj, ok := byID[id]
		if !ok {
			continue
		}
		for _, refID := range obj.objectRefs() {
			if excludedIDs[refID] && !selectedIDs[refID] {
				continue
			}
			if _, exists := byID[refID]; !exists {
				continue
			}
			if metadataIDs[refID] {
				continue
			}
			metadataIDs[refID] = true
			queue = append(queue, refID)
		}
	}

	// Include container hierarchy needed for selected/closure objects.
	_ = dependencies.IncludeParentContainers(exportedRefs, metadataIDs)
	for id := range excludedIDs {
		if selectedIDs[id] {
			continue
		}
		delete(metadataIDs, id)
	}
	return metadataIDs
}

func buildContentsOfExportPackageCSV(objects []exportedObject) ([]byte, error) {
	rows := make([][]string, 0, len(objects)+1)
	rows = append(rows, []string{"objectPath", "objectName", "objectType", "id"})

	sorted := make([]exportedObject, len(objects))
	copy(sorted, objects)
	sort.SliceStable(sorted, func(i, j int) bool {
		a := client.NormalizeLocationPath(sorted[i].Path) + "/" + sorted[i].ObjectName + "." + sorted[i].ObjectType
		b := client.NormalizeLocationPath(sorted[j].Path) + "/" + sorted[j].ObjectName + "." + sorted[j].ObjectType
		return a < b
	})

	for _, o := range sorted {
		rows = append(rows, []string{
			o.Path,
			o.ObjectName,
			o.ObjectType,
			o.ObjectGUID,
		})
	}

	var b bytes.Buffer
	w := csv.NewWriter(&b)
	if err := w.WriteAll(rows); err != nil {
		return nil, fmt.Errorf("writing ContentsofExportPackage csv: %w", err)
	}
	return b.Bytes(), nil
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

// typePriority controls sort order for publish-mode dependency output (0 = unknown/sort last).
// Order reflects the publish dependency chain: connectors must exist before connections,
// connections before processes, etc.
var typePriority = map[string]int{
	"AI_SERVICE_CONNECTOR": 1,
	"AI_CONNECTION":        2,
	"PROCESS":              3,
	"GUIDE":                4,
	"TASKFLOW":             5,
}

// exportMetadata represents the exportMetadata.v2.json file in an IICS export package.
type exportMetadata struct {
	Name            string           `json:"name"`
	SourceOrgID     string           `json:"sourceOrgId"`
	SourceOrgName   string           `json:"sourceOrgName"`
	Tags            []exportTag      `json:"tags,omitempty"`
	ExportedObjects []exportedObject `json:"exportedObjects"`
}

type exportTag struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

// exportedObject is one asset record in exportMetadata.v2.json.
type exportedObject struct {
	ObjectGUID   string          `json:"objectGuid"`
	ObjectName   string          `json:"objectName"`
	ObjectType   string          `json:"objectType"`
	Path         string          `json:"path"`
	ProviderName json.RawMessage `json:"providerName,omitempty"`
	Metadata     json.RawMessage `json:"metadata"`
}

func exportedObjectsToManifestLog(objects []exportedObject) []release.ManifestLogAsset {
	rows := make([]release.ManifestLogAsset, 0, len(objects))
	for _, object := range objects {
		rows = append(rows, release.ManifestLogAsset{
			ID:   object.ObjectGUID,
			Type: object.ObjectType,
			Path: object.Path,
		})
	}
	return rows
}

// objectRefs extracts metadata.objectRefs while preserving all other metadata fields.
func (o exportedObject) objectRefs() []string {
	metadata := bytes.TrimSpace(o.Metadata)
	if len(metadata) == 0 || bytes.Equal(metadata, []byte("null")) {
		return nil
	}
	var m struct {
		ObjectRefs []string `json:"objectRefs"`
	}
	if err := json.Unmarshal(metadata, &m); err != nil {
		return nil
	}
	return append([]string(nil), m.ObjectRefs...)
}

// setObjectRefs updates metadata.objectRefs without dropping other metadata fields.
func (o *exportedObject) setObjectRefs(refs []string) error {
	metadata := bytes.TrimSpace(o.Metadata)
	m := make(map[string]json.RawMessage)
	if len(metadata) > 0 && !bytes.Equal(metadata, []byte("null")) {
		if err := json.Unmarshal(metadata, &m); err != nil {
			return fmt.Errorf("parsing metadata for %s: %w", o.ObjectGUID, err)
		}
	}
	rawRefs, err := json.Marshal(refs)
	if err != nil {
		return fmt.Errorf("serializing metadata refs for %s: %w", o.ObjectGUID, err)
	}
	m["objectRefs"] = rawRefs
	updated, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("serializing metadata for %s: %w", o.ObjectGUID, err)
	}
	o.Metadata = updated
	return nil
}

// dependencyItem is one row in the dependency output.
type dependencyItem struct {
	ID         string `json:"id,omitempty"`
	Path       string `json:"path"`
	Type       string `json:"type"`
	Location   string `json:"location"`
	Dependency string `json:"dependency"`
	Status     string `json:"status,omitempty"`
	Warning    string `json:"warning,omitempty"`
}

// reportItem is one row in multi-profile report output (--report flag).
type reportItem struct {
	ID         string                   `json:"id"`
	Path       string                   `json:"path"`
	Type       string                   `json:"type"`
	Location   string                   `json:"location"`
	Dependency string                   `json:"dependency"`
	Profiles   map[string]profileResult `json:"profiles"`
}

// profileResult holds the validation result for one profile.
type profileResult struct {
	Status  string `json:"status"`
	Warning string `json:"warning,omitempty"`
}

// depField returns the named field of a dependencyItem as a string for sorting.
func depField(item dependencyItem, field string) string {
	switch field {
	case "path":
		return item.Path
	case "type":
		return item.Type
	case "status":
		return item.Status
	case "warning":
		return item.Warning
	case "location":
		return item.Location
	case "dependency":
		return item.Dependency
	default:
		return ""
	}
}

// makeProfileStatusFunc returns a column Func that color-codes a per-profile status cell.
func makeProfileStatusFunc(profileKey string) func(interface{}) string {
	return func(v interface{}) string {
		row, _ := v.(map[string]interface{})
		status, _ := row["status_"+profileKey].(string)
		if noColor {
			return status
		}
		switch status {
		case "found":
			return lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true).Render(status)
		case "missing":
			return lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Render(status)
		default:
			return lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render(status)
		}
	}
}

// validateMultiProfile validates deps against multiple profiles in parallel.
// Returns a map of profileName -> validated copy of deps.
func validateMultiProfile(ctx context.Context, profiles []string, deps []dependencyItem) (map[string][]dependencyItem, error) {
	results := make(map[string][]dependencyItem, len(profiles))
	errs := make([]error, len(profiles))
	var mu sync.Mutex
	var wg sync.WaitGroup

	for idx, prof := range profiles {
		wg.Add(1)
		go func(i int, profileName string) {
			defer wg.Done()
			depsCopy := make([]dependencyItem, len(deps))
			copy(depsCopy, deps)
			slog.Info("report: starting validation", "profile", profileName, "total", len(depsCopy))
			if err := validateTargetDependencies(ctx, profileName, depsCopy); err != nil {
				errs[i] = fmt.Errorf("profile %q: %w", profileName, err)
				return
			}
			mu.Lock()
			results[profileName] = depsCopy
			mu.Unlock()
		}(idx, prof)
	}

	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}
	return results, nil
}

// buildReportRows builds output rows for multi-profile report mode.
// Returns table rows ([]map[string]interface{}) and JSON rows ([]reportItem).
func buildReportRows(deps []dependencyItem, profiles []string, profileResults map[string][]dependencyItem) ([]map[string]interface{}, []reportItem) {
	tableRows := make([]map[string]interface{}, len(deps))
	jsonRows := make([]reportItem, len(deps))

	for i, dep := range deps {
		id := dep.Location
		row := map[string]interface{}{
			"id":         id,
			"path":       dep.Path,
			"type":       dep.Type,
			"location":   dep.Location,
			"dependency": dep.Dependency,
		}
		ri := reportItem{
			ID:         id,
			Path:       dep.Path,
			Type:       dep.Type,
			Location:   dep.Location,
			Dependency: dep.Dependency,
			Profiles:   make(map[string]profileResult, len(profiles)),
		}
		for _, prof := range profiles {
			profDeps := profileResults[prof]
			var status, warning string
			if i < len(profDeps) {
				status = profDeps[i].Status
				warning = profDeps[i].Warning
			}
			key := strings.ReplaceAll(prof, "-", "_")
			row["status_"+key] = status
			row["warning_"+key] = warning
			ri.Profiles[prof] = profileResult{Status: status, Warning: warning}
		}
		tableRows[i] = row
		jsonRows[i] = ri
	}
	return tableRows, jsonRows
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

func readExportChecksumEntries(filePath, workspace string) (map[string]bool, error) {
	if workspace != "" {
		data, err := os.ReadFile(filepath.Join(workspace, "exportPackage.chksum"))
		if err != nil {
			return nil, fmt.Errorf("reading exportPackage.chksum: %w", err)
		}
		return dependencies.ParseChecksumEntries(string(data)), nil
	}
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening package file: %w", err)
	}
	defer func() { _ = r.Close() }()
	for _, f := range r.File {
		if f.Name == "exportPackage.chksum" {
			rc, oErr := f.Open()
			if oErr != nil {
				return nil, fmt.Errorf("opening exportPackage.chksum in ZIP: %w", oErr)
			}
			data, rErr := io.ReadAll(rc)
			_ = rc.Close()
			if rErr != nil {
				return nil, fmt.Errorf("reading exportPackage.chksum from ZIP: %w", rErr)
			}
			return dependencies.ParseChecksumEntries(string(data)), nil
		}
	}
	return nil, fmt.Errorf("exportPackage.chksum not found in package")
}

// resolveDependencies performs a BFS over package metadata plus source-org lookups
// to produce a flat dependency list and directed edges for graph rendering.
//
// Traversal continues until queue exhaustion. publishMode filters output rows but
// traversal still walks through all types so deeper publishable assets are found.
// excludeRe, when non-nil, is matched against "path/name.type" to skip both output
// and recursion through matching assets.
func resolveDependencies(
	ctx context.Context,
	meta *exportMetadata,
	c *client.Client,
	publishMode bool,
	orderBy string,
	excludeRe *regexp.Regexp,
	explicitInPackage map[string]bool,
) ([]dependencyItem, []dependencyEdge, error) {
	pkgMap := make(map[string]*exportedObject, len(meta.ExportedObjects))
	for i := range meta.ExportedObjects {
		pkgMap[meta.ExportedObjects[i].ObjectGUID] = &meta.ExportedObjects[i]
	}

	guidToMatchKey := make(map[string]string)
	resultMap := make(map[string]dependencyItem)
	var rawEdges [][2]string

	apiPath := func(p string) string {
		p = client.NormalizeLocationPath(p)
		if p == "" || p == "Explore" || p == "SYS" {
			return ""
		}
		return p
	}

	mkFromPkg := func(obj *exportedObject) string {
		base := apiPath(obj.Path)
		if base == "" {
			return obj.ObjectName + "." + obj.ObjectType
		}
		return base + "/" + obj.ObjectName + "." + obj.ObjectType
	}
	mkFromLookup := func(r client.LookupResult) string {
		return apiPath(r.Path) + "." + r.Type
	}

	shouldInclude := func(typ string) bool {
		return !publishMode || publishableTypes[typ]
	}
	shouldExclude := func(matchKey string) bool {
		return excludeRe != nil && excludeRe.MatchString(matchKey)
	}

	slog.Info("resolving dependencies: scanning package objects",
		"total", len(meta.ExportedObjects),
		"publishMode", publishMode,
	)

	queue := make([]string, 0, len(meta.ExportedObjects))
	for i := range meta.ExportedObjects {
		if meta.ExportedObjects[i].ObjectGUID != "" {
			queue = append(queue, meta.ExportedObjects[i].ObjectGUID)
		}
	}
	visited := make(map[string]bool, len(queue))
	externalResolved := make(map[string]bool)

	for len(queue) > 0 {
		current := queue
		queue = nil

		pkgGUIDs := make([]string, 0, len(current))
		externalGUIDs := make([]string, 0, len(current))

		for _, guid := range current {
			if guid == "" || visited[guid] {
				continue
			}
			visited[guid] = true
			if _, ok := pkgMap[guid]; ok {
				pkgGUIDs = append(pkgGUIDs, guid)
			} else {
				externalGUIDs = append(externalGUIDs, guid)
			}
		}

		for _, guid := range pkgGUIDs {
			obj := pkgMap[guid]
			mk := mkFromPkg(obj)
			guidToMatchKey[guid] = mk

			if shouldExclude(mk) {
				continue
			}

			if shouldInclude(obj.ObjectType) {
				base := apiPath(obj.Path)
				fullPath := obj.ObjectName
				if base != "" {
					fullPath = base + "/" + obj.ObjectName
				}
				depClass := "transitive"
				if explicitInPackage[guid] {
					depClass = "explicit"
				}
				resultMap[mk] = dependencyItem{
					Path:       fullPath,
					Type:       obj.ObjectType,
					Location:   client.BuildLocation(fullPath, obj.ObjectType),
					Dependency: depClass,
				}
			}

			for _, refGUID := range obj.objectRefs() {
				rawEdges = append(rawEdges, [2]string{guid, refGUID})
				if refGUID != "" && !visited[refGUID] {
					queue = append(queue, refGUID)
				}
			}
		}

		if c == nil || len(externalGUIDs) == 0 {
			continue
		}
		seedExternal := make([]string, 0, len(externalGUIDs))
		for _, guid := range externalGUIDs {
			if !externalResolved[guid] {
				seedExternal = append(seedExternal, guid)
			}
		}
		if len(seedExternal) == 0 {
			continue
		}

		graph, err := dependencies.TraverseByIDs(ctx, c, seedExternal, "uses", 0, 0)
		if err != nil {
			return nil, nil, fmt.Errorf("resolving transitive external dependencies: %w", err)
		}
		for id, n := range graph.Nodes {
			externalResolved[id] = true
			if id == "" || n.Type == "" {
				continue
			}
			result := client.LookupResult{ID: n.ID, Path: n.Path, Type: n.Type}
			mk := mkFromLookup(result)
			guidToMatchKey[id] = mk
			if shouldExclude(mk) {
				continue
			}
			if shouldInclude(n.Type) {
				pathOnly := apiPath(n.Path)
				resultMap[mk] = dependencyItem{
					Path:       pathOnly,
					Type:       n.Type,
					Location:   client.BuildLocation(pathOnly, n.Type),
					Dependency: "transitive",
				}
			}
		}
		for _, e := range graph.Edges {
			rawEdges = append(rawEdges, [2]string{e.FromID, e.ToID})
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
		if orderBy != "" {
			vi, vj := depField(items[i], orderBy), depField(items[j], orderBy)
			if vi != vj {
				return vi < vj
			}
			return items[i].Path < items[j].Path
		}
		if publishMode {
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

	var targetOrgName string
	tc.SetOnLoginSuccess(func(resp *client.LoginResponse) {
		targetOrgName = resp.UserInfo.OrgName
	})

	start := time.Now()
	slog.Info("validating dependencies against target org",
		"profile", targetProfileName,
		"count", len(deps),
	)

	for i := range deps {
		d := &deps[i]
		if i > 0 && i%50 == 0 {
			slog.Debug("validating dependencies progress",
				"profile", targetProfileName,
				"processed", i,
				"total", len(deps),
			)
		}
		slog.Debug("looking up dependency in target org",
			"profile", targetProfileName,
			"org", targetOrgName,
			"path", d.Path,
			"type", d.Type,
		)

		// CDI connections live under SYS/ and must be looked up via the V2
		// Connection API by name; the V3 Lookup API returns V3API_LookupError_012
		// for Connection type.
		if d.Type == "Connection" {
			name := d.Path
			if idx := strings.LastIndex(d.Path, "/"); idx >= 0 {
				name = d.Path[idx+1:]
			}
			_, connErr := tc.GetConnectionByName(ctx, name)
			if connErr == nil {
				d.Status = "found"
				slog.Debug("connection found in target org", "profile", targetProfileName, "org", targetOrgName, "name", name)
			} else {
				var apiErr *client.APIError
				if errors.As(connErr, &apiErr) && apiErr.IsNotFound() {
					d.Status = "missing"
					d.Warning = "connection not found in target org"
					slog.Debug("connection missing in target org", "profile", targetProfileName, "org", targetOrgName, "name", name)
				} else {
					d.Status = "unknown"
					d.Warning = fmt.Sprintf("lookup error: %v", connErr)
					slog.Warn("connection lookup failed", "profile", targetProfileName, "org", targetOrgName, "name", name, "error", connErr)
				}
			}
			continue
		}

		resp, lookupErr := tc.Lookup(ctx, []client.LookupObject{{Path: d.Path, Type: d.Type}})
		if lookupErr != nil {
			var apiErr *client.APIError
			// V3API_LookupError_012: the Lookup API nests error code under "error"."code"
			// so APIError.Code is empty; detect it by scanning the raw response body.
			if errors.As(lookupErr, &apiErr) &&
				bytes.Contains(apiErr.ResponseBody, []byte(`"V3API_LookupError_012"`)) {
				d.Status = "missing"
				d.Warning = "asset not found in target org"
				slog.Debug("dependency missing in target org", "profile", targetProfileName, "org", targetOrgName, "path", d.Path, "type", d.Type)
			} else {
				// Other errors (unsupported type, network, auth) - report but continue.
				d.Status = "unknown"
				d.Warning = fmt.Sprintf("lookup error: %v", lookupErr)
				slog.Warn("dependency lookup failed", "profile", targetProfileName, "org", targetOrgName, "path", d.Path, "type", d.Type, "error", lookupErr)
			}
			continue
		}

		// Any non-empty result means the object exists in the target org.
		// We intentionally do NOT match by path because some types (AgentGroup,
		// Connection) return a different path format than what we sent.
		if len(resp.Objects) > 0 {
			d.Status = "found"
			slog.Debug("dependency found in target org", "profile", targetProfileName, "org", targetOrgName, "path", d.Path, "type", d.Type)
		} else {
			d.Status = "missing"
			d.Warning = "asset not found in target org"
			slog.Debug("dependency missing in target org (empty result)", "profile", targetProfileName, "org", targetOrgName, "path", d.Path, "type", d.Type)
		}
	}

	var found, missing, unknown int
	for _, d := range deps {
		switch d.Status {
		case "found":
			found++
		case "missing":
			missing++
		default:
			unknown++
		}
	}
	slog.Info("dependencies validated",
		"profile", targetProfileName,
		"found", found,
		"missing", missing,
		"unknown", unknown,
		"elapsed", time.Since(start).Round(time.Millisecond).String(),
	)
	return nil
}

// applyFilter returns only the dependency items whose "location" matches pattern.
func applyFilter(deps []dependencyItem, pattern string) ([]dependencyItem, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	filtered := deps[:0:0]
	for _, d := range deps {
		if re.MatchString(d.Location) {
			filtered = append(filtered, d)
		}
	}
	return filtered, nil
}

func dependencyItemToFieldMap(item dependencyItem) map[string]interface{} {
	return map[string]interface{}{
		"id":         item.ID,
		"path":       item.Path,
		"type":       item.Type,
		"location":   item.Location,
		"dependency": item.Dependency,
		"status":     item.Status,
		"warning":    item.Warning,
	}
}

func dependencyItemsForOutputFile(items []dependencyItem, fields []string) []map[string]interface{} {
	rows := make([]map[string]interface{}, len(items))
	for i, item := range items {
		all := dependencyItemToFieldMap(item)
		row := make(map[string]interface{}, len(fields))
		for _, f := range fields {
			if v, ok := all[f]; ok {
				row[f] = v
			}
		}
		rows[i] = row
	}
	return rows
}

func writeOutputFile(rows interface{}, columns []output.Column, filePath, format string) error {
	fileFmt, err := output.ParseFormat(format)
	if err != nil {
		return fmt.Errorf("--output-file-format: %w", err)
	}
	fh, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("creating output file %s: %w", filePath, err)
	}
	defer func() { _ = fh.Close() }()

	fileFmtr := output.New(fileFmt, fh, output.TableStyle{NoColor: true})
	if err := fileFmtr.Format(rows, columns); err != nil {
		return fmt.Errorf("writing output file: %w", err)
	}
	return nil
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
		if item.Status == "missing" {
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
		orderBy        string
		reportProfiles []string
		excludePattern string
		excludeFile    string
		filterPattern  string
		targetProfile  string
		outputFile     string
		outputFileFmt  string
		outputFileRows string
	)

	validOrderByFields := []string{"path", "type", "status", "warning", "location", "dependency"}
	defaultOutputFileFields := "location,dependency,type,path,status,warning"

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
			for _, rp := range reportProfiles {
				if strings.HasPrefix(rp, "-") {
					return fmt.Errorf(
						"--report requires a profile name, not a flag (%q);\n"+
							"  correct usage: --report=dev,qa or --report dev,qa", rp)
				}
			}
			if targetProfile != "" && len(reportProfiles) > 0 {
				return fmt.Errorf("--target-profile and --report are mutually exclusive")
			}
			if publishMode && targetProfile == "" && len(reportProfiles) == 0 {
				return fmt.Errorf("--target-profile or --report is required when --publish is set")
			}
			if orderBy != "" {
				valid := false
				for _, f := range validOrderByFields {
					if f == orderBy {
						valid = true
						break
					}
				}
				if !valid {
					return fmt.Errorf("invalid --order-by value %q: must be one of %s", orderBy, strings.Join(validOrderByFields, ", "))
				}
			}

			// Compile exclude regex.
			var excludeRe *regexp.Regexp
			if excludePattern != "" || excludeFile != "" {
				patterns := make([]string, 0, 8)
				if excludePattern != "" {
					patterns = append(patterns, excludePattern)
				}
				if excludeFile != "" {
					compiledPatterns, err := release.LoadExcludePatterns(excludeFile)
					if err != nil {
						return err
					}
					for _, re := range compiledPatterns {
						patterns = append(patterns, re.String())
					}
				}
				combined := "(?:" + strings.Join(patterns, ")|(?:") + ")"
				var err error
				excludeRe, err = regexp.Compile(combined)
				if err != nil {
					return fmt.Errorf("invalid --exclude pattern: %w", err)
				}
			}

			// Read package metadata.
			meta, err := readExportMetadata(file, workspace)
			if err != nil {
				return err
			}
			checksumEntries, err := readExportChecksumEntries(file, workspace)
			if err != nil {
				return err
			}
			explicitInPackage := make(map[string]bool, len(meta.ExportedObjects))
			for i := range meta.ExportedObjects {
				obj := meta.ExportedObjects[i]
				explicitInPackage[obj.ObjectGUID] = dependencies.IsObjectChecksumBacked(
					obj.Path, obj.ObjectName, obj.ObjectType, checksumEntries,
				)
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
			deps, edges, err := resolveDependencies(ctx, meta, srcClient, publishMode, orderBy, excludeRe, explicitInPackage)
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

			// Multi-profile report mode.
			if len(reportProfiles) > 0 {
				if outputFmt == "mermaid" {
					return fmt.Errorf("--output mermaid is not supported with --report")
				}

				profileResults, reportErr := validateMultiProfile(ctx, reportProfiles, deps)
				if reportErr != nil {
					return reportErr
				}

				tableRows, jsonRows := buildReportRows(deps, reportProfiles, profileResults)

				f, fErr := getFormatter()
				if fErr != nil {
					return fErr
				}

				if outputFmt == "json" || outputFmt == "yaml" {
					return f.Format(jsonRows, nil)
				}

				cols := []output.Column{
					{Header: "LOCATION", Field: "location"},
					{Header: "DEPENDENCY", Field: "dependency", Width: 12},
				}
				for _, prof := range reportProfiles {
					key := strings.ReplaceAll(prof, "-", "_")
					cols = append(cols, output.Column{
						Header: fmt.Sprintf("STATUS (%s)", prof),
						Field:  "status_" + key,
						Func:   makeProfileStatusFunc(key),
					})
				}
				if outputFmt == "csv" {
					for _, prof := range reportProfiles {
						key := strings.ReplaceAll(prof, "-", "_")
						cols = append(cols, output.Column{
							Header: fmt.Sprintf("WARNING (%s)", prof),
							Field:  "warning_" + key,
						})
					}
				}
				if formatErr := f.Format(tableRows, cols); formatErr != nil {
					return formatErr
				}
				if outputFile != "" {
					return writeOutputFile(tableRows, cols, outputFile, outputFileFmt)
				}
				return nil
			}

			// Single-profile validation.
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
				{Header: "LOCATION", Field: "location"},
				{Header: "DEPENDENCY", Field: "dependency", Width: 12},
				{Header: "TYPE", Field: "type", Width: 20},
				{Header: "PATH", Field: "path", Width: 55},
			}
			if targetProfile != "" {
				columns = append(columns,
					output.Column{
						Header: fmt.Sprintf("STATUS (%s)", targetProfile),
						Field:  "status",
						Func:   targetStatusFunc,
					},
				)
				hasWarnings := false
				for _, d := range deps {
					if d.Warning != "" {
						hasWarnings = true
						break
					}
				}
				if hasWarnings {
					columns = append(columns,
						output.Column{Header: "WARNING", Field: "warning", Width: 40},
					)
				}
			}

			if err := f.Format(deps, columns); err != nil {
				return err
			}
			if outputFile != "" {
				fields := parseFields(outputFileRows)
				if len(fields) == 0 {
					fields = parseFields(defaultOutputFileFields)
				}
				fileRows := dependencyItemsForOutputFile(deps, fields)
				fileCols := make([]output.Column, 0, len(fields))
				for _, field := range fields {
					fileCols = append(fileCols, output.Column{
						Header: strings.ToUpper(field),
						Field:  field,
					})
				}
				return writeOutputFile(fileRows, fileCols, outputFile, outputFileFmt)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "path to IICS export ZIP package (mutually exclusive with --workspace)")
	cmd.Flags().StringVarP(&workspace, "workspace", "w", "", "path to expanded workspace directory (mutually exclusive with --file)")
	cmd.Flags().BoolVar(&publishMode, "publish", false, "restrict output to publishable types only; requires --target-profile or --report")
	cmd.Flags().StringSliceVar(&reportProfiles, "report", nil, "compare dependencies across one or more target profiles (mutually exclusive with --target-profile); accepts comma-separated values or repeated flags")
	cmd.Flags().StringVar(&orderBy, "order-by", "", "sort output by field: path, type, status, warning (overrides default sort)")
	cmd.Flags().StringVarP(&excludePattern, "exclude", "e", "", "regex matched against path/name.type to exclude assets from resolution")
	cmd.Flags().StringVar(&excludeFile, "exclude-file", "", "path to regex patterns file; each matching path/name.type is excluded from resolution")
	cmd.Flags().StringVar(&filterPattern, "filter", "", "regex matched against location ((Explore|SYS)/path.type) to filter final output (does not affect resolution)")
	cmd.Flags().StringVarP(&targetProfile, "target-profile", "t", "", "profile name for target org validation (mutually exclusive with --report)")
	cmd.Flags().StringVar(&outputFile, "output-file", "", "path to write output file")
	cmd.Flags().StringVar(&outputFileFmt, "output-file-format", "yaml", "format for output file: yaml, json, csv, table")
	cmd.Flags().StringVar(&outputFileRows, "output-file-fields", defaultOutputFileFields, "comma-separated fields for file output: location,dependency,type,path,status,warning")

	return cmd
}
