# profile

Manage IICS connection profiles stored in `~/.iics/config.yaml`.

Profiles hold the credentials and region needed to connect to an IICS org. The interactive
`add` subcommand prompts for all required values and saves them, eliminating the need to edit
the config file manually.

## Synopsis

```bash
iics profile <subcommand> [flags]
```

## Subcommands

| Subcommand    | Description                                    |
| ------------- | ---------------------------------------------- |
| `add`         | Add or update a profile interactively          |
| `edit`        | Edit an existing profile interactively         |
| `list`        | List all configured profiles                   |
| `delete`      | Delete a profile                               |
| `set-default` | Set the default profile used when no `--profile` flag is given |
| `show`        | Show the details of a single profile           |

## Flags

### `add [name]`

No command-specific flags. `name` defaults to `"default"` when omitted.

### `edit [name]`

No command-specific flags. `name` defaults to `"default"` when omitted.

### `delete <name>`

| Flag       | Short | Description              |
| ---------- | ----- | ------------------------ |
| `--yes`    | `-y`  | Skip confirmation prompt |

All other subcommands accept only the [global flags](../../README.md#global-flags).

## Description

Profiles are stored under the `profiles` key in `~/.iics/config.yaml`. Each profile
specifies a `username`, `password`, and either a `region` code (resolved to a login URL via
the built-in POD registry) or an explicit `loginUrl`. The `baseApiUrl` and `caiUrl` fields
are populated automatically after the first successful `iics login`.

The optional top-level `style` key controls table output appearance and is shared across
all profiles:

```yaml
style:
  theme: default     # default | minimal | compact | plain | markdown | gh
  noColor: false     # true = disable color permanently (same as --no-color flag)
  headerColor: ""    # lipgloss color: "6"=cyan, "244"=gray, "#FF0000"=hex
```

The `add` subcommand:

- Prompts for `username`, `password` (input is masked), and `region` or custom login URL.
- Derives `loginUrl` from the region automatically (if a known region code is entered).
- Derives `caiUrl` from the login URL and shows it as the default for the CAI URL prompt;
  the user can accept the derived value or type a custom URL.
- When invoked on an existing profile name, shows the current value in brackets so you can
  press Enter to keep it.
- Asks whether to set the profile as the default.
- Presents a live preview of all available table themes and prompts you to select one.
  The selected theme is saved to the global `style.theme` in `~/.iics/config.yaml`.
  Press Enter to keep the current theme.
- Saves the result to `~/.iics/config.yaml`.
- Does not validate credentials. Run `iics login --profile <name>` afterwards to verify and
  populate `baseApiUrl` and `caiUrl`.

The `edit` subcommand:

- Requires the profile to already exist (use `profile add` to create new profiles).
- Behaves identically to `add` for prompting - shows current values as defaults.
- After the prompts, **validates credentials by logging in** immediately. If the login fails
  the profile is not saved and the error is shown.
- On success: saves the updated profile with org-specific `baseApiUrl` and `caiUrl` derived
  from the login response, and refreshes the session cache. You do not need to run
  `iics login` separately.
- Presents the same live theme preview and selection menu as `add`.

On first `iics login` after creating a profile with `add`, the `baseApiUrl` (org-specific,
not known before a real login) and any remaining derived URLs are written back to the profile
automatically.

The `delete` subcommand removes the profile from the config file and also clears any cached
session for that profile from `~/.iics/sessions.yaml`.

### Auto-trigger on missing credentials

When any command that requires authentication (e.g. `iics connection list`) is run and no
credentials are found for the active profile, the CLI automatically launches the interactive
setup wizard - provided stdin is a terminal. After the wizard completes and the profile is
saved, the original command continues normally.

This behaviour does not trigger in non-interactive environments (CI/CD pipelines, cron jobs,
or piped input). In those cases the command fails with an error and you should supply
credentials via environment variables (`IICS_USERNAME`, `IICS_PASSWORD`, `IICS_REGION`) or
a pre-configured profile.

## Examples

```bash
# First-time setup - create the default profile interactively
iics profile add

# Create a named profile for a production org
iics profile add prod

# Update credentials for an existing profile (validates by logging in)
iics profile edit qa

# Update the default profile
iics profile edit

# List all profiles (shows which one is the default)
iics profile list

# Show full details of a specific profile (password is masked)
iics profile show prod

# Switch the default profile
iics profile set-default prod

# Delete a profile (prompts for confirmation)
iics profile delete staging

# Delete without confirmation
iics profile delete staging --yes

# Use a specific profile for a single command (without changing the default)
iics --profile prod connection list
```

```powershell
# First-time setup - create the default profile interactively
iics profile add

# Create a named profile for a production org
iics profile add prod

# Update credentials for an existing profile (validates by logging in)
iics profile edit qa

# Update the default profile
iics profile edit

# List all profiles (shows which one is the default)
iics profile list

# Show full details of a specific profile (password is masked)
iics profile show prod

# Switch the default profile
iics profile set-default prod

# Delete a profile (prompts for confirmation)
iics profile delete staging

# Delete without confirmation
iics profile delete staging --yes

# Use a specific profile for a single command (without changing the default)
iics --profile prod connection list
```

## `profile list` output

`profile list` renders a table with one row per profile. The REGION column shows the
configured region code; the ENDPOINT column shows the discovered login URL (populated after
the first `iics login`).

```text
+--------+---------+--------+---------------------------------------------------+-------------------+
| NAME   | DEFAULT | REGION | ENDPOINT                                          | USERNAME          |
+--------+---------+--------+---------------------------------------------------+-------------------+
| prod   | yes     | USE4   | https://use4.dm-us.informaticacloud.com/saas/...  | admin@company.com |
| qa     |         | EU1    | https://dm-em.informaticacloud.com/saas/...       | qa@company.com    |
+--------+---------+--------+---------------------------------------------------+-------------------+
```

## `profile show` output

`profile show` renders a vertical FIELD/VALUE table including both config-file fields and
session-derived fields read from the local session cache. Session fields show
`(no active session)` when the profile has never been used or after `iics logout`.

```text
+----------------+----------------------------------------------------------+
| FIELD          | VALUE                                                    |
+----------------+----------------------------------------------------------+
| Name           | prod                                                     |
| Default        | yes                                                      |
| Region         | USE4                                                     |
| Login URL      | https://use4.dm-us.informaticacloud.com/saas/...         |
| Base API URL   | https://use4.dm-us.informaticacloud.com/saas             |
| CAI URL        | https://use4-cai.dm-us.informaticacloud.com              |
| Username       | admin@company.com                                        |
| Password       | ***                                                      |
| Org Name       | My Production Org                                        |
| Org ID         | a1B2c3D4E5F6                                             |
| Session User   | admin@company.com                                        |
| Last Login     | 2026-03-23 14:05:00 UTC                                  |
| Session Expires| 2026-03-23 14:35:00 UTC                                  |
+----------------+----------------------------------------------------------+
```

`Base API URL`, `CAI URL`, and session fields are empty until after the first `iics login`.

## See also

- [login](login.md) - Authenticate and cache a session
- [logout](logout.md) - Invalidate a cached session
- [Configuration reference](../../README.md#configuration)
