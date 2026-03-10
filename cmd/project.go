package cmd

import (
	"context"
	"fmt"

	"github.com/jbrazda/iics-cli/internal/client"
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
		name        string
		description string
	)

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" {
				return fmt.Errorf("--id is required")
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			project := &client.Project{Name: name, Description: description}
			updated, err := c.UpdateProject(context.Background(), id, project)
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Project updated: %s (ID: %s)\n", updated.Name, updated.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "project ID (required)")
	cmd.Flags().StringVar(&name, "name", "", "project name")
	cmd.Flags().StringVar(&description, "description", "", "project description")
	return cmd
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

			if !yes {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Are you sure you want to delete project %s? [y/N]: ", id)
				var confirm string
				_, _ = fmt.Scanln(&confirm)
				if confirm != "y" && confirm != "Y" {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Canceled.")
					return nil
				}
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
