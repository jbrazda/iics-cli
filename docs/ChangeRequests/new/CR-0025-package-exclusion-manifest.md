# CR-0025: Package Create Exclusion Manifest and Regex Support

## CR Type

- [ ] New resource
- [x] Enhancement to existing command

## Problem

`iics package create` accepts an inclusion manifest (`--manifest-file` / `-m`) that controls which
assets are included in the selective package. However, there is no way to remove specific assets
from an otherwise complete or manifest-driven package without editing the inclusion manifest itself.

Two concrete pain points arise in CI/CD pipelines:

1. A base inclusion manifest is reused across environments, but certain assets (e.g., sandbox-only
   connectors, test scaffolding) must be dropped for production packages without maintaining
   separate manifest files for each environment.
2. Operators need to exclude assets whose names or paths match a pattern (e.g., all assets under
   a `_DEV` folder, or all `GUIDE` types with a particular naming convention) without enumerating
   every asset individually.

This CR covers deferred item 2 from CR-0023:

> "Exclusion manifest and exclusion regex support"

## Proposed Solution

Add two new flags to `iics package create`:

- `--exclude-manifest-file <path>` - path to an exclusion manifest file in any format already
  supported by `--manifest-file` (`.txt`, `.csv`, `.json`, `.yaml`/`.yml`). Assets resolved from
  this manifest are removed from the selected set before the package is assembled.
- `--exclude <pattern>` - a Go regular expression matched against each asset's normalized
  `path/name.type` key (the same format used by `package dependencies --exclude`). Assets whose
  key matches are removed from the selected set.

Both flags may be combined with `--manifest-file` and with each other. Neither flag is required,
and the command behaves identically to today when neither flag is provided.

## CLI Changes

```text
iics package create \
  --source <dir>                   # existing, required
  --target <zip>                   # existing, required
  [--manifest-file/-m <path>]      # existing, optional inclusion manifest
  [--exclude-manifest-file <path>] # NEW: exclusion manifest (same formats as --manifest-file)
  [--exclude/-e <regex>]           # NEW: regex exclusion applied to path/name.type
  [--exclude-found-transitive]     # existing
  [--status-target <key>]          # existing
  [--name/-n <name>]               # existing
  [--include-tags]                 # existing
  [--force/-f]                     # existing
```

Flag details:

| Flag                    | Short | Type   | Default | Description |
|-------------------------|-------|--------|---------|-------------|
| `--exclude-manifest-file` | -   | string | `""`    | Exclusion manifest path; same formats as `--manifest-file` |
| `--exclude`             | `-e`  | string | `""`    | Go regex matched against `path/name.type`; matching assets are excluded |

The short alias `-e` is not currently used by `package create`. It mirrors the same flag on
`package dependencies --exclude / -e` for consistency.

## Selection Semantics

Exclusion is applied **after** the inclusion phase completes and **before** the package is
assembled. The processing order is:

1. Inclusion manifest (`--manifest-file` / stdin) is resolved to a set of selected asset GUIDs.
   - If no inclusion manifest is provided, the full set of exported objects is used (current behavior).
2. Transitive dependency closure and parent-container inference run on the inclusion set (existing logic).
3. Exclusion manifest (`--exclude-manifest-file`) is parsed using the same `readPackageSelectionManifest`
   path (without `excludeFoundTransitive` semantics - exclusion manifests are always treated as plain
   asset lists). The resolved GUIDs are subtracted from the selected set.
4. Exclusion regex (`--exclude`) is compiled and matched against each remaining asset's
   `path/name.type` key. Matching assets are removed from the selected set.
5. The remaining set is used to filter package files and regenerate `exportMetadata.v2.json`,
   `ContentsofExportPackage_<name>.csv`, and `exportPackage.chksum`.

Precedence rules:

- Exclusion always wins over inclusion. An asset present in both the inclusion manifest and the
  exclusion manifest is excluded.
- Exclusion regex is applied after the exclusion manifest so both filters are cumulative.
- Parent containers (Project, Folder) inferred for the inclusion set are also subject to exclusion.
  If all assets within an inferred container are excluded, the container metadata entry is dropped.
- `--exclude-found-transitive` runs before exclusion step 3 (it filters the inclusion-side manifest
  rows). The two mechanisms are orthogonal.

Error conditions:

- If every selected asset is excluded, the command must return an error:
  `"all selected assets were excluded; package would be empty"`.
- If `--exclude-manifest-file` resolves to no recognized assets, a warning is printed to stderr
  (not a fatal error), because the exclusion file may target assets that are simply not present in
  the package.
- An invalid regex for `--exclude` returns a fatal error before any file I/O.

## Use Cases

**Environment-specific packages from a single base manifest:**

```bash
# Base package (all assets)
iics package create -s workspace/ -t deploy/base.zip -m manifests/base.csv

# Production package: exclude sandbox-only assets
iics package create -s workspace/ -t deploy/prod.zip \
  -m manifests/base.csv \
  --exclude-manifest-file manifests/exclude-sandbox.txt

# Connector-free package: exclude all connector and connection types by regex
iics package create -s workspace/ -t deploy/no-connectors.zip \
  -m manifests/base.csv \
  --exclude '\.(AI_SERVICE_CONNECTOR|AI_CONNECTION|Connection)$'
```

**Excluding a whole folder by name pattern:**

```bash
iics package create -s workspace/ -t deploy/release.zip \
  -m manifests/all-assets.csv \
  --exclude 'DEV/'
```

**Combining both exclusion methods:**

```bash
iics package create -s workspace/ -t deploy/qa.zip \
  -m manifests/full.csv \
  --exclude-manifest-file manifests/exclude-prod-only.txt \
  --exclude '_SANDBOX\.'
```

## Acceptance Criteria

1. `--exclude-manifest-file` accepts `.txt`, `.csv`, `.json`, `.yaml`, `.yml` formats (same parser
   as `--manifest-file`).
2. Assets resolved from `--exclude-manifest-file` are removed from the selected set before
   package assembly.
3. `--exclude <regex>` is matched against each asset's `path/name.type` key and matching assets
   are removed from the selected set.
4. Both `--exclude-manifest-file` and `--exclude` may be specified together; their effects are
   cumulative.
5. Both flags are optional. Omitting both preserves existing command behavior exactly.
6. If `--manifest-file` is omitted (full-package mode), exclusion still applies to the full set
   of exported objects.
7. If all assets are excluded, the command returns a descriptive error and creates no output ZIP.
8. An invalid regex for `--exclude` produces a fatal error with the offending pattern quoted.
9. When `--verbose` is set, the command prints the count of assets excluded by manifest and by
   regex separately, e.g.:
   - `Exclusion filter: removed 3 assets via --exclude-manifest-file`
   - `Exclusion filter: removed 7 assets via --exclude pattern`
10. The generated `exportMetadata.v2.json`, `ContentsofExportPackage_<name>.csv`, and
    `exportPackage.chksum` reflect only the assets that survived both inclusion and exclusion.

## Files to Change

- `cmd/package.go`
  - Add `excludeManifestFile string` and `excludePattern string` local variables in
    `newPackageCreateCmd()`.
  - Register `--exclude-manifest-file` (no short alias) and `--exclude` / `-e` flags.
  - After the existing inclusion-manifest and transitive-closure logic, call a new helper
    `applyExclusionManifest(excludeManifestFile, selectedIDs, exported)` and then call
    `applyExclusionRegex(excludePattern, selectedObjects, selectedIDs)`.
  - Emit verbose counts for each exclusion step.
  - Add the post-exclusion empty-set guard with the appropriate error message.

- `internal/dependencies/selective.go` (or a new file in the same package)
  - Add `ExcludeByManifest(entries []client.ArtifactEntry, exported []ExportedObjectRef, selectedIDs map[string]bool) (removed int, warnings []string)`.
  - Add `ExcludeByRegex(pattern string, objects []ExportedObjectRef, selectedIDs map[string]bool) (removed int, err error)`.
  - The regex in `ExcludeByRegex` is matched against `path/name.type` using the same key format as
    `ObjectChecksumCandidates` / `mkFromPkg` to stay consistent with `package dependencies --exclude`.

No changes are required to `internal/client/`, `internal/release/`, or `internal/output/`.

## Notes and Considerations

- The `--exclude` short alias `-e` already exists on `package dependencies` with the same
  semantics (regex against path/name.type). Reusing it here creates a consistent mental model
  across subcommands.
- `--exclude-manifest-file` intentionally does not support stdin. Stdin is already reserved for
  the inclusion manifest when `--manifest-file` is omitted. Accepting a second manifest from stdin
  simultaneously is not possible and would require a breaking change to the existing interface.
- The exclusion manifest parser must not pass `excludeFoundTransitive` or `statusTarget` options;
  exclusion manifests are always treated as simple asset lists regardless of CSV column structure.
- Parent container objects (Project, Folder) that are inferred during inclusion are also candidates
  for exclusion. If a project or folder entry appears in the exclusion manifest, it should remove
  the container metadata entry but should not cascade-remove assets within it (the assets remain
  selected unless they are also excluded explicitly). This preserves the principle that exclusion
  is precise rather than hierarchical.
- Regex exclusion is intentionally not anchored by default. Users can anchor with `^` or `$` as
  needed. This matches the behavior of `package dependencies --exclude`.
- Consider a future CR to add `--exclude-file` (a file of regex patterns, one per line) mirroring
  `package dependencies --exclude-file`, but that is out of scope here.
