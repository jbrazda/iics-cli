# CR-0026: Build Manifest Report File Output

## CR Type

- [ ] New resource
- [x] Enhancement to existing command

## Problem

`package create` currently writes all diagnostic information (warnings, selection refinement
counts, file lists) to stdout/stderr during the run. Once the command exits, there is no
machine-readable or human-readable record of what was built: which manifest drove selection,
how many assets were included or excluded, and what warnings were raised.

In a CI pipeline that runs `package create` for multiple environments in sequence, this
information is interleaved in build logs and is difficult to extract after the fact. There is
no audit trail linking a ZIP artifact to the manifest that produced it.

This is deferred item 3 from CR-0023.

## Proposed Solution

Add a `--build-manifest-report` flag to `iics package create`. When the flag is set (or
defaults to the standard path), the command appends a Markdown section to a report file
after the package is successfully created. If the file does not exist it is created. If it
already exists, the new section is appended so that multiple builds accumulate in one file.

The default path is `target/iics/import/logs/release-manifest.md`, which places the report
alongside the other generated artifacts under `target/iics/import/`.

The flag can be set to an empty string (`--build-manifest-report=""`) to disable report
generation entirely, so callers that do not want a report do not have to opt out.

## CLI Changes

### New flag on `iics package create`

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--build-manifest-report` | `string` | `target/iics/import/logs/release-manifest.md` | Path to the build manifest report file; append if it exists; leave empty to disable |

### Behavior

- When the flag value is empty, report generation is skipped silently.
- When the flag value is non-empty:
  - The parent directory is created with `os.MkdirAll` (mode `0o755`) if absent.
  - The file is opened in append mode (`os.O_CREATE|os.O_APPEND|os.O_WRONLY`, mode `0o644`).
  - The Markdown section is written after a successful `zw.Close()` / `outFile.Close()`.
  - If the file cannot be opened or written, the command returns an error (the package ZIP
    was already written successfully at that point, so this is a non-destructive failure).
- Report generation is skipped if no selection manifest was used (i.e., `!hasSelectionManifest`),
  because there is nothing selection-specific to record. A simple log line is emitted to
  stdout instead: `"Note: no selection manifest used; skipping build manifest report"`.

## Report Format

Each build appends one top-level section. The heading uses a timestamp so that multiple
builds in the same file are visually distinct and sortable.

```markdown
## Build: <RFC3339 UTC timestamp>

- **Package:** `<target path>`
- **Package Name:** `<finalPackageName>`
- **Source:** `<absSource>`
- **Selection Manifest:** `<manifestFile or "stdin">`
- **Built At:** `<RFC3339 UTC timestamp>`

### Selection Summary

| Field | Value |
|-------|-------|
| Assets selected | N |
| Parent containers added | N |
| Transitive deps included | N |
| Transitive found excluded | N |
| Dangling refs pruned | N |
| Total files in package | N |

### Included Assets

| Path | Name | Type | ID |
|------|------|------|----|
| ... | ... | ... | ... |

### Warnings

- `<warning text>` (one bullet per warning; omit section if no warnings)

### Errors

- `<error text>` (one bullet per error encountered during selection; omit section if none)
```

Field notes:

- All counts come from values already computed during `newPackageCreateCmd` `RunE`: the
  `warnings` slice, `parentAdded`, `closureAdded`, `manifestStats.ExcludedTransitiveFound`,
  `prunedCount`, and `len(sortedPaths)+1` (total files including checksum).
- **Included Assets** lists the final `selectedObjects` slice (already sorted by
  `buildContentsOfExportPackageCSV`). The `ID` column contains `ObjectGUID`.
- When the Included Assets table would exceed 500 rows, emit the first 500 rows followed by
  a line: `_(truncated - N total assets)_`. This prevents unbounded file growth in full-org
  packages that bypass selection.
- **Warnings** and **Errors** sections are omitted entirely (not just left empty) when there
  are no entries, to keep the report concise.

## Use Cases

1. **CI audit trail** - A Jenkins or GitHub Actions pipeline runs `package create` for each
   target environment. All builds append to `target/iics/import/logs/release-manifest.md`.
   After all environments are built the log is archived as a build artifact, providing a
   complete record of what was packaged per environment.

2. **Regression review** - A developer compares two consecutive build reports to verify that
   a change to the selection manifest removed exactly the expected assets and introduced no
   new warnings.

3. **Compliance** - An audit requires proof that a production package was built from a
   specific manifest file and contained a specific set of asset IDs. The report provides
   that evidence without requiring the operator to reconstruct it from raw build logs.

4. **Troubleshooting** - When a deployment fails, the report shows exactly which assets were
   included and what warnings were raised, without having to re-run the build.

## Acceptance Criteria

1. Running `iics package create` with `--build-manifest-report` set to a non-empty path
   creates (or appends to) the report file after a successful build.
2. Running the command twice against the same report path appends a second section; the
   file contains exactly two `## Build:` headings.
3. Running without `--build-manifest-report` (flag absent) writes to the default path
   `target/iics/import/logs/release-manifest.md`.
4. Setting `--build-manifest-report=""` disables report generation; no file is created or
   modified.
5. When no selection manifest is used (`!hasSelectionManifest`), the report is skipped and
   a note is emitted to stdout; no partial report is written.
6. The **Included Assets** table lists every asset in `selectedObjects` (up to the 500-row
   truncation limit), matching the content of `ContentsofExportPackage_<name>.csv`.
7. Each warning emitted to stderr during selection appears as a bullet in the **Warnings**
   section.
8. If the report directory does not exist it is created automatically.
9. A failure to write the report file returns an error from `RunE`; the already-written
   package ZIP is not affected.
10. The command produces identical package output whether or not `--build-manifest-report`
    is set (report generation is a pure side-effect).

## Files to Change

- `cmd/package.go` - add `buildManifestReport string` local variable and flag in
  `newPackageCreateCmd`; add a `writeBuildManifestReport(...)` helper function called after
  the package ZIP is closed; collect the additional counters needed for the report
  (`parentAdded`, `closureAdded`, `prunedCount`) as named variables rather than discarding
  them with `_` in format strings.
- `docs/documentation/package.md` - add `--build-manifest-report` to the flags table for
  `package create` and add an example showing CI usage.
- `README.md` - no structural change required; the `package create` row already links to
  `docs/documentation/package.md`.
- `completions/` - regenerate after flag addition (`make completions`).

No changes to `internal/` packages are required. All data needed for the report is already
available in `newPackageCreateCmd` `RunE` at the point where the package is finalized.

## Notes / Considerations

- **Append semantics** are intentional. Overwrite semantics would erase the history that
  makes this feature useful in CI. Callers that want a fresh report each run should delete
  the file beforehand or write it to a timestamped path.
- **Markdown format** was chosen (over JSON or YAML) because the default output path
  (`logs/release-manifest.md`) is a human-readable artifact and Markdown renders well in
  GitHub Actions summary pages and GitLab CI job artifacts.
- The `writeBuildManifestReport` helper should use `bufio.Writer` for efficiency when
  writing the asset table, as `selectedObjects` can be several hundred rows.
- Timestamps should use `time.Now().UTC().Format(time.RFC3339)` for sortability and
  timezone-unambiguous output.
- The flag name `--build-manifest-report` was specified in CR-0023. Do not shorten it to
  `--report` as that flag is already used on `package dependencies` (as `--report` for
  multi-profile validation).
- The default path `target/iics/import/logs/release-manifest.md` is resolved relative to
  the current working directory at command invocation time, not relative to `--source` or
  `--target`. Document this in `docs/documentation/package.md` and show an absolute path
  example in the CI use-case section.
- No new dependencies are required. The implementation uses only `os`, `bufio`, `fmt`, and
  `time` from the standard library.
