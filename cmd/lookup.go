package cmd

import (
	"context"
	"fmt"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/output"
	"github.com/spf13/cobra"
)

func newLookupCmd() *cobra.Command {
	var (
		id       string
		path     string
		objType  string
	)

	cmd := &cobra.Command{
		Use:   "lookup",
		Short: "Resolve object IDs, names, and paths",
		Long: `Look up objects by ID or by path and type to resolve their metadata.
This is commonly used to obtain an object's ID for export or job requests.`,
		Example: `  iics lookup --id 2iXOKghGpySlgv6ifQImyl
  iics lookup --path "Default/My Mapping" --type DTEMPLATE
  iics lookup --path "Default/Sync Task1" --type DSS`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" && path == "" {
				return fmt.Errorf("either --id or --path (with --type) is required")
			}
			if path != "" && objType == "" {
				return fmt.Errorf("--type is required when using --path")
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			lookupObj := client.LookupObject{}
			if id != "" {
				lookupObj.ID = id
			} else {
				lookupObj.Path = path
				lookupObj.Type = objType
			}

			resp, err := c.Lookup(context.Background(), []client.LookupObject{lookupObj})
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
				{Header: "DESCRIPTION", Field: "description"},
				{Header: "UPDATED", Field: "updateTime", Width: 20},
			}

			return f.Format(resp.Objects, columns)
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "object ID to look up")
	cmd.Flags().StringVar(&path, "path", "", "object path to look up (requires --type)")
	cmd.Flags().StringVar(&objType, "type", "", "object type (PROJECT, FOLDER, DTEMPLATE, MTT, DSS, CONNECTION, etc.)")

	return cmd
}
