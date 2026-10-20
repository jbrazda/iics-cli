package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/output"
	"github.com/spf13/cobra"
)

func newConnectorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "connector",
		Short: "List connectors and their connection attributes",
	}

	cmd.AddCommand(newConnectorListCmd())
	cmd.AddCommand(newConnectorMetadataCmd())
	return cmd
}

func newConnectorListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List the connectors available to the organization",
		Example: `  iics connector list
  iics connector list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			connectors, err := c.ListConnectors(context.Background())
			if err != nil {
				return err
			}

			f, err := getFormatter()
			if err != nil {
				return err
			}

			columns := []output.Column{
				{Header: "NAME", Field: "name", Width: 30, Priority: 1},
				{Header: "SHORT NAME", Field: "shortName", Width: 20, Priority: 3},
				{Header: "SDK", Field: "sdkType", Width: 5, Priority: 2, Shrink: output.ShrinkNever},
				{Header: "VERSION", Field: "connectorVersion", Width: 10, Priority: 4, Shrink: output.ShrinkNever},
				{Header: "PUBLIC", Field: "isPublic", Width: 6, Priority: 5, Shrink: output.ShrinkNever},
				{Header: "UPDATED", Field: "updateTime", Width: 20, Priority: 4, Shrink: output.ShrinkNever},
			}

			return f.Format(connectors, columns)
		},
	}
	return cmd
}

func newConnectorMetadataCmd() *cobra.Command {
	var (
		name  string
		exact bool
	)

	cmd := &cobra.Command{
		Use:   "metadata",
		Short: "Show the connection attributes of a connector",
		Long: `Show the connection attributes of a connector, the fields a connection of
that type takes.

--name is the connector name from "connector list" (not the short name).
The API matches it as a prefix, so "Oracle" also returns "Oracle Autonomous
Database". When one connector matches (or --exact selects one), its attributes
are listed; otherwise the matching connectors are summarized.

For CCI connectors the attributes are read from the connector's CCI definition.`,
		Example: `  iics connector metadata --name SqlServer
  iics connector metadata --name Oracle --exact
  iics connector metadata --name "Amazon S3 v2" --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			metas, err := c.GetConnectorMetadata(context.Background(), name)
			if err != nil {
				return err
			}
			if exact {
				var matched []client.ConnectorMetadata
				for _, m := range metas {
					if strings.EqualFold(m.Type, name) {
						matched = append(matched, m)
					}
				}
				metas = matched
			}
			if len(metas) == 0 {
				return fmt.Errorf("no connector named %q", name)
			}

			f, err := getFormatter()
			if err != nil {
				return err
			}

			if outputFmt != "" && outputFmt != "table" {
				return f.Format(metas, nil)
			}

			if len(metas) > 1 {
				type summary struct {
					Type       string `json:"type"`
					Standard   bool   `json:"isStandardConnType"`
					CCI        bool   `json:"cci"`
					Attributes int    `json:"attributes"`
				}
				rows := make([]summary, 0, len(metas))
				for _, m := range metas {
					fields, ferr := m.Fields()
					if ferr != nil {
						return ferr
					}
					rows = append(rows, summary{Type: m.Type, Standard: m.IsStandardConnType, CCI: m.CCIMeta != "", Attributes: len(fields)})
				}
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "%d connectors match %q; use the full name with --exact to list attributes.\n", len(metas), name)
				return f.Format(rows, []output.Column{
					{Header: "CONNECTOR", Field: "type", Width: 40},
					{Header: "STANDARD", Field: "isStandardConnType", Width: 8},
					{Header: "CCI", Field: "cci", Width: 5},
					{Header: "ATTRIBUTES", Field: "attributes", Width: 10},
				})
			}

			fields, err := metas[0].Fields()
			if err != nil {
				return err
			}
			columns := []output.Column{
				{Header: "NAME", Field: "name", Width: 30, Priority: 1},
				{Header: "LABEL", Field: "label", Width: 25, Priority: 4},
				{Header: "DATA TYPE", Field: "dataType", Width: 10, Priority: 2, Shrink: output.ShrinkNever},
				{Header: "MANDATORY", Field: "mandatory", Width: 9, Priority: 2, Shrink: output.ShrinkNever},
				{Header: "ENCRYPTED", Field: "encrypted", Width: 9, Priority: 5, Shrink: output.ShrinkNever},
				{Header: "DEFAULT", Field: "defaultValue", Width: 15, Priority: 3},
				{Header: "VALUES", Field: "values", MaxWidth: 60, Priority: 4, Shrink: output.ShrinkWrap},
			}
			return f.Format(fields, columns)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "connector name, e.g. SqlServer, Salesforce, \"Amazon S3 v2\" (required)")
	cmd.Flags().BoolVar(&exact, "exact", false, "only the connector whose name matches exactly (case-insensitive)")
	return cmd
}
