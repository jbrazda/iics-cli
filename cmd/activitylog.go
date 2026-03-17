package cmd

import (
	"context"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/output"
	"github.com/spf13/cobra"
)

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

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List activity log entries for completed jobs",
		Example: `  iics activitylog list --task-id abc123
  iics activitylog list --run-id 42 --task-id abc123
  iics activitylog list --limit 50`,
		RunE: func(cmd *cobra.Command, args []string) error {
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
			columns := []output.Column{
				{Header: "ID", Field: "id", Width: 20},
				{Header: "NAME", Field: "objectName", Width: 30},
				{Header: "TYPE", Field: "type", Width: 10},
				{Header: "STATE", Field: "state", Width: 8},
				{Header: "START TIME", Field: "startTimeUtc", Width: 22},
				{Header: "END TIME", Field: "endTimeUtc", Width: 22},
				{Header: "STARTED BY", Field: "startedBy", Width: 20},
				{Header: "RUN CONTEXT", Field: "runContextType", Width: 15},
			}
			return f.Format(logs, columns)
		},
	}

	cmd.Flags().StringVar(&opts.TaskID, "task-id", "", "filter by task ID")
	cmd.Flags().Int64Var(&opts.RunID, "run-id", 0, "filter by run ID (requires --task-id)")
	cmd.Flags().IntVar(&opts.Offset, "offset", 0, "number of rows to skip")
	cmd.Flags().IntVar(&opts.RowLimit, "limit", 200, "max results (max 1000)")
	return cmd
}

func newActivitylogGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get a single activity log entry by ID",
		Example: `  iics activitylog get abc123`,
		Args:  cobra.ExactArgs(1),
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
			columns := []output.Column{
				{Header: "ID", Field: "id", Width: 20},
				{Header: "NAME", Field: "objectName", Width: 30},
				{Header: "TYPE", Field: "type", Width: 10},
				{Header: "STATE", Field: "state", Width: 8},
				{Header: "START TIME", Field: "startTimeUtc", Width: 22},
				{Header: "END TIME", Field: "endTimeUtc", Width: 22},
				{Header: "STARTED BY", Field: "startedBy", Width: 20},
				{Header: "RUN CONTEXT", Field: "runContextType", Width: 15},
				{Header: "ERROR", Field: "errorMsg", Width: 40},
			}
			return f.Format([]client.ActivityLogEntry{*entry}, columns)
		},
	}
	return cmd
}
