package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/config"
	"golang.org/x/term"
)

// isInteractiveTTY reports whether both stdin and stdout are terminals, so a
// prompt neither blocks a script nor ends up in piped output.
func isInteractiveTTY() bool {
	return config.IsTerminal() && term.IsTerminal(int(os.Stdout.Fd()))
}

// promptShowPrivileges asks whether to display the role's privileges. Only
// "y" or "yes" answers yes; anything else (including q or Enter) quits.
func promptShowPrivileges() bool {
	_, _ = fmt.Fprint(os.Stderr, "\nDisplay role privileges? [y/Q]: ")
	// A read error (e.g. Ctrl-D) leaves line empty, which is treated as quit.
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	line = strings.ToLower(strings.TrimSpace(line))
	return line == "y" || line == "yes"
}

// pickRole lists the roles and lets the operator choose one. The roles API
// documents no paging parameters, so a single unpaged request returns them all.
func pickRole(ctx context.Context, c *client.Client, action string) (*client.Role, error) {
	if !config.IsTerminal() {
		return nil, fmt.Errorf("provide --id or --name (stdin is not a terminal)")
	}
	roles, err := c.ListRoles(ctx, client.RoleListOptions{})
	if err != nil {
		return nil, err
	}
	if len(roles) == 0 {
		return nil, fmt.Errorf("no roles found")
	}
	sort.SliceStable(roles, func(i, j int) bool {
		return strings.ToLower(roles[i].RoleName) < strings.ToLower(roles[j].RoleName)
	})
	labels := make([]string, len(roles))
	for i, r := range roles {
		kind := "custom"
		if r.SystemRole {
			kind = "system"
		}
		labels[i] = fmt.Sprintf("%s (%s)", r.RoleName, kind)
	}
	idx, err := promptSelect(fmt.Sprintf("Select a role to %s", action), labels)
	if err != nil {
		return nil, err
	}
	if idx < 0 {
		return nil, fmt.Errorf("canceled")
	}
	return &roles[idx], nil
}

// sortRolePrivileges orders privileges by service, then privilege name.
func sortRolePrivileges(privs []client.RolePrivilege) {
	sort.SliceStable(privs, func(i, j int) bool {
		si, sj := strings.ToLower(privs[i].Service), strings.ToLower(privs[j].Service)
		if si != sj {
			return si < sj
		}
		return strings.ToLower(privs[i].Name) < strings.ToLower(privs[j].Name)
	})
}

// privilegeStatusFunc colors a privilege status: green Enabled, orange
// Unassigned, red Disabled, blue Default.
func privilegeStatusFunc(v interface{}) string {
	row, _ := v.(map[string]interface{})
	status, _ := row["status"].(string)
	if noColor {
		return status
	}
	var color string
	switch status {
	case "Enabled":
		color = "2"
	case "Unassigned":
		color = "208"
	case "Disabled":
		color = "1"
	case "Default":
		color = "4"
	default:
		return status
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(status)
}
