# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-03-23

### Added

- `profile edit` subcommand: interactively update credentials for an existing profile;
  validates by logging in, derives org-specific `baseApiUrl`/`caiUrl` from the login
  response, and refreshes the session cache

### Fixed

- `config.ResolveProfile` now returns a copy of the stored profile before applying
  `IICS_*` env-var overrides; previously the mutation caused env-var values to be written
  to `~/.iics/config.yaml` by `iics login`, potentially corrupting stored credentials
  and causing account lock-outs (IICS error IDS_086)
- `iics login`: discovered URLs (`loginUrl`, `baseApiUrl`, `caiUrl`) are now written to
  the original stored profile, not to the env-var-overridden copy
- `profile add` and `profile edit`: username and password prompts now always appear even
  when an existing profile already has those fields populated (previously the loops were
  skipped, so editing credentials was impossible without manual YAML editing)
- Auto-setup wizard showed `""` instead of the actual profile name when credentials were
  not configured for a named `--profile`

## [0.1.0-beta] - 2026-03-22

### Added

- `profile` command: `add`, `list`, `delete`, `set-default`, `show` subcommands; interactive
  setup wizard with masked password input; auto-triggers on first run when no credentials are
  configured (TTY only)
- `package` command: `expand` and `create` subcommands for local IICS ZIP inspection and
  assembly (no API calls required)
- `activitylog` command: `list` and `get` subcommands with field selection, human-readable
  state labels, and nested section display
- `user change-password` and `user reset-password` subcommands
- `publish` and `unpublish` commands for CAI asset deployment: `start`, `status`, `run`
  subcommands; `--ids` flag accepts comma-separated IDs or a file path (`.txt`, `.csv`,
  `.json`, `.yaml`); multi-batch result aggregation with per-asset status summary
- `export run` and `import run` one-shot convenience commands (upload + start + poll in one step)
- Verbose export mode (`--verbose`) prints a three-column artifact table (`ID`, `PATH`, `TYPE`)
  to stderr showing the assets included in the package
- CAI URL auto-derived from the login response and persisted to the profile on first login
- Login URL auto-derived from region code - no manual URL configuration required
- Table output themes via `charmbracelet/lipgloss`: `default` (Unicode rounded borders, cyan
  bold headers), `minimal` (underline separator, colored headers), `compact` (bold headers,
  no borders), `plain` (ASCII borders, no color - used automatically for non-TTY output)
- `style.theme` and `style.noColor` keys in `~/.iics/config.yaml` to persist the preferred theme
- `--no-color` global flag and `NO_COLOR` environment variable disable all color output
- Shell completion scripts for bash, zsh, fish, and PowerShell (`iics completion <shell>`)
- Structured logging via `log/slog`; `--verbose` enables info-level progress, `--debug` enables
  full HTTP request/response trace to stderr
- `usergroup list --query` / `-q` flag for server-side filtering
  (e.g. `userGroupName=="Administrator"`)
- `usergroup list --fields` flag for custom column selection

### Fixed

- `schedules list` response now correctly unwraps the `schedules` wrapper key
- `securitylog list` endpoint path, response wrapper struct, and field mapping corrected
- `objects list` query filter separator changed from `,` to `;` (IICS API requirement);
  removed broken `--tag` flag (tag filtering is done via the `q` expression)
- `import status` table corrected to show `source` and `target` columns instead of duplicating
  the same field
- `import upload` now establishes the session before constructing the upload URL
- `publish`/`unpublish` HTTP 500 resolved by adding `Accept: application/json` header to
  CAI requests
- `Accept` header now preserved across 401 re-authentication retry
- `publish`/`unpublish` DTEMPLATE and PROCESS_OBJECT asset path and URL branching restored
- `usergroup list` NAME column was blank due to wrong JSON tag (`name` vs `userGroupName`)

### Changed

- Table renderer replaced: `github.com/olekukonko/tablewriter` removed, replaced with custom
  `charmbracelet/lipgloss` renderer supporting configurable themes and TTY-aware color fallback
- `usergroup list` default columns now show `countMembers` and `countRoles` instead of
  `description`
- Minimum Go version bumped to 1.25
- CI matrix reduced to Go 1.25 only

### Added (initial release)

- Full IICS REST API v3 coverage: objects, connections, schedules, export/import, users,
  roles, permissions, runtime environments, agents, tags, source control, state snapshots
- Multi-profile configuration (`~/.iics/config.yaml`) with environment variable overrides
  (`IICS_USERNAME`, `IICS_PASSWORD`, `IICS_REGION`, `IICS_LOGIN_URL`, `IICS_CAI_URL`)
- Session caching with 30-minute expiry and automatic 401 retry with re-login
- Table, JSON, CSV, and YAML output formats (`--output` flag)
- Cross-platform builds: Linux, macOS, Windows (amd64, arm64)
- GitHub Actions CI pipeline with goreleaser release automation
- golangci-lint configuration
- Support for all IICS regions (US, USW1-USW5, USE2-USE6, EMEA, EMWE1, APJ, APSE1,
  APNE1, CAC1, and more)
