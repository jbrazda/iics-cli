package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// readPassword is a package-level var so tests can stub it out without a real TTY.
var readPassword = func(fd int) ([]byte, error) { return term.ReadPassword(fd) }

// IsTerminal reports whether stdin is an interactive terminal.
func IsTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// PromptProfile interactively collects profile credentials from the user.
// existing may be nil when creating a new profile; its values are used as defaults.
// profileName is used only in display messages.
// Returns the filled profile, whether the user wants it as the default, and any error.
// Returns an error immediately if stdin is not a terminal.
func PromptProfile(existing *Profile, profileName string) (*Profile, bool, error) {
	if !IsTerminal() {
		return nil, false, errors.New("stdin is not a terminal; use --profile flag or IICS_* env vars")
	}
	return promptProfileInternal(existing, profileName)
}

// promptProfileInternal contains the prompting logic. Separated from PromptProfile
// so tests can exercise it via a pipe-based stdin without a real TTY.
func promptProfileInternal(existing *Profile, profileName string) (*Profile, bool, error) {
	p := &Profile{}
	if existing != nil {
		*p = *existing
	}

	r := bufio.NewReader(os.Stdin)

	_, _ = fmt.Fprintf(os.Stderr, "Setting up profile %q\n\n", profileName)

	// Username
	for p.Username == "" {
		if existing != nil && existing.Username != "" {
			_, _ = fmt.Fprintf(os.Stderr, "Username [%s]: ", existing.Username)
		} else {
			_, _ = fmt.Fprint(os.Stderr, "Username: ")
		}
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, false, fmt.Errorf("reading username: %w", err)
		}
		line = strings.TrimSpace(line)
		if line != "" {
			p.Username = line
		} else if existing != nil && existing.Username != "" {
			p.Username = existing.Username
		}
		if p.Username == "" {
			_, _ = fmt.Fprintln(os.Stderr, "Username is required.")
		}
	}

	// Password
	for p.Password == "" {
		if existing != nil && existing.Password != "" {
			_, _ = fmt.Fprint(os.Stderr, "Password [current: ***]: ")
		} else {
			_, _ = fmt.Fprint(os.Stderr, "Password: ")
		}
		pw, err := readPassword(int(os.Stdin.Fd()))
		_, _ = fmt.Fprintln(os.Stderr)
		if err != nil {
			return nil, false, fmt.Errorf("reading password: %w", err)
		}
		if len(pw) > 0 {
			p.Password = string(pw)
		} else if existing != nil && existing.Password != "" {
			p.Password = existing.Password
		}
		if p.Password == "" {
			_, _ = fmt.Fprintln(os.Stderr, "Password is required.")
		}
	}

	// Region or custom login URL
	regionCurrent := p.Region
	if regionCurrent == "" && p.LoginURL != "" {
		regionCurrent = p.LoginURL
	}
	if regionCurrent != "" {
		_, _ = fmt.Fprintf(os.Stderr, "Region or login URL [%s]: ", regionCurrent)
	} else {
		_, _ = fmt.Fprintf(os.Stderr, "Region (%s)\n  or custom login URL: ", ValidRegions())
	}
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, false, fmt.Errorf("reading region: %w", err)
	}
	line = strings.TrimSpace(line)
	if line == "" {
		// Keep existing values (already copied into p above)
	} else if strings.HasPrefix(line, "https://") || strings.HasPrefix(line, "http://") {
		p.LoginURL = line
		p.Region = ""
	} else {
		p.Region = strings.ToUpper(line)
		p.LoginURL = ""
	}

	// Derive loginUrl from region if not set explicitly.
	if p.LoginURL == "" && p.Region != "" {
		if derivedLogin, lerr := LoginURL(p.Region); lerr == nil {
			p.LoginURL = derivedLogin
		}
	}

	// CAI URL - derive from login URL and offer for override.
	derivedCaiURL := ""
	if p.LoginURL != "" {
		derivedCaiURL = DeriveCaiURL(p.LoginURL)
	}
	caiDefault := p.CaiURL
	if caiDefault == "" {
		caiDefault = derivedCaiURL
	}
	if caiDefault != "" {
		_, _ = fmt.Fprintf(os.Stderr, "CAI URL [%s]: ", caiDefault)
	} else {
		_, _ = fmt.Fprint(os.Stderr, "CAI URL (leave blank to derive automatically on login): ")
	}
	line, err = r.ReadString('\n')
	if err != nil {
		return nil, false, fmt.Errorf("reading CAI URL: %w", err)
	}
	line = strings.TrimSpace(line)
	if line != "" {
		p.CaiURL = line
	} else {
		p.CaiURL = caiDefault
	}

	// Default profile prompt (default: yes)
	_, _ = fmt.Fprint(os.Stderr, "Set as default profile? [Y/n]: ")
	line, err = r.ReadString('\n')
	if err != nil {
		return nil, false, fmt.Errorf("reading default choice: %w", err)
	}
	line = strings.TrimSpace(strings.ToLower(line))
	makeDefault := line == "" || line == "y" || line == "yes"

	return p, makeDefault, nil
}
