package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/output"
	"github.com/spf13/cobra"
)

func newImportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "import",
		Aliases: []string{"imp"},
		Short:   "Import organization assets",
		Long:    `Import assets into the IICS organization from a ZIP package.`,
	}

	cmd.AddCommand(newImportUploadCmd())
	cmd.AddCommand(newImportStartCmd())
	cmd.AddCommand(newImportStatusCmd())
	cmd.AddCommand(newImportRunCmd())
	cmd.AddCommand(newImportDownloadLogCmd())
	return cmd
}

func newImportUploadCmd() *cobra.Command {
	var file string

	cmd := &cobra.Command{
		Use:     "upload",
		Short:   "Upload a ZIP package for import",
		Example: `  iics import upload --file backup.zip`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if file == "" {
				return fmt.Errorf("--file is required")
			}

			f, err := os.Open(file)
			if err != nil {
				return fmt.Errorf("opening file %s: %w", file, err)
			}
			defer f.Close()

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Uploading %s...\n", file)

			resp, err := c.UploadImportPackage(context.Background(), file, f)
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Upload complete.\n")
			fmt.Fprintf(cmd.OutOrStdout(), "  Job ID: %s\n", resp.JobID)
			fmt.Fprintf(cmd.OutOrStdout(), "  Status: %s\n", resp.JobStatus.State)
			fmt.Fprintf(cmd.OutOrStdout(), "  Checksum valid: %v\n", resp.ChecksumValid)

			return nil
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "ZIP file to upload (required)")
	return cmd
}

func newImportStartCmd() *cobra.Command {
	var (
		id                 string
		name               string
		conflictResolution string
		fromFile           string
		wait               bool
	)

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start an import job",
		Example: `  iics import start --id <job-id> --name "import-job" --conflict-resolution OVERWRITE
  iics import start --id <job-id> --from-file import-spec.json --wait`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" {
				return fmt.Errorf("--id (job ID from upload) is required")
			}

			var req client.ImportStartRequest
			if fromFile != "" {
				data, err := os.ReadFile(fromFile)
				if err != nil {
					return fmt.Errorf("reading file %s: %w", fromFile, err)
				}
				if err := json.Unmarshal(data, &req); err != nil {
					return fmt.Errorf("parsing import spec: %w", err)
				}
			} else {
				if name == "" {
					name = "import-job"
				}
				if conflictResolution == "" {
					conflictResolution = "REUSE"
				}
				req = client.ImportStartRequest{
					Name: name,
					ImportSpecification: client.ImportSpecification{
						DefaultConflictResolution: conflictResolution,
					},
				}
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			job, err := c.StartImport(context.Background(), id, &req)
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Import job started: %s (status: %s)\n", job.ID, job.Status.State)

			if wait {
				fmt.Fprintln(cmd.OutOrStdout(), "Waiting for import to complete...")
				for job.Status.State == "IN_PROGRESS" || job.Status.State == "QUEUED" {
					time.Sleep(3 * time.Second)
					job, err = c.GetImportStatus(context.Background(), job.ID, false)
					if err != nil {
						return err
					}
					fmt.Fprintf(cmd.OutOrStdout(), "  Status: %s\n", job.Status.State)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Import completed: %s (status: %s)\n", job.ID, job.Status.State)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "job ID from upload (required)")
	cmd.Flags().StringVar(&name, "name", "", "import job name")
	cmd.Flags().StringVar(&conflictResolution, "conflict-resolution", "REUSE", "conflict resolution: REUSE or OVERWRITE")
	cmd.Flags().StringVar(&fromFile, "from-file", "", "JSON file with import specification")
	cmd.Flags().BoolVar(&wait, "wait", false, "wait for import to complete")

	return cmd
}

func newImportStatusCmd() *cobra.Command {
	var (
		id     string
		expand bool
	)

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Check import job status",
		Example: `  iics import status --id <job-id>
  iics import status --id <job-id> --expand`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" {
				return fmt.Errorf("--id is required")
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			job, err := c.GetImportStatus(context.Background(), id, expand)
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

	cmd.Flags().StringVar(&id, "id", "", "import job ID (required)")
	cmd.Flags().BoolVar(&expand, "expand", false, "expand to show individual object status")

	return cmd
}

// newImportRunCmd creates the combined upload+start+poll import command.
func newImportRunCmd() *cobra.Command {
	var (
		zipFile            string
		name               string
		conflictResolution string
		fromFile           string
		pollingInterval    int
		maxWaitTime        int
		detailedPolling    bool
		printImportLog     bool
		expand             bool
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Upload, start, and wait for an import job to complete",
		Long: `Uploads a ZIP package, starts the import job, and polls until completion.
Always prints the final job summary and object list. Downloads and prints
the import log automatically on failure or when --print-import-log is set.`,
		Example: `  iics import run -z backup.zip
  iics import run -z backup.zip --conflict-resolution OVERWRITE --verbose
  iics import run -z backup.zip --from-file spec.json --detailed-polling
  iics import run -z backup.zip --max-wait-time 600 --polling-interval 15`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if zipFile == "" {
				return fmt.Errorf("--zip-file (-z) is required")
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			startWall := time.Now()

			// --- Step 1: Upload ---
			f, err := os.Open(zipFile)
			if err != nil {
				return fmt.Errorf("opening %s: %w", zipFile, err)
			}
			defer f.Close()

			if verbose {
				fmt.Fprintf(out, "[%s] Uploading %s...\n", ts(), zipFile)
			}

			uploadResp, err := c.UploadImportPackage(context.Background(), filepath.Base(zipFile), f)
			if err != nil {
				return fmt.Errorf("upload failed: %w", err)
			}

			if verbose {
				fmt.Fprintf(out, "[%s] Upload complete — job ID: %s, checksum valid: %v\n",
					ts(), uploadResp.JobID, uploadResp.ChecksumValid)
			}

			// --- Step 2: Build start request ---
			var startReq client.ImportStartRequest
			if fromFile != "" {
				data, err := os.ReadFile(fromFile)
				if err != nil {
					return fmt.Errorf("reading %s: %w", fromFile, err)
				}
				if err := json.Unmarshal(data, &startReq); err != nil {
					return fmt.Errorf("parsing import spec: %w", err)
				}
			} else {
				if name == "" {
					name = strings.TrimSuffix(filepath.Base(zipFile), ".zip")
				}
				startReq = client.ImportStartRequest{
					Name: name,
					ImportSpecification: client.ImportSpecification{
						DefaultConflictResolution: conflictResolution,
					},
				}
			}

			// --- Step 3: Start ---
			if verbose {
				fmt.Fprintf(out, "[%s] Starting import job \"%s\"...\n", ts(), startReq.Name)
			}

			job, err := c.StartImport(context.Background(), uploadResp.JobID, &startReq)
			if err != nil {
				return fmt.Errorf("start failed: %w", err)
			}

			if verbose {
				fmt.Fprintf(out, "[%s] Import job started — ID: %s, status: %s\n",
					ts(), job.ID, job.Status.State)
				if job.StartTime != "" {
					fmt.Fprintf(out, "[%s] Start time: %s\n", ts(), job.StartTime)
				}
			}

			// --- Step 4: Poll ---
			deadline := startWall.Add(time.Duration(maxWaitTime) * time.Second)
			interval := time.Duration(pollingInterval) * time.Second

			for isImportInProgress(job.Status.State) {
				if time.Now().After(deadline) {
					return fmt.Errorf("timed out after %ds waiting for import job %s (last status: %s)",
						maxWaitTime, job.ID, job.Status.State)
				}
				time.Sleep(interval)

				job, err = c.GetImportStatus(context.Background(), job.ID, detailedPolling || expand)
				if err != nil {
					return fmt.Errorf("polling status: %w", err)
				}

				elapsed := time.Since(startWall).Round(time.Second)
				if verbose {
					fmt.Fprintf(out, "[%s] Status: %-15s elapsed: %s\n", ts(), job.Status.State, elapsed)
				}

				if detailedPolling && len(job.Objects) > 0 {
					printImportObjects(cmd, job)
				}
			}

			elapsed := time.Since(startWall).Round(time.Second)
			if verbose {
				fmt.Fprintf(out, "[%s] Import finished — status: %s, total time: %s\n",
					ts(), job.Status.State, elapsed)
			}

			// --- Step 5: Final status output ---
			formatter, err := getFormatter()
			if err != nil {
				return err
			}

			// Fetch with full object list for final output
			finalJob, err := c.GetImportStatus(context.Background(), job.ID, true)
			if err != nil {
				finalJob = job // fall back to last polled state
			}

			fmt.Fprintln(out, "\nImport Summary:")
			summaryColumns := []output.Column{
				{Header: "ID", Field: "id", Width: 24},
				{Header: "NAME", Field: "name", Width: 25},
				{Header: "STATUS", Field: "status.state", Width: 15},
				{Header: "MESSAGE", Field: "status.message", Width: 40},
				{Header: "START TIME", Field: "startTime", Width: 22},
				{Header: "END TIME", Field: "endTime", Width: 22},
			}
			if err := formatter.Format(finalJob, summaryColumns); err != nil {
				return err
			}

			if len(finalJob.Objects) > 0 {
				fmt.Fprintln(out, "\nImported Objects:")
				printImportObjects(cmd, finalJob)
			}

			// --- Step 6: Import log ---
			needLog := printImportLog || isImportFailed(finalJob.Status.State)
			if needLog {
				fmt.Fprintln(out, "\nImport Log:")
				var logBuf bytes.Buffer
				if err := c.DownloadImportLog(context.Background(), finalJob.ID, &logBuf); err != nil {
					fmt.Fprintf(out, "  (could not download log: %v)\n", err)
				} else {
					fmt.Fprintln(out, logBuf.String())
				}
			}

			if isImportFailed(finalJob.Status.State) {
				return fmt.Errorf("import job %s ended with status: %s", finalJob.ID, finalJob.Status.State)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&zipFile, "zip-file", "z", "", "ZIP package to import (required)")
	cmd.Flags().StringVar(&name, "name", "", "import job name (default: zip filename without extension)")
	cmd.Flags().StringVar(&conflictResolution, "conflict-resolution", "REUSE", "conflict resolution strategy: REUSE or OVERWRITE")
	cmd.Flags().StringVar(&fromFile, "from-file", "", "JSON file with full import specification (overrides --name and --conflict-resolution)")
	cmd.Flags().IntVar(&pollingInterval, "polling-interval", 10, "seconds between status polls")
	cmd.Flags().IntVar(&maxWaitTime, "max-wait-time", 300, "maximum seconds to wait for completion")
	cmd.Flags().BoolVar(&detailedPolling, "detailed-polling", false, "print object list on every poll")
	cmd.Flags().BoolVar(&printImportLog, "print-import-log", false, "print import log after completion")
	cmd.Flags().BoolVar(&expand, "expand", false, "expand object list in final status output")

	return cmd
}

// newImportDownloadLogCmd creates the download-log subcommand.
func newImportDownloadLogCmd() *cobra.Command {
	var (
		id      string
		logPath string
		logName string
	)

	cmd := &cobra.Command{
		Use:   "download-log",
		Short: "Download the import job log to a file",
		Example: `  iics import download-log --id <job-id> --log-path ./logs
  iics import download-log --id <job-id> --log-path ./logs --log-name my-import.log`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" {
				return fmt.Errorf("--id is required")
			}
			if logPath == "" {
				return fmt.Errorf("--log-path is required")
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			// Resolve default filename from job metadata
			filename := logName
			if filename == "" {
				job, err := c.GetImportStatus(context.Background(), id, false)
				if err != nil {
					// Fall back to a safe default if we can't fetch status
					filename = fmt.Sprintf("import_%s.log", id)
				} else {
					filename = buildLogFileName(job.Name, job.ID, job.Status.State)
				}
			}

			if err := os.MkdirAll(logPath, 0o755); err != nil {
				return fmt.Errorf("creating log directory: %w", err)
			}

			dest := filepath.Join(logPath, filename)
			file, err := os.Create(dest)
			if err != nil {
				return fmt.Errorf("creating log file %s: %w", dest, err)
			}
			defer file.Close()

			if err := c.DownloadImportLog(context.Background(), id, file); err != nil {
				return fmt.Errorf("downloading log: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Import log saved to: %s\n", dest)
			return nil
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "import job ID (required)")
	cmd.Flags().StringVar(&logPath, "log-path", "", "directory to save the log file (required)")
	cmd.Flags().StringVar(&logName, "log-name", "", "log filename (default: <name>_<id>_<status>.log)")

	return cmd
}

// isImportInProgress returns true when the job state means still running.
func isImportInProgress(state string) bool {
	return state == "IN_PROGRESS" || state == "QUEUED" || state == "STARTING"
}

// isImportFailed returns true when the job ended in a failure state.
func isImportFailed(state string) bool {
	return state == "FAILED" || state == "ERROR"
}

// ts returns the current time formatted for progress output.
func ts() string {
	return time.Now().Format("15:04:05")
}

// buildLogFileName builds a default log filename from job metadata.
func buildLogFileName(name, id, state string) string {
	safe := strings.NewReplacer(" ", "_", "/", "_", "\\", "_").Replace(name)
	return fmt.Sprintf("%s_%s_%s.log", safe, id, state)
}

// printImportObjects renders the object list table for an import job.
func printImportObjects(cmd *cobra.Command, job *client.ImportJob) {
	formatter, err := getFormatter()
	if err != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "  (formatter error: %v)\n", err)
		return
	}
	objColumns := []output.Column{
		{Header: "ID", Field: "id", Width: 24},
		{Header: "NAME", Field: "name", Width: 35},
		{Header: "TYPE", Field: "type", Width: 15},
		{Header: "STATUS", Field: "status.state", Width: 15},
		{Header: "MESSAGE", Field: "status.message", Width: 40},
	}
	if err := formatter.Format(job.Objects, objColumns); err != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "  (render error: %v)\n", err)
	}
}
