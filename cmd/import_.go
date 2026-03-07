package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
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
	return cmd
}

func newImportUploadCmd() *cobra.Command {
	var file string

	cmd := &cobra.Command{
		Use:   "upload",
		Short: "Upload a ZIP package for import",
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
