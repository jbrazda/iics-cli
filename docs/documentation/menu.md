# menu

Interactive main menu. Alias: `tui`.

Run `iics` without arguments in a terminal (or `iics menu`) to browse the
interactive commands, pick one, and return to the menu when it finishes. Every
menu entry runs a real command line, which is shown before it runs, so the
menu also teaches the scriptable form.

## Synopsis

```bash
iics            # opens the menu in a terminal, otherwise prints help
iics menu       # always opens the menu (terminal required)
iics tui
iics -p prd     # opens the menu on profile prd
```

## When the menu opens

`iics` without arguments opens the menu only when **stdin and stdout are both
terminals**. Otherwise it prints the help text, as before. Scripts, pipes
(`iics | less`) and CI are not affected.

| Setting                         | Effect on bare `iics`                         |
| ------------------------------- | --------------------------------------------- |
| `IICS_NO_MENU=1`                | Print help instead of opening the menu        |
| `ui.menu: false` in the config  | Print help instead of opening the menu        |
| `CI` set (any value)            | Print help instead of opening the menu        |

`iics --help` and `iics help` always print help. `iics menu` opens the menu
regardless of these settings, but still requires a terminal.

```yaml
# ~/.iics/config.yaml
ui:
  menu: false
```

## Screen

```text
iics  main menu
 PRODUCTION  profile prd · org Acme Prod · admin@acme.com · session ok (12 min left)

Session
> Switch profile
  Set as default profile ✎
  Login
Users
  List users
  Create user ✎
  ...

Use another org profile for this menu session (key p)
enter run · / filter · p profile · ? help · q quit
```

- The header shows the active profile, and the org, user and session time
  from the local session cache (no network call). Without a valid session it
  shows "not logged in - signs in on first command".
- Entries are grouped by resource. `✎` marks entries that change data.
- The bottom lines describe the focused entry and show its command line.

## Keys

| Key                 | Action                                        |
| ------------------- | --------------------------------------------- |
| `↑` / `↓`, `k` / `j` | Move                                         |
| `PgUp` / `PgDn`, `g` / `G` | Page, first / last entry               |
| `Enter`             | Run the focused entry                         |
| `/`                 | Filter by group, label, description or command |
| `Esc`               | Clear the filter                              |
| `p`                 | Switch profile                                |
| `?`                 | Show or hide key help                         |
| `q`, `Ctrl+C`       | Quit the menu                                 |

## Running a command

1. The command line is printed, for example `$ iics role edit --profile dev`.
2. The command runs as its own process in the terminal, exactly as if you had
   typed it. Global flags the menu was started with (for example `--config`,
   `--theme`, `--no-color`) are passed on.
3. When it finishes, `✓ done` or `✗ failed (exit N)` is shown. Press `Enter`
   to return to the menu (on the last used entry) or `q` to quit.

`Ctrl+C` while a command runs cancels only that command; the menu stays open.

## Profiles

- The menu starts with the usual profile: `--profile`, then `IICS_PROFILE`,
  then `defaultProfile` from the config.
- **Switch profile** (`p`) changes the profile for this menu session only.
- **Set as default profile** writes `defaultProfile` to the config file.
- With no profiles configured, the menu starts `profile add` first. If that is
  canceled, only the Session entries are available.

## Production profiles

A profile is treated as production when its config has `production: true`.
Without that setting, profile names `prd`, `prod`, `production`, or names
starting with `prd-` / `prod-` count as production; `production: false` turns
this off. `iics profile add` and `iics profile edit` ask for the setting.

On a production profile:

- the header shows a red `PRODUCTION` marker,
- entries marked `✎` ask for confirmation (default **No**) before they run.

## Entries

| Group                   | Entries                                                                 |
| ----------------------- | ----------------------------------------------------------------------- |
| Session                 | Switch profile, Set as default profile ✎, Login, List profiles, Add profile ✎, Edit active profile ✎ |
| Users                   | List users, Create user ✎, Edit user groups and roles ✎, Delete user ✎, Change password ✎ |
| User groups             | List, Show, Create ✎, Edit roles ✎, Delete ✎                             |
| Roles                   | List, Show (with privileges), Create ✎, Edit privileges ✎               |
| Privileges              | List privileges                                                         |
| Permissions             | Edit object permissions ✎                                               |
| Environments and agents | List runtime environments, Create runtime environment ✎, List agents, Start / Stop / Restart agent service ✎, Download agent installer |
| Help                    | Show CLI help                                                           |

## Accessible mode and small terminals

With `IICS_ACCESSIBLE=1`, or when the terminal is smaller than 60 columns or
15 rows, the menu is a numbered list with the same entries and the same
run-and-return behavior. See [Interactive prompts](interactive.md).

## See also

- [Interactive prompts](interactive.md) - keys and accessible mode for wizards
- [profile](profile.md) - manage profiles and the production flag
