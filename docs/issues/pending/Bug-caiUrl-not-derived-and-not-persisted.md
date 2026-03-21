# BUG: caiUrl not derived correctly from login response and not persisted to profile or session cache

## Symptoms

Running any `publish` or `state` command that requires the CAI endpoint fails with:

```
Error: batch 1: submitting: CAI URL not configured; set caiUrl in profile config,
IICS_CAI_URL env var, or --cai-url flag
```

Users must manually add `caiUrl` to their profile config to work around this, even though all
the information needed to derive it is present in the login response.

## Root Causes

There are four distinct problems that together cause this failure.

### 1. Wrong CAI URL derivation formula in `auth.go`

`Login()` already attempts to auto-derive `caiURL` from `baseApiUrl`, but uses the wrong
transformation - it strips only the path and returns the bare `scheme://host`:

```go
// current - WRONG
c.caiURL = fmt.Sprintf("%s://%s", u.Scheme, u.Host)
// baseApiUrl  https://use4.dm-us.informaticacloud.com/saas
// produces    https://use4.dm-us.informaticacloud.com  <-- wrong host
```

The correct CAI URL inserts `-cai` into the first DNS label of the hostname:

```
baseApiUrl:  https://use4.dm-us.informaticacloud.com/saas
caiUrl:      https://use4-cai.dm-us.informaticacloud.com
```

Derivation rule: split host on `.`, append `-cai` to the first label, rebuild:
`use4` + `.` + `dm-us.informaticacloud.com` -> `use4-cai.dm-us.informaticacloud.com`.

### 2. Derived URLs not persisted back to the profile config after login

`cmd/login.go` calls `saveSession()` but never writes the derived `loginUrl`, `baseApiUrl`,
or `caiUrl` back to the profile in `~/.iics/config.yaml`. On the next command that needs the
CAI URL, the profile still has no `caiUrl` set.

The `Profile` struct also lacks a `BaseAPIURL` field, so there is no place to store the
org-specific base API URL discovered during login.

### 3. Session cache does not store the CAI URL

`SessionEntry` (`internal/config/session.go`) has no `CAIUrl` field. `saveSession()` does
not write it. `getClient()` does not restore it when restoring from cache:

```go
// getClient - session restore path
c.SetSession(entry.SessionID, entry.BaseAPIURL)
// caiURL from the session is never restored
```

So even if `Login()` computed the right `caiURL` in memory, it is thrown away the next time
the session is loaded from cache.

### 4. `profile show` renders a horizontal table that is hard to read

`newProfileShowCmd()` builds a single-row map and passes it to `formatter.Format()` as a
`[]map[string]interface{}`, which produces an overly wide horizontal table. A profile's
detail view should be a vertical two-column table (FIELD | VALUE), one attribute per row.
The current output also omits `caiUrl` from the shown fields.

## Affected Files

- `internal/client/auth.go` - wrong `caiURL` derivation formula
- `internal/config/config.go` - `Profile` missing `BaseAPIURL` field; no URL-derivation helper
- `internal/config/session.go` - `SessionEntry` missing `CAIUrl` field
- `internal/config/prompt.go` - `PromptProfile` does not derive or offer computed URLs
- `cmd/root.go` - `saveSession` does not persist derived URLs; `getClient` does not restore
  `caiURL` from session cache
- `cmd/login.go` - does not write derived URLs back to profile config after successful login
- `cmd/profile.go` - `profile show` is horizontal; `profile add`/edit missing derived URL
  preview and override prompts

## Fix

### 1. Fix `DeriveCaiURL` in `internal/config/config.go`

Add a package-level helper (also usable from `auth.go`) that applies the correct rule:

```go
// DeriveCaiURL derives the CAI base URL from an IICS product baseApiUrl.
// Example: https://use4.dm-us.informaticacloud.com/saas
//       -> https://use4-cai.dm-us.informaticacloud.com
// Returns "" if the URL cannot be parsed or the host has no dot separator.
func DeriveCaiURL(baseAPIURL string) string {
    u, err := url.Parse(baseAPIURL)
    if err != nil || u.Host == "" {
        return ""
    }
    dot := strings.Index(u.Host, ".")
    if dot < 0 {
        return ""
    }
    return fmt.Sprintf("%s://%s-cai%s", u.Scheme, u.Host[:dot], u.Host[dot:])
}
```

Update `auth.go` to call `DeriveCaiURL` instead of the inline extraction.

### 2. Add `BaseAPIURL` to `Profile`; enrich profile after login

```go
// internal/config/config.go
type Profile struct {
    Name       string `yaml:"name" mapstructure:"name"`
    Region     string `yaml:"region,omitempty" mapstructure:"region"`
    Username   string `yaml:"username" mapstructure:"username"`
    Password   string `yaml:"password" mapstructure:"password"`
    LoginURL   string `yaml:"loginUrl,omitempty" mapstructure:"loginUrl"`
    BaseAPIURL string `yaml:"baseApiUrl,omitempty" mapstructure:"baseApiUrl"`
    CaiURL     string `yaml:"caiUrl,omitempty" mapstructure:"caiUrl"`
}
```

In `cmd/login.go`, after a successful login, write `loginUrl`, `baseApiUrl`, and `caiUrl`
back to the profile and save the config:

```go
// After Login() succeeds:
p.LoginURL  = loginURL                                // the URL actually used
p.BaseAPIURL = loginResp.Products[0].BaseAPIURL       // from response
if p.CaiURL == "" {
    p.CaiURL = config.DeriveCaiURL(p.BaseAPIURL)      // derive if not already set
}
cfg.Profiles[profileName] = p
_ = cfg.Save(cfgFile)                                 // best-effort; warn on error
```

### 3. Add `CAIUrl` to `SessionEntry`; persist and restore it

```go
// internal/config/session.go
type SessionEntry struct {
    SessionID  string    `yaml:"sessionId"`
    BaseAPIURL string    `yaml:"baseApiUrl"`
    CAIUrl     string    `yaml:"caiUrl,omitempty"`
    LoginURL   string    `yaml:"loginUrl,omitempty"`
    OrgID      string    `yaml:"orgId"`
    OrgName    string    `yaml:"orgName"`
    UserName   string    `yaml:"userName"`
    CreatedAt  time.Time `yaml:"createdAt"`
}
```

In `saveSession` (`cmd/root.go`), include the CAI URL and the login URL:

```go
cache.Set(profileName, &config.SessionEntry{
    SessionID:  loginResp.UserInfo.SessionID,
    BaseAPIURL: c.BaseAPIURL(),
    CAIUrl:     c.CAIUrl(),       // new Client getter needed
    LoginURL:   loginURL,
    OrgID:      loginResp.UserInfo.OrgID,
    OrgName:    loginResp.UserInfo.OrgName,
    UserName:   loginResp.UserInfo.Name,
})
```

In `getClient`, restore `CAIUrl` when loading from cache:

```go
if entry, ok := cache.Get(profileName); ok {
    c.SetSession(entry.SessionID, entry.BaseAPIURL)
    if entry.CAIUrl != "" && p.CaiURL == "" {
        c.SetCAIUrl(entry.CAIUrl)   // new Client setter needed
    }
    return c, nil
}
```

### 4. Vertical `profile show`

Replace the single-row `[]map[string]interface{}` formatter call with a slice of
`{field, value}` rows printed as a two-column table:

```go
rows := []map[string]interface{}{
    {"field": "Name",        "value": name},
    {"field": "Default",     "value": defaultMark},
    {"field": "Region",      "value": p.Region},
    {"field": "Login URL",   "value": p.LoginURL},
    {"field": "Base API URL","value": p.BaseAPIURL},
    {"field": "CAI URL",     "value": p.CaiURL},
    {"field": "Username",    "value": p.Username},
    {"field": "Password",    "value": maskedPassword},
}
columns := []output.Column{
    {Header: "FIELD", Field: "field", Width: 15},
    {Header: "VALUE", Field: "value"},
}
return f.Format(rows, columns)
```

### 5. Derive URLs in `profile add`/`profile edit`

In `PromptProfile` (`internal/config/prompt.go`), after the user provides a region or login
URL, derive `baseApiUrl` and `caiUrl` and show them for confirmation:

- If a known region is entered, look up the pod hostname and construct both URLs.
- Show the computed values and prompt: `CAI URL [<derived>]: ` so the user can override.
- Store all three fields (`loginUrl`, `baseApiUrl` if deterministic, `caiUrl`) in the
  returned `Profile`.

Note: `baseApiUrl` is org-specific (includes a pod-instance prefix like `use4`) and cannot
be known before a real login. The profile add flow should derive and store `loginUrl` and
`caiUrl` where deterministic (for known regions), and note that `baseApiUrl` is populated
on first successful login.

## Acceptance Criteria

- [ ] `iics login` writes `loginUrl`, `baseApiUrl`, and `caiUrl` back to the profile config
- [ ] `iics login` stores `caiUrl` and `loginUrl` in the session cache entry
- [ ] After `iics login`, subsequent `publish run` / `state` commands succeed without
  manually configuring `caiUrl`
- [ ] Session restore in `getClient` applies `caiUrl` from the cache when the profile has
  none set
- [ ] `DeriveCaiURL("https://use4.dm-us.informaticacloud.com/saas")` returns
  `"https://use4-cai.dm-us.informaticacloud.com"`
- [ ] `profile show <name>` renders a vertical two-column (FIELD/VALUE) table including
  `baseApiUrl` and `caiUrl`
- [ ] `profile add` prompts for and derives `caiUrl` from the region/login URL, allowing
  user override
- [ ] All existing tests pass; new unit tests cover `DeriveCaiURL` edge cases
