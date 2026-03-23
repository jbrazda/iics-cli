package cmd

import (
	"context"
	"fmt"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/config"
	"github.com/spf13/cobra"
)

func newLoginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with IICS and store the session",
		Long: `Authenticate with the Informatica Intelligent Cloud Services API.
The session is cached locally so subsequent commands don't require re-authentication.`,
		Example: `  iics login
  iics login --profile prod
  IICS_USERNAME=user@company.com IICS_PASSWORD=secret IICS_REGION=USW3 iics login`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, p, profileName, err := resolveProfile()
			if err != nil {
				return err
			}

			loginURL, err := p.GetLoginURL()
			if err != nil {
				return err
			}

			c := client.NewClient(loginURL, p.Username, p.Password, client.WithVerbose(verbose))

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Logging in as %s...\n", p.Username)

			loginResp, err := c.Login(context.Background())
			if err != nil {
				return fmt.Errorf("login failed: %w", err)
			}

			// Persist discovered URLs back to the stored profile so subsequent
			// commands (e.g. publish, state) can find them without manual config.
			// Write directly to cfg.Profiles[profileName] (the original, unmodified
			// entry) rather than to p, which may hold env-var-overridden credentials.
			if len(loginResp.Products) > 0 {
				baseAPIURL := loginResp.Products[0].BaseAPIURL
				stored := cfg.Profiles[profileName]
				if stored == nil {
					stored = &config.Profile{}
					cfg.Profiles[profileName] = stored
				}
				stored.LoginURL = loginURL
				stored.BaseAPIURL = baseAPIURL
				if stored.CaiURL == "" {
					stored.CaiURL = config.DeriveCaiURL(baseAPIURL)
				}
				if saveErr := cfg.Save(cfgFile); saveErr != nil {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not update profile: %v\n", saveErr)
				}
			}

			// Save session to cache
			if err := saveSession(profileName, loginURL, c, loginResp); err != nil {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not cache session: %v\n", err)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Logged in successfully.\n")
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  User:    %s\n", loginResp.UserInfo.Name)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Org:     %s (%s)\n", loginResp.UserInfo.OrgName, loginResp.UserInfo.OrgID)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  BaseURL: %s\n", loginResp.Products[0].BaseAPIURL)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  CAI URL: %s\n", c.CAIURL())
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Profile: %s\n", profileName)

			return nil
		},
	}

	return cmd
}
