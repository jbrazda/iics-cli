package cmd

import (
	"context"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/output"
	"github.com/spf13/cobra"
)

func newSecuritylogCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "securitylog",
		Aliases: []string{"auditlog"},
		Short:   "View security logs",
	}
	cmd.AddCommand(newSecuritylogListCmd())
	return cmd
}

func newSecuritylogListCmd() *cobra.Command {
	var opts client.SecurityLogListOptions

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List security log entries",
		Example: `  iics securitylog list --start "2024-01-01T00:00:00Z" --end "2024-01-31T23:59:59Z"
  iics securitylog list --limit 50`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			logs, err := c.ListSecurityLogs(context.Background(), opts)
			if err != nil {
				return err
			}
			f, err := getFormatter()
			if err != nil {
				return err
			}
			columns := []output.Column{
				{Header: "TIME", Field: "entryTime", Width: 22},
				{Header: "USER", Field: "userName", Width: 25},
				{Header: "ACTION", Field: "action", Width: 20},
				{Header: "OBJECT TYPE", Field: "objectType", Width: 15},
				{Header: "STATUS", Field: "status", Width: 10},
				{Header: "SOURCE IP", Field: "sourceIp", Width: 15},
			}
			return f.Format(logs, columns)
		},
	}

	cmd.Flags().StringVar(&opts.StartTime, "start", "", "start time (ISO 8601)")
	cmd.Flags().StringVar(&opts.EndTime, "end", "", "end time (ISO 8601)")
	cmd.Flags().IntVar(&opts.Limit, "limit", 200, "max results")
	cmd.Flags().IntVar(&opts.Skip, "skip", 0, "number of results to skip")
	return cmd
}
