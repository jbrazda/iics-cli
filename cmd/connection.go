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

func newConnectionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "connection",
		Aliases: []string{"conn"},
		Short:   "Manage connections",
	}

	cmd.AddCommand(newConnectionListCmd())
	cmd.AddCommand(newConnectionGetCmd())
	cmd.AddCommand(newConnectionCreateCmd())
	cmd.AddCommand(newConnectionUpdateCmd())
	cmd.AddCommand(newConnectionDeleteCmd())
	return cmd
}

func newConnectionListCmd() *cobra.Command {
	var opts client.ConnectionListOptions

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List connections",
		Example: `  iics connection list
  iics connection list --type TOOLKIT
  iics connection list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			conns, err := c.ListConnections(context.Background(), opts)
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
				{Header: "TYPE", Field: "type", Width: 15},
				{Header: "UPDATED BY", Field: "updatedBy", Width: 20},
				{Header: "UPDATED", Field: "updateTime", Width: 20},
			}

			return f.Format(conns, columns)
		},
	}

	cmd.Flags().StringVar(&opts.Type, "type", "", "filter by connection type")
	cmd.Flags().StringVar(&opts.Name, "name", "", "filter by name")
	cmd.Flags().IntVar(&opts.Limit, "limit", 200, "max results")
	cmd.Flags().IntVar(&opts.Skip, "skip", 0, "number of results to skip")

	return cmd
}

func newConnectionGetCmd() *cobra.Command {
	var id string

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get connection details",
		Example: `  iics connection get --id <connection-id>
  iics connection get --id <connection-id> --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" {
				return fmt.Errorf("--id is required")
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			conn, err := c.GetConnection(context.Background(), id)
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
				{Header: "TYPE", Field: "type", Width: 15},
				{Header: "DESCRIPTION", Field: "description"},
			}

			return f.Format(conn, columns)
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "connection ID (required)")
	return cmd
}

func newConnectionCreateCmd() *cobra.Command {
	var fromFile string

	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a connection",
		Example: `  iics connection create --from-file connection.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if fromFile == "" {
				return fmt.Errorf("--from-file is required")
			}

			data, err := os.ReadFile(fromFile)
			if err != nil {
				return fmt.Errorf("reading file %s: %w", fromFile, err)
			}

			var conn client.Connection
			err = json.Unmarshal(data, &conn)
			if err != nil {
				return fmt.Errorf("parsing connection JSON: %w", err)
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			created, err := c.CreateConnection(context.Background(), &conn)
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Connection created: %s (ID: %s)\n", created.Name, created.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&fromFile, "from-file", "", "JSON file with connection definition (required)")
	return cmd
}

func newConnectionUpdateCmd() *cobra.Command {
	var (
		id       string
		fromFile string
	)

	cmd := &cobra.Command{
		Use:     "update",
		Short:   "Update a connection",
		Example: `  iics connection update --id <connection-id> --from-file connection.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" {
				return fmt.Errorf("--id is required")
			}
			if fromFile == "" {
				return fmt.Errorf("--from-file is required")
			}

			data, err := os.ReadFile(fromFile)
			if err != nil {
				return fmt.Errorf("reading file %s: %w", fromFile, err)
			}

			var conn client.Connection
			err = json.Unmarshal(data, &conn)
			if err != nil {
				return fmt.Errorf("parsing connection JSON: %w", err)
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			updated, err := c.UpdateConnection(context.Background(), id, &conn)
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Connection updated: %s (ID: %s)\n", updated.Name, updated.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "connection ID (required)")
	cmd.Flags().StringVar(&fromFile, "from-file", "", "JSON file with connection updates (required)")
	return cmd
}

func newConnectionDeleteCmd() *cobra.Command {
	var (
		id  string
		yes bool
	)

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a connection",
		Example: `  iics connection delete --id <connection-id>
  iics connection delete --id <connection-id> --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" {
				return fmt.Errorf("--id is required")
			}

			if !yes {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Are you sure you want to delete connection %s? [y/N]: ", id)
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

			if err := c.DeleteConnection(context.Background(), id); err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Connection deleted: %s\n", id)
			return nil
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "connection ID (required)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompt")
	return cmd
}
