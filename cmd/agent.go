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
	cmd.AddCommand(newAgentGetCmd())
	cmd.AddCommand(newAgentDetailsCmd())
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
				{Header: "HOST", Field: "agentHost", Width: 20},
				{Header: "ACTIVE", Field: "active", Width: 8},
				{Header: "READY", Field: "readyToRun", Width: 8},
				{Header: "PLATFORM", Field: "platform", Width: 10},
				{Header: "VERSION", Field: "agentVersion", Width: 15},
				{Header: "GROUP ID", Field: "agentGroupId", Width: 24},
			}
			return f.Format(agents, columns)
		},
	}
	cmd.Flags().IntVar(&opts.Limit, "limit", 200, "max results")
	cmd.Flags().IntVar(&opts.Skip, "skip", 0, "number of results to skip")
	cmd.Flags().BoolVar(&opts.IncludeUnassignedOnly, "unassigned", false, "include only unassigned agents")
	return cmd
}

func newAgentGetCmd() *cobra.Command {
	var id string
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a secure agent by ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" {
				return fmt.Errorf("--id is required")
			}
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			agent, err := c.GetAgent(context.Background(), id)
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
				{Header: "HOST", Field: "agentHost", Width: 30},
				{Header: "ACTIVE", Field: "active", Width: 8},
				{Header: "READY", Field: "readyToRun", Width: 8},
				{Header: "PLATFORM", Field: "platform", Width: 10},
				{Header: "VERSION", Field: "agentVersion", Width: 15},
				{Header: "UPGRADE STATUS", Field: "upgradeStatus", Width: 15},
				{Header: "GROUP ID", Field: "agentGroupId", Width: 24},
				{Header: "CREATED BY", Field: "createdBy", Width: 20},
				{Header: "CREATE TIME", Field: "createTime", Width: 22},
				{Header: "UPDATE TIME", Field: "updateTime", Width: 22},
			}
			return f.Format(agent, columns)
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "agent ID (required)")
	return cmd
}

func newAgentDetailsCmd() *cobra.Command {
	var id string
	cmd := &cobra.Command{
		Use:   "details",
		Short: "Get agent service engine details",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" {
				return fmt.Errorf("--id is required")
			}
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			details, err := c.GetAgentDetails(context.Background(), id)
			if err != nil {
				return err
			}
			f, err := getFormatter()
			if err != nil {
				return err
			}
			// Print agent summary first
			agentColumns := []output.Column{
				{Header: "ID", Field: "id", Width: 24},
				{Header: "NAME", Field: "name", Width: 25},
				{Header: "HOST", Field: "agentHost", Width: 30},
				{Header: "ACTIVE", Field: "active", Width: 8},
				{Header: "READY", Field: "readyToRun", Width: 8},
				{Header: "VERSION", Field: "agentVersion", Width: 15},
			}
			if err := f.Format(&details.Agent, agentColumns); err != nil {
				return err
			}
			// Print engine service status
			if len(details.AgentEngineStatus) > 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "\nService Status:") //nolint:errcheck
				svcColumns := []output.Column{
					{Header: "SERVICE", Field: "appDisplayName", Width: 30},
					{Header: "APP NAME", Field: "appname", Width: 25},
					{Header: "VERSION", Field: "appversion", Width: 15},
					{Header: "STATUS", Field: "status", Width: 14},
					{Header: "SUB STATE", Field: "subState", Width: 14},
				}
				if err := f.Format(details.AgentEngineStatus, svcColumns); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "agent ID (required)")
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
			fmt.Fprintf(cmd.OutOrStdout(), "Service %s started on agent %s\n", service, id) //nolint:errcheck
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
			fmt.Fprintf(cmd.OutOrStdout(), "Service %s stopped on agent %s\n", service, id) //nolint:errcheck
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "agent ID (required)")
	cmd.Flags().StringVar(&service, "service", "", "service name (required)")
	return cmd
}
