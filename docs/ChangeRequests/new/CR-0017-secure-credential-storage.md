# CR-0017: Secure Credential Storage via OS Keychain

## Type

- [ ] New resource
- [ ] New subcommand
- [x] Enhancement - change behaviour of an existing command
- [x] Flag / config change - add/rename/remove a CLI flag or config field

---

## Problem

Passwords are stored in plaintext in `~/.iics/config.yaml`. Session tokens are stored in
plaintext in `~/.iics/sessions.yaml`. Anyone with read access to the user's home directory
can read both files. This violates the principle of least privilege and creates risk in
shared machines, laptop theft, and accidental credential leaks (e.g. config file committed
to source control).

The CLI already supports `IICS_PASSWORD` as an env-var override, which covers CI/CD, but
there is no interactive path to store a password securely for day-to-day developer use.

---

## Desired Change

Integrate OS-native secure credential storage using the sentinel value pattern so that:

1. A password stored in the OS keychain is represented in `config.yaml` as `password: "@keyring"`.
2. `ResolveProfile()` detects the sentinel and fetches the real password from the OS keychain
   at runtime - it never appears on disk.
3. `profile add` and `profile edit` ask "Store password in OS keychain? [Y/n]" and handle
   both paths (keyring store vs. plaintext in config file).
4. A new `profile set-password` subcommand lets users migrate an existing plaintext password
   to the keychain without re-running the full profile setup.
5. Graceful fallback: if the keychain is unavailable (headless server, container, CI),
   `IICS_PASSWORD` env var and then plaintext config are tried in order, with a clear warning
   that keyring access failed.

---

## New Dependency

**`github.com/zalando/go-keyring`** (1 new direct dependency).

| Platform | Backend used | CGO required |
| -------- | ------------ | ------------ |
| macOS | macOS Keychain (via `security` CLI tool) | No |
| Windows | Windows Credential Manager (WinAPI) | Yes (syscall wrapper) |
| Linux | D-Bus Secret Service (GNOME Keyring / KDE Wallet) | No |

The library is pure-Go on macOS and Linux; it uses a thin CGO syscall shim on Windows.
It is MIT-licensed, actively maintained, and has no transitive dependencies beyond
`golang.org/x/sys` on Windows (already likely present as a transitive dependency).

**Linux headless / CI caveat:** Secret Service requires a running D-Bus session, which is
not present in Docker containers, SSH sessions without a session bus, or minimal CI
images. In those environments the keyring lookup returns an error and the fallback chain
takes over. No action is needed in CI - `IICS_PASSWORD` env var continues to work.

---

## Sentinel Value

The config file uses a reserved string as a pointer to the keychain:

```yaml
profiles:
  prod:
    username: admin@company.com
    password: "@keyring"     # real password is in OS keychain, not here
```

The sentinel `"@keyring"` is chosen because it is not a valid IICS password (IICS passwords
do not begin with `@`). It is case-sensitive. Any other value is treated as a literal
password as before, preserving full backward compatibility.

---

## Keyring Key Scheme

Keychain entries use two fields:

| Keyring field | Value |
| ------------- | ----- |
| Service | `iics-cli` |
| User / Account | `<profileName>` (e.g. `prod`, `dev`, `default`) |

This maps cleanly to the profile name used throughout the CLI. Multiple profiles are
stored as separate keychain entries under the same service name.

---

## Password Resolution Order in `ResolveProfile()`

Highest to lowest precedence (no change to existing env-var override behaviour):

| Rank | Source | Condition |
| ---- | ------ | --------- |
| 1 | `IICS_PASSWORD` env var | Always checked first |
| 2 | OS keychain | Only when `password == "@keyring"` in config |
| 3 | Plaintext in config file | `password` is set to a non-sentinel value |
| 4 | Error | Password still empty after all sources |

If keychain lookup is attempted but fails (keyring unavailable), a warning is printed to
stderr and the error is returned - plaintext fallback is NOT automatic, to avoid silently
using stale plaintext credentials after a user has opted into keyring storage.

---

## Scope

### New file

```text
internal/config/keyring.go     - GetKeychainPassword / SetKeychainPassword / DeleteKeychainPassword
                                  IsKeyringSentinel() constant and helper
```

### Files to modify

```text
internal/config/config.go      - ResolveProfile(): detect sentinel, call GetKeychainPassword
internal/config/prompt.go      - promptProfileInternal(): offer keyring storage after password prompt
cmd/profile.go                 - profile add/edit: save to keyring or config depending on user choice
                                  new profile set-password subcommand
                                  profile delete: also delete keychain entry if present
docs/documentation/profile.md  - document keyring storage, sentinel, fallback behaviour
README.md                      - update config example, env-var table, security note
go.mod / go.sum                - add github.com/zalando/go-keyring
```

### Do NOT touch

```text
internal/client/    - no API changes
cmd/*.go            - all files except profile.go
```

---

## Detailed Design

### `internal/config/keyring.go`

```go
package config

import (
    "fmt"

    "github.com/zalando/go-keyring"
)

const (
    // KeyringService is the service name used for all iics-cli keychain entries.
    KeyringService = "iics-cli"

    // KeyringSentinel is the placeholder stored in config.yaml when the real
    // password lives in the OS keychain.
    KeyringSentinel = "@keyring"
)

// IsKeyringSentinel reports whether the password field value indicates that
// the real credential should be fetched from the OS keychain.
func IsKeyringSentinel(password string) bool {
    return password == KeyringSentinel
}

// GetKeychainPassword retrieves the password for the given profile name from the
// OS keychain. Returns an error if the keychain is unavailable or the entry is not found.
func GetKeychainPassword(profileName string) (string, error) {
    pw, err := keyring.Get(KeyringService, profileName)
    if err != nil {
        return "", fmt.Errorf("keyring lookup for profile %q: %w", profileName, err)
    }
    return pw, nil
}

// SetKeychainPassword stores a password in the OS keychain for the given profile name.
// Overwrites any existing entry for the same profile.
func SetKeychainPassword(profileName, password string) error {
    if err := keyring.Set(KeyringService, profileName, password); err != nil {
        return fmt.Errorf("keyring store for profile %q: %w", profileName, err)
    }
    return nil
}

// DeleteKeychainPassword removes the keychain entry for the given profile name.
// Returns nil if the entry does not exist.
func DeleteKeychainPassword(profileName string) error {
    err := keyring.Delete(KeyringService, profileName)
    if err != nil && err != keyring.ErrNotFound {
        return fmt.Errorf("keyring delete for profile %q: %w", profileName, err)
    }
    return nil
}
```

---

### `internal/config/config.go` - `ResolveProfile()` change

After the `IICS_PASSWORD` env-var override block, add keychain resolution:

```go
// Keychain resolution: if password is the sentinel, fetch from OS keychain.
if IsKeyringSentinel(resolved.Password) {
    pw, err := GetKeychainPassword(profileName)
    if err != nil {
        return nil, fmt.Errorf(
            "profile %q uses keychain storage but keychain lookup failed: %w\n"+
                "  Set IICS_PASSWORD env var to override, or run 'iics profile set-password %s' to re-store",
            profileName, err, profileName)
    }
    resolved.Password = pw
}
```

The `IICS_PASSWORD` env-var check runs before this block (existing behaviour), so setting
`IICS_PASSWORD` always bypasses both the sentinel and the plaintext config value.

---

### `internal/config/prompt.go` - keyring offer in `promptProfileInternal()`

After the password is collected, ask whether to store it in the keychain. Return both the
password and a `storeInKeyring bool` flag to the caller.

Change `PromptProfile` return signature:

```go
// before
func PromptProfile(existing *Profile, profileName string) (*Profile, bool, error)

// after
func PromptProfile(existing *Profile, profileName string) (*Profile, bool, bool, error)
// returns: profile, makeDefault, storeInKeyring, error
```

In `promptProfileInternal`, after the password loop:

```go
// Offer keychain storage (skip if stdin is not a terminal or keyring is not available).
storeInKeyring := false
existingIsKeyring := existing != nil && IsKeyringSentinel(existing.Password)
if existingIsKeyring {
    _, _ = fmt.Fprint(os.Stderr, "Store password in OS keychain? [Y/n]: ")
} else {
    _, _ = fmt.Fprint(os.Stderr, "Store password in OS keychain (recommended)? [Y/n]: ")
}
line, err = r.ReadString('\n')
if err != nil {
    return nil, false, false, fmt.Errorf("reading keyring choice: %w", err)
}
line = strings.TrimSpace(strings.ToLower(line))
storeInKeyring = line == "" || line == "y" || line == "yes"
```

Return `storeInKeyring` as the third return value. The profile's `Password` field is
always set to the actual password at this point; callers decide whether to save to keychain
and write the sentinel.

---

### `cmd/profile.go` - callers of `PromptProfile`

All three call sites that call `config.PromptProfile` must be updated to handle the new
return value.

**`newProfileAddCmd` RunE:**

```go
p, makeDefault, storeInKeyring, err := config.PromptProfile(existing, name)
if err != nil {
    return err
}

plainPassword := p.Password

if storeInKeyring {
    if keyErr := config.SetKeychainPassword(name, plainPassword); keyErr != nil {
        _, _ = fmt.Fprintf(cmd.ErrOrStderr(),
            "Warning: could not store password in keychain: %v\n"+
                "  Storing password in config file instead.\n", keyErr)
    } else {
        p.Password = config.KeyringSentinel
    }
}
// ... rest of save logic
```

**`newProfileEditCmd` RunE:** same pattern.

**Auto-setup wizard in `cmd/root.go` `getClient()`:** the inline `config.PromptProfile`
call there must also handle the new third return value; apply the same keyring-or-plaintext
logic.

**`newProfileDeleteCmd` RunE:** after deleting the profile from `cfg.Profiles`, also clean
up any keychain entry:

```go
if err := config.DeleteKeychainPassword(name); err != nil {
    _, _ = fmt.Fprintf(cmd.ErrOrStderr(),
        "Warning: could not remove keychain entry: %v\n", err)
}
```

---

### New subcommand: `profile set-password`

Lets users migrate an existing plaintext password to the keychain, or update a stored
keychain password, without re-running the full profile setup.

```
iics profile set-password [name]
```

Behaviour:

1. Load config; resolve profile name (default `"default"`).
2. Prompt for new password (masked input via `term.ReadPassword`).
3. Call `config.SetKeychainPassword(name, password)`.
4. If successful: update `cfg.Profiles[name].Password = config.KeyringSentinel` and save config.
5. If keychain unavailable: print error and exit without modifying config.

```go
func newProfileSetPasswordCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "set-password [name]",
        Short: "Store the profile password in the OS keychain",
        Long: `Prompts for a password and stores it in the OS keychain (macOS Keychain,
Windows Credential Manager, or Linux Secret Service). The config file is
updated to use the "@keyring" sentinel so the plaintext password is no
longer stored on disk.`,
        Args: cobra.MaximumNArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            name := "default"
            if len(args) == 1 {
                name = args[0]
            }
            cfg, err := loadConfig()
            if err != nil {
                return err
            }
            if cfg.Profiles[name] == nil {
                return fmt.Errorf("profile %q not found", name)
            }

            _, _ = fmt.Fprintf(cmd.OutOrStdout(),
                "Enter new password for profile %q (input is masked): ", name)
            pw, err := term.ReadPassword(int(os.Stdin.Fd()))
            fmt.Println()
            if err != nil {
                return fmt.Errorf("reading password: %w", err)
            }
            if len(pw) == 0 {
                return fmt.Errorf("password must not be empty")
            }

            if err := config.SetKeychainPassword(name, string(pw)); err != nil {
                return fmt.Errorf("keychain store failed: %w\n"+
                    "  Use 'iics profile edit' to update the password in the config file instead", err)
            }

            cfg.Profiles[name].Password = config.KeyringSentinel
            if err := cfg.Save(cfgFile); err != nil {
                return fmt.Errorf("saving config: %w", err)
            }

            _, _ = fmt.Fprintf(cmd.OutOrStdout(),
                "Password for profile %q stored in OS keychain.\n", name)
            return nil
        },
    }
}
```

Register in `newProfileCmd()`:

```go
cmd.AddCommand(newProfileSetPasswordCmd())
```

---

## Config File After CR

Profile using keychain:

```yaml
profiles:
  prod:
    username: admin@company.com
    password: "@keyring"     # stored in OS keychain under service=iics-cli, account=prod
    region: EMEA
```

Profile using plaintext (unchanged behavior):

```yaml
profiles:
  dev:
    username: dev@company.com
    password: "my-plaintext-password"
    region: USW3
```

---

## `go.mod` Change

```
require (
    ...
    github.com/zalando/go-keyring v0.2.x
)
```

Run `go get github.com/zalando/go-keyring` to fetch and update `go.sum`.

---

## Documentation Updates

### `README.md`

1. Add a **Security** section describing keychain storage, the sentinel, and the fallback
   chain.
2. Add `profile set-password` to the profile commands table.
3. Update the config example to show both the sentinel and plaintext variants.

### `docs/documentation/profile.md`

1. Document the keychain offer in `profile add` and `profile edit`.
2. Document the new `profile set-password` subcommand with flags table and examples.
3. Add a **Credential security** section explaining:
   - What `@keyring` means in the config file
   - How to migrate an existing plaintext config: `iics profile set-password <name>`
   - Fallback behaviour when keychain is unavailable
   - CI/CD recommendation: use `IICS_PASSWORD` env var

---

## Testing

Unit tests live in `internal/config/` (same package, `package config`).

The `go-keyring` library exposes `keyring.MockInit()` which replaces the real OS backend
with an in-memory store for tests. Use it in `TestGetKeychainPassword`,
`TestSetKeychainPassword`, `TestDeleteKeychainPassword`.

For `ResolveProfile` tests, add cases:

| Test | Setup | Expected |
| ---- | ----- | -------- |
| `TestResolveProfileKeyring` | password=`@keyring`, mock keyring set | returns real password |
| `TestResolveProfileKeyringMissing` | password=`@keyring`, mock keyring empty | returns error with hint |
| `TestResolveProfileKeyringEnvOverride` | password=`@keyring`, `IICS_PASSWORD` set | env var wins, no keyring call |
| `TestResolveProfilePlaintext` | password=`secret` | unchanged behavior |

---

## Acceptance Criteria

- [ ] `profile add` asks "Store password in OS keychain?" and writes `@keyring` sentinel
      when the user answers yes
- [ ] `profile add` stores plaintext in config when the user answers no (unchanged behavior)
- [ ] `profile edit` same behavior as `profile add` for keyring offer
- [ ] `profile set-password` migrates a plaintext config password to the keychain and
      updates the sentinel in config
- [ ] `profile delete` removes the keychain entry for the profile (no error if none exists)
- [ ] `iics login` works transparently with a sentinel profile on macOS, Windows, and Linux desktop
- [ ] When keychain lookup fails (keyring unavailable), a clear error and hint are shown;
      `IICS_PASSWORD` env var is documented as the override
- [ ] `IICS_PASSWORD` env var overrides both sentinel and plaintext (existing behavior preserved)
- [ ] Config files with plaintext passwords continue to work unchanged (backward compatibility)
- [ ] `go build ./...` passes on macOS, Windows, Linux
- [ ] `go test ./...` passes with mocked keyring
- [ ] `go vet ./...` reports no issues
- [ ] No other commands are modified

---

## Do NOT

- Store session tokens (`~/.iics/sessions.yaml`) in the keychain - they are short-lived
  (30 min TTL) and low-risk; keychain lookups add latency to every command invocation
- Add a `--no-keyring` flag - use `IICS_PASSWORD` env var for CI instead
- Silently fall back from keyring to plaintext when the keychain lookup fails - this would
  mask keyring misconfiguration and could expose stale plaintext credentials
- Use `github.com/99designs/keyring` - adds 8-12 transitive dependencies for no benefit
  over `go-keyring` given that CI/headless environments are covered by `IICS_PASSWORD`
- Add per-field keychain storage for tokens or session IDs - password only
- Modify `internal/client/` - no API changes required
