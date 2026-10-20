package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func newAgentRenameCmd() *cobra.Command {
	var (
		id, fid, name, hostname, newName string
		yes                              bool
	)
	cmd := &cobra.Command{
		Use:   "rename",
		Short: "Rename a Secure Agent (experimental)",
		Long: `Rename a Secure Agent. The installer registers an agent under its host name;
use this to give it a logical name, for example CAI-A03.

The new name must be unique in the organization and at most 100 characters.
The name is saved through the agent's Administrator form, like
"agent configs set", together with its unchanged service properties.

` + agentConfigsExperimental,
		Example: `  iics agent rename --hostname cletinfacld07 --new-name CAI-A03
  iics agent rename --name cletinfacld07.corp.example.com --new-name CAI-A03 --yes
  iics agent rename --id <agent-id> --new-name CDI-G01-A02`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" && fid == "" && name == "" && hostname == "" {
				return fmt.Errorf("one of --id, --fid, --name, or --hostname is required")
			}
			if newName == "" {
				return fmt.Errorf("--new-name is required")
			}
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()
			agent, err := resolveAgent(ctx, c, id, name, hostname, fid)
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			if agent.Name == newName {
				_, _ = fmt.Fprintf(w, "Agent %s is already named %s.\n", agent.ID, newName)
				return nil
			}
			if !yes && !confirmAction(cmd, fmt.Sprintf("Rename agent %q (%s, host %s) to %q?", agent.Name, agent.ID, agent.AgentHost, newName)) {
				return nil
			}
			renamed, err := c.RenameAgent(ctx, agent.ID, newName)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(w, "Renamed agent %s from %q to %q.\n", renamed.ID, agent.Name, renamed.Name)
			return nil
		},
	}
	addAgentSelectorFlags(cmd, &id, &fid, &name, &hostname)
	cmd.Flags().StringVar(&newName, "new-name", "", "new agent name (required)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "rename without asking for confirmation")
	return cmd
}
