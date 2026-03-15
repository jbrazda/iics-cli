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

Requires Go 1.23 or later.

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
profiles:
  dev:
    name: "Development Org"
    region: "us"
    username: "user@company.com"
    password: ""
  prod:
    name: "Production Org"
    region: "EMEA"
    username: "admin@company.com"
    password: ""
    loginUrl: "https://dm-em.informaticacloud.com/saas/public/core/v3/login"
```

### Environment variable overrides

| Variable         | Description                    |
| ---------------- | ------------------------------ |
| `IICS_PROFILE`   | Override default profile       |
| `IICS_USERNAME`  | Override profile username      |
| `IICS_PASSWORD`  | Override profile password      |
| `IICS_REGION`    | Override profile region        |
| `IICS_LOGIN_URL` | Override computed login URL    |
| `IICS_OUTPUT`    | Override default output format |

Environment variables take precedence over config file values.

### Supported regions

| Region                                 | ssLogin Host                |
| -------------------------------------- | --------------------------- |
| US, USW1, USE2, USW3, USE4, USW5, USE6 | dm-us.informaticacloud.com  |
| USW1-1, USW3-1                         | dm1-us.informaticacloud.com |
| USW1-2                                 | dm2-us.informaticacloud.com |
| CAC1                                   | dm-na.informaticacloud.com  |
| APSE1, APJ                             | dm-ap.informaticacloud.com  |
| APNE1                                  | dm1-ap.informaticacloud.com |
| EMEA, EMWE1                            | dm-em.informaticacloud.com  |

## Commands

| Command | Alias | Subcommands | Description |
| ------- | ----- | ----------- | ----------- |
| [profile](docs/documentation/profile.md) | | `add`, `list`, `delete`, `set-default`, `show` | Manage connection profiles |
| [login](docs/documentation/login.md) | | | Authenticate and cache session |
| [logout](docs/documentation/logout.md) | | | Invalidate session |
| [objects](docs/documentation/objects.md) | | `list`, `dependencies` | List/search assets, find dependencies |
| [lookup](docs/documentation/lookup.md) | | | Resolve object IDs, names, and paths |
| [connection](docs/documentation/connection.md) | `conn` | `list`, `get`, `create`, `update`, `delete` | Manage connections |
| [export](docs/documentation/export.md) | | `run`, `start`, `status`, `download`, `create` | Export asset packages |
| [import](docs/documentation/import.md) | `imp` | `run`, `upload`, `start`, `status`, `download-log` | Import asset packages |
| [package](docs/documentation/package.md) | | `expand`, `create` | Extract or assemble IICS export package files (local, no API) |
| [project](docs/documentation/project.md) | | `create`, `update`, `delete` | Manage projects |
| [folder](docs/documentation/folder.md) | | `create`, `update`, `delete` | Manage folders |
| [schedule](docs/documentation/schedule.md) | | `list`, `get`, `create`, `update`, `delete` | Manage schedules |
| [user](docs/documentation/user.md) | | `list`, `get`, `create`, `update`, `delete` | Manage users |
| [usergroup](docs/documentation/usergroup.md) | `ug` | `list`, `get`, `create`, `update`, `delete` | Manage user groups |
| [role](docs/documentation/role.md) | | `list`, `get`, `create`, `update`, `delete` | Manage roles |
| [privilege](docs/documentation/privilege.md) | | `list` | List available privileges |
| [runtime](docs/documentation/runtime.md) | `rt` | `list`, `get`, `create`, `update` | Manage runtime environments |
| [agent](docs/documentation/agent.md) | | `list`, `get`, `details`, `start`, `stop` | Manage Secure Agents |
| [tag](docs/documentation/tag.md) | | `assign`, `remove` | Assign/remove tags on objects |
| [permission](docs/documentation/permission.md) | `perm` | `get`, `set`, `delete` | Manage object-level permissions |
| [securitylog](docs/documentation/securitylog.md) | `auditlog` | `list` | Query security audit log |
| [metering](docs/documentation/metering.md) | | `get`, `download` | Query usage and metering data |
| [sourcecontrol](docs/documentation/sourcecontrol.md) | `sc` | `checkout`, `checkin`, `pull`, `commit` | Source control operations |
| [state](docs/documentation/state.md) | | `fetch`, `load` | Fetch/load object state snapshots |

### Global flags

| Flag         | Short | Description                                      |
| ------------ | ----- | ------------------------------------------------ |
| `--profile`  | `-p`  | Profile to use (overrides default)               |
| `--output`   | `-o`  | Output format: `table` (default), `json`, `csv`  |
| `--verbose`  | `-v`  | Enable verbose output                            |
| `--no-color` |       | Disable colored output                           |
| `--config`   |       | Config file path (default `~/.iics/config.yaml`) |
| `--debug`    |       | Print request body to stderr on API errors       |

## Development

### Prerequisites

- Go 1.23+
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
