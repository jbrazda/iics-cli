package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/config"
	"github.com/jbrazda/iics-cli/internal/output"
	"github.com/spf13/cobra"
)

var (
	cfgFile    string
	profile    string
	outputFmt  string
	verbose    bool
	noColor    bool
	debug      bool
	versionStr = "dev"
	commitStr  = "none"
	dateStr    = "unknown"
)

// SetVersionInfo sets the version information from build flags.
func SetVersionInfo(version, commit, date string) {
	versionStr = version
	commitStr = commit
	dateStr = date
}

var rootCmd = &cobra.Command{
	Use:     "iics",
	Short:   "CLI for Informatica Intelligent Cloud Services (IICS) REST API v3",
	Version: "dev",
	Long: `A comprehensive command-line interface for managing Informatica IICS
resources including connections, mappings, tasks, exports, imports,
users, roles, and more.

Configure profiles in ~/.iics/config.yaml or use environment variables
(IICS_USERNAME, IICS_PASSWORD, IICS_REGION) for authentication.`,
	// Silence Cobra's default error and usage printing so we control output.
	SilenceErrors: true,
	SilenceUsage:  true,
}

// Execute runs the root command and returns a POSIX exit code.
//
//	0 — success
//	1 — runtime / API error
//	2 — usage error (bad flags, missing arguments)
func Execute() int {
	rootCmd.Version = fmt.Sprintf("%s (commit: %s, built: %s)", versionStr, commitStr, dateStr)

	err := rootCmd.Execute()
	if err == nil {
		return client.ExitOK
	}

	return handleError(err)
}

// handleError formats the error to stderr and returns the appropriate exit code.
func handleError(err error) int {
	var apiErr *client.APIError
	if errors.As(err, &apiErr) {
		// Print short summary line
		_, _ = fmt.Fprintf(os.Stderr, "Error: %s\n", apiErr.Error())
		// Print full HTTP details (status, headers, formatted JSON body)
		_, _ = fmt.Fprint(os.Stderr, apiErr.Verbose())
		return client.ExitError
	}

	// Check if this is a Cobra usage error (unknown command, bad flags, etc.)
	if isUsageError(err) {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		return client.ExitUsageError
	}

	// General runtime error
	_, _ = fmt.Fprintf(os.Stderr, "Error: %s\n", err)
	return client.ExitError
}

// isUsageError returns true if the error originated from invalid command usage.
// Cobra wraps these with specific prefixes.
func isUsageError(err error) bool {
	msg := err.Error()
	for _, prefix := range []string{
		"unknown command",
		"unknown flag",
		"unknown shorthand flag",
		"required flag",
		"invalid argument",
		"accepts ", // "accepts N arg(s)"
		"at least", // "at least N arg(s)"
		"at most",  // "at most N arg(s)"
		"flag needs an argument",
	} {
		if len(msg) >= len(prefix) && msg[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default ~/.iics/config.yaml)")
	rootCmd.PersistentFlags().StringVarP(&profile, "profile", "p", "", "IICS profile name (default from config)")
	rootCmd.PersistentFlags().StringVarP(&outputFmt, "output", "o", "table", "output format: table|json|csv")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "print request body to stderr on API error")

	rootCmd.AddCommand(newLoginCmd())
	rootCmd.AddCommand(newLogoutCmd())
	rootCmd.AddCommand(newObjectsCmd())
	rootCmd.AddCommand(newLookupCmd())
	rootCmd.AddCommand(newConnectionCmd())
	rootCmd.AddCommand(newExportCmd())
	rootCmd.AddCommand(newImportCmd())
	rootCmd.AddCommand(newScheduleCmd())
	rootCmd.AddCommand(newProjectCmd())
	rootCmd.AddCommand(newFolderCmd())
	rootCmd.AddCommand(newUserCmd())
	rootCmd.AddCommand(newUsergroupCmd())
	rootCmd.AddCommand(newRoleCmd())
	rootCmd.AddCommand(newPrivilegeCmd())
	rootCmd.AddCommand(newRuntimeCmd())
	rootCmd.AddCommand(newAgentCmd())
	rootCmd.AddCommand(newTagCmd())
	rootCmd.AddCommand(newPermissionCmd())
	rootCmd.AddCommand(newSecuritylogCmd())
	rootCmd.AddCommand(newMeteringCmd())
	rootCmd.AddCommand(newSourcecontrolCmd())
	rootCmd.AddCommand(newStateCmd())
	rootCmd.AddCommand(newProfileCmd())
	rootCmd.AddCommand(newPackageCmd())
	rootCmd.AddCommand(newActivitylogCmd())
	rootCmd.AddCommand(newPublishCmd())
	rootCmd.AddCommand(newUnpublishCmd())
	rootCmd.AddCommand(newCompletionCmd())
}

// loadConfig loads and returns the configuration.
func loadConfig() (*config.Config, error) {
	return config.Load(cfgFile)
}

// resolveProfile loads config and resolves the active profile.
func resolveProfile() (*config.Config, *config.Profile, string, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, nil, "", err
	}

	profileName := profile
	if profileName == "" {
		if v := os.Getenv("IICS_PROFILE"); v != "" {
			profileName = v
		} else {
			profileName = cfg.DefaultProfile
		}
	}
	if profileName == "" {
		profileName = "default"
	}

	p, err := cfg.ResolveProfile(profileName)
	if err != nil {
		return nil, nil, "", err
	}

	return cfg, p, profileName, nil
}

// isMissingCredentialsError returns true for errors from ResolveProfile when
// username or password is not configured.
func isMissingCredentialsError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "username not configured") ||
		strings.Contains(msg, "password not configured")
}

// isProfileCommand returns true if cmd or any ancestor is the "profile" command.
// Used to prevent the auto-setup wizard from triggering recursively.
func isProfileCommand(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if c.Use == "profile" {
			return true
		}
	}
	return false
}

// getClient creates an authenticated IICS API client.
// It attempts to use a cached session first, then falls back to login.
// When no profile credentials are configured and stdin is a terminal, it
// launches an interactive setup wizard before proceeding.
func getClient(cmd *cobra.Command) (*client.Client, error) {
	_, p, profileName, err := resolveProfile()
	if err != nil {
		if isMissingCredentialsError(err) && config.IsTerminal() && !isProfileCommand(cmd) {
			_, _ = fmt.Fprintf(os.Stderr,
				"\nNo credentials configured for profile %q.\nLet's set up a profile now.\n\n",
				profileName)
			cfg, loadErr := config.Load(cfgFile)
			if loadErr != nil {
				return nil, err
			}
			newProfile, makeDefault, promptErr := config.PromptProfile(cfg.Profiles[profileName], profileName)
			if promptErr != nil {
				return nil, fmt.Errorf("profile setup: %w", promptErr)
			}
			if cfg.Profiles == nil {
				cfg.Profiles = make(map[string]*config.Profile)
			}
			cfg.Profiles[profileName] = newProfile
			if makeDefault {
				cfg.DefaultProfile = profileName
			}
			if saveErr := cfg.Save(cfgFile); saveErr != nil {
				return nil, fmt.Errorf("saving profile: %w", saveErr)
			}
			_, _ = fmt.Fprintf(os.Stderr, "\nProfile %q saved.\n\n", profileName)
			_, p, profileName, err = resolveProfile()
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	loginURL, err := p.GetLoginURL()
	if err != nil {
		return nil, err
	}

	opts := []client.ClientOption{client.WithVerbose(verbose), client.WithDebug(debug)}
	if p.CaiURL != "" {
		opts = append(opts, client.WithCAIURL(p.CaiURL))
	}
	c := client.NewClient(loginURL, p.Username, p.Password, opts...)

	// Try to load cached session
	cache, err := config.LoadSessionCache("")
	if err == nil {
		if entry, ok := cache.Get(profileName); ok {
			c.SetSession(entry.SessionID, entry.BaseAPIURL)
			return c, nil
		}
	}

	return c, nil
}

// getFormatter returns a formatter for the current output format.
func getFormatter() (output.Formatter, error) {
	f, err := output.ParseFormat(outputFmt)
	if err != nil {
		return nil, err
	}
	return output.New(f, os.Stdout), nil
}

// saveSession saves the current session to the cache.
func saveSession(profileName string, c *client.Client, loginResp *client.LoginResponse) error {
	cache, err := config.LoadSessionCache("")
	if err != nil {
		cache = &config.SessionCache{Sessions: make(map[string]*config.SessionEntry)}
	}

	cache.Set(profileName, &config.SessionEntry{
		SessionID:  loginResp.UserInfo.SessionID,
		BaseAPIURL: c.BaseAPIURL(),
		OrgID:      loginResp.UserInfo.OrgID,
		OrgName:    loginResp.UserInfo.OrgName,
		UserName:   loginResp.UserInfo.Name,
	})

	if err := cache.Save(""); err != nil {
		return fmt.Errorf("saving session cache: %w", err)
	}

	return nil
}
