package cmd

import (
	"context"
	"fmt"
	"os"
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

			fmt.Fprintf(cmd.OutOrStdout(), "Export job created: %s (status: %s)\n", job.ID, job.Status.State)

			if wait {
				fmt.Fprintln(cmd.OutOrStdout(), "Waiting for export to complete...")
				for job.Status.State == "IN_PROGRESS" || job.Status.State == "QUEUED" {
					time.Sleep(3 * time.Second)
					job, err = c.GetExportStatus(context.Background(), job.ID, false)
					if err != nil {
						return err
					}
					fmt.Fprintf(cmd.OutOrStdout(), "  Status: %s\n", job.Status.State)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Export completed: %s (status: %s)\n", job.ID, job.Status.State)
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
			defer file.Close()

			fmt.Fprintf(cmd.OutOrStdout(), "Downloading export package to %s...\n", outputFile)

			if err := c.DownloadExportPackage(context.Background(), id, file); err != nil {
				os.Remove(outputFile)
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Download complete: %s\n", outputFile)
			return nil
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "export job ID (required)")
	cmd.Flags().StringVarP(&outputFile, "output-file", "f", "", "output file path (default: export_<id>.zip)")

	return cmd
}
