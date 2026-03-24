package cmd

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/jbrazda/iics-cli/internal/client"
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
	cmd.AddCommand(newProfileEditCmd())
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
				defaultMark := ""
				if cfg.DefaultProfile == name {
					defaultMark = "yes"
				}
				rows = append(rows, map[string]interface{}{
					"name":     name,
					"default":  defaultMark,
					"region":   p.Region,
					"endpoint": p.LoginURL,
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
				{Header: "REGION", Field: "region"},
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

func newProfileEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit [name]",
		Short: "Edit an existing profile interactively",
		Long: `Interactively update credentials for an existing profile.
Current values are shown as defaults; press Enter to keep them.
After saving, validates the credentials by logging in and refreshes
the session cache with the org-specific API URLs discovered from the response.`,
		Example: `  iics profile edit
  iics profile edit qa`,
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
			existing := cfg.Profiles[name]
			if existing == nil {
				return fmt.Errorf("profile %q not found; use 'profile add' to create it", name)
			}

			// Interactive prompt with existing values as defaults.
			p, makeDefault, err := config.PromptProfile(existing, name)
			if err != nil {
				return err
			}

			// Validate credentials and discover org-specific URLs via login.
			loginURL, err := p.GetLoginURL()
			if err != nil {
				return err
			}
			c := client.NewClient(loginURL, p.Username, p.Password, client.WithVerbose(verbose))
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Validating credentials for %q...\n", p.Username)
			loginResp, err := c.Login(context.Background())
			if err != nil {
				return fmt.Errorf("login validation failed: %w", err)
			}

			// Update URL fields from login response.
			if len(loginResp.Products) > 0 {
				baseAPIURL := loginResp.Products[0].BaseAPIURL
				p.LoginURL = loginURL
				p.BaseAPIURL = baseAPIURL
				if p.CaiURL == "" {
					p.CaiURL = config.DeriveCaiURL(baseAPIURL)
				}
			}

			// Persist updated profile.
			cfg.Profiles[name] = p
			if makeDefault {
				cfg.DefaultProfile = name
			}
			if err := cfg.Save(cfgFile); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			// Refresh session cache.
			if err := saveSession(name, loginURL, c, loginResp); err != nil {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not cache session: %v\n", err)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Profile %q updated and session refreshed.\n", name)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  User:    %s\n", loginResp.UserInfo.Name)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Org:     %s (%s)\n", loginResp.UserInfo.OrgName, loginResp.UserInfo.OrgID)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  BaseURL: %s\n", c.BaseAPIURL())
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  CAI URL: %s\n", c.CAIURL())
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

			// Append session-derived fields from the cache.
			const noSession = "(no active session)"
			orgName, orgID, sessionUser, lastLogin, sessionExpires := noSession, noSession, noSession, noSession, noSession
			if cache, cacheErr := config.LoadSessionCache(""); cacheErr == nil {
				if entry, ok := cache.Sessions[name]; ok && entry != nil {
					orgName = entry.OrgName
					orgID = entry.OrgID
					sessionUser = entry.UserName
					if !entry.LastLoginTime.IsZero() {
						lastLogin = entry.LastLoginTime.UTC().Format("2006-01-02 15:04:05 UTC")
					} else if !entry.CreatedAt.IsZero() {
						lastLogin = entry.CreatedAt.UTC().Format("2006-01-02 15:04:05 UTC")
					}
					if !entry.CreatedAt.IsZero() {
						exp := entry.CreatedAt.Add(30 * time.Minute)
						expStr := exp.UTC().Format("2006-01-02 15:04:05 UTC")
						if entry.IsExpired() {
							expStr += " (expired)"
						}
						sessionExpires = expStr
					}
				}
			}
			rows = append(rows,
				map[string]interface{}{"field": "Org Name", "value": orgName},
				map[string]interface{}{"field": "Org ID", "value": orgID},
				map[string]interface{}{"field": "Session User", "value": sessionUser},
				map[string]interface{}{"field": "Last Login", "value": lastLogin},
				map[string]interface{}{"field": "Session Expires", "value": sessionExpires},
			)

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
