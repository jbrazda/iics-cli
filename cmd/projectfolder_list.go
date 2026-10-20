package cmd

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/config"
	"github.com/jbrazda/iics-cli/internal/output"
	"github.com/spf13/cobra"
)

// promptNamePattern asks for a name regex for the list commands.
func promptNamePattern(kind string) (string, error) {
	text, err := promptText(fmt.Sprintf("Search %s by path (regular expression, empty for all)", kind), "")
	if err != nil {
		return "", cancelAsNil(err)
	}
	return strings.TrimSpace(text), nil
}

// printContainerList filters a project or folder list by pattern and prints
// it. When the FRS API failed, a note goes to stderr and the created columns
// are left out.
func printContainerList(cmd *cobra.Command, res *client.ContainerList, pattern string) error {
	items, err := client.FilterContainersByPath(res.Items, pattern)
	if err != nil {
		return err
	}
	sort.SliceStable(items, func(i, j int) bool { return strings.ToLower(items[i].Path) < strings.ToLower(items[j].Path) })
	if res.FRSError != nil {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Note: created time and created by are not available (%v); showing objects API data.\n", res.FRSError)
	}

	f, err := getFormatter()
	if err != nil {
		return err
	}

	columns := []output.Column{
		{Header: "PATH", Field: "path", Width: 30, Priority: 1},
		{Header: "DESCRIPTION", Field: "description", Width: 20, MaxWidth: 50, Priority: 4, Shrink: output.ShrinkWrap},
	}
	if res.FRSError == nil {
		columns = append(columns,
			output.Column{Header: "CREATED", Field: "createTime", Width: 20, Priority: 3, Shrink: output.ShrinkNever},
			output.Column{Header: "CREATED BY", Field: "createdBy", Width: 20, Priority: 4, Shrink: output.ShrinkTruncate},
		)
	}
	columns = append(columns,
		output.Column{Header: "UPDATED", Field: "updateTime", Width: 20, Priority: 2, Shrink: output.ShrinkNever},
		output.Column{Header: "UPDATED BY", Field: "updatedBy", Width: 20, Priority: 3, Shrink: output.ShrinkTruncate},
		output.Column{Header: "ID", Field: "id", Width: 22, Priority: 5, Shrink: output.ShrinkNever},
	)
	return f.Format(items, columns)
}

func newProjectListCmd() *cobra.Command {
	var (
		pattern     string
		interactive bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List projects with created and updated details",
		Long: `List projects with description, created and updated time and user.

--name filters by a regular expression matched against the project name
(case-insensitive); --interactive asks for it. Created time and created by come
from the internal FRS API used by the IICS web UI; when it is not available the
list falls back to the objects API without those columns.`,
		Example: `  iics project list
  iics project list --name '^fin'
  iics project list -i`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if interactive {
				if !config.IsTerminal() {
					return fmt.Errorf("--interactive requires a terminal")
				}
				p, err := promptNamePattern("projects")
				if err != nil {
					return err
				}
				pattern = p
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			res, err := c.ListProjectInfo(context.Background())
			if err != nil {
				return err
			}
			return printContainerList(cmd, res, pattern)
		},
	}

	cmd.Flags().StringVar(&pattern, "name", "", "regular expression matched against the project name (case-insensitive)")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "ask for the name pattern")
	return cmd
}

func newFolderListCmd() *cobra.Command {
	var (
		pattern     string
		projectID   string
		projectName string
		interactive bool
		pickProj    bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List folders with created and updated details",
		Long: `List folders with their full path (project/folder), description, created
and updated time and user.

Without a project all folders are listed. Limit the list to one project with
--project-name or --project-id, or pick the project from a list with
--pick-project. --name filters by a regular expression matched against the
full path (case-insensitive); --interactive asks for it.

Created time and created by come from the internal FRS API used by the IICS
web UI; when it is not available the list falls back to the objects API
without those columns.`,
		Example: `  iics folder list
  iics folder list --name 'sales.*_v1$'
  iics folder list --project-name "Finance ETL"
  iics folder list --pick-project
  iics folder list -i`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			if (interactive || pickProj) && !config.IsTerminal() {
				return fmt.Errorf("--interactive and --pick-project require a terminal")
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			if pickProj {
				project, perr := pickProject(ctx, c)
				if perr != nil {
					return perr
				}
				if project == nil {
					_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Canceled.")
					return nil
				}
				projectID, projectName = project.ID, project.Path
			}
			if interactive {
				p, perr := promptNamePattern("folders")
				if perr != nil {
					return perr
				}
				pattern = p
			}

			res, err := c.ListFolderInfo(ctx, projectID, projectName)
			if err != nil {
				return err
			}
			return printContainerList(cmd, res, pattern)
		},
	}

	cmd.Flags().StringVar(&pattern, "name", "", "regular expression matched against the full folder path (case-insensitive)")
	cmd.Flags().StringVar(&projectID, "project-id", "", "list only the folders of this project ID")
	cmd.Flags().StringVar(&projectName, "project-name", "", "list only the folders of this project")
	cmd.Flags().BoolVar(&pickProj, "pick-project", false, "pick the project from a list")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "ask for the name pattern")
	cmd.MarkFlagsMutuallyExclusive("project-id", "project-name", "pick-project")
	return cmd
}
