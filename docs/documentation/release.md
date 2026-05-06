# release

Generate and validate CI/CD release manifest and plan files.

## Synopsis

```bash
iics release <subcommand> [flags]
```

## Subcommands

| Subcommand | Description                                                                                  |
|------------|----------------------------------------------------------------------------------------------|
| `manifest` | Generate `target/iics/conf/release_manifest.yaml` and `target/iics/logs/release_manifest.md` |
| `validate` | Validate release manifest schema and options                                                 |
| `plan`     | Generate per-environment package and publish CSV files from a release manifest               |

---

## release manifest

Generates release manifest files from either a PR description markdown file (Deployment Options section) or explicit flags.

### Flags

| Flag                   | Type        | Default       | Description                                                                         |
|------------------------|-------------|---------------|-------------------------------------------------------------------------------------|
| `--from-file`          | string      |               | PR description markdown file to parse                                               |
| `--output-root`        | string      | `target/iics` | Root output directory                                                               |
| `--mode`               | string      |               | Override mode: `full` or `tag-based`                                                |
| `--tag`                | string      |               | Deployment tag when mode is `tag-based`                                             |
| `--targets`            | string list |               | Target environments (e.g. `tst,qa`)                                                 |
| `--valid-targets`      | string      |               | Comma-separated allowlist for valid targets (overrides `IICS_VALID_DEPLOY_TARGETS`) |
| `--include-connectors` | bool        | `false`       | Include connector assets                                                            |
| `--connectors-only`    | bool        | `false`       | Connector-only deployment                                                           |
| `--exclude-file`       | string      |               | Regex policy file path                                                              |
| `--source`             | string      |               | Optional source identifier stored in manifest                                       |

### Example

```bash
iics release manifest \
  --from-file pr-description.md \
  --output-root target/iics
```

---

## release validate

Validates manifest schema version and normalized options.

### Flags

| Flag              | Type   | Default                                  | Description                                                                         |
|-------------------|--------|------------------------------------------|-------------------------------------------------------------------------------------|
| `--manifest`      | string | `target/iics/conf/release_manifest.yaml` | Manifest path                                                                       |
| `--valid-targets` | string |                                          | Comma-separated allowlist for valid targets (overrides `IICS_VALID_DEPLOY_TARGETS`) |

### Example

```bash
iics release validate --manifest target/iics/conf/release_manifest.yaml
```

---

## release plan

Reads `release_manifest.yaml` and generates deterministic per-environment plan files under `target/iics/conf/<env>/`.

For `tag-based` mode:

- `tag_build.package.csv`
- `publish_assets.csv`
- optional global `target/iics/conf/connectors.package.csv`

For `full` mode:

- copies `all_exclude_connections.package.csv` to each environment folder
- creates empty `publish_assets.csv` headers for each environment

### Flags

| Flag                    | Type   | Default                                      | Description                                                                         |
| ----------------------- | ------ | -------------------------------------------- | ----------------------------------------------------------------------------------- |
| `--manifest`            | string | `target/iics/conf/release_manifest.yaml`     | Manifest path                                                                       |
| `--output-root`         | string | `target/iics`                                | Root output directory                                                               |
| `--full-package-config` | string | `./conf/all_exclude_connections.package.csv` | Source config copied in full mode                                                   |
| `--valid-targets`       | string |                                              | Comma-separated allowlist for valid targets (overrides `IICS_VALID_DEPLOY_TARGETS`) |
| `--package-fields`      | string | `location,dependency,type,path`              | CSV fields for package files                                                        |
| `--publish-fields`      | string | `location,dependency`                        | CSV fields for publish files                                                        |

### Example

```bash
iics release plan \
  --manifest target/iics/conf/release_manifest.yaml \
  --output-root target/iics
```

---

## Environment variable overrides

| Variable                    | Description                                                                                                                                 |
| --------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| `IICS_VALID_DEPLOY_TARGETS` | Comma-separated valid target allowlist used by `release manifest`, `release validate`, and `release plan` when `--valid-targets` is not set |
