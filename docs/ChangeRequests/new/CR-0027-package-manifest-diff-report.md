# CR-0027: Package Manifest Diff Report

## CR Type

- [ ] New resource
- [x] Enhancement to existing command

## Problem

`iics package create` with `--manifest-file` performs selective packaging but produces no
machine-readable audit trail of the differences between what was requested and what was
assembled. Operators cannot easily answer questions such as:

- Which manifest entries matched no asset in the source workspace?
- Which assets were auto-included (parent containers, SYS-rooted assets, transitive
  closure members) that were not listed in the manifest?
- Which manifest entries were deduplicated and why?

The only feedback currently available is a small set of `--verbose` messages written to
stdout and warnings written to stderr. These are not persisted and cannot be post-processed
by CI pipelines or stored as build artifacts.

## Proposed Solution

Add a `--diff-report` flag to `iics package create`. When provided, the command writes a
diff report file after the package is assembled. The report compares:

1. The resolved manifest entries (what was requested).
2. The final `selectedObjects` slice (what was included in the package).

The report is written in Markdown by default. An optional `--diff-report-format` flag
selects `markdown`, `json`, or `csv`. When `--diff-report` is omitted, no report is
produced and existing behavior is unchanged.

## CLI Changes

```text
iics package create \
  --source <dir> \
  --target <out.zip> \
  --manifest-file <manifest.csv> \
  [--diff-report <path>] \
  [--diff-report-format markdown|json|csv]
```

| Flag | Short | Default | Description |
|---|---|---|---|
| `--diff-report` | - | - | Path to write the diff report; if omitted, no report is generated |
| `--diff-report-format` | - | `markdown` | Report format: `markdown`, `json`, `csv` |

Constraints:

- `--diff-report` and `--diff-report-format` are only meaningful when a selection
  manifest is active (via `--manifest-file` or piped stdin). If `--diff-report` is
  given but no manifest is provided, return an error:
  `--diff-report requires a selection manifest (--manifest-file or stdin)`.
- The diff report is written after the ZIP is finalized successfully. A report write
  failure is reported as an error but does not delete the already-created ZIP.

## Diff Report Format

The report captures four categories of differences, in the order below.

### 1. Manifest entries with no matching asset (missing from workspace)

These are manifest rows where `SelectExportedObjects` returned an error or where the entry
path/id did not resolve to any `exportedObject` in `exportMetadata.v2.json`. Currently,
a non-matching entry causes the command to abort with an error. After this CR, the check
remains fatal; this section is therefore only populated when the user pre-validates the
manifest separately and generates the report from a partial run, OR when future CRs add a
`--allow-missing` option. For now, include the section header with a note that unresolved
entries abort the command.

### 2. Assets included from manifest (explicit selections)

Objects whose GUID appears in `selectedIDs` and whose inclusion reason is a direct manifest
match (id, path+type, or path). Columns: `objectName`, `objectType`, `path`, `objectGuid`,
`reason` (`direct`).

### 3. Assets auto-included beyond the manifest (extras)

Objects in the final `selectedObjects` slice whose inclusion was triggered by logic beyond
the literal manifest rows:

| Reason | Source |
|---|---|
| `parent-container` | Added by `IncludeParentContainers` |
| `transitive-closure` | Added by `IncludeReferencedClosure` |
| `sys-implied` | SYS-rooted objects added when the manifest references only containers |
| `duplicate-ignored` | Entry appeared more than once; subsequent occurrences skipped |

Columns: `objectName`, `objectType`, `path`, `objectGuid`, `reason`.

### 4. Summary counters

| Counter | Description |
|---|---|
| `manifestEntries` | Number of rows in the input manifest after parsing |
| `excludedTransitiveFound` | Rows dropped by `--exclude-found-transitive` (from `BuildManifestStats`) |
| `directMatches` | Objects matched directly from manifest rows |
| `parentContainersAdded` | Count returned by `IncludeParentContainers` |
| `transitiveClosureAdded` | Count returned by `IncludeReferencedClosure` |
| `sysImplied` | SYS objects added by container-only expansion |
| `duplicatesIgnored` | Entries skipped because the GUID was already selected |
| `finalPackageObjects` | Total objects in the assembled package |

### Markdown layout (default)

```markdown
# Package Manifest Diff Report

**Package:** <target filename>
**Manifest:** <manifest file path or "stdin">
**Generated:** <RFC3339 timestamp>

## Summary

| Counter | Value |
|---------|-------|
| manifestEntries | N |
...

## Explicit Selections (N)

| NAME | TYPE | PATH | GUID |
|------|------|------|------|
...

## Auto-included Assets (N)

| NAME | TYPE | PATH | GUID | REASON |
|------|------|------|------|--------|
...

## Notes

- Unresolved manifest entries cause the command to abort before this report is written.
- Use `--verbose` for additional diagnostic messages during packaging.
```

### JSON layout

A single JSON object with keys `summary`, `explicitSelections` (array), and
`autoIncluded` (array). Each array element mirrors the table columns above as a JSON
object.

### CSV layout

Two sections separated by a blank line: one CSV block for explicit selections and one for
auto-included assets. Each block has its own header row. The summary counters appear as
a trailing block with two columns `counter,value`.

## Use Cases

### Deployment auditing

A CI pipeline creates a selective package for the `prod` profile and stores the diff
report as a build artifact. Reviewers verify that only intended assets are in the package
before approving the deployment gate.

### Debugging missing assets

A developer runs `package create --manifest-file release.csv --diff-report /tmp/diff.md`
and inspects the `Auto-included Assets` section to confirm that the expected parent
folders and SYS connections were pulled in by the transitive closure.

### Compliance documentation

The Markdown diff report is committed alongside the deployment ZIP in an audit branch,
providing a permanent record of exactly which assets were deployed and how each was
selected.

## Acceptance Criteria

1. `--diff-report <path>` writes a diff report when a selection manifest is active.
2. `--diff-report` without an active manifest returns an error and exits without writing
   a ZIP.
3. `--diff-report-format markdown` (default) produces valid Markdown with all four
   sections.
4. `--diff-report-format json` produces a valid JSON object readable by `jq`.
5. `--diff-report-format csv` produces parseable CSV with a header row per block.
6. The `summary` section counts match the actual objects in the assembled package.
7. Each auto-included object carries the correct `reason` value from the list in
   section 3 above.
8. A failure to write the diff report file returns an error but does not delete the
   already-created ZIP.
9. When `--diff-report` is not provided, command output and behavior are identical to
   the current implementation.
10. The diff report is written only after the ZIP is successfully finalized.

## Files to Change

- `cmd/package.go` - add `diffReport` and `diffReportFormat` local variables in
  `newPackageCreateCmd`; populate a `packageDiffReport` struct from the data already
  computed during selection (`warnings`, `parentAdded`, `closureAdded`,
  `manifestStats`, `selectedObjects`, and the reason each object was selected); call
  `writeDiffReport` after the ZIP is closed.
- `cmd/package.go` - add unexported helpers:
  - `type packageDiffReport struct` - holds all diff data
  - `func buildDiffReport(...) packageDiffReport` - assembles the struct from selection
    outputs
  - `func writeDiffReport(r packageDiffReport, path, format string) error` - dispatches
    to format-specific writers
  - `func writeDiffReportMarkdown(r packageDiffReport, w io.Writer) error`
  - `func writeDiffReportJSON(r packageDiffReport, w io.Writer) error`
  - `func writeDiffReportCSV(r packageDiffReport, w io.Writer) error`
- `internal/dependencies/selective.go` - extend `SelectExportedObjects` to return
  per-object reason strings alongside `selectedIDs` so `buildDiffReport` can populate
  the `reason` column without re-deriving it. The existing return signature
  `(map[string]bool, []string, error)` becomes
  `(map[string]bool, map[string]string, []string, error)` where the new map is
  `guid -> reason`.
- `docs/documentation/package.md` - document the two new flags.
- `README.md` - no new row needed; the `package create` entry already exists.

## Notes and Considerations

### Recommended implementation order

Implement CR-0025 (exclusion manifest) before this CR. When both CRs are active,
the diff report must reflect the post-exclusion asset set so that the `Auto-included
Assets` and `Explicit Selections` sections match the actual ZIP content. If CR-0025
has not yet been implemented, the diff report reflects the inclusion-only set, which
is still correct for that code state.

The `SelectExportedObjects` signature change proposed here (`guid -> reason` map in
the return values) is backward-compatible at the only call site in `cmd/package.go`.
Coordinate with CR-0025 implementors: if CR-0025 adds a new `ExcludeByManifest` /
`ExcludeByRegex` layer after `SelectExportedObjects`, the exclusion-reason strings
(`excluded-by-manifest`, `excluded-by-regex`) belong in a separate `excludedReasons`
map populated in `newPackageCreateCmd`, not in the `SelectExportedObjects` return
value, to keep concerns separated.

### Relationship to CR-0026 (build manifest report)

CR-0026 covers writing a build manifest report that summarizes the overall build process
(package name, manifest file used, build history). CR-0027 is narrower and orthogonal:
it compares manifest intent versus assembled content for a single `package create` run.
The two reports have different audiences (build history vs. content audit) and different
scopes, but they could share the same `writeOutputFile` helper already present in
`cmd/package.go` for the CSV and JSON paths. The Markdown writer will be specific to
this CR.

### Reason tracking requires a small selective.go change

Currently `SelectExportedObjects` tracks selection reasons only in `warnings` (for
duplicate entries). To populate the `reason` column accurately, the function must record
the reason for every selected GUID. The proposed signature change is backward-compatible
at the call site in `cmd/package.go`; the only addition is a new `map[string]string`
return value. No other callers of `SelectExportedObjects` exist in the codebase.

### No new dependencies

All report formats can be implemented with the Go standard library (`encoding/json`,
`encoding/csv`, `fmt`). No new imports are required.

### Parent container and SYS reasons are derived outside selective.go

`IncludeParentContainers` and the SYS-implied logic run after `SelectExportedObjects`
returns. Reason assignment for those categories must happen in `buildDiffReport` by
comparing the GUIDs added in each phase.

### Deduplication warnings are already collected

The `warnings` slice returned by `SelectExportedObjects` already identifies duplicate
manifest entries. `buildDiffReport` can parse these strings to populate the
`duplicate-ignored` reason without additional changes to `selective.go`.
