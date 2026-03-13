package config

import (
	"os"
	"strings"
	"testing"
)

func TestIsTerminal(t *testing.T) {
	// In test environments stdin is a pipe, not a terminal.
	if IsTerminal() {
		t.Skip("skipping: stdin is a real terminal (run in CI for proper coverage)")
	}
}

func TestPromptProfile_NonTerminalReturnsError(t *testing.T) {
	// Ensure stdin is not a terminal (it won't be in tests).
	if IsTerminal() {
		t.Skip("test requires non-terminal stdin")
	}
	_, _, err := PromptProfile(nil, "test")
	if err == nil {
		t.Fatal("expected error when stdin is not a terminal")
	}
	if !strings.Contains(err.Error(), "stdin is not a terminal") {
		t.Errorf("unexpected error: %v", err)
	}
}

// withFakeStdin replaces os.Stdin with a pipe fed by input, calls f, then restores.
func withFakeStdin(t *testing.T, input string, f func()) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	old := os.Stdin
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = old
		r.Close()
	})
	if _, err := w.WriteString(input); err != nil {
		t.Fatalf("writing to pipe: %v", err)
	}
	w.Close()
	f()
}

// stubReadPassword replaces the readPassword var so tests don't need a real TTY.
func stubReadPassword(t *testing.T, password string) {
	t.Helper()
	orig := readPassword
	readPassword = func(_ int) ([]byte, error) { return []byte(password), nil }
	t.Cleanup(func() { readPassword = orig })
}

// stubIsTerminal overrides IsTerminal by temporarily marking stdin as a pipe that
// reports true. We achieve this by monkey-patching readPassword AND feeding stdin via
// withFakeStdin. But IsTerminal() checks os.Stdin.Fd() which is a pipe — not a real
// terminal. We need to make the prompt proceed past the IsTerminal guard.
// Strategy: directly call the internal prompting after skipping the terminal check by
// temporarily replacing os.Stdin with a real TTY — but that's not portable. Instead,
// we test the individual path by checking that PromptProfile errors correctly, and we
// test the prompting logic via a helper that bypasses the terminal check.
//
// For portability the tests below work around the guard by calling promptProfile
// (the unexported implementation) directly. We expose it via a test-only shim.
func TestPromptProfile_KeepsExistingOnEmptyInput(t *testing.T) {
	existing := &Profile{
		Username: "user@example.com",
		Password: "secret",
		Region:   "USW3",
	}
	stubReadPassword(t, "") // empty — should keep existing password
	withFakeStdin(t, "\n\n\n", func() {
		p, makeDefault, err := promptProfileInternal(existing, "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.Username != "user@example.com" {
			t.Errorf("username = %q; want %q", p.Username, "user@example.com")
		}
		if p.Password != "secret" {
			t.Errorf("password should have been kept")
		}
		if p.Region != "USW3" {
			t.Errorf("region = %q; want USW3", p.Region)
		}
		// Default answer is "y" (empty line → yes)
		if !makeDefault {
			t.Errorf("expected makeDefault=true for empty input")
		}
	})
}

func TestPromptProfile_URLTreatedAsLoginURL(t *testing.T) {
	existing := &Profile{
		Username: "user@example.com",
		Password: "secret",
	}
	customURL := "https://custom.example.com/saas/public/core/v3/login"
	stubReadPassword(t, "newpass")
	// existing already has username+password set, so those loops are skipped;
	// stdin only needs: region line, default line.
	withFakeStdin(t, customURL+"\n\n", func() {
		p, _, err := promptProfileInternal(existing, "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.LoginURL != customURL {
			t.Errorf("LoginURL = %q; want %q", p.LoginURL, customURL)
		}
		if p.Region != "" {
			t.Errorf("Region should be empty when URL is provided, got %q", p.Region)
		}
	})
}

func TestPromptProfile_RegionUppercased(t *testing.T) {
	existing := &Profile{Username: "u@example.com", Password: "pass"}
	stubReadPassword(t, "pass")
	// existing already has username+password set; stdin only needs: region, default.
	withFakeStdin(t, "usw3\n\n", func() {
		p, _, err := promptProfileInternal(existing, "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.Region != "USW3" {
			t.Errorf("Region = %q; want USW3", p.Region)
		}
	})
}
