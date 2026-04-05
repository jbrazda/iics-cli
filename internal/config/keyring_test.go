package config

import (
	"os"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"
)

// initMockKeyring sets up an in-memory keyring backend for tests.
// Called at the start of each test that touches the keyring so tests are isolated.
func initMockKeyring() {
	keyring.MockInit()
}

func TestIsKeyringSentinel(t *testing.T) {
	if !IsKeyringSentinel("@keyring") {
		t.Error("expected true for @keyring")
	}
	if IsKeyringSentinel("") {
		t.Error("expected false for empty string")
	}
	if IsKeyringSentinel("somepassword") {
		t.Error("expected false for regular password")
	}
	if IsKeyringSentinel("@KEYRING") {
		t.Error("sentinel is case-sensitive; expected false for @KEYRING")
	}
}

func TestSetAndGetKeychainPassword(t *testing.T) {
	initMockKeyring()

	if err := SetKeychainPassword("mypro", "s3cr3t"); err != nil {
		t.Fatalf("SetKeychainPassword() error: %v", err)
	}

	got, err := GetKeychainPassword("mypro")
	if err != nil {
		t.Fatalf("GetKeychainPassword() error: %v", err)
	}
	if got != "s3cr3t" {
		t.Errorf("got %q, want %q", got, "s3cr3t")
	}
}

func TestGetKeychainPasswordMissing(t *testing.T) {
	initMockKeyring()

	_, err := GetKeychainPassword("nonexistent-profile")
	if err == nil {
		t.Fatal("expected error for missing keychain entry")
	}
	if !strings.Contains(err.Error(), "nonexistent-profile") {
		t.Errorf("error should mention profile name, got: %v", err)
	}
}

func TestDeleteKeychainPassword(t *testing.T) {
	initMockKeyring()

	if err := SetKeychainPassword("todel", "pw"); err != nil {
		t.Fatalf("SetKeychainPassword() error: %v", err)
	}

	if err := DeleteKeychainPassword("todel"); err != nil {
		t.Fatalf("DeleteKeychainPassword() error: %v", err)
	}

	_, err := GetKeychainPassword("todel")
	if err == nil {
		t.Fatal("expected error after deletion")
	}
}

func TestDeleteKeychainPasswordNotFound(t *testing.T) {
	initMockKeyring()

	// Deleting a non-existent entry must return nil (not an error).
	if err := DeleteKeychainPassword("ghost-profile"); err != nil {
		t.Errorf("expected nil for missing entry, got: %v", err)
	}
}

func TestResolveProfileKeyring(t *testing.T) {
	initMockKeyring()

	if err := SetKeychainPassword("ktest", "realpassword"); err != nil {
		t.Fatalf("SetKeychainPassword() error: %v", err)
	}

	cfg := &Config{
		Profiles: map[string]*Profile{
			"ktest": {Username: "user@example.com", Password: KeyringSentinel},
		},
	}

	p, err := cfg.ResolveProfile("ktest")
	if err != nil {
		t.Fatalf("ResolveProfile() error: %v", err)
	}
	if p.Password != "realpassword" {
		t.Errorf("password = %q; want %q", p.Password, "realpassword")
	}
}

func TestResolveProfileKeyringMissing(t *testing.T) {
	initMockKeyring()
	// No keychain entry stored - lookup must fail with a clear message.

	cfg := &Config{
		Profiles: map[string]*Profile{
			"ktest2": {Username: "user@example.com", Password: KeyringSentinel},
		},
	}

	_, err := cfg.ResolveProfile("ktest2")
	if err == nil {
		t.Fatal("expected error for missing keychain entry")
	}
	if !strings.Contains(err.Error(), "keychain lookup failed") {
		t.Errorf("error should mention keychain lookup failure, got: %v", err)
	}
	if !strings.Contains(err.Error(), "IICS_PASSWORD") {
		t.Errorf("error should mention IICS_PASSWORD fallback, got: %v", err)
	}
}

func TestResolveProfileKeyringEnvOverride(t *testing.T) {
	initMockKeyring()
	// IICS_PASSWORD must win over the sentinel without touching the keychain.

	t.Setenv("IICS_PASSWORD", "envpassword")

	cfg := &Config{
		Profiles: map[string]*Profile{
			"ktest3": {Username: "user@example.com", Password: KeyringSentinel},
		},
	}

	p, err := cfg.ResolveProfile("ktest3")
	if err != nil {
		t.Fatalf("ResolveProfile() error: %v", err)
	}
	if p.Password != "envpassword" {
		t.Errorf("password = %q; want env override %q", p.Password, "envpassword")
	}
}

func TestResolveProfilePlaintext(t *testing.T) {
	// Plaintext password must pass through unchanged - no keyring interaction.
	cfg := &Config{
		Profiles: map[string]*Profile{
			"plain": {Username: "u@example.com", Password: "plaintextpw"},
		},
	}

	p, err := cfg.ResolveProfile("plain")
	if err != nil {
		t.Fatalf("ResolveProfile() error: %v", err)
	}
	if p.Password != "plaintextpw" {
		t.Errorf("password = %q; want %q", p.Password, "plaintextpw")
	}
}

func TestResolveProfileKeyringMutatesOnlyCopy(t *testing.T) {
	initMockKeyring()

	if err := SetKeychainPassword("immut", "realpass"); err != nil {
		t.Fatalf("SetKeychainPassword() error: %v", err)
	}

	prof := &Profile{Username: "u@example.com", Password: KeyringSentinel}
	cfg := &Config{Profiles: map[string]*Profile{"immut": prof}}

	_, _ = cfg.ResolveProfile("immut")

	// Original profile in the map must not be mutated.
	if cfg.Profiles["immut"].Password != KeyringSentinel {
		t.Errorf("ResolveProfile mutated the stored profile; sentinel should remain")
	}
}

// TestResolveProfileKeyringEnvOverrideDoesNotCallKeychain verifies that when
// IICS_PASSWORD is set, the sentinel check is bypassed entirely. We prove this
// indirectly: if the keychain were called for a missing entry it would return
// an error, but with the env var set the call must succeed.
func TestResolveProfileKeyringEnvOverrideDoesNotCallKeychain(t *testing.T) {
	initMockKeyring()
	// No keychain entry - if keychain were called, it would error.

	if err := os.Setenv("IICS_PASSWORD", "fromenv"); err != nil {
		t.Fatalf("Setenv: %v", err)
	}
	t.Cleanup(func() { _ = os.Unsetenv("IICS_PASSWORD") })

	cfg := &Config{
		Profiles: map[string]*Profile{
			"bypass": {Username: "u@example.com", Password: KeyringSentinel},
		},
	}

	p, err := cfg.ResolveProfile("bypass")
	if err != nil {
		t.Fatalf("expected no error when IICS_PASSWORD overrides sentinel, got: %v", err)
	}
	if p.Password != "fromenv" {
		t.Errorf("password = %q; want %q", p.Password, "fromenv")
	}
}
