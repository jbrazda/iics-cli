# CR-0005: Refine activitylog command - state mapping, --fields selection, nested detail display

---

## CR Type

- [ ] New resource
- [ ] New subcommand
- [x] **Enhancement** - change behaviour of an existing command (modify specific files)
- [x] **Output change** - add/remove/rename columns, change default format, fix display
- [x] **Flag / config change** - add/rename/remove a CLI flag or config field

---

## Problem

The `activitylog` command was added with a basic implementation. Three issues make it unsuitable
for production use:

1. The `state` field is displayed as a raw integer (1, 2, 3, 4) instead of a human-readable label
   such as SUCCESS, ERRORS, FAILED, NOT STARTED.
2. There is no way to select which fields to include in table or CSV output. The column set is
   hardcoded and cannot be customised per invocation.
3. `activitylog get` silently drops all nested data (child entries, subtask entries, session
   variables, transformation entries, item attributes) because the `ActivityLogEntry` struct does
   not define typed fields for them and the display logic renders only a flat table.

---

## Desired Change

**Informatica API docs:**
`https://docs.informatica.com/integration-cloud/data-integration/current-version/rest-api-reference/platform-rest-api-version-2-resources/activity-logs/logs-for-completed-jobs.html`

### 1. State-to-label mapping

The `state` column in both `list` and `get` output must display a human-readable label:

| state value | label       |
| ----------- | ----------- |
| 1           | SUCCESS     |
| 2           | ERRORS      |
| 3           | FAILED      |
| 4           | NOT STARTED |

This is implemented via a `Column.Func` callback on the `state` column - no changes to the output
layer are needed.

### 2. `--fields` flag for column selection

Both `activitylog list` and `activitylog get` gain a `--fields` flag accepting a comma-separated
list of API field names (JSON tag names):

```bash
iics activitylog list --fields id,objectName,state,errorMsg
iics activitylog get <id> --fields id,objectName,state,runId,failedTargetRows,errorMsg
```

Rules:

- Default value: `id,objectName,type,state,runId,startTimeUtc,endTimeUtc,startedBy`
  (removes `runContextType` from original default; adds `runId`)
- Field names match JSON tags exactly (see full column map in Implementation Instructions)
- Unknown field names are silently skipped
- `--fields` affects table and CSV output only; JSON and YAML always include all struct fields
- Header labels are taken from the predefined column map (capitalised, human-readable)

### 3. Expanded `ActivityLogEntry` struct

Add the following missing top-level fields to `ActivityLogEntry`:

| JSON tag               | Go type | Notes                    |
| ---------------------- | ------- | ------------------------ |
| `totalSuccessRows`     | int64   | optional                 |
| `totalFailedRows`      | int64   | optional                 |
| `stopOnError`          | bool    | optional                 |
| `hasStopOnErrorRecord` | bool    | optional                 |
| `contextExternalId`    | string  | optional                 |

Add typed structs for nested objects (exact field names must be verified from the API response
examples noted in the Implementation Instructions below before writing code):

- `SubTaskEntry`
- `LogEntryItemAttr`
- `SessionVariable`
- `TransformationEntry`

Wire all four into `ActivityLogEntry` via the JSON tags `subTaskEntries`, `logEntryItemAttrs`,
`sessionVariables`, `transformationEntries`.

### 4. `activitylog get` multi-section display

For **table and CSV** output, `activitylog get` prints:

1. A main table with the selected fields (same columns as `list`)
2. If `entry.Entries` is non-empty: a labelled section `Entries:` with a table using the
   same selected columns
3. If `entry.SubTaskEntries` is non-empty: a labelled section `Sub-task Entries:` with a table
   using the columns defined for `SubTaskEntry`
4. If `entry.LogEntryItemAttrs` is non-empty: a labelled section `Item Attributes:` with a table
   using the columns defined for `LogEntryItemAttr`
5. If `entry.SessionVariables` is non-empty: a labelled section `Session Variables:` with a table
   using the columns defined for `SessionVariable`
6. If `entry.TransformationEntries` is non-empty: a labelled section `Transformation Entries:`
   with a table using the columns defined for `TransformationEntry`

For **JSON and YAML** output, print the full `ActivityLogEntry` struct (all nested fields
included automatically once the struct is populated).

Section labels are printed with `fmt.Fprintf(os.Stdout, "\n%s\n", label)` between tables.

---

## Scope

### Files to MODIFY

```text
internal/client/activitylogs.go      # add missing top-level fields + 4 new nested structs
internal/client/activitylogs_test.go # update fixtures; add tests for expanded struct fields
cmd/activitylog.go                   # column map, buildActivityLogColumns helper,
                                     # --fields flag on list and get, nested-table logic in get
docs/documentation/activitylog.md   # update flags, output columns, add --fields examples
completions/                         # regenerate via make completions
```

### Files to READ (context only - do NOT modify)

```text
docs/CLAUDE.md
internal/client/client.go
internal/output/formatter.go
internal/output/table.go
cmd/root.go
```

### Forbidden (do NOT touch)

```text
internal/output/    # output layer is correct as-is; no changes needed
cmd/root.go         # activitylog already registered
```

---

## API Details

| Field          | Value                                |
| -------------- | ------------------------------------ |
| API version    | V2 (`api/v2`)                        |
| HTTP method    | GET                                  |
| Endpoint path  | `api/v2/activity/activityLog`        |
| Session header | auto-detected (v2: `icSessionId`)    |
| Request body   | no                                   |
| Response type  | array (list) / single object (get)   |

### Full response field inventory

Top-level fields on `activityLogEntry`:

| JSON tag               | Go type | Already in struct |
| ---------------------- | ------- | :---------------: |
| `id`                   | string  | yes               |
| `type`                 | string  | yes               |
| `objectId`             | string  | yes               |
| `objectName`           | string  | yes               |
| `runId`                | int64   | yes               |
| `agentId`              | string  | yes               |
| `runtimeEnvironmentId` | string  | yes               |
| `startTime`            | string  | yes               |
| `endTime`              | string  | yes               |
| `startTimeUtc`         | string  | yes               |
| `endTimeUtc`           | string  | yes               |
| `state`                | int     | yes               |
| `failedSourceRows`     | int64   | yes               |
| `successSourceRows`    | int64   | yes               |
| `failedTargetRows`     | int64   | yes               |
| `successTargetRows`    | int64   | yes               |
| `scheduleName`         | string  | yes               |
| `errorMsg`             | string  | yes               |
| `startedBy`            | string  | yes               |
| `runContextType`       | string  | yes               |
| `isStopped`            | bool    | yes               |
| `totalSuccessRows`     | int64   | **no - add**      |
| `totalFailedRows`      | int64   | **no - add**      |
| `stopOnError`          | bool    | **no - add**      |
| `hasStopOnErrorRecord` | bool    | **no - add**      |
| `contextExternalId`    | string  | **no - add**      |

### Nested object schemas

> **IMPORTANT:** The field names below are stubs. Before writing any struct code, obtain the
> real API response examples (provided by the user) and replace every stub field with the
> exact JSON tag names from the response. Wrong JSON tags are the primary source of bugs.

#### `subTaskEntries` (JSON array key: `subTaskEntries`)

Schema confirmed as `[]ActivityLogEntry` (same recursive structure as `entries`).
No separate struct needed.

#### `logEntryItemAttrs` (JSON key: `logEntryItemAttrs`)

**This is a `map[string]string`, not an array.** Example from API:

```json
"logEntryItemAttrs": {
    "CONSUMED_COMPUTE_UNITS": "0.0",
    "ERROR_CODE": "0",
    "IS_SERVER_LESS": "false",
    "REQUESTED_COMPUTE_UNITS": "0.0",
    "Session Log File Name": "s_mtt_....log"
}
```

Go type: `map[string]string`. For tabular display, convert to sorted `[]kvPair{Attribute, Value}`.

#### `sessionVariables` (JSON key: `sessionVariables`)

Not observed in API responses; treat as `map[string]string` (same pattern as `logEntryItemAttrs`).

#### `TransformationEntry` (JSON array key: `transformationEntries`)

Confirmed schema from API response:

```json
{
    "@type": "transformationLogEntry",
    "id": "141332309",
    "txName": "FFSource2",
    "txType": "SOURCE",
    "successRows": 600,
    "failedRows": 0,
    "affectedRows": 600
}
```

```go
type TransformationEntry struct {
    ID           string `json:"id"`
    TxName       string `json:"txName"`
    TxType       string `json:"txType"`
    SuccessRows  int64  `json:"successRows"`
    AffectedRows int64  `json:"affectedRows,omitempty"`
    FailedRows   int64  `json:"failedRows"`
}
```

---

## Implementation Instructions

> Read `docs/CLAUDE.md` and `internal/output/formatter.go` before starting.

### Step 1 - Expand `internal/client/activitylogs.go`

1. Add the five missing top-level fields to `ActivityLogEntry` (with `omitempty`):

   ```go
   TotalSuccessRows     int64  `json:"totalSuccessRows,omitempty"`
   TotalFailedRows      int64  `json:"totalFailedRows,omitempty"`
   StopOnError          bool   `json:"stopOnError,omitempty"`
   HasStopOnErrorRecord bool   `json:"hasStopOnErrorRecord,omitempty"`
   ContextExternalID    string `json:"contextExternalId,omitempty"`
   ```

2. Define four new typed structs. **Replace the stub bodies with exact fields from the
   real API response examples before writing this code.** Follow the same struct conventions
   (JSON tags match API exactly, `omitempty` on optional fields, typed structs for nested objects).

3. Add the four new slice fields to `ActivityLogEntry`:

   ```go
   SubTaskEntries        []SubTaskEntry         `json:"subTaskEntries,omitempty"`
   LogEntryItemAttrs     []LogEntryItemAttr     `json:"logEntryItemAttrs,omitempty"`
   SessionVariables      []SessionVariable      `json:"sessionVariables,omitempty"`
   TransformationEntries []TransformationEntry  `json:"transformationEntries,omitempty"`
   ```

4. No changes to `ListActivityLogs` or `GetActivityLog` methods - the client layer is correct.

### Step 2 - Update `internal/client/activitylogs_test.go`

1. Update the fixture in `TestListActivityLogs` to include at least one of the new top-level
   fields (`totalSuccessRows`) and assert its value.
2. Update the fixture in `TestGetActivityLog` to include a non-empty `entries` slice and assert
   that the slice is populated after unmarshalling.
3. Do not add tests for the nested struct content beyond basic nil checks - functional display
   testing is done manually.

### Step 3 - Rewrite `cmd/activitylog.go`

#### 3a. State Func

Add a package-level function (not exported) for state label conversion:

```go
func activityLogStateLabel(v interface{}) string {
    row, ok := v.(map[string]interface{})
    if !ok {
        return ""
    }
    // JSON numbers unmarshal as float64
    state, _ := row["state"].(float64)
    switch int(state) {
    case 1:
        return "SUCCESS"
    case 2:
        return "ERRORS"
    case 3:
        return "FAILED"
    case 4:
        return "NOT STARTED"
    default:
        if v, ok := row["state"]; ok {
            return fmt.Sprintf("%v", v)
        }
        return ""
    }
}
```

#### 3b. Column map

Add a package-level `activityLogColumnMap` mapping every available JSON field name to a
pre-configured `output.Column`. The `state` entry must set `Func: activityLogStateLabel`.
Include columns for all 25 top-level fields (existing + new ones added in Step 1).

```go
var activityLogColumnMap = map[string]output.Column{
    "id":                   {Header: "ID", Field: "id", Width: 20},
    "objectName":           {Header: "NAME", Field: "objectName", Width: 30},
    "type":                 {Header: "TYPE", Field: "type", Width: 10},
    "state":                {Header: "STATE", Field: "state", Width: 12, Func: activityLogStateLabel},
    "runId":                {Header: "RUN ID", Field: "runId", Width: 12},
    "objectId":             {Header: "OBJECT ID", Field: "objectId", Width: 24},
    "agentId":              {Header: "AGENT ID", Field: "agentId", Width: 24},
    "runtimeEnvironmentId": {Header: "RUNTIME ENV", Field: "runtimeEnvironmentId", Width: 24},
    "startTime":            {Header: "START TIME (ET)", Field: "startTime", Width: 22},
    "endTime":              {Header: "END TIME (ET)", Field: "endTime", Width: 22},
    "startTimeUtc":         {Header: "START TIME", Field: "startTimeUtc", Width: 22},
    "endTimeUtc":           {Header: "END TIME", Field: "endTimeUtc", Width: 22},
    "failedSourceRows":     {Header: "FAILED SRC", Field: "failedSourceRows", Width: 12},
    "successSourceRows":    {Header: "SUCCESS SRC", Field: "successSourceRows", Width: 12},
    "failedTargetRows":     {Header: "FAILED TGT", Field: "failedTargetRows", Width: 12},
    "successTargetRows":    {Header: "SUCCESS TGT", Field: "successTargetRows", Width: 12},
    "scheduleName":         {Header: "SCHEDULE", Field: "scheduleName", Width: 20},
    "errorMsg":             {Header: "ERROR", Field: "errorMsg", Width: 40},
    "startedBy":            {Header: "STARTED BY", Field: "startedBy", Width: 20},
    "runContextType":       {Header: "RUN CONTEXT", Field: "runContextType", Width: 15},
    "isStopped":            {Header: "STOPPED", Field: "isStopped", Width: 8},
    "totalSuccessRows":     {Header: "TOTAL SUCCESS", Field: "totalSuccessRows", Width: 14},
    "totalFailedRows":      {Header: "TOTAL FAILED", Field: "totalFailedRows", Width: 13},
    "stopOnError":          {Header: "STOP ON ERR", Field: "stopOnError", Width: 12},
    "contextExternalId":    {Header: "CONTEXT ID", Field: "contextExternalId", Width: 24},
}

const activityLogDefaultFields = "id,objectName,type,state,runId,startTimeUtc,endTimeUtc,startedBy"
```

#### 3c. `buildActivityLogColumns` helper

```go
func buildActivityLogColumns(fields string) []output.Column {
    names := strings.Split(fields, ",")
    cols := make([]output.Column, 0, len(names))
    for _, name := range names {
        name = strings.TrimSpace(name)
        if col, ok := activityLogColumnMap[name]; ok {
            cols = append(cols, col)
        }
    }
    return cols
}
```

Add `"strings"` to the import block.

#### 3d. Update `newActivitylogListCmd`

- Add `--fields` flag: `cmd.Flags().StringVar(&fields, "fields", activityLogDefaultFields, "comma-separated list of fields to display (table/csv only)")`
- Replace the hardcoded `columns` slice with `buildActivityLogColumns(fields)` inside `RunE`.
- Declare `var fields string` at the top of the function (alongside `var opts`).

#### 3e. Update `newActivitylogGetCmd`

- Add `var fields string` local variable.
- Add `--fields` flag with the same default.
- In `RunE`:

  ```go
  cols := buildActivityLogColumns(fields)
  f, err := getFormatter()
  if err != nil {
      return err
  }

  // JSON/YAML: full struct
  if outputFmt == "json" || outputFmt == "yaml" {
      return f.Format([]client.ActivityLogEntry{*entry}, cols)
  }

  // table/csv: main table first
  if err := f.Format([]client.ActivityLogEntry{*entry}, cols); err != nil {
      return err
  }

  // Nested: child entries (same columns)
  if len(entry.Entries) > 0 {
      fmt.Fprintf(os.Stdout, "\nEntries:\n")
      if err := f.Format(entry.Entries, cols); err != nil {
          return err
      }
  }

  // Nested: sub-task entries (define subTaskCols below based on SubTaskEntry fields)
  if len(entry.SubTaskEntries) > 0 {
      fmt.Fprintf(os.Stdout, "\nSub-task Entries:\n")
      if err := f.Format(entry.SubTaskEntries, subTaskCols); err != nil {
          return err
      }
  }

  // Nested: log entry item attrs
  if len(entry.LogEntryItemAttrs) > 0 {
      fmt.Fprintf(os.Stdout, "\nItem Attributes:\n")
      if err := f.Format(entry.LogEntryItemAttrs, logEntryItemAttrCols); err != nil {
          return err
      }
  }

  // Nested: session variables
  if len(entry.SessionVariables) > 0 {
      fmt.Fprintf(os.Stdout, "\nSession Variables:\n")
      if err := f.Format(entry.SessionVariables, sessionVariableCols); err != nil {
          return err
      }
  }

  // Nested: transformation entries
  if len(entry.TransformationEntries) > 0 {
      fmt.Fprintf(os.Stdout, "\nTransformation Entries:\n")
      if err := f.Format(entry.TransformationEntries, transformationEntryCols); err != nil {
          return err
      }
  }

  return nil
  ```

  Define `subTaskCols`, `logEntryItemAttrCols`, `sessionVariableCols`, and
  `transformationEntryCols` as package-level `[]output.Column` variables once the nested
  struct fields are confirmed from the real API response examples.

  Add `"fmt"` and `"os"` to the import block.

### Step 4 - Update `docs/documentation/activitylog.md`

1. Add `--fields` flag row to both the `list` and `get` flag tables.
2. Expand the output columns table to list all 25 available field names with descriptions.
3. Add usage examples for `--fields`.
4. Document the nested section tables shown by `activitylog get`.

### Step 5 - Regenerate completions

```bash
make completions
```

Commit updated `completions/` files together with the code change.

### Step 6 - Verify

```bash
/opt/local/bin/go build ./...
/opt/local/bin/go test ./internal/client/... -v -run TestListActivityLogs
/opt/local/bin/go test ./internal/client/... -v -run TestGetActivityLog
/opt/local/bin/go test ./...
/opt/local/bin/go vet ./...
```

All must pass with zero new errors or warnings.

---

## Output Columns

### `activitylog list` and `activitylog get` main table (selectable via `--fields`)

| Header          | Field (JSON tag)       | Width | Notes                              |
| --------------- | ---------------------- | ----- | ---------------------------------- |
| ID              | `id`                   | 20    | default                            |
| NAME            | `objectName`           | 30    | default                            |
| TYPE            | `type`                 | 10    | default; task type values          |
| STATE           | `state`                | 12    | default; mapped via Func           |
| RUN ID          | `runId`                | 12    | default                            |
| START TIME      | `startTimeUtc`         | 22    | default (UTC)                      |
| END TIME        | `endTimeUtc`           | 22    | default (UTC)                      |
| STARTED BY      | `startedBy`            | 20    | default                            |
| OBJECT ID       | `objectId`             | 24    | optional                           |
| AGENT ID        | `agentId`              | 24    | optional                           |
| RUNTIME ENV     | `runtimeEnvironmentId` | 24    | optional                           |
| START TIME (ET) | `startTime`            | 22    | optional; Eastern Time             |
| END TIME (ET)   | `endTime`              | 22    | optional; Eastern Time             |
| FAILED SRC      | `failedSourceRows`     | 12    | optional                           |
| SUCCESS SRC     | `successSourceRows`    | 12    | optional                           |
| FAILED TGT      | `failedTargetRows`     | 12    | optional                           |
| SUCCESS TGT     | `successTargetRows`    | 12    | optional                           |
| SCHEDULE        | `scheduleName`         | 20    | optional                           |
| ERROR           | `errorMsg`             | 40    | optional                           |
| RUN CONTEXT     | `runContextType`       | 15    | optional                           |
| STOPPED         | `isStopped`            | 8     | optional                           |
| TOTAL SUCCESS   | `totalSuccessRows`     | 14    | optional                           |
| TOTAL FAILED    | `totalFailedRows`      | 13    | optional                           |
| STOP ON ERR     | `stopOnError`          | 12    | optional                           |
| CONTEXT ID      | `contextExternalId`    | 24    | optional                           |

### Nested section column sets

Define after receiving real API response examples. Each nested type gets its own fixed
`[]output.Column` variable (not selectable via `--fields`).

---

## Acceptance Criteria

- [ ] `iics activitylog list` shows STATE column as SUCCESS / ERRORS / FAILED / NOT STARTED
- [ ] `iics activitylog list --fields id,objectName,state,errorMsg` shows exactly those 4 columns
- [ ] `iics activitylog list --output json` includes all struct fields regardless of `--fields`
- [ ] `iics activitylog list --output csv` respects `--fields`
- [ ] `iics activitylog get <id>` (table) prints main table then labelled nested sections when non-empty
- [ ] `iics activitylog get <id> --output json` prints full nested structure
- [ ] `go build ./...` succeeds with no errors
- [ ] `go test ./...` passes with no failures
- [ ] `go vet ./...` reports no issues
- [ ] No unrelated code was modified
- [ ] Two-layer rule respected: no API logic in `cmd/`, no Cobra in `internal/client/`

---

## Do NOT

- Refactor, reformat, or add comments to code outside the CR scope
- Modify any file not listed in the Scope section
- Add error handling for scenarios that cannot happen
- Create helpers or abstractions used only once (exception: `buildActivityLogColumns` is used in
  both `list` and `get`, so it is justified)
- Use `os.Exit()` - return errors from `RunE`
- Use tablewriter v0.x API
- Hard-code base API paths - use `BaseAPIPathV2`
- Guess JSON field names for the nested structs - use the exact tags from the API response examples
- Add `Co-Authored-By` trailers to commit messages

---

## Implementation Notes (filled in by Claude during implementation)

- Nested struct field names (`SubTaskEntry`, `LogEntryItemAttr`, `SessionVariable`,
  `TransformationEntry`) are stubs. The implementor must obtain real API response examples
  (ask the user) and replace every stub field before writing code for those structs.
