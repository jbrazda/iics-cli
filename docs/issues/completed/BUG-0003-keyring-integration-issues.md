---
id: BUG-0003
title: Keyring integration causes authentication failures and profile corruption
status: new
priority: critical
affects: cmd/profile.go, internal/config/prompt.go, cmd/package.go
---

# BUG-0003: Keyring integration causes authentication failures and profile corruption

## Summary

CR-0017 (v0.3.0) introduced OS keychain storage for passwords. Several regressions were
found immediately after release:

1. (Critical) `profile edit` on a keyring-backed profile corrupts the keychain by storing
   the literal `"@keyring"` sentinel string as the password value, breaking all subsequent
   authentication for that profile and causing IICS account lockout.
2. `profile add/edit` does not detect when the user kept an existing sentinel password
   unchanged and still tries to store it in the keychain.
3. `profile list` and `profile show` give no indication of whether a profile's password is
   stored in the OS keychain.
4. `package dependencies --report` prints an unhelpful "profile not found" error when the
   user omits the required profile-name value (pflag silently consumes the next flag token
   as the value).

## Issue 1: Keychain corruption in `profile edit` (Critical - causes account lockout)

### Observed behavior

Running `iics profile edit dev` on a profile that already uses `password: "@keyring"`:

1. The password prompt shows `[current: ***]` - no indication of keychain storage.
2. User presses Enter to keep the existing password.
3. `PromptProfile` returns `p.Password = "@keyring"` (the sentinel string literally).
4. `storeInKeyring = true` (default answer to the re-offer prompt).
5. `cmd/profile.go` calls `config.SetKeychainPassword("dev", "@keyring")` - storing the
   **literal sentinel string** in the OS keychain instead of the real password.
6. Config file still has `password: "@keyring"`.
7. On the next command, `ResolveProfile` fetches `"@keyring"` from the keychain and uses
   it as the IICS API password.
8. IICS returns HTTP 400 (bad credentials) on every login attempt.

The same sentinel leak exists in `newProfileAddCmd` when re-adding an existing keyring profile.

### Additional sub-issue in `profile edit`

Login validation inside `newProfileEditCmd` uses `plainPassword` (which equals `"@keyring"`
after the above corruption) to create the login client, so the edit validation step itself
sends `"@keyring"` as the IICS password.

### Impact

All commands that require authentication fail for the affected profile with:
```
[WARN] dependency lookup failed profile=dev error=auto-login failed: IICS API error (HTTP 400):
```

Because `package dependencies --report` fires one login attempt per dependency object in
parallel, a profile with a corrupted keychain can trigger IICS account lockout within seconds.

### Root cause

`internal/config/prompt.go` line 84: when the user presses Enter to keep the existing
password, `p.Password = existing.Password` copies the sentinel string literally. There is
no detection of the sentinel in `cmd/profile.go` before calling `SetKeychainPassword`.

### Fix

In `cmd/profile.go`, guard `SetKeychainPassword` in both `newProfileAddCmd` and
`newProfileEditCmd`:

```go
plainPassword := p.Password
if config.IsKeyringSentinel(plainPassword) {
    // User kept the existing @keyring password - the keychain entry is already correct.
    // Do not overwrite the keychain with the sentinel string itself.
    p.Password = config.KeyringSentinel
} else if storeInKeyring {
    if keyErr := config.SetKeychainPassword(name, plainPassword); keyErr != nil {
        // warn, fall back to plaintext
    } else {
        p.Password = config.KeyringSentinel
    }
}
```

For `newProfileEditCmd`, retrieve the real password from the keychain for login validation:

```go
passwordForLogin := plainPassword
if config.IsKeyringSentinel(passwordForLogin) {
    if kp, kerr := config.GetKeychainPassword(name); kerr == nil {
        passwordForLogin = kp
    }
}
c := client.NewClient(loginURL, p.Username, passwordForLogin, ...)
```

### Recovery for affected profiles

After applying the fix, repair the corrupted keychain entry:

```bash
iics profile set-password <profile-name>
```

## Issue 2: `profile edit` re-offers keyring storage when password is unchanged

### Observed behavior

When editing a keyring profile and pressing Enter to keep the password, the prompt still
asks "Store password in OS keychain? [Y/n]:" - even though the password has not changed
and the keychain already holds the real password. The UX is confusing and, as shown in
Issue 1, dangerous (the default "yes" causes corruption).

### Fix

In `internal/config/prompt.go`:

1. Show `[keychain]` instead of `[***]` when the existing password is the sentinel, so the
   user knows they are editing a keychain-backed profile.
2. When the user kept the sentinel unchanged (`p.Password` is still `@keyring` after the
   prompt loop), skip the keyring offer entirely and return `storeInKeyring = false`:

```go
existingIsKeyring := existing != nil && IsKeyringSentinel(existing.Password)
passwordUnchanged := existingIsKeyring && IsKeyringSentinel(p.Password)
if passwordUnchanged {
    return p, makeDefault, false, nil
}
```

## Issue 3: No keychain indicator in `profile list` and `profile show`

### Observed behavior

`profile list` has no column showing whether a profile uses keychain storage. The only way
to tell is to inspect `~/.iics/config.yaml` manually.

`profile show` displays `***` for the password field regardless of whether the password is
stored in the config file or the OS keychain.

### Fix

`profile list`: add a `KEYCHAIN` column (empty string or `"yes"`).

`profile show`: change the Password row value from `***` to `*** (keychain)` when
`config.IsKeyringSentinel(p.Password)` is true.

## Issue 4: `package dependencies --report` silently consumes next flag as profile name

### Observed behavior

```
./iics package dependencies -f file.zip --report --target-profile 'dev' --verbose
[INFO] report: starting validation profile=--target-profile total=10
Error: profile "--target-profile": username not configured for target profile "--target-profile"
```

`--report` is a `StringSliceVar` (pflag). When no value is written directly after the flag,
pflag consumes the next command-line argument as the value - even if that argument is another
flag name starting with `--`. The profile name becomes `"--target-profile"` and the actual
`dev` value is discarded.

### Fix

Add an early validation in `RunE` before the rest of the flag-validation logic:

```go
for _, rp := range reportProfiles {
    if strings.HasPrefix(rp, "-") {
        return fmt.Errorf(
            "--report expects a profile name as its value, not a flag (%q);\n"+
                "  use: --report=%s  or  --target-profile=%s",
            rp, strings.TrimLeft(rp, "-"), strings.TrimLeft(rp, "-"))
    }
}
```

Note: `--report` and `--target-profile` are mutually exclusive. To validate against a
single profile, use `--target-profile dev`. To produce a multi-profile report, use
`--report=dev` or `--report dev,qa`.

## Related commands and files

| File | Change needed |
| ---- | ------------- |
| `cmd/profile.go` | Sentinel guard in add/edit; keychain password for login validation; KEYCHAIN column |
| `internal/config/prompt.go` | `[keychain]` hint; skip re-offer when password unchanged |
| `internal/config/prompt_test.go` | Add test for kept-sentinel path |
| `cmd/package.go` | Guard against `--report` values starting with `-` |
