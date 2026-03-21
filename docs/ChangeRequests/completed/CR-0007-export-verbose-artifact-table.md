# CR-0007: Export Verbose Artifact Table Logging

**Status:** NEW

## Problem

When `export start` or `export run` receives input from stdin or a file and `--verbose` is enabled,
the current logging prints only the resolved object ID per line:

```text
  - abc123def456 (includeDeps: true)
```

This is not useful for verifying the correct set of objects is being exported, because:

- It discards path and type information that was present in the input.
- It gives no visibility into what path-only or path+type entries resolved to after lookup.

## Desired Change

In verbose mode, after reading and resolving artifacts, print the full artifact list to stderr as a
formatted table with three columns: `ID`, `PATH`, `TYPE`. Cells are blank when a field is not
available for a given entry.

The table is printed **after** `resolveExportObjects` completes so that lookup results are used to
back-fill resolved IDs and types into the entries before display. This ensures path-only and
path+type entries show their fully resolved values.

The existing per-object `- <id> (includeDeps: ...)` lines currently in `newExportStartCmd` and
`newExportRunCmd` are replaced by this table. The lookup progress lines inside
`resolveExportObjects` (`Looking up IDs...`, `Lookup complete: N resolved`) remain.

All verbose output goes to `cmd.ErrOrStderr()` (stderr) to avoid polluting piped stdout.

## Affected Commands

- `export start` (`newExportStartCmd`)
- `export run` (`newExportRunCmd`)

## Supported Input Scenarios

### 1. CSV from `objects list` - has id, path, type

All three fields are present in the input; no lookup needed.

```text
[2026-03-20 12:00:00] Read 3 artifact entries
[2026-03-20 12:00:00] Objects to export:
+------------------------+---------------------------+-----------+
| ID                     | PATH                      | TYPE      |
+------------------------+---------------------------+-----------+
| abc123def456789012345  | /ZZ_TEST_CLI/MyMapping    | MAPPINGS  |
| bcd234ef5678901234567  | /ZZ_TEST_CLI/MySession    | MTT       |
| cde345f6789012345678a  | /ZZ_TEST_CLI/MyTask       | DSS       |
+------------------------+---------------------------+-----------+
[2026-03-20 12:00:00] Starting export job "..." with 3 objects...
```

### 2. Plain text file - has path + type, no id

Lookup resolves the ID; table is printed after resolution so the ID column is filled in.

```text
[2026-03-20 12:00:00] Read 2 artifact entries
[2026-03-20 12:00:00] Looking up IDs for 2 objects...
[2026-03-20 12:00:00] Lookup complete: 2 resolved
[2026-03-20 12:00:00] Objects to export:
+------------------------+---------------------------+-----------+
| ID                     | PATH                      | TYPE      |
+------------------------+---------------------------+-----------+
| abc123def456789012345  | /ZZ_TEST_CLI/MyMapping    | MAPPINGS  |
| bcd234ef5678901234567  | /ZZ_TEST_CLI/MySession    | MTT       |
+------------------------+---------------------------+-----------+
```

### 3. Path-only input - has path, no type, no id

Lookup resolves both ID and type from the path alone (the `LookupObject` struct sends
`{path: "..."}` with `type` omitted, which the API accepts). The resolved type from
`LookupResult` is back-filled into the entry so the TYPE column is also populated.

```text
[2026-03-20 12:00:00] Read 2 artifact entries
[2026-03-20 12:00:00] Looking up IDs for 2 objects...
[2026-03-20 12:00:00] Lookup complete: 2 resolved
[2026-03-20 12:00:00] Objects to export:
+------------------------+---------------------------+-----------+
| ID                     | PATH                      | TYPE      |
+------------------------+---------------------------+-----------+
| abc123def456789012345  | /ZZ_TEST_CLI/MyMapping    | MAPPINGS  |
| bcd234ef5678901234567  | /ZZ_TEST_CLI/MySession    | MTT       |
+------------------------+---------------------------+-----------+
```

### 4. ID-only input - id only, no path or type

No lookup needed. PATH and TYPE columns are blank.

```text
[2026-03-20 12:00:00] Read 2 artifact entries
[2026-03-20 12:00:00] Objects to export:
+------------------------+------+------+
| ID                     | PATH | TYPE |
+------------------------+------+------+
| abc123def456789012345  |      |      |
| bcd234ef5678901234567  |      |      |
+------------------------+------+------+
```

## Required Code Changes

### `internal/client/export.go`

1. **Support path-only lookup entries.** In `resolveExportObjects`, change the condition that
   decides which entries need lookup from `e.ID == ""` to also include path-only entries (where
   `e.Type == ""`). The `LookupObject` struct already uses `omitempty` on `Type`, so sending
   `{path: "..."}` without type is valid.

2. **Back-fill resolved data into entries.** After receiving the `LookupResponse`, update each
   source entry with the full `LookupResult` values:

   ```go
   for i, result := range resp.Objects {
       if i < len(lookupOrigIdx) {
           origIdx := lookupOrigIdx[i]
           resolvedIDs[origIdx] = result.ID
           entries[origIdx].ID = result.ID
           if entries[origIdx].Path == "" {
               entries[origIdx].Path = result.Path
           }
           if entries[origIdx].Type == "" {
               entries[origIdx].Type = result.Type
           }
       }
   }
   ```

3. **Update `resolveExportObjects` signature** to return the enriched entries alongside the
   `ExportObject` slice:

   ```go
   func resolveExportObjects(
       ctx context.Context, c *client.Client,
       entries []client.ArtifactEntry, includeDeps bool, out io.Writer,
   ) ([]client.ExportObject, []client.ArtifactEntry, error)
   ```

### `cmd/export.go`

1. **Update both call sites** (`newExportStartCmd`, `newExportRunCmd`) to capture the returned
   enriched entries.

2. **Add helper:**

   ```go
   func printArtifactTable(entries []client.ArtifactEntry, w io.Writer)
   ```

   Always renders three columns: `ID`, `PATH`, `TYPE`. Uses tablewriter v1.x:
   `tablewriter.NewTable(w)`, `table.Header(...)`, `table.Append([]interface{}{...})`,
   `err := table.Render()`.

3. **In both commands**, after `resolveExportObjects` and when `verbose` is true:

   ```go
   stderr := cmd.ErrOrStderr()
   _, _ = fmt.Fprintf(stderr, "[%s] Objects to export:\n", ts())
   printArtifactTable(enrichedEntries, stderr)
   ```

4. **Remove** the inline `- %s (includeDeps: %v)` per-object loops from both commands.

## Affected Files

- `internal/client/export.go` - back-fill lookup results into entries; update
  `resolveExportObjects` signature; allow path-only entries in lookup
- `cmd/export.go` - add `printArtifactTable`; update call sites; print table to stderr;
  remove old per-object loops

## Acceptance Criteria

- [ ] `export run --verbose` with CSV stdin (id+path+type) prints three-column table to stderr
- [ ] `export run --verbose` with TXT input (path+type, no id) prints table with resolved IDs after lookup
- [ ] `export run --verbose` with path-only input prints table with resolved id and type after lookup
- [ ] `export run --verbose` with ID-only input prints three-column table with blank PATH and TYPE
- [ ] Verbose table output does not appear on stdout (does not corrupt piped output)
- [ ] Non-verbose mode is unchanged
- [ ] All existing tests pass
- [ ] New test covers back-fill logic for path-only entries in `resolveExportObjects`

## Do NOT

- Change non-verbose output
- Print the table to stdout
- Add new flags
- Refactor unrelated export code
