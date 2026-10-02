package cmd

import (
	"fmt"
	"os"

	"github.com/jbrazda/iics-cli/internal/config"
	"github.com/jbrazda/iics-cli/internal/tui"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage global settings",
		Long:  `Manage the global settings in the config file (~/.iics/config.yaml).`,
	}
	cmd.AddCommand(newConfigEditCmd())
	return cmd
}

func newConfigEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit",
		Short: "Edit global settings interactively",
		Long: `Edit the global settings in an interactive form, followed by a review
of the changes:

  New user    domain, user name pattern and email pattern defaults for all
              profiles (profiles can override each value)
  Appearance  table theme (with a preview), header color, no color,
              responsive tables
  Other       HTTP timeout, main menu for bare "iics"

Profile settings are edited with "iics profile edit". Requires a terminal.`,
		Example: `  iics config edit`,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !isInteractiveTTY() {
				return fmt.Errorf("config edit requires a terminal; edit %s directly in scripts", configFilePath())
			}
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			saved, err := tui.RunConfigWizard(tui.ConfigWizardInput{
				Config:       cfg,
				Themes:       themeNames,
				ThemePreview: themePreview,
				Out:          os.Stderr,
				Accessible:   prompter.Accessible,
			})
			if err != nil {
				return err
			}
			if !saved {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "No changes saved.")
				return nil
			}
			if err := cfg.Save(cfgFile); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Global settings saved.")
			return nil
		},
	}
}

// configFilePath returns the config file in use (--config or the default).
func configFilePath() string {
	if cfgFile != "" {
		return cfgFile
	}
	return config.DefaultConfigPath()
}
