package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/config"
	"github.com/jbrazda/iics-cli/internal/tui"
)

// prompter renders interactive prompts: huh forms on a terminal, line-based
// prompts (the historical input format) when stdin is not a terminal.
var prompter = tui.Default()

// promptText prompts for a line of text. Returns defaultVal if the user presses Enter.
func promptText(label, defaultVal string) (string, error) {
	return prompter.Text(label, defaultVal)
}

// promptPassword reads a password without echoing it to the terminal.
func promptPassword(label string) (string, error) {
	return prompter.Secret(label)
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

// promptSelect presents a menu and returns the 0-based index of the chosen
// option, or -1 if the user cancels.
func promptSelect(label string, options []string) (int, error) {
	return prompter.Select(label, options)
}

// promptYesNo prompts for a yes/no answer. defaultYes controls what pressing Enter
// returns. Returns true for yes, false for no.
func promptYesNo(label string, defaultYes bool) (bool, error) {
	return prompter.Confirm(label, defaultYes)
}

// promptUserSearch asks for a user name, user ID, or part of a name or
// email. Exact user names and IDs are looked up on the server; other text
// is matched against user name, first/last name and email, and several
// matches are offered in a filterable list. Returns nil if the user cancels
// (empty input). Returns an error if stdin is not a terminal.
func promptUserSearch(ctx context.Context, c *client.Client) (*client.User, error) {
	if !config.IsTerminal() {
		return nil, fmt.Errorf("stdin is not a terminal; provide --id or --username flag")
	}
	for {
		text, err := promptText("User name, ID, or part of a name/email (empty to cancel)", "")
		if err != nil {
			return nil, err
		}
		if text == "" {
			return nil, nil
		}
		if u, lerr := c.GetUserByName(ctx, text); lerr == nil {
			return u, nil
		}
		if !strings.ContainsAny(text, " @") {
			if u, lerr := c.GetUser(ctx, text); lerr == nil {
				return u, nil
			}
		}
		users, err := c.SearchUsers(ctx, text)
		if err != nil {
			return nil, err
		}
		switch len(users) {
		case 0:
			_, _ = fmt.Fprintf(os.Stderr, "No users match %q.\n", text)
			continue
		case 1:
			return &users[0], nil
		}
		sort.Slice(users, func(i, j int) bool { return strings.ToLower(users[i].UserName) < strings.ToLower(users[j].UserName) })
		idx, err := tui.PickOne(prompter, fmt.Sprintf("%d users match %q", len(users), text), users, userLabel)
		if err != nil {
			return nil, err
		}
		if idx >= 0 {
			return &users[idx], nil
		}
	}
}

// userLabel describes a user in pickers.
func userLabel(u client.User) string {
	label := u.UserName
	if name := strings.TrimSpace(u.FirstName + " " + u.LastName); name != "" {
		label += "  " + name
	}
	if u.Email != "" && !strings.EqualFold(u.Email, u.UserName) {
		label += "  <" + u.Email + ">"
	}
	if u.State != "" {
		label += "  (" + u.State + ")"
	}
	return label
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
