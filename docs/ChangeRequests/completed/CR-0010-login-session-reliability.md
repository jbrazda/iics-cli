# CR-0010 - Login, Session, and Profile Reliability

| Field | Value |
|-------|-------|
| ID | CR-0010 |
| Status | New |
| Priority | High |
| Component | `cmd/login.go`, `cmd/logout.go`, `cmd/profile.go`, `cmd/root.go`, `internal/client/client.go`, `internal/config/session.go` |

---

## Summary

Several related issues make the login/session lifecycle unreliable:

- Unknown profiles produce a hard error instead of prompting the user to create a new profile.
- Sessions are never persisted after auto-login or 401 renewal, so every command re-authenticates unnecessarily.
- Account lockout risk is not communicated clearly when renewal fails.
- The `profile list` command hides the `Region` field.
- The `profile show` command does not surface session-derived information (org, last login time).
- The sessions file has no `lastLoginTime` field, making it impossible to track when credentials were last used.

---

## Issue 1 - Unknown profile gives a hard error

### Observed behaviour

```
./iics login --profile ttt
Error: username not configured for profile "ttt"
```

### Root cause

`config.ResolveProfile()` returns an error when the named profile exists but has no username. When the profile name is completely absent from the config file, the same error path fires because an empty struct has no username. There is no distinction between "profile exists but credentials are missing" and "profile does not exist at all".

The interactive profile-creation wizard in `getClient()` (`cmd/root.go`) activates only when the terminal is interactive and credentials are partially missing, not when the profile name is entirely unknown.

### Proposed fix

In `cmd/login.go`, before calling `resolveProfile`, check whether the named profile key exists in the loaded config. If it does not exist and the command is running on a TTY, prompt the user:

```
Profile "ttt" does not exist. Create it now? [Y/n]:
```

If confirmed, invoke `config.PromptProfile(nil, name)`, save the new profile to config, then continue with login. If declined or not a TTY, return a clear error:

```
profile "ttt" not found; run 'iics profile add ttt' to create it
```

---

## Issue 2 - Session is never saved after auto-login

### Observed behaviour

After any command other than `iics login`, the auto-login inside `do()` produces a valid session token, but that token is never written to `~/.iics/sessions.yaml`. Every subsequent command re-authenticates from scratch. Renaming or deleting the sessions file has no observable effect because it is never written.

### Root cause

`saveSession()` in `cmd/root.go` is called only from `cmd/login.go` after an explicit login command. The `do()` method in `internal/client/client.go` calls `c.Login()` silently (lazy login) but has no mechanism to trigger `saveSession()` — it is a `cmd`-layer function and the client has no dependency on it.

### Proposed fix

Add an `OnLoginSuccess func(loginResp *LoginResponse)` callback field to the `Client` struct in `internal/client/client.go`. Call this callback inside `Login()` after a successful authentication response is fully stored. In `getClient()` (`cmd/root.go`), wire the callback to `saveSession()`. This keeps all file-system persistence in the `cmd` layer while allowing the client to notify callers of every successful login.

---

## Issue 3 - Session is never saved after 401 renewal

### Observed behaviour

When a cached session expires mid-command, `do()` successfully re-authenticates (401 retry path), but the renewed token is never written to the sessions file. The next command will again find an expired entry and re-authenticate.

### Root cause

Same as Issue 2: the 401 retry path in `do()` calls `c.Login()` but has no way to call `saveSession()`.

### Proposed fix

The `OnLoginSuccess` callback from Issue 2 covers this automatically, since `Login()` fires the callback regardless of whether it was triggered by a fresh start or a 401 renewal.

---

## Issue 4 - Account lockout not clearly communicated on renewal failure

### Observed behaviour

If the renewal login fails (e.g. wrong password, account locked), the current error surfaces as a generic `SessionExpiredError` without actionable guidance.

### Root cause

The error message does not tell the user what to do next and does not distinguish a renewal failure from a first-time login failure.

### Proposed fix

When the 401-renewal `Login()` call fails, return a wrapped error with a clear message:

```
session renewal failed: <original error>; run 'iics login --profile <name>' to re-authenticate
```

Do not retry more than once. This prevents any risk of account lockout from repeated failed attempts.

---

## Issue 5 - Logout behaviour (documented, no code change needed)

`cmd/logout.go` already correctly calls `cache.Delete(profileName)` and `cache.Save()`, removing the session entry from the file. API-side logout failure is logged as a warning and does not block local cache cleanup. This behaviour is correct and should be preserved.

Note: full benefit of logout cleanup requires Issue 2 to be fixed so that sessions are written in the first place.

---

## Issue 6 - Region not shown in `profile list`

### Observed behaviour

`profile list` shows an ENDPOINT column that displays `LoginURL` if set, or `Region` otherwise. When a profile has a discovered `LoginURL`, the `Region` value is invisible.

### Proposed fix

Add a dedicated `REGION` column to `profile list`:

| NAME | DEFAULT | REGION | ENDPOINT | USERNAME |
|------|---------|--------|----------|----------|
| prod | yes | us1 | https://dm-us.informaticacloud.com/saas/api | user@example.com |
| dev | | eu1 | https://dm-em.informaticacloud.com/saas/api | dev@example.com |

Both fields are shown independently. ENDPOINT continues to prefer `LoginURL` over `Region` as a resolved value.

---

## Issue 7 - `profile show` missing session-derived fields

### Observed behaviour

`profile show <name>` displays config-file fields only (Name, Default, Region, Login URL, Base API URL, CAI URL, Username, Password). Information discovered at login time (org name, org ID, last login time, session expiry) is not shown.

### Proposed fix

`profile show` should additionally load the session cache and append session-derived rows:

| FIELD | VALUE |
|-------|-------|
| Name | prod |
| Default | yes |
| Region | us1 |
| Login URL | https://... |
| Base API URL | https://... |
| CAI URL | https://... |
| Username | user@example.com |
| Password | *** |
| Org Name | My Org |
| Org ID | abc123 |
| Session User | user@example.com |
| Last Login | 2026-03-23 14:05:00 UTC |
| Session Expires | 2026-03-23 14:35:00 UTC |

If no session cache entry exists, the session-derived rows display `(no active session)`.

---

## Issue 8 - `LastLoginTime` not persisted in sessions file

### Observed behaviour

`SessionEntry` has a `CreatedAt` field that records when the session entry was written. There is no explicit `lastLoginTime` field, making it ambiguous whether `CreatedAt` represents session creation time or last authentication time.

### Proposed fix

Add a `LastLoginTime time.Time` field to `SessionEntry` in `internal/config/session.go`:

```go
type SessionEntry struct {
    SessionID     string    `yaml:"sessionId"`
    BaseAPIURL    string    `yaml:"baseApiUrl"`
    CAIUrl        string    `yaml:"caiUrl,omitempty"`
    LoginURL      string    `yaml:"loginUrl,omitempty"`
    OrgID         string    `yaml:"orgId"`
    OrgName       string    `yaml:"orgName"`
    UserName      string    `yaml:"userName"`
    CreatedAt     time.Time `yaml:"createdAt"`
    LastLoginTime time.Time `yaml:"lastLoginTime"`
}
```

Set `LastLoginTime = time.Now()` in `saveSession()` whenever a new session is written. This field persists across session renewals and is displayed by `profile show`.

---

## Acceptance Criteria

1. `./iics login --profile newprofile` (not in config, running on TTY) prompts to create the profile, saves it, logs in, and writes the session to `~/.iics/sessions.yaml` with `lastLoginTime` set.
2. `./iics login --profile newprofile` (not in config, non-TTY or user declines) returns `profile "newprofile" not found; run 'iics profile add newprofile' to create it`.
3. `./iics login --profile existing` logs in and updates the session file with a fresh `createdAt` and `lastLoginTime`.
4. After deleting the sessions file, any command (e.g. `./iics connection list`) silently re-logins and re-creates `~/.iics/sessions.yaml`.
5. After manually expiring a session entry (set `createdAt` to > 30 min ago), the next command silently renews the session and updates the sessions file.
6. Wrong credentials during renewal produce: `session renewal failed: <err>; run 'iics login --profile <name>' to re-authenticate`. No second retry occurs.
7. `./iics logout --profile prod` removes the session entry from `~/.iics/sessions.yaml`. Confirmed by inspecting the file.
8. `./iics profile list` shows NAME, DEFAULT, REGION, ENDPOINT, USERNAME columns.
9. `./iics profile show prod` shows all config fields plus Org Name, Org ID, Session User, Last Login, Session Expires from the cache. Shows `(no active session)` for session rows when no valid cache entry exists.

---

## Files to Modify

| File | Change |
|------|--------|
| `internal/config/session.go` | Add `LastLoginTime time.Time` field to `SessionEntry` |
| `internal/client/client.go` | Add `OnLoginSuccess func(*LoginResponse)` callback field; call it in `Login()` after successful auth |
| `cmd/root.go` | Wire `OnLoginSuccess` to `saveSession()` in `getClient()` |
| `cmd/login.go` | Detect missing profile; prompt to create on TTY; clear error on non-TTY |
| `cmd/profile.go` | Add REGION column to `profile list`; add session-derived rows to `profile show` |
