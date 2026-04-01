package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// StyleConfig holds user preferences for table output styling.
type StyleConfig struct {
	Theme   string `yaml:"theme,omitempty"   mapstructure:"theme"`
	NoColor bool   `yaml:"noColor,omitempty" mapstructure:"noColor"`
}

// Config represents the full YAML configuration file.
type Config struct {
	DefaultProfile string              `yaml:"defaultProfile" mapstructure:"defaultProfile"`
	Profiles       map[string]*Profile `yaml:"profiles"       mapstructure:"profiles"`
	Style          StyleConfig         `yaml:"style,omitempty" mapstructure:"style"`
}

// Profile represents a single IICS org connection profile.
type Profile struct {
	Name       string `yaml:"name" mapstructure:"name"`
	Region     string `yaml:"region,omitempty" mapstructure:"region"`
	Username   string `yaml:"username" mapstructure:"username"`
	Password   string `yaml:"password" mapstructure:"password"`
	LoginURL   string `yaml:"loginUrl,omitempty" mapstructure:"loginUrl"`
	BaseAPIURL string `yaml:"baseApiUrl,omitempty" mapstructure:"baseApiUrl"`
	CaiURL     string `yaml:"caiUrl,omitempty" mapstructure:"caiUrl"`
}

// DeriveCaiURL derives the CAI base URL from an IICS product baseApiUrl.
// The CAI hostname is formed by inserting "-cai" after the first DNS label:
//
//	https://use4.dm-us.informaticacloud.com/saas -> https://use4-cai.dm-us.informaticacloud.com
//
// Returns "" if baseAPIURL cannot be parsed or the host has no dot separator.
func DeriveCaiURL(baseAPIURL string) string {
	u, err := url.Parse(baseAPIURL)
	if err != nil || u.Host == "" {
		return ""
	}
	dot := strings.Index(u.Host, ".")
	if dot < 0 {
		return ""
	}
	return fmt.Sprintf("%s://%s-cai%s", u.Scheme, u.Host[:dot], u.Host[dot:])
}

// DefaultConfigDir returns the default config directory path.
func DefaultConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".iics"
	}
	return filepath.Join(home, ".iics")
}

// DefaultConfigPath returns the default config file path.
func DefaultConfigPath() string {
	return filepath.Join(DefaultConfigDir(), "config.yaml")
}

// Load reads the configuration from the given file path.
// If configPath is empty, it uses the default path.
func Load(configPath string) (*Config, error) {
	if configPath == "" {
		configPath = DefaultConfigPath()
	}

	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// Environment variable overrides
	v.SetEnvPrefix("IICS")
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return &Config{
				Profiles: make(map[string]*Profile),
			}, nil
		}
		if os.IsNotExist(err) {
			return &Config{
				Profiles: make(map[string]*Profile),
			}, nil
		}
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]*Profile)
	}

	return &cfg, nil
}

// ResolveProfile returns the profile for the given name.
// If profileName is empty, it uses the default profile.
// Environment variables override profile values.
func (c *Config) ResolveProfile(profileName string) (*Profile, error) {
	if profileName == "" {
		profileName = os.Getenv("IICS_PROFILE")
	}
	if profileName == "" {
		profileName = c.DefaultProfile
	}
	if profileName == "" {
		profileName = "default"
	}

	// Copy the stored profile so env-var overrides never mutate cfg.Profiles.
	var resolved Profile
	if existing, ok := c.Profiles[profileName]; ok {
		resolved = *existing
	}

	// Environment variable overrides (applied to copy only)
	if v := os.Getenv("IICS_USERNAME"); v != "" {
		resolved.Username = v
	}
	if v := os.Getenv("IICS_PASSWORD"); v != "" {
		resolved.Password = v
	}
	if v := os.Getenv("IICS_REGION"); v != "" {
		resolved.Region = v
	}
	if v := os.Getenv("IICS_LOGIN_URL"); v != "" {
		resolved.LoginURL = v
	}
	if v := os.Getenv("IICS_CAI_URL"); v != "" {
		resolved.CaiURL = v
	}

	if resolved.Username == "" {
		return nil, fmt.Errorf("username not configured for profile %q; set in config file or IICS_USERNAME env var", profileName)
	}
	if resolved.Password == "" {
		return nil, fmt.Errorf("password not configured for profile %q; set in config file or IICS_PASSWORD env var", profileName)
	}

	return &resolved, nil
}

// GetLoginURL returns the login URL for the profile.
// If LoginURL is set on the profile, it is used directly.
// Otherwise, it is resolved from the Region via the POD registry.
func (p *Profile) GetLoginURL() (string, error) {
	if p.LoginURL != "" {
		return p.LoginURL, nil
	}
	if p.Region == "" {
		return "", fmt.Errorf("region not configured; set region in config file or IICS_REGION env var, or provide loginUrl directly")
	}
	return LoginURL(p.Region)
}

// ResolveTargetProfile returns the profile for the given name, applying IICS_TARGET_* env vars.
// Unlike ResolveProfile, it does not fall back to IICS_PROFILE or defaultProfile;
// profileName is required.
func (c *Config) ResolveTargetProfile(profileName string) (*Profile, error) {
	var resolved Profile
	if existing, ok := c.Profiles[profileName]; ok {
		resolved = *existing
	}

	if v := os.Getenv("IICS_TARGET_USERNAME"); v != "" {
		resolved.Username = v
	}
	if v := os.Getenv("IICS_TARGET_PASSWORD"); v != "" {
		resolved.Password = v
	}
	if v := os.Getenv("IICS_TARGET_REGION"); v != "" {
		resolved.Region = v
	}
	if v := os.Getenv("IICS_TARGET_LOGIN_URL"); v != "" {
		resolved.LoginURL = v
	}

	if resolved.Username == "" {
		return nil, fmt.Errorf("username not configured for target profile %q; set in config file or IICS_TARGET_USERNAME env var", profileName)
	}
	if resolved.Password == "" {
		return nil, fmt.Errorf("password not configured for target profile %q; set in config file or IICS_TARGET_PASSWORD env var", profileName)
	}

	return &resolved, nil
}

// Save writes the configuration to disk.
func (c *Config) Save(configPath string) error {
	if configPath == "" {
		configPath = DefaultConfigPath()
	}

	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	v := viper.New()
	v.Set("defaultProfile", c.DefaultProfile)
	v.Set("profiles", c.Profiles)
	if c.Style.Theme != "" || c.Style.NoColor {
		v.Set("style", c.Style)
	}

	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	return v.WriteConfig()
}
