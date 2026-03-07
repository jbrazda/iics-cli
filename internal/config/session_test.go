package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSessionCache(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "sessions.yaml")

	cache := &SessionCache{Sessions: make(map[string]*SessionEntry)}

	// Set a session
	cache.Set("dev", &SessionEntry{
		SessionID:  "session-abc",
		BaseAPIURL: "https://example.com",
		OrgID:      "org123",
		OrgName:    "Test Org",
		UserName:   "testuser",
	})

	// Save to disk
	if err := cache.Save(path); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("session file not created: %v", err)
	}

	// Load from disk
	loaded, err := LoadSessionCache(path)
	if err != nil {
		t.Fatalf("LoadSessionCache() error: %v", err)
	}

	entry, ok := loaded.Get("dev")
	if !ok {
		t.Fatal("expected session for 'dev' profile")
	}
	if entry.SessionID != "session-abc" {
		t.Errorf("expected session ID 'session-abc', got %s", entry.SessionID)
	}
	if entry.OrgName != "Test Org" {
		t.Errorf("expected org name 'Test Org', got %s", entry.OrgName)
	}
}

func TestSessionCacheExpiry(t *testing.T) {
	cache := &SessionCache{Sessions: make(map[string]*SessionEntry)}

	// Set an expired session (created 31 minutes ago)
	cache.Sessions["expired"] = &SessionEntry{
		SessionID: "old-session",
		CreatedAt: time.Now().Add(-31 * time.Minute),
	}

	_, ok := cache.Get("expired")
	if ok {
		t.Error("expected expired session to not be returned")
	}

	// Verify it was cleaned up
	if _, exists := cache.Sessions["expired"]; exists {
		t.Error("expected expired session to be deleted from map")
	}
}

func TestSessionCacheDelete(t *testing.T) {
	cache := &SessionCache{Sessions: make(map[string]*SessionEntry)}
	cache.Set("dev", &SessionEntry{SessionID: "s1"})
	cache.Set("prod", &SessionEntry{SessionID: "s2"})

	cache.Delete("dev")

	_, ok := cache.Get("dev")
	if ok {
		t.Error("expected 'dev' session to be deleted")
	}

	entry, ok := cache.Get("prod")
	if !ok {
		t.Error("expected 'prod' session to still exist")
	}
	if entry.SessionID != "s2" {
		t.Errorf("expected session ID 's2', got %s", entry.SessionID)
	}
}

func TestLoadSessionCacheNonExistent(t *testing.T) {
	cache, err := LoadSessionCache("/nonexistent/path/sessions.yaml")
	if err != nil {
		t.Fatalf("expected no error for non-existent file, got: %v", err)
	}
	if cache == nil {
		t.Fatal("expected non-nil cache")
	}
	if len(cache.Sessions) != 0 {
		t.Errorf("expected empty sessions, got %d", len(cache.Sessions))
	}
}

func TestSessionEntryIsExpired(t *testing.T) {
	fresh := &SessionEntry{CreatedAt: time.Now()}
	if fresh.IsExpired() {
		t.Error("fresh session should not be expired")
	}

	old := &SessionEntry{CreatedAt: time.Now().Add(-31 * time.Minute)}
	if !old.IsExpired() {
		t.Error("31-minute-old session should be expired")
	}
}
