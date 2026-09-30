package menu

import (
	"strconv"
	"strings"
)

// OpenInput holds what decides whether bare "iics" opens the menu.
type OpenInput struct {
	StdinTTY, StdoutTTY bool
	// Env returns an environment variable (os.Getenv).
	Env func(string) string
	// ConfigMenu is the config ui.menu setting (nil when unset).
	ConfigMenu *bool
}

// ShouldOpen reports whether bare "iics" should open the menu instead of
// printing help: stdin and stdout are terminals, CI is not set,
// IICS_NO_MENU is not true, and config ui.menu is not false.
func ShouldOpen(in OpenInput) bool {
	if !in.StdinTTY || !in.StdoutTTY {
		return false
	}
	env := in.Env
	if env == nil {
		env = func(string) string { return "" }
	}
	if strings.TrimSpace(env("CI")) != "" {
		return false
	}
	if v, err := strconv.ParseBool(strings.TrimSpace(env("IICS_NO_MENU"))); err == nil && v {
		return false
	}
	if in.ConfigMenu != nil && !*in.ConfigMenu {
		return false
	}
	return true
}
