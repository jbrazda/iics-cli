package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/config"
	"github.com/jbrazda/iics-cli/internal/dependencies"
	"github.com/jbrazda/iics-cli/internal/output"
	"github.com/jbrazda/iics-cli/internal/release"
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
// The location field is computed as "<Explore|SYS>/<path>.<type>".
func objectToFilteredMap(obj client.Object, fields []string) map[string]interface{} {
	location := client.BuildLocation(obj.Path, obj.Type)
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
updateTime, location. The location field is computed as "<Explore|SYS>/<path>.<type>".`,
		Example: `  iics objects list                          # all objects (auto-paginated)
  iics objects list --type MTT               # all mappings
  iics objects list --type MTT --limit 50    # first 50 mappings
  iics objects list --query "location=='Default/Sales'"
  iics objects list --query "type=='DTEMPLATE';location=='Default/Sales'"
  iics objects list --query "tag=='production'" --output json
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
				// Single page — honor explicit limit and skip
				resp, err = c.ListObjects(context.Background(), opts)
				if err != nil {
					return err
				}
			} else {
				// No limit: fetch all pages with optional verbose progress
				var progressFn func(page, fetched int)
				if verbose {
					progressFn = func(page, fetched int) {
						_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "%s Fetched page %d (%d objects total)\n",
							ts(), page, fetched)
					}
				}
				resp, err = c.ListAllObjects(context.Background(), opts, progressFn)
				if err != nil {
					return err
				}
			}

			// Populate computed Location field on each object
			for i := range resp.Objects {
				resp.Objects[i].Location = client.BuildLocation(resp.Objects[i].Path, resp.Objects[i].Type)
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
				defer func() { _ = fh.Close() }()

				fileFields := parseFields(outputFileFields)
				fileRows := make([]map[string]interface{}, len(resp.Objects))
				for i, obj := range resp.Objects {
					fileRows[i] = objectToFilteredMap(obj, fileFields)
				}

				fileFmtr := output.New(fileFmt, fh, output.TableStyle{NoColor: true})
				fileCols := buildObjectColumns(fileFields)
				if err := fileFmtr.Format(fileRows, fileCols); err != nil {
					return fmt.Errorf("writing output file: %w", err)
				}

				if verbose {
					fi, err := fh.Stat()
					if err == nil {
						_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "%s Wrote %d objects to %s (%d bytes)\n",
							ts(), len(resp.Objects), outputFile, fi.Size())
					} else {
						_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "%s Wrote %d objects to %s\n",
							ts(), len(resp.Objects), outputFile)
					}
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&opts.Type, "type", "", "filter by object type (MTT, DTEMPLATE, DSS, etc.)")
	cmd.Flags().StringVarP(&opts.Query, "query", "q", "", "q filter expression (e.g. \"tag=='prod';location=='Default'\")")
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
		objectID      string
		tagName       string
		refType       string
		limit         int
		skip          int
		targets       []string
		publishMode   bool
		filterPattern string
		excludeFile   string
		outputFile    string
		outputFileFmt string
		outputFields  string
	)
	defaultOutputFileFields := "location,dependency,id,type,path,status,warning"

	cmd := &cobra.Command{
		Use:   "dependencies",
		Short: "Find asset dependencies",
		Long: `Find what objects a given asset depends on, or which assets depend on it.

When --id is omitted and stdin is not a terminal, object rows are read from stdin
with auto-detection for JSON, CSV, and YAML (e.g. piped from
"objects list --output json/csv/yaml"). Seed rows can include id, location, or
path+type, and dependencies for all resolved seed objects are collected.

Without --limit all dependency pages are fetched automatically in batches of 50.

Use --targets to validate each dependency against one or more target profiles and
produce a cross-profile status report (found/missing/unknown).

Use --publish to restrict output to publishable asset types only, sorted in the
correct publish dependency order (connectors before connections before processes).`,
		Example: `  iics objects dependencies --id <id>
  iics objects dependencies --id <id> --ref-type uses
  iics objects list -q "tag==sprint9" --output json | iics objects dependencies --ref-type uses
  iics objects list -q "tag==sprint9" --output csv | iics objects dependencies --ref-type uses
  iics objects list -q "tag==sprint9" --output yaml | iics objects dependencies --ref-type uses
  iics objects list -q "tag==sprint9" --output json | iics objects dependencies --ref-type uses --targets dev,qa
  iics objects list -q "tag==sprint9" --output json | iics objects dependencies --ref-type uses --publish | iics publish run
  iics objects list -q "tag==sprint9" --output json | iics objects dependencies --ref-type uses --targets qa --publish | iics publish run --profile qa`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			// Collect input seed entries from --id, --tag, or auto-detected stdin rows.
			var seedEntries []client.ArtifactEntry
			if objectID != "" && tagName != "" {
				return fmt.Errorf("--id and --tag are mutually exclusive")
			}
			if objectID != "" {
				seedEntries = []client.ArtifactEntry{{ID: objectID}}
			} else if tagName != "" {
				listResp, listErr := c.ListAllObjects(ctx, client.ObjectsListOptions{
					Query: fmt.Sprintf("tag=='%s'", tagName),
				}, nil)
				if listErr != nil {
					return fmt.Errorf("listing objects for tag %q: %w", tagName, listErr)
				}
				for _, o := range listResp.Objects {
					if o.ID != "" {
						seedEntries = append(seedEntries, client.ArtifactEntry{ID: o.ID})
					}
				}
			} else if !config.IsTerminal() {
				data, readErr := io.ReadAll(os.Stdin)
				if readErr != nil {
					return fmt.Errorf("reading stdin: %w", readErr)
				}
				parsedEntries, parseErr := dependencies.ParseSeedEntriesFromInput(data)
				if parseErr != nil {
					return parseErr
				}
				seedEntries = append(seedEntries, parsedEntries...)
			} else {
				return fmt.Errorf("--id is required (or pipe JSON/CSV/YAML rows with id/location/path+type)")
			}

			if len(seedEntries) == 0 {
				return fmt.Errorf("no object seed entries provided")
			}

			resolvedAssets, _, err := dependencies.ResolveSeedAssets(ctx, c, seedEntries, refType, limit, skip)
			if err != nil {
				return err
			}
			deps := resolvedSeedAssetsToDeps(resolvedAssets)
			if filterPattern != "" {
				deps, err = applyFilter(deps, filterPattern)
				if err != nil {
					return fmt.Errorf("invalid --filter pattern: %w", err)
				}
			}
			if excludeFile != "" {
				patterns, patternErr := release.LoadExcludePatterns(excludeFile)
				if patternErr != nil {
					return patternErr
				}
				deps = applyExcludeRegexes(deps, patterns)
			}

			if len(deps) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No dependencies found.")
				return nil
			}

			// Apply --publish filter then sort by typePriority.
			if publishMode {
				filtered := deps[:0:0]
				for _, r := range deps {
					if publishableTypes[r.Type] {
						filtered = append(filtered, r)
					}
				}
				deps = filtered
				sort.Slice(deps, func(i, j int) bool {
					pi, pj := typePriority[deps[i].Type], typePriority[deps[j].Type]
					if pi != pj {
						if pi == 0 {
							return false
						}
						if pj == 0 {
							return true
						}
						return pi < pj
					}
					return deps[i].Path < deps[j].Path
				})
				if len(deps) == 0 {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No publishable dependencies found.")
					return nil
				}
			}

			f, err := getFormatter()
			if err != nil {
				return err
			}

			// Multi-target report mode: validate each dependency against each profile.
			if len(targets) > 0 {
				profileResults, err := validateMultiProfile(ctx, targets, deps)
				if err != nil {
					return err
				}
				tableRows, jsonRows := buildReportRows(deps, targets, profileResults)

				if outputFmt == "json" || outputFmt == "yaml" {
					return f.Format(jsonRows, nil)
				}

				cols := []output.Column{
					{Header: "LOCATION", Field: "location"},
					{Header: "DEPENDENCY", Field: "dependency", Width: 12},
				}
				for _, prof := range targets {
					key := strings.ReplaceAll(prof, "-", "_")
					cols = append(cols, output.Column{
						Header: fmt.Sprintf("STATUS (%s)", prof),
						Field:  "status_" + key,
						Func:   makeProfileStatusFunc(key),
					})
				}
				if outputFmt == "csv" {
					for _, prof := range targets {
						key := strings.ReplaceAll(prof, "-", "_")
						cols = append(cols, output.Column{
							Header: fmt.Sprintf("WARNING (%s)", prof),
							Field:  "warning_" + key,
						})
					}
				}
				if err := f.Format(tableRows, cols); err != nil {
					return err
				}
				if outputFile != "" {
					return writeOutputFile(tableRows, cols, outputFile, outputFileFmt)
				}
				return nil
			}

			// Standard output.
			columns := []output.Column{
				{Header: "LOCATION", Field: "location"},
				{Header: "DEPENDENCY", Field: "dependency", Width: 12},
				{Header: "ID", Field: "id", Width: 24},
				{Header: "TYPE", Field: "type", Width: 20},
				{Header: "PATH", Field: "path", Width: 55},
			}
			if err := f.Format(deps, columns); err != nil {
				return err
			}
			if outputFile != "" {
				fields := parseFields(outputFields)
				if len(fields) == 0 {
					fields = parseFields(defaultOutputFileFields)
				}
				fileRows := dependencyItemsForOutputFile(deps, fields)
				fileCols := make([]output.Column, 0, len(fields))
				for _, field := range fields {
					fileCols = append(fileCols, output.Column{
						Header: strings.ToUpper(field),
						Field:  field,
					})
				}
				return writeOutputFile(fileRows, fileCols, outputFile, outputFileFmt)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&objectID, "id", "", "object ID (or pipe JSON/CSV/YAML rows with id/location/path+type)")
	cmd.Flags().StringVar(&tagName, "tag", "", "tag name to resolve seed objects before dependency traversal")
	cmd.Flags().StringVar(&refType, "ref-type", "", "reference type: uses or usedBy")
	cmd.Flags().IntVar(&limit, "limit", 0, "max results; 0 fetches all pages in batches of 50")
	cmd.Flags().IntVar(&skip, "skip", 0, "number of results to skip (only used with --limit)")
	cmd.Flags().StringSliceVar(&targets, "targets", nil, "comma-separated profiles to validate dependencies against")
	cmd.Flags().BoolVar(&publishMode, "publish", false, "restrict output to publishable types only and sort by publish dependency order")
	cmd.Flags().StringVar(&filterPattern, "filter", "", "regex matched against location ((Explore|SYS)/path.type) to filter final output")
	cmd.Flags().StringVar(&excludeFile, "exclude-file", "", "path to regex patterns file; matching locations are excluded from final output")
	cmd.Flags().StringVar(&outputFile, "output-file", "", "path to write output file")
	cmd.Flags().StringVar(&outputFileFmt, "output-file-format", "yaml", "format for output file: yaml, json, csv, table")
	cmd.Flags().StringVar(&outputFields, "output-file-fields", defaultOutputFileFields, "comma-separated fields for file output: location,dependency,id,type,path,status,warning")

	return cmd
}

func resolvedSeedAssetsToDeps(assets []dependencies.ResolvedSeedAsset) []dependencyItem {
	deps := make([]dependencyItem, len(assets))
	for i, asset := range assets {
		deps[i] = dependencyItem{
			ID:         asset.ID,
			Path:       asset.Path,
			Type:       asset.Type,
			Location:   asset.Location,
			Dependency: asset.Dependency,
		}
	}
	return deps
}

func applyExcludeRegexes(deps []dependencyItem, patterns []*regexp.Regexp) []dependencyItem {
	if len(patterns) == 0 {
		return deps
	}
	filtered := deps[:0:0]
	for _, d := range deps {
		excluded := false
		for _, re := range patterns {
			if re.MatchString(d.Location) {
				excluded = true
				break
			}
		}
		if !excluded {
			filtered = append(filtered, d)
		}
	}
	return filtered
}
