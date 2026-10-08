# CR-0046: Profile wizard and global settings editor

## Status

Completed.

## Summary

Replace the line-by-line profile prompts with a paged TUI form used by every
profile entry point, add global new-user defaults with per-profile
overrides, and add `iics config edit` to edit global settings.

## Motivation

- `profile add` / `profile edit` ask one line at a time, show the region list
  as a long text line and cannot go back to a previous answer.
- The new user patterns (CR-0045) can only be set per profile by editing
  `~/.iics/config.yaml`. The same patterns usually apply to every org, so
  each profile needs a copy.
- Global settings (theme, colors, responsive tables, HTTP timeout, main menu)
  are only partly editable: the theme is asked at the end of `profile add` /
  `profile edit`, the rest only in the YAML file.

## Desired Behaviour

### Global new-user defaults

A top-level `newUser` block holds the global defaults. A profile `newUser`
block overrides them field by field.

```yaml
newUser:
  domain: acme.com
  userNamePattern: "{firstName}.{lastName}.{profileName}@{domain}"
  emailPattern: "{firstName}.{lastName}@{domain}"
profiles:
  dev:
    username: admin@acme.com
    newUser:
      emailPattern: "{firstInitial}{lastName}@{domain}"   # override for dev only
```

Resolution per field: profile `newUser` -> global `newUser` -> built-in
default. `domain` falls back to the domain of the profile username when
neither block sets it.

### Profile wizard

`profile add`, `profile edit`, `login` (profile missing) and first-run setup
use one form:

1. **Connection** - user name, password (masked; empty keeps the current
   password), region (filterable list) or custom login URL, CAI URL
   (optional).
2. **Options** - production org, set as default profile, store password in
   the OS keychain (not asked when an unchanged keychain password is kept).
3. **New user** - domain, user name pattern and email pattern overrides for
   this profile. Empty inherits; each field shows the inherited value and
   its source. Patterns are validated (unknown placeholders are rejected).
4. **Review** - summary (new profile) or changes (existing profile) with
   **Save**, **Back to editing** or **Cancel**.

The theme is no longer asked by the profile commands; it moves to
`iics config edit`. `profile edit` still validates the credentials by
logging in after the form. In accessible mode the form runs as huh
accessible prompts.

### `iics config edit`

Interactive form for global settings, followed by a review of the changes:

- New user defaults: domain, user name pattern, email pattern (empty uses
  the built-in default).
- Appearance: table theme, header color, no color, responsive tables.
- HTTP timeout in seconds (empty uses the built-in default).
- Main menu for bare `iics`.

### Other

- `profile show` lists the effective new-user values with their source.
- The main menu gets a **Session > Edit global settings** entry.
- `Config.Save` writes `httpTimeout`, `ui` and `newUser`. It previously wrote
  only `defaultProfile`, `profiles` and `style`, so saving a profile dropped
  `httpTimeout` and `ui.menu` from the config file.

## Files to Modify

- `internal/config/config.go`, `internal/config/userpattern.go` - global
  `newUser`, merge, pattern validation, region list
- `internal/config/prompt.go` - remove the line-based profile prompts
- `internal/tui/profilewizard.go`, `internal/tui/configwizard.go` - new forms
- `cmd/profile.go`, `cmd/login.go`, `cmd/root.go` - use the wizard
- `cmd/config.go` - new `config edit` command
- `internal/menu/entries.go` - menu entry
- `docs/documentation/profile.md`, `docs/documentation/config.md`,
  `docs/documentation/user.md`, `README.md`, `completions/`
