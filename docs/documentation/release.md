# release

Generate and validate CI/CD release manifest and plan files.

## Synopsis

```bash
iics release <subcommand> [flags]
```

## Subcommands

| Subcommand | Description                                                                                                |
|------------|------------------------------------------------------------------------------------------------------------|
| `manifest` | Generate `target/iics/import/conf/release_manifest.yaml` and `target/iics/import/logs/release_manifest.md` |
| `validate` | Validate release manifest schema and options                                                               |
| `plan`     | Generate per-environment package and publish CSV files from a release manifest                             |

---

## release manifest

Generates release manifest files from either a PR description markdown file (Deployment Options section) or explicit flags.

### Flags

| Flag                   | Type        | Default              | Description                                                                         |
|------------------------|-------------|----------------------|-------------------------------------------------------------------------------------|
| `--from-file`          | string      |                      | PR description markdown file to parse                                               |
| `--output-root`        | string      | `target/iics/import` | Root output directory                                                               |
| `--mode`               | string      |                      | Override mode: `full` or `tag-based`                                                |
| `--tag`                | string      |                      | Deployment tag when mode is `tag-based`                                             |
| `--targets`            | string list |                      | Target environments (e.g. `tst,qa`)                                                 |
| `--valid-targets`      | string      |                      | Comma-separated allowlist for valid targets (overrides `IICS_VALID_DEPLOY_TARGETS`) |
| `--include-connectors` | bool        | `false`              | Include connector assets                                                            |
| `--connectors-only`    | bool        | `false`              | Connector-only deployment                                                           |
| `--exclude-file`       | string      |                      | Regex policy file path                                                              |
| `--source`             | string      |                      | Optional source identifier stored in manifest                                       |

### Example

```bash
iics release manifest \
  --from-file pr-description.md \
  --output-root target/iics/import
```

---

## release validate

Validates manifest schema version and normalized options.

### Flags

| Flag              | Type   | Default                                         | Description                                                                         |
|-------------------|--------|-------------------------------------------------|-------------------------------------------------------------------------------------|
| `--manifest`      | string | `target/iics/import/conf/release_manifest.yaml` | Manifest path                                                                       |
| `--valid-targets` | string |                                                 | Comma-separated allowlist for valid targets (overrides `IICS_VALID_DEPLOY_TARGETS`) |

### Example

```bash
iics release validate --manifest target/iics/import/conf/release_manifest.yaml
```

---

## release plan

Reads `release_manifest.yaml` and generates deterministic per-environment plan files under `target/iics/import/<env>/`.

With `--verbose`, INFO logs include:

- step-by-step progress
- dependency status table matching `objects dependencies --targets` style, with
  per-target `STATUS` columns
- per-target package/publish output file paths
- grouped package/publish counts rendered as themed tables (type + total)
- target validation in status tables uses the same target-profile resolution rules as
  `--add-missing-transitive-deps` (`--target-profile-map`, env map, case-insensitive profile match)

For `tag-based` mode:

- `tag_build.package.csv`
- `publish_assets.csv` (ordered for publish execution: `AI_SERVICE_CONNECTOR`,
  `AI_CONNECTION`, `PROCESS`, `GUIDE`, `TASKFLOW`, matching `publish.md`)
- optional global `target/iics/import/connectors.package.csv`
- with `--add-missing-transitive-deps`: explicit assets are always included; transitive
  assets are included only when missing in the specific target environment profile

For `full` mode:

- copies `all_exclude_connections.package.csv` to each environment folder
- creates empty `publish_assets.csv` headers for each environment

### Flags

| Flag                            | Type   | Default                                         | Description                                                                                                              |
|---------------------------------|--------|-------------------------------------------------|--------------------------------------------------------------------------------------------------------------------------|
| `--manifest`                    | string | `target/iics/import/conf/release_manifest.yaml` | Manifest path                                                                                                            |
| `--output-root`                 | string | `target/iics/import`                            | Root output directory                                                                                                    |
| `--full-package-config`         | string | `./conf/all_exclude_connections.package.csv`    | Source config copied in full mode                                                                                        |
| `--valid-targets`               | string |                                                 | Comma-separated allowlist for valid targets (overrides `IICS_VALID_DEPLOY_TARGETS`)                                      |
| `--target-profile-map`          | string |                                                 | Comma-separated mapping `TARGET=profile` used for target org credential resolution (overrides `IICS_TARGET_PROFILE_MAP`) |
| `--add-missing-transitive-deps` | bool   | `false`                                         | Include transitive dependencies only when missing in each target environment; explicit assets are always included        |
| `--package-fields`              | string | `location,dependency,type,path`                 | CSV fields for package files                                                                                             |
| `--publish-fields`              | string | `location,dependency`                           | CSV fields for publish files                                                                                             |

### Example

```bash
iics release plan \
  --manifest target/iics/import/conf/release_manifest.yaml \
  --output-root target/iics/import

# Show INFO step summaries and dependency table
iics --verbose release plan \
  --manifest target/iics/import/conf/release_manifest.yaml \
  --output-root target/iics/import

# Include transitive dependencies only when missing in target environments
iics release plan \
  --manifest target/iics/import/conf/release_manifest.yaml \
  --output-root target/iics/import \
  --add-missing-transitive-deps

# Override target profile mapping for envs
iics release plan \
  --manifest target/iics/import/conf/release_manifest.yaml \
  --target-profile-map TST=tst-ci,QA=qa-ci,PROD=prod-ci \
  --add-missing-transitive-deps
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
