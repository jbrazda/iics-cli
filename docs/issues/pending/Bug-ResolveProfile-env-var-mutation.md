# Bug: ResolveProfile mutates shared profile pointer with env-var overrides, corrupting saved config

## Status

New

## Summary

`config.ResolveProfile` returns a pointer to the **same** `*Profile` object stored in
`cfg.Profiles[profileName]` and then overwrites its credential fields with values from
`IICS_USERNAME`, `IICS_PASSWORD`, `IICS_REGION`, `IICS_LOGIN_URL`, and `IICS_CAI_URL`
environment variables. Because the pointer is shared, those env-var values are also written
into the in-memory `Config.Profiles` map.

Commit `a9c5208` added `cfg.Save(cfgFile)` to `cmd/login.go` after a successful login. Now
that `cfg.Save` is called, the env-var-overridden credentials are persisted to
`~/.iics/config.yaml`, potentially overwriting the correct stored password for the profile
that was just logged into. The next invocation (without the env vars set) reads the now-
incorrect password from the config file, causing login failures. Repeated failures can
trigger IICS account lock-out (error code `IDS_086`: "User account is locked").

## Steps to Reproduce

1. Store correct credentials for profile `qa` in `~/.iics/config.yaml`.
2. Set `IICS_PASSWORD=wrongpassword` in the shell environment.
3. Run `iics login --profile qa` (or any command that triggers an implicit login).
4. Login succeeds (because step 2's env var overrides the config password and the wrong
   password happened to work, OR the env var matches a different valid credential).
5. `cfg.Save()` writes `wrongpassword` as `qa`'s stored password.
6. Unset `IICS_PASSWORD` and run `iics user list --profile qa`.
7. Login fails with incorrect credentials; after several failures the account is locked.

## Root Cause

In `internal/config/config.go`, `ResolveProfile`:

```go
profile, ok := c.Profiles[profileName]  // pointer into the map
if !ok {
    profile = &Profile{}
}
if v := os.Getenv("IICS_PASSWORD"); v != "" {
    profile.Password = v  // mutates cfg.Profiles[profileName].Password
}
```

The mutation reaches `cfg.Profiles` because `profile` is the same pointer.

## Impact

- Any IICS_* env-var override can corrupt the stored credentials in `~/.iics/config.yaml`
  whenever `cfg.Save()` is called (currently only in `cmd/login.go`).
- Corrupted credentials cause repeated login failures and potential account lock-out.

## Fix

In `ResolveProfile`, copy the Profile struct into a local variable before applying env-var
overrides:

```go
var resolved Profile
if existing, ok := c.Profiles[profileName]; ok {
    resolved = *existing  // shallow copy - env overrides go to copy only
}
if v := os.Getenv("IICS_PASSWORD"); v != "" {
    resolved.Password = v  // does NOT mutate cfg.Profiles[profileName]
}
return &resolved, nil
```

Additionally, fix `cmd/login.go` to write discovered URL fields (`loginUrl`, `baseApiUrl`,
`caiUrl`) directly to `cfg.Profiles[profileName]` (the original, unmodified entry) instead
of reassigning `p` (the env-var copy).

## Files Affected

- `internal/config/config.go` - `ResolveProfile` function
- `cmd/login.go` - URL persistence after successful login
