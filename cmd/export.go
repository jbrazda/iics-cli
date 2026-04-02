package cmd

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/output"
	"github.com/spf13/cobra"
)

func newExportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export organization assets",
		Long:  `Export assets from the IICS organization to a ZIP package.`,
	}

	cmd.AddCommand(newExportCreateCmd())
	cmd.AddCommand(newExportStatusCmd())
	cmd.AddCommand(newExportDownloadCmd())
	cmd.AddCommand(newExportStartCmd())
	cmd.AddCommand(newExportRunCmd())
	return cmd
}

func newExportCreateCmd() *cobra.Command {
	var (
		name string
		ids  string
		deps bool
		wait bool
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Start an export job",
		Example: `  iics export create --name "backup" --ids id1,id2
  iics export create --name "full-backup" --ids id1,id2 --include-deps --wait`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if ids == "" {
				return fmt.Errorf("--ids is required")
			}

			idList := strings.Split(ids, ",")
			objects := make([]client.ExportObject, len(idList))
			for i, id := range idList {
				objects[i] = client.ExportObject{
					ID:                  strings.TrimSpace(id),
					IncludeDependencies: deps,
				}
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			req := &client.ExportRequest{
				Name:    name,
				Objects: objects,
			}

			job, err := c.CreateExport(context.Background(), req)
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Export job created: %s (status: %s)\n", job.ID, job.Status.State)

			if wait {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Waiting for export to complete...")
				for job.Status.State == "IN_PROGRESS" || job.Status.State == "QUEUED" {
					time.Sleep(3 * time.Second)
					job, err = c.GetExportStatus(context.Background(), job.ID, false)
					if err != nil {
						return err
					}
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Status: %s\n", job.Status.State)
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Export completed: %s (status: %s)\n", job.ID, job.Status.State)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "export job name (required)")
	cmd.Flags().StringVar(&ids, "ids", "", "comma-separated object IDs to export (required)")
	cmd.Flags().BoolVar(&deps, "include-deps", false, "include dependent objects")
	cmd.Flags().BoolVar(&wait, "wait", false, "wait for export to complete")

	return cmd
}

func newExportStatusCmd() *cobra.Command {
	var (
		id     string
		expand bool
	)

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Check export job status",
		Example: `  iics export status --id <job-id>
  iics export status --id <job-id> --expand`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" {
				return fmt.Errorf("--id is required")
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			job, err := c.GetExportStatus(context.Background(), id, expand)
			if err != nil {
				return err
			}

			f, err := getFormatter()
			if err != nil {
				return err
			}

			columns := []output.Column{
				{Header: "ID", Field: "id", Width: 24},
				{Header: "NAME", Field: "name", Width: 20},
				{Header: "STATE", Field: "status.state", Width: 15},
				{Header: "MESSAGE", Field: "status.message"},
			}

			return f.Format(job, columns)
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "export job ID (required)")
	cmd.Flags().BoolVar(&expand, "expand", false, "expand to show individual object status")

	return cmd
}

func newExportDownloadCmd() *cobra.Command {
	var (
		id         string
		outputFile string
	)

	cmd := &cobra.Command{
		Use:     "download",
		Short:   "Download export ZIP package",
		Example: `  iics export download --id <job-id> --output-file backup.zip`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" {
				return fmt.Errorf("--id is required")
			}
			if outputFile == "" {
				outputFile = fmt.Sprintf("export_%s.zip", id)
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			file, err := os.Create(outputFile)
			if err != nil {
				return fmt.Errorf("creating output file: %w", err)
			}
			defer func() { _ = file.Close() }()

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Downloading export package to %s...\n", outputFile)

			if err := c.DownloadExportPackage(context.Background(), id, file); err != nil {
				_ = os.Remove(outputFile)
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Download complete: %s\n", outputFile)
			return nil
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "export job ID (required)")
	cmd.Flags().StringVarP(&outputFile, "output-file", "f", "", "output file path (default: export_<id>.zip)")

	return cmd
}

// newExportStartCmd starts an export job from an artifacts list and returns the job ID.
// Performs steps 1-5: parse input → resolve IDs → start job.
func newExportStartCmd() *cobra.Command {
	var (
		artifactsFile       string
		name                string
		includeTags         bool
		excludeDependencies bool
	)

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start an export job from an artifacts list and return the job ID",
		Long:  `Reads an artifacts list, resolves object IDs via lookup if needed, and starts an export job.`,
		Example: `  iics export start --artifacts-file ./config/export_list.txt
  iics export start --artifacts-file ./config/export_list.json --include-tags
  iics objects list -q "location==ZZ_TEST_CLI" -o csv | iics export start --export-file-path ./out.zip`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()
			out := cmd.OutOrStdout()

			entries, err := readArtifacts(artifactsFile)
			if err != nil {
				return err
			}
			if verbose {
				slog.Info("artifacts read", "count", len(entries))
			}

			objects, enrichedEntries, err := resolveExportObjects(ctx, c, entries, !excludeDependencies, out)
			if err != nil {
				return err
			}

			jobName := name
			if jobName == "" {
				jobName = defaultExportName()
			}
			req := &client.ExportRequest{
				Name:    jobName,
				Objects: objects,
			}

			if verbose {
				stderr := cmd.ErrOrStderr()
				slog.Info("starting export", "name", jobName, "objects", len(objects), "includeDeps", !excludeDependencies)
				_, _ = fmt.Fprintf(stderr, "Objects to export:\n")
				printArtifactTable(enrichedEntries, stderr)
			}

			job, err := c.StartExport(ctx, req, client.ExportCreateOptions{IncludeTags: includeTags})
			if err != nil {
				return fmt.Errorf("starting export: %w", err)
			}

			_, _ = fmt.Fprintf(out, "Export job started: %s (status: %s)\n", job.ID, job.Status.State)
			return nil
		},
	}

	cmd.Flags().StringVar(&artifactsFile, "artifacts-file", "", "file with artifacts list (.txt/.json/.yaml/.csv); omit to read from stdin")
	cmd.Flags().StringVarP(&name, "name", "n", "", "export job name (default: iics-cli(version) yyyy-mm-dd hh-mm-ss)")
	cmd.Flags().BoolVar(&includeTags, "include-tags", false, "export tag information with assets")
	cmd.Flags().BoolVar(&excludeDependencies, "exclude-dependencies", false, "exclude dependent objects from export")

	return cmd
}

// newExportRunCmd runs the full export pipeline: resolve objects → start → poll → download.
func newExportRunCmd() *cobra.Command {
	var (
		artifactsFile       string
		name                string
		exportFilePath      string
		pollingInterval     int
		maxWaitTime         int
		printFileContents   bool
		expandStatus        bool
		includeTags         bool
		excludeDependencies bool
		downloadExportLog   string
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run a complete export: resolve objects, start job, wait, and download",
		Long: `Reads an artifacts list, resolves object IDs, starts an export job, polls until
completion, and downloads the ZIP package. Always prints the job summary.`,
		Example: `  iics export run --artifacts-file ./config/export_list.txt --export-file-path ./backup.zip
  iics export run --artifacts-file ./config/export_list.json --export-file-path ./backup.zip --expand-status --verbose
  iics objects list -q "location==ZZ_TEST_CLI" -o csv | iics export run --export-file-path ./backup.zip`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if exportFilePath == "" {
				return fmt.Errorf("--export-file-path is required")
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()
			out := cmd.OutOrStdout()
			startWall := time.Now()

			// Step 1-2: Read input and resolve artifact IDs.
			entries, err := readArtifacts(artifactsFile)
			if err != nil {
				return err
			}
			if verbose {
				slog.Info("artifacts read", "count", len(entries))
			}

			objects, enrichedEntries, err := resolveExportObjects(ctx, c, entries, !excludeDependencies, out)
			if err != nil {
				return err
			}

			// Step 3-4: Construct the export request.
			jobName := name
			if jobName == "" {
				jobName = defaultExportName()
			}
			req := &client.ExportRequest{
				Name:    jobName,
				Objects: objects,
			}
			if verbose {
				stderr := cmd.ErrOrStderr()
				slog.Info("starting export", "name", req.Name, "objects", len(req.Objects), "includeDeps", !excludeDependencies)
				_, _ = fmt.Fprintf(stderr, "Objects to export:\n")
				printArtifactTable(enrichedEntries, stderr)
			}

			// Step 5: Start export job.
			job, err := c.StartExport(ctx, req, client.ExportCreateOptions{IncludeTags: includeTags})
			if err != nil {
				return fmt.Errorf("starting export: %w", err)
			}
			_, _ = fmt.Fprintf(out, "Export job started: %s (status: %s)\n", job.ID, job.Status.State)

			// Step 6: Poll until complete or max-wait exceeded.
			deadline := startWall.Add(time.Duration(maxWaitTime) * time.Second)
			interval := time.Duration(pollingInterval) * time.Second

			for isExportInProgress(job.Status.State) {
				if time.Now().After(deadline) {
					return fmt.Errorf("timed out after %ds waiting for export job %s (last status: %s)",
						maxWaitTime, job.ID, job.Status.State)
				}
				time.Sleep(interval)

				job, err = c.GetExportStatus(ctx, job.ID, false)
				if err != nil {
					return fmt.Errorf("polling export status: %w", err)
				}
				if verbose {
					slog.Info("export status", "state", job.Status.State, "elapsed", time.Since(startWall).Round(time.Second).String())
				}
			}

			elapsed := time.Since(startWall).Round(time.Second)
			if verbose {
				slog.Info("export complete", "state", job.Status.State, "elapsed", elapsed.String())
			}

			// Fetch final status with expanded objects if requested.
			finalJob := job
			if expandStatus {
				var expanded *client.ExportJob
				expanded, err = c.GetExportStatus(ctx, job.ID, true)
				if err == nil {
					finalJob = expanded
				}
			}

			// Print summary.
			_, _ = fmt.Fprintf(out, "\nExport Summary:\n")
			_, _ = fmt.Fprintf(out, "  Job ID:  %s\n", finalJob.ID)
			_, _ = fmt.Fprintf(out, "  Name:    %s\n", finalJob.Name)
			_, _ = fmt.Fprintf(out, "  Status:  %s\n", finalJob.Status.State)
			if finalJob.Status.Message != "" {
				_, _ = fmt.Fprintf(out, "  Message: %s\n", finalJob.Status.Message)
			}

			if expandStatus && len(finalJob.Objects) > 0 {
				_, _ = fmt.Fprintln(out, "\nExported Objects:")
				printExportObjects(cmd, finalJob)
			}

			if isExportFailed(finalJob.Status.State) {
				return fmt.Errorf("export job %s failed with status: %s", finalJob.ID, finalJob.Status.State)
			}

			// Step 7: Download the export package.
			if verbose {
				slog.Info("downloading export package", "path", exportFilePath)
			}

			zipFile, err := os.Create(exportFilePath)
			if err != nil {
				return fmt.Errorf("creating export file %s: %w", exportFilePath, err)
			}
			defer func() { _ = zipFile.Close() }()

			if err := c.DownloadExportPackage(ctx, finalJob.ID, zipFile); err != nil {
				_ = os.Remove(exportFilePath)
				return fmt.Errorf("downloading export package: %w", err)
			}

			info, statErr := zipFile.Stat()
			if statErr == nil {
				_, _ = fmt.Fprintf(out, "Downloaded: %s (%d bytes)\n", exportFilePath, info.Size())
			} else {
				_, _ = fmt.Fprintf(out, "Downloaded: %s\n", exportFilePath)
			}

			// Step 8: Optionally print zip contents.
			if printFileContents || verbose {
				if err := printZipContents(exportFilePath, out); err != nil {
					_, _ = fmt.Fprintf(out, "  (could not list zip contents: %v)\n", err)
				}
			}

			// Optional: Download export log.
			if cmd.Flags().Changed("download-export-log") {
				logPath := downloadExportLog
				if logPath == "" {
					ext := filepath.Ext(exportFilePath)
					logPath = strings.TrimSuffix(exportFilePath, ext) + ".log"
				}
				if verbose {
					slog.Info("downloading export log", "path", logPath)
				}
				logFile, err := os.Create(logPath)
				if err != nil {
					_, _ = fmt.Fprintf(out, "Warning: could not create log file %s: %v\n", logPath, err)
				} else {
					defer func() { _ = logFile.Close() }()
					if err := c.DownloadExportLog(ctx, finalJob.ID, logFile); err != nil {
						_, _ = fmt.Fprintf(out, "Warning: could not download export log: %v\n", err)
					} else {
						_, _ = fmt.Fprintf(out, "Export log saved to: %s\n", logPath)
					}
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&artifactsFile, "artifacts-file", "", "file with artifacts list (.txt/.json/.yaml/.csv); omit to read from stdin")
	cmd.Flags().StringVarP(&name, "name", "n", "", "export job name (default: iics-cli(version) yyyy-mm-dd hh-mm-ss)")
	cmd.Flags().StringVar(&exportFilePath, "export-file-path", "", "output ZIP file path (required)")
	cmd.Flags().IntVar(&pollingInterval, "polling-interval", 10, "seconds between status polls")
	cmd.Flags().IntVar(&maxWaitTime, "max-wait-time", 300, "maximum seconds to wait for completion")
	cmd.Flags().BoolVar(&printFileContents, "print-file-contents", false, "print ZIP file contents listing after download")
	cmd.Flags().BoolVar(&expandStatus, "expand-status", false, "fetch and print the exported object list after completion")
	cmd.Flags().BoolVar(&includeTags, "include-tags", false, "export tag information with assets")
	cmd.Flags().BoolVar(&excludeDependencies, "exclude-dependencies", false, "exclude dependent objects from export")
	cmd.Flags().StringVar(&downloadExportLog, "download-export-log", "", "download export log (default path: <export-file-path>.log)")

	return cmd
}

// isExportInProgress returns true when the job state is still running.
func isExportInProgress(state string) bool {
	return state == "IN_PROGRESS" || state == "QUEUED" || state == "STARTING"
}

// isExportFailed returns true when the job ended in a failure state.
func isExportFailed(state string) bool {
	return state == "FAILED" || state == "ERROR"
}

// defaultExportName returns a default export job name with version and timestamp.
func defaultExportName() string {
	return fmt.Sprintf("iics-cli(version:%s) %s", versionStr, time.Now().Format("2006-01-02 15-04-05"))
}

// readArtifacts reads artifact entries from a file or stdin.
// If artifactsFile is "" or "-", reads from stdin with auto-detected format.
func readArtifacts(artifactsFile string) ([]client.ArtifactEntry, error) {
	if artifactsFile == "" || artifactsFile == "-" {
		return readArtifactsFromStdin()
	}
	return client.ParseArtifactsFile(artifactsFile)
}

// readArtifactsFromStdin reads all of stdin and auto-detects the format.
func readArtifactsFromStdin() ([]client.ArtifactEntry, error) {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, fmt.Errorf("reading stdin: %w", err)
	}
	format := detectDataFormat(data)
	return client.ParseArtifactsReader(bytes.NewReader(data), format)
}

// detectDataFormat sniffs the first bytes to decide txt/json/csv.
func detectDataFormat(data []byte) string {
	preview := strings.TrimSpace(string(data))
	if len(preview) == 0 {
		return "txt"
	}
	if preview[0] == '{' || preview[0] == '[' {
		return "json"
	}
	// If the first line contains commas it's likely CSV output from objects list.
	firstLine := preview
	if idx := strings.IndexByte(preview, '\n'); idx >= 0 {
		firstLine = preview[:idx]
	}
	if strings.Contains(firstLine, ",") {
		return "csv"
	}
	return "txt"
}

// resolveExportObjects converts ArtifactEntries to ExportObjects, looking up IDs as needed.
// It returns both the ExportObject slice and an enriched copy of entries with resolved
// ID, Path, and Type back-filled from lookup results.
func resolveExportObjects(ctx context.Context, c *client.Client, entries []client.ArtifactEntry, includeDeps bool, out io.Writer) ([]client.ExportObject, []client.ArtifactEntry, error) {
	// Work on a copy so callers see enriched data without aliasing surprises.
	enriched := make([]client.ArtifactEntry, len(entries))
	copy(enriched, entries)

	// Collect entries that require lookup (any entry missing an ID).
	var lookupObjs []client.LookupObject
	var lookupOrigIdx []int

	for i, e := range enriched {
		if e.ID == "" {
			lookupObjs = append(lookupObjs, client.LookupObject{Path: e.Path, Type: e.Type})
			lookupOrigIdx = append(lookupOrigIdx, i)
		}
	}

	// Perform lookup for entries without IDs.
	if len(lookupObjs) > 0 {
		if verbose {
			slog.Info("looking up object IDs", "count", len(lookupObjs))
		}
		resp, err := c.Lookup(ctx, lookupObjs)
		if err != nil {
			return nil, nil, fmt.Errorf("looking up object IDs: %w", err)
		}
		for i, result := range resp.Objects {
			if i < len(lookupOrigIdx) {
				origIdx := lookupOrigIdx[i]
				enriched[origIdx].ID = result.ID
				if enriched[origIdx].Path == "" {
					enriched[origIdx].Path = result.Path
				}
				if enriched[origIdx].Type == "" {
					enriched[origIdx].Type = result.Type
				}
			}
		}
		if verbose {
			slog.Info("lookup complete", "resolved", len(resp.Objects))
		}
	}

	// Build ExportObject list preserving original order.
	objects := make([]client.ExportObject, 0, len(enriched))
	for i, e := range enriched {
		if e.ID == "" {
			return nil, nil, fmt.Errorf("could not resolve ID for artifact %d (path=%s, type=%s)", i, e.Path, e.Type)
		}
		objects = append(objects, client.ExportObject{
			ID:                  e.ID,
			IncludeDependencies: includeDeps,
		})
	}
	return objects, enriched, nil
}

// printArtifactTable writes a three-column (ID, PATH, TYPE) table of artifact entries to w.
// Cells are blank when the field is not populated on an entry.
func printArtifactTable(entries []client.ArtifactEntry, w io.Writer) {
	f := output.New(output.FormatTable, w, output.TableStyle{})
	cols := []output.Column{
		{Header: "ID", Field: "ID"},
		{Header: "PATH", Field: "Path"},
		{Header: "TYPE", Field: "Type"},
	}
	_ = f.Format(entries, cols)
}

// printExportObjects renders the object list table for an export job.
func printExportObjects(cmd *cobra.Command, job *client.ExportJob) {
	formatter, err := getFormatter()
	if err != nil {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  (formatter error: %v)\n", err)
		return
	}
	columns := []output.Column{
		{Header: "ID", Field: "id", Width: 24},
		{Header: "NAME", Field: "name", Width: 35},
		{Header: "TYPE", Field: "type", Width: 15},
		{Header: "STATUS", Field: "status.state", Width: 15},
	}
	if err := formatter.Format(job.Objects, columns); err != nil {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  (render error: %v)\n", err)
	}
}

// printZipContents lists the files inside a ZIP archive.
func printZipContents(zipPath string, out io.Writer) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("opening zip %s: %w", zipPath, err)
	}
	defer func() { _ = r.Close() }()

	_, _ = fmt.Fprintf(out, "\nZIP contents (%d files):\n", len(r.File))
	for _, f := range r.File {
		_, _ = fmt.Fprintf(out, "  %s\n", f.Name)
	}
	return nil
}
