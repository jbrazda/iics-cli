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
	cmd.AddCommand(newUserChangePasswordCmd())
	cmd.AddCommand(newUserResetPasswordCmd())
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

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "User created: %s (ID: %s)\n", created.UserName, created.ID)
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

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "User updated: %s (ID: %s)\n", updated.UserName, updated.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "user ID (required)")
	cmd.Flags().StringVar(&fromFile, "from-file", "", "JSON file with user updates (required)")
	return cmd
}

func newUserChangePasswordCmd() *cobra.Command {
	var (
		newPassword string
		oldPassword string
		userID      string
	)

	cmd := &cobra.Command{
		Use:   "change-password",
		Short: "Change a user password",
		RunE: func(cmd *cobra.Command, args []string) error {
			if newPassword == "" {
				return fmt.Errorf("--new-password is required")
			}
			if oldPassword == "" && userID == "" {
				return fmt.Errorf("either --old-password or --id is required")
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			req := &client.ChangePasswordRequest{
				NewPassword: newPassword,
				OldPassword: oldPassword,
				UserID:      userID,
			}
			if err := c.ChangePassword(context.Background(), req); err != nil {
				return err
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Password changed successfully.")
			return nil
		},
	}

	cmd.Flags().StringVar(&newPassword, "new-password", "", "new password (required)")
	cmd.Flags().StringVar(&oldPassword, "old-password", "", "current password (required when changing own password)")
	cmd.Flags().StringVar(&userID, "id", "", "user ID (required when admin changes another user's password)")
	return cmd
}

func newUserResetPasswordCmd() *cobra.Command {
	var (
		userID         string
		securityAnswer string
		newPassword    string
	)

	cmd := &cobra.Command{
		Use:   "reset-password",
		Short: "Reset a user password using the security answer",
		RunE: func(cmd *cobra.Command, args []string) error {
			if userID == "" {
				return fmt.Errorf("--id is required")
			}
			if securityAnswer == "" {
				return fmt.Errorf("--security-answer is required")
			}
			if newPassword == "" {
				return fmt.Errorf("--new-password is required")
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			req := &client.ResetPasswordRequest{
				UserID:         userID,
				SecurityAnswer: securityAnswer,
				NewPassword:    newPassword,
			}
			if err := c.ResetPassword(context.Background(), req); err != nil {
				return err
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Password reset successfully.")
			return nil
		},
	}

	cmd.Flags().StringVar(&userID, "id", "", "user ID (required)")
	cmd.Flags().StringVar(&securityAnswer, "security-answer", "", "answer to the security question (required)")
	cmd.Flags().StringVar(&newPassword, "new-password", "", "new password (required)")
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
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Are you sure you want to delete user %s? [y/N]: ", id)
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

			if err := c.DeleteUser(context.Background(), id); err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "User deleted: %s\n", id)
			return nil
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "user ID (required)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompt")
	return cmd
}
