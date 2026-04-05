package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/config"
	"github.com/jbrazda/iics-cli/internal/output"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// sampleThemeCols and sampleThemeData provide a fixed two-row preview table
// rendered for each theme during interactive theme selection.
var sampleThemeCols = []output.Column{
	{Header: "NAME", Field: "name", Width: 12},
	{Header: "REGION", Field: "region", Width: 6},
	{Header: "STATUS", Field: "status", Width: 8},
}

var sampleThemeData = []map[string]interface{}{
	{"name": "dev-org", "region": "USW3", "status": "active"},
	{"name": "prod-org", "region": "EMEA", "status": "active"},
}

// promptThemeSelection renders a live sample for each theme and presents a numbered
// menu. currentTheme is shown as the default; pressing Enter keeps it.
// All output goes to stderr so stdout piping is not contaminated.
// Returns the selected theme name, or currentTheme if the user pressed Enter.
func promptThemeSelection(currentTheme string) (string, error) {
	themes := []string{"default", "minimal", "compact", "plain", "markdown", "gh"}

	_, _ = fmt.Fprint(os.Stderr, "\nTable theme selection - preview of each theme:\n\n")

	for i, theme := range themes {
		_, _ = fmt.Fprintf(os.Stderr, "  [%d] %s\n", i+1, theme)
		f := output.New(output.FormatTable, os.Stderr, output.TableStyle{Theme: theme})
		_ = f.Format(sampleThemeData, sampleThemeCols)
		_, _ = fmt.Fprintln(os.Stderr)
	}

	currentIdx := 0
	for i, t := range themes {
		if t == currentTheme {
			currentIdx = i + 1
			break
		}
	}

	r := bufio.NewReader(os.Stdin)
	for {
		if currentIdx > 0 {
			_, _ = fmt.Fprintf(os.Stderr,
				"Select theme [1-%d, current: %d (%s), Enter to keep]: ",
				len(themes), currentIdx, currentTheme)
		} else {
			_, _ = fmt.Fprintf(os.Stderr, "Select theme [1-%d]: ", len(themes))
		}

		line, err := r.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("reading theme selection: %w", err)
		}
		line = strings.TrimSpace(line)

		if line == "" {
			return currentTheme, nil
		}

		var n int
		if _, scanErr := fmt.Sscanf(line, "%d", &n); scanErr == nil && n >= 1 && n <= len(themes) {
			return themes[n-1], nil
		}
		_, _ = fmt.Fprintf(os.Stderr,
			"Invalid selection %q. Enter a number from 1 to %d.\n", line, len(themes))
	}
}

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
	cmd.AddCommand(newProfileSetPasswordCmd())

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

			p, makeDefault, storeInKeyring, err := config.PromptProfile(existing, name)
			if err != nil {
				return err
			}

			// Keyring storage: save password in OS keychain and write sentinel to config.
			plainPassword := p.Password
			if storeInKeyring {
				if keyErr := config.SetKeychainPassword(name, plainPassword); keyErr != nil {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(),
						"Warning: could not store password in keychain: %v\n"+
							"  Storing password in config file instead.\n", keyErr)
				} else {
					p.Password = config.KeyringSentinel
				}
			}

			if cfg.Profiles == nil {
				cfg.Profiles = make(map[string]*config.Profile)
			}
			cfg.Profiles[name] = p
			if makeDefault {
				cfg.DefaultProfile = name
			}

			// Interactive theme selection (terminal only)
			if config.IsTerminal() {
				selected, styleErr := promptThemeSelection(cfg.Style.Theme)
				if styleErr != nil {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(),
						"Warning: theme selection skipped: %v\n", styleErr)
				} else if selected != "" {
					cfg.Style.Theme = selected
				}
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
			p, makeDefault, storeInKeyring, err := config.PromptProfile(existing, name)
			if err != nil {
				return err
			}

			// Keyring storage: save password in OS keychain and write sentinel to config.
			plainPassword := p.Password
			if storeInKeyring {
				if keyErr := config.SetKeychainPassword(name, plainPassword); keyErr != nil {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(),
						"Warning: could not store password in keychain: %v\n"+
							"  Storing password in config file instead.\n", keyErr)
				} else {
					p.Password = config.KeyringSentinel
				}
			}

			// Validate credentials and discover org-specific URLs via login.
			// Use the plain password for login even if the sentinel will be stored.
			loginURL, err := p.GetLoginURL()
			if err != nil {
				return err
			}
			c := client.NewClient(loginURL, p.Username, plainPassword, client.WithVerbose(verbose))
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

			// Interactive theme selection (terminal only)
			if config.IsTerminal() {
				selected, styleErr := promptThemeSelection(cfg.Style.Theme)
				if styleErr != nil {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(),
						"Warning: theme selection skipped: %v\n", styleErr)
				} else if selected != "" {
					cfg.Style.Theme = selected
				}
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

			// Best-effort: remove keychain entry for the deleted profile.
			if err := config.DeleteKeychainPassword(name); err != nil {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not remove keychain entry: %v\n", err)
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

func newProfileSetPasswordCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set-password [name]",
		Short: "Store the profile password in the OS keychain",
		Long: `Prompts for a password and stores it in the OS keychain (macOS Keychain,
Windows Credential Manager, or Linux Secret Service). The config file is
updated to use the "@keyring" sentinel so the plaintext password is no
longer stored on disk.`,
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
			if cfg.Profiles[name] == nil {
				return fmt.Errorf("profile %q not found; use 'profile add' to create it first", name)
			}

			_, _ = fmt.Fprintf(os.Stderr, "Enter new password for profile %q (input is masked): ", name)
			pw, err := term.ReadPassword(int(os.Stdin.Fd()))
			_, _ = fmt.Fprintln(os.Stderr)
			if err != nil {
				return fmt.Errorf("reading password: %w", err)
			}
			if len(pw) == 0 {
				return fmt.Errorf("password must not be empty")
			}

			if err := config.SetKeychainPassword(name, string(pw)); err != nil {
				return fmt.Errorf("keychain store failed: %w\n  Use 'iics profile edit' to update the password in the config file instead", err)
			}

			cfg.Profiles[name].Password = config.KeyringSentinel
			if err := cfg.Save(cfgFile); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(),
				"Password for profile %q stored in OS keychain.\n", name)
			return nil
		},
	}
}
