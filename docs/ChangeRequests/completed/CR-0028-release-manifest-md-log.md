# CR-0028: Append Markdown Release Manifest Log

## CR Type

- [ ] New resource
- [x] Enhancement to existing commands

## Problem

Release automation currently generates multiple operational artifacts across
`release plan`, `package create`, `export run`, `import run`, and `publish run`,
but there is no single append-only Markdown report that captures what happened
across the CI/CD pipeline. Operators must reconstruct the release from command
output, generated CSV files, job status responses, and downloaded logs.

The sample structure in `testdata/release/release-manifest.md` describes the
desired report shape, including release planning details, package content per
target, backup/export details, import results, and publish results.

This CR supersedes CR-0026. CR-0026 covered only `package create`; this CR
generalizes the same audit/reporting requirement across the release pipeline.

## Proposed Solution

Add a consistent `--log-file` flag to the release pipeline commands listed
below. When the flag is present, the command appends a Markdown section to the
specified release manifest log file.

If `--log-file` is present without a value, it defaults to:

```text
target/iics/import/logs/release_manifest.md
```

If `--log-file=PATH` is provided, the command appends to that path instead.
If `--log-file` is omitted, behavior is unchanged and no Markdown release log is
written.

Log write failures should not fail the deployment operation. The command should
print a warning to stderr and keep the original command result.

## Commands in Scope

| Command               | Report section appended                                                                                                                                                              |
|-----------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `iics release plan`   | Release manifest header, included items, excluded items when available, dependency status summary, release plan file tree, package content per target, publishable assets per target |
| `iics package create` | Package build report for the generated ZIP, selection summary, included assets, warnings                                                                                             |
| `iics export run`     | Backup and rollback export report, exported objects, output ZIP path, optional export log path                                                                                       |
| `iics import run`     | Import report summary, imported objects, and import log content when available or when the import fails                                                                              |
| `iics publish run`    | Publish report summary, publish item details, and publish error details when any item fails                                                                                          |

## CLI Changes

Add the following flag to each command in scope:

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--log-file` | optional string | disabled when omitted, `target/iics/import/logs/release_manifest.md` when present without value | Append a Markdown release report section to the given file |

Recommended Cobra setup:

```go
cmd.Flags().StringVar(&logFile, "log-file", "", "append Markdown release log to this file")
_ = cmd.Flags().Lookup("log-file").NoOptDefVal = release.DefaultManifestLogPath
```

Implementation notes:

- Use the flag's changed state to distinguish omitted from enabled.
- Use the flag value when present with an explicit path.
- Use `release.DefaultManifestLogPath` when present without a path.
- Create parent directories with mode `0o755`.
- Open the file with `os.O_CREATE|os.O_APPEND|os.O_WRONLY` and mode `0o644`.
- On write failure, print a warning to `cmd.ErrOrStderr()` and do not change the
  command's success or failure result.

## Report Format

The report is Markdown only. Each command appends one or more sections matching
the sample file in `testdata/release/release-manifest.md`.

### Release plan

`iics release plan --log-file` initializes or appends the planning sections:

```markdown
# Release Manifest

- Schema Version: `v1`
- Generated At (UTC): `<timestamp>`
- Source: `<manifest path>`
- Mode: `<tag-based|full>`
- Tag: `<tag or empty>`
- Targets: `<target list>`
- Include Connectors: `<true|false>`
- Include Connections: `<true|false>`

## Included Items

| TYPE | COUNT |
|------|------:|
| ... | ... |

## Excluded Items

...

## Deployment Dependency Status Summary

| LOCATION | DEPENDENCY | STATUS (PROD) | STATUS (QA) | STATUS (TST) |
|----------|:----------:|:-------------:|:-----------:|:------------:|
| ... | ... | ... | ... | ... |

## Release Plan

```text
target
...
```

## Package Content per Target

| TYPE | COUNT (PROD) | COUNT (QA) | COUNT (TST) |
|------|-------------:|-----------:|------------:|
| ... | ... | ... | ... |

## Publishable Assets per Target

| TYPE | COUNT (PROD) | COUNT (QA) | COUNT (TST) |
|------|-------------:|-----------:|------------:|
| ... | ... | ... | ... |
```

### Package create

`iics package create --log-file` appends package build details:

```markdown
## Package Build Report

| FIELD | VALUE |
|-------|-------|
| Package | `<target zip>` |
| Source | `<source directory>` |
| Selection Manifest | `<manifest path or stdin>` |
| Package Name | `<package name>` |
| Files | `<count>` |

### Selection Summary

| FIELD | VALUE |
|-------|------:|
| Assets selected | ... |
| Parent containers added | ... |
| Transitive deps included | ... |
| Transitive found excluded | ... |
| Dangling refs pruned | ... |

### Included Assets

| PATH | NAME | TYPE | ID |
|------|------|------|----|
| ... | ... | ... | ... |
```

### Export run

`iics export run --log-file` appends backup and rollback content:

```markdown
## Backup and Rollback Plan

### Export Summary

| FIELD | VALUE |
|-------|-------|
| Job ID | ... |
| Name | ... |
| State | ... |
| Export ZIP | ... |
| Export Log | ... |

### Exported Objects

| ID | PATH | TYPE | STATUS |
|----|------|------|--------|
| ... | ... | ... | ... |
```

### Import run

`iics import run --log-file` appends import results:

```markdown
## Import Report

### Import Summary

| FIELD | VALUE |
|-------|-------|
| Job ID | ... |
| State | ... |
| Start Date | ... |
| End Date | ... |
| Total | ... |
| Published | ... |
| Errors | ... |

### Imported Objects

| SOURCE ID | SOURCE PATH | SOURCE NAME | TARGET NAME | SOURCE TYPE | STATE | MESSAGE |
|-----------|-------------|-------------|-------------|-------------|-------|---------|
| ... | ... | ... | ... | ... | ... | ... |
```

If an import fails and the log can be downloaded, append the log content in a
fenced `txt` block.

### Publish run

`iics publish run --log-file` appends publish results:

```markdown
## Publish Report

### Publish Summary

| FIELD | VALUE |
|-------|-------|
| Job ID | ... |
| State | ... |
| Start Date | ... |
| End Date | ... |
| Duration | ... |

### Publish Items

| INDEX | GUID | ASSET PATH | STATE | START DATE | END DATE | DURATION |
|-------|------|------------|-------|------------|----------|----------|
| ... | ... | ... | ... | ... | ... | ... |
```

If any publish item fails, append:

```markdown
## Publish Errors

| GUID | ASSET PATH | STATE | DETAIL |
|------|------------|-------|--------|
| ... | ... | ... | ... |
```

## Implementation Approach

1. Add a small release report writer package under `internal/release/`:
   - `const DefaultManifestLogPath = "target/iics/import/logs/release_manifest.md"`
   - `type MarkdownLogOptions struct`
   - helper to resolve `--log-file` presence and value
   - helper to append Markdown while creating parent directories
   - table rendering helpers that escape Markdown table cells
2. Keep command wiring thin:
   - commands collect data they already compute
   - commands call exactly one report writer helper per append point
   - report helpers live outside `cmd/` when they contain formatting logic
3. Add `--log-file` to all commands in scope:
   - `release plan`
   - `package create`
   - `export run`
   - `import run`
   - `publish run`
4. For each command, build a command-specific report input struct in
   `internal/release/` and render a Markdown section.
5. Use warn-only behavior for log write failures:
   - `fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not append release log %s: %v\n", path, err)`
   - never mask a successful deployment operation with report write errors
6. Update documentation:
   - `docs/documentation/release.md`
   - `docs/documentation/package.md`
   - `docs/documentation/export.md`
   - `docs/documentation/import.md`
   - `docs/documentation/publish.md`
7. Update `README.md` only if command summary text needs to mention release
   report logging.
8. Regenerate shell completions.

## Tests

Add tests under `internal/` where possible:

1. Report writer appends to a new file and creates parent directories.
2. Report writer appends multiple sections without overwriting existing content.
3. Markdown table escaping handles pipes, newlines, and empty values.
4. `--log-file` present without value resolves to the default path.
5. `--log-file=PATH` resolves to the explicit path.
6. `--log-file` omitted disables logging.
7. Release plan report uses generated package and publish counts per target.
8. Package create report includes selected asset counts and warnings.
9. Export run report renders final job state and exported objects.
10. Import run report renders final summary and object statuses.
11. Publish run report renders all batch item details and error details.
12. Log write errors are converted to warnings at command boundaries.

## Acceptance Criteria

1. Each command in scope accepts `--log-file`.
2. Omitting `--log-file` preserves current behavior and writes no Markdown log.
3. Passing `--log-file` with no value writes to
   `target/iics/import/logs/release_manifest.md`.
4. Passing `--log-file=PATH` appends to the explicit path.
5. Parent directories are created automatically.
6. Repeated commands append sections without overwriting previous sections.
7. Release plan appends the sections shown in the sample release manifest.
8. Package create appends package build details after a successful ZIP build.
9. Export run appends backup/export details after the export package is
   downloaded.
10. Import run appends import details after final status is known, including
    import log content on failure when available.
11. Publish run appends publish details after all batches reach terminal state.
12. Report write failures emit warnings but do not change the command exit
    status.
13. Generated Markdown is stable enough for CI artifacts and code review.
14. Documentation and shell completions include the new flag.

## Files to Change

- `internal/release/` - add release manifest log helpers and command-specific
  report renderers.
- `cmd/release.go` - add `--log-file` to `release plan` and append planning
  sections.
- `cmd/package.go` - add `--log-file` to `package create` and append package
  build details.
- `cmd/export.go` - add `--log-file` to `export run` and append backup/export
  details.
- `cmd/import_.go` - add `--log-file` to `import run` and append import
  details.
- `cmd/publish.go` - add `--log-file` to `publish run` and append publish
  details.
- `docs/documentation/*.md` - document the new flag on the affected command
  pages.
- `completions/` - regenerate after adding flags.

## Notes and Considerations

- This CR intentionally uses only `--log-file`; there is no separate `--md-log`
  flag.
- The flag is opt-in. No report file is generated unless `--log-file` is
  present.
- The default path uses `release_manifest.md` with an underscore to match the
  requested path.
- The report format is append-only because CI/CD pipelines may run multiple
  commands sequentially and archive a single release artifact.
- The implementation should avoid parsing command stdout. It should render from
  structured data already available in each command.
- CR-0026 should not be implemented separately after this CR because its package
  report scope is included here.
