package cmd

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/output"
	"github.com/jbrazda/iics-cli/internal/release"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newPublishCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "publish",
		Short: "Publish CAI assets to the runtime",
		Long:  `Publish Cloud Application Integration (CAI) assets to the IICS runtime.`,
	}
	cmd.AddCommand(newPublishStartCmd())
	cmd.AddCommand(newPublishStatusCmd())
	cmd.AddCommand(newPublishRunCmd())
	return cmd
}

func newPublishStartCmd() *cobra.Command {
	var (
		assets   []string
		fromFile string
		caiURL   string
		name     string
	)

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Submit a publish job (fire-and-forget)",
		Example: `  iics publish start --asset "Explore/Default/MyProcess.PROCESS.xml"
  iics publish start --from-file assets.txt
  iics objects list -q "type=='PROCESS'" -o csv | iics publish start`,
		RunE: func(cmd *cobra.Command, args []string) error {
			paths, err := resolvePublishAssets(assets, fromFile)
			if err != nil {
				return err
			}
			if len(paths) == 0 {
				return fmt.Errorf("no asset paths provided; use --asset, --from-file, or pipe from stdin")
			}
			paths = filterPublishablePaths(paths)
			paths = sortPathsForPublish(paths)

			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()
			out := cmd.OutOrStdout()

			batches := client.SplitPublishBatches(paths, client.PublishMaxBatchSize)
			if verbose && name != "" {
				slog.Info("publishing assets", "count", len(paths), "batches", len(batches), "name", name)
			} else if verbose {
				slog.Info("publishing assets", "count", len(paths), "batches", len(batches))
			}

			for i, batch := range batches {
				resp, err := c.StartPublish(ctx, caiURL, batch.Paths)
				if err != nil {
					return fmt.Errorf("batch %d (%s): starting publish: %w", i+1, batch.Kind, err)
				}
				_, _ = fmt.Fprintf(out, "Publish job started: %s (batch %d/%d, group: %s, assets: %d)\n",
					resp.Data.ID, i+1, len(batches), batch.Kind, len(batch.Paths))
			}
			return nil
		},
	}

	cmd.Flags().StringArrayVar(&assets, "asset", nil, "asset path(s) to publish (repeatable)")
	cmd.Flags().StringVar(&fromFile, "from-file", "", "file with asset paths (txt/json/csv); omit to read from stdin")
	cmd.Flags().StringVar(&caiURL, "cai-url", "", "CAI base URL override")
	cmd.Flags().StringVar(&name, "name", "", "optional job label for verbose output")
	return cmd
}

func newPublishStatusCmd() *cobra.Command {
	var (
		id     string
		caiURL string
		full   bool
	)

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Get the status of a publish job",
		Example: `  iics publish status --id <job-id>
  iics publish status --id <job-id> --full`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" {
				return fmt.Errorf("--id is required")
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			resp, err := c.GetPublishStatus(context.Background(), caiURL, id, full)
			if err != nil {
				return err
			}

			return printPublishSummary(cmd, "publish", resp, defaultItemFields, full)
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "publish job ID (required)")
	cmd.Flags().StringVar(&caiURL, "cai-url", "", "CAI base URL override")
	cmd.Flags().BoolVar(&full, "full", false, "fetch full job object including asset list")
	return cmd
}

func newPublishRunCmd() *cobra.Command {
	var (
		assets          []string
		fromFile        string
		caiURL          string
		name            string
		pollingInterval int
		maxWaitTime     int
		detailedPolling bool
		itemFields      string
		logFile         string
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Submit, poll, and report a publish job",
		Long: `Resolves asset inputs, auto-batches into 199-asset chunks, submits each batch,
polls to completion, and prints a detailed summary.`,
		Example: `  iics publish run --asset "Explore/Default/MyProcess.PROCESS.xml"
  iics publish run --from-file assets.txt --verbose
  iics publish run --from-file assets.csv --polling-interval 15 --max-wait-time 600
  iics objects list -q "type=='PROCESS'" -o csv --output-fields location | iics publish run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			paths, err := resolvePublishAssets(assets, fromFile)
			if err != nil {
				return err
			}
			if len(paths) == 0 {
				return fmt.Errorf("no asset paths provided; use --asset, --from-file, or pipe from stdin")
			}
			paths = filterPublishablePaths(paths)
			paths = sortPathsForPublish(paths)

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			return runPublishOp(cmd, c, paths, caiURL, name, "publish", pollingInterval, maxWaitTime, detailedPolling,
				itemFields, logFile, c.StartPublish, c.GetPublishStatus)
		},
	}

	cmd.Flags().StringArrayVar(&assets, "asset", nil, "asset path(s) to publish (repeatable)")
	cmd.Flags().StringVar(&fromFile, "from-file", "", "file with asset paths (.txt/.csv/.json/.yaml); omit to read from stdin")
	cmd.Flags().StringVar(&caiURL, "cai-url", "", "CAI base URL override")
	cmd.Flags().StringVar(&name, "name", "", "optional job label")
	cmd.Flags().IntVar(&pollingInterval, "polling-interval", 10, "seconds between status polls")
	cmd.Flags().IntVar(&maxWaitTime, "max-wait-time", 300, "maximum seconds to wait for completion")
	cmd.Flags().BoolVar(&detailedPolling, "detailed-polling", false, "print item detail table on each poll (requires --verbose)")
	cmd.Flags().StringVar(&itemFields, "item-fields", defaultItemFields, "comma-separated item detail fields to display")
	bindManifestLogFlag(cmd, &logFile)
	return cmd
}

// batchResult captures the terminal state of a single batch after polling completes.
type batchResult struct {
	batchNum int
	kind     client.AssetBatchKind // CAI or TASKFLOW group this batch belongs to
	jobID    string
	resp     *client.PublishJobResponse // final terminal response
	timedOut bool
}

// batchPublished returns the count of successfully published assets in this batch.
func (b batchResult) batchPublished() int {
	return b.resp.Data.Attributes.JobStatusDetail.ItemStateSummary["SUCCESS"]
}

// batchErrors returns the count of non-SUCCESS assets in this batch.
func (b batchResult) batchErrors() int {
	total := 0
	for state, cnt := range b.resp.Data.Attributes.JobStatusDetail.ItemStateSummary {
		if state != "SUCCESS" {
			total += cnt
		}
	}
	return total
}

// runPublishOp is the shared run loop for publish and unpublish.
func runPublishOp(
	cmd *cobra.Command,
	c *client.Client,
	paths []string,
	caiURL, name, verb string,
	pollingInterval, maxWaitTime int,
	detailedPolling bool,
	itemFields string,
	logFile string,
	startFn func(context.Context, string, []string) (*client.PublishJobResponse, error),
	statusFn func(context.Context, string, string, bool) (*client.PublishJobResponse, error),
) error {
	ctx := context.Background()

	batches := client.SplitPublishBatches(paths, client.PublishMaxBatchSize)
	if verbose && name != "" {
		slog.Info(verb+"ing assets", "count", len(paths), "batches", len(batches), "name", name)
	} else if verbose {
		slog.Info(verb+"ing assets", "count", len(paths), "batches", len(batches))
	}

	interval := time.Duration(pollingInterval) * time.Second
	startWall := time.Now()

	var results []batchResult
	for batchIdx, batch := range batches {
		batchLabel := fmt.Sprintf("%d/%d", batchIdx+1, len(batches))

		resp, err := startFn(ctx, caiURL, batch.Paths)
		if err != nil {
			return fmt.Errorf("batch %d (%s): submitting: %w", batchIdx+1, batch.Kind, err)
		}
		jobID := resp.Data.ID
		slog.Info("batch submitted", "batch", batchLabel, "group", batch.Kind, "job", jobID, "assets", len(batch.Paths))

		// Poll until terminal or timeout.
		timedOut := false
		deadline := time.Now().Add(time.Duration(maxWaitTime) * time.Second)
		for !client.PublishIsTerminal(resp.Data.Attributes.JobState) {
			if time.Now().After(deadline) {
				slog.Warn("timed out waiting for job",
					"verb", verb, "job", jobID, "batch", batchLabel,
					"elapsed", fmt.Sprintf("%ds", maxWaitTime),
					"state", resp.Data.Attributes.JobState)
				timedOut = true
				break
			}
			time.Sleep(interval)

			resp, err = statusFn(ctx, caiURL, jobID, false)
			if err != nil {
				return fmt.Errorf("polling %s job %s: %w", verb, jobID, err)
			}

			elapsed := time.Since(startWall).Round(time.Second)
			attrs := resp.Data.Attributes
			slog.Info(verb+" progress",
				"processed", attrs.ProcessedCount,
				"total", attrs.TotalCount,
				"state", attrs.JobState,
				"elapsed", elapsed.String())

			if verbose && detailedPolling && len(attrs.ItemDetail) > 0 {
				printPublishItems(cmd, attrs.ItemDetail, itemFields)
			}
		}

		results = append(results, batchResult{
			batchNum: batchIdx + 1,
			kind:     batch.Kind,
			jobID:    jobID,
			resp:     resp,
			timedOut: timedOut,
		})
	}

	if len(results) == 0 {
		return nil
	}
	var err error
	if len(batches) == 1 {
		err = printPublishSummary(cmd, verb, results[0].resp, itemFields, verbose)
	} else {
		err = printMultiBatchSummary(cmd, verb, results, itemFields, verbose)
	}
	if err != nil {
		return err
	}
	if verb == "publish" {
		logEnabled, logPath := manifestLogPath(cmd, logFile)
		if logEnabled {
			appendManifestLogWarning(cmd, logPath, release.RenderPublishRunLog(release.PublishRunLog{
				Batches: publishBatchesToManifestLog(results),
			}))
		}
	}
	return nil
}

// titleCase uppercases the first letter of s.
func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func publishBatchesToManifestLog(results []batchResult) []release.PublishBatchLog {
	batches := make([]release.PublishBatchLog, 0, len(results))
	for _, result := range results {
		attrs := result.resp.Data.Attributes
		batch := release.PublishBatchLog{
			Batch:     result.batchNum,
			Group:     string(result.kind),
			JobID:     result.jobID,
			State:     attrs.JobState,
			StartDate: attrs.StartDate,
			EndDate:   attrs.EndDate,
			Duration:  publishItemDuration(attrs.StartDate, attrs.EndDate),
			Total:     attrs.TotalCount,
			Published: result.batchPublished(),
			Errors:    result.batchErrors(),
		}
		if result.timedOut {
			batch.State = "TIMED_OUT"
			batch.Errors++
		}
		for _, item := range result.resp.Data.Attributes.ItemDetail {
			batch.Items = append(batch.Items, release.PublishItemLog{
				Batch:     result.batchNum,
				Index:     item.ItemIndex,
				GUID:      item.ItemGUID,
				AssetPath: item.AssetPath,
				State:     item.ItemState,
				StartDate: item.ItemStartDate,
				EndDate:   item.ItemEndDate,
				Duration:  publishItemDuration(item.ItemStartDate, item.ItemEndDate),
				Detail:    item.ItemStatusDetail,
			})
		}
		batches = append(batches, batch)
	}
	return batches
}

// defaultItemFields is the default set of columns for the item detail table.
// itemStatusDetail is intentionally excluded; errors are shown in a separate section.
const defaultItemFields = "itemIndex,itemGUID,assetPath,itemState,itemStartDate,itemEndDate,duration"

// publishItemColumns defines all available columns for the per-asset item detail table.
var publishItemColumns = map[string]output.Column{
	"itemIndex":        {Header: "INDEX", Field: "itemIndex", Width: 6},
	"itemGUID":         {Header: "GUID", Field: "itemGUID", Width: 24},
	"assetPath":        {Header: "ASSET PATH", Field: "assetPath", Width: 60},
	"itemState":        {Header: "STATE", Field: "itemState", Width: 12},
	"itemStatusDetail": {Header: "DETAIL", Field: "itemStatusDetail", Width: 30},
	"itemStartDate":    {Header: "START DATE", Field: "itemStartDate", Width: 30},
	"itemEndDate":      {Header: "END DATE", Field: "itemEndDate", Width: 30},
	"duration":         {Header: "DURATION", Field: "duration", Width: 10},
}

// flatPublishItem is a display-ready struct with a pre-calculated duration field.
type flatPublishItem struct {
	ItemIndex        int    `json:"itemIndex"`
	ItemGUID         string `json:"itemGUID"`
	AssetPath        string `json:"assetPath"`
	ItemState        string `json:"itemState"`
	ItemStatusDetail string `json:"itemStatusDetail"`
	ItemStartDate    string `json:"itemStartDate"`
	ItemEndDate      string `json:"itemEndDate"`
	Duration         string `json:"duration"`
}

// publishItemDuration computes a human-readable elapsed time between two IICS date strings.
func publishItemDuration(start, end string) string {
	const layout = "2006-01-02T15:04:05.000-0700"
	if start == "" || end == "" {
		return ""
	}
	s, err := time.Parse(layout, start)
	if err != nil {
		return ""
	}
	e, err := time.Parse(layout, end)
	if err != nil {
		return ""
	}
	d := e.Sub(s)
	if d < 0 {
		d = -d
	}
	d = d.Round(time.Second)
	if d == 0 {
		return "< 1s"
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s2 := int(d.Seconds()) % 60
	switch {
	case h > 0:
		return fmt.Sprintf("%dh %dm %ds", h, m, s2)
	case m > 0:
		return fmt.Sprintf("%dm %ds", m, s2)
	default:
		return fmt.Sprintf("%ds", s2)
	}
}

// buildItemColumns returns the ordered output.Column slice from a comma-separated field list.
func buildItemColumns(fields string) []output.Column {
	var cols []output.Column
	for _, f := range strings.Split(fields, ",") {
		f = strings.TrimSpace(f)
		if col, ok := publishItemColumns[f]; ok {
			cols = append(cols, col)
		}
	}
	if len(cols) == 0 {
		for _, f := range strings.Split(defaultItemFields, ",") {
			if col, ok := publishItemColumns[f]; ok {
				cols = append(cols, col)
			}
		}
	}
	return cols
}

// printPublishItems renders the per-asset item detail table to stdout.
func printPublishItems(cmd *cobra.Command, items []client.PublishItemDetail, itemFields string) {
	if len(items) == 0 {
		return
	}
	flat := make([]flatPublishItem, 0, len(items))
	for _, it := range items {
		flat = append(flat, flatPublishItem{
			ItemIndex:        it.ItemIndex,
			ItemGUID:         it.ItemGUID,
			AssetPath:        it.AssetPath,
			ItemState:        it.ItemState,
			ItemStatusDetail: it.ItemStatusDetail,
			ItemStartDate:    it.ItemStartDate,
			ItemEndDate:      it.ItemEndDate,
			Duration:         publishItemDuration(it.ItemStartDate, it.ItemEndDate),
		})
	}
	cols := buildItemColumns(itemFields)
	f, err := getFormatter()
	if err != nil {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: formatter: %v\n", err)
		return
	}
	if err := f.Format(flat, cols); err != nil {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: format items: %v\n", err)
	}
}

// printPublishSummary renders a vertical summary table for the completed job.
// showItems controls whether the per-asset item detail table is printed when available.
func printPublishSummary(cmd *cobra.Command, verb string, resp *client.PublishJobResponse, itemFields string, showItems bool) error {
	attrs := resp.Data.Attributes
	successCount := attrs.JobStatusDetail.ItemStateSummary["SUCCESS"]
	errorCount := 0
	for state, cnt := range attrs.JobStatusDetail.ItemStateSummary {
		if state != "SUCCESS" {
			errorCount += cnt
		}
	}
	dur := publishItemDuration(attrs.StartDate, attrs.EndDate)
	rows := []map[string]interface{}{
		{"field": "Job ID", "value": resp.Data.ID},
		{"field": "State", "value": attrs.JobState},
		{"field": "Start Date", "value": attrs.StartDate},
		{"field": "End Date", "value": attrs.EndDate},
		{"field": "Duration", "value": dur},
		{"field": "Total", "value": attrs.TotalCount},
		{"field": "Published", "value": successCount},
		{"field": "Errors", "value": errorCount},
	}
	summCols := []output.Column{
		{Header: "FIELD", Field: "field", Width: 12},
		{Header: "VALUE", Field: "value"},
	}
	f, err := getFormatter()
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\n%s Summary:\n", titleCase(verb))
	if err := f.Format(rows, summCols); err != nil {
		return err
	}
	if showItems && len(attrs.ItemDetail) > 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\n%s Items:\n", titleCase(verb))
		printPublishItems(cmd, attrs.ItemDetail, itemFields)
		return printSingleBatchErrors(cmd, verb, attrs.ItemDetail)
	}
	return nil
}

// batchSummaryRow is one row in the horizontal multi-batch summary table.
type batchSummaryRow struct {
	Batch     string `json:"batch"`
	Group     string `json:"group"`
	JobID     string `json:"jobID"`
	State     string `json:"state"`
	Total     int    `json:"total"`
	Published int    `json:"published"`
	Errors    int    `json:"errors"`
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
	Duration  string `json:"duration"`
}

// flatBatchPublishItem is a display-ready struct for combined multi-batch item tables.
type flatBatchPublishItem struct {
	Batch         int    `json:"batch"`
	ItemIndex     int    `json:"itemIndex"`
	ItemGUID      string `json:"itemGUID"`
	AssetPath     string `json:"assetPath"`
	ItemState     string `json:"itemState"`
	ItemStartDate string `json:"itemStartDate"`
	ItemEndDate   string `json:"itemEndDate"`
	Duration      string `json:"duration"`
}

// publishErrorItem holds a failed item for the error summary section.
type publishErrorItem struct {
	Batch  int    `json:"batch"` // 0 = single-batch (column omitted)
	Index  int    `json:"index"`
	GUID   string `json:"guid"`
	Path   string `json:"path"`
	Detail string `json:"detail"`
}

// printMultiBatchSummary renders aggregated results for multi-batch publish/unpublish runs.
func printMultiBatchSummary(cmd *cobra.Command, verb string, results []batchResult, itemFields string, showItems bool) error {
	out := cmd.OutOrStdout()
	f, err := getFormatter()
	if err != nil {
		return err
	}

	// Build per-batch rows and aggregate totals.
	var rows []batchSummaryRow
	var totalTotal, totalPublished, totalErrors int
	var firstStart, lastEnd string
	var anyError bool
	var allItems []flatBatchPublishItem
	var errorItems []publishErrorItem

	for _, br := range results {
		attrs := br.resp.Data.Attributes
		pub := br.batchPublished()
		errs := br.batchErrors()
		dur := publishItemDuration(attrs.StartDate, attrs.EndDate)
		state := attrs.JobState
		if br.timedOut {
			state = "TIMEOUT"
		}
		if state != "SUCCESS" && state != "COMPLETED" {
			anyError = true
		}

		rows = append(rows, batchSummaryRow{
			Batch:     fmt.Sprintf("%d", br.batchNum),
			Group:     string(br.kind),
			JobID:     br.jobID,
			State:     state,
			Total:     attrs.TotalCount,
			Published: pub,
			Errors:    errs,
			StartDate: attrs.StartDate,
			EndDate:   attrs.EndDate,
			Duration:  dur,
		})

		totalTotal += attrs.TotalCount
		totalPublished += pub
		totalErrors += errs
		if firstStart == "" {
			firstStart = attrs.StartDate
		}
		lastEnd = attrs.EndDate

		// Collect items for verbose display.
		for _, it := range attrs.ItemDetail {
			fi := flatBatchPublishItem{
				Batch:         br.batchNum,
				ItemIndex:     it.ItemIndex,
				ItemGUID:      it.ItemGUID,
				AssetPath:     it.AssetPath,
				ItemState:     it.ItemState,
				ItemStartDate: it.ItemStartDate,
				ItemEndDate:   it.ItemEndDate,
				Duration:      publishItemDuration(it.ItemStartDate, it.ItemEndDate),
			}
			allItems = append(allItems, fi)
			if it.ItemState != "SUCCESS" && it.ItemStatusDetail != "" {
				errorItems = append(errorItems, publishErrorItem{
					Batch:  br.batchNum,
					Index:  it.ItemIndex,
					GUID:   it.ItemGUID,
					Path:   it.AssetPath,
					Detail: it.ItemStatusDetail,
				})
			}
		}
	}

	// Add TOTAL aggregate row.
	overallState := "SUCCESS"
	if anyError {
		overallState = "ERROR"
	}
	rows = append(rows, batchSummaryRow{
		Batch:     "TOTAL",
		Group:     "",
		JobID:     "",
		State:     overallState,
		Total:     totalTotal,
		Published: totalPublished,
		Errors:    totalErrors,
		StartDate: firstStart,
		EndDate:   lastEnd,
		Duration:  publishItemDuration(firstStart, lastEnd),
	})

	summCols := []output.Column{
		{Header: "BATCH", Field: "batch", Width: 6},
		{Header: "GROUP", Field: "group", Width: 10},
		{Header: "JOB ID", Field: "jobID", Width: 22},
		{Header: "STATE", Field: "state", Width: 10},
		{Header: "TOTAL", Field: "total", Width: 7},
		{Header: "PUBLISHED", Field: "published", Width: 10},
		{Header: "ERRORS", Field: "errors", Width: 8},
		{Header: "START DATE", Field: "startDate", Width: 30},
		{Header: "END DATE", Field: "endDate", Width: 30},
		{Header: "DURATION", Field: "duration", Width: 10},
	}

	_, _ = fmt.Fprintf(out, "\n%s Summary (%d batches):\n", titleCase(verb), len(results))
	if err := f.Format(rows, summCols); err != nil {
		return err
	}

	// Combined items table (verbose only).
	if showItems && len(allItems) > 0 {
		itemCols := []output.Column{
			{Header: "BATCH", Field: "batch", Width: 6},
			{Header: "INDEX", Field: "itemIndex", Width: 6},
			{Header: "GUID", Field: "itemGUID", Width: 24},
			{Header: "ASSET PATH", Field: "assetPath", Width: 60},
			{Header: "STATE", Field: "itemState", Width: 12},
			{Header: "START DATE", Field: "itemStartDate", Width: 30},
			{Header: "END DATE", Field: "itemEndDate", Width: 30},
			{Header: "DURATION", Field: "duration", Width: 10},
		}
		_, _ = fmt.Fprintf(out, "\n%s Items (%d assets, %d batches):\n", titleCase(verb), len(allItems), len(results))
		if err := f.Format(allItems, itemCols); err != nil {
			return err
		}
	}

	// Error detail section.
	if len(errorItems) > 0 {
		errCols := []output.Column{
			{Header: "BATCH", Field: "batch", Width: 6},
			{Header: "INDEX", Field: "index", Width: 6},
			{Header: "GUID", Field: "guid", Width: 24},
			{Header: "ASSET PATH", Field: "path", Width: 60},
			{Header: "DETAIL", Field: "detail", Width: 60},
		}
		_, _ = fmt.Fprintf(out, "\n%s Errors (%d failed):\n", titleCase(verb), len(errorItems))
		if err := f.Format(errorItems, errCols); err != nil {
			return err
		}
	}

	return nil
}

// printSingleBatchErrors prints a summary of failed items for a single-batch job.
func printSingleBatchErrors(cmd *cobra.Command, verb string, items []client.PublishItemDetail) error {
	var errorItems []publishErrorItem
	for _, it := range items {
		if it.ItemState != "SUCCESS" && it.ItemStatusDetail != "" {
			errorItems = append(errorItems, publishErrorItem{
				Index:  it.ItemIndex,
				GUID:   it.ItemGUID,
				Path:   it.AssetPath,
				Detail: it.ItemStatusDetail,
			})
		}
	}
	if len(errorItems) == 0 {
		return nil
	}
	f, err := getFormatter()
	if err != nil {
		return err
	}
	errCols := []output.Column{
		{Header: "INDEX", Field: "index", Width: 6},
		{Header: "GUID", Field: "guid", Width: 24},
		{Header: "ASSET PATH", Field: "path", Width: 60},
		{Header: "DETAIL", Field: "detail", Width: 60},
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\n%s Errors (%d failed):\n", titleCase(verb), len(errorItems))
	return f.Format(errorItems, errCols)
}

// resolvePublishAssets returns asset paths from explicit flags, a file, or stdin.
func resolvePublishAssets(assets []string, fromFile string) ([]string, error) {
	if len(assets) > 0 {
		return assets, nil
	}

	var data []byte
	var err error
	var ext string
	if fromFile != "" {
		data, err = os.ReadFile(fromFile)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", fromFile, err)
		}
		ext = strings.ToLower(filepath.Ext(fromFile))
	} else {
		data, err = io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("reading stdin: %w", err)
		}
	}

	return parsePublishAssets(data, ext)
}

// parsePublishAssets routes to the right parser based on the file extension, or auto-detects
// for stdin (ext == ""). Supported extensions: .txt, .csv, .json, .yaml, .yml.
func parsePublishAssets(data []byte, ext string) ([]string, error) {
	trimmed := strings.TrimSpace(string(data))
	if len(trimmed) == 0 {
		return nil, nil
	}

	switch ext {
	case ".txt":
		return parsePlainPaths(trimmed), nil
	case ".csv":
		return parsePublishCSV([]byte(trimmed))
	case ".json":
		return parsePublishJSON([]byte(trimmed))
	case ".yaml", ".yml":
		return parsePublishYAML([]byte(trimmed))
	}

	// Auto-detect for stdin or unknown extensions.
	if trimmed[0] == '[' {
		return parsePublishJSON([]byte(trimmed))
	}
	firstLine := strings.TrimSpace(trimmed)
	if idx := strings.IndexByte(trimmed, '\n'); idx >= 0 {
		firstLine = strings.TrimSpace(trimmed[:idx])
	}
	// CSV: commas in header line, or single-column CSV header "PATH" / "LOCATION".
	if strings.Contains(firstLine, ",") ||
		strings.EqualFold(firstLine, "path") ||
		strings.EqualFold(firstLine, "location") {
		return parsePublishCSV([]byte(trimmed))
	}
	return parsePlainPaths(trimmed), nil
}

// parsePublishCSV reads CSV (e.g. from `iics objects list -o csv`) and returns asset paths.
// LOCATION column is preferred (value is "Explore/<path>.<TYPE>"; ".xml" is appended).
// When LOCATION is absent, PATH is required; TYPE is optional. With PATH+TYPE the row is
// converted to "Explore/<PATH>.<TYPE>.xml" and non-publishable types are skipped silently.
// With PATH only the value is used as-is.
func parsePublishCSV(data []byte) ([]string, error) {
	r := csv.NewReader(strings.NewReader(string(data)))
	headers, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("reading CSV headers: %w", err)
	}

	pathIdx, typeIdx, locationIdx := -1, -1, -1
	for i, h := range headers {
		switch strings.ToUpper(strings.TrimSpace(h)) {
		case "PATH":
			pathIdx = i
		case "TYPE":
			typeIdx = i
		case "LOCATION":
			locationIdx = i
		}
	}

	if locationIdx < 0 && pathIdx < 0 {
		return nil, fmt.Errorf("CSV must have a LOCATION or PATH column (got: %s)", strings.Join(headers, ", "))
	}

	var paths []string
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading CSV row: %w", err)
		}
		// Prefer LOCATION column.
		if locationIdx >= 0 && locationIdx < len(record) {
			loc := strings.TrimSpace(record[locationIdx])
			if loc != "" {
				paths = append(paths, loc+".xml")
				continue
			}
		}
		if pathIdx < 0 || pathIdx >= len(record) {
			continue
		}
		p := strings.TrimSpace(record[pathIdx])
		if p == "" {
			continue
		}
		if typeIdx >= 0 && typeIdx < len(record) {
			obj := client.Object{
				Path: p,
				Type: strings.TrimSpace(record[typeIdx]),
			}
			assetPath, err := client.AssetPathFromObject(obj)
			if err != nil {
				// Skip non-publishable types silently (e.g. MTT mappings in a mixed list).
				continue
			}
			paths = append(paths, assetPath)
		} else {
			paths = append(paths, ensureXMLSuffix(p))
		}
	}
	return paths, nil
}

// parsePlainPaths splits trimmed text into one asset path per line.
// Lines starting with # and blank lines are ignored.
// Each path is ensured to end with ".xml" as required by the publish API.
func parsePlainPaths(trimmed string) []string {
	var paths []string
	for _, line := range strings.Split(trimmed, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			paths = append(paths, ensureXMLSuffix(line))
		}
	}
	return paths
}

// ensureXMLSuffix appends ".xml" to p if it does not already end with ".xml".
func ensureXMLSuffix(p string) string {
	if strings.HasSuffix(p, ".xml") {
		return p
	}
	return p + ".xml"
}

// publishTypeRank returns the dependency-order rank for an asset path.
// Lower rank = publish first. Unknown types get rank 99.
// Path format: "Explore/some/path/Name.TYPE.xml"
func publishTypeRank(p string) int {
	s := strings.TrimSuffix(p, ".xml")
	if idx := strings.LastIndex(s, "."); idx >= 0 {
		switch s[idx+1:] {
		case "AI_SERVICE_CONNECTOR":
			return 0
		case "AI_CONNECTION":
			return 1
		case "PROCESS":
			return 2
		case "GUIDE":
			return 3
		case "TASKFLOW":
			return 4
		}
	}
	return 99
}

// sortPathsForPublish sorts paths in dependency order (stable) so dependencies
// are published before dependents.
func sortPathsForPublish(paths []string) []string {
	out := make([]string, len(paths))
	copy(out, paths)
	sort.SliceStable(out, func(i, j int) bool {
		return publishTypeRank(out[i]) < publishTypeRank(out[j])
	})
	return out
}

// sortPathsForUnpublish sorts paths in reverse dependency order (stable) so
// dependents are unpublished before their dependencies.
func sortPathsForUnpublish(paths []string) []string {
	out := make([]string, len(paths))
	copy(out, paths)
	sort.SliceStable(out, func(i, j int) bool {
		return publishTypeRank(out[i]) > publishTypeRank(out[j])
	})
	return out
}

// filterPublishablePaths removes paths whose asset type is not one of the supported
// publishable types (AI_SERVICE_CONNECTOR, AI_CONNECTION, PROCESS, GUIDE, TASKFLOW).
// Type is derived from the path suffix: "Explore/path/Name.TYPE.xml"
func filterPublishablePaths(paths []string) []string {
	out := paths[:0:0]
	for _, p := range paths {
		if publishTypeRank(ensureXMLSuffix(p)) < 99 {
			out = append(out, p)
		}
	}
	return out
}

// parsePublishJSON handles a JSON array of strings (direct asset paths) or a JSON array of
// objects from `iics objects list -o json`. Resolution order per row:
//  1. location field non-empty: use location+".xml"
//  2. path + type non-empty: convert via AssetPathFromObject; skip non-publishable types
//  3. path only: use as-is
func parsePublishJSON(data []byte) ([]string, error) {
	trimmed := strings.TrimSpace(string(data))
	// Try array of strings first.
	var strPaths []string
	if err := json.Unmarshal([]byte(trimmed), &strPaths); err == nil {
		var out []string
		for _, p := range strPaths {
			if p != "" {
				out = append(out, p)
			}
		}
		return out, nil
	}
	// Try array of objects (iics objects list -o json output).
	var objs []struct {
		Path     string `json:"path"`
		Type     string `json:"type"`
		Location string `json:"location"`
	}
	if err := json.Unmarshal([]byte(trimmed), &objs); err != nil {
		return nil, fmt.Errorf("parsing JSON asset list: %w", err)
	}
	var paths []string
	for _, o := range objs {
		if o.Location != "" {
			paths = append(paths, o.Location+".xml")
			continue
		}
		if o.Path == "" {
			continue
		}
		if o.Type != "" {
			assetPath, err := client.AssetPathFromObject(client.Object{Path: o.Path, Type: o.Type})
			if err != nil {
				continue // skip non-publishable types
			}
			paths = append(paths, assetPath)
		} else {
			paths = append(paths, ensureXMLSuffix(o.Path))
		}
	}
	return paths, nil
}

// parsePublishYAML handles a YAML list of strings (direct asset paths) or a YAML list of
// objects from `iics objects list -o yaml`. Resolution order per row:
//  1. location field non-empty: use location+".xml"
//  2. path + type non-empty: convert via AssetPathFromObject; skip non-publishable types
//  3. path only: use as-is
func parsePublishYAML(data []byte) ([]string, error) {
	// Try list of strings first.
	var strPaths []string
	if err := yaml.Unmarshal(data, &strPaths); err == nil && len(strPaths) > 0 {
		var out []string
		for _, p := range strPaths {
			if p != "" {
				out = append(out, p)
			}
		}
		return out, nil
	}
	// Try list of objects.
	var objs []struct {
		Path     string `yaml:"path"`
		Type     string `yaml:"type"`
		Location string `yaml:"location"`
	}
	if err := yaml.Unmarshal(data, &objs); err != nil {
		return nil, fmt.Errorf("parsing YAML asset list: %w", err)
	}
	var paths []string
	for _, o := range objs {
		if o.Location != "" {
			paths = append(paths, o.Location+".xml")
			continue
		}
		if o.Path == "" {
			continue
		}
		if o.Type != "" {
			assetPath, err := client.AssetPathFromObject(client.Object{Path: o.Path, Type: o.Type})
			if err != nil {
				continue // skip non-publishable types
			}
			paths = append(paths, assetPath)
		} else {
			paths = append(paths, ensureXMLSuffix(o.Path))
		}
	}
	return paths, nil
}
