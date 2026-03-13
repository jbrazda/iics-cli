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
| `list`        | List all configured profiles                   |
| `delete`      | Delete a profile                               |
| `set-default` | Set the default profile used when no `--profile` flag is given |
| `show`        | Show the details of a single profile           |

## Flags

### `add [name]`

No command-specific flags. `name` defaults to `"default"` when omitted.

### `delete <name>`

| Flag       | Short | Description              |
| ---------- | ----- | ------------------------ |
| `--yes`    | `-y`  | Skip confirmation prompt |

All other subcommands accept only the [global flags](../../README.md#global-flags).

## Description

Profiles are stored under the `profiles` key in `~/.iics/config.yaml`. Each profile
specifies a `username`, `password`, and either a `region` code (resolved to a login URL via
the built-in POD registry) or an explicit `loginUrl`.

The `add` subcommand:

- Prompts for `username`, `password` (input is masked), and `region` or custom login URL.
- When editing an existing profile, shows the current value in brackets so you can press
  Enter to keep it.
- Asks whether to set the profile as the default.
- Saves the result to `~/.iics/config.yaml`.

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

## See also

- [login](login.md) - Authenticate and cache a session
- [logout](logout.md) - Invalidate a cached session
- [Configuration reference](../../README.md#configuration)
