package cmd

import (
	"context"

	"github.com/jbrazda/iics-cli/internal/output"
	"github.com/spf13/cobra"
)

func newPrivilegeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "privilege",
		Short: "View privileges",
	}
	cmd.AddCommand(newPrivilegeListCmd())
	return cmd
}

func newPrivilegeListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available privileges",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			privileges, err := c.ListPrivileges(context.Background())
			if err != nil {
				return err
			}
			f, err := getFormatter()
			if err != nil {
				return err
			}
			columns := []output.Column{
				{Header: "ID", Field: "id", Width: 24},
				{Header: "NAME", Field: "name", Width: 30},
				{Header: "SERVICE", Field: "service", Width: 20},
				{Header: "DESCRIPTION", Field: "description"},
			}
			return f.Format(privileges, columns)
		},
	}
	return cmd
}
