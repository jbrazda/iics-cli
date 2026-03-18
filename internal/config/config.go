package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config represents the full YAML configuration file.
type Config struct {
	DefaultProfile string              `yaml:"defaultProfile" mapstructure:"defaultProfile"`
	Profiles       map[string]*Profile `yaml:"profiles" mapstructure:"profiles"`
}

// Profile represents a single IICS org connection profile.
type Profile struct {
	Name     string `yaml:"name" mapstructure:"name"`
	Region   string `yaml:"region" mapstructure:"region"`
	Username string `yaml:"username" mapstructure:"username"`
	Password string `yaml:"password" mapstructure:"password"`
	LoginURL string `yaml:"loginUrl,omitempty" mapstructure:"loginUrl"`
	CaiURL   string `yaml:"caiUrl,omitempty" mapstructure:"caiUrl"`
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

	profile, ok := c.Profiles[profileName]
	if !ok {
		// If no profiles exist, build one from env vars
		profile = &Profile{}
	}

	// Environment variable overrides
	if v := os.Getenv("IICS_USERNAME"); v != "" {
		profile.Username = v
	}
	if v := os.Getenv("IICS_PASSWORD"); v != "" {
		profile.Password = v
	}
	if v := os.Getenv("IICS_REGION"); v != "" {
		profile.Region = v
	}
	if v := os.Getenv("IICS_LOGIN_URL"); v != "" {
		profile.LoginURL = v
	}
	if v := os.Getenv("IICS_CAI_URL"); v != "" {
		profile.CaiURL = v
	}

	if profile.Username == "" {
		return nil, fmt.Errorf("username not configured for profile %q; set in config file or IICS_USERNAME env var", profileName)
	}
	if profile.Password == "" {
		return nil, fmt.Errorf("password not configured for profile %q; set in config file or IICS_PASSWORD env var", profileName)
	}

	return profile, nil
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

	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	return v.WriteConfig()
}
