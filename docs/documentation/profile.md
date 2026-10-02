# profile

Manage IICS connection profiles stored in `~/.iics/config.yaml`.

Profiles hold the credentials and region needed to connect to an IICS org. The interactive
`add` and `edit` subcommands collect all values in a form and save them, eliminating the need
to edit the config file manually. Global settings shared by all profiles are edited with
[config edit](config.md).

## Synopsis

```bash
iics profile <subcommand> [flags]
```

## Subcommands

| Subcommand      | Description                                    |
| --------------- | ---------------------------------------------- |
| `add`           | Add or update a profile interactively          |
| `edit`          | Edit an existing profile interactively         |
| `list`          | List all configured profiles                   |
| `delete`        | Delete a profile                               |
| `set-default`   | Set the default profile used when no `--profile` flag is given |
| `set-password`  | Store a profile's password in the OS keychain  |
| `show`          | Show the details of a single profile           |

## Flags

### `add [name]`

No command-specific flags. `name` defaults to `"default"` when omitted.

### `edit [name]`

No command-specific flags. `name` defaults to `"default"` when omitted.

### `delete <name>`

| Flag       | Short | Description              |
| ---------- | ----- | ------------------------ |
| `--yes`    | `-y`  | Skip confirmation prompt |

### `set-password <name>`

No command-specific flags. `name` is required.

All other subcommands accept only the [global flags](../../README.md#global-flags).

## Description

Profiles are stored under the `profiles` key in `~/.iics/config.yaml`. Each profile
specifies a `username`, `password`, and either a `region` code (resolved to a login URL via
the built-in POD registry) or an explicit `loginUrl`. The `baseApiUrl` and `caiUrl` fields
are populated automatically after the first successful `iics login`.

The optional `production` field marks a production org. The interactive
[main menu](menu.md) then shows a red `PRODUCTION` marker and asks for
confirmation before running entries that change data. `profile add` and
`profile edit` ask "Production org?"; the default is the current setting, or
yes for profile names `prd`, `prod`, `production` and names starting with
`prd-` / `prod-`.

```yaml
profiles:
  prd:
    region: "USE4"
    username: "admin@company.com"
    password: "@keyring"
    production: true
```

The optional `newUser` block overrides, for this profile, the user name and
email suggested by `iics user create --interactive`. Each field overrides the
global `newUser` default of the same name (set with [config edit](config.md));
see [user - New user patterns](user.md#new-user-patterns):

```yaml
newUser:                       # global defaults (all profiles)
  domain: company.com
profiles:
  dev:
    newUser:                   # overrides for dev only
      emailPattern: "{firstInitial}{lastName}@{domain}"
```

`profile show` lists `Production`, `New User Domain`, `User Name Pattern` and
`Email Pattern` with the effective value and its source (`profile`, `global`,
`profile username` or `built-in default`).

Table appearance (`style`), the HTTP timeout and the main menu setting are
global; see [config edit](config.md).

### Profile form

`add`, `edit`, `login` (when the profile does not exist) and the automatic
setup on missing credentials use the same paged form (`Enter` next field,
`Shift+Tab` back). Stored values are filled in.

1. **Connection** - user name, password (masked; leave empty to keep the
   current password) and region. The region list filters as you type;
   **Custom login URL** asks for a login URL on the next page instead.
   `loginUrl` is derived from the region.
2. **Options** - CAI URL (optional; derived from the org URL when known),
   production org, and whether to set the profile as the default.
3. **Keychain** - **store the password in the OS keychain** (default: yes).
   When accepted, `password: "@keyring"` is written to the config file and the
   real password goes to the native OS keychain (macOS Keychain, Windows
   Credential Manager, or Linux D-Bus Secret Service). Skipped when an
   existing keychain password is kept.
4. **New user** - domain, user name pattern and email pattern overrides for
   this profile. Leave a field empty to inherit; each field shows the
   inherited value and its source. Unknown placeholders are rejected.
5. **Review** - the new profile, or the changes to an existing one, with
   **Save**, **Back to editing** or **Cancel**.

In accessible mode (`IICS_ACCESSIBLE=1`) the form runs as numbered line
prompts.

The `add` subcommand saves the profile without validating the credentials.
Run `iics login --profile <name>` afterwards to verify them and populate
`baseApiUrl` and `caiUrl`. When the name already exists, the stored values
are filled in and the profile is updated.

The `edit` subcommand:

- Requires the profile to already exist (use `profile add` to create new profiles).
- After the review, **validates credentials by logging in**. If the login fails
  the profile is not saved and the error is shown.
- On success: saves the updated profile with org-specific `baseApiUrl` and `caiUrl` derived
  from the login response, and refreshes the session cache. You do not need to run
  `iics login` separately.

On first `iics login` after creating a profile with `add`, the `baseApiUrl` (org-specific,
not known before a real login) and any remaining derived URLs are written back to the profile
automatically.

The `delete` subcommand removes the profile from the config file, clears any cached session
for that profile from `~/.iics/sessions.yaml`, and also removes any stored keychain entry for
that profile.

The `set-password` subcommand stores a password directly into the OS keychain for a profile
that already exists in the config file. It:

- Prompts for the new password (masked input).
- Stores it in the OS keychain under the profile name.
- Writes `password: "@keyring"` to the profile in `~/.iics/config.yaml`.

Use this to migrate a profile that has a plaintext password in the config file to keychain
storage without going through the full `edit` flow.

### Credential security

Passwords stored in `~/.iics/config.yaml` can be kept out of the config file using OS-native
secure storage. When a profile uses keychain storage, the config file contains:

```yaml
profiles:
  prod:
    username: "admin@company.com"
    password: "@keyring"
```

The real password is retrieved from the OS keychain at runtime. The `IICS_PASSWORD` environment
variable always takes precedence over both the keychain and plaintext config, making it easy to
override in CI/CD pipelines:

```bash
IICS_PASSWORD=secret iics --profile prod objects list
```

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

# Migrate a plaintext password to OS keychain storage
iics profile set-password prod

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

# Migrate a plaintext password to OS keychain storage
iics profile set-password prod

# Use a specific profile for a single command (without changing the default)
iics --profile prod connection list
```

## `profile list` output

`profile list` renders a table with one row per profile: NAME, USERNAME, ENDPOINT, POD,
DEFAULT, KEYCHAIN. The ENDPOINT column shows the discovered login URL (populated after the
first `iics login`); POD shows the configured region/pod code; KEYCHAIN shows `yes` when the
profile's password is stored in the OS keychain rather than in the config file.

```text
+--------+-------------------+---------------------------------------------------+------+---------+----------+
| NAME   | USERNAME          | ENDPOINT                                          | POD  | DEFAULT | KEYCHAIN |
+--------+-------------------+---------------------------------------------------+------+---------+----------+
| prod   | admin@company.com | https://use4.dm-us.informaticacloud.com/saas/...  | USE4 | yes     | yes      |
| qa     | qa@company.com    | https://dm-em.informaticacloud.com/saas/...       | EU1  |         | no       |
+--------+-------------------+---------------------------------------------------+------+---------+----------+
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
