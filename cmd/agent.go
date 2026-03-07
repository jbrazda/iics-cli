package cmd

import (
	"context"
	"fmt"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/output"
	"github.com/spf13/cobra"
)

func newAgentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Manage secure agents",
	}
	cmd.AddCommand(newAgentListCmd())
	cmd.AddCommand(newAgentStartCmd())
	cmd.AddCommand(newAgentStopCmd())
	return cmd
}

func newAgentListCmd() *cobra.Command {
	var opts client.AgentListOptions
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List secure agents",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			agents, err := c.ListAgents(context.Background(), opts)
			if err != nil {
				return err
			}
			f, err := getFormatter()
			if err != nil {
				return err
			}
			columns := []output.Column{
				{Header: "ID", Field: "id", Width: 24},
				{Header: "NAME", Field: "name", Width: 25},
				{Header: "HOST", Field: "host", Width: 20},
				{Header: "STATUS", Field: "status", Width: 12},
				{Header: "PLATFORM", Field: "platform", Width: 10},
				{Header: "GROUP", Field: "agentGroupName", Width: 20},
			}
			return f.Format(agents, columns)
		},
	}
	cmd.Flags().IntVar(&opts.Limit, "limit", 200, "max results")
	cmd.Flags().IntVar(&opts.Skip, "skip", 0, "number of results to skip")
	return cmd
}

func newAgentStartCmd() *cobra.Command {
	var (
		id      string
		service string
	)
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start an agent service",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" {
				return fmt.Errorf("--id is required")
			}
			if service == "" {
				return fmt.Errorf("--service is required")
			}
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			if err := c.StartAgentService(context.Background(), id, service); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Service %s started on agent %s\n", service, id)
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "agent ID (required)")
	cmd.Flags().StringVar(&service, "service", "", "service name (required)")
	return cmd
}

func newAgentStopCmd() *cobra.Command {
	var (
		id      string
		service string
	)
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop an agent service",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" {
				return fmt.Errorf("--id is required")
			}
			if service == "" {
				return fmt.Errorf("--service is required")
			}
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			if err := c.StopAgentService(context.Background(), id, service); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Service %s stopped on agent %s\n", service, id)
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "agent ID (required)")
	cmd.Flags().StringVar(&service, "service", "", "service name (required)")
	return cmd
}
