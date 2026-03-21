package cmd

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/output"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// publishJobDisplay is a flat struct for formatting publish job output.
type publishJobDisplay struct {
	ID             string `json:"id"`
	Type           string `json:"type"`
	State          string `json:"state"`
	TotalCount     int    `json:"totalCount"`
	ProcessedCount int    `json:"processedCount"`
	StartDate      string `json:"startDate"`
	StartedBy      string `json:"startedBy"`
}

func publishJobToDisplay(r *client.PublishJobResponse) publishJobDisplay {
	return publishJobDisplay{
		ID:             r.Data.ID,
		Type:           r.Data.Type,
		State:          r.Data.Attributes.JobState,
		TotalCount:     r.Data.Attributes.TotalCount,
		ProcessedCount: r.Data.Attributes.ProcessedCount,
		StartDate:      r.Data.Attributes.StartDate,
		StartedBy:      r.Data.Attributes.StartedBy,
	}
}

var publishColumns = []output.Column{
	{Header: "ID", Field: "id", Width: 24},
	{Header: "TYPE", Field: "type", Width: 12},
	{Header: "STATE", Field: "state", Width: 12},
	{Header: "TOTAL", Field: "totalCount", Width: 8},
	{Header: "PROCESSED", Field: "processedCount", Width: 10},
	{Header: "STARTED", Field: "startDate", Width: 22},
	{Header: "BY", Field: "startedBy", Width: 20},
}

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

			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()
			out := cmd.OutOrStdout()

			batches := client.SplitIntoBatches(paths, client.PublishMaxBatchSize)
			if verbose && name != "" {
				_, _ = fmt.Fprintf(out, "[%s] Publishing %d assets (%d batch(es)) — %s\n", ts(), len(paths), len(batches), name)
			} else if verbose {
				_, _ = fmt.Fprintf(out, "[%s] Publishing %d assets in %d batch(es)...\n", ts(), len(paths), len(batches))
			}

			for i, batch := range batches {
				resp, err := c.StartPublish(ctx, caiURL, batch)
				if err != nil {
					return fmt.Errorf("batch %d: starting publish: %w", i+1, err)
				}
				_, _ = fmt.Fprintf(out, "Publish job started: %s (batch %d/%d, assets: %d)\n",
					resp.Data.ID, i+1, len(batches), len(batch))
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

			f, err := getFormatter()
			if err != nil {
				return err
			}

			return f.Format(publishJobToDisplay(resp), publishColumns)
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
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Submit, poll, and report a publish job",
		Long: `Resolves asset inputs, auto-batches into 199-asset chunks, submits each batch,
polls to completion, and prints a detailed summary.`,
		Example: `  iics publish run --asset "Explore/Default/MyProcess.PROCESS.xml"
  iics publish run --from-file assets.txt --verbose
  iics publish run --from-file assets.csv --polling-interval 15 --max-wait-time 600
  iics objects list -q "type=='PROCESS'" -o csv | iics publish run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			paths, err := resolvePublishAssets(assets, fromFile)
			if err != nil {
				return err
			}
			if len(paths) == 0 {
				return fmt.Errorf("no asset paths provided; use --asset, --from-file, or pipe from stdin")
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			return runPublishOp(cmd, c, paths, caiURL, name, "publish", pollingInterval, maxWaitTime, detailedPolling,
				c.StartPublish, c.GetPublishStatus)
		},
	}

	cmd.Flags().StringArrayVar(&assets, "asset", nil, "asset path(s) to publish (repeatable)")
	cmd.Flags().StringVar(&fromFile, "from-file", "", "file with asset paths (txt/json/csv); omit to read from stdin")
	cmd.Flags().StringVar(&caiURL, "cai-url", "", "CAI base URL override")
	cmd.Flags().StringVar(&name, "name", "", "optional job label")
	cmd.Flags().IntVar(&pollingInterval, "polling-interval", 10, "seconds between status polls")
	cmd.Flags().IntVar(&maxWaitTime, "max-wait-time", 300, "maximum seconds to wait for completion")
	cmd.Flags().BoolVar(&detailedPolling, "detailed-polling", false, "print totalCount/processedCount on each poll")
	return cmd
}

// runPublishOp is the shared run loop for publish and unpublish.
func runPublishOp(
	cmd *cobra.Command,
	c *client.Client,
	paths []string,
	caiURL, name, verb string,
	pollingInterval, maxWaitTime int,
	detailedPolling bool,
	startFn func(context.Context, string, []string) (*client.PublishJobResponse, error),
	statusFn func(context.Context, string, string, bool) (*client.PublishJobResponse, error),
) error {
	ctx := context.Background()
	out := cmd.OutOrStdout()

	batches := client.SplitIntoBatches(paths, client.PublishMaxBatchSize)
	verbLabel := titleCase(verb) + "ing"
	if verbose {
		label := ""
		if name != "" {
			label = " — " + name
		}
		_, _ = fmt.Fprintf(out, "[%s] %s %d assets in %d batch(es)%s...\n",
			ts(), verbLabel, len(paths), len(batches), label)
	}

	interval := time.Duration(pollingInterval) * time.Second
	startWall := time.Now()

	var lastResp *client.PublishJobResponse
	for batchIdx, batch := range batches {
		if verbose {
			_, _ = fmt.Fprintf(out, "[%s] Submitting batch %d/%d (%d assets)...\n",
				ts(), batchIdx+1, len(batches), len(batch))
		}

		resp, err := startFn(ctx, caiURL, batch)
		if err != nil {
			return fmt.Errorf("batch %d: submitting: %w", batchIdx+1, err)
		}
		jobID := resp.Data.ID
		_, _ = fmt.Fprintf(out, "[%s] Batch %d/%d job ID: %s\n", ts(), batchIdx+1, len(batches), jobID)

		// Poll until terminal or timeout.
		deadline := time.Now().Add(time.Duration(maxWaitTime) * time.Second)
		for client.PublishIsInProgress(resp.Data.Attributes.JobState) {
			if time.Now().After(deadline) {
				return fmt.Errorf("timed out after %ds waiting for %s job %s (last status: %s)",
					maxWaitTime, verb, jobID, resp.Data.Attributes.JobState)
			}
			time.Sleep(interval)

			resp, err = statusFn(ctx, caiURL, jobID, false)
			if err != nil {
				return fmt.Errorf("polling %s job %s: %w", verb, jobID, err)
			}

			elapsed := time.Since(startWall).Round(time.Second)
			if verbose {
				msg := fmt.Sprintf("[%s] Status: %s", ts(), resp.Data.Attributes.JobState)
				if detailedPolling {
					msg += fmt.Sprintf(" (%d/%d processed)", resp.Data.Attributes.ProcessedCount, resp.Data.Attributes.TotalCount)
				}
				msg += fmt.Sprintf(" elapsed: %s", elapsed)
				_, _ = fmt.Fprintln(out, msg)
			}
		}

		if resp.Data.Attributes.JobState == "FAILED" || resp.Data.Attributes.JobState == "ERROR" {
			return fmt.Errorf("batch %d: %s job %s ended with status: %s",
				batchIdx+1, verb, jobID, resp.Data.Attributes.JobState)
		}
		lastResp = resp
	}

	// Print final summary.
	_, _ = fmt.Fprintf(out, "\n%s Summary:\n", titleCase(verb))
	f, err := getFormatter()
	if err != nil {
		return err
	}
	if lastResp != nil {
		return f.Format(publishJobToDisplay(lastResp), publishColumns)
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
	// CSV: commas in header line, or single-column CSV header "PATH".
	if strings.Contains(firstLine, ",") || strings.EqualFold(firstLine, "path") {
		return parsePublishCSV([]byte(trimmed))
	}
	return parsePlainPaths(trimmed), nil
}

// parsePublishCSV reads CSV (e.g. from `iics objects list -o csv`) and returns asset paths.
// PATH column is required. TYPE column is optional: when present rows are converted to full
// Explore/... CAI paths and non-publishable types are silently skipped; when absent the PATH
// value is used as-is (caller is responsible for providing the correct asset path format).
func parsePublishCSV(data []byte) ([]string, error) {
	r := csv.NewReader(strings.NewReader(string(data)))
	headers, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("reading CSV headers: %w", err)
	}

	pathIdx, typeIdx := -1, -1
	for i, h := range headers {
		switch strings.ToUpper(strings.TrimSpace(h)) {
		case "PATH":
			pathIdx = i
		case "TYPE":
			typeIdx = i
		}
	}

	if pathIdx < 0 {
		return nil, fmt.Errorf("CSV must have a PATH column (got: %s)", strings.Join(headers, ", "))
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
		if pathIdx >= len(record) {
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
			paths = append(paths, p)
		}
	}
	return paths, nil
}

// parsePlainPaths splits trimmed text into one asset path per line.
// Lines starting with # and blank lines are ignored.
func parsePlainPaths(trimmed string) []string {
	var paths []string
	for _, line := range strings.Split(trimmed, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			paths = append(paths, line)
		}
	}
	return paths
}

// parsePublishJSON handles a JSON array of strings (direct asset paths) or a JSON array of
// objects from `iics objects list -o json`. When objects have a non-empty type field they are
// converted to full Explore/... paths; non-publishable types are silently skipped. When type
// is absent the path value is used as-is.
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
		Path string `json:"path"`
		Type string `json:"type"`
	}
	if err := json.Unmarshal([]byte(trimmed), &objs); err != nil {
		return nil, fmt.Errorf("parsing JSON asset list: %w", err)
	}
	var paths []string
	for _, o := range objs {
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
			paths = append(paths, o.Path)
		}
	}
	return paths, nil
}

// parsePublishYAML handles a YAML list of strings (direct asset paths) or a YAML list of
// objects from `iics objects list -o yaml`. When objects have a non-empty type field they are
// converted to full Explore/... paths; non-publishable types are silently skipped. When type
// is absent the path value is used as-is.
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
		Path string `yaml:"path"`
		Type string `yaml:"type"`
	}
	if err := yaml.Unmarshal(data, &objs); err != nil {
		return nil, fmt.Errorf("parsing YAML asset list: %w", err)
	}
	var paths []string
	for _, o := range objs {
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
			paths = append(paths, o.Path)
		}
	}
	return paths, nil
}
