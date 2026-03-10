package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

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

// defaultObjectFields is the default set of fields shown in console and written to file.
const defaultObjectFields = "id,path,type,updatedBy,updateTime"

// objectsColumnDefs maps field names to their column definitions for console output.
var objectsColumnDefs = map[string]output.Column{
	"id":          {Header: "ID", Field: "id", Width: 24},
	"type":        {Header: "TYPE", Field: "type", Width: 12},
	"path":        {Header: "PATH", Field: "path"},
	"description": {Header: "DESCRIPTION", Field: "description"},
	"updatedBy":   {Header: "UPDATED BY", Field: "updatedBy", Width: 20},
	"updateTime":  {Header: "UPDATED", Field: "updateTime", Width: 20},
	"location":    {Header: "LOCATION", Field: "location"},
}

func parseFields(s string) []string {
	parts := strings.Split(s, ",")
	fields := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			fields = append(fields, p)
		}
	}
	return fields
}

func buildObjectColumns(fields []string) []output.Column {
	cols := make([]output.Column, 0, len(fields))
	for _, f := range fields {
		if col, ok := objectsColumnDefs[f]; ok {
			cols = append(cols, col)
		}
	}
	return cols
}

// objectToFilteredMap returns a map containing only the requested fields for an object.
// The location field is computed as "Explore/<path>.<type>".
func objectToFilteredMap(obj client.Object, fields []string) map[string]interface{} {
	location := "Explore/" + obj.Path + "." + obj.Type
	allFields := map[string]interface{}{
		"id":          obj.ID,
		"path":        obj.Path,
		"type":        obj.Type,
		"description": obj.Description,
		"updatedBy":   obj.UpdatedBy,
		"updateTime":  obj.UpdateTime,
		"location":    location,
	}
	result := make(map[string]interface{}, len(fields))
	for _, f := range fields {
		if v, ok := allFields[f]; ok {
			result[f] = v
		}
	}
	return result
}

func newObjectsListCmd() *cobra.Command {
	var (
		opts             client.ObjectsListOptions
		outputFields     string
		outputFile       string
		outputFileFmt    string
		outputFileFields string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List organization assets",
		Long: `List assets in the IICS organization.

Without --limit, all matching objects are returned by automatically paging
through results in batches of 200. Use --limit to cap the results and --skip
to offset into the result set.

Use --output-file to write results to a file in a different format (yaml, json,
csv, table) independently of the console --output format.

The --output-fields and --output-file-fields flags accept a comma-separated
list of field names. Available fields: id, path, type, description, updatedBy,
updateTime, location. The location field is computed as "Explore/<path>.<type>".`,
		Example: `  iics objects list                          # all objects (auto-paginated)
  iics objects list --type MTT               # all mappings
  iics objects list --type MTT --limit 50    # first 50 mappings
  iics objects list --query "type=='DTEMPLATE' and location=='Default/Sales'"
  iics objects list --tag production --output json
  iics objects list --output-file objects.yaml --output-file-format yaml
  iics objects list --output-file objects.yaml --output-file-format yaml \
    --output-file-fields id,path,type,location`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			var resp *client.ObjectsListResponse

			if opts.Limit > 0 {
				// Single page — honour explicit limit and skip
				resp, err = c.ListObjects(context.Background(), opts)
				if err != nil {
					return err
				}
			} else {
				// No limit: fetch all pages with optional verbose progress
				var progressFn func(page, fetched int)
				if verbose {
					progressFn = func(page, fetched int) {
						fmt.Fprintf(cmd.ErrOrStderr(), "[%s] Fetched page %d (%d objects total)\n",
							time.Now().Format(time.RFC3339), page, fetched)
					}
				}
				resp, err = c.ListAllObjects(context.Background(), opts, progressFn)
				if err != nil {
					return err
				}
			}

			// Populate computed Location field on each object
			for i := range resp.Objects {
				resp.Objects[i].Location = "Explore/" + resp.Objects[i].Path + "." + resp.Objects[i].Type
			}

			// Console output
			consoleFields := parseFields(outputFields)
			consoleCols := buildObjectColumns(consoleFields)

			f, err := getFormatter()
			if err != nil {
				return err
			}
			if err := f.Format(resp.Objects, consoleCols); err != nil {
				return err
			}

			// File output
			if outputFile != "" {
				fileFmt, err := output.ParseFormat(outputFileFmt)
				if err != nil {
					return fmt.Errorf("--output-file-format: %w", err)
				}

				fh, err := os.Create(outputFile)
				if err != nil {
					return fmt.Errorf("creating output file %s: %w", outputFile, err)
				}
				defer fh.Close()

				fileFields := parseFields(outputFileFields)
				fileRows := make([]map[string]interface{}, len(resp.Objects))
				for i, obj := range resp.Objects {
					fileRows[i] = objectToFilteredMap(obj, fileFields)
				}

				fileFmtr := output.New(fileFmt, fh)
				fileCols := buildObjectColumns(fileFields)
				if err := fileFmtr.Format(fileRows, fileCols); err != nil {
					return fmt.Errorf("writing output file: %w", err)
				}

				if verbose {
					fi, err := fh.Stat()
					if err == nil {
						fmt.Fprintf(cmd.ErrOrStderr(), "[%s] Wrote %d objects to %s (%d bytes)\n",
							time.Now().Format(time.RFC3339), len(resp.Objects), outputFile, fi.Size())
					} else {
						fmt.Fprintf(cmd.ErrOrStderr(), "[%s] Wrote %d objects to %s\n",
							time.Now().Format(time.RFC3339), len(resp.Objects), outputFile)
					}
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&opts.Type, "type", "", "filter by object type (MTT, DTEMPLATE, DSS, etc.)")
	cmd.Flags().StringVar(&opts.Tag, "tag", "", "filter by tag")
	cmd.Flags().StringVarP(&opts.Query, "query", "q", "", "raw query filter expression")
	cmd.Flags().IntVar(&opts.Limit, "limit", 0, "max results to return (default 0 = all, pages of 200)")
	cmd.Flags().IntVar(&opts.Skip, "skip", 0, "number of results to skip (only used with --limit)")
	cmd.Flags().StringVar(&outputFields, "output-fields", defaultObjectFields,
		"comma-separated fields for console output: id,path,type,description,updatedBy,updateTime,location")
	cmd.Flags().StringVar(&outputFile, "output-file", "", "path to write output file")
	cmd.Flags().StringVar(&outputFileFmt, "output-file-format", "yaml",
		"format for output file: yaml, json, csv, table")
	cmd.Flags().StringVar(&outputFileFields, "output-file-fields", defaultObjectFields,
		"comma-separated fields for file output: id,path,type,description,updatedBy,updateTime,location")

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
