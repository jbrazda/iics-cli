package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newTagCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tag",
		Short: "Manage tags on objects",
	}
	cmd.AddCommand(newTagAssignCmd())
	cmd.AddCommand(newTagRemoveCmd())
	return cmd
}

func newTagAssignCmd() *cobra.Command {
	var (
		objectID string
		tags     string
	)
	cmd := &cobra.Command{
		Use:     "assign",
		Short:   "Assign tags to an object",
		Example: `  iics tag assign --object-id <id> --tags "tag1,tag2"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if objectID == "" {
				return fmt.Errorf("--object-id is required")
			}
			if tags == "" {
				return fmt.Errorf("--tags is required")
			}
			tagList := strings.Split(tags, ",")
			for i := range tagList {
				tagList[i] = strings.TrimSpace(tagList[i])
			}
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			if err := c.AssignTags(context.Background(), objectID, tagList); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Tags assigned to object %s: %s\n", objectID, tags)
			return nil
		},
	}
	cmd.Flags().StringVar(&objectID, "object-id", "", "object ID (required)")
	cmd.Flags().StringVar(&tags, "tags", "", "comma-separated tags (required)")
	return cmd
}

func newTagRemoveCmd() *cobra.Command {
	var (
		objectID string
		tags     string
	)
	cmd := &cobra.Command{
		Use:     "remove",
		Short:   "Remove tags from an object",
		Example: `  iics tag remove --object-id <id> --tags "tag1,tag2"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if objectID == "" {
				return fmt.Errorf("--object-id is required")
			}
			if tags == "" {
				return fmt.Errorf("--tags is required")
			}
			tagList := strings.Split(tags, ",")
			for i := range tagList {
				tagList[i] = strings.TrimSpace(tagList[i])
			}
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			if err := c.RemoveTags(context.Background(), objectID, tagList); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Tags removed from object %s: %s\n", objectID, tags)
			return nil
		},
	}
	cmd.Flags().StringVar(&objectID, "object-id", "", "object ID (required)")
	cmd.Flags().StringVar(&tags, "tags", "", "comma-separated tags (required)")
	return cmd
}
