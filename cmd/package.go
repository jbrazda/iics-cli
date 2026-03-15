package cmd

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

func newPackageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "package",
		Short: "Work with IICS export package files",
	}
	cmd.AddCommand(newPackageExpandCmd())
	cmd.AddCommand(newPackageCreateCmd())
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
