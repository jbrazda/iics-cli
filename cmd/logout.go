package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/jbrazda/iics-cli/internal/config"
	"github.com/spf13/cobra"
)

func newLogoutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Invalidate the current session",
		Long:  `Log out from IICS and remove the cached session.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			profileName := profile
			if profileName == "" {
				if v := os.Getenv("IICS_PROFILE"); v != "" {
					profileName = v
				}
			}
			if profileName == "" {
				cfg, err := loadConfig()
				if err == nil {
					profileName = cfg.DefaultProfile
				}
			}
			if profileName == "" {
				profileName = "default"
			}

			// Try to logout via API if we have a session
			c, err := getClient(cmd)
			if err == nil && c.SessionID() != "" {
				if err := c.Logout(context.Background()); err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "Warning: API logout failed: %v\n", err)
				}
			}

			// Remove cached session
			cache, err := config.LoadSessionCache("")
			if err == nil {
				cache.Delete(profileName)
				err = cache.Save("")
				if err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not update session cache: %v\n", err)
				}
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Logged out (profile: %s).\n", profileName)
			return nil
		},
	}

	return cmd
}
