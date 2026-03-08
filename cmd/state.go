package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newStateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "state",
		Short: "Manage object state (fetchState/loadState)",
		Long:  `Fetch and load object state for migration between organizations.`,
	}
	cmd.AddCommand(newStateFetchCmd())
	cmd.AddCommand(newStateLoadCmd())
	return cmd
}

func newStateFetchCmd() *cobra.Command {
	var (
		objectID   string
		outputFile string
	)
	cmd := &cobra.Command{
		Use:     "fetch",
		Short:   "Fetch object state",
		Example: `  iics state fetch --object-id <id> --output-file state.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if objectID == "" {
				return fmt.Errorf("--object-id is required")
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			var dest *os.File
			if outputFile != "" {
				dest, err = os.Create(outputFile)
				if err != nil {
					return fmt.Errorf("creating output file: %w", err)
				}
				defer dest.Close()
			} else {
				dest = os.Stdout
			}

			if err := c.FetchState(context.Background(), objectID, dest); err != nil {
				if outputFile != "" {
					os.Remove(outputFile)
				}
				return err
			}

			if outputFile != "" {
				fmt.Fprintf(cmd.ErrOrStderr(), "State fetched to %s\n", outputFile)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&objectID, "object-id", "", "object ID (required)")
	cmd.Flags().StringVarP(&outputFile, "output-file", "f", "", "output file (default: stdout)")
	return cmd
}

func newStateLoadCmd() *cobra.Command {
	var (
		objectID  string
		inputFile string
	)
	cmd := &cobra.Command{
		Use:     "load",
		Short:   "Load object state",
		Example: `  iics state load --object-id <id> --input-file state.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if objectID == "" {
				return fmt.Errorf("--object-id is required")
			}
			if inputFile == "" {
				return fmt.Errorf("--input-file is required")
			}

			file, err := os.Open(inputFile)
			if err != nil {
				return fmt.Errorf("opening input file: %w", err)
			}
			defer file.Close()

			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			result, err := c.LoadState(context.Background(), objectID, file)
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "State loaded for object %s (status: %s)\n", objectID, result.Status)
			if result.Message != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "  Message: %s\n", result.Message)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&objectID, "object-id", "", "object ID (required)")
	cmd.Flags().StringVarP(&inputFile, "input-file", "f", "", "input file (required)")
	return cmd
}
