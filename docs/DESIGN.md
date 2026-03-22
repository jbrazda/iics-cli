# IICS CLI - Design Document

## Context

A comprehensive Go CLI tool (`iics`) to interact with the Informatica Intelligent Cloud Services (IICS) platform REST API v3. Provides a full-featured command-line interface covering all v3 API resources, enabling automation of IICS operations (asset management, export/import, user administration, etc.) from terminals and CI/CD pipelines.

- [API Documentation (PDF)](https://docs.informatica.com/content/dam/source/GUID-B/GUID-BA8ED0B4-AB32-46B1-A7B8-358FAB844D6B/64/en/IICS_November2025_(REST-API)Reference_en.pdf)
- [API Documentation (html)](https://docs.informatica.com/cloud-common-services/administrator/current-version/rest-api-reference/preface.html)

## Technology Choices

- **CLI Framework**: [Cobra](https://github.com/spf13/cobra) (with [Viper](https://github.com/spf13/viper) for config)
- **Config**: YAML config file (`~/.iics/config.yaml`) with env var overrides, multi-profile support
- **Output**: Human-readable table (default) + JSON (`--output json`)
- **HTTP Client**: Standard `net/http` (no external HTTP libs needed)
- **Testing**: `net/http/httptest` mocks

## Project Structure

```text
iics_cli/
├── main.go                          # Entry point: cmd.Execute()
├── go.mod / go.sum
├── Makefile
├── cmd/                             # Cobra command definitions (thin layer)
│   ├── root.go                      # Root cmd, global flags, config/client init
│   ├── login.go / logout.go
│   ├── objects.go                   # objects list, dependencies
│   ├── lookup.go
│   ├── connection.go                # connection list/get/create/update/delete
│   ├── export.go                    # export create/status/download
│   ├── import_.go                   # import upload/start/status
│   ├── schedule.go                  # schedule CRUD
│   ├── project.go / folder.go       # project/folder CRUD
│   ├── user.go / usergroup.go       # user/group CRUD
│   ├── role.go / privilege.go       # role CRUD, privilege list
│   ├── runtime.go / agent.go        # runtime env, secure agents
│   ├── tag.go / permission.go       # tags, object permissions
│   ├── securitylog.go / metering.go # audit logs, usage data
│   ├── sourcecontrol.go             # checkout/checkin/pull/commit
│   └── state.go                     # fetchState/loadState
├── internal/
│   ├── client/                      # IICS API client
│   │   ├── client.go                # HTTP client with session header + 401 retry
│   │   ├── auth.go                  # Login/Logout
│   │   ├── errors.go                # APIError, SessionExpiredError
│   │   ├── objects.go, lookup.go, connections.go, export.go, imports.go
│   │   ├── schedules.go, projects.go, folders.go
│   │   ├── users.go, usergroups.go, roles.go, privileges.go
│   │   ├── runtimes.go, agents.go, tags.go, permissions.go
│   │   ├── securitylogs.go, metering.go, sourcecontrol.go, state.go
│   │   └── *_test.go
│   ├── config/                      # Configuration management
│   │   ├── config.go                # Config struct, Viper init, profile resolution
│   │   ├── pods.go                  # POD URL registry (region -> login URL)
│   │   └── session.go               # File-based session cache
│   └── output/                      # Output formatting
│       ├── formatter.go             # Formatter interface + factory
│       ├── table.go                 # lipgloss table renderer
│       └── json.go                  # JSON renderer
└── testdata/                        # Sample config and test fixtures
```

## Command Hierarchy

```text
iics
├── login                     # Authenticate, store session
├── logout                    # Invalidate session
├── objects list|dependencies # List/search assets, find deps
├── lookup                    # Resolve object IDs/names/paths
├── connection list|get|create|update|delete
├── export create|status|download
├── import upload|start|status
├── schedule list|get|create|update|delete
├── project create|update|delete
├── folder create|update|delete
├── user list|get|create|update|delete
├── usergroup list|get|create|update|delete
├── role list|get|create|update|delete
├── privilege list
├── runtime list|get|create|update
├── agent list|start|stop
├── tag assign|remove
├── permission get|set|delete
├── securitylog list
├── metering get|download
├── sourcecontrol checkout|checkin|pull|commit
└── state fetch|load

Global flags: --profile, --output (table|json), --region, --verbose, --no-color, --config
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

Session cache at `~/.iics/sessions.yaml` (ephemeral, never version-controlled).

## Key Architecture Decisions

1. **Session auto-refresh**: `Client.do()` intercepts 401 responses, re-authenticates, and retries once. Transparent to callers.
2. **Session file cache**: Reuses sessions across CLI invocations within 30-min window to avoid login on every command.
3. **`internal/` not `pkg/`**: CLI is not a library; the compiler enforces this boundary.
4. **`--from-file` for create/update**: Complex resources (connections with 50+ connector-specific params) accept a JSON file rather than dozens of flags.
5. **`map[string]interface{}` for ConnParams**: Connection parameters vary by connector type; typed structs are impractical for 100+ types.
6. **Confirmation prompts** for delete operations; `--yes` flag to skip in automation.

## Supported IICS Regions

| Region                                 | Login Host                  |
| -------------------------------------- | --------------------------- |
| US, USW1, USE2, USW3, USE4, USW5, USE6 | dm-us.informaticacloud.com  |
| USW1-1, USW3-1                         | dm1-us.informaticacloud.com |
| USW1-2                                 | dm2-us.informaticacloud.com |
| CAC1                                   | dm-na.informaticacloud.com  |
| APSE1, APJ                             | dm-ap.informaticacloud.com  |
| APNE1                                  | dm1-ap.informaticacloud.com |
| EMEA, EMWE1                            | dm-em.informaticacloud.com  |
