package tui

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/jbrazda/iics-cli/internal/config"
)

// customLoginURL is the region option that selects a custom login URL.
const customLoginURL = "custom"

// ProfileWizardInput configures RunProfileWizard.
type ProfileWizardInput struct {
	// Name is the profile name.
	Name string
	// Existing is the stored profile, or nil for a new one. It is not
	// modified.
	Existing *config.Profile
	// Global holds the global new-user defaults, shown as inherited values.
	Global *config.NewUserConfig
	// IsDefault reports whether the profile is the default profile now.
	IsDefault bool

	Out        io.Writer
	Accessible bool
}

// ProfileWizardResult is the outcome of RunProfileWizard.
type ProfileWizardResult struct {
	Profile     *config.Profile
	MakeDefault bool
	// StoreInKeychain is true when Profile.Password should be moved to the
	// OS keychain. It is false when an unchanged keychain password is kept.
	StoreInKeychain bool
}

// profileForm holds the form values of the profile wizard.
type profileForm struct {
	username, password            string
	region, loginURL              string
	caiURL                        string
	production, makeDflt          bool
	keychain                      bool
	domain, unPat, emPat          string
	existingKeychain, hasPassword bool
}

// RunProfileWizard collects a profile in a paged form (Connection, Login
// URL when custom, Options, Keychain, New user) followed by a review. It
// returns nil when the user canceled.
func RunProfileWizard(in ProfileWizardInput) (*ProfileWizardResult, error) {
	f := newProfileForm(in)
	for {
		form := huh.NewForm(profilePages(in, f)...).WithOutput(in.Out).WithAccessible(in.Accessible)
		if err := form.Run(); err != nil {
			if errors.Is(err, huh.ErrUserAborted) {
				return nil, nil
			}
			return nil, err
		}
		res := f.result(in)
		var lines []string
		title := "Save profile " + in.Name
		if in.Existing == nil {
			title = "Create profile " + in.Name
			lines = ProfileSummary(res.Profile)
		} else {
			lines = ProfileChanges(in.Existing, res.Profile)
			if len(lines) == 0 {
				lines = []string{"No changes."}
			}
		}
		if res.MakeDefault && !in.IsDefault {
			lines = append(lines, "Set as default profile")
		}
		if res.StoreInKeychain {
			lines = append(lines, "Store password in OS keychain")
		}
		choice, err := confirmReview(in.Out, in.Accessible, title, lines, "Save", true)
		if err != nil {
			return nil, err
		}
		switch choice {
		case reviewApply:
			return res, nil
		case reviewCancel:
			return nil, nil
		}
	}
}

func newProfileForm(in ProfileWizardInput) *profileForm {
	p := &config.Profile{}
	if in.Existing != nil {
		*p = *in.Existing
	}
	f := &profileForm{
		username:         p.Username,
		caiURL:           p.CaiURL,
		production:       config.IsProductionProfile(in.Name, p),
		makeDflt:         in.Existing == nil || in.IsDefault,
		keychain:         true,
		existingKeychain: config.IsKeyringSentinel(p.Password),
		hasPassword:      p.Password != "",
	}
	if f.caiURL == "" && p.BaseAPIURL != "" {
		f.caiURL = config.DeriveCaiURL(p.BaseAPIURL)
	}
	region := strings.ToUpper(p.Region)
	switch {
	case region != "" && slices.Contains(config.Regions(), region):
		f.region = region
	case p.LoginURL != "":
		f.region, f.loginURL = customLoginURL, p.LoginURL
	}
	if p.NewUser != nil {
		f.domain, f.unPat, f.emPat = p.NewUser.Domain, p.NewUser.UserNamePattern, p.NewUser.EmailPattern
	}
	return f
}

// keepsKeychainPassword reports whether the form keeps an existing keychain
// password unchanged (the keychain question is skipped then).
func (f *profileForm) keepsKeychainPassword() bool {
	return f.password == "" && f.existingKeychain
}

func profilePages(in ProfileWizardInput, f *profileForm) []*huh.Group {
	title := "New profile " + in.Name
	if in.Existing != nil {
		title = "Profile " + in.Name
	}
	pwDesc := "Required."
	switch {
	case f.existingKeychain:
		pwDesc = "Leave empty to keep the current password (OS keychain)."
	case f.hasPassword:
		pwDesc = "Leave empty to keep the current password."
	}
	regionOpts := make([]huh.Option[string], 0, len(config.Regions())+1)
	for _, r := range config.Regions() {
		regionOpts = append(regionOpts, huh.NewOption(r, r))
	}
	regionOpts = append(regionOpts, huh.NewOption("Custom login URL", customLoginURL))

	connection := huh.NewGroup(
		huh.NewNote().Title(title),
		huh.NewInput().Title("User name").Value(&f.username).Validate(required("user name")),
		huh.NewInput().Title("Password").Description(pwDesc).
			EchoMode(huh.EchoModePassword).Value(&f.password).
			Validate(func(v string) error {
				if v == "" && !f.hasPassword {
					return errors.New("password is required")
				}
				return nil
			}),
		huh.NewSelect[string]().Title("Region").
			Description("/ filter; choose \"Custom login URL\" for a login URL").
			Options(regionOpts...).Height(8).Value(&f.region).
			Validate(func(v string) error {
				if v == "" {
					return errors.New("choose a region")
				}
				return nil
			}),
	).Title("Connection")

	login := huh.NewGroup(
		huh.NewInput().Title("Login URL").
			Description("For example https://dm-us.informaticacloud.com/saas/public/core/v3/login").
			Value(&f.loginURL).Validate(validLoginURL),
	).Title("Login URL").WithHideFunc(func() bool { return f.region != customLoginURL })

	options := huh.NewGroup(
		huh.NewInput().Title("CAI URL (optional)").
			Description("Leave empty to derive it from the org URL on login.").
			Value(&f.caiURL),
		huh.NewConfirm().Title("Production org?").
			Description("The main menu asks before running changes against a production org.").
			Value(&f.production),
		huh.NewConfirm().Title("Set as default profile?").Value(&f.makeDflt),
	).Title("Options")

	keychain := huh.NewGroup(
		huh.NewConfirm().Title("Store password in OS keychain?").
			Description("Recommended. Otherwise the password is stored in the config file.").
			Value(&f.keychain),
	).Title("Keychain").WithHideFunc(f.keepsKeychainPassword)

	inherited := func(pick func(config.NewUserConfig, config.NewUserSources) (string, string)) func() string {
		return func() string {
			v, src := config.ResolveNewUser(in.Global, &config.Profile{Username: f.username})
			val, source := pick(v, src)
			if val == "" {
				return "Leave empty to inherit (no value: set a domain here or globally)."
			}
			return fmt.Sprintf("Leave empty to inherit %s (%s).", val, source)
		}
	}
	newUser := huh.NewGroup(
		huh.NewNote().Title("New user defaults for this profile").
			Description("Placeholders: {"+strings.Join(config.UserPatternPlaceholders, "}, {")+"}.\nGlobal defaults: iics config edit."),
		huh.NewInput().Title("Domain").Value(&f.domain).
			DescriptionFunc(inherited(func(v config.NewUserConfig, s config.NewUserSources) (string, string) {
				return v.Domain, s.Domain
			}), &f.username),
		huh.NewInput().Title("User name pattern").Value(&f.unPat).Validate(optionalPattern).
			DescriptionFunc(inherited(func(v config.NewUserConfig, s config.NewUserSources) (string, string) {
				return v.UserNamePattern, s.UserNamePattern
			}), &f.username),
		huh.NewInput().Title("Email pattern").Value(&f.emPat).Validate(optionalPattern).
			DescriptionFunc(inherited(func(v config.NewUserConfig, s config.NewUserSources) (string, string) {
				return v.EmailPattern, s.EmailPattern
			}), &f.username),
	).Title("New user")

	return []*huh.Group{connection, login, options, keychain, newUser}
}

// result builds the profile from the form values.
func (f *profileForm) result(in ProfileWizardInput) *ProfileWizardResult {
	p := &config.Profile{}
	if in.Existing != nil {
		*p = *in.Existing
	}
	p.Name = in.Name
	p.Username = strings.TrimSpace(f.username)
	if f.password != "" {
		p.Password = f.password
	}
	if f.region == customLoginURL {
		p.Region, p.LoginURL = "", strings.TrimSpace(f.loginURL)
	} else {
		p.Region = f.region
		if u, err := config.LoginURL(f.region); err == nil {
			p.LoginURL = u
		}
	}
	p.CaiURL = strings.TrimSpace(f.caiURL)
	production := f.production
	p.Production = &production
	nu := config.NewUserConfig{
		Domain:          strings.TrimSpace(f.domain),
		UserNamePattern: strings.TrimSpace(f.unPat),
		EmailPattern:    strings.TrimSpace(f.emPat),
	}
	p.NewUser = nil
	if !nu.IsZero() {
		p.NewUser = &nu
	}
	return &ProfileWizardResult{
		Profile:         p,
		MakeDefault:     f.makeDflt,
		StoreInKeychain: f.keychain && !f.keepsKeychainPassword(),
	}
}

// ProfileSummary lists the settings of a profile.
func ProfileSummary(p *config.Profile) []string {
	lines := []string{
		"User name:         " + p.Username,
		"Password:          " + passwordLabel(p.Password),
		"Region:            " + orDash(p.Region),
		"Login URL:         " + orDash(p.LoginURL),
		"CAI URL:           " + orDash(p.CaiURL),
		fmt.Sprintf("Production:        %t", p.Production != nil && *p.Production),
	}
	if nu := p.NewUser; nu != nil {
		lines = append(lines,
			"New user domain:   "+orDash(nu.Domain),
			"User name pattern: "+orDash(nu.UserNamePattern),
			"Email pattern:     "+orDash(nu.EmailPattern))
	}
	return lines
}

// ProfileChanges describes the differences between two versions of a
// profile as "field: old -> new" lines. Passwords are never shown.
func ProfileChanges(before, after *config.Profile) []string {
	nu := func(p *config.Profile) config.NewUserConfig {
		if p.NewUser == nil {
			return config.NewUserConfig{}
		}
		return *p.NewUser
	}
	prod := func(p *config.Profile) string {
		return fmt.Sprint(p.Production != nil && *p.Production)
	}
	var out []string
	for _, c := range []struct{ name, old, new string }{
		{"User name", before.Username, after.Username},
		{"Region", before.Region, after.Region},
		{"Login URL", before.LoginURL, after.LoginURL},
		{"CAI URL", before.CaiURL, after.CaiURL},
		{"Production", prod(before), prod(after)},
		{"New user domain", nu(before).Domain, nu(after).Domain},
		{"User name pattern", nu(before).UserNamePattern, nu(after).UserNamePattern},
		{"Email pattern", nu(before).EmailPattern, nu(after).EmailPattern},
	} {
		if c.old != c.new {
			out = append(out, fmt.Sprintf("%s: %s -> %s", c.name, orDash(c.old), orDash(c.new)))
		}
	}
	if before.Password != after.Password {
		out = append(out, "Password: changed")
	}
	return out
}

func passwordLabel(pw string) string {
	switch {
	case pw == "":
		return "-"
	case config.IsKeyringSentinel(pw):
		return "*** (keychain)"
	default:
		return "***"
	}
}

func validLoginURL(v string) error {
	v = strings.TrimSpace(v)
	if !strings.HasPrefix(v, "https://") && !strings.HasPrefix(v, "http://") {
		return errors.New("enter a URL starting with https://")
	}
	return nil
}

func optionalPattern(v string) error {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return config.ValidateUserPattern(v)
}
