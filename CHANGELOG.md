# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed

- `package create --exclude-found-transitive` now builds
  `exportMetadata.v2.json` from selected assets plus dependency closure and
  required containers, while still excluding filtered transitive-found
  dependencies from package content
- `release plan` now defaults missing-transitive filtering to enabled so
  per-target package manifests are authoritative by default

### Changed

- `release plan` replaced `--add-missing-transitive-deps` and
  `--skip-missing-transitive-deps` with a single opt-out flag
  `--include-found-transitive` that keeps all resolved transitive dependencies
  in generated package files
- `package create` now uses CDI asset `objectRefs` from source metadata to
  retain required `Connection` and `AgentGroup` references for selected CDI
  assets, without hidden payload parsing

## [0.4.10] - 2026-07-01

### Fixed

- `package create --exclude-found-transitive` now regenerates
  `exportMetadata.v2.json` from the selected package assets instead of
  retaining the full source metadata object list

## [0.4.9] - 2026-07-01

### Fixed

- `release plan` now generates `connectors.package.<ext>` files from the
  resolved dependency tree instead of per-target missing-transitive filtered
  assets, keeping connector and connection dependencies in the standalone
  connectors package even when they already exist in target orgs

## [0.4.8] - 2026-07-01

### Fixed

- `release plan` target dependency validation no longer fails when a missing
  connection lookup returns `APP_13436` (HTTP 403); it is now classified as a
  missing dependency and planning continues
- `--debug` now emits request and response traces for target-org validation
  calls used by release planning
- Dependency status tables are now sorted by `LOCATION`
- `connectors.package.<ext>` generation is now consistently policy-driven by
  the manifest `Include Connectors` and `Include Connections` settings in both
  full and tag modes

## [0.4.7] - 2026-06-01

### Fixed

- `release plan` full mode now generates `connectors.package` files when
  connectors or connections are included, and emits verbose dependency and
  package composition output to match tag-based planning behavior

## [0.4.6] - 2026-06-01

### Added

- `release manifest` now writes a Markdown log file alongside the JSON manifest,
  summarising the release plan in a human-readable format

### Changed

- Removed the redundant `release manifest --connectors-only` option; use
  `--include-connectors` and `--include-connections` to include both asset types

### Fixed

- `release plan` full mode now writes environment `publish_assets` files from
  resolved publishable package assets instead of creating empty files

## [0.4.5] - 2026-05-29

### Fixed

- Full package config now supports single-column CSV headers (files with only an `assetPath`
  column and no `targetEnvironments` column) without failing on missing column index

### Changed

- Updated release manifest samples

## [0.4.4] - 2026-05-22

### Changed

- `manifest.options.targets` values are now normalized to lower case (`tst`, `qa`, `stg`, `prod`)
  for consistency with generated folder paths, status field names, and profile names. Existing
  manifests with upper case targets continue to parse correctly. CI env var names
  (`IICS_USER_TST`, `IICS_PWD_TST`, etc.) are unaffected.

## [0.4.3] - 2026-05-21

### Changed

- Full-build dependency resolution improved: more accurate transitive dependency tracking
  and cleaner status output during release plan generation
- Duplicate dependency warnings are now deduplicated and consolidated in release output
- Release config documentation aligns `full_build.package.csv` as the canonical config name

## [0.4.2] - 2026-05-19

### Changed

- `release manifest` splits connector/connection include options into separate flags for
  finer control over deployment scope
- Release manifest output cleanup: improved formatting of generated markdown logs

### Fixed

- `user get --id` now correctly scans the user list instead of calling the unsupported
  `GET /users/{id}` endpoint (which only allows DELETE), resolving HTTP 405 errors (BUG-0004)

### Docs

- Refreshed command reference pages: `metering`, `objects`, `publish`, `release`

## [0.4.1] - 2026-05-13

### Added

- `package create --include-tags` preserves tag propagation in regenerated
  `exportMetadata.v2.json` when using manifest-driven selective packaging (CR-0023)

### Changed

- `package create --exclude-found-transitive` now preserves the full CDI dependency
  graph in `exportMetadata.v2.json` while still excluding transitive asset files from
  the package content, preventing broken metadata for downstream imports (CR-0023)
- Selective packaging enforces strict single-asset-per-manifest-entry behavior; duplicate
  manifest entries are deduplicated and reported (CR-0023)

## [0.4.0] - 2026-05-11

### Added

- `expand` command strips `CurrentServerDateTime` elements from XML assets on expand,
  preventing spurious diffs in source-controlled packages (CR-0005)
- `objects dependencies` now accepts JSON, CSV, and YAML input via stdin for seed IDs,
  supports `--targets` report across multiple profiles, and `--publish` to trigger
  publication after dependency resolution (CR-0019)
- Release plan output enforces publish order for dependent assets, ensuring correct
  sequencing in verbose plan tables
- Target-aware missing transitive dependency filtering: dependencies absent in a given
  target org are filtered per-target rather than globally
- Deploy target validation is now configurable via release manifest options
- Dependency resolution enhancements: improved output export and connection lookup accuracy
- `package create --manifest-file/-m` for manifest-driven selective packaging: accepts
  JSON, CSV, YAML, or TXT format with format auto-detection; stdin pipe supported (CR-0023)
- `package create --name/-n` overrides the default package name used in generated
  `ContentsofExportPackage_<name>.csv` (CR-0023)
- `package create --exclude-found-transitive` combined with `--status-target` filters
  out transitive dependencies already present in the target org from the build manifest (CR-0023)
- Artifact format detection centralized; YAML stdin auto-detection added (CR-0023)

### Fixed

- Export artifact key lookup reconciliation now correctly matches artifacts when keys are
  reordered or partially specified
- `objects list`, `objects dependencies`, and `package dependencies` now build location
  strings with `SYS/` prefix for Connection and AgentGroup assets instead of the
  incorrect `Explore/` prefix (BUG-0007)
- `objects dependencies` default limit corrected; auto-pagination added; `location` field
  now included in all dependency output (BUG-0006)
- `ObjectDependenciesResponse` struct corrected to match actual IICS v3 API response shape
- `ObjectReference` field tags and deduplication key fixed
- Interactive `user create`/`user update` wizard now paginates groups and roles to avoid
  hitting the API maximum limit
- Replaced deprecated `reflect.Ptr` with `reflect.Pointer` (golangci-lint)

### Changed

- Go and GitHub Actions dependency bumps (group updates #18 and #19)

## [0.3.2] - 2026-04-09

### Added

- `user create` and `user update` interactive wizard (`--interactive`) with prompts for all
  user fields including authentication type, roles, and groups (CR-0018)
- `user list` now supports `--limit` and `--skip` for pagination (CR-0018)
- `user get` supports `--username` flag for lookup by userName (CR-0018)

### Fixed

- `user list` and `role list` capped at 200 results with automatic pagination to avoid
  exceeding the API maximum `limit` parameter (BUG-0005)
- Interactive user wizard `State` prompt no longer shadows the outer `err` variable,
  eliminating a golangci-lint warning

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
