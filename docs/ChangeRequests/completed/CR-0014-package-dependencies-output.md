# CR-0014: `package dependencies` Output Improvements

## CR Type

- [x] **Enhancement** - improve output format, add multi-profile comparison report

---

## Problem

Several output UX issues observed during real production usage (279-asset publish-mode
run, ~36 seconds per org):

1. **PATH and TYPE are redundant columns.** The natural identifier for an asset in IICS is
   `path.type` (e.g., `Corvel_v1/Corvel-GetValidationService-v1.AI_SERVICE_CONNECTOR`). Showing
   them as two narrow columns wastes space and makes the output harder to scan.

2. **SOURCE column adds no value.** Every row is either `package` or `external` and this
   information is rarely acted on. It takes up column space in every run.

3. **TARGET column header does not identify the profile.** When validating against `qa`, the
   column says `TARGET` - the user cannot tell at a glance which org was checked.

4. **WARNING column appears even with zero warnings.** This causes misaligned headers and
   wastes width on every clean run. The column should only appear when there are actual warnings.

5. **No way to compare multiple target orgs in one run.** A common pre-deploy workflow requires
   checking whether assets are present in both `dev` and `qa`. Today this requires two separate
   commands and manual comparison.

6. **No progress feedback during validation.** A 279-asset single-org validation takes ~36s
   with no progress output. Multi-org runs will take proportionally longer.

---

## Desired Change

### 1. PATH column - show `path.type`

Render the PATH column as the concatenation `path.type` using a column `Func`. The underlying
`path` and `type` fields remain separate in the struct (required for the publish pipeline and
JSON output). The column header stays `PATH`.

Example:

```text
PATH
ClaimCenter_GW/Corvel_v1/Corvel-GetValidationService-v1.AI_SERVICE_CONNECTOR
ClaimCenter_GW/AdjusterAssignment_v1/SF-GW-AutomatedAdjusterAssignment.AI_SERVICE_CONNECTOR
```

### 2. Remove SOURCE column

Remove the `Source` field from `dependencyItem` entirely. Stop populating it in
`resolveDependencies`. Remove `source` from the `--order-by` valid field list. Remove the
SOURCE column from all output modes.

### 3. Rename STATUS column with profile name

Change the `TARGET` column header to `STATUS (profilename)` where `profilename` is the value
passed to `--target-profile`. Example: `--target-profile qa` produces column `STATUS (qa)`.

Rename `TargetStatus string json:"targetStatus,omitempty"` to `Status string json:"status,omitempty"`
in the `dependencyItem` struct.

### 4. Conditional WARNING column

The `WARNING` column must only be appended to the columns slice when at least one
`dep.Warning != ""` after validation completes. When all assets are `found`, omit the column.
This applies to single-profile mode only (multi-profile table has no separate WARNING column -
see section 6).

Single-profile output with warnings:

```text
PATH                                                                  STATUS (qa)  WARNING
ClaimCenter_GW/Corvel_v1/Corvel-GetValidationService-v1.AI_SVC_CONN  missing      asset not found in target org
ClaimCenter_GW/LSS_v1/GWCC.AI_SERVICE_CONNECTOR                      missing      asset not found in target org
ClaimCenter_GW/Optum_V1/Optum-GW-CreateActivity.AI_SERVICE_CONNECTOR found
```

Single-profile output with no warnings (WARNING column omitted):

```text
PATH                                                                   STATUS (qa)
ClaimCenter_GW/AdjusterAssignment_v1/SF-GW-Automated.AI_SERVICE_CONN  found
ClaimCenter_GW/AssociatedPolicies_v1/AssociatePolicy.AI_SERVICE_CONN  found
```

### 5. Fix column width consistency

Set fixed `Width` values on the PATH and STATUS columns so headers and data rows align
consistently regardless of whether WARNING is present:

| Column | Width |
| ------ | ----- |
| PATH | 90 |
| STATUS (profile) | 10 |
| WARNING | 40 |

### 6. New `--report` flag - multi-profile comparison

Add `--report` as a `StringSliceVar` flag. It accepts one or more profile names either
comma-separated or as repeated flags:

```bash
iics package dependencies -f pkg.zip --report dev,qa
iics package dependencies -f pkg.zip --report dev --report qa
iics package dependencies -f pkg.zip --publish --report dev,qa
```

`--report` is mutually exclusive with `--target-profile`. Either flag may be used but not
both. `--publish` can be combined with `--report`.

`--report` with a single profile name is valid and produces the same STATUS column format
as `--target-profile`, but uses the report output structure.

#### 6a. Parallel org validation

When `--report` specifies multiple profiles, launch one goroutine per profile using
`sync.WaitGroup`. Each goroutine calls `validateTargetDependencies` on a deep copy of
the `deps` slice (to avoid data races). Results are collected into a
`map[string][]dependencyItem` protected by a `sync.Mutex`. The main goroutine waits for
all profiles to complete before rendering output.

No new Go module dependencies. Use standard `sync` package only.

#### 6b. Multi-profile table output

Build `[]map[string]interface{}` dynamically. Each map represents one asset row with keys:

| Key | Value |
| --- | ----- |
| `id` | `path.type` concatenated |
| `path` | original path (for CSV/JSON consumers) |
| `type` | original type |
| `status_<profile>` | `found`, `missing`, or `unknown` for each profile |

Build columns dynamically - one `STATUS (<profile>)` column per profile, in the order
profiles were specified. Color-code each STATUS cell via a per-column `Func` using the same
color logic as the existing `targetStatusFunc`. No separate WARNING columns in the table;
full warning text is available via `-o json` or `-o csv`.

Example table output for `--report dev,qa`:

```text
PATH                                                                  STATUS (dev)  STATUS (qa)
ClaimCenter_GW/Corvel_v1/Corvel-GetValidation.AI_SERVICE_CONNECTOR   found         missing
ClaimCenter_GW/Corvel_v1/Corvel-Notification.AI_SERVICE_CONNECTOR    missing       missing
ClaimCenter_GW/LSS_v1/GWCC.AI_SERVICE_CONNECTOR                      found         found
```

#### 6c. Multi-profile JSON / YAML output

Define two new structs in `cmd/package.go`:

```go
// reportItem is one row in multi-profile report output.
type reportItem struct {
    ID       string                   `json:"id"`
    Path     string                   `json:"path"`
    Type     string                   `json:"type"`
    Profiles map[string]profileResult `json:"profiles"`
}

// profileResult holds the validation result for one profile.
type profileResult struct {
    Status  string `json:"status"`
    Warning string `json:"warning,omitempty"`
}
```

Build `[]reportItem` from the per-profile results map and pass it to `f.Format()` for
`json` and `yaml` output formats.

Example JSON output:

```json
[
  {
    "id": "Corvel_v1/Corvel-GetValidationService-v1.AI_SERVICE_CONNECTOR",
    "path": "Corvel_v1/Corvel-GetValidationService-v1",
    "type": "AI_SERVICE_CONNECTOR",
    "profiles": {
      "dev": { "status": "found" },
      "qa":  { "status": "missing", "warning": "asset not found in target org" }
    }
  }
]
```

#### 6d. Multi-profile CSV output

For CSV, build the same `[]map[string]interface{}` used for the table but include warning
keys as well:

| Key | Value |
| --- | ----- |
| `id` | `path.type` |
| `path` | path |
| `type` | type |
| `status_<profile>` | status for each profile |
| `warning_<profile>` | warning text for each profile |

Columns: `id`, `path`, `type`, then for each profile in order: `status_<profile>`,
`warning_<profile>`.

### 7. Verbose progress logging

#### Single-profile mode

Add a `slog.Debug` tick inside `validateTargetDependencies` every 50 items to show
progress during long runs:

```
[DEBUG] validating dependencies progress profile=qa processed=50 total=279
[DEBUG] validating dependencies progress profile=qa processed=100 total=279
```

#### Multi-profile report mode

Log goroutine lifecycle and per-profile summary:

```
[INFO] report: starting validation profile=dev total=279
[INFO] report: starting validation profile=qa total=279
[INFO] report: profile complete profile=dev found=279 missing=0 elapsed=34.2s
[INFO] report: profile complete profile=qa found=239 missing=40 elapsed=35.9s
```

---

## Scope

### Files to MODIFY

```text
cmd/package.go                - struct changes, new flag, new helpers, output logic
docs/documentation/package.md - update flags table, output columns, examples
```

### Files to REGENERATE

```text
completions/                  - run `make completions` after cmd changes
```

### Files to READ (context only - do NOT modify)

```text
docs/CLAUDE.md
internal/output/formatter.go  - Column struct, Formatter interface
internal/output/table.go      - how Func is called, extractField
internal/config/config.go     - ResolveTargetProfile
cmd/root.go                   - getFormatter, targetStatusFunc pattern
```

### Forbidden

```text
internal/output/    - do NOT modify
internal/config/    - do NOT modify
cmd/root.go         - do NOT modify
```

---

## Implementation Instructions

### Step 1 - Update `dependencyItem` struct

```go
type dependencyItem struct {
    Path    string `json:"path"`
    Type    string `json:"type"`
    Status  string `json:"status,omitempty"`
    Warning string `json:"warning,omitempty"`
}
```

- Remove `Source` field
- Rename `TargetStatus` to `Status`, change JSON tag from `targetStatus` to `status`

### Step 2 - Update `depField` helper

Remove `"source"` and `"targetStatus"` cases; add `"status"`:

```go
func depField(item dependencyItem, field string) string {
    switch field {
    case "path":
        return item.Path
    case "type":
        return item.Type
    case "status":
        return item.Status
    case "warning":
        return item.Warning
    default:
        return ""
    }
}
```

Update `validOrderByFields` in `newPackageDependenciesCmd` to `[]string{"path", "type", "status", "warning"}`.

### Step 3 - Remove `source` population in `resolveDependencies`

Remove all `Source: "package"` and `Source: "external"` assignments from `resolveDependencies`.
Update all uses of `TargetStatus` to `Status`.

### Step 4 - Add `reportItem` and `profileResult` structs

```go
type reportItem struct {
    ID       string                   `json:"id"`
    Path     string                   `json:"path"`
    Type     string                   `json:"type"`
    Profiles map[string]profileResult `json:"profiles"`
}

type profileResult struct {
    Status  string `json:"status"`
    Warning string `json:"warning,omitempty"`
}
```

### Step 5 - Update `validateTargetDependencies`

Change field references from `d.TargetStatus` to `d.Status`. Add progress debug tick:

```go
for i := range deps {
    if verbose && i > 0 && i%50 == 0 {
        slog.Debug("validating dependencies progress",
            "profile", targetProfileName,
            "processed", i,
            "total", len(deps),
        )
    }
    // ... existing lookup logic ...
}
```

### Step 6 - Add `--report` flag and multi-profile helpers

In `newPackageDependenciesCmd`, add:

```go
var reportProfiles []string
```

Register:
```go
cmd.Flags().StringSliceVar(&reportProfiles, "report", nil,
    "compare dependencies across one or more target profiles (mutually exclusive with --target-profile)")
```

Validation in `RunE`:
```go
if targetProfile != "" && len(reportProfiles) > 0 {
    return fmt.Errorf("--target-profile and --report are mutually exclusive")
}
if publishMode && targetProfile == "" && len(reportProfiles) == 0 {
    return fmt.Errorf("--target-profile or --report is required when --publish is set")
}
```

### Step 7 - Multi-profile parallel validation helper

```go
// validateMultiProfile validates deps against multiple profiles in parallel.
// Returns a map of profileName -> validated copy of deps.
func validateMultiProfile(
    ctx context.Context,
    profiles []string,
    deps []dependencyItem,
) (map[string][]dependencyItem, error) {
    results := make(map[string][]dependencyItem, len(profiles))
    errs := make([]error, len(profiles))
    var mu sync.Mutex
    var wg sync.WaitGroup

    for idx, prof := range profiles {
        wg.Add(1)
        go func(i int, profileName string) {
            defer wg.Done()
            // deep copy so goroutines don't share slice elements
            depsCopy := make([]dependencyItem, len(deps))
            copy(depsCopy, deps)

            slog.Info("report: starting validation",
                "profile", profileName, "total", len(depsCopy))
            start := time.Now()
            if err := validateTargetDependencies(ctx, profileName, depsCopy); err != nil {
                errs[i] = fmt.Errorf("profile %q: %w", profileName, err)
                return
            }

            found, missing := 0, 0
            for _, d := range depsCopy {
                if d.Status == "found" {
                    found++
                } else if d.Status == "missing" {
                    missing++
                }
            }
            slog.Info("report: profile complete",
                "profile", profileName,
                "found", found,
                "missing", missing,
                "elapsed", time.Since(start).Round(time.Millisecond).String(),
            )

            mu.Lock()
            results[profileName] = depsCopy
            mu.Unlock()
        }(idx, prof)
    }

    wg.Wait()
    for _, err := range errs {
        if err != nil {
            return nil, err
        }
    }
    return results, nil
}
```

### Step 8 - Build multi-profile output rows

```go
// buildReportRows builds output rows for multi-profile report mode.
// For table/CSV: returns []map[string]interface{} (includes warning_<profile> keys for CSV).
// For JSON/YAML: returns []reportItem.
func buildReportRows(deps []dependencyItem, profiles []string,
    profileResults map[string][]dependencyItem) ([]map[string]interface{}, []reportItem) {

    tableRows := make([]map[string]interface{}, len(deps))
    jsonRows := make([]reportItem, len(deps))

    for i, dep := range deps {
        id := dep.Path + "." + dep.Type
        row := map[string]interface{}{
            "id":   id,
            "path": dep.Path,
            "type": dep.Type,
        }
        ri := reportItem{
            ID:       id,
            Path:     dep.Path,
            Type:     dep.Type,
            Profiles: make(map[string]profileResult, len(profiles)),
        }
        for _, prof := range profiles {
            profDeps := profileResults[prof]
            status, warning := "", ""
            if i < len(profDeps) {
                status = profDeps[i].Status
                warning = profDeps[i].Warning
            }
            key := strings.ReplaceAll(prof, "-", "_")
            row["status_"+key] = status
            row["warning_"+key] = warning
            ri.Profiles[prof] = profileResult{Status: status, Warning: warning}
        }
        tableRows[i] = row
        jsonRows[i] = ri
    }
    return tableRows, jsonRows
}
```

### Step 9 - Wire report output in `RunE`

After resolving and optionally filtering deps:

```go
if len(reportProfiles) > 0 {
    profileResults, err := validateMultiProfile(ctx, reportProfiles, deps)
    if err != nil {
        return err
    }

    tableRows, jsonRows := buildReportRows(deps, reportProfiles, profileResults)

    if outputFmt == "mermaid" {
        return fmt.Errorf("--output mermaid is not supported with --report")
    }
    if outputFmt == "json" || outputFmt == "yaml" {
        f, err := getFormatter()
        if err != nil {
            return err
        }
        return f.Format(jsonRows, nil)
    }

    // Table and CSV: dynamic columns
    f, err := getFormatter()
    if err != nil {
        return err
    }
    idFunc := func(v interface{}) string {
        row, _ := v.(map[string]interface{})
        id, _ := row["id"].(string)
        return id
    }
    cols := []output.Column{
        {Header: "PATH", Field: "id", Width: 90, Func: idFunc},
    }
    for _, prof := range reportProfiles {
        key := strings.ReplaceAll(prof, "-", "_")
        profCopy := prof
        cols = append(cols, output.Column{
            Header: fmt.Sprintf("STATUS (%s)", profCopy),
            Field:  "status_" + key,
            Width:  10 + len(profCopy),
            Func:   makeProfileStatusFunc(key),
        })
    }
    if outputFmt == "csv" {
        for _, prof := range reportProfiles {
            key := strings.ReplaceAll(prof, "-", "_")
            cols = append(cols, output.Column{
                Header: fmt.Sprintf("WARNING (%s)", prof),
                Field:  "warning_" + key,
            })
        }
    }
    return f.Format(tableRows, cols)
}
```

Add `makeProfileStatusFunc` helper:

```go
func makeProfileStatusFunc(profileKey string) func(interface{}) string {
    return func(v interface{}) string {
        row, _ := v.(map[string]interface{})
        status, _ := row["status_"+profileKey].(string)
        if noColor {
            return status
        }
        switch status {
        case "found":
            return lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true).Render(status)
        case "missing":
            return lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Render(status)
        default:
            return lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render(status)
        }
    }
}
```

### Step 10 - Update single-profile columns in `RunE`

Replace the existing columns block:

```go
pathFunc := func(v interface{}) string {
    row, _ := v.(map[string]interface{})
    path, _ := row["path"].(string)
    typ, _ := row["type"].(string)
    return path + "." + typ
}
columns := []output.Column{
    {Header: "PATH", Field: "path", Width: 90, Func: pathFunc},
}
if targetProfile != "" {
    columns = append(columns,
        output.Column{
            Header: fmt.Sprintf("STATUS (%s)", targetProfile),
            Field:  "status",
            Width:  10 + len(targetProfile),
            Func:   targetStatusFunc,
        },
    )
    hasWarnings := false
    for _, d := range deps {
        if d.Warning != "" {
            hasWarnings = true
            break
        }
    }
    if hasWarnings {
        columns = append(columns,
            output.Column{Header: "WARNING", Field: "warning", Width: 40},
        )
    }
}
```

Note: `targetStatusFunc` reads `row["targetStatus"]` - update it to read `row["status"]` to
match the renamed JSON tag.

### Step 11 - Update `targetStatusFunc`

Change the key it reads from `"targetStatus"` to `"status"`:

```go
func targetStatusFunc(v interface{}) string {
    row, _ := v.(map[string]interface{})
    status, _ := row["status"].(string)   // was "targetStatus"
    // ... rest unchanged ...
}
```

### Step 12 - Documentation (`docs/documentation/package.md`)

Update the `## package dependencies` section:

- Flags table: add `--report`, remove `SOURCE` from output columns, rename `TARGET` to
  `STATUS (profile)`, note WARNING is conditional
- Add `--report` usage description and examples (bash + PowerShell)
- Update output columns table
- Note that `--report` and `--target-profile` are mutually exclusive
- Add CI/CD note that `--report` supports `IICS_TARGET_*` env vars only for single-profile
  `--target-profile` (multi-profile report requires named config profiles)

### Step 13 - Regenerate completions

```bash
make completions
```

---

## Output Columns

### Default mode (no `--target-profile`, no `--report`)

| Header | Field | Width | Notes |
| ------ | ----- | ----- | ----- |
| PATH | path (via Func: path.type) | 90 | Full path + type concatenated |

### Single-profile mode (`--target-profile <name>`)

| Header | Field | Width | Notes |
| ------ | ----- | ----- | ----- |
| PATH | path (via Func) | 90 | path.type |
| STATUS (name) | status | 10 + len(name) | found/missing/unknown; color-coded |
| WARNING | warning | 40 | Only shown when at least one warning exists |

### Multi-profile report mode (`--report dev,qa`)

| Header | Field | Width | Notes |
| ------ | ----- | ----- | ----- |
| PATH | id | 90 | path.type (from map key `id`) |
| STATUS (dev) | status_dev | 10 + len(name) | Color-coded per profile |
| STATUS (qa) | status_qa | 10 + len(name) | Color-coded per profile |
| WARNING (dev) | warning_dev | auto | CSV only |
| WARNING (qa) | warning_qa | auto | CSV only |

---

## Acceptance Criteria

- [ ] PATH column shows `path.type` in all output modes (table, json, csv, yaml)
- [ ] SOURCE column is gone from all outputs
- [ ] Single-profile TARGET column is renamed to `STATUS (<profilename>)`
- [ ] `dependencyItem.Status` JSON tag is `status` (was `targetStatus`)
- [ ] WARNING column is omitted when all assets are `found` (single-profile)
- [ ] WARNING column appears when any asset has a warning (single-profile)
- [ ] `--target-profile` still works unchanged for single-profile validation
- [ ] `--report dev` works as a single-profile report (same layout as `--target-profile`)
- [ ] `--report dev,qa` queries both orgs in parallel and produces side-by-side columns
- [ ] `--report dev --report qa` is equivalent to `--report dev,qa`
- [ ] `--report` and `--target-profile` together return a descriptive error
- [ ] `--publish --report dev,qa` works (report mode with publishable types only)
- [ ] Multi-profile JSON output uses `profiles` map per item
- [ ] Multi-profile CSV includes `status_<profile>` and `warning_<profile>` columns
- [ ] Multi-profile table has no WARNING columns (warnings available via `-o json`)
- [ ] `slog.Debug` progress tick logged every 50 items during validation
- [ ] `slog.Info` goroutine start + completion logged per profile in report mode
- [ ] `--order-by source` returns an error (field removed)
- [ ] `--order-by status` works (was `targetStatus`)
- [ ] Column widths are consistent; headers and data align correctly
- [ ] `-o mermaid --report` returns an unsupported error
- [ ] `go build ./... && go vet ./...` pass with no issues
- [ ] `golangci-lint run ./...` passes
- [ ] `docs/documentation/package.md` updated
- [ ] `make completions` run and `completions/` committed

---

## Do NOT

- Add new Go module dependencies - use standard `sync` package only
- Modify `internal/output/`, `internal/config/`, or `cmd/root.go`
- Refactor code outside the scope of this CR
- Add `Co-Authored-By` trailers to commit messages
- Use em dashes in documentation Markdown
- Support `--report` with `-o mermaid`
- Add backward-compatibility shims for the removed `source` field or renamed `targetStatus`
- Add a `--fields` flag for column selection (out of scope for this CR)
