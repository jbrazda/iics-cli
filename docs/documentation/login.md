# login

Authenticate with IICS and cache the session for subsequent commands.

The session token is stored in `~/.iics/sessions.yaml` and reused automatically for 30 minutes. After expiry the next command triggers a silent re-login.

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

On success the command prints the authenticated user name, organisation info, and the base API URL that will be used for subsequent requests.

```
Logged in as Jane Smith (jane.smith@company.com)
Org: My Production Org (orgId: abcdef1234)
Base URL: https://usw3.dm.informaticacloud.com
Session cached for 30 minutes.
```

## See also

- [logout](logout.md)
- [Configuration reference](../../README.md#configuration)
