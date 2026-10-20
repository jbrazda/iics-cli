package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/config"
	"github.com/jbrazda/iics-cli/internal/output"
	"github.com/jbrazda/iics-cli/internal/tui"
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

// themeNames are the table themes offered by config edit.
var themeNames = []string{"default", "minimal", "compact", "plain", "markdown", "gh"}

// themePreview renders the sample table in a theme as on a terminal.
func themePreview(theme, headerColor string) string {
	var b strings.Builder
	f := output.New(output.FormatTable, &b, output.TableStyle{Theme: theme, HeaderColor: headerColor, ForceTerminal: true})
	_ = f.Format(sampleThemeData, sampleThemeCols)
	return strings.TrimRight(b.String(), "\n")
}

// promptProfile runs the profile wizard for profile name with the stored
// profile (if any) as the starting values. It returns nil when the user
// canceled.
func promptProfile(cfg *config.Config, name string) (*tui.ProfileWizardResult, error) {
	if !config.IsTerminal() {
		return nil, errors.New("stdin is not a terminal; use --profile flag or IICS_* env vars")
	}
	return tui.RunProfileWizard(tui.ProfileWizardInput{
		Name:       name,
		Existing:   cfg.Profiles[name],
		Global:     cfg.GlobalNewUser(),
		IsDefault:  cfg.DefaultProfile == name,
		Out:        os.Stderr,
		Accessible: prompter.Accessible,
	})
}

// storeProfile puts the wizard result into cfg as profile name, moving a new
// password to the OS keychain when requested. It does not save cfg.
func storeProfile(cfg *config.Config, name string, res *tui.ProfileWizardResult, errOut io.Writer) {
	p := res.Profile
	if config.IsKeyringSentinel(p.Password) {
		// The existing keychain entry is kept; never write the sentinel to it.
		p.Password = config.KeyringSentinel
	} else if res.StoreInKeychain {
		if err := config.SetKeychainPassword(name, p.Password); err != nil {
			_, _ = fmt.Fprintf(errOut,
				"Warning: could not store password in keychain: %v\n"+
					"  Storing password in config file instead.\n", err)
		} else {
			p.Password = config.KeyringSentinel
		}
	}
	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]*config.Profile)
	}
	cfg.Profiles[name] = p
	if res.MakeDefault {
		cfg.DefaultProfile = name
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
				keychain := "no"
				if config.IsKeyringSentinel(p.Password) {
					keychain = "yes"
				}
				rows = append(rows, map[string]interface{}{
					"name":     name,
					"default":  defaultMark,
					"region":   p.Region,
					"endpoint": p.LoginURL,
					"username": p.Username,
					"keychain": keychain,
				})
			}

			f, err := getFormatter()
			if err != nil {
				return err
			}
			columns := []output.Column{
				{Header: "NAME", Field: "name", Priority: 1},
				{Header: "USERNAME", Field: "username", Priority: 2, Shrink: output.ShrinkTruncate},
				{Header: "ENDPOINT", Field: "endpoint", Priority: 3, Shrink: output.ShrinkTruncate},
				{Header: "POD", Field: "region", Width: 8, Priority: 2, Shrink: output.ShrinkNever},
				{Header: "DEFAULT", Field: "default", Width: 8, Priority: 2, Shrink: output.ShrinkNever},
				{Header: "KEYCHAIN", Field: "keychain", Width: 8, Priority: 2, Shrink: output.ShrinkNever},
			}
			return f.Format(rows, columns)
		},
	}
}

func newProfileAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add [name]",
		Short: "Add or update a profile interactively",
		Long: `Add a profile in an interactive form (connection, options, new user
defaults), followed by a review. If name is omitted, the profile is saved as
"default". An existing profile of that name is edited instead.`,
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

			res, err := promptProfile(cfg, name)
			if err != nil {
				return err
			}
			if res == nil {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Canceled.")
				return nil
			}
			storeProfile(cfg, name, res, cmd.ErrOrStderr())

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
		Long: `Edit an existing profile in an interactive form (connection, options,
new user defaults) with the current values filled in, followed by a review.
After the review, validates the credentials by logging in and refreshes the
session cache with the org-specific API URLs discovered from the response.
The profile is not saved when the login fails.`,
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

			res, err := promptProfile(cfg, name)
			if err != nil {
				return err
			}
			if res == nil {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Canceled.")
				return nil
			}
			p := res.Profile
			plainPassword := p.Password

			// Validate credentials and discover org-specific URLs via login.
			// When the password is the sentinel (user kept existing keychain password),
			// retrieve the real password from the keychain for the login call.
			passwordForLogin := plainPassword
			if config.IsKeyringSentinel(passwordForLogin) {
				if kp, kerr := config.GetKeychainPassword(name); kerr == nil {
					passwordForLogin = kp
				}
			}
			loginURL, err := p.GetLoginURL()
			if err != nil {
				return err
			}
			c := client.NewClient(loginURL, p.Username, passwordForLogin, client.WithVerbose(verbose))
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
				p.CaiURL = config.DeriveCaiURL(baseAPIURL)
			}

			// Persist updated profile (moves a new password to the keychain).
			storeProfile(cfg, name, res, cmd.ErrOrStderr())

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

			if !yes && !confirmAction(cmd, fmt.Sprintf("Delete profile %q?", name)) {
				return nil
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

			var maskedPassword string
			switch {
			case config.IsKeyringSentinel(p.Password):
				maskedPassword = "*** (keychain)"
			case p.Password != "":
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
			production := "no"
			if config.IsProductionProfile(name, p) {
				production = "yes"
			}
			nu, src := config.ResolveNewUser(cfg.GlobalNewUser(), p)
			withSource := func(v, source string) string {
				if v == "" {
					return ""
				}
				return v + " (" + source + ")"
			}
			rows = append(rows,
				map[string]interface{}{"field": "Production", "value": production},
				map[string]interface{}{"field": "New User Domain", "value": withSource(nu.Domain, src.Domain)},
				map[string]interface{}{"field": "User Name Pattern", "value": withSource(nu.UserNamePattern, src.UserNamePattern)},
				map[string]interface{}{"field": "Email Pattern", "value": withSource(nu.EmailPattern, src.EmailPattern)},
			)

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
				{Header: "FIELD", Field: "field", Width: 17},
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
