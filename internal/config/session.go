package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const sessionTimeout = 30 * time.Minute

// SessionEntry holds a cached session for a profile.
type SessionEntry struct {
	SessionID     string    `yaml:"sessionId"`
	BaseAPIURL    string    `yaml:"baseApiUrl"`
	CAIUrl        string    `yaml:"caiUrl,omitempty"`
	LoginURL      string    `yaml:"loginUrl,omitempty"`
	OrgID         string    `yaml:"orgId"`
	OrgName       string    `yaml:"orgName"`
	UserName      string    `yaml:"userName"`
	CreatedAt     time.Time `yaml:"createdAt"`
	LastLoginTime time.Time `yaml:"lastLoginTime,omitempty"`
}

// SessionCache stores active session data on disk.
type SessionCache struct {
	Sessions map[string]*SessionEntry `yaml:"sessions"`
}

// DefaultSessionPath returns the default session cache file path.
func DefaultSessionPath() string {
	return filepath.Join(DefaultConfigDir(), "sessions.yaml")
}

// LoadSessionCache reads the session cache from disk.
func LoadSessionCache(path string) (*SessionCache, error) {
	if path == "" {
		path = DefaultSessionPath()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &SessionCache{Sessions: make(map[string]*SessionEntry)}, nil
		}
		return nil, fmt.Errorf("reading session cache: %w", err)
	}

	var cache SessionCache
	if err := yaml.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("parsing session cache: %w", err)
	}

	if cache.Sessions == nil {
		cache.Sessions = make(map[string]*SessionEntry)
	}

	return &cache, nil
}

// Save writes the session cache to disk.
func (sc *SessionCache) Save(path string) error {
	if path == "" {
		path = DefaultSessionPath()
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating session cache directory: %w", err)
	}

	data, err := yaml.Marshal(sc)
	if err != nil {
		return fmt.Errorf("marshaling session cache: %w", err)
	}

	return os.WriteFile(path, data, 0600)
}

// Get returns the session entry for the given profile if it exists and is not expired.
func (sc *SessionCache) Get(profileName string) (*SessionEntry, bool) {
	entry, ok := sc.Sessions[profileName]
	if !ok {
		return nil, false
	}

	if time.Since(entry.CreatedAt) >= sessionTimeout {
		delete(sc.Sessions, profileName)
		return nil, false
	}

	return entry, true
}

// Set stores a session entry for the given profile.
func (sc *SessionCache) Set(profileName string, entry *SessionEntry) {
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}
	sc.Sessions[profileName] = entry
}

// Delete removes the session entry for the given profile.
func (sc *SessionCache) Delete(profileName string) {
	delete(sc.Sessions, profileName)
}

// IsExpired returns true if the session has expired.
func (e *SessionEntry) IsExpired() bool {
	return time.Since(e.CreatedAt) >= sessionTimeout
}
