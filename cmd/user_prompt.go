package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/config"
	"golang.org/x/term"
)

// promptText prompts for a line of text. Returns defaultVal if the user presses Enter.
func promptText(label, defaultVal string) (string, error) {
	if defaultVal != "" {
		_, _ = fmt.Fprintf(os.Stderr, "%s [%s]: ", label, defaultVal)
	} else {
		_, _ = fmt.Fprintf(os.Stderr, "%s: ", label)
	}
	r := bufio.NewReader(os.Stdin)
	line, err := r.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("reading input: %w", err)
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultVal, nil
	}
	return line, nil
}

// promptPassword reads a password without echoing it to the terminal.
func promptPassword(label string) (string, error) {
	_, _ = fmt.Fprint(os.Stderr, label)
	pw, err := term.ReadPassword(int(os.Stdin.Fd()))
	_, _ = fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", fmt.Errorf("reading password: %w", err)
	}
	return string(pw), nil
}

// promptPasswordConfirm reads a new password and a confirmation. Repeats until they
// match or the user cancels by entering an empty password.
func promptPasswordConfirm(label string) (string, error) {
	for {
		pw, err := promptPassword(label + ": ")
		if err != nil {
			return "", err
		}
		if pw == "" {
			return "", nil
		}
		pw2, err := promptPassword("Confirm " + label + ": ")
		if err != nil {
			return "", err
		}
		if pw == pw2 {
			return pw, nil
		}
		_, _ = fmt.Fprintln(os.Stderr, "Passwords do not match. Please try again.")
	}
}

// promptSelect presents a numbered menu and returns the 0-based index of the chosen
// option, or -1 if the user selects 0 (exit/cancel).
func promptSelect(label string, options []string) (int, error) {
	_, _ = fmt.Fprintf(os.Stderr, "%s:\n", label)
	for i, opt := range options {
		_, _ = fmt.Fprintf(os.Stderr, "  [%d] %s\n", i+1, opt)
	}
	_, _ = fmt.Fprintln(os.Stderr, "  [0] Cancel")

	r := bufio.NewReader(os.Stdin)
	for {
		_, _ = fmt.Fprint(os.Stderr, "Selection: ")
		line, err := r.ReadString('\n')
		if err != nil {
			return -1, fmt.Errorf("reading selection: %w", err)
		}
		line = strings.TrimSpace(line)
		if line == "0" || line == "" {
			return -1, nil
		}
		n, convErr := strconv.Atoi(line)
		if convErr != nil || n < 1 || n > len(options) {
			_, _ = fmt.Fprintf(os.Stderr, "Enter a number between 0 and %d.\n", len(options))
			continue
		}
		return n - 1, nil
	}
}

// promptMultiSelect presents a numbered checklist where items marked with * are the
// current defaults. Returns the 0-based indices of selected items, or nil to skip.
func promptMultiSelect(label string, options []string, defaults []int) ([]int, error) {
	defaultSet := make(map[int]bool, len(defaults))
	for _, d := range defaults {
		defaultSet[d] = true
	}

	_, _ = fmt.Fprintf(os.Stderr, "%s (comma-separated numbers, 0 = none):\n", label)
	for i, opt := range options {
		mark := " "
		if defaultSet[i] {
			mark = "*"
		}
		_, _ = fmt.Fprintf(os.Stderr, "  [%d]%s %s\n", i+1, mark, opt)
	}

	if len(defaults) > 0 {
		defaultNums := make([]string, len(defaults))
		for i, d := range defaults {
			defaultNums[i] = strconv.Itoa(d + 1)
		}
		_, _ = fmt.Fprintf(os.Stderr, "Selections [%s]: ", strings.Join(defaultNums, ","))
	} else {
		_, _ = fmt.Fprint(os.Stderr, "Selections: ")
	}

	r := bufio.NewReader(os.Stdin)
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("reading selections: %w", err)
	}
	line = strings.TrimSpace(line)

	if line == "" && len(defaults) > 0 {
		return defaults, nil
	}
	if line == "0" || line == "" {
		return nil, nil
	}

	parts := strings.Split(line, ",")
	selected := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		n, convErr := strconv.Atoi(p)
		if convErr != nil || n < 0 || n > len(options) {
			return nil, fmt.Errorf("invalid selection %q; enter numbers between 0 and %d", p, len(options))
		}
		if n == 0 {
			return nil, nil
		}
		selected = append(selected, n-1)
	}
	return selected, nil
}

// promptYesNo prompts for a yes/no answer. defaultYes controls what pressing Enter
// returns. Returns true for yes, false for no.
func promptYesNo(label string, defaultYes bool) (bool, error) {
	hint := "[Y/n]"
	if !defaultYes {
		hint = "[y/N]"
	}
	_, _ = fmt.Fprintf(os.Stderr, "%s %s: ", label, hint)
	r := bufio.NewReader(os.Stdin)
	line, err := r.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("reading input: %w", err)
	}
	line = strings.TrimSpace(strings.ToLower(line))
	if line == "" {
		return defaultYes, nil
	}
	return line == "y" || line == "yes", nil
}

// promptUserSearch interactively finds a user by searching by name or exact ID.
// Returns nil if the user cancels. Returns an error if stdin is not a terminal.
func promptUserSearch(ctx context.Context, c *client.Client) (*client.User, error) {
	if !config.IsTerminal() {
		return nil, fmt.Errorf("stdin is not a terminal; provide --id or --username flag")
	}

	r := bufio.NewReader(os.Stdin)
	for {
		_, _ = fmt.Fprintln(os.Stderr, "\nSearch user by:")
		_, _ = fmt.Fprintln(os.Stderr, "  [1] User Name (partial match)")
		_, _ = fmt.Fprintln(os.Stderr, "  [2] ID (exact match)")
		_, _ = fmt.Fprintln(os.Stderr, "  [0] Exit")
		_, _ = fmt.Fprint(os.Stderr, "Selection: ")

		line, err := r.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("reading selection: %w", err)
		}
		line = strings.TrimSpace(line)

		switch line {
		case "0", "":
			return nil, nil

		case "1":
			_, _ = fmt.Fprint(os.Stderr, "User Name (partial match): ")
			query, qErr := r.ReadString('\n')
			if qErr != nil {
				return nil, fmt.Errorf("reading user name: %w", qErr)
			}
			query = strings.TrimSpace(query)
			if query == "" {
				continue
			}
			users, sErr := c.SearchUsers(ctx, query)
			if sErr != nil {
				return nil, sErr
			}
			if len(users) == 0 {
				_, _ = fmt.Fprintf(os.Stderr, "No users found matching %q.\n", query)
				continue
			}
			if len(users) == 1 {
				return &users[0], nil
			}
			opts := make([]string, len(users))
			for i, u := range users {
				opts[i] = fmt.Sprintf("%s (%s)", u.UserName, u.ID)
			}
			idx, sErr := promptSelect("Select user", opts)
			if sErr != nil {
				return nil, sErr
			}
			if idx < 0 {
				continue
			}
			return &users[idx], nil

		case "2":
			_, _ = fmt.Fprint(os.Stderr, "User ID (exact match): ")
			id, iErr := r.ReadString('\n')
			if iErr != nil {
				return nil, fmt.Errorf("reading user ID: %w", iErr)
			}
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			u, gErr := c.GetUser(ctx, id)
			if gErr != nil {
				_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", gErr)
				continue
			}
			return u, nil

		default:
			_, _ = fmt.Fprintln(os.Stderr, "Enter 0, 1, or 2.")
		}
	}
}

// resolveUser returns the user identified by id or userName flags, falling back to
// interactive search when both are empty and stdin is a terminal.
func resolveUser(ctx context.Context, c *client.Client, id, userName string) (*client.User, error) {
	switch {
	case id != "":
		return c.GetUser(ctx, id)
	case userName != "":
		return c.GetUserByName(ctx, userName)
	default:
		return promptUserSearch(ctx, c)
	}
}
