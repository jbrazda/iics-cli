package cmd

import (
	"context"
	"fmt"
	"os"

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

// promptMultiSelect presents a checklist with defaults pre-selected. Returns the
// 0-based indices of selected items, or nil for none.
func promptMultiSelect(label string, options []string, defaults []int) ([]int, error) {
	return prompter.MultiSelect(label, options, defaults)
}

// promptYesNo prompts for a yes/no answer. defaultYes controls what pressing Enter
// returns. Returns true for yes, false for no.
func promptYesNo(label string, defaultYes bool) (bool, error) {
	return prompter.Confirm(label, defaultYes)
}

// promptUserSearch interactively finds a user by searching by name or exact ID.
// Returns nil if the user cancels. Returns an error if stdin is not a terminal.
func promptUserSearch(ctx context.Context, c *client.Client) (*client.User, error) {
	if !config.IsTerminal() {
		return nil, fmt.Errorf("stdin is not a terminal; provide --id or --username flag")
	}

	for {
		mode, err := promptSelect("Search user by", []string{"User Name (partial match)", "ID (exact match)"})
		if err != nil {
			return nil, err
		}
		switch mode {
		case -1:
			return nil, nil

		case 0:
			query, qErr := promptText("User Name (partial match)", "")
			if qErr != nil {
				return nil, fmt.Errorf("reading user name: %w", qErr)
			}
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

		case 1:
			id, iErr := promptText("User ID (exact match)", "")
			if iErr != nil {
				return nil, fmt.Errorf("reading user ID: %w", iErr)
			}
			if id == "" {
				continue
			}
			u, gErr := c.GetUser(ctx, id)
			if gErr != nil {
				_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", gErr)
				continue
			}
			return u, nil
		}
	}
}

// iicsTimezones is the complete list of timezone IDs accepted by the IICS v3 API.
// Source: https://docs.informatica.com/cloud-common-services/administrator/current-version/rest-api-reference/rest-api-codes/time-zone-codes.html
var iicsTimezones = []string{
	"ACT", "AET", "Africa/Cairo", "Africa/Casablanca", "Africa/Johannesburg",
	"Africa/Nairobi", "America/Barbados", "America/Bogota", "America/Buenos_Aires",
	"America/Caracas", "America/Chicago", "America/Costa_Rica", "America/Dawson_Creek",
	"America/Denver", "America/Dominica", "America/El_Salvador", "America/Guadeloupe",
	"America/Halifax", "America/Havana", "America/Jamaica", "America/La_Paz",
	"America/Los_Angeles", "America/Mexico_City", "America/Montreal", "America/New_York",
	"America/Panama", "America/Phoenix", "America/Puerto_Rico", "America/Santiago",
	"America/Tijuana", "America/Vancouver", "Asia/Baghdad", "Asia/Bahrain", "Asia/Dubai",
	"Asia/Hong_Kong", "Asia/Jerusalem", "Asia/Karachi", "Asia/Katmandu",
	"Asia/Kuala_Lumpur", "Asia/Kuwait", "Asia/Magadan", "Asia/Muscat", "Asia/Qatar",
	"Asia/Rangoon", "Asia/Riyadh", "Asia/Seoul", "Asia/Singapore", "AST",
	"Atlantic/Cape_Verde", "Atlantic/South_Georgia", "Australia/Lord_Howe",
	"Australia/Perth", "Brazil/Acre", "Brazil/DeNoronha", "Brazil/East", "Brazil/West",
	"BST", "CNT", "CTT", "Europe/Amsterdam", "Europe/Athens", "Europe/Belgrade",
	"Europe/Berlin", "Europe/Brussels", "Europe/Bucharest", "Europe/Budapest",
	"Europe/Copenhagen", "Europe/Istanbul", "Europe/London", "Europe/Luxembourg",
	"Europe/Madrid", "Europe/Moscow", "Europe/Paris", "Europe/Prague", "Europe/Rome",
	"Europe/Stockholm", "Europe/Vienna", "Europe/Warsaw", "Europe/Zurich",
	"GMT", "HST", "Indian/Mauritius", "IST", "JST", "Pacific/Apia", "Pacific/Auckland",
	"Pacific/Chatham", "Pacific/Enderbury", "Pacific/Fiji", "Pacific/Gambier",
	"Pacific/Kiritimati", "Pacific/Norfolk", "Pacific/Tahiti", "UTC", "VST",
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
