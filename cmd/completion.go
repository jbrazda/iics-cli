package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

func newCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for iics.

The scripts enable tab-completion of commands, subcommands, and flags.
They are generated dynamically from the current binary, so they stay in
sync with the installed version automatically.

See the subcommand for your shell for installation instructions.`,
	}
	cmd.AddCommand(newCompletionBashCmd())
	cmd.AddCommand(newCompletionZshCmd())
	cmd.AddCommand(newCompletionFishCmd())
	cmd.AddCommand(newCompletionPowerShellCmd())
	return cmd
}

func newCompletionBashCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "bash",
		Short: "Generate bash completion script",
		Long: `Generate a bash completion script for iics.

To load completions in the current shell session:

  source <(iics completion bash)

To install permanently (Linux):

  iics completion bash > /etc/bash_completion.d/iics

To install permanently (macOS with Homebrew bash-completion@2):

  iics completion bash > $(brew --prefix)/etc/bash_completion.d/iics`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenBashCompletion(os.Stdout)
		},
	}
}

func newCompletionZshCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "zsh",
		Short: "Generate zsh completion script",
		Long: `Generate a zsh completion script for iics.

To load completions in the current shell session:

  source <(iics completion zsh)

To install permanently, place the script on your fpath.
If shell completion is not already enabled in your zsh environment,
enable it first:

  echo "autoload -U compinit; compinit" >> ~/.zshrc

Then install the completion:

  iics completion zsh > "${fpath[1]}/_iics"

Start a new shell or run:

  source ~/.zshrc`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenZshCompletion(os.Stdout)
		},
	}
}

func newCompletionFishCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fish",
		Short: "Generate fish completion script",
		Long: `Generate a fish completion script for iics.

To load completions in the current shell session:

  iics completion fish | source

To install permanently:

  iics completion fish > ~/.config/fish/completions/iics.fish`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenFishCompletion(os.Stdout, true)
		},
	}
}

func newCompletionPowerShellCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "powershell",
		Short: "Generate PowerShell completion script",
		Long: `Generate a PowerShell completion script for iics.

To load completions in the current shell session:

  iics completion powershell | Out-String | Invoke-Expression

To install permanently, add the output to your PowerShell profile:

  iics completion powershell >> $PROFILE`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		},
	}
}
