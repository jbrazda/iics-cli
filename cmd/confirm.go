package cmd

import (
	"fmt"
	"os"

	"github.com/jbrazda/iics-cli/internal/config"
	"github.com/jbrazda/iics-cli/internal/tui"
	"github.com/spf13/cobra"
)

// confirmAction asks question followed by " [y/N]: " on the command's output.
// On a terminal the answer is a single key (y yes; n, Enter, Esc, q or ctrl+c
// no; arrow keys are ignored). Otherwise one line is read from stdin, so
// scripts can pipe an answer. It returns true only for yes; any other answer
// prints "Canceled." and returns false.
func confirmAction(cmd *cobra.Command, question string) bool {
	out := cmd.OutOrStdout()
	var yes bool
	if config.IsTerminal() {
		yes, _ = tui.KeyConfirm(os.Stdin, out, question, false)
	} else {
		_, _ = fmt.Fprintf(out, "%s [y/N]: ", question)
		var confirm string
		_, _ = fmt.Scanln(&confirm)
		yes = confirm == "y" || confirm == "Y"
	}
	if !yes {
		_, _ = fmt.Fprintln(out, "Canceled.")
	}
	return yes
}
