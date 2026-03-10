package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/jbrazda/iics-cli/internal/output"
	"github.com/spf13/cobra"
)

func newMeteringCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "metering",
		Short: "View metering and usage data",
	}
	cmd.AddCommand(newMeteringGetCmd())
	cmd.AddCommand(newMeteringDownloadCmd())
	return cmd
}

func newMeteringGetCmd() *cobra.Command {
	var (
		meteringType string
		startDate    string
		endDate      string
	)

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get metering data",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			data, err := c.GetMeteringData(context.Background(), meteringType, startDate, endDate)
			if err != nil {
				return err
			}
			f, err := getFormatter()
			if err != nil {
				return err
			}
			columns := []output.Column{
				{Header: "ID", Field: "id", Width: 24},
				{Header: "TYPE", Field: "type", Width: 15},
				{Header: "START", Field: "startDate", Width: 12},
				{Header: "END", Field: "endDate", Width: 12},
			}
			return f.Format(data, columns)
		},
	}

	cmd.Flags().StringVar(&meteringType, "type", "", "metering type")
	cmd.Flags().StringVar(&startDate, "start", "", "start date")
	cmd.Flags().StringVar(&endDate, "end", "", "end date")
	return cmd
}

func newMeteringDownloadCmd() *cobra.Command {
	var (
		id         string
		outputFile string
	)

	cmd := &cobra.Command{
		Use:   "download",
		Short: "Download metering report",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" {
				return fmt.Errorf("--id is required")
			}
			if outputFile == "" {
				outputFile = fmt.Sprintf("metering_%s.csv", id)
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

			if err := c.DownloadMeteringReport(context.Background(), id, file); err != nil {
				_ = os.Remove(outputFile)
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Report downloaded: %s\n", outputFile)
			return nil
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "report ID (required)")
	cmd.Flags().StringVarP(&outputFile, "output-file", "f", "", "output file path")
	return cmd
}
