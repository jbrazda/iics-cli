package cmd

import (
	"context"
	"fmt"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/output"
	"github.com/spf13/cobra"
)

func newObjectsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "objects",
		Short: "Manage organization assets",
		Long:  `List, search, and inspect assets in the IICS organization.`,
	}

	cmd.AddCommand(newObjectsListCmd())
	cmd.AddCommand(newObjectsDependenciesCmd())
	return cmd
}

func newObjectsListCmd() *cobra.Command {
	var opts client.ObjectsListOptions

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List organization assets",
		Example: `  iics objects list --type MTT --limit 50
  iics objects list --query "type=='DTEMPLATE' and location=='Default/Sales'"
  iics objects list --tag production --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			resp, err := c.ListObjects(context.Background(), opts)
			if err != nil {
				return err
			}

			f, err := getFormatter()
			if err != nil {
				return err
			}

			columns := []output.Column{
				{Header: "ID", Field: "id", Width: 24},
				{Header: "TYPE", Field: "type", Width: 12},
				{Header: "PATH", Field: "path"},
				{Header: "UPDATED BY", Field: "updatedBy", Width: 20},
				{Header: "UPDATED", Field: "updateTime", Width: 20},
			}

			return f.Format(resp.Objects, columns)
		},
	}

	cmd.Flags().StringVar(&opts.Type, "type", "", "filter by object type (MTT, DTEMPLATE, DSS, etc.)")
	cmd.Flags().StringVar(&opts.Tag, "tag", "", "filter by tag")
	cmd.Flags().StringVarP(&opts.Query, "query", "q", "", "raw query filter expression")
	cmd.Flags().IntVar(&opts.Limit, "limit", 200, "max results (up to 200)")
	cmd.Flags().IntVar(&opts.Skip, "skip", 0, "number of results to skip")

	return cmd
}

func newObjectsDependenciesCmd() *cobra.Command {
	var (
		objectID string
		refType  string
		limit    int
		skip     int
	)

	cmd := &cobra.Command{
		Use:   "dependencies",
		Short: "Find asset dependencies",
		Example: `  iics objects dependencies --id <object-id>
  iics objects dependencies --id <object-id> --ref-type uses`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if objectID == "" {
				return fmt.Errorf("--id is required")
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			resp, err := c.GetObjectDependencies(context.Background(), objectID, refType, limit, skip)
			if err != nil {
				return err
			}

			f, err := getFormatter()
			if err != nil {
				return err
			}

			columns := []output.Column{
				{Header: "ID", Field: "appContextId", Width: 24},
				{Header: "TYPE", Field: "type", Width: 12},
				{Header: "PATH", Field: "path"},
				{Header: "UPDATED BY", Field: "updatedBy", Width: 20},
			}

			// Show uses or usedBy depending on what's returned
			if len(resp.Uses) > 0 {
				return f.Format(resp.Uses, columns)
			}
			if len(resp.UsedBy) > 0 {
				return f.Format(resp.UsedBy, columns)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "No dependencies found.")
			return nil
		},
	}

	cmd.Flags().StringVar(&objectID, "id", "", "object ID (required)")
	cmd.Flags().StringVar(&refType, "ref-type", "", "reference type: uses or usedBy")
	cmd.Flags().IntVar(&limit, "limit", 200, "max results")
	cmd.Flags().IntVar(&skip, "skip", 0, "number of results to skip")

	return cmd
}
