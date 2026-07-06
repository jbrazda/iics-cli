# release

Generate and validate CI/CD release manifest and plan files.

## Synopsis

```bash
iics release <subcommand> [flags]
```

## Subcommands

| Subcommand | Description                                                                                                |
|------------|------------------------------------------------------------------------------------------------------------|
| `manifest` | Generate `release_manifest.<ext>` (`yaml`, `json`, or `properties`) and `release_manifest.md`              |
| `validate` | Validate release manifest schema and options                                                               |
| `plan`     | Generate per-environment package and publish files (`csv`, `json`, or `yaml`) from a release manifest      |

---

## release manifest

Generates release manifest files from either a PR description markdown file (Deployment Options section) or explicit flags.

### Flags

| Flag                   | Type        | Default              | Description                                                                         |
|------------------------|-------------|----------------------|-------------------------------------------------------------------------------------|
| `--from-file`          | string      |                      | PR description markdown file to parse                                               |
| `--output-root`        | string      | `target/iics/import` | Root output directory                                                               |
| `--output`             | string      | `yaml`               | Manifest output format: `yaml`, `json`, `properties`                               |
| `--mode`               | string      |                      | Override mode: `full` or `tag-based`                                                |
| `--tag`                | string      |                      | Deployment tag when mode is `tag-based`                                             |
| `--targets`            | string list |                      | Target environments (e.g. `tst,qa`)                                                 |
| `--valid-targets`      | string      |                      | Comma-separated allowlist for valid targets (overrides `IICS_VALID_DEPLOY_TARGETS`) |
| `--include-connectors` | bool        | `false`              | Include connector assets                                                            |
| `--include-connections` | bool       | `false`              | Include connection assets                                                           |
| `--exclude-file`       | string      |                      | Regex policy file path                                                              |
| `--source`             | string      |                      | Optional source identifier stored in manifest                                       |

### Example

```bash
iics release manifest \
  --from-file pr-description.md \
  --output-root target/iics/import

# Generate Java properties output for Ant/PowerShell build scripts
iics release manifest \
  --from-file pr-description.md \
  --output properties

# Include connectors but not connections
iics release manifest \
  --from-file pr-description.md \
  --include-connectors
```

---

## release validate

Validates manifest schema version and normalized options.

### Flags

| Flag              | Type   | Default                                         | Description                                                                         |
|-------------------|--------|-------------------------------------------------|-------------------------------------------------------------------------------------|
| `--manifest`      | string | `target/iics/import/conf/release_manifest.yaml` | Manifest path (`.yaml`, `.json`, `.properties`), or `-` for stdin                 |
| `--valid-targets` | string |                                                 | Comma-separated allowlist for valid targets (overrides `IICS_VALID_DEPLOY_TARGETS`) |

### Example

```bash
iics release validate --manifest target/iics/import/conf/release_manifest.yaml

# Validate JSON manifest
iics release validate --manifest target/iics/import/conf/release_manifest.json

# Validate properties manifest from stdin (auto-detect content type)
cat target/iics/import/conf/release_manifest.properties | iics release validate --manifest -
```

---

## release plan

Reads `release_manifest` input (`yaml`, `json`, or `properties`) and generates deterministic per-environment plan files under `target/iics/import/<env>/`.

With `--verbose`, INFO logs include:

- step-by-step progress
- dependency status table matching `objects dependencies --targets` style, with
  per-target `STATUS` columns
- per-target package/publish output file paths
- grouped package/publish counts rendered as themed tables (type + total)
- target validation in status tables uses the same target-profile resolution rules as
  `--add-missing-transitive-deps` (`--target-profile-map`, env map, case-insensitive profile match)

For `tag-based` mode:

- `tag_build.package.<ext>`
- `publish_assets.<ext>` (ordered for publish execution: `AI_SERVICE_CONNECTOR`,
  `AI_CONNECTION`, `PROCESS`, `GUIDE`, `TASKFLOW`, matching `publish.md`)
- optional global `target/iics/import/connectors.package.<ext>` when connectors
  or connections are included
- each per-environment `tag_build.package.<ext>` includes `STATUS (<env>)` for that file's environment
- by default, explicit assets are always included and transitive assets are
  included only when missing in the specific target environment profile
- use `--skip-missing-transitive-deps` to keep all transitive assets

For `full` mode:

- resolves dependencies from `--full-package-config` seed rows (`id`, `location`, or `path+type`)
- when seed rows include `Project` or `Folder`, all nested objects are treated as explicit
- marks objects outside explicit scope as transitive
- writes `full_build.package.<ext>` per environment using `--output` format
- each per-environment `full_build.package.<ext>` includes `STATUS (<env>)` for that file's environment
- writes `publish_assets.<ext>` per environment from all publishable resolved full-package assets
  (`AI_SERVICE_CONNECTOR`, `AI_CONNECTION`, `PROCESS`, `GUIDE`, `TASKFLOW`)
- writes optional global `target/iics/import/connectors.package.<ext>` when
  connectors or connections are included

### How dependency resolution works

#### Tag-based mode

1. Queries the source org with `ListAllObjects` using `tag=='<tag>'` to find all explicitly-tagged objects.
2. For each tagged object ID, runs a BFS traversal (`TraverseByIDs`) that repeatedly calls
   the `GetObjectDependencies` API with `refType=uses`.
   - References that carry an ID are queued directly.
   - References with only `path+type` are resolved to IDs via `Lookup`.
3. After the graph is fully traversed, a bulk `Lookup` by ID populates path/type metadata for
   every visited node.
4. Each resulting asset is labelled `explicit` (directly tagged) or `transitive` (pulled in as a
   dependency).

#### Full deployment mode

1. Reads a seed config file (`--full-package-config`) containing rows with `id`, `location`, or
   `path+type`.
2. Entries that have only `path+type` are resolved to IDs via `Lookup`.
3. Seeds of type `Project` or `Folder` are **expanded recursively**: the CLI calls
   `ListAllObjects` with `location=='<path>'` for every container root, and all contained
   objects are treated as `explicit` seeds.
4. The same BFS traversal as tag-based mode then resolves transitive dependencies from all
   explicit seed IDs.

### How target org presence is checked

Target credentials are resolved in this order:

1. `--target-profile-map` flag or `IICS_TARGET_PROFILE_MAP` env var (e.g. `TST=my-tst-profile,PROD=prod`).
2. Profile in `~/.iics/config.yaml` with a name matching the target (case-insensitive; `STG` and
   `STAGE` are treated as aliases).
3. CI env vars `IICS_USER_<TARGET>` + `IICS_PWD_<TARGET>` with optional
   `IICS_LOGIN_URL_<TARGET>` / `IICS_REGION_<TARGET>`.
4. Global fallback `IICS_TARGET_USERNAME` + `IICS_TARGET_PASSWORD`.

Two checks are performed against the resolved target client:

#### Missing-transitive filter (default enabled)

- Missing-transitive filtering is enabled by default.
- Applied only to **transitive** assets; explicit assets are always included.
- For each transitive asset, calls `assetExistsInTarget` (see below).
- Transitive assets that are **missing** in the target are **included** in the plan (they need
  to be deployed); assets that **already exist** are **excluded** (assumed stable).
- Use `--skip-missing-transitive-deps` to disable this filtering.

#### Per-asset validation (always runs)

- Checks every asset and annotates the output with `Status` and `Warning` values, written as a
  target-specific column (`STATUS (tst)`, `STATUS (prod)`, etc.) in each per-environment package
  file.

#### `assetExistsInTarget` lookup logic

| Asset type   | API call                                                                          | Treated as absent when                              |
|--------------|-----------------------------------------------------------------------------------|-----------------------------------------------------|
| `Connection` | `GetConnectionByName` using the last path segment                                 | 404 or `APP_13436` (no object named connection)     |
| All others   | `Lookup` with `{Path, Type}`                                                      | `V3API_LookupError_012` error code in response body |

### Flags

| Flag                            | Type   | Default                                         | Description                                                                                                              |
|---------------------------------|--------|-------------------------------------------------|--------------------------------------------------------------------------------------------------------------------------|
| `--manifest`                    | string | `target/iics/import/conf/release_manifest.yaml` | Manifest path                                                                                                            |
| `--output-root`                 | string | `target/iics/import`                            | Root output directory                                                                                                    |
| `--full-package-config`         | string | `./conf/full_build.package.csv`                 | Full mode seed config (`id`, `location`, or `path+type`) used to resolve package assets                                |
| `--valid-targets`               | string |                                                 | Comma-separated allowlist for valid targets (overrides `IICS_VALID_DEPLOY_TARGETS`)                                      |
| `--target-profile-map`          | string |                                                 | Comma-separated mapping `TARGET=profile` used for target org credential resolution (overrides `IICS_TARGET_PROFILE_MAP`) |
| `--add-missing-transitive-deps` | bool   | `true`                                          | Legacy control for missing-transitive filtering; default behavior is enabled                                               |
| `--skip-missing-transitive-deps` | bool  | `false`                                         | Disable missing-transitive filtering and keep all transitive dependencies                                                   |
| `--output`                      | string | `csv`                                           | Plan file output format: `csv`, `json`, `yaml`                                                                           |
| `--package-fields`              | string | `location,type,path,dependency`                 | Fields for package files; `STATUS (<env>)` is auto-added per environment file                                            |
| `--publish-fields`              | string | `location,type,path,dependency`                 | Fields for publish files                                                                                                 |
| `--log-file`                    | string | disabled; default path when flag has no value   | Append a Markdown release report section; use `--log-file=PATH` to override `target/iics/import/logs/release_manifest.md` |

### Example

```bash
iics release plan \
  --manifest target/iics/import/conf/release_manifest.yaml \
  --output-root target/iics/import

# Show INFO step summaries and dependency table
iics --verbose release plan \
  --manifest target/iics/import/conf/release_manifest.yaml \
  --output-root target/iics/import

# Include transitive dependencies only when missing in target environments (default)
iics release plan \
  --manifest target/iics/import/conf/release_manifest.yaml \
  --output-root target/iics/import

# Override target profile mapping for envs
iics release plan \
  --manifest target/iics/import/conf/release_manifest.yaml \
  --target-profile-map tst=tst-ci,qa=qa-ci,prod=prod-ci \
  --skip-missing-transitive-deps

# Generate JSON plan files
iics release plan \
  --manifest target/iics/import/conf/release_manifest.properties \
  --output json
```

---

## Environment variable overrides

| Variable                    | Description                                                                                                                                 |
|-----------------------------|---------------------------------------------------------------------------------------------------------------------------------------------|
| `IICS_VALID_DEPLOY_TARGETS` | Comma-separated valid target allowlist used by `release manifest`, `release validate`, and `release plan` when `--valid-targets` is not set |
| `IICS_TARGET_PROFILE_MAP`   | Comma-separated target profile map used by `release plan` when `--target-profile-map` is not set                                            |
| `IICS_USER_<TARGET>`        | CI username for target when profile is not configured (for example `IICS_USER_TST`)                                                         |
| `IICS_PWD_<TARGET>`         | CI password for target when profile is not configured (for example `IICS_PWD_TST`)                                                          |
| `IICS_REGION_<TARGET>`      | Optional target region used with `IICS_USER_<TARGET>` and `IICS_PWD_<TARGET>`                                                               |
| `IICS_LOGIN_URL_<TARGET>`   | Optional target login URL used with `IICS_USER_<TARGET>` and `IICS_PWD_<TARGET>`                                                            |
