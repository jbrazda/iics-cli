package tui

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/jbrazda/iics-cli/internal/config"
)

// ConfigWizardInput configures RunConfigWizard.
type ConfigWizardInput struct {
	// Config is edited in place when the user saves.
	Config *config.Config
	// Themes are the table theme names.
	Themes []string
	// ThemePreview renders a sample table in a theme; nil shows no preview.
	ThemePreview func(theme, headerColor string) string

	Out        io.Writer
	Accessible bool
}

// configForm holds the form values of the config wizard.
type configForm struct {
	domain, unPat, emPat string
	theme, headerColor   string
	noColor, responsive  bool
	timeout              string
	menu                 bool
}

// RunConfigWizard edits the global settings (new user defaults,
// appearance, HTTP timeout, main menu) followed by a review of the changes.
// It returns false when nothing was saved.
func RunConfigWizard(in ConfigWizardInput) (bool, error) {
	f := newConfigForm(in.Config)
	for {
		form := huh.NewForm(configPages(in, f)...).WithOutput(in.Out).WithAccessible(in.Accessible)
		if err := form.Run(); err != nil {
			if errors.Is(err, huh.ErrUserAborted) {
				return false, nil
			}
			return false, err
		}
		after := f.apply(in.Config)
		lines := ConfigChanges(in.Config, after)
		canApply := len(lines) > 0
		if !canApply {
			lines = []string{"No changes."}
		}
		choice, err := confirmReview(in.Out, in.Accessible, "Global settings", lines, "Save", canApply)
		if err != nil {
			return false, err
		}
		switch choice {
		case reviewApply:
			*in.Config = *after
			return true, nil
		case reviewCancel:
			return false, nil
		}
	}
}

func newConfigForm(c *config.Config) *configForm {
	f := &configForm{
		theme:       c.Style.Theme,
		headerColor: c.Style.HeaderColor,
		noColor:     c.Style.NoColor,
		responsive:  c.Style.ResponsiveTablesEnabled(),
		menu:        c.UI.Menu == nil || *c.UI.Menu,
	}
	if c.NewUser != nil {
		f.domain, f.unPat, f.emPat = c.NewUser.Domain, c.NewUser.UserNamePattern, c.NewUser.EmailPattern
	}
	if c.HTTPTimeout > 0 {
		f.timeout = strconv.Itoa(c.HTTPTimeout)
	}
	return f
}

func configPages(in ConfigWizardInput, f *configForm) []*huh.Group {
	newUser := huh.NewGroup(
		huh.NewNote().Title("New user defaults (all profiles)").
			Description("Profiles can override each value (iics profile edit).\nPlaceholders: {"+strings.Join(config.UserPatternPlaceholders, "}, {")+"}."),
		huh.NewInput().Title("Domain").
			Description("Leave empty to use the domain of each profile's user name.").
			Value(&f.domain),
		huh.NewInput().Title("User name pattern").
			Description("Leave empty for the built-in default "+config.DefaultUserNamePattern).
			Value(&f.unPat).Validate(optionalPattern),
		huh.NewInput().Title("Email pattern").
			Description("Leave empty for the built-in default "+config.DefaultEmailPattern).
			Value(&f.emPat).Validate(optionalPattern),
	).Title("New user")

	themeOpts := []huh.Option[string]{huh.NewOption("(automatic) default on a terminal, markdown when piped", "")}
	for _, t := range in.Themes {
		themeOpts = append(themeOpts, huh.NewOption(t, t))
	}
	theme := huh.NewSelect[string]().Title("Table theme").Options(themeOpts...).Value(&f.theme)
	if in.ThemePreview != nil {
		theme.DescriptionFunc(func() string {
			t := f.theme
			if t == "" {
				t = "default"
			}
			return in.ThemePreview(t, f.headerColor)
		}, &f.theme)
	}
	appearance := huh.NewGroup(
		theme,
		huh.NewInput().Title("Header color").
			Description(`Lipgloss color for default and minimal themes: "6" cyan, "244" gray, "#FF0000". Empty uses the theme default.`).
			Value(&f.headerColor),
		huh.NewConfirm().Title("Disable colors?").Value(&f.noColor),
		huh.NewConfirm().Title("Responsive tables?").
			Description("Drop, truncate or wrap columns to fit the terminal width (off is the same as --wide).").
			Value(&f.responsive),
	).Title("Appearance")

	other := huh.NewGroup(
		huh.NewInput().Title("HTTP timeout (seconds)").
			Description(fmt.Sprintf("Per request. Leave empty for the built-in default (%d).", config.DefaultHTTPTimeoutSeconds)).
			Value(&f.timeout).Validate(func(v string) error {
			v = strings.TrimSpace(v)
			if v == "" {
				return nil
			}
			if n, err := strconv.Atoi(v); err != nil || n <= 0 {
				return errors.New("enter a whole number of seconds greater than 0")
			}
			return nil
		}),
		huh.NewConfirm().Title("Open the main menu when iics runs without a command?").Value(&f.menu),
	).Title("Other")

	return []*huh.Group{newUser, appearance, other}
}

// apply returns a copy of c with the form values.
func (f *configForm) apply(c *config.Config) *config.Config {
	out := *c
	nu := config.NewUserConfig{
		Domain:          strings.TrimSpace(f.domain),
		UserNamePattern: strings.TrimSpace(f.unPat),
		EmailPattern:    strings.TrimSpace(f.emPat),
	}
	out.NewUser = nil
	if !nu.IsZero() {
		out.NewUser = &nu
	}
	out.Style.Theme = f.theme
	out.Style.HeaderColor = strings.TrimSpace(f.headerColor)
	out.Style.NoColor = f.noColor
	// Keep "unset" when the value matches the default (true).
	if f.responsive != c.Style.ResponsiveTablesEnabled() || c.Style.ResponsiveTables != nil {
		r := f.responsive
		out.Style.ResponsiveTables = &r
	}
	out.HTTPTimeout, _ = strconv.Atoi(strings.TrimSpace(f.timeout))
	if f.menu != (c.UI.Menu == nil || *c.UI.Menu) || c.UI.Menu != nil {
		m := f.menu
		out.UI.Menu = &m
	}
	return &out
}

// ConfigChanges describes the differences in global settings between two
// configs as "setting: old -> new" lines.
func ConfigChanges(before, after *config.Config) []string {
	nu := func(c *config.Config) config.NewUserConfig {
		if c.NewUser == nil {
			return config.NewUserConfig{}
		}
		return *c.NewUser
	}
	timeout := func(c *config.Config) string {
		if c.HTTPTimeout <= 0 {
			return ""
		}
		return strconv.Itoa(c.HTTPTimeout)
	}
	menu := func(c *config.Config) string { return fmt.Sprint(c.UI.Menu == nil || *c.UI.Menu) }
	var out []string
	for _, c := range []struct{ name, old, new string }{
		{"New user domain", nu(before).Domain, nu(after).Domain},
		{"User name pattern", nu(before).UserNamePattern, nu(after).UserNamePattern},
		{"Email pattern", nu(before).EmailPattern, nu(after).EmailPattern},
		{"Table theme", before.Style.Theme, after.Style.Theme},
		{"Header color", before.Style.HeaderColor, after.Style.HeaderColor},
		{"No color", fmt.Sprint(before.Style.NoColor), fmt.Sprint(after.Style.NoColor)},
		{"Responsive tables", fmt.Sprint(before.Style.ResponsiveTablesEnabled()), fmt.Sprint(after.Style.ResponsiveTablesEnabled())},
		{"HTTP timeout", timeout(before), timeout(after)},
		{"Main menu", menu(before), menu(after)},
	} {
		if c.old != c.new {
			out = append(out, fmt.Sprintf("%s: %s -> %s", c.name, orDash(c.old), orDash(c.new)))
		}
	}
	return out
}
