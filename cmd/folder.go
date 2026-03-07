package cmd

import (
	"context"
	"fmt"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/spf13/cobra"
)

func newFolderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "folder",
		Short: "Manage folders",
	}

	cmd.AddCommand(newFolderCreateCmd())
	cmd.AddCommand(newFolderUpdateCmd())
	cmd.AddCommand(newFolderDeleteCmd())
	return cmd
}

func newFolderCreateCmd() *cobra.Command {
	var (
		name        string
		description string
		parentID    string
	)

	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a folder",
		Example: `  iics folder create --name "My Folder" --parent-id <project-id>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			folder := &client.Folder{Name: name, Description: description, ParentID: parentID}
			created, err := c.CreateFolder(context.Background(), folder)
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Folder created: %s (ID: %s)\n", created.Name, created.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "folder name (required)")
	cmd.Flags().StringVar(&description, "description", "", "folder description")
	cmd.Flags().StringVar(&parentID, "parent-id", "", "parent project or folder ID")
	return cmd
}

func newFolderUpdateCmd() *cobra.Command {
	var (
		id          string
		name        string
		description string
	)

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a folder",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" {
				return fmt.Errorf("--id is required")
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			folder := &client.Folder{Name: name, Description: description}
			updated, err := c.UpdateFolder(context.Background(), id, folder)
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Folder updated: %s (ID: %s)\n", updated.Name, updated.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "folder ID (required)")
	cmd.Flags().StringVar(&name, "name", "", "folder name")
	cmd.Flags().StringVar(&description, "description", "", "folder description")
	return cmd
}

func newFolderDeleteCmd() *cobra.Command {
	var (
		id  string
		yes bool
	)

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a folder",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" {
				return fmt.Errorf("--id is required")
			}

			if !yes {
				fmt.Fprintf(cmd.OutOrStdout(), "Are you sure you want to delete folder %s? [y/N]: ", id)
				var confirm string
				fmt.Scanln(&confirm)
				if confirm != "y" && confirm != "Y" {
					fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
					return nil
				}
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			if err := c.DeleteFolder(context.Background(), id); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Folder deleted: %s\n", id)
			return nil
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "folder ID (required)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompt")
	return cmd
}
