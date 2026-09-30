package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// confirmAction asks question followed by " [y/N]: " on the command's output
// and reads one line from stdin. It returns true only for "y" or "Y"; any other
// answer prints "Canceled." and returns false. The plain line format is kept
// deliberately so scripts can pipe an answer.
func confirmAction(cmd *cobra.Command, question string) bool {
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s [y/N]: ", question)
	var confirm string
	_, _ = fmt.Scanln(&confirm)
	if confirm != "y" && confirm != "Y" {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Canceled.")
		return false
	}
	return true
}
