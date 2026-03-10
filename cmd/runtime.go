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

func newRuntimeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "runtime",
		Aliases: []string{"rt"},
		Short:   "Manage runtime environments",
	}
	cmd.AddCommand(newRuntimeListCmd())
	cmd.AddCommand(newRuntimeGetCmd())
	cmd.AddCommand(newRuntimeCreateCmd())
	cmd.AddCommand(newRuntimeUpdateCmd())
	return cmd
}

func newRuntimeListCmd() *cobra.Command {
	var opts client.RuntimeListOptions
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List runtime environments",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			runtimes, err := c.ListRuntimeEnvironments(context.Background(), opts)
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
				{Header: "TYPE", Field: "type", Width: 12},
				{Header: "STATUS", Field: "status", Width: 12},
			}
			return f.Format(runtimes, columns)
		},
	}
	cmd.Flags().IntVar(&opts.Limit, "limit", 200, "max results")
	cmd.Flags().IntVar(&opts.Skip, "skip", 0, "number of results to skip")
	return cmd
}

func newRuntimeGetCmd() *cobra.Command {
	var id string
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get runtime environment details",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" {
				return fmt.Errorf("--id is required")
			}
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			rt, err := c.GetRuntimeEnvironment(context.Background(), id)
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
				{Header: "TYPE", Field: "type", Width: 12},
				{Header: "STATUS", Field: "status", Width: 12},
				{Header: "DESCRIPTION", Field: "description"},
			}
			return f.Format(rt, columns)
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "runtime environment ID (required)")
	return cmd
}

func newRuntimeCreateCmd() *cobra.Command {
	var fromFile string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a runtime environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			if fromFile == "" {
				return fmt.Errorf("--from-file is required")
			}
			data, err := os.ReadFile(fromFile)
			if err != nil {
				return fmt.Errorf("reading file: %w", err)
			}
			var rt client.RuntimeEnvironment
			err = json.Unmarshal(data, &rt)
			if err != nil {
				return fmt.Errorf("parsing JSON: %w", err)
			}
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			created, err := c.CreateRuntimeEnvironment(context.Background(), &rt)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Runtime environment created: %s (ID: %s)\n", created.Name, created.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&fromFile, "from-file", "", "JSON file (required)")
	return cmd
}

func newRuntimeUpdateCmd() *cobra.Command {
	var (
		id       string
		fromFile string
	)
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a runtime environment",
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
			var rt client.RuntimeEnvironment
			err = json.Unmarshal(data, &rt)
			if err != nil {
				return fmt.Errorf("parsing JSON: %w", err)
			}
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			updated, err := c.UpdateRuntimeEnvironment(context.Background(), id, &rt)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Runtime environment updated: %s (ID: %s)\n", updated.Name, updated.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "runtime environment ID (required)")
	cmd.Flags().StringVar(&fromFile, "from-file", "", "JSON file (required)")
	return cmd
}
