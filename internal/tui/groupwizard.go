package tui

import (
	"errors"
	"io"
	"sort"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/jbrazda/iics-cli/internal/client"
)

// GroupWizardInput configures RunGroupWizard.
type GroupWizardInput struct {
	// Group holds the starting values and receives the result.
	Group *client.UserGroup
	// Update is true when editing an existing group. Only roles can change;
	// the API cannot rename a group or edit its description.
	Update bool
	// Roles are all roles in the organization.
	Roles []client.Role
	// ExistingNames are the names of existing groups, rejected on create.
	ExistingNames []string

	Out        io.Writer
	Accessible bool
}

// RunGroupWizard collects a user group's name, description (create only) and
// roles, then shows a review. It returns false if the user canceled.
func RunGroupWizard(in GroupWizardInput) (bool, error) {
	g := in.Group
	before := roleNames(g.Roles)
	roleIDs := make([]string, 0, len(g.Roles))
	for _, r := range g.Roles {
		roleIDs = append(roleIDs, r.ID)
	}

	for {
		var groups []*huh.Group
		if in.Update {
			desc := g.Description
			if desc == "" {
				desc = "-"
			}
			groups = append(groups, huh.NewGroup(
				huh.NewNote().Title("User group: "+g.UserGroupName).
					Description("Description: "+desc+"\nName and description cannot be changed through the API."),
				rolesField(in.Roles, &roleIDs),
			))
		} else {
			groups = append(groups,
				huh.NewGroup(
					huh.NewInput().Title("Group name").Value(&g.UserGroupName).
						Validate(requiredUnique("group name", in.ExistingNames)),
					huh.NewInput().Title("Description (optional)").Value(&g.Description),
				).Title("User group"),
				huh.NewGroup(rolesField(in.Roles, &roleIDs)).Title("Roles"),
			)
		}
		err := huh.NewForm(groups...).WithOutput(in.Out).WithAccessible(in.Accessible).Run()
		if errors.Is(err, huh.ErrUserAborted) {
			return false, nil
		}
		if err != nil {
			return false, err
		}

		g.UserGroupName = strings.TrimSpace(g.UserGroupName)
		g.Description = strings.TrimSpace(g.Description)
		g.Roles = rolesByID(in.Roles, roleIDs)

		var title, applyLabel string
		var lines []string
		canApply := true
		if in.Update {
			title, applyLabel = "Update user group "+g.UserGroupName, "Apply changes"
			lines = SetChanges("Role", before, roleNames(g.Roles))
			if len(lines) == 0 {
				lines, canApply = []string{"No changes."}, false
			}
		} else {
			title, applyLabel = "Create user group "+g.UserGroupName, "Create group"
			lines = []string{
				"Name:        " + g.UserGroupName,
				"Description: " + orDash(g.Description),
				"Roles:       " + strings.Join(roleNames(g.Roles), ", "),
			}
		}
		choice, err := confirmReview(in.Out, in.Accessible, title, lines, applyLabel, canApply)
		if err != nil {
			return false, err
		}
		switch choice {
		case reviewApply:
			return true, nil
		case reviewCancel:
			return false, nil
		}
	}
}

func rolesField(roles []client.Role, roleIDs *[]string) huh.Field {
	sorted := append([]client.Role(nil), roles...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return strings.ToLower(sorted[i].RoleName) < strings.ToLower(sorted[j].RoleName)
	})
	opts := make([]huh.Option[string], len(sorted))
	for i, r := range sorted {
		opts[i] = huh.NewOption(optionLabel(r.RoleName, r.Description), r.ID)
	}
	return huh.NewMultiSelect[string]().
		Title("Roles (at least one)").
		Description("space toggle, / filter, ctrl+a all").
		Options(opts...).
		Filterable(true).
		Height(min(len(opts)+3, 14)).
		Validate(atLeastOne("role")).
		Value(roleIDs)
}
