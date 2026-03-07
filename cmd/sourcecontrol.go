package cmd

import (
	"context"
	"fmt"

	"github.com/jbrazda/iics-cli/internal/output"
	"github.com/spf13/cobra"
)

func newSourcecontrolCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "sourcecontrol",
		Aliases: []string{"sc"},
		Short:   "Manage source control operations",
	}
	cmd.AddCommand(newSourcecontrolCheckoutCmd())
	cmd.AddCommand(newSourcecontrolCheckinCmd())
	cmd.AddCommand(newSourcecontrolPullCmd())
	cmd.AddCommand(newSourcecontrolCommitCmd())
	return cmd
}

func newSourcecontrolCheckoutCmd() *cobra.Command {
	var objectID string
	cmd := &cobra.Command{
		Use:   "checkout",
		Short: "Check out an object from source control",
		RunE: func(cmd *cobra.Command, args []string) error {
			if objectID == "" {
				return fmt.Errorf("--object-id is required")
			}
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			result, err := c.Checkout(context.Background(), objectID)
			if err != nil {
				return err
			}
			f, err := getFormatter()
			if err != nil {
				return err
			}
			columns := []output.Column{
				{Header: "OBJECT ID", Field: "objectId", Width: 24},
				{Header: "STATUS", Field: "status", Width: 12},
				{Header: "MESSAGE", Field: "message"},
			}
			return f.Format(result, columns)
		},
	}
	cmd.Flags().StringVar(&objectID, "object-id", "", "object ID (required)")
	return cmd
}

func newSourcecontrolCheckinCmd() *cobra.Command {
	var (
		objectID string
		comment  string
	)
	cmd := &cobra.Command{
		Use:   "checkin",
		Short: "Check in an object to source control",
		RunE: func(cmd *cobra.Command, args []string) error {
			if objectID == "" {
				return fmt.Errorf("--object-id is required")
			}
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			result, err := c.Checkin(context.Background(), objectID, comment)
			if err != nil {
				return err
			}
			f, err := getFormatter()
			if err != nil {
				return err
			}
			columns := []output.Column{
				{Header: "OBJECT ID", Field: "objectId", Width: 24},
				{Header: "STATUS", Field: "status", Width: 12},
				{Header: "MESSAGE", Field: "message"},
			}
			return f.Format(result, columns)
		},
	}
	cmd.Flags().StringVar(&objectID, "object-id", "", "object ID (required)")
	cmd.Flags().StringVar(&comment, "comment", "", "checkin comment")
	return cmd
}

func newSourcecontrolPullCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull changes from source control",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			result, err := c.Pull(context.Background())
			if err != nil {
				return err
			}
			f, err := getFormatter()
			if err != nil {
				return err
			}
			columns := []output.Column{
				{Header: "STATUS", Field: "status", Width: 12},
				{Header: "MESSAGE", Field: "message"},
			}
			return f.Format(result, columns)
		},
	}
	return cmd
}

func newSourcecontrolCommitCmd() *cobra.Command {
	var comment string
	cmd := &cobra.Command{
		Use:   "commit",
		Short: "Commit changes to source control",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			result, err := c.Commit(context.Background(), comment)
			if err != nil {
				return err
			}
			f, err := getFormatter()
			if err != nil {
				return err
			}
			columns := []output.Column{
				{Header: "STATUS", Field: "status", Width: 12},
				{Header: "MESSAGE", Field: "message"},
			}
			return f.Format(result, columns)
		},
	}
	cmd.Flags().StringVar(&comment, "comment", "", "commit comment")
	return cmd
}
