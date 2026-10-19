package cmd

import (
	"context"
	"fmt"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/config"
	"github.com/spf13/cobra"
)

func newProjectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage projects",
	}

	cmd.AddCommand(newProjectCreateCmd())
	cmd.AddCommand(newProjectUpdateCmd())
	cmd.AddCommand(newProjectDeleteCmd())
	return cmd
}

func newProjectCreateCmd() *cobra.Command {
	var (
		name        string
		description string
	)

	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a project",
		Example: `  iics project create --name "My Project" --description "Project description"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			project := &client.Project{Name: name, Description: description}
			created, err := c.CreateProject(context.Background(), project)
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Project created: %s (ID: %s)\n", created.Name, created.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "project name (required)")
	cmd.Flags().StringVar(&description, "description", "", "project description")
	return cmd
}

func newProjectUpdateCmd() *cobra.Command {
	var (
		id          string
		projectName string
		name        string
		description string
		interactive bool
	)

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a project's name or description",
		Long: `Change the name and/or description of a project (PATCH).

Identify the project with --id or its current name (--project-name). With
--interactive (or with neither flag on a terminal) pick the project from a
list, filtered by name, then edit its name and description. A description
cannot be cleared.`,
		Example: `  iics project update --id <project-id> --name "Finance Data Engineering"
  iics project update --project-name "Orders" --description "Orders from all regions"
  iics project update -i`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			if id == "" && projectName == "" && !interactive {
				if !config.IsTerminal() {
					return fmt.Errorf("--id or --project-name is required")
				}
				interactive = true
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			if interactive {
				if !config.IsTerminal() {
					return fmt.Errorf("--interactive requires a terminal")
				}
				var current *client.Object
				if id == "" && projectName == "" {
					current, err = pickProject(ctx, c)
					if err != nil {
						return err
					}
					if current == nil {
						_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Canceled.")
						return nil
					}
					projectName = current.Path
				} else {
					projects, lerr := listProjects(ctx, c)
					if lerr != nil {
						return lerr
					}
					current = findObject(projects, id, projectName)
				}
				curName, curDesc := projectName, ""
				if current != nil {
					curName, curDesc = current.Path, current.Description
				}
				edit, ok, perr := promptProjectFolderEdit("project", fmt.Sprintf("%q", firstNonEmpty(curName, id)), curName, curDesc, name, description)
				if perr != nil || !ok {
					return perr
				}
				name, description = edit.Name, edit.Description
			}

			updated, err := c.UpdateProject(ctx, id, projectName, name, description)
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Project updated: %s\n", firstNonEmpty(updated.Name, name, projectName, id))
			return nil
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "project ID")
	cmd.Flags().StringVar(&projectName, "project-name", "", "current project name")
	cmd.Flags().StringVar(&name, "name", "", "new project name")
	cmd.Flags().StringVar(&description, "description", "", "new project description")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "pick the project and edit it interactively")
	cmd.MarkFlagsMutuallyExclusive("id", "project-name")
	return cmd
}

// firstNonEmpty returns the first non-empty string.
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func newProjectDeleteCmd() *cobra.Command {
	var (
		id  string
		yes bool
	)

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" {
				return fmt.Errorf("--id is required")
			}

			if !yes && !confirmAction(cmd, fmt.Sprintf("Are you sure you want to delete project %s?", id)) {
				return nil
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			if err := c.DeleteProject(context.Background(), id); err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Project deleted: %s\n", id)
			return nil
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "project ID (required)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompt")
	return cmd
}
