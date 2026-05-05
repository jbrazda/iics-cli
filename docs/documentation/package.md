# package

<!-- markdownlint-disable MD013 MD024 MD036 MD060 -->

Work with IICS export package files locally. No authentication or API calls are required - all
operations are performed on the local file system.

## Synopsis

```bash
iics package <subcommand> [flags]
```

## Subcommands

| Subcommand      | Description |
| --------------- | ----------- |
| `expand`        | Extract an IICS export ZIP package to a directory |
| `create`        | Assemble a new IICS export ZIP package from a directory |
| `dependencies`  | Resolve and list transitive dependencies of an export package |

---

## package expand

Extracts the contents of an IICS export ZIP package to a target directory. JSON files are
pretty-printed (2-space indent) to produce clean diffs in version control. Nested ZIP files
(e.g. `m_Test_Git.DTEMPLATE.zip`) can be recursively expanded into directories.

The typical workflow for version-controlling IICS assets is:

1. `iics export run` - export assets to a ZIP package
2. `iics package expand` - extract the package to a versioned directory
3. Commit the directory to git
4. `iics package create` - reassemble the ZIP from the directory
5. `iics import run` - import the package to a target org

### Flags

| Flag | Short | Type | Required | Default | Description |
| ---- | ----- | ---- | -------- | ------- | ----------- |
| `--file` | `-f` | string | yes | | Path to the source ZIP package |
| `--target` | `-t` | string | yes | | Destination directory; created if absent |
| `--recursive` | `-r` | bool | | false | Recursively expand nested ZIPs into `<name>.zip/` directories |
| `--clean` | `-c` | bool | | false | Delete target contents before expanding |
| `--yes` | `-y` | bool | | false | Suppress the `--clean` confirmation prompt |

All [global flags](../../README.md#global-flags) apply.

### Behavior details

**Target directory**

- Created automatically with `os.MkdirAll` if it does not exist.
- If the target contains files and `--clean` is not set, the command returns an error. This
  is intentional: in CI/CD pipelines the source directory should be empty so that deleted or
  renamed assets are detected by `git status`.
- With `--clean`, the command counts and removes existing contents, prompting for confirmation
  unless `--yes` is also given.

**JSON pretty-printing**

All `.json` files are reformatted with 2-space indentation on extraction. This produces
readable, diff-friendly output in git. It also means the SHA-256 hashes stored in
`exportPackage.chksum` will differ from the originals; `package create` always regenerates
the checksum from current file contents.

**Recursive expand (`--recursive`)**

Nested ZIP files (e.g. `Mappings/m_Test_Git.DTEMPLATE.zip`) are expanded into a directory
with the same name including the `.zip` extension
(e.g. `Mappings/m_Test_Git.DTEMPLATE.zip/`). The source zip file is not written to disk.
When `package create` encounters a directory whose name ends in `.zip`, it re-packs it into
a nested ZIP automatically.

**Verbose output (`--verbose`)**

Lists the ZIP entries before extraction and lists the target directory contents after.

### Examples

```bash
# Basic expand
iics package expand \
  --file exports/my-project.zip \
  --target src/iics

# Expand and overwrite an existing directory (with confirmation)
iics package expand \
  --file exports/my-project.zip \
  --target src/iics \
  --clean

# Non-interactive expand (CI/CD)
iics package expand \
  --file exports/my-project.zip \
  --target src/iics \
  --clean --yes

# Recursively expand all nested ZIPs (e.g. DTEMPLATE, MTT assets)
iics package expand \
  --file exports/my-project.zip \
  --target src/iics \
  --recursive --verbose
```

```powershell
# Basic expand
iics package expand `
  --file exports/my-project.zip `
  --target src/iics

# Expand and overwrite an existing directory (with confirmation)
iics package expand `
  --file exports/my-project.zip `
  --target src/iics `
  --clean

# Non-interactive expand (CI/CD)
iics package expand `
  --file exports/my-project.zip `
  --target src/iics `
  --clean --yes

# Recursively expand all nested ZIPs (e.g. DTEMPLATE, MTT assets)
iics package expand `
  --file exports/my-project.zip `
  --target src/iics `
  --recursive --verbose
```

---

## package create

Assembles a new IICS export ZIP package from the contents of a source directory and
regenerates `exportPackage.chksum` to match. The resulting ZIP is suitable for import
via `iics import run`.

Dot files (names starting with `.`) and any existing `exportPackage.chksum` are
excluded. Directories whose name ends in `.zip` (produced by `package expand --recursive`)
are re-packed into nested ZIP entries automatically.

### Flags

| Flag | Short | Type | Required | Default | Description |
| ---- | ----- | ---- | -------- | ------- | ----------- |
| `--source` | `-s` | string | yes | | Source directory to package |
| `--target` | `-t` | string | yes | | Output ZIP file path |
| `--force` | `-f` | bool | | false | Overwrite existing output file |

All [global flags](../../README.md#global-flags) apply.

### Checksum file

`exportPackage.chksum` is always regenerated from the actual files in the source directory.
The format matches the Informatica CLI v2 package format:

- One `path=SHA256` entry per file, sorted alphabetically
- Spaces in paths escaped as `\` followed by a space
- SHA-256 hash in uppercase hex
- No header or timestamp line

### Verbose output (`--verbose`)

Prints the list of files added to the ZIP before the summary line.

### Examples

```bash
# Basic create
iics package create \
  --source src/iics \
  --target dist/my-project-reimported.zip

# Overwrite existing output
iics package create \
  --source src/iics \
  --target dist/my-project.zip \
  --force --verbose
```

```powershell
# Basic create
iics package create `
  --source src/iics `
  --target dist/my-project-reimported.zip

# Overwrite existing output
iics package create `
  --source src/iics `
  --target dist/my-project.zip `
  --force --verbose
```

---

## Round-trip workflow

```bash
# 1. Export from source org
iics export run \
  --artifacts-file objects.txt \
  --export-file-path dist/export.zip \
  --profile dev

# 2. Expand to versioned directory
iics package expand \
  --file dist/export.zip \
  --target src/iics \
  --clean --yes

# 3. Commit to git
git add src/iics
git commit -m "Update IICS assets"

# 4. Reassemble package from versioned sources
iics package create \
  --source src/iics \
  --target dist/import.zip \
  --force

# 5. Import to target org
iics import run \
  --zip-file dist/import.zip \
  --profile prod
```

```powershell
# 1. Export from source org
iics export run `
  --artifacts-file objects.txt `
  --export-file-path dist/export.zip `
  --profile dev

# 2. Expand to versioned directory
iics package expand `
  --file dist/export.zip `
  --target src/iics `
  --clean --yes

# 3. Commit to git
git add src/iics
git commit -m "Update IICS assets"

# 4. Reassemble package from versioned sources
iics package create `
  --source src/iics `
  --target dist/import.zip `
  --force

# 5. Import to target org
iics import run `
  --zip-file dist/import.zip `
  --profile prod
```

---

## Package file format

An IICS export package is a ZIP archive with the following structure:

```text
<package-name>.zip
├── exportMetadata.v2.json        # Asset metadata with GUIDs, objectRefs, repoHandles
├── exportPackage.chksum          # SHA-256 checksums for all files
├── ContentsofExportPackage_*.csv # Asset inventory (present in server exports)
├── Explore/
│   └── <Project>/
│       ├── <Project>.Project.json
│       ├── <Folder>.Folder.json
│       └── <Folder>/
│           └── <Asset>.<TYPE>.xml  # or .zip for binary asset types
└── SYS/                          # Present when exported with dependencies
    └── <Connection>.<TYPE>.zip
```

Key files:

- **`exportMetadata.v2.json`** - mandatory for import; contains per-asset metadata
  including `repoInfo.repoHandle` and `objectRefs` arrays
- **`exportPackage.chksum`** - always regenerated by `package create`; IICS validates
  this on import

---

## package dependencies

Resolves and lists transitive dependencies of every object in an IICS export package.
The data source is `exportMetadata.v2.json` inside the package. No authentication is required
for package-only analysis; a source org profile is used when available to resolve external
GUIDs (assets referenced but not included in the package).
Traversal continues recursively until no new dependency assets are discovered.

`dependency=explicit` is based on object definition files present in
`exportPackage.chksum` (physically packaged assets). Metadata-only references that do not have
backing object files in checksum are reported as `dependency=transitive`.

The command operates in two modes:

**Default mode** (no `--publish`): lists ALL dependencies regardless of type, including
Folder, Project, Connection, AgentGroup, MTT assets, and more. Useful for auditing the full
dependency graph. `--target-profile` is optional; when provided, each dependency is validated
against the target org and missing assets are flagged. Output is sorted by `path`.

**Publish mode** (`--publish`): restricts output to publishable types only:
`AI_SERVICE_CONNECTOR`, `AI_CONNECTION`, `PROCESS`, `GUIDE`, `TASKFLOW`.
`--target-profile` or `--report` is required in this mode. Output is sorted by publish
dependency order (primary) then `path` (secondary):

In publish mode, recursion still traverses through non-publishable intermediate assets so
deeper publishable dependencies are included in the result.

| Order | Type |
| ----- | ---- |
| 1 | `AI_SERVICE_CONNECTOR` |
| 2 | `AI_CONNECTION` |
| 3 | `PROCESS` |
| 4 | `GUIDE` |
| 5 | `TASKFLOW` |

Use `--order-by` to override the default sort in either mode.

### Flags

| Flag | Short | Type | Required | Default | Description |
| ---- | ----- | ---- | -------- | ------- | ----------- |
| `--file` | `-f` | string | yes* | | Path to IICS export ZIP package |
| `--workspace` | `-w` | string | yes* | | Path to expanded workspace directory |
| `--publish` | | bool | | false | Restrict to publishable types; requires `--target-profile` or `--report` |
| `--report` | | string list | | | Compare dependencies across one or more target profiles (mutually exclusive with `--target-profile`) |
| `--target-profile` | `-t` | string | | | Profile name for single-profile target org validation (mutually exclusive with `--report`) |
| `--order-by` | | string | | | Sort output by field: `path`, `type`, `status`, `warning`, `location`, `dependency` |
| `--exclude` | `-e` | string | | | Regex matched against `path.type` - excludes from BFS resolution |
| `--filter` | | string | | | Regex matched against `location` (`Explore/path.type`) - filters final output only |
| `--output-file` | | string | | | Write output to a file |
| `--output-file-format` | | string | | `yaml` | Output file format: `table`, `json`, `csv`, `yaml` |
| `--output-file-fields` | | string | | `location,dependency,type,path,status,warning` | Comma-separated fields for output file rows |

\* `--file` and `--workspace` are mutually exclusive; exactly one is required.

When `--order-by` is provided it overrides the default sort (path-only or publish-priority).
The secondary sort key is always `path`.

All [global flags](../../README.md#global-flags) apply, including `--output` / `-o`.

### Output columns

Each row includes a normalized `location` (`Explore/path.type`) and `dependency` marker.
For `package dependencies`, `dependency=explicit` means the asset has a backing object file in
`exportPackage.chksum`; `dependency=transitive` means it is reference-only in metadata or
API-resolved during traversal.

#### Single-profile mode (no `--target-profile` or `--report`)

| Column | JSON field | Description |
| ------ | ---------- | ----------- |
| `LOCATION` | `location` | Asset identifier as `Explore/path.type` |
| `DEPENDENCY` | `dependency` | `explicit` for checksum-backed package assets, `transitive` for metadata-only or API-resolved assets |
| `TYPE` | `type` | Asset type |
| `PATH` | `path` | Asset path without `Explore/` prefix |

#### With `--target-profile <name>`

| Column | JSON field | Description |
| ------ | ---------- | ----------- |
| `LOCATION` | `location` | Asset identifier as `Explore/path.type` |
| `DEPENDENCY` | `dependency` | `explicit` for checksum-backed package assets, `transitive` for metadata-only or API-resolved assets |
| `TYPE` | `type` | Asset type |
| `PATH` | `path` | Asset path without `Explore/` prefix |
| `STATUS (name)` | `status` | `found`, `missing`, or `unknown`; color-coded |
| `WARNING` | `warning` | Only shown when at least one warning exists |

#### With `--report <profiles>`

| Column | JSON field | Description |
| ------ | ---------- | ----------- |
| `LOCATION` | `location` | Asset identifier as `Explore/path.type` |
| `DEPENDENCY` | `dependency` | `explicit` for checksum-backed package assets, `transitive` for metadata-only or API-resolved assets |
| `STATUS (dev)` | `status_dev` | Per-profile status; color-coded |
| `STATUS (qa)` | `status_qa` | Per-profile status; color-coded |
| `WARNING (dev)` | `warning_dev` | CSV only - warning text per profile |
| `WARNING (qa)` | `warning_qa` | CSV only - warning text per profile |

JSON and YAML output for `--report` uses a nested `profiles` map:

```json
{
  "id": "Explore/MyProject/MyConn.AI_CONNECTION",
  "path": "MyProject/MyConn",
  "type": "AI_CONNECTION",
  "location": "Explore/MyProject/MyConn.AI_CONNECTION",
  "dependency": "explicit",
  "profiles": {
    "dev": { "status": "found" },
    "qa":  { "status": "missing", "warning": "asset not found in target org" }
  }
}
```

### Output formats

Supports all standard formats via `--output` / `-o`: `table`, `json`, `csv`, `yaml`.

Additionally, `-o mermaid` renders the dependency graph as a Mermaid `graph TD` diagram
(no extra Go dependencies - pure text generation). Missing assets (when `--target-profile`
is set) are highlighted in red using a `classDef missing` style. Mermaid output is not
supported with `--report`.

### CI/CD environment variables

These environment variables override the target profile settings without modifying
`~/.iics/config.yaml`, which is useful in CI/CD pipelines. Applies to `--target-profile`
only; `--report` uses named profiles from the config file.

| Variable | Overrides |
| -------- | --------- |
| `IICS_TARGET_USERNAME` | target profile username |
| `IICS_TARGET_PASSWORD` | target profile password |
| `IICS_TARGET_REGION` | target profile region |
| `IICS_TARGET_LOGIN_URL` | target profile loginUrl |

### Examples

```bash
# Inspect full dependency graph from a ZIP file
iics package dependencies --file mypackage.zip

# Inspect from an expanded workspace
iics package dependencies --workspace ./src/iics

# Single-profile: validate publish deps and pipe to publish
iics package dependencies -f pkg.zip --publish --target-profile prod -o csv | iics publish run

# Single-profile: sort missing assets to the top
iics package dependencies -f pkg.zip --target-profile qa --order-by status

# Multi-profile: compare dev and qa side-by-side
iics package dependencies -f pkg.zip --report dev,qa

# Multi-profile: publish mode comparison across three orgs
iics package dependencies -f pkg.zip --publish --report dev,qa,prod

# Multi-profile: export comparison as JSON for downstream processing
iics package dependencies -f pkg.zip --report dev,qa -o json

# Multi-profile: export comparison as CSV for spreadsheet review
iics package dependencies -f pkg.zip --report dev,qa -o csv

# Exclude system connections, filter to a specific project
iics package dependencies -f pkg.zip --exclude '^SYS' --filter 'MyProject'

# Write dependency list to a JSON file with selected fields
iics package dependencies -f pkg.zip \
  --output-file deps.json \
  --output-file-format json \
  --output-file-fields location,dependency,type,status,warning

# Render dependency graph as Mermaid diagram
iics package dependencies -f pkg.zip -o mermaid

# Render publish-only graph with target validation for review
iics package dependencies -f pkg.zip --publish --target-profile prod -o mermaid
```

```powershell
# Inspect full dependency graph from a ZIP file
iics package dependencies --file mypackage.zip

# Inspect from an expanded workspace
iics package dependencies --workspace ./src/iics

# Single-profile: validate publish deps and pipe to publish
iics package dependencies -f pkg.zip --publish --target-profile prod -o csv | iics publish run

# Multi-profile: compare dev and qa side-by-side
iics package dependencies -f pkg.zip --report dev,qa

# Multi-profile: publish mode comparison
iics package dependencies -f pkg.zip --publish --report dev,qa,prod

# Multi-profile: export as CSV for spreadsheet review
iics package dependencies -f pkg.zip --report dev,qa -o csv

# Render dependency graph as Mermaid diagram
iics package dependencies -f pkg.zip -o mermaid
```

---

## See also

- [export](export.md) - export assets from IICS to a ZIP package
- [import](import.md) - import a ZIP package into IICS
