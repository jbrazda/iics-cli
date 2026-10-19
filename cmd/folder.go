package cmd

import (
	"context"
	"fmt"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/config"
	"github.com/spf13/cobra"
)

func newFolderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "folder",
		Short: "Manage folders",
	}

	cmd.AddCommand(newFolderListCmd())
	cmd.AddCommand(newFolderCreateCmd())
	cmd.AddCommand(newFolderUpdateCmd())
	cmd.AddCommand(newFolderDeleteCmd())
	return cmd
}

func newFolderCreateCmd() *cobra.Command {
	var (
		name        string
		description string
		projectID   string
		projectName string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a folder",
		Example: `  iics folder create --name "My Folder"
  iics folder create --name "My Folder" --project-id <project-id>
  iics folder create --name "My Folder" --project-name "Orders"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			created, err := c.CreateFolder(context.Background(), name, description, projectID, projectName)
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Folder created: %s (ID: %s)\n", created.Name, created.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "folder name (required)")
	cmd.Flags().StringVar(&description, "description", "", "folder description")
	cmd.Flags().StringVar(&projectID, "project-id", "", "parent project ID")
	cmd.Flags().StringVar(&projectName, "project-name", "", "parent project name (mutually exclusive with --project-id)")
	return cmd
}

func newFolderUpdateCmd() *cobra.Command {
	var (
		id          string
		name        string
		description string
		projectID   string
		projectName string
		folderName  string
		interactive bool
	)

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a folder's name or description",
		Long: `Change the name and/or description of a folder (PATCH).

Identify the folder by ID (--id, works for folders in any project) or by name
(--project-name and --folder-name); IDs and names cannot be mixed. With --interactive (or with no identifier on a terminal) pick
a project, then one of its folders, filtered by name, and edit the folder's
name and description. A description cannot be cleared.`,
		Example: `  iics folder update --id <folder-id> --name "New Name"
  iics folder update --project-name "Orders" --folder-name "Northeast Orders" --name "Northeast Requests"
  iics folder update -i`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			noTarget := id == "" && projectID == "" && projectName == "" && folderName == ""
			if noTarget && !interactive && config.IsTerminal() {
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
				switch {
				case noTarget:
					project, folder, perr := pickFolder(ctx, c)
					if perr != nil {
						return perr
					}
					if folder == nil {
						_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Canceled.")
						return nil
					}
					current = folder
					projectName, folderName = project.Path, objectName(*folder)
				case projectName != "" && folderName != "":
					folders, lerr := listFolders(ctx, c, projectName)
					if lerr != nil {
						return lerr
					}
					current = findObject(folders, "", folderName)
				}
				curName, curDesc := folderName, ""
				if current != nil {
					curName, curDesc = objectName(*current), current.Description
				}
				target := fmt.Sprintf("%q", firstNonEmpty(curName, id))
				if projectName != "" {
					target = fmt.Sprintf("%q in project %q", curName, projectName)
				}
				edit, ok, perr := promptProjectFolderEdit("folder", target, curName, curDesc, name, description)
				if perr != nil || !ok {
					return perr
				}
				name, description = edit.Name, edit.Description
			}

			updated, err := c.UpdateFolder(ctx, id, name, description, projectID, projectName, folderName)
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Folder updated: %s\n", firstNonEmpty(updated.Name, name, folderName, id))
			return nil
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "folder ID")
	cmd.Flags().StringVar(&name, "name", "", "new folder name")
	cmd.Flags().StringVar(&description, "description", "", "new folder description")
	cmd.Flags().StringVar(&projectID, "project-id", "", "parent project ID (not needed; --id identifies the folder)")
	_ = cmd.Flags().MarkDeprecated("project-id", "folder IDs are unique in the org; --id alone identifies the folder")
	cmd.Flags().StringVar(&projectName, "project-name", "", "parent project name (use with --folder-name)")
	cmd.Flags().StringVar(&folderName, "folder-name", "", "current folder name (use with --project-name)")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "pick the folder and edit it interactively")
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

			if !yes && !confirmAction(cmd, fmt.Sprintf("Are you sure you want to delete folder %s?", id)) {
				return nil
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			if err := c.DeleteFolder(context.Background(), id); err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Folder deleted: %s\n", id)
			return nil
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "folder ID (required)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompt")
	return cmd
}
