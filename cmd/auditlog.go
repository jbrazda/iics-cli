package cmd

import (
	"context"
	"strings"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/output"
	"github.com/spf13/cobra"
)

var auditLogColumnMap = map[string]output.Column{
	"id":           {Header: "ID", Field: "id", Width: 24},
	"username":     {Header: "USERNAME", Field: "username", Width: 30},
	"category":     {Header: "CATEGORY", Field: "category", Width: 16},
	"event":        {Header: "EVENT", Field: "event", Width: 16},
	"entryTimeUTC": {Header: "TIME (UTC)", Field: "entryTimeUTC", Width: 24},
	"entryTime":    {Header: "TIME (ET)", Field: "entryTime", Width: 24},
	"objectId":     {Header: "OBJECT ID", Field: "objectId", Width: 24},
	"objectName":   {Header: "OBJECT NAME", Field: "objectName", Width: 30},
	"eventParam":   {Header: "EVENT PARAM", Field: "eventParam", Width: 40},
	"message":      {Header: "MESSAGE", Field: "message", Width: 40},
	"orgId":        {Header: "ORG ID", Field: "orgId", Width: 24},
	"version":      {Header: "VERSION", Field: "version", Width: 8},
}

const auditLogDefaultFields = "id,username,category,event,entryTimeUTC,objectName"

func buildAuditLogColumns(fields string) []output.Column {
	names := strings.Split(fields, ",")
	cols := make([]output.Column, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if col, ok := auditLogColumnMap[name]; ok {
			cols = append(cols, col)
		}
	}
	return cols
}

func newAuditlogCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auditlog",
		Short: "Retrieve audit log entries",
	}
	cmd.AddCommand(newAuditlogListCmd())
	return cmd
}

func newAuditlogListCmd() *cobra.Command {
	var opts client.AuditLogListOptions
	var fields string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List audit log entries",
		Example: `  iics auditlog list
  iics auditlog list --limit 50
  iics auditlog list --limit 100 --skip 2
  iics auditlog list --fields id,username,category,event,entryTimeUTC
  iics auditlog list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			logs, err := c.ListAuditLogs(context.Background(), opts)
			if err != nil {
				return err
			}
			f, err := getFormatter()
			if err != nil {
				return err
			}
			return f.Format(logs, buildAuditLogColumns(fields))
		},
	}
	cmd.Flags().IntVar(&opts.Limit, "limit", 0, "entries per page - maps to API batchSize (0 = return most recent 200)")
	cmd.Flags().IntVar(&opts.Skip, "skip", 0, "page number to retrieve, 0-based - maps to API batchId (only used when --limit > 0)")
	cmd.Flags().StringVar(&fields, "fields", auditLogDefaultFields, "comma-separated list of fields to display (table/csv only)")
	return cmd
}
