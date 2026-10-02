# config

Manage the global settings in `~/.iics/config.yaml` (or the file given with
`--config`). Profile settings are managed with [profile](profile.md).

## Synopsis

```bash
iics config <subcommand> [flags]
```

## Subcommands

| Subcommand | Description                         |
| ---------- | ----------------------------------- |
| `edit`     | Edit global settings interactively  |

## config edit

Edit the global settings in an interactive form, followed by a review of the
changes (**Save**, **Back to editing** or **Cancel**). Requires a terminal;
in scripts, edit the config file directly. In accessible mode
(`IICS_ACCESSIBLE=1`) the form runs as numbered line prompts.

| Page       | Setting                 | Config key               | Empty value                                  |
| ---------- | ----------------------- | ------------------------ | -------------------------------------------- |
| New user   | Domain                  | `newUser.domain`         | Domain of each profile's user name           |
| New user   | User name pattern       | `newUser.userNamePattern` | `{firstName}.{lastName}.{profileName}@{domain}` |
| New user   | Email pattern           | `newUser.emailPattern`   | `{firstName}.{lastName}@{domain}`            |
| Appearance | Table theme             | `style.theme`            | `default` on a terminal, `markdown` when piped |
| Appearance | Header color            | `style.headerColor`      | Theme default                                |
| Appearance | Disable colors          | `style.noColor`          | Colors on                                    |
| Appearance | Responsive tables       | `style.responsiveTables` | On                                           |
| Other      | HTTP timeout (seconds)  | `httpTimeout`            | 120                                          |
| Other      | Main menu for bare `iics` | `ui.menu`              | On                                           |

The table theme list shows a sample table in the highlighted theme. Pattern
fields reject unknown placeholders; see
[user - New user patterns](user.md#new-user-patterns) for the placeholders.

### New user defaults

The `newUser` values apply to every profile. A profile's own `newUser` block
(set with [profile edit](profile.md#profile-form)) overrides them field by
field:

```yaml
newUser:
  domain: company.com
  emailPattern: "{firstInitial}{lastName}@{domain}"
profiles:
  dev:
    username: admin@company.com
    newUser:
      domain: dev.company.com      # dev only; the email pattern is inherited
```

Each value resolves as: profile `newUser`, then global `newUser`, then the
built-in default (for the domain: the domain of the profile user name).
`iics profile show <name>` lists the effective values and their sources.

### Flags

No command-specific flags. All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
iics config edit

# Edit another config file
iics config edit --config ./team-config.yaml
```

```powershell
iics config edit

iics config edit --config .\team-config.yaml
```

## See also

- [profile](profile.md)
- [user - New user patterns](user.md#new-user-patterns)
- [Main menu](menu.md)
