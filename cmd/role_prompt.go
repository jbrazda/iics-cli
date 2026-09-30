package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/config"
	"github.com/jbrazda/iics-cli/internal/tui"
	"github.com/jbrazda/iics-cli/internal/tui/privmatrix"
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

// pickRole lists the roles in a filterable menu and lets the operator choose
// one. The roles API documents no paging parameters, so a single unpaged
// request returns them all.
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
	sortRoles(roles)
	idx := 0
	sel := huh.NewSelect[int]().
		Title(fmt.Sprintf("Select a role to %s (/ to filter)", action)).
		Options(roleOptions(roles)...).
		Height(min(len(roles)+2, 22)).
		Value(&idx)
	if err := huh.NewForm(huh.NewGroup(sel)).WithOutput(os.Stderr).Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return nil, fmt.Errorf("canceled")
		}
		return nil, err
	}
	return &roles[idx], nil
}

func sortRoles(roles []client.Role) {
	sort.SliceStable(roles, func(i, j int) bool {
		return strings.ToLower(roles[i].RoleName) < strings.ToLower(roles[j].RoleName)
	})
}

func roleOptions(roles []client.Role) []huh.Option[int] {
	opts := make([]huh.Option[int], len(roles))
	for i, r := range roles {
		kind := "custom"
		if r.SystemRole {
			kind = "system"
		}
		opts[i] = huh.NewOption(fmt.Sprintf("%s (%s)", r.RoleName, kind), i)
	}
	return opts
}

// editorNoColor reports whether interactive editors should render without color.
func editorNoColor() bool {
	cfg, _ := loadConfig()
	return resolveTableStyle(cfg).NoColor
}

// editRolePrivilegesInteractive opens the privilege editor for a role and
// applies the resulting changes (add first, then remove, so the role never
// drops to zero privileges in between).
func editRolePrivilegesInteractive(ctx context.Context, c *client.Client, w io.Writer, ref client.RoleRef, mode privmatrix.Mode) error {
	if !isInteractiveTTY() {
		return fmt.Errorf("interactive editing requires a terminal; use role add-privileges / remove-privileges with --privilege")
	}
	opts := client.RoleGetOptions{ID: ref.ID, Name: ref.Name, ExpandPrivileges: true}
	if opts.ID == "" && opts.Name == "" {
		picked, err := pickRole(ctx, c, "edit")
		if err != nil {
			return err
		}
		opts.ID = picked.ID
	}
	role, err := c.GetRole(ctx, opts)
	if err != nil {
		return err
	}
	if role.SystemRole {
		return fmt.Errorf("role %q is a system role; only custom roles can be changed", role.RoleName)
	}
	all, err := c.ListPrivileges(ctx)
	if err != nil {
		return err
	}
	assigned := make([]string, len(role.Privileges))
	for i, p := range role.Privileges {
		assigned[i] = p.Name
	}

	desired, err := tui.EditPrivileges(all, assigned, assigned, tui.PrivilegeEditorOptions{
		Title:   "Role: " + role.RoleName,
		Mode:    mode,
		NoColor: editorNoColor(),
	})
	if errors.Is(err, tui.ErrCanceled) {
		_, _ = fmt.Fprintln(w, "Canceled.")
		return nil
	}
	if err != nil {
		return err
	}

	add, remove := client.DiffPrivileges(assigned, desired)
	if len(add) == 0 && len(remove) == 0 {
		_, _ = fmt.Fprintln(w, "No changes.")
		return nil
	}
	target := client.RoleRef{ID: role.ID}
	if len(add) > 0 {
		if err := c.AddRolePrivileges(ctx, target, add); err != nil {
			return err
		}
		printPrivilegeChanges(w, "Added", role.RoleName, "+", add)
	}
	if len(remove) > 0 {
		if err := c.RemoveRolePrivileges(ctx, target, remove); err != nil {
			return err
		}
		printPrivilegeChanges(w, "Removed", role.RoleName, "-", remove)
	}
	return nil
}

func printPrivilegeChanges(w io.Writer, verb, roleName, sign string, names []string) {
	_, _ = fmt.Fprintf(w, "%s %d privilege(s) for role %s:\n", verb, len(names), roleName)
	for _, n := range names {
		_, _ = fmt.Fprintf(w, "  %s %s\n", sign, n)
	}
}

// runRoleCreateWizard collects the name, description and optional clone
// source for a new role, then opens the privilege editor. It fills body with
// privilege IDs. ok is false when the user canceled.
func runRoleCreateWizard(ctx context.Context, c *client.Client, body *client.CreateRoleRequest, fromRole string, extra []string) (bool, error) {
	roles, err := c.ListRoles(ctx, client.RoleListOptions{})
	if err != nil {
		return false, err
	}
	sortRoles(roles)
	existing := make(map[string]bool, len(roles))
	for _, r := range roles {
		existing[strings.ToLower(r.RoleName)] = true
	}

	cloneIdx := -1
	if fromRole != "" {
		src, ferr := c.FindRole(ctx, fromRole, false)
		if ferr != nil {
			return false, ferr
		}
		for i := range roles {
			if roles[i].ID == src.ID {
				cloneIdx = i
			}
		}
	}
	cloneOpts := append([]huh.Option[int]{huh.NewOption("None - start empty", -1)}, roleOptions(roles)...)

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Role name").Value(&body.Name).Validate(func(v string) error {
				v = strings.TrimSpace(v)
				if v == "" {
					return fmt.Errorf("name is required")
				}
				if existing[strings.ToLower(v)] {
					return fmt.Errorf("a role named %q already exists", v)
				}
				return nil
			}),
			huh.NewInput().Title("Description (optional)").Value(&body.Description),
			huh.NewSelect[int]().
				Title("Copy privileges from an existing role? (/ to filter)").
				Options(cloneOpts...).
				Height(12).
				Value(&cloneIdx),
		),
	).WithOutput(os.Stderr)
	if err = form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return false, nil
		}
		return false, err
	}
	body.Name = strings.TrimSpace(body.Name)

	var initial []string
	if cloneIdx >= 0 {
		src, gerr := c.GetRole(ctx, client.RoleGetOptions{ID: roles[cloneIdx].ID, ExpandPrivileges: true})
		if gerr != nil {
			return false, gerr
		}
		for _, p := range src.Privileges {
			initial = append(initial, p.Name)
		}
		if body.Description == "" {
			body.Description = src.Description
		}
	}
	if len(extra) > 0 {
		resolved, rerr := c.ResolvePrivileges(ctx, extra)
		if rerr != nil {
			return false, rerr
		}
		for _, p := range resolved {
			initial = append(initial, p.Name)
		}
	}

	all, err := c.ListPrivileges(ctx)
	if err != nil {
		return false, err
	}
	desired, err := tui.EditPrivileges(all, nil, initial, tui.PrivilegeEditorOptions{
		Title:      "New role: " + body.Name,
		ApplyLabel: "Create role",
		NoColor:    editorNoColor(),
	})
	if errors.Is(err, tui.ErrCanceled) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	idByName := make(map[string]string, len(all))
	for _, p := range all {
		idByName[p.Name] = p.ID
	}
	body.Privileges = body.Privileges[:0]
	for _, n := range desired {
		if id, ok := idByName[n]; ok {
			body.Privileges = append(body.Privileges, id)
		}
	}
	return true, nil
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
