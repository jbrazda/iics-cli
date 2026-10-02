// Package menu implements the interactive main menu shown when "iics" runs
// without arguments in a terminal. Each entry maps to a real command line,
// which the menu runs as a subprocess and then returns to the menu.
package menu

import "strings"

// Special entry actions handled by the menu itself instead of a command.
const (
	ActionSwitchProfile = "switch-profile"
	ActionSetDefault    = "set-default"
)

// Entry is one menu item.
type Entry struct {
	Group       string
	Label       string
	Description string
	// Args are the command arguments, e.g. ["role", "edit"].
	Args []string
	// Writes marks entries that change data (confirmation on production).
	Writes bool
	// Action is set for entries handled by the menu (no command).
	Action string
	// NoProfileFlag omits "--profile <active>" (profile management commands).
	NoProfileFlag bool
	// ActiveProfileArg appends the active profile name as an argument.
	ActiveProfileArg bool
	// AskArg prompts for a text argument appended to Args (label shown).
	AskArg string
	// SessionOnly entries stay usable when no profile is configured.
	SessionOnly bool
}

// CommandArgs returns the arguments to run for the entry with the active
// profile, and the text argument from AskArg when given.
func (e Entry) CommandArgs(profile, asked string) []string {
	args := append([]string(nil), e.Args...)
	if e.ActiveProfileArg && profile != "" {
		args = append(args, profile)
	}
	if e.AskArg != "" && asked != "" {
		args = append(args, asked)
	}
	if !e.NoProfileFlag && profile != "" {
		args = append(args, "--profile", profile)
	}
	return args
}

// CommandLine formats a command line for display, quoting arguments that
// contain spaces.
func CommandLine(binary string, args []string) string {
	parts := []string{binary}
	for _, a := range args {
		if a == "" || strings.ContainsAny(a, " \t\"'") {
			a = "'" + strings.ReplaceAll(a, "'", `'\''`) + "'"
		}
		parts = append(parts, a)
	}
	return strings.Join(parts, " ")
}

// DefaultEntries is the curated menu.
func DefaultEntries() []Entry {
	return []Entry{
		{Group: "Session", Label: "Switch profile", Description: "Use another org profile for this menu session (key p)", Action: ActionSwitchProfile, SessionOnly: true},
		{Group: "Session", Label: "Set as default profile", Description: "Make the active profile the default in ~/.iics/config.yaml", Action: ActionSetDefault, Writes: true},
		{Group: "Session", Label: "Login", Description: "Sign in and refresh the session for the active profile", Args: []string{"login"}},
		{Group: "Session", Label: "List profiles", Description: "Show configured profiles", Args: []string{"profile", "list"}, NoProfileFlag: true, SessionOnly: true},
		{Group: "Session", Label: "Add profile", Description: "Set up a new org profile", Args: []string{"profile", "add"}, AskArg: "Profile name", NoProfileFlag: true, Writes: true, SessionOnly: true},
		{Group: "Session", Label: "Edit active profile", Description: "Update credentials, region, production flag and new user defaults of the active profile", Args: []string{"profile", "edit"}, ActiveProfileArg: true, NoProfileFlag: true, Writes: true},
		{Group: "Session", Label: "Edit global settings", Description: "New user defaults for all profiles, table theme, HTTP timeout, main menu", Args: []string{"config", "edit"}, NoProfileFlag: true, SessionOnly: true},

		{Group: "Users", Label: "List users", Description: "Show all users", Args: []string{"user", "list"}},
		{Group: "Users", Label: "Show user", Description: "Find a user by user name, ID, or part of a name or email and show the details", Args: []string{"user", "get"}},
		{Group: "Users", Label: "Create user", Description: "Create a user with the paged wizard", Args: []string{"user", "create", "--interactive"}, Writes: true},
		{Group: "Users", Label: "Edit user groups and roles", Description: "Change a user's group and role assignments", Args: []string{"user", "edit"}, Writes: true},
		{Group: "Users", Label: "Delete user", Description: "Search for a user and delete it", Args: []string{"user", "delete"}, Writes: true},
		{Group: "Users", Label: "Change password", Description: "Change your own or another user's password", Args: []string{"user", "change-password"}, Writes: true},

		{Group: "User groups", Label: "List user groups", Description: "Show all user groups", Args: []string{"group", "list"}},
		{Group: "User groups", Label: "Show user group", Description: "Pick a user group and show its roles and members", Args: []string{"group", "get"}},
		{Group: "User groups", Label: "Create user group", Description: "Create a user group with roles", Args: []string{"group", "create", "-i"}, Writes: true},
		{Group: "User groups", Label: "Edit user group roles", Description: "Change the roles of a user group", Args: []string{"group", "update", "-i"}, Writes: true},
		{Group: "User groups", Label: "Delete user group", Description: "Pick a user group and delete it", Args: []string{"group", "delete"}, Writes: true},

		{Group: "Roles", Label: "List roles", Description: "Show all roles", Args: []string{"role", "list"}},
		{Group: "Roles", Label: "Show role", Description: "Pick a role and show its details and privileges", Args: []string{"role", "get"}},
		{Group: "Roles", Label: "Create role", Description: "Create a custom role, optionally copying another role", Args: []string{"role", "create"}, Writes: true},
		{Group: "Roles", Label: "Edit role privileges", Description: "Edit a custom role's privileges in the grid editor", Args: []string{"role", "edit"}, Writes: true},

		{Group: "Privileges", Label: "List privileges", Description: "Show all privileges", Args: []string{"privilege", "list"}},

		{Group: "Permissions", Label: "Edit object permissions", Description: "Pick a project, folder or asset and edit its ACLs", Args: []string{"permission", "edit"}, Writes: true},

		{Group: "Environments and agents", Label: "List runtime environments", Description: "Show runtime environments (Secure Agent groups)", Args: []string{"environment", "list"}},
		{Group: "Environments and agents", Label: "Create runtime environment", Description: "Create a runtime environment and add agents", Args: []string{"environment", "create", "-i"}, Writes: true},
		{Group: "Environments and agents", Label: "List agents", Description: "Show Secure Agents", Args: []string{"agent", "list"}},
		{Group: "Environments and agents", Label: "Start agent service", Description: "Pick an agent and start a service", Args: []string{"agent", "start", "-i"}, Writes: true},
		{Group: "Environments and agents", Label: "Stop agent service", Description: "Pick an agent and stop a service", Args: []string{"agent", "stop", "-i"}, Writes: true},
		{Group: "Environments and agents", Label: "Restart agent service", Description: "Pick an agent and restart a service", Args: []string{"agent", "restart", "-i"}, Writes: true},
		{Group: "Environments and agents", Label: "Show agent installer info", Description: "Show the installer download URL, checksum URL and install token", Args: []string{"agent", "installer-info"}},
		{Group: "Environments and agents", Label: "Download agent installer", Description: "Download the Secure Agent installer; choose the folder, reuse a matching file", Args: []string{"agent", "installer-download", "--interactive", "--progress"}},

		{Group: "Help", Label: "Show CLI help", Description: "All commands and global flags", Args: []string{"--help"}, NoProfileFlag: true, SessionOnly: true},
	}
}
