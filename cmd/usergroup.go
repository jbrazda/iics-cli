package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/output"
	"github.com/spf13/cobra"
)

// allUsergroupColumns defines all available columns for usergroup output.
var allUsergroupColumns = map[string]output.Column{
	"id":            {Header: "ID", Field: "id", Width: 24},
	"userGroupName": {Header: "NAME", Field: "userGroupName", Width: 30},
	"description":   {Header: "DESCRIPTION", Field: "description", Width: 30},
	"updatedBy":     {Header: "UPDATED BY", Field: "updatedBy", Width: 30},
	"updateTime":    {Header: "UPDATED", Field: "updateTime", Width: 24},
	"createdBy":     {Header: "CREATED BY", Field: "createdBy", Width: 30},
	"createTime":    {Header: "CREATED", Field: "createTime", Width: 24},
	"countMembers":  {Header: "MEMBERS", Field: "countMembers", Width: 8},
	"countRoles":    {Header: "ROLES", Field: "countRoles", Width: 8},
}

func columnsFromFields(fields string) []output.Column {
	var columns []output.Column
	for _, name := range strings.Split(fields, ",") {
		name = strings.TrimSpace(name)
		if col, ok := allUsergroupColumns[name]; ok {
			columns = append(columns, col)
		}
	}
	return columns
}

func newUsergroupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "usergroup",
		Aliases: []string{"ug"},
		Short:   "Manage user groups",
	}

	cmd.AddCommand(newUsergroupListCmd())
	cmd.AddCommand(newUsergroupGetCmd())
	cmd.AddCommand(newUsergroupCreateCmd())
	cmd.AddCommand(newUsergroupUpdateCmd())
	cmd.AddCommand(newUsergroupDeleteCmd())
	return cmd
}

const (
	usergroupTableFields = "id,userGroupName,updatedBy,updateTime,countMembers,countRoles"
	usergroupCSVFields   = "id,userGroupName,updatedBy,updateTime,description,countMembers,countRoles"
)

func newUsergroupListCmd() *cobra.Command {
	var opts client.UserGroupListOptions
	var fields string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List user groups",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			groups, err := c.ListUserGroups(context.Background(), opts)
			if err != nil {
				return err
			}
			f, err := getFormatter()
			if err != nil {
				return err
			}
			if fields == "" {
				if outputFmt == "csv" {
					fields = usergroupCSVFields
				} else {
					fields = usergroupTableFields
				}
			}
			return f.Format(groups, columnsFromFields(fields))
		},
	}

	cmd.Flags().IntVar(&opts.Limit, "limit", 200, "max results")
	cmd.Flags().IntVar(&opts.Skip, "skip", 0, "number of results to skip")
	cmd.Flags().StringVarP(&opts.Query, "query", "q", "", `filter query (e.g. userGroupName=="Administrator")`)
	cmd.Flags().StringVar(&fields, "fields", "", "comma-separated list of fields to display (default varies by output format)")
	return cmd
}

func newUsergroupGetCmd() *cobra.Command {
	var id string
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get user group details",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" {
				return fmt.Errorf("--id is required")
			}
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			group, err := c.GetUserGroup(context.Background(), id)
			if err != nil {
				return err
			}
			f, err := getFormatter()
			if err != nil {
				return err
			}
			columns := []output.Column{
				{Header: "ID", Field: "id", Width: 24},
				{Header: "NAME", Field: "userGroupName", Width: 30},
				{Header: "DESCRIPTION", Field: "description"},
			}
			return f.Format(group, columns)
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "user group ID (required)")
	return cmd
}

func newUsergroupCreateCmd() *cobra.Command {
	var fromFile string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a user group",
		RunE: func(cmd *cobra.Command, args []string) error {
			if fromFile == "" {
				return fmt.Errorf("--from-file is required")
			}
			data, err := os.ReadFile(fromFile)
			if err != nil {
				return fmt.Errorf("reading file: %w", err)
			}
			var group client.UserGroup
			err = json.Unmarshal(data, &group)
			if err != nil {
				return fmt.Errorf("parsing JSON: %w", err)
			}
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			created, err := c.CreateUserGroup(context.Background(), &group)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "User group created: %s (ID: %s)\n", created.UserGroupName, created.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&fromFile, "from-file", "", "JSON file with group definition (required)")
	return cmd
}

func newUsergroupUpdateCmd() *cobra.Command {
	var (
		id       string
		fromFile string
	)
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a user group",
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
			var group client.UserGroup
			err = json.Unmarshal(data, &group)
			if err != nil {
				return fmt.Errorf("parsing JSON: %w", err)
			}
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			updated, err := c.UpdateUserGroup(context.Background(), id, &group)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "User group updated: %s (ID: %s)\n", updated.UserGroupName, updated.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "user group ID (required)")
	cmd.Flags().StringVar(&fromFile, "from-file", "", "JSON file with group updates (required)")
	return cmd
}

func newUsergroupDeleteCmd() *cobra.Command {
	var (
		id  string
		yes bool
	)
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a user group",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" {
				return fmt.Errorf("--id is required")
			}
			if !yes {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Are you sure you want to delete user group %s? [y/N]: ", id)
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
			if err := c.DeleteUserGroup(context.Background(), id); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "User group deleted: %s\n", id)
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "user group ID (required)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompt")
	return cmd
}
