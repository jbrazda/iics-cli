package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/jbrazda/iics-cli/internal/config"
	"github.com/jbrazda/iics-cli/internal/menu"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/term"
)

func newMenuCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "menu",
		Aliases: []string{"tui"},
		Short:   "Open the interactive main menu",
		Long: `Open the interactive main menu: pick an entry to run the matching command,
then return to the menu. Running "iics" without arguments in a terminal opens
the same menu (disable with IICS_NO_MENU=1 or "ui: {menu: false}" in the
config file). Requires a terminal.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !isInteractiveTTY() {
				return fmt.Errorf("the menu requires a terminal; run 'iics --help' for commands")
			}
			return runMenu(cmd)
		},
	}
}

// rootRun opens the menu for bare "iics" on a terminal, otherwise prints help.
func rootRun(cmd *cobra.Command, args []string) error {
	cfg, _ := loadConfig()
	var cfgMenu *bool
	if cfg != nil {
		cfgMenu = cfg.UI.Menu
	}
	open := menu.ShouldOpen(menu.OpenInput{
		StdinTTY:   term.IsTerminal(int(os.Stdin.Fd())),
		StdoutTTY:  term.IsTerminal(int(os.Stdout.Fd())),
		Env:        os.Getenv,
		ConfigMenu: cfgMenu,
	})
	if len(args) > 0 || !open {
		return cmd.Help()
	}
	return runMenu(cmd)
}

func runMenu(cmd *cobra.Command) error {
	binary, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locating the iics binary: %w", err)
	}
	display := filepath.Base(os.Args[0])
	passthrough := globalFlagArgs(cmd.Root())

	_, _, start, _ := resolveProfile()

	line := prompter.Accessible
	if w, h, serr := term.GetSize(int(os.Stdout.Fd())); serr == nil && (w < 60 || h < 15) {
		line = true
	}

	return menu.Run(menu.Deps{
		Runner:      passthroughRunner{inner: menu.ExecRunner{Binary: binary}, extra: passthrough},
		Prompter:    prompter,
		Out:         os.Stderr,
		Binary:      display,
		NoColor:     editorNoColor(),
		Line:        line,
		LoadState:   loadMenuState,
		SetDefault:  setDefaultProfile,
		ShowMenu:    menu.DefaultShowMenu,
		Pause:       menu.DefaultPause(os.Stderr),
		ClearScreen: menu.DefaultClearScreen(os.Stderr),
	}, start)
}

// passthroughRunner appends the global flags the menu was started with.
type passthroughRunner struct {
	inner menu.Runner
	extra []string
}

func (r passthroughRunner) Run(args []string) int {
	return r.inner.Run(append(append([]string(nil), args...), r.extra...))
}

// globalFlagArgs returns explicitly set persistent flags (except --profile,
// which the menu manages, and --output) as "--name=value" arguments.
func globalFlagArgs(root *cobra.Command) []string {
	var out []string
	// Not Visit: cobra parses on the invoked command's own flag set, so the
	// root set never records which of its flags were set.
	root.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		if !f.Changed {
			return
		}
		switch f.Name {
		case "profile", "output", "help":
			return
		}
		out = append(out, fmt.Sprintf("--%s=%s", f.Name, f.Value.String()))
	})
	return out
}

func loadMenuState(profileName string) (menu.State, error) {
	var st menu.State
	cfg, err := loadConfig()
	if err != nil {
		return st, err
	}
	if profileName == "" {
		profileName = cfg.DefaultProfile
	}
	names := make([]string, 0, len(cfg.Profiles))
	for n := range cfg.Profiles {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		p := cfg.Profiles[n]
		st.Profiles = append(st.Profiles, menu.ProfileInfo{
			Name:       n,
			Region:     p.Region,
			Production: config.IsProductionProfile(n, p),
			Default:    n == cfg.DefaultProfile,
		})
	}

	p := cfg.Profiles[profileName]
	st.HasProfile = p != nil || os.Getenv("IICS_USERNAME") != ""
	st.Header = menu.Header{
		Profile:    profileName,
		Production: config.IsProductionProfile(profileName, p),
		Session:    "not logged in - signs in on first command",
	}
	if cache, cerr := config.LoadSessionCache(""); cerr == nil && cache != nil {
		if e := cache.Sessions[profileName]; e != nil {
			st.Header.Org = e.OrgName
			st.Header.User = e.UserName
			if rem := e.Remaining(); rem > 0 {
				st.Header.Session = fmt.Sprintf("session ok (%d min left)", int(rem.Minutes())+1)
			} else {
				st.Header.Session = "session expired - signs in on first command"
			}
		}
	}
	return st, nil
}

func setDefaultProfile(name string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	if _, ok := cfg.Profiles[name]; !ok {
		return fmt.Errorf("profile %q is not in the config file", name)
	}
	cfg.DefaultProfile = name
	return cfg.Save(cfgFile)
}
