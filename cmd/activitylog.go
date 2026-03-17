package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/output"
	"github.com/spf13/cobra"
)

// activityLogColumnMap maps API field names (JSON tags) to pre-configured output columns.
// The state column uses a Func to render human-readable labels.
var activityLogColumnMap = map[string]output.Column{
	"id":                   {Header: "ID", Field: "id", Width: 20},
	"objectName":           {Header: "NAME", Field: "objectName", Width: 30},
	"type":                 {Header: "TYPE", Field: "type", Width: 10},
	"state":                {Header: "STATE", Field: "state", Width: 12, Func: activityLogStateLabel},
	"runId":                {Header: "RUN ID", Field: "runId", Width: 12},
	"objectId":             {Header: "TASK ID", Field: "objectId", Width: 24},
	"agentId":              {Header: "AGENT ID", Field: "agentId", Width: 24},
	"runtimeEnvironmentId": {Header: "RUNTIME ENV", Field: "runtimeEnvironmentId", Width: 24},
	"startTime":            {Header: "START TIME (ET)", Field: "startTime", Width: 22},
	"endTime":              {Header: "END TIME (ET)", Field: "endTime", Width: 22},
	"startTimeUtc":         {Header: "START TIME", Field: "startTimeUtc", Width: 22},
	"endTimeUtc":           {Header: "END TIME", Field: "endTimeUtc", Width: 22},
	"failedSourceRows":     {Header: "FAILED SRC", Field: "failedSourceRows", Width: 12},
	"successSourceRows":    {Header: "SUCCESS SRC", Field: "successSourceRows", Width: 12},
	"failedTargetRows":     {Header: "FAILED TGT", Field: "failedTargetRows", Width: 12},
	"successTargetRows":    {Header: "SUCCESS TGT", Field: "successTargetRows", Width: 12},
	"scheduleName":         {Header: "SCHEDULE", Field: "scheduleName", Width: 20},
	"errorMsg":             {Header: "ERROR", Field: "errorMsg", Width: 40},
	"startedBy":            {Header: "STARTED BY", Field: "startedBy", Width: 20},
	"runContextType":       {Header: "RUN CONTEXT", Field: "runContextType", Width: 15},
	"isStopped":            {Header: "STOPPED", Field: "isStopped", Width: 8},
	"totalSuccessRows":     {Header: "TOTAL SUCCESS", Field: "totalSuccessRows", Width: 14},
	"totalFailedRows":      {Header: "TOTAL FAILED", Field: "totalFailedRows", Width: 13},
	"stopOnError":          {Header: "STOP ON ERR", Field: "stopOnError", Width: 12},
	"contextExternalId":    {Header: "CONTEXT ID", Field: "contextExternalId", Width: 24},
}

const activityLogDefaultFields = "id,objectName,type,state,runId,objectId,startTimeUtc,endTimeUtc,startedBy"

// transformationEntryCols defines fixed columns for the transformation entries nested table.
var transformationEntryCols = []output.Column{
	{Header: "ID", Field: "id", Width: 12},
	{Header: "NAME", Field: "txName", Width: 30},
	{Header: "TYPE", Field: "txType", Width: 10},
	{Header: "SUCCESS ROWS", Field: "successRows", Width: 14},
	{Header: "AFFECTED ROWS", Field: "affectedRows", Width: 14},
	{Header: "FAILED ROWS", Field: "failedRows", Width: 12},
}

// mapAttrCols defines fixed columns for key-value map sections (logEntryItemAttrs, sessionVariables).
var mapAttrCols = []output.Column{
	{Header: "ATTRIBUTE", Field: "attribute", Width: 40},
	{Header: "VALUE", Field: "value", Width: 60},
}

// kvPair is used to render map[string]string as a sortable table.
type kvPair struct {
	Attribute string `json:"attribute"`
	Value     string `json:"value"`
}

// activityLogStateLabel maps numeric state codes to human-readable labels.
func activityLogStateLabel(v interface{}) string {
	row, ok := v.(map[string]interface{})
	if !ok {
		return ""
	}
	state, _ := row["state"].(float64)
	switch int(state) {
	case 1:
		return "SUCCESS"
	case 2:
		return "ERRORS"
	case 3:
		return "FAILED"
	case 4:
		return "NOT STARTED"
	default:
		if s, ok := row["state"]; ok {
			return fmt.Sprintf("%v", s)
		}
		return ""
	}
}

// buildActivityLogColumns builds an ordered column slice from a comma-separated field name list.
// Unknown field names are silently skipped. Only affects table and CSV output.
func buildActivityLogColumns(fields string) []output.Column {
	names := strings.Split(fields, ",")
	cols := make([]output.Column, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if col, ok := activityLogColumnMap[name]; ok {
			cols = append(cols, col)
		}
	}
	return cols
}

// mapToKVPairs converts a map[string]string to a sorted slice for tabular display.
func mapToKVPairs(m map[string]string) []kvPair {
	pairs := make([]kvPair, 0, len(m))
	for k, v := range m {
		pairs = append(pairs, kvPair{Attribute: k, Value: v})
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].Attribute < pairs[j].Attribute })
	return pairs
}

// printActivityLogNestedSections prints labeled sub-tables for any non-empty nested
// fields on a single ActivityLogEntry. Only meaningful for table output format.
func printActivityLogNestedSections(entry client.ActivityLogEntry, f output.Formatter, cols []output.Column) error {
	if len(entry.Entries) > 0 {
		fmt.Fprintf(os.Stdout, "\nEntries:\n")
		if err := f.Format(entry.Entries, cols); err != nil {
			return err
		}
	}
	if len(entry.SubTaskEntries) > 0 {
		fmt.Fprintf(os.Stdout, "\nSub-task Entries:\n")
		if err := f.Format(entry.SubTaskEntries, cols); err != nil {
			return err
		}
	}
	if len(entry.LogEntryItemAttrs) > 0 {
		fmt.Fprintf(os.Stdout, "\nItem Attributes:\n")
		if err := f.Format(mapToKVPairs(entry.LogEntryItemAttrs), mapAttrCols); err != nil {
			return err
		}
	}
	if len(entry.SessionVariables) > 0 {
		fmt.Fprintf(os.Stdout, "\nSession Variables:\n")
		if err := f.Format(mapToKVPairs(entry.SessionVariables), mapAttrCols); err != nil {
			return err
		}
	}
	if len(entry.TransformationEntries) > 0 {
		fmt.Fprintf(os.Stdout, "\nTransformation Entries:\n")
		if err := f.Format(entry.TransformationEntries, transformationEntryCols); err != nil {
			return err
		}
	}
	return nil
}

func newActivitylogCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "activitylog",
		Short: "View activity logs for completed jobs",
	}
	cmd.AddCommand(newActivitylogListCmd())
	cmd.AddCommand(newActivitylogGetCmd())
	return cmd
}

func newActivitylogListCmd() *cobra.Command {
	var opts client.ActivityLogListOptions
	var fields string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List activity log entries for completed jobs",
		Example: `  iics activitylog list --task-id abc123
  iics activitylog list --run-id 42 --task-id abc123
  iics activitylog list --limit 50
  iics activitylog list --fields id,objectName,state,errorMsg`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.RunID > 0 && opts.TaskID == "" {
				return fmt.Errorf("flag --run-id requires --task-id to be set")
			}
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			logs, err := c.ListActivityLogs(context.Background(), opts)
			if err != nil {
				return err
			}
			f, err := getFormatter()
			if err != nil {
				return err
			}
			cols := buildActivityLogColumns(fields)
			if err := f.Format(logs, cols); err != nil {
				return err
			}
			// When filtering by a specific run, print nested sections for each entry.
			if opts.RunID > 0 && outputFmt == "table" {
				for _, entry := range logs {
					if err := printActivityLogNestedSections(entry, f, cols); err != nil {
						return err
					}
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.TaskID, "task-id", "", "filter by task ID")
	cmd.Flags().Int64Var(&opts.RunID, "run-id", 0, "filter by run ID (requires --task-id)")
	cmd.Flags().IntVar(&opts.Offset, "offset", 0, "number of rows to skip")
	cmd.Flags().IntVar(&opts.RowLimit, "limit", 200, "max results (max 1000)")
	cmd.Flags().StringVar(&fields, "fields", activityLogDefaultFields, "comma-separated field names for table/csv output")
	return cmd
}

func newActivitylogGetCmd() *cobra.Command {
	var fields string

	cmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get a single activity log entry by ID",
		Example: `  iics activitylog get abc123
  iics activitylog get abc123 --output json
  iics activitylog get abc123 --fields id,objectName,state,runId,failedTargetRows,errorMsg`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			entry, err := c.GetActivityLog(context.Background(), args[0])
			if err != nil {
				return err
			}
			f, err := getFormatter()
			if err != nil {
				return err
			}

			cols := buildActivityLogColumns(fields)

			// JSON/YAML: emit full struct (columns ignored by those formatters)
			if outputFmt == "json" || outputFmt == "yaml" {
				return f.Format([]client.ActivityLogEntry{*entry}, cols)
			}

			// table/csv: main entry first
			if err := f.Format([]client.ActivityLogEntry{*entry}, cols); err != nil {
				return err
			}

			// For CSV, nested sections with mixed structures are not meaningful - stop here.
			if outputFmt == "csv" {
				return nil
			}

			// table only: print labeled nested sections when non-empty
			return printActivityLogNestedSections(*entry, f, cols)
		},
	}

	cmd.Flags().StringVar(&fields, "fields", activityLogDefaultFields, "comma-separated field names for table/csv output")
	return cmd
}
