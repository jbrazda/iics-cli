package cmd

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/spf13/cobra"
)

func newUnpublishCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unpublish",
		Short: "Unpublish CAI assets from the runtime",
		Long:  `Unpublish Cloud Application Integration (CAI) assets from the IICS runtime.`,
	}
	cmd.AddCommand(newUnpublishStartCmd())
	cmd.AddCommand(newUnpublishStatusCmd())
	cmd.AddCommand(newUnpublishRunCmd())
	return cmd
}

func newUnpublishStartCmd() *cobra.Command {
	var (
		assets   []string
		fromFile string
		caiURL   string
		name     string
	)

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Submit an unpublish job (fire-and-forget)",
		Example: `  iics unpublish start --asset "Explore/Default/MyProcess.PROCESS.xml"
  iics unpublish start --from-file assets.txt
  iics objects list -q "type=='PROCESS'" -o csv | iics unpublish start`,
		RunE: func(cmd *cobra.Command, args []string) error {
			paths, err := resolvePublishAssets(assets, fromFile)
			if err != nil {
				return err
			}
			if len(paths) == 0 {
				return fmt.Errorf("no asset paths provided; use --asset, --from-file, or pipe from stdin")
			}
			paths = filterPublishablePaths(paths)
			paths = sortPathsForUnpublish(paths)

			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()
			out := cmd.OutOrStdout()

			batches := client.SplitIntoBatches(paths, client.PublishMaxBatchSize)
			if verbose && name != "" {
				slog.Info("unpublishing assets", "count", len(paths), "batches", len(batches), "name", name)
			} else if verbose {
				slog.Info("unpublishing assets", "count", len(paths), "batches", len(batches))
			}

			for i, batch := range batches {
				resp, err := c.StartUnpublish(ctx, caiURL, batch)
				if err != nil {
					return fmt.Errorf("batch %d: starting unpublish: %w", i+1, err)
				}
				_, _ = fmt.Fprintf(out, "Unpublish job started: %s (batch %d/%d, assets: %d)\n",
					resp.Data.ID, i+1, len(batches), len(batch))
			}
			return nil
		},
	}

	cmd.Flags().StringArrayVar(&assets, "asset", nil, "asset path(s) to unpublish (repeatable)")
	cmd.Flags().StringVar(&fromFile, "from-file", "", "file with asset paths (txt/json/csv); omit to read from stdin")
	cmd.Flags().StringVar(&caiURL, "cai-url", "", "CAI base URL override")
	cmd.Flags().StringVar(&name, "name", "", "optional job label for verbose output")
	return cmd
}

func newUnpublishStatusCmd() *cobra.Command {
	var (
		id     string
		caiURL string
		full   bool
	)

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Get the status of an unpublish job",
		Example: `  iics unpublish status --id <job-id>
  iics unpublish status --id <job-id> --full`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" {
				return fmt.Errorf("--id is required")
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			resp, err := c.GetUnpublishStatus(context.Background(), caiURL, id, full)
			if err != nil {
				return err
			}

			return printPublishSummary(cmd, "unpublish", resp, defaultItemFields, full)
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "unpublish job ID (required)")
	cmd.Flags().StringVar(&caiURL, "cai-url", "", "CAI base URL override")
	cmd.Flags().BoolVar(&full, "full", false, "fetch full job object including asset list")
	return cmd
}

func newUnpublishRunCmd() *cobra.Command {
	var (
		assets          []string
		fromFile        string
		caiURL          string
		name            string
		pollingInterval int
		maxWaitTime     int
		detailedPolling bool
		itemFields      string
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Submit, poll, and report an unpublish job",
		Long: `Resolves asset inputs, auto-batches into 199-asset chunks, submits each batch,
polls to completion, and prints a detailed summary.`,
		Example: `  iics unpublish run --asset "Explore/Default/MyProcess.PROCESS.xml"
  iics unpublish run --from-file assets.txt --verbose
  iics objects list -q "type=='PROCESS'" -o csv --output-fields location | iics unpublish run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			paths, err := resolvePublishAssets(assets, fromFile)
			if err != nil {
				return err
			}
			if len(paths) == 0 {
				return fmt.Errorf("no asset paths provided; use --asset, --from-file, or pipe from stdin")
			}
			paths = filterPublishablePaths(paths)
			paths = sortPathsForUnpublish(paths)

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			return runPublishOp(cmd, c, paths, caiURL, name, "unpublish", pollingInterval, maxWaitTime, detailedPolling,
				itemFields, c.StartUnpublish, c.GetUnpublishStatus)
		},
	}

	cmd.Flags().StringArrayVar(&assets, "asset", nil, "asset path(s) to unpublish (repeatable)")
	cmd.Flags().StringVar(&fromFile, "from-file", "", "file with asset paths (.txt/.csv/.json/.yaml); omit to read from stdin")
	cmd.Flags().StringVar(&caiURL, "cai-url", "", "CAI base URL override")
	cmd.Flags().StringVar(&name, "name", "", "optional job label")
	cmd.Flags().IntVar(&pollingInterval, "polling-interval", 10, "seconds between status polls")
	cmd.Flags().IntVar(&maxWaitTime, "max-wait-time", 300, "maximum seconds to wait for completion")
	cmd.Flags().BoolVar(&detailedPolling, "detailed-polling", false, "print item detail table on each poll (requires --verbose)")
	cmd.Flags().StringVar(&itemFields, "item-fields", defaultItemFields, "comma-separated item detail fields to display")
	return cmd
}
