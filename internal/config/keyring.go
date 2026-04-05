package config

import (
	"fmt"

	"github.com/zalando/go-keyring"
)

const (
	// KeyringService is the service name used for all iics-cli keychain entries.
	KeyringService = "iics-cli"

	// KeyringSentinel is the placeholder stored in config.yaml when the real
	// password lives in the OS keychain.
	KeyringSentinel = "@keyring"
)

// IsKeyringSentinel reports whether the password field value indicates that
// the real credential should be fetched from the OS keychain.
func IsKeyringSentinel(password string) bool {
	return password == KeyringSentinel
}

// GetKeychainPassword retrieves the password for the given profile name from the
// OS keychain. Returns an error if the keychain is unavailable or the entry is not found.
func GetKeychainPassword(profileName string) (string, error) {
	pw, err := keyring.Get(KeyringService, profileName)
	if err != nil {
		return "", fmt.Errorf("keyring lookup for profile %q: %w", profileName, err)
	}
	return pw, nil
}

// SetKeychainPassword stores a password in the OS keychain for the given profile name.
// Overwrites any existing entry for the same profile.
func SetKeychainPassword(profileName, password string) error {
	if err := keyring.Set(KeyringService, profileName, password); err != nil {
		return fmt.Errorf("keyring store for profile %q: %w", profileName, err)
	}
	return nil
}

// DeleteKeychainPassword removes the keychain entry for the given profile name.
// Returns nil if the entry does not exist.
func DeleteKeychainPassword(profileName string) error {
	err := keyring.Delete(KeyringService, profileName)
	if err != nil && err != keyring.ErrNotFound {
		return fmt.Errorf("keyring delete for profile %q: %w", profileName, err)
	}
	return nil
}
