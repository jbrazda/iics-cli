package tui

import (
	"errors"
	"fmt"
	"io"
	"net/mail"
	"sort"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/jbrazda/iics-cli/internal/client"
)

// UserWizardInput configures RunUserWizard.
type UserWizardInput struct {
	// User holds the starting values and receives the result.
	User *client.User
	// Update is true when editing an existing user. Only user group and role
	// assignments are edited then; other fields are shown read-only because
	// the API cannot update them.
	Update bool
	// Groups and Roles are all groups and roles in the organization.
	Groups []client.UserGroup
	Roles  []client.Role
	// UserNameDomain suggests "first.last@<domain>" as the user name.
	UserNameDomain string

	Out        io.Writer
	Accessible bool
}

// RunUserWizard collects user fields in a paged form (Identity, Details,
// Membership) followed by a review. On update only the Membership page is
// shown. It returns false if the user canceled.
func RunUserWizard(in UserWizardInput) (bool, error) {
	before := cloneUser(in.User)
	u := in.User

	auth := u.Authentication
	if auth == "" {
		auth = "Native"
	}
	groupIDs := make([]string, 0, len(u.Groups))
	for _, g := range u.Groups {
		groupIDs = append(groupIDs, g.ID)
	}
	roleIDs := make([]string, 0, len(u.Roles))
	for _, r := range u.Roles {
		roleIDs = append(roleIDs, r.ID)
	}

	for {
		pages := []*huh.Group{
			identityGroup(in, u, &auth),
			ssoGroup(in, u, &auth),
			detailsGroup(u),
			membershipGroup(in, &groupIDs, &roleIDs),
		}
		if in.Update {
			// Only role and user group assignments can be updated.
			pages = []*huh.Group{membershipGroup(in, &groupIDs, &roleIDs)}
		}
		form := huh.NewForm(pages...).WithOutput(in.Out).WithAccessible(in.Accessible)
		if err := form.Run(); err != nil {
			if errors.Is(err, huh.ErrUserAborted) {
				return false, nil
			}
			return false, err
		}

		if !in.Update {
			u.Authentication = auth
			if auth != "SSO" {
				u.AliasName = ""
			}
			if strings.TrimSpace(u.UserName) == "" {
				u.UserName = suggestUserName(u.FirstName, u.LastName, in.UserNameDomain)
			}
		}
		trimUser(u)
		u.Groups = groupsByID(in.Groups, groupIDs)
		u.Roles = rolesByID(in.Roles, roleIDs)

		choice, err := reviewUser(in, before, u)
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

func identityGroup(in UserWizardInput, u *client.User, auth *string) *huh.Group {
	return huh.NewGroup(
		huh.NewSelect[string]().
			Title("Authentication").
			Options(huh.NewOption("Native", "Native"), huh.NewOption("SSO", "SSO")).
			Value(auth),
		huh.NewInput().Title("First name").Value(&u.FirstName).Validate(required("first name")),
		huh.NewInput().Title("Last name").Value(&u.LastName).Validate(required("last name")),
		huh.NewInput().
			Title("User name").
			Description("Leave empty to use the suggestion").
			PlaceholderFunc(func() string {
				return suggestUserName(u.FirstName, u.LastName, in.UserNameDomain)
			}, []any{&u.FirstName, &u.LastName}).
			Value(&u.UserName).
			Validate(func(v string) error {
				if strings.TrimSpace(v) == "" && suggestUserName(u.FirstName, u.LastName, in.UserNameDomain) == "" {
					return errors.New("user name is required")
				}
				return nil
			}),
		huh.NewInput().Title("Email").Value(&u.Email).Validate(validEmail),
	).Title("Identity")
}

// ssoGroup asks for the SSO alias name; the create API requires it when
// authentication is SSO. Hidden otherwise.
func ssoGroup(in UserWizardInput, u *client.User, auth *string) *huh.Group {
	return huh.NewGroup(
		huh.NewInput().
			Title("SSO alias name").
			Description("User identifier in the identity provider").
			Value(&u.AliasName).
			Validate(required("alias name")),
	).Title("Single sign-on").WithHideFunc(func() bool {
		return in.Update || *auth != "SSO"
	})
}

func detailsGroup(u *client.User) *huh.Group {
	return huh.NewGroup(
		huh.NewInput().Title("Phone (optional)").Value(&u.Phone),
		huh.NewInput().Title("Title (optional)").Value(&u.Title),
		huh.NewInput().Title("Description (optional)").Value(&u.Description),
		huh.NewConfirm().Title("Force password change on next login?").Value(&u.ForcePasswordChange),
	).Title("Details")
}

func membershipGroup(in UserWizardInput, groupIDs, roleIDs *[]string) *huh.Group {
	groups := append([]client.UserGroup(nil), in.Groups...)
	sort.SliceStable(groups, func(i, j int) bool {
		return strings.ToLower(groups[i].UserGroupName) < strings.ToLower(groups[j].UserGroupName)
	})
	gOpts := make([]huh.Option[string], len(groups))
	for i, g := range groups {
		gOpts[i] = huh.NewOption(optionLabel(g.UserGroupName, g.Description), g.ID)
	}
	roles := append([]client.Role(nil), in.Roles...)
	sort.SliceStable(roles, func(i, j int) bool {
		return strings.ToLower(roles[i].RoleName) < strings.ToLower(roles[j].RoleName)
	})
	rOpts := make([]huh.Option[string], len(roles))
	for i, r := range roles {
		rOpts[i] = huh.NewOption(optionLabel(r.RoleName, r.Description), r.ID)
	}

	var fields []huh.Field
	if in.Update {
		u := in.User
		fields = append(fields, huh.NewNote().
			Title("User: "+u.UserName).
			Description(fmt.Sprintf("%s  %s   Authentication: %s   State: %s\nOnly user group and role assignments can be changed.",
				strings.TrimSpace(u.FirstName+" "+u.LastName), orDash(u.Email), orDash(u.Authentication), orDash(u.State))))
	}
	if len(gOpts) > 0 {
		fields = append(fields, huh.NewMultiSelect[string]().
			Title("User groups").
			Description("space toggle, / filter, ctrl+a all").
			Options(gOpts...).
			Filterable(true).
			Height(min(len(gOpts)+3, 12)).
			Value(groupIDs))
	}
	if len(rOpts) > 0 {
		fields = append(fields, huh.NewMultiSelect[string]().
			Title("Roles").
			Description("space toggle, / filter, ctrl+a all").
			Options(rOpts...).
			Filterable(true).
			Height(min(len(rOpts)+3, 12)).
			Value(roleIDs))
	}
	if len(fields) == 0 {
		fields = append(fields, huh.NewNote().Title("No user groups or roles available"))
	}
	return huh.NewGroup(fields...).Title("Membership")
}

// reviewUser shows the result (create) or the changes (update) and returns
// reviewApply, reviewBack or reviewCancel.
func reviewUser(in UserWizardInput, before, after *client.User) (string, error) {
	if in.Update {
		lines := UserChanges(before, after)
		canApply := len(lines) > 0
		if !canApply {
			lines = []string{"No changes."}
		}
		return confirmReview(in.Out, in.Accessible, "Update user "+after.UserName, lines, "Apply changes", canApply)
	}
	lines := UserSummary(after)
	// The create API requires at least one role or user group.
	canApply := len(after.Groups) > 0 || len(after.Roles) > 0
	if !canApply {
		lines = append(lines, "", "Select at least one user group or role.")
	}
	return confirmReview(in.Out, in.Accessible, "Create user "+after.UserName, lines, "Create user", canApply)
}

// UserSummary lists the fields of a user to be created.
func UserSummary(u *client.User) []string {
	lines := []string{
		"User name:      " + u.UserName,
		"Name:           " + strings.TrimSpace(u.FirstName+" "+u.LastName),
		"Email:          " + orDash(u.Email),
		"Authentication: " + orDash(u.Authentication),
	}
	if u.AliasName != "" {
		lines = append(lines, "SSO alias:      "+u.AliasName)
	}
	for _, f := range []struct{ label, v string }{
		{"Phone:          ", u.Phone},
		{"Title:          ", u.Title},
		{"Description:    ", u.Description},
		{"Time zone:      ", u.TimeZoneID},
	} {
		if f.v != "" {
			lines = append(lines, f.label+f.v)
		}
	}
	if u.ForcePasswordChange {
		lines = append(lines, "Force password change on next login")
	}
	lines = append(lines,
		"Groups:         "+orDash(strings.Join(groupNames(u.Groups), ", ")),
		"Roles:          "+orDash(strings.Join(roleNames(u.Roles), ", ")))
	return lines
}

// UserChanges describes the differences between two versions of a user in
// the fields the update applies: scalar fields as "field: old -> new",
// groups and roles as "+ name" / "- name".
func UserChanges(before, after *client.User) []string {
	var out []string
	for _, f := range []struct{ name, old, new string }{
		{"First name", before.FirstName, after.FirstName},
		{"Last name", before.LastName, after.LastName},
		{"Email", before.Email, after.Email},
		{"Phone", before.Phone, after.Phone},
		{"Title", before.Title, after.Title},
		{"Description", before.Description, after.Description},
		{"Time zone", before.TimeZoneID, after.TimeZoneID},
	} {
		if f.old != f.new {
			out = append(out, fmt.Sprintf("%s: %s -> %s", f.name, orDash(f.old), orDash(f.new)))
		}
	}
	if before.ForcePasswordChange != after.ForcePasswordChange {
		out = append(out, fmt.Sprintf("Force password change: %t -> %t", before.ForcePasswordChange, after.ForcePasswordChange))
	}
	out = append(out, SetChanges("Group", groupNames(before.Groups), groupNames(after.Groups))...)
	out = append(out, SetChanges("Role", roleNames(before.Roles), roleNames(after.Roles))...)
	return out
}

func suggestUserName(first, last, domain string) string {
	first, last = strings.TrimSpace(first), strings.TrimSpace(last)
	if first == "" || last == "" || domain == "" {
		return ""
	}
	return fmt.Sprintf("%s.%s@%s", strings.ToLower(first), strings.ToLower(last), domain)
}

func required(what string) func(string) error {
	return func(v string) error {
		if strings.TrimSpace(v) == "" {
			return fmt.Errorf("%s is required", what)
		}
		return nil
	}
}

func validEmail(v string) error {
	v = strings.TrimSpace(v)
	if v == "" {
		return errors.New("email is required")
	}
	if _, err := mail.ParseAddress(v); err != nil {
		return fmt.Errorf("invalid email address")
	}
	return nil
}

func optionLabel(name, desc string) string {
	desc = strings.Join(strings.Fields(desc), " ")
	if desc == "" {
		return name
	}
	const maxDesc = 50
	if r := []rune(desc); len(r) > maxDesc {
		desc = string(r[:maxDesc-1]) + "…"
	}
	return name + " - " + desc
}

func trimUser(u *client.User) {
	for _, s := range []*string{&u.FirstName, &u.LastName, &u.UserName, &u.Email, &u.Phone, &u.Title, &u.Description} {
		*s = strings.TrimSpace(*s)
	}
}

func groupsByID(all []client.UserGroup, ids []string) []client.UserGroupRef {
	want := toSet(ids)
	var out []client.UserGroupRef
	for _, g := range all {
		if want[g.ID] {
			out = append(out, client.UserGroupRef{ID: g.ID, UserGroupName: g.UserGroupName})
		}
	}
	return out
}

func rolesByID(all []client.Role, ids []string) []client.UserRole {
	want := toSet(ids)
	var out []client.UserRole
	for _, r := range all {
		if want[r.ID] {
			out = append(out, client.UserRole{ID: r.ID, RoleName: r.RoleName, DisplayName: r.DisplayName})
		}
	}
	return out
}

func groupNames(gs []client.UserGroupRef) []string {
	out := make([]string, len(gs))
	for i, g := range gs {
		out[i] = g.UserGroupName
	}
	sort.Strings(out)
	return out
}

func roleNames(rs []client.UserRole) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.RoleName
	}
	sort.Strings(out)
	return out
}

func cloneUser(u *client.User) *client.User {
	c := *u
	c.Groups = append([]client.UserGroupRef(nil), u.Groups...)
	c.Roles = append([]client.UserRole(nil), u.Roles...)
	return &c
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
