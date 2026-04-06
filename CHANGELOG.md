# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.3.1] - 2026-04-06

### Fixed

- `profile edit` and `profile add` on a keychain-backed profile (`password: "@keyring"`)
  corrupted the OS keychain by storing the literal sentinel string `"@keyring"` as the
  password value when the user pressed Enter to keep the existing password; subsequent
  login attempts returned HTTP 400 and could trigger IICS account lockout (BUG-0003)
- `profile edit` login validation used the sentinel string `"@keyring"` as the IICS
  password instead of retrieving the real password from the keychain first (BUG-0003)
- `package dependencies --target-profile` and `--report` now resolve the keychain sentinel
  in `ResolveTargetProfile`, fixing HTTP 401 on every dependency lookup for keychain-backed
  profiles (BUG-0003)
- `ResolveTargetProfile` now returns `profile "X" not found` instead of the misleading
  `username not configured` error when the profile name does not exist in the config file
  (BUG-0003)
- `package dependencies --report` now returns a clear error when pflag silently consumes
  the next flag token as its value (e.g. `--report --target-profile dev,qa`), with a hint
  pointing to the correct `--report=dev,qa` syntax (BUG-0003)
- `profile list` now shows a `KEYCHAIN` column indicating which profiles use OS keychain
  storage; `profile show` displays `*** (keychain)` for the password field (BUG-0003)
- Interactive password prompt now shows `[keychain]` instead of `[***]` when editing a
  keychain-backed profile, and skips the keyring re-offer when the password is kept
  unchanged (BUG-0003)

## [0.3.0] - 2026-04-05

### Added

- `package dependencies` subcommand: validates package assets against a target org, checking
  whether each referenced connection, runtime environment, and schedule exists (CR-0011)
- `package dependencies --report`: multi-profile dependency report across several orgs at once;
  renders a per-profile summary table (CR-0014)
- `auditlog list` command for the V2 audit log endpoint with field selection and filtering (CR-0015)
- Table output themes `markdown` (GitHub-flavored markdown) and `gh` (GitHub CLI style);
  row-count footer on all table output; `--theme` global flag and `IICS_THEME` environment
  variable for CI-friendly theme override; `style.headerColor` config field for custom header
  color (CR-0016)
- Secure credential storage via OS keychain using sentinel `password: "@keyring"` in config;
  `profile set-password` subcommand to migrate existing plaintext passwords to keychain;
  `profile add` and `profile edit` offer keyring storage as the last prompt step; `IICS_PASSWORD`
  env var always takes precedence over keychain (CR-0017)
- Consistent `[timestamp][LEVEL] message key=value` log format across all commands via a custom
  `slog.Handler`; `--verbose` enables INFO-level, `--debug` enables DEBUG-level (CR-0013)
- Connection lookup: auto-size table columns to content width; color-coded `TARGET STATUS`
  column (green/red/yellow); fixed `GetConnectionByName` lookup (CR-0012)

### Fixed

- Correct publish/unpublish sort order for dependent assets; added `--order-by` flag for
  `package dependencies` output (BUG-0001)
- Table column misalignment when cell content contained ANSI escape codes; column widths now
  measured on the visible (stripped) string length (BUG-0002)
- Runtime commands now use the V2 API with correct structs; added `--name` flag and tree-view
  output for runtime environments

## [0.2.0] - 2026-03-23

### Added

- `iics login --profile <name>`: when the named profile does not exist in config and stdin is
  a terminal, the command now offers to create it interactively before proceeding with login;
  non-TTY environments receive a clear error with the `iics profile add` hint
- `profile list` now includes a dedicated `REGION` column alongside the existing `ENDPOINT`
  column so both fields are always visible regardless of whether a login URL has been discovered
- `profile show` now includes session-derived fields read from the local cache: Org Name,
  Org ID, Session User, Last Login, and Session Expires; shows `(no active session)` when
  no cache entry exists
- `SessionEntry` in `~/.iics/sessions.yaml` now persists a `lastLoginTime` field recording
  when credentials were last used to authenticate

### Fixed

- Sessions were never written to `~/.iics/sessions.yaml` after auto-login or 401 renewal;
  every non-`login` command re-authenticated from scratch on each invocation. Fixed by
  wiring an `OnLoginSuccess` callback on `Client` that persists the session after every
  successful login (initial, auto, or 401 renewal)
- 401 renewal failure now surfaces a clear error: `session renewal failed: ...; run 'iics
  login' to re-authenticate` instead of the generic `session expired: ...`

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
