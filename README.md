# iics - Informatica Intelligent Cloud Services CLI

A comprehensive command-line interface for the [Informatica Intelligent Cloud Services (IICS)](https://www.informatica.com/products/cloud-integration.html) platform REST API v3. Manage assets, users, connections, export/import packages, and more - from your terminal or CI/CD pipelines.

## Features

- **Full API v3 coverage** - objects, connections, schedules, export/import, users, roles, permissions, runtime environments, agents, tags, source control, and more
- **Interactive setup** - guided profile wizard on first run, or via `iics profile add`
- **Multi-profile configuration** - switch between dev/staging/prod orgs with `--profile`
- **Session caching** - reuses sessions across invocations (30-min window) to avoid repeated logins
- **Automatic session refresh** - transparent 401 retry with re-authentication
- **Flexible output** - human-readable tables (default) or JSON (`--output json`)
- **CI/CD friendly** - environment variable overrides, `--yes` flag for non-interactive use, JSON output for scripting
- **Cross-platform** - builds for Linux, macOS, and Windows (amd64 & arm64)

## Installation

### From source

Requires Go 1.25 or later.

```bash
go install github.com/jbrazda/iics-cli@latest
```

### From releases

Download the pre-built binary for your platform from the [Releases](https://github.com/jbrazda/iics-cli/releases) page.

### Build from source

```bash
git clone https://github.com/jbrazda/iics-cli.git
cd iics-cli
make build
```

## Quick Start

### 1. Set up a profile

```bash
iics profile add
```

The wizard prompts for your username, password, and region, then saves the profile to
`~/.iics/config.yaml`. You can also set up multiple named profiles:

```bash
iics profile add dev
iics profile add prod
iics profile set-default dev
```

Alternatively, create the config file manually - see the [Configuration](#configuration) section.

### 2. Login

```bash
# Uses default profile; prompts for password if not in config
iics login

# Use a specific profile
iics login --profile prod

# Override credentials via environment
IICS_USERNAME=user@company.com IICS_PASSWORD=secret iics login
```

### 3. Use commands

```bash
# List objects
iics objects list --type MTT

iics objects list                       # all objects, auto-paginated
iics objects list --type MTT            # all mappings, auto-paginated
iics objects list --type MTT --limit 50 # first 50 only
iics objects list --limit 50 --skip 100 # results 101-150

# List objects as JSON
iics objects list --type DTEMPLATE --output json

# List connections
iics connection list

# Get a specific connection
iics connection get --id <connection-id>

# Export assets
iics export create --name "my-export" --ids <asset-id-1>,<asset-id-2>
iics export status --id <job-id>
iics export download --id <job-id> --output export.zip

# Import assets
iics import upload --file export.zip
iics import start --id <job-id>
iics import status --id <job-id>

# List users
iics user list

# Logout
iics logout
```

## Configuration

### Config file (`~/.iics/config.yaml`)

```yaml
defaultProfile: dev
httpTimeout: 120     # seconds; per-HTTP-request timeout (default 120)
style:
  theme: default     # default | minimal | compact | plain | markdown | gh
  noColor: false     # true = disable color permanently (same as --no-color)
  headerColor: ""    # lipgloss color: "6"=cyan, "244"=gray, "#FF0000"=hex (empty = theme default)
profiles:
  dev:
    name: "Development Org"
    region: "USE4"
    username: "user@company.com"
    password: ""
    loginUrl: "https://use4.dm-us.informaticacloud.com/saas/public/core/v3/login"
    baseApiUrl: "https://use4.dm-us.informaticacloud.com/saas"
    caiUrl: "https://use4-cai.dm-us.informaticacloud.com"
  prod:
    name: "Production Org"
    region: "EMEA"
    username: "admin@company.com"
    password: "@keyring"   # real password stored in OS keychain
    loginUrl: "https://dm-em.informaticacloud.com/saas/public/core/v3/login"
    baseApiUrl: "https://dm-em.informaticacloud.com/saas"
    caiUrl: "https://dm-em-cai.informaticacloud.com"
```

The `loginUrl`, `baseApiUrl`, and `caiUrl` fields are populated automatically after the first
`iics login` - you do not need to set them manually.

The `style` section controls table output appearance. Available themes:

| Theme      | Description                                                                                            |
|------------|--------------------------------------------------------------------------------------------------------|
| `default`  | Unicode rounded borders, cyan bold headers (TTY only)                                                  |
| `minimal`  | No borders, colored bold headers with unicode underline                                                |
| `compact`  | No borders, gray bold headers, 1-space column gap (TTY only)                                           |
| `plain`    | ASCII borders, no color - used automatically for non-TTY output                                        |
| `markdown` | GitHub-flavored markdown table, no color, always rendered regardless of TTY                            |
| `gh`       | GitHub CLI-style: no borders, no separator, plain headers, no color, always rendered regardless of TTY |

Non-TTY output (piped, redirected) always uses `plain` regardless of the configured theme.
The `markdown` and `gh` themes are exceptions - they always render as-is even when output
is piped, since they are already colorless.
The `NO_COLOR` environment variable is also respected.

The `style.headerColor` field accepts a lipgloss color string (`"6"` for cyan, `"244"` for
gray, `"#FF0000"` for hex) and overrides the built-in header color for `default` and `minimal`
themes. Leave empty to use the theme default.

### Environment variable overrides

| Variable                    | Description                                                                              |
|-----------------------------|------------------------------------------------------------------------------------------|
| `IICS_PROFILE`              | Override default profile                                                                 |
| `IICS_USERNAME`             | Override profile username                                                                |
| `IICS_PASSWORD`             | Override profile password (takes precedence over keychain)                               |
| `IICS_REGION`               | Override profile region                                                                  |
| `IICS_LOGIN_URL`            | Override computed login URL                                                              |
| `IICS_CAI_URL`              | Override profile `caiUrl`                                                                |
| `IICS_OUTPUT`               | Override default output format                                                           |
| `IICS_THEME`                | Override table theme (same values as `--theme` flag)                                     |
| `IICS_HTTP_TIMEOUT`         | Override per-HTTP-request timeout in seconds (default `120`; `--http-timeout` flag wins if set) |
| `IICS_VALID_DEPLOY_TARGETS` | Override valid target allowlist for `iics release` commands (comma-separated)            |
| `IICS_TARGET_PROFILE_MAP`   | Override target to profile mapping for `iics release plan` (format `TARGET=profile,...`) |
| `IICS_USER_<TARGET>`        | CI target username fallback for `iics release plan` when profile is not configured       |
| `IICS_PWD_<TARGET>`         | CI target password fallback for `iics release plan` when profile is not configured       |
| `IICS_REGION_<TARGET>`      | Optional CI target region for `iics release plan`                                        |
| `IICS_LOGIN_URL_<TARGET>`   | Optional CI target login URL for `iics release plan`                                     |

Environment variables take precedence over config file values.

### Supported regions

| Region                                 | ssLogin Host                |
|----------------------------------------|-----------------------------|
| US, USW1, USE2, USW3, USE4, USW5, USE6 | dm-us.informaticacloud.com  |
| USW1-1, USW3-1                         | dm1-us.informaticacloud.com |
| USW1-2                                 | dm2-us.informaticacloud.com |
| CAC1                                   | dm-na.informaticacloud.com  |
| APSE1, APJ                             | dm-ap.informaticacloud.com  |
| APNE1                                  | dm1-ap.informaticacloud.com |
| EMEA, EMWE1                            | dm-em.informaticacloud.com  |

## Security

Passwords stored in `~/.iics/config.yaml` can be protected using the OS keychain instead of
keeping them in plaintext. When you add or edit a profile interactively, iics-cli offers to
store the password in the OS keychain and writes the sentinel value `@keyring` in the config
file instead of the real password.

Supported backends:

| Platform | Backend                                            |
|----------|----------------------------------------------------|
| macOS    | macOS Keychain (`security` framework)              |
| Linux    | D-Bus Secret Service (e.g. GNOME Keyring, KWallet) |
| Windows  | Windows Credential Manager                         |

To store a password for an existing profile:

```bash
iics profile set-password <profile-name>
```

To move a profile's password into the keychain after manual config edits:

```bash
iics profile set-password dev
```

`IICS_PASSWORD` always takes precedence over both the keychain and plaintext config, making
it easy to override credentials in CI pipelines without touching the config file.

## Commands

| Command                                              | Alias  | Subcommands                                                            | Description                                                        |
|------------------------------------------------------|--------|------------------------------------------------------------------------|--------------------------------------------------------------------|
| [activitylog](docs/documentation/activitylog.md)     |        | `list`, `get`                                                          | Query activity logs for completed jobs                             |
| [agent](docs/documentation/agent.md)                 |        | `list`, `get`, `details`, `start`, `stop`                              | Manage Secure Agents                                               |
| [auditlog](docs/documentation/auditlog.md)           |        | `list`                                                                 | Query organization audit log (V2 API)                              |
| [completion](docs/documentation/completion.md)       |        | `bash`, `zsh`, `fish`, `powershell`                                    | Generate shell completion scripts                                  |
| [connection](docs/documentation/connection.md)       | `conn` | `list`, `get`, `create`, `update`, `delete`                            | Manage connections                                                 |
| [export](docs/documentation/export.md)               |        | `run`, `start`, `status`, `download`, `create`                         | Export asset packages                                              |
| [folder](docs/documentation/folder.md)               |        | `create`, `update`, `delete`                                           | Manage folders                                                     |
| [import](docs/documentation/import.md)               | `imp`  | `run`, `upload`, `start`, `status`, `download-log`                     | Import asset packages                                              |
| [login](docs/documentation/login.md)                 |        |                                                                        | Authenticate and cache session                                     |
| [logout](docs/documentation/logout.md)               |        |                                                                        | Invalidate session                                                 |
| [lookup](docs/documentation/lookup.md)               |        |                                                                        | Resolve object IDs, names, and paths                               |
| [metering](docs/documentation/metering.md)           |        | `get`, `download`                                                      | Query usage and metering data                                      |
| [objects](docs/documentation/objects.md)             |        | `list`, `dependencies`                                                 | List/search assets, find dependencies                              |
| [package](docs/documentation/package.md)             |        | `expand`, `create`, `dependencies`                                     | Extract, assemble, or inspect dependencies of IICS export packages |
| [permission](docs/documentation/permission.md)       | `perm` | `get`, `set`, `delete`                                                 | Manage object-level permissions                                    |
| [privilege](docs/documentation/privilege.md)         |        | `list`                                                                 | List available privileges                                          |
| [profile](docs/documentation/profile.md)             |        | `add`, `edit`, `list`, `delete`, `set-default`, `set-password`, `show` | Manage connection profiles                                         |
| [project](docs/documentation/project.md)             |        | `create`, `update`, `delete`                                           | Manage projects                                                    |
| [publish](docs/documentation/publish.md)             |        | `start`, `status`, `run`                                               | Publish CAI assets to the runtime                                  |
| [release](docs/documentation/release.md)             |        | `manifest`, `validate`, `plan`                                         | Generate multi-format release manifests and per-environment plans  |
| [role](docs/documentation/role.md)                   |        | `list`, `get`, `create`, `update`, `delete`                            | Manage roles                                                       |
| [runtime](docs/documentation/runtime.md)             | `rt`   | `list`, `get`, `create`, `update`                                      | Manage runtime environments                                        |
| [schedule](docs/documentation/schedule.md)           |        | `list`, `get`, `create`, `update`, `delete`                            | Manage schedules                                                   |
| [securitylog](docs/documentation/securitylog.md)     |        | `list`                                                                 | Query security audit log                                           |
| [sourcecontrol](docs/documentation/sourcecontrol.md) | `sc`   | `checkout`, `checkin`, `pull`, `commit`                                | Source control operations                                          |
| [state](docs/documentation/state.md)                 |        | `fetch`, `load`                                                        | Fetch/load object state snapshots                                  |
| [tag](docs/documentation/tag.md)                     |        | `assign`, `remove`                                                     | Assign/remove tags on objects                                      |
| [unpublish](docs/documentation/unpublish.md)         |        | `start`, `status`, `run`                                               | Unpublish CAI assets from the runtime                              |
| [user](docs/documentation/user.md)                   |        | `list`, `get`, `create`, `update`, `delete`                            | Manage users                                                       |
| [usergroup](docs/documentation/usergroup.md)         | `ug`   | `list`, `get`, `create`, `update`, `delete`                            | Manage user groups                                                 |

> **Keeping completions up to date:** After adding or changing any command or flag, regenerate
> the shell completion scripts by running `make completions`. The pre-generated scripts live in
> the `completions/` directory and must be committed together with the code change.

### Global flags

| Flag         | Short | Description                                                                                |
|--------------|-------|--------------------------------------------------------------------------------------------|
| `--profile`  | `-p`  | Profile to use (overrides default)                                                         |
| `--output`   | `-o`  | Output format: `table` (default), `json`, `csv`, `yaml`                                    |
| `--verbose`  | `-v`  | Enable verbose output                                                                      |
| `--no-color` |       | Disable colored output and force `plain` table theme                                       |
| `--theme`    |       | Table theme: `default`, `minimal`, `compact`, `plain`, `markdown`, `gh` (overrides config) |
| `--config`   |       | Config file path (default `~/.iics/config.yaml`)                                           |
| `--debug`    |       | Print full HTTP request/response trace to stderr                                           |
| `--http-timeout` |   | Per-HTTP-request timeout in seconds (default `120`; overrides config `httpTimeout` / `IICS_HTTP_TIMEOUT` env var). Independent of `export`/`import` `--max-wait-time`, which only bounds job polling. |

## Development

### Prerequisites

- Go 1.25+
- Make

### Build and test

```bash
# Build
make build

# Run tests
make test

# Run linter
make lint

# Run all checks
make all
```

### Install from Build

```bash
make install
```

Compiles the binary and installs it to `$GOPATH/bin` (typically `~/go/bin`), making `iics` available system-wide without specifying a path. The build injects the current git tag or commit SHA as the version string via `-ldflags`, and strips debug symbols (`-s -w`) to reduce binary size.

### Project structure

See [docs/DESIGN.md](docs/DESIGN.md) for the full design document.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.
