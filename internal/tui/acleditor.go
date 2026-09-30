package tui

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/tui/grid"
)

// ACLEditorInput configures EditACLs.
type ACLEditorInput struct {
	// ObjectLabel names the object in titles, e.g. "Default/Sales (Folder)".
	ObjectLabel string
	// Current are the object's ACLs now.
	Current []client.ObjectACL
	// Users and Groups are the principals that can be added.
	Users, Groups []string
	// CurrentUser is the signed-in user name, used to warn when the change
	// removes that user's own change permission.
	CurrentUser string

	Out        io.Writer
	Accessible bool
	NoColor    bool
}

var aclColumns = []string{"READ", "UPDATE", "DELETE", "EXEC", "PERM"}

// aclPermissionKeys names the permission of each column, in order.
var aclPermissionKeys = []string{"read", "update", "delete", "execute", "changePermission"}

type aclRow struct {
	principal client.ACLPrincipal
	added     bool
}

func aclCellKey(p client.ACLPrincipal, perm string) string {
	return client.PrincipalKey(p) + "\x1e" + perm
}

func permsToSet(set map[string]bool, p client.ACLPrincipal, perms client.ACLPermissions) {
	set[aclCellKey(p, "read")] = perms.Read
	set[aclCellKey(p, "update")] = perms.Update
	set[aclCellKey(p, "delete")] = perms.Delete
	set[aclCellKey(p, "execute")] = perms.Execute
	set[aclCellKey(p, "changePermission")] = perms.ChangePermission
}

func permsFromSet(set map[string]bool, p client.ACLPrincipal) client.ACLPermissions {
	return client.ACLPermissions{
		Read:             set[aclCellKey(p, "read")],
		Update:           set[aclCellKey(p, "update")],
		Delete:           set[aclCellKey(p, "delete")],
		Execute:          set[aclCellKey(p, "execute")],
		ChangePermission: set[aclCellKey(p, "changePermission")],
	}
}

// EditACLs runs the ACL grid for one object and returns the change plan. ok
// is false when the user canceled. A principal left with no permissions has
// its ACL deleted.
func EditACLs(in ACLEditorInput) (client.ACLPlan, bool, error) {
	original := map[string]bool{}
	selected := map[string]bool{}
	var rows []aclRow
	for _, a := range in.Current {
		rows = append(rows, aclRow{principal: a.Principal})
		permsToSet(original, a.Principal, a.Permissions)
		permsToSet(selected, a.Principal, a.Permissions)
	}

	for {
		m := grid.New(aclGridRows(rows), original, selected, grid.Options{
			Title:       "Permissions: " + in.ObjectLabel,
			Columns:     aclColumns,
			KeyHeader:   "PRINCIPAL",
			NoColor:     in.NoColor,
			ClearKey:    "d",
			EmptyText:   "No ACLs: access follows roles. Press n to add a user group or user.",
			Actions:     map[string]string{"n": "add principal"},
			ActionOrder: []string{"n"},
		})
		out := in.Out
		if out == nil {
			out = os.Stderr
		}
		if _, err := tea.NewProgram(m, tea.WithAltScreen(), tea.WithOutput(out)).Run(); err != nil {
			return client.ACLPlan{}, false, fmt.Errorf("permission editor: %w", err)
		}

		if m.Action() == "n" {
			p, ok, err := pickPrincipal(in, rows)
			if err != nil {
				return client.ACLPlan{}, false, err
			}
			if ok {
				rows = append(rows, aclRow{principal: p, added: true})
				selected[aclCellKey(p, "read")] = true
			}
			continue
		}

		desired := desiredACLs(rows, selected)
		plan := client.DiffACLs(in.Current, desired, true)
		lines := ACLPlanLines(plan)
		canApply := !plan.Empty()
		if !canApply {
			lines = []string{"No changes."}
		}
		if w := selfLockoutWarning(in.CurrentUser, in.Current, desired); w != "" {
			lines = append(lines, "", w)
		}
		choice, err := confirmReview(in.Out, in.Accessible, "Permissions: "+in.ObjectLabel, lines, "Apply changes", canApply)
		if err != nil {
			return client.ACLPlan{}, false, err
		}
		switch choice {
		case reviewApply:
			return plan, true, nil
		case reviewCancel:
			return client.ACLPlan{}, false, nil
		}
	}
}

func aclGridRows(rows []aclRow) []grid.Row {
	sorted := append([]aclRow(nil), rows...)
	sort.SliceStable(sorted, func(i, j int) bool {
		a, b := sorted[i].principal, sorted[j].principal
		if a.Type != b.Type {
			return a.Type == client.PrincipalGroup // groups first
		}
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	})
	out := make([]grid.Row, len(sorted))
	for i, r := range sorted {
		tag := strings.ToLower(r.principal.Type)
		if r.added {
			tag += " (new)"
		}
		cells := make([]grid.Cell, len(aclPermissionKeys))
		for j, perm := range aclPermissionKeys {
			cells[j] = grid.Cell{
				Key:    aclCellKey(r.principal, perm),
				Detail: fmt.Sprintf("%s %s - %s", strings.ToLower(r.principal.Type), r.principal.Name, perm),
			}
		}
		out[i] = grid.Row{Label: r.principal.Name, Tag: tag, Cells: cells, Match: r.principal.Type}
	}
	return out
}

// desiredACLs returns the ACLs for principals with at least one permission.
func desiredACLs(rows []aclRow, selected map[string]bool) []client.ObjectACL {
	var out []client.ObjectACL
	for _, r := range rows {
		perms := permsFromSet(selected, r.principal)
		if perms != (client.ACLPermissions{}) {
			out = append(out, client.ObjectACL{Principal: r.principal, Permissions: perms})
		}
	}
	return out
}

// ACLPlanLines describes a plan: "+ group X: read", "~ user Y: read,update",
// "- group Z".
func ACLPlanLines(plan client.ACLPlan) []string {
	var lines []string
	for _, a := range plan.Create {
		lines = append(lines, fmt.Sprintf("+ %s %s: %s", strings.ToLower(a.Principal.Type), a.Principal.Name, a.Permissions))
	}
	for _, a := range plan.Update {
		lines = append(lines, fmt.Sprintf("~ %s %s: %s", strings.ToLower(a.Principal.Type), a.Principal.Name, a.Permissions))
	}
	for _, a := range plan.Delete {
		lines = append(lines, fmt.Sprintf("- %s %s", strings.ToLower(a.Principal.Type), a.Principal.Name))
	}
	return lines
}

// selfLockoutWarning returns a warning when the signed-in user had change
// permission through a user ACL and the desired ACLs drop it.
func selfLockoutWarning(user string, current, desired []client.ObjectACL) string {
	if user == "" {
		return ""
	}
	me := client.ACLPrincipal{Type: client.PrincipalUser, Name: user}
	key := client.PrincipalKey(me)
	had := false
	for _, a := range current {
		if client.PrincipalKey(a.Principal) == key && a.Permissions.ChangePermission {
			had = true
		}
	}
	if !had {
		return ""
	}
	for _, a := range desired {
		if client.PrincipalKey(a.Principal) == key && a.Permissions.ChangePermission {
			return ""
		}
	}
	return fmt.Sprintf("Warning: this removes change permission from your own user (%s).", user)
}

// pickPrincipal asks for a user or group that has no row yet.
func pickPrincipal(in ACLEditorInput, rows []aclRow) (client.ACLPrincipal, bool, error) {
	present := make(map[string]bool, len(rows))
	for _, r := range rows {
		present[client.PrincipalKey(r.principal)] = true
	}
	var opts []huh.Option[string]
	add := func(typ string, names []string) {
		sorted := append([]string(nil), names...)
		sort.Slice(sorted, func(i, j int) bool { return strings.ToLower(sorted[i]) < strings.ToLower(sorted[j]) })
		for _, n := range sorted {
			p := client.ACLPrincipal{Type: typ, Name: n}
			if !present[client.PrincipalKey(p)] {
				opts = append(opts, huh.NewOption(fmt.Sprintf("%-5s %s", strings.ToLower(typ), n), typ+"\x1f"+n))
			}
		}
	}
	add(client.PrincipalGroup, in.Groups)
	add(client.PrincipalUser, in.Users)
	if len(opts) == 0 {
		return client.ACLPrincipal{}, false, nil
	}
	var choice string
	sel := huh.NewSelect[string]().
		Title("Add a user group or user").
		Description("/ to filter, ctrl+c to go back").
		Options(opts...).
		Height(min(len(opts)+3, 18)).
		Value(&choice)
	err := huh.NewForm(huh.NewGroup(sel)).WithOutput(in.Out).WithAccessible(in.Accessible).Run()
	if errors.Is(err, huh.ErrUserAborted) {
		return client.ACLPrincipal{}, false, nil
	}
	if err != nil {
		return client.ACLPrincipal{}, false, err
	}
	typ, name, _ := strings.Cut(choice, "\x1f")
	return client.ACLPrincipal{Type: typ, Name: name}, true, nil
}
