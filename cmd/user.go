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

func newUserCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Manage users",
	}

	cmd.AddCommand(newUserListCmd())
	cmd.AddCommand(newUserGetCmd())
	cmd.AddCommand(newUserCreateCmd())
	cmd.AddCommand(newUserUpdateCmd())
	cmd.AddCommand(newUserDeleteCmd())
	return cmd
}

func newUserListCmd() *cobra.Command {
	var opts client.UserListOptions

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List users",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			users, err := c.ListUsers(context.Background(), opts)
			if err != nil {
				return err
			}

			f, err := getFormatter()
			if err != nil {
				return err
			}

			columns := []output.Column{
				{Header: "ID", Field: "id", Width: 24},
				{Header: "USERNAME", Field: "userName", Width: 25},
				{Header: "EMAIL", Field: "email", Width: 30},
				{Header: "STATE", Field: "state", Width: 12},
				{Header: "UPDATED", Field: "updateTime", Width: 22},
			}

			return f.Format(users, columns)
		},
	}

	cmd.Flags().IntVar(&opts.Limit, "limit", 200, "max results")
	cmd.Flags().IntVar(&opts.Skip, "skip", 0, "number of results to skip")
	return cmd
}

func newUserGetCmd() *cobra.Command {
	var id string

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get user details",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" {
				return fmt.Errorf("--id is required")
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			user, err := c.GetUser(context.Background(), id)
			if err != nil {
				return err
			}

			f, err := getFormatter()
			if err != nil {
				return err
			}

			columns := []output.Column{
				{Header: "ID", Field: "id", Width: 24},
				{Header: "USERNAME", Field: "userName", Width: 25},
				{Header: "EMAIL", Field: "email", Width: 30},
				{Header: "STATE", Field: "state", Width: 12},
			}

			return f.Format(user, columns)
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "user ID (required)")
	return cmd
}

func newUserCreateCmd() *cobra.Command {
	var fromFile string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a user",
		RunE: func(cmd *cobra.Command, args []string) error {
			if fromFile == "" {
				return fmt.Errorf("--from-file is required")
			}

			data, err := os.ReadFile(fromFile)
			if err != nil {
				return fmt.Errorf("reading file: %w", err)
			}

			var user client.User
			err = json.Unmarshal(data, &user)
			if err != nil {
				return fmt.Errorf("parsing user JSON: %w", err)
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			created, err := c.CreateUser(context.Background(), &user)
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "User created: %s (ID: %s)\n", created.UserName, created.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&fromFile, "from-file", "", "JSON file with user definition (required)")
	return cmd
}

func newUserUpdateCmd() *cobra.Command {
	var (
		id       string
		fromFile string
	)

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a user",
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

			var user client.User
			err = json.Unmarshal(data, &user)
			if err != nil {
				return fmt.Errorf("parsing user JSON: %w", err)
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			updated, err := c.UpdateUser(context.Background(), id, &user)
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "User updated: %s (ID: %s)\n", updated.UserName, updated.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "user ID (required)")
	cmd.Flags().StringVar(&fromFile, "from-file", "", "JSON file with user updates (required)")
	return cmd
}

func newUserDeleteCmd() *cobra.Command {
	var (
		id  string
		yes bool
	)

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a user",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" {
				return fmt.Errorf("--id is required")
			}

			if !yes {
				fmt.Fprintf(cmd.OutOrStdout(), "Are you sure you want to delete user %s? [y/N]: ", id)
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

			if err := c.DeleteUser(context.Background(), id); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "User deleted: %s\n", id)
			return nil
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "user ID (required)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompt")
	return cmd
}
