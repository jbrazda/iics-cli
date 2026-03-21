package cmd

import (
	"fmt"
	"sort"

	"github.com/jbrazda/iics-cli/internal/config"
	"github.com/jbrazda/iics-cli/internal/output"
	"github.com/spf13/cobra"
)

func newProfileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Manage IICS connection profiles",
		Long: `Manage IICS connection profiles stored in ~/.iics/config.yaml.

Profiles store the credentials and region needed to connect to an IICS org.
Use 'profile add' to create a profile interactively, 'profile list' to see
all configured profiles, and 'profile set-default' to choose the active one.`,
	}

	cmd.AddCommand(newProfileListCmd())
	cmd.AddCommand(newProfileAddCmd())
	cmd.AddCommand(newProfileDeleteCmd())
	cmd.AddCommand(newProfileSetDefaultCmd())
	cmd.AddCommand(newProfileShowCmd())

	return cmd
}

func newProfileListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all configured profiles",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			names := make([]string, 0, len(cfg.Profiles))
			for name := range cfg.Profiles {
				names = append(names, name)
			}
			sort.Strings(names)

			rows := make([]map[string]interface{}, 0, len(names))
			for _, name := range names {
				p := cfg.Profiles[name]
				endpoint := p.Region
				if p.LoginURL != "" {
					endpoint = p.LoginURL
				}
				defaultMark := ""
				if cfg.DefaultProfile == name {
					defaultMark = "yes"
				}
				rows = append(rows, map[string]interface{}{
					"name":     name,
					"default":  defaultMark,
					"endpoint": endpoint,
					"username": p.Username,
				})
			}

			f, err := getFormatter()
			if err != nil {
				return err
			}
			columns := []output.Column{
				{Header: "NAME", Field: "name"},
				{Header: "DEFAULT", Field: "default"},
				{Header: "ENDPOINT", Field: "endpoint"},
				{Header: "USERNAME", Field: "username"},
			}
			return f.Format(rows, columns)
		},
	}
}

func newProfileAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add [name]",
		Short: "Add or update a profile interactively",
		Long: `Interactively prompt for credentials and save them as a named profile.
If name is omitted, the profile is saved as "default".`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := "default"
			if len(args) == 1 {
				name = args[0]
			}

			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			existing := cfg.Profiles[name] // nil if not found

			p, makeDefault, err := config.PromptProfile(existing, name)
			if err != nil {
				return err
			}

			if cfg.Profiles == nil {
				cfg.Profiles = make(map[string]*config.Profile)
			}
			cfg.Profiles[name] = p
			if makeDefault {
				cfg.DefaultProfile = name
			}

			if err := cfg.Save(cfgFile); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Profile %q saved.\n", name)
			return nil
		},
	}
}

func newProfileDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			if _, ok := cfg.Profiles[name]; !ok {
				return fmt.Errorf("profile %q not found", name)
			}

			if !yes {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Delete profile %q? [y/N]: ", name)
				var confirm string
				_, _ = fmt.Scanln(&confirm)
				if confirm != "y" && confirm != "Y" {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Canceled.")
					return nil
				}
			}

			delete(cfg.Profiles, name)
			if cfg.DefaultProfile == name {
				cfg.DefaultProfile = ""
			}

			if err := cfg.Save(cfgFile); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			// Best-effort: remove cached session for the deleted profile.
			if cache, err := config.LoadSessionCache(""); err == nil {
				cache.Delete(name)
				if err := cache.Save(""); err != nil {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not remove cached session: %v\n", err)
				}
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Profile %q deleted.\n", name)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompt")
	return cmd
}

func newProfileSetDefaultCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set-default <name>",
		Short: "Set the default profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			if _, ok := cfg.Profiles[name]; !ok {
				return fmt.Errorf("profile %q not found", name)
			}

			cfg.DefaultProfile = name

			if err := cfg.Save(cfgFile); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Default profile set to %q.\n", name)
			return nil
		},
	}
}

func newProfileShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Show details of a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			p, ok := cfg.Profiles[name]
			if !ok {
				return fmt.Errorf("profile %q not found", name)
			}

			maskedPassword := ""
			if p.Password != "" {
				maskedPassword = "***"
			}
			defaultMark := ""
			if cfg.DefaultProfile == name {
				defaultMark = "yes"
			}

			rows := []map[string]interface{}{
				{"field": "Name", "value": name},
				{"field": "Default", "value": defaultMark},
				{"field": "Region", "value": p.Region},
				{"field": "Login URL", "value": p.LoginURL},
				{"field": "Base API URL", "value": p.BaseAPIURL},
				{"field": "CAI URL", "value": p.CaiURL},
				{"field": "Username", "value": p.Username},
				{"field": "Password", "value": maskedPassword},
			}

			f, err := getFormatter()
			if err != nil {
				return err
			}
			columns := []output.Column{
				{Header: "FIELD", Field: "field", Width: 15},
				{Header: "VALUE", Field: "value"},
			}
			return f.Format(rows, columns)
		},
	}
}
