---
id: CR-0019
title: objects dependencies - pipe input, multi-target report, --publish flag, publish-compatible output
status: new
priority: high
affects: cmd/objects.go, internal/client/objects.go, docs/documentation/objects.md
---

# CR-0019: objects dependencies - pipe input, multi-target report, --publish flag, publish-compatible output

## Background

CI/CD pipelines need to determine which assets a tagged set of objects depends on,
then verify that those dependencies exist in one or more target environments before
importing or publishing a package. Today this requires multiple manual steps and
custom scripting. This CR extends `objects dependencies` to support:

1. Accepting a JSON array of objects piped from `objects list` (multi-object mode)
2. Checking dependency presence across multiple target profiles (`--targets`)
3. Restricting output to publishable asset types only (`--publish`)
4. Producing output that can be piped directly into `publish start` / `publish run`

These capabilities mirror those already present in `package dependencies` but operate
on live object queries rather than package manifests, enabling tag-driven impact analysis.

---

## Feature 1: Pipe JSON from `objects list`

### Description

When `--id` is not provided and stdin is not a terminal, read a JSON array from stdin.
Each element must have at least an `id` field (the `Object` struct from `objects list`
output satisfies this). Collect dependencies for all input objects, deduplicate by
`appContextId`, and output the combined set.

### Example

```bash
iics objects list -q "tag==Project_sprint_9" --output json \
  | iics objects dependencies --ref-type uses
```

### Implementation notes

- In `newObjectsDependenciesCmd()` RunE:
  - If `--id` is empty and `!isatty.IsTerminal(int(os.Stdin.Fd()))`: read and
    unmarshal stdin as `[]struct{ ID string \`json:"id"\` }`
  - If `--id` is empty and stdin is a terminal: return a usage error (same as today)
- Loop over each ID, call `GetAllObjectDependencies()` (added in BUG-0006), append
  results to a map keyed by `AppContextID` to deduplicate
- Output the deduplicated slice using the existing table columns

---

## Feature 2: Multi-target comparison (`--targets`)

### Description

Add a `--targets` flag accepting a comma-separated list of profile names. When set,
after collecting all dependencies the command validates each dependency against each
target profile using the Lookup API, producing a cross-profile status matrix.

Output columns: `path`, `type`, and one `STATUS:<profile>` column per target.
Status values: `found`, `missing`, `unknown` (same semantics and color scheme as
`package dependencies --report`).

### Example

```bash
iics objects list -q "tag==Project_sprint_9" --output json \
  | iics objects dependencies --ref-type uses --targets dev,qa
```

Sample table output:

```
PATH                              TYPE       STATUS:dev   STATUS:qa
Sales/ETL/LoadOrders              MTT        found        missing
Shared/Connections/SalesDB        DTEMPLATE  found        found
```

### Implementation notes

- Add flag: `cmd.Flags().StringSliceVar(&targets, "targets", nil, "comma-separated profiles to validate dependencies against")`
- Reuse the validation logic pattern from `validateTargetDependencies()` in
  `cmd/package.go:966` - load each target profile, call Lookup for each dependency
  path+type, record `found`/`missing`/`unknown`
- Reuse `makeProfileStatusFunc()` from `cmd/package.go:557` for color-coded status
  columns in table output
- For JSON/YAML output, produce a slice of structs:

```go
type depReportItem struct {
    ID       string                   `json:"id"`
    Path     string                   `json:"path"`
    Type     string                   `json:"type"`
    Profiles map[string]profileResult `json:"profiles"`
}
```

  where `profileResult` matches the existing type in `cmd/package.go`.

---

## Feature 3: `--publish` flag

### Description

When `--publish` is set, filter the dependency list to include only asset types that
require publishing after import. This flag requires `--targets` to be set.

Publishable types: `AI_SERVICE_CONNECTOR`, `AI_CONNECTION`, `PROCESS`, `GUIDE`, `TASKFLOW`.

### Example

```bash
iics objects list -q "tag==Project_sprint_9" --output json \
  | iics objects dependencies --ref-type uses --targets dev,qa --publish
```

### Implementation notes

- Add flag: `cmd.Flags().BoolVar(&publishMode, "publish", false, "restrict output to publishable types only; requires --targets")`
- Validate: if `publishMode && len(targets) == 0` return usage error
- After collecting dependencies, filter to publishable types before running the
  target validation loop
- Reuse the `publishableTypes` set (or equivalent constant) already defined in
  `cmd/package.go` - do not duplicate the list

### Sort order when `--publish` is set

When `--publish` is active, results must be sorted by the same `typePriority` order
used in `package dependencies` (`cmd/package.go:489`). This order reflects the publish
dependency chain (connectors must exist before connections, connections before processes):

| Priority | Type                  |
| -------- | --------------------- |
| 1        | `AI_SERVICE_CONNECTOR`|
| 2        | `AI_CONNECTION`       |
| 3        | `PROCESS`             |
| 4        | `GUIDE`               |
| 5        | `TASKFLOW`            |

Within the same type, sort by `path` ascending (same tiebreak as `package dependencies`).
Types not in `typePriority` sort last (map returns `0` for missing keys).

Reuse the `typePriority` map from `cmd/package.go:489` - do not duplicate it.
Apply the same sort logic from `cmd/package.go:931-952`:

```go
sort.Slice(items, func(i, j int) bool {
    pi, pj := typePriority[items[i].Type], typePriority[items[j].Type]
    if pi != pj {
        if pi == 0 { return false }
        if pj == 0 { return true }
        return pi < pj
    }
    return items[i].Path < items[j].Path
})
```

---

## Feature 4: Publish-compatible output

### Description

The `ObjectReference` struct returned by the dependencies API has `path` and `type`
fields but no `location` field. The `publish` command accepts JSON or CSV input and
resolves assets using (in priority order):

1. `location` field: `Explore/<path>.<TYPE>` - preferred
2. `path` + `type` fields together
3. `path` field alone

To enable direct piping from `objects dependencies` into `publish start` / `publish run`,
the JSON and CSV output of `objects dependencies` must include a `location` field
computed as `Explore/<path>.<TYPE>`.

The `package` command does NOT accept object lists via stdin or file - it reads only
from a package ZIP or expanded workspace directory. Compatibility with `package` is
therefore not required.

### Piping into publish

```bash
# Find missing publishable dependencies for tagged objects and publish them to dev
iics objects list -q "tag==Project_sprint_9" --output json \
  | iics objects dependencies --ref-type uses --publish \
  | iics publish run

# Full pipeline: identify missing deps in QA, then publish to QA
iics objects list -q "tag==Project_sprint_9" --output json \
  | iics objects dependencies --ref-type uses --targets qa --publish \
  | iics publish run --profile qa
```

### Implementation notes

**Struct change** - add a computed `Location` field to `ObjectReference` in
`internal/client/objects.go`:

```go
type ObjectReference struct {
    AppContextID string `json:"appContextId"`
    Path         string `json:"path"`
    Type         string `json:"type"`
    UpdatedBy    string `json:"updatedBy,omitempty"`
    UpdateTime   string `json:"updateTime,omitempty"`
    Location     string `json:"location,omitempty"`  // computed: Explore/<path>.<TYPE>
}
```

The `Location` field is not returned by the API. It must be populated in
`GetObjectDependencies()` after unmarshalling the response, by computing
`"Explore/" + ref.Path + "." + ref.Type` for each `ObjectReference`.

This matches the `location` field format already used in the `Object` struct
(`internal/client/objects.go`) and accepted by `parsePublishJSON()` in
`cmd/publish.go:874`.

**CSV output** - the `location` field must appear in the default `--output-fields`
for `objects dependencies` so that `| iics publish run` works without extra flags:

```bash
iics objects dependencies --id <id> --output csv | iics publish run
```

The `publish` CSV parser (`cmd/publish.go:732`) looks for a `LOCATION` column header
(case-insensitive match on the JSON tag name used as column header).

**Output columns** - add `location` to the default column set for table and CSV output:

| Column         | Field          | Description                               |
| -------------- | -------------- | ----------------------------------------- |
| `appContextId` | `appContextId` | Dependent object ID                       |
| `type`         | `type`         | Object type                               |
| `path`         | `path`         | Full path of the dependent object         |
| `location`     | `location`     | Computed publish path: `Explore/path.TYPE`|
| `updatedBy`    | `updatedBy`    | Last modifier                             |

---

## New flag summary

| Flag        | Short | Type     | Default | Description                                                         |
| ----------- | ----- | -------- | ------- | ------------------------------------------------------------------- |
| `--targets` |       | []string | (none)  | Comma-separated profiles to validate dependencies against           |
| `--publish` |       | bool     | false   | Restrict output to publishable types only; requires `--targets`     |

Existing flags `--id`, `--ref-type`, `--limit`, `--skip` are unchanged (see also BUG-0006
which changes `--limit` default to `0`).

---

## Documentation updates required

`docs/documentation/objects.md` must be updated to document:

- New `--targets` and `--publish` flags in the `objects dependencies` flags table
- New output columns for report mode
- New examples for pipe + targets + publish workflows (bash and PowerShell)

---

## Reference implementations

| Pattern | Location |
| ------- | -------- |
| Multi-profile validation loop | `cmd/package.go:966` (`validateTargetDependencies`) |
| Parallel multi-profile runner | `cmd/package.go:576` (`validateMultiProfile`) |
| Report row builder | `cmd/package.go:610` (`buildReportRows`) |
| Color-coded status columns | `cmd/package.go:557` (`makeProfileStatusFunc`) |
| Publishable types constant/check | `cmd/package.go` (search for `publishableTypes` or equivalent) |
| Publish JSON input parser | `cmd/publish.go:874` (`parsePublishJSON`) - shows accepted field names |
| Publish CSV input parser | `cmd/publish.go:725` (`parsePublishCSV`) - shows accepted column names |
| Location field computation | `internal/client/objects.go` `Object.Location` field - same format |
| Stdin JSON reading | Use `io.ReadAll(os.Stdin)` then `json.Unmarshal` |
| TTY detection | `github.com/mattn/go-isatty` - already a dependency |

---

## Files Affected

- `internal/client/objects.go` - add `Location` field to `ObjectReference`; populate in `GetObjectDependencies()`; add `GetAllObjectDependencies()` (prerequisite from BUG-0006)
- `cmd/objects.go` - stdin reading, `--targets`, `--publish`, report output mode, add `location` to output columns
- `docs/documentation/objects.md` - flags, columns, examples including publish pipeline examples
