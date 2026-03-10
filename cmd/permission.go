package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/output"
	"github.com/spf13/cobra"
)

func newPermissionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "permission",
		Aliases: []string{"perm"},
		Short:   "Manage object permissions",
	}
	cmd.AddCommand(newPermissionGetCmd())
	cmd.AddCommand(newPermissionSetCmd())
	cmd.AddCommand(newPermissionDeleteCmd())
	return cmd
}

func newPermissionGetCmd() *cobra.Command {
	var objectID string
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get object permissions",
		RunE: func(cmd *cobra.Command, args []string) error {
			if objectID == "" {
				return fmt.Errorf("--object-id is required")
			}
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			perms, err := c.GetObjectPermissions(context.Background(), objectID)
			if err != nil {
				return err
			}
			f, err := getFormatter()
			if err != nil {
				return err
			}
			columns := []output.Column{
				{Header: "PRINCIPAL ID", Field: "principalId", Width: 24},
				{Header: "PRINCIPAL TYPE", Field: "principalType", Width: 15},
				{Header: "PRINCIPAL NAME", Field: "principalName", Width: 20},
				{Header: "PERMISSION", Field: "permission", Width: 15},
			}
			return f.Format(perms.Permissions, columns)
		},
	}
	cmd.Flags().StringVar(&objectID, "object-id", "", "object ID (required)")
	return cmd
}

func newPermissionSetCmd() *cobra.Command {
	var (
		objectID string
		fromFile string
	)
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set object permissions",
		RunE: func(cmd *cobra.Command, args []string) error {
			if objectID == "" {
				return fmt.Errorf("--object-id is required")
			}
			if fromFile == "" {
				return fmt.Errorf("--from-file is required")
			}
			data, err := os.ReadFile(fromFile)
			if err != nil {
				return fmt.Errorf("reading file: %w", err)
			}
			var perms client.ObjectPermission
			err = json.Unmarshal(data, &perms)
			if err != nil {
				return fmt.Errorf("parsing JSON: %w", err)
			}
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			_, err = c.SetObjectPermissions(context.Background(), objectID, &perms)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Permissions set on object %s\n", objectID)
			return nil
		},
	}
	cmd.Flags().StringVar(&objectID, "object-id", "", "object ID (required)")
	cmd.Flags().StringVar(&fromFile, "from-file", "", "JSON file with permissions (required)")
	return cmd
}

func newPermissionDeleteCmd() *cobra.Command {
	var (
		objectID string
		yes      bool
	)
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete object permissions",
		RunE: func(cmd *cobra.Command, args []string) error {
			if objectID == "" {
				return fmt.Errorf("--object-id is required")
			}
			if !yes {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Are you sure you want to delete permissions for object %s? [y/N]: ", objectID)
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
			if err := c.DeleteObjectPermissions(context.Background(), objectID); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Permissions deleted for object %s\n", objectID)
			return nil
		},
	}
	cmd.Flags().StringVar(&objectID, "object-id", "", "object ID (required)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompt")
	return cmd
}
