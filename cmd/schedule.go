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

func newScheduleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schedule",
		Short: "Manage schedules",
	}

	cmd.AddCommand(newScheduleListCmd())
	cmd.AddCommand(newScheduleGetCmd())
	cmd.AddCommand(newScheduleCreateCmd())
	cmd.AddCommand(newScheduleUpdateCmd())
	cmd.AddCommand(newScheduleDeleteCmd())
	return cmd
}

func newScheduleListCmd() *cobra.Command {
	var opts client.ScheduleListOptions

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List schedules",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			schedules, err := c.ListSchedules(context.Background(), opts)
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
				{Header: "STATUS", Field: "status", Width: 12},
				{Header: "INTERVAL", Field: "interval", Width: 12},
				{Header: "UPDATED", Field: "updateTime", Width: 20},
			}

			return f.Format(schedules, columns)
		},
	}

	cmd.Flags().IntVar(&opts.Limit, "limit", 200, "max results")
	cmd.Flags().IntVar(&opts.Skip, "skip", 0, "number of results to skip")
	return cmd
}

func newScheduleGetCmd() *cobra.Command {
	var id string

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get schedule details",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" {
				return fmt.Errorf("--id is required")
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			schedule, err := c.GetSchedule(context.Background(), id)
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
				{Header: "STATUS", Field: "status", Width: 12},
				{Header: "INTERVAL", Field: "interval"},
				{Header: "TIMEZONE", Field: "timezone"},
			}

			return f.Format(schedule, columns)
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "schedule ID (required)")
	return cmd
}

func newScheduleCreateCmd() *cobra.Command {
	var fromFile string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a schedule",
		RunE: func(cmd *cobra.Command, args []string) error {
			if fromFile == "" {
				return fmt.Errorf("--from-file is required")
			}

			data, err := os.ReadFile(fromFile)
			if err != nil {
				return fmt.Errorf("reading file: %w", err)
			}

			var schedule client.Schedule
			err = json.Unmarshal(data, &schedule)
			if err != nil {
				return fmt.Errorf("parsing schedule JSON: %w", err)
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			created, err := c.CreateSchedule(context.Background(), &schedule)
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Schedule created: %s (ID: %s)\n", created.Name, created.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&fromFile, "from-file", "", "JSON file with schedule definition (required)")
	return cmd
}

func newScheduleUpdateCmd() *cobra.Command {
	var (
		id       string
		fromFile string
	)

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a schedule",
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

			var schedule client.Schedule
			err = json.Unmarshal(data, &schedule)
			if err != nil {
				return fmt.Errorf("parsing schedule JSON: %w", err)
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			updated, err := c.UpdateSchedule(context.Background(), id, &schedule)
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Schedule updated: %s (ID: %s)\n", updated.Name, updated.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "schedule ID (required)")
	cmd.Flags().StringVar(&fromFile, "from-file", "", "JSON file with schedule updates (required)")
	return cmd
}

func newScheduleDeleteCmd() *cobra.Command {
	var (
		id  string
		yes bool
	)

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a schedule",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" {
				return fmt.Errorf("--id is required")
			}

			if !yes {
				fmt.Fprintf(cmd.OutOrStdout(), "Are you sure you want to delete schedule %s? [y/N]: ", id)
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

			if err := c.DeleteSchedule(context.Background(), id); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Schedule deleted: %s\n", id)
			return nil
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "schedule ID (required)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompt")
	return cmd
}
