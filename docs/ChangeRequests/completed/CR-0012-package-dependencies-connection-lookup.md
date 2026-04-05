# CR-0012: Fix Connection Lookup, Auto-size Columns, Color-code Target Status

| Field     | Value                                       |
| --------- | ------------------------------------------- |
| ID        | CR-0012                                     |
| Status    | New                                         |
| Priority  | Medium                                      |
| Component | `cmd/package.go`                            |

---

## Summary

Three targeted improvements to `iics package dependencies --target-profile`:

1. `Connection` type assets (CDI connections, typically under `/SYS`) must be validated
   against a target org via the V2 Connection API (`GET /api/v2/connection/name/<name>`),
   not the V3 Lookup API. The Lookup API does not reliably resolve these assets by
   path+type, which causes them to always appear as "unknown" or "missing" even when
   present in the target org.

2. Output column widths are hardcoded to large fixed minimums (PATH=70, TYPE=22,
   WARNING=50). Since `Column.Width` is already a floor, setting it to 0 lets the
   table renderer auto-size every column to `max(header_len, longest_cell_value)`,
   producing a compact table that fits the actual data.

3. The TARGET column shows plain "found" / "missing" / "unknown" text. Color coding
   would make validation results instantly readable without scanning each cell.

---

## Issue 1 - Connection assets always appear "missing" or "unknown" in target validation

### Observed behaviour

Running `iics package dependencies -f pkg.zip --target-profile prod` with a package that
contains a `Connection` type dependency (e.g., `/SYS/Telematics`) produces:

```
SYS/Telematics   Connection   package   unknown   lookup error: ...
```

or

```
SYS/Telematics   Connection   package   missing   asset not found in target org
```

even when the connection exists in the target org.

### Root cause

`validateTargetDependencies` in `cmd/package.go` calls
`tc.Lookup(ctx, []client.LookupObject{{Path: d.Path, Type: d.Type}})` for every
dependency type including `Connection`. CDI connections are stored outside the V3
object model and cannot be reliably resolved by the V3 Lookup API using a path+type
pair. The correct API is `GET /api/v2/connection/name/<name>`.

`GetConnectionByName` is already implemented in `internal/client/connections.go`.

### Proposed fix

In `validateTargetDependencies`, add a special case before the general Lookup call:

```go
if d.Type == "Connection" {
    name := d.Path
    if idx := strings.LastIndex(d.Path, "/"); idx >= 0 {
        name = d.Path[idx+1:]
    }
    _, connErr := tc.GetConnectionByName(ctx, name)
    if connErr == nil {
        d.TargetStatus = "found"
    } else {
        var apiErr *client.APIError
        if errors.As(connErr, &apiErr) && apiErr.IsNotFound() {
            d.TargetStatus = "missing"
            d.Warning = "connection not found in target org"
        } else {
            d.TargetStatus = "unknown"
            d.Warning = fmt.Sprintf("lookup error: %v", connErr)
        }
    }
    continue
}
```

`strings` is already imported in `cmd/package.go`. `errors` and `client` are already
imported. No new imports required.

---

## Issue 2 - Column widths are unnecessarily wide fixed minimums

### Observed behaviour

The dependencies table always renders with very wide columns regardless of actual content:

```
PATH                                                                   TYPE                   SOURCE
Explore/ZZ_TEST_CLI/Connections/TestServiceConn1                       AI_CONNECTION          package
```

The PATH column is padded to 70 characters. For packages with short paths the table
is far wider than necessary.

### Root cause

`Column.Width` in the `output` package acts as a minimum floor. The table renderer
computes `max(header_len, Column.Width, max_cell_content_len)`. Setting PATH to 70
forces a 70-char column even when all paths are under 50 characters.

Current widths in `newPackageDependenciesCmd()`:

| Column  | Current Width | Header len |
| ------- | ------------- | ---------- |
| PATH    | 70            | 4          |
| TYPE    | 22            | 4          |
| SOURCE  | 10            | 6          |
| TARGET  | 10            | 6          |
| WARNING | 50            | 7          |

### Proposed fix

Set `Width: 0` on all five columns. The renderer already auto-sizes to
`max(header_len, longest_cell_value)` when `Width` is 0:

```go
columns := []output.Column{
    {Header: "PATH",   Field: "path"},
    {Header: "TYPE",   Field: "type"},
    {Header: "SOURCE", Field: "source"},
}
if targetProfile != "" {
    columns = append(columns,
        output.Column{Header: "TARGET",  Field: "targetStatus", Func: targetStatusFunc},
        output.Column{Header: "WARNING", Field: "warning"},
    )
}
```

(Note: `Width` omitted = zero value = auto-size.)

---

## Issue 3 - Debug log messages do not identify which target org is being queried

### Observed behaviour

```
level=DEBUG msg="looking up dependency in target org" path=Connections/LexisNexis/GW-Update-Police-Order-Api-v1 type=AI_CONNECTION
level=DEBUG msg="dependency found in target org"      path=Connections/LexisNexis/GW-Update-Police-Order-Api-v1 type=AI_CONNECTION
```

When multiple target profiles are in use (e.g., staging and production), the logs give
no indication of which org the lookup is happening against. The top-level
`"validating dependencies against target org"` message logs `profile` but the per-dep
messages do not, making it hard to filter or correlate log lines.

### Root cause

The per-dep `slog.Debug` calls in `validateTargetDependencies` only log `path` and
`type`. Neither `targetProfileName` nor the actual IICS org name is included.

The `Client` struct does not expose an `OrgName()` getter - it stores only `sessionID`
and `baseAPIURL` after login. However, `Client.SetOnLoginSuccess(func(*LoginResponse))`
fires on the first API call and receives the full `LoginResponse`, which includes
`UserInfo.OrgName`.

### Proposed fix

Register an `onLoginSuccess` callback immediately after creating `tc` to capture the
org name into a local variable:

```go
var targetOrgName string
tc.SetOnLoginSuccess(func(resp *client.LoginResponse) {
    targetOrgName = resp.UserInfo.OrgName
})
```

Then add `"profile", targetProfileName, "org", targetOrgName` to all six `slog` calls
in the loop body. For the very first "looking up" log line the org name will be empty
(login has not happened yet); every subsequent line will have both profile and org name.

Expected output after the fix:

```
level=INFO  msg="validating dependencies against target org"      profile=prod count=12
level=DEBUG msg="looking up dependency in target org"             profile=prod org=""                  path=Connections/LexisNexis/GW-Update... type=AI_CONNECTION
level=DEBUG msg="dependency found in target org"                  profile=prod org="National Interstate - PROD" path=Connections/LexisNexis/GW-Update... type=AI_CONNECTION
level=DEBUG msg="looking up dependency in target org"             profile=prod org="National Interstate - PROD" path=SYS/Telematics type=Connection
```

---

## Issue 4 - TARGET column has no visual differentiation

### Observed behaviour

All target statuses ("found", "missing", "unknown") render in the same default text
color, requiring the user to read every cell to identify missing dependencies.

### Proposed fix

Add a package-level `targetStatusFunc` in `cmd/package.go` using the same lipgloss
pattern as `agentActiveFunc` in `cmd/runtime.go`:

```go
func targetStatusFunc(v interface{}) string {
    row, _ := v.(map[string]interface{})
    status, _ := row["targetStatus"].(string)
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
```

Color mapping:

- `found` - green (ANSI 2), bold
- `missing` - orange (256-color 208)
- `unknown` / anything else - yellow (ANSI 3)

`github.com/charmbracelet/lipgloss` is already a direct dependency (used in `cmd/runtime.go`).
Add import to `cmd/package.go`.

---

## Acceptance Criteria

1. `Connection` type deps use `GetConnectionByName` for target validation
2. Connection name extracted as last path segment (e.g. `SYS/Telematics` -> `Telematics`)
3. 404 from Connection API sets `targetStatus="missing"`, `warning="connection not found in target org"`
4. Other Connection API errors set `targetStatus="unknown"`, `warning` with error detail
5. All five output columns have no hardcoded `Width` (auto-sized from content)
6. `targetStatusFunc` renders "found" in green/bold, "missing" in orange, others in yellow
7. `noColor` global disables color output (plain text returned)
8. All existing behavior for non-Connection assets is unchanged
9. `SetOnLoginSuccess` callback captures `resp.UserInfo.OrgName` into local `targetOrgName`
10. All six per-dep `slog` calls in `validateTargetDependencies` include `"profile", targetProfileName, "org", targetOrgName`
11. `go build ./...` and `go vet ./...` pass with no issues
12. `golangci-lint run ./...` reports no new issues

---

## Files to Modify

| File | Change |
| ---- | ------ |
| `cmd/package.go` | Add `targetStatusFunc`; route `Connection` type in `validateTargetDependencies`; set `Width: 0` on all dependency columns; add `lipgloss` import |

## Do NOT

- Modify `internal/client/connections.go` (already has `GetConnectionByName`)
- Modify `internal/output/`
- Add new Go module dependencies
- Change behavior for any non-Connection asset types
