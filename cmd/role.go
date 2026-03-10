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

func newRoleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "role",
		Short: "Manage roles",
	}
	cmd.AddCommand(newRoleListCmd())
	cmd.AddCommand(newRoleGetCmd())
	cmd.AddCommand(newRoleCreateCmd())
	cmd.AddCommand(newRoleUpdateCmd())
	cmd.AddCommand(newRoleDeleteCmd())
	return cmd
}

func newRoleListCmd() *cobra.Command {
	var opts client.RoleListOptions
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List roles",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			roles, err := c.ListRoles(context.Background(), opts)
			if err != nil {
				return err
			}
			f, err := getFormatter()
			if err != nil {
				return err
			}
			columns := []output.Column{
				{Header: "ID", Field: "id", Width: 24},
				{Header: "NAME", Field: "name", Width: 30},
				{Header: "SYSTEM", Field: "systemRole", Width: 8},
				{Header: "DESCRIPTION", Field: "description"},
			}
			return f.Format(roles, columns)
		},
	}
	cmd.Flags().IntVar(&opts.Limit, "limit", 200, "max results")
	cmd.Flags().IntVar(&opts.Skip, "skip", 0, "number of results to skip")
	return cmd
}

func newRoleGetCmd() *cobra.Command {
	var id string
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get role details",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" {
				return fmt.Errorf("--id is required")
			}
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			role, err := c.GetRole(context.Background(), id)
			if err != nil {
				return err
			}
			f, err := getFormatter()
			if err != nil {
				return err
			}
			columns := []output.Column{
				{Header: "ID", Field: "id", Width: 24},
				{Header: "NAME", Field: "name", Width: 30},
				{Header: "SYSTEM", Field: "systemRole", Width: 8},
				{Header: "DESCRIPTION", Field: "description"},
			}
			return f.Format(role, columns)
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "role ID (required)")
	return cmd
}

func newRoleCreateCmd() *cobra.Command {
	var fromFile string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a role",
		RunE: func(cmd *cobra.Command, args []string) error {
			if fromFile == "" {
				return fmt.Errorf("--from-file is required")
			}
			data, err := os.ReadFile(fromFile)
			if err != nil {
				return fmt.Errorf("reading file: %w", err)
			}
			var role client.Role
			err = json.Unmarshal(data, &role)
			if err != nil {
				return fmt.Errorf("parsing JSON: %w", err)
			}
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			created, err := c.CreateRole(context.Background(), &role)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Role created: %s (ID: %s)\n", created.Name, created.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&fromFile, "from-file", "", "JSON file with role definition (required)")
	return cmd
}

func newRoleUpdateCmd() *cobra.Command {
	var (
		id       string
		fromFile string
	)
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a role",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" {
				return fmt.Errorf("--id is required")
			}
			if fromFile == "" {
				return fmt.Errorf("--from-file is required")
			}
			data, err := os.ReadFile(fromFile)
			if err != nil {
				return fmt.Errorf("reading file: %w", err)
			}
			var role client.Role
			err = json.Unmarshal(data, &role)
			if err != nil {
				return fmt.Errorf("parsing JSON: %w", err)
			}
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			updated, err := c.UpdateRole(context.Background(), id, &role)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Role updated: %s (ID: %s)\n", updated.Name, updated.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "role ID (required)")
	cmd.Flags().StringVar(&fromFile, "from-file", "", "JSON file with role updates (required)")
	return cmd
}

func newRoleDeleteCmd() *cobra.Command {
	var (
		id  string
		yes bool
	)
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a role",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" {
				return fmt.Errorf("--id is required")
			}
			if !yes {
				fmt.Fprintf(cmd.OutOrStdout(), "Are you sure you want to delete role %s? [y/N]: ", id)
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
			if err := c.DeleteRole(context.Background(), id); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Role deleted: %s\n", id)
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "role ID (required)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompt")
	return cmd
}
