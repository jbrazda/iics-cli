# CR: Add `profile edit` subcommand

## Status

New

## Summary

Add an `iics profile edit [name]` command that interactively updates an existing profile's
credentials, validates them by logging in, derives the correct org-specific URLs from the
login response, and refreshes the session cache - all in one step.

## Motivation

Currently there is no way to update the credentials of an existing profile without manually
editing `~/.iics/config.yaml`. When a password expires or an account is re-keyed:

1. The user must open the YAML file and find the correct profile entry.
2. Paste the new password (taking care not to break YAML formatting).
3. Run `iics login --profile <name>` to refresh the session and discover org-specific URLs.

`profile add [name]` re-creates a profile from scratch but does not validate credentials or
derive `baseApiUrl`/`caiUrl` from the login response - those URLs are only updated by
`iics login`. The proposed `profile edit` command closes this gap.

## Desired Behaviour

```
$ iics profile edit qa
Setting up profile "qa"

Username [jaroslav.brazda.qa@natl.com]:
Password [current: ***]:
Region or login URL [USE4]:
CAI URL [https://use4-cai.dm-us.informaticacloud.com]:
Set as default profile? [Y/n]: n

Validating credentials for "jaroslav.brazda.qa@natl.com"...
Profile "qa" updated and session refreshed.
  User:    Jaroslav Brazda (QA)
  Org:     Natl QA (abc123)
  BaseURL: https://use4.dm-us.informaticacloud.com/saas
  CAI URL: https://use4-cai.dm-us.informaticacloud.com
```

- Prompts use existing stored values as defaults (press Enter to keep).
- On login failure the profile is NOT saved; the user sees a clear error.
- On success: profile config is saved and session cache is refreshed.
- `baseApiUrl` and `caiUrl` are derived from the login response (org-specific), not the
  generic login URL.

## Implementation Notes

- Reuse `config.PromptProfile(existing, name)` for the interactive prompts.
- After prompting, call `client.Login()` to validate and discover URLs.
- Update `p.BaseAPIURL` and `p.CaiURL` from login response before saving.
- Call `saveSession(name, loginURL, c, loginResp)` to refresh the cache.
- Wire into `newProfileCmd()` alongside the existing subcommands.
- Add `context` and `internal/client` imports to `cmd/profile.go`.
- Requires fix for Bug-ResolveProfile-env-var-mutation (applied first so
  `cfg.Profiles[name]` is not mutated by env-var overrides during the edit flow).

## Files to Modify

- `cmd/profile.go` - add `newProfileEditCmd()`, register it
- `docs/documentation/profile.md` - document new subcommand
- `completions/` - regenerate
