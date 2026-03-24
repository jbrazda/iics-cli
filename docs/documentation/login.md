# login

Authenticate with IICS and cache the session for subsequent commands.

The session token is stored in `~/.iics/sessions.yaml` and reused automatically for 30 minutes.
After expiry the next command triggers a silent re-login and refreshes the cache, so subsequent
commands do not re-authenticate unnecessarily.

On a successful login the command also writes the discovered `loginUrl`, `baseApiUrl`, and
`caiUrl` back to the active profile in `~/.iics/config.yaml`. This means commands that require
the CAI endpoint (such as `publish run` and `state fetch`) work automatically after the first
login without any manual configuration.

## Synopsis

```bash
iics login [flags]
```

## Flags

This command has no command-specific flags. All [global flags](../../README.md#global-flags) apply.

## Authentication sources

Credentials are resolved in this order (highest to lowest priority):

1. Environment variables: `IICS_USERNAME`, `IICS_PASSWORD`
2. `--profile` flag
3. `defaultProfile` in `~/.iics/config.yaml`

If no password is available, the command prompts interactively.

## Creating a profile on first use

If the named profile does not exist in `~/.iics/config.yaml` and stdin is a terminal, the
command offers to create it interactively before proceeding with the login:

```text
Profile "prod" does not exist. Create it now? [Y/n]:
```

Pressing Enter (or entering `y`) launches the interactive setup wizard. The new profile is
saved to the config file and the login continues immediately without a separate
`iics profile add` step.

In non-interactive environments (CI/CD, cron, piped input) or when the user declines, the
command exits with:

```text
profile "prod" not found; run 'iics profile add prod' to create it
```

## Profile enrichment after login

After a successful login the following fields are written back to the active profile and saved
to `~/.iics/config.yaml`:

| Field        | Value                                                         |
| ------------ | ------------------------------------------------------------- |
| `loginUrl`   | The login URL actually used (resolved from region or explicit)|
| `baseApiUrl` | Org-specific base API URL from the login response             |
| `caiUrl`     | CAI endpoint URL derived from `baseApiUrl` (see rule below)   |

The `caiUrl` derivation inserts `-cai` after the first DNS label of the `baseApiUrl` hostname:

```text
baseApiUrl: https://use4.dm-us.informaticacloud.com/saas
caiUrl:     https://use4-cai.dm-us.informaticacloud.com
```

If `caiUrl` is already set in the profile, it is not overwritten.

The same `caiUrl` and `loginUrl` are also stored in the session cache (`~/.iics/sessions.yaml`)
so that they are available when restoring a cached session without a full re-login.

## Examples

```bash
# Login with the default profile
iics login

# Login with a named profile
iics login --profile prod

# Login using environment variable overrides
IICS_USERNAME=ops@company.com IICS_PASSWORD=s3cret iics login

# Login to a specific region by overriding the login URL
IICS_LOGIN_URL=https://dm-em.informaticacloud.com/saas/public/core/v3/login iics login

# CI/CD: non-interactive login
export IICS_USERNAME=svc-account@company.com
export IICS_PASSWORD=$SECRET_PASSWORD
iics login --profile prod
```

## Output

On success the command prints the authenticated user, org info, base API URL, and derived CAI URL:

```
Logging in as jane.smith@company.com...
Logged in successfully.
  User:    Jane Smith
  Org:     My Production Org (a1B2c3D4E5F6)
  BaseURL: https://use4.dm-us.informaticacloud.com/saas
  CAI URL: https://use4-cai.dm-us.informaticacloud.com
  Profile: prod
```

## See also

- [logout](logout.md)
- [profile](profile.md) - manage connection profiles
- [Configuration reference](../../README.md#configuration)
