# CR-0044: Interactive main menu

## CR Type

- [x] New command
- [x] Enhancement to existing command

## Problem

Many commands now have interactive modes (role, user, group, permission,
environment and agent wizards and editors), but users must know each command
name and flag to reach them. Running `iics` without arguments only prints the
help text.

## Proposed Solution

A hybrid CLI main menu: interactive when a person runs `iics` in a terminal,
unchanged for scripts.

### When the menu opens

- `iics` with no arguments opens the menu when stdin **and** stdout are
  terminals. Otherwise the help text is printed as today.
- `iics menu` (alias `iics tui`) always opens it (terminal required).
- `iics --help` / `iics help` always print help.
- Opt-out: `IICS_NO_MENU=1`, config `ui.menu: false`, or `CI` set in the
  environment - bare `iics` then prints help.

### Running commands

- Each entry runs the real command as a subprocess of the same binary with
  the terminal passed through and `--profile <active>` appended. No flag or
  global state leaks between runs.
- The equivalent command line is printed (dimmed) before it runs, e.g.
  `$ iics role edit --profile dev`.
- Output stays on the normal screen. After the command exits, a status line
  (`✓ done` or `✗ failed (exit N)`) is shown; `Enter` returns to the menu
  (screen cleared), `q` quits.
- `Ctrl+C` while a command runs cancels that command only (the menu ignores
  the signal while the child runs). In the menu, `Ctrl+C` or `q` quits with
  exit code 0.
- The cursor stays on the last entry for the session (not persisted).

### Profiles and production safety

- The active profile starts from the normal resolution order (`--profile`,
  `IICS_PROFILE`, config default).
- Switching profile (`p` or the Session entry) applies to the menu session;
  a separate "Set as default profile" entry writes `defaultProfile`.
- Header (from the local session cache, no network call): profile, org,
  user, session status ("session ok (12 min left)" or "not logged in - signs
  in on first command"), and a red `PRODUCTION` marker.
- Production detection: per-profile config `production: true|false`; when
  unset, names `prd`, `prod`, `production` (or starting with `prod-` / `prd-`)
  are treated as production. `profile add` / `profile edit` ask for the flag.
- On a production profile, entries that change data ask for confirmation
  (default No) before running. Read-only entries never ask.
- With no profiles configured, `profile add` runs first; canceling it opens
  the menu with only the Session group usable.

### Menu content

A curated list of entries (group, label, description, arguments, writes
flag), grouped by resource:

| Group | Entries |
| ----- | ------- |
| Session | Switch profile, Set as default profile, Login, List profiles, Add profile ✎, Edit active profile ✎ |
| Users | List, Create ✎, Edit groups and roles ✎, Delete ✎, Change password ✎ |
| User groups | List, Show, Create ✎, Edit roles ✎, Delete ✎ |
| Roles | List, Show with privileges, Create ✎, Edit privileges ✎ |
| Privileges | List |
| Permissions | Edit object permissions ✎ |
| Environments and agents | List environments, Create environment ✎, List agents, Start / Stop / Restart agent service ✎, Download agent installer |
| Help | Show CLI help |

✎ = changes data (production confirmation).

### Look and fallback

- Full-screen Bubble Tea menu: header bar, section headers, `/` filter
  across groups, description and equivalent command for the focused entry,
  `p` switch profile, `?` key help, `q` quit.
- `IICS_ACCESSIBLE=1` or a terminal smaller than 60x15: a numbered,
  line-based menu with the same entries and the same run/return loop.

## Implementation

- `internal/config`: `Profile.Production *bool`, `Config.UI.Menu *bool`,
  `IsProductionProfile`, `SessionEntry.Remaining`; production question in the
  profile prompt.
- `internal/menu`: entries, menu model, `ShouldOpen` decision, `Runner`
  interface with an exec-based implementation, the run/return loop and the
  line-based fallback.
- `cmd/menu.go`: `menu` command and root `RunE` (bare `iics`).
- Tests: `ShouldOpen`, production detection, session remaining time, menu
  model (navigation, filter, sections, selection, profile switch action),
  loop with a fake runner (production confirmation, status, return).
- Docs: `docs/documentation/menu.md`, README quick start and commands table,
  link from `interactive.md`, `profile.md` production flag;
  `make completions`.

## Acceptance Criteria

- [x] Bare `iics` on a terminal opens the menu; piped or `CI`/opt-out prints help
- [x] `iics menu` / `iics tui` open the menu
- [x] Entries run as subprocesses, show the command line, pause with status,
      and return to the menu
- [x] Ctrl+C cancels only the running command
- [x] Profile switch (session) and set-default work; header shows session info
- [x] Production profiles are marked and write entries ask for confirmation
- [x] First run without profiles starts `profile add`
- [x] Accessible / small-terminal line menu works
- [x] `go build`, `go vet`, `golangci-lint`, `go test ./...` pass

## Implementation Notes

- `internal/menu/entries.go` - curated `DefaultEntries`, `CommandArgs`,
  `CommandLine` (shell-quoted display).
- `internal/menu/open.go` - `ShouldOpen` (TTY on stdin and stdout, `CI`,
  `IICS_NO_MENU`, `ui.menu`).
- `internal/menu/model.go` - full-screen menu (header, sections, filter,
  disabled entries without a profile, key help, equivalent command).
- `internal/menu/loop.go` - `Run` loop with `Deps` (runner, prompter, state
  loader, set-default, show-menu, pause, clear-screen) so it is testable;
  `ExecRunner` ignores SIGINT in the menu while the child runs; line-based
  menu for accessible mode and small terminals.
- `cmd/menu.go` - `menu` / `tui` command, root `RunE`, state from config and
  session cache, global flag pass-through (all explicitly set persistent
  flags except `--profile` and `--output`).
- `internal/config` - `Profile.Production`, `UIConfig.Menu`,
  `IsProductionProfile`, `SessionEntry.Remaining`, production question in the
  profile prompt.
- Verified on `dev` through a pty: menu opens on bare `iics`, filter, run
  with command line / status / pause / return to the last entry, production
  header and declined confirmation on `prd`, profile picker, Ctrl+C in a
  running command returns to the menu, accessible line menu. Non-TTY `iics`
  still prints help; unknown commands still fail.
