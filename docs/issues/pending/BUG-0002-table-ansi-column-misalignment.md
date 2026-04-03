---
id: BUG-0002
title: Table columns misalign when cells contain ANSI color codes
status: new
priority: high
affects: internal/output/table.go, cmd/package.go
---

# BUG-0002: Table columns misalign when cells contain ANSI color codes

## Summary

When a column uses a `Func` that returns ANSI-escaped text (e.g. colored `found` /
`missing` cells produced by `targetStatusFunc` or `makeProfileStatusFunc`), the table
renderer miscalculates column widths and padding. All status values appear on one line
without proper column separation.

## Observed output (`--report dev,qa,tst`)

```
PATH                                                                                       STATUS (dev)  STATUS (qa)  STATUS (tst)
─────────────────────────────────────────────────────────────────────────────────────────  ────────────  ───────────  ───────────
Corvel_v1/Corvel-GW-SubmitClaimFeedService.AI_SERVICE_CONNECTOR                            found  found        found
Corvel_v1/Corvel-GetValidationService-v1.AI_SERVICE_CONNECTOR                              found  missing  found
Corvel_v1/GuideWire-CreateInvoice-v1.AI_SERVICE_CONNECTOR                                  found  missing  missing
```

## Expected output

```
PATH                                                                                       STATUS (dev)  STATUS (qa)  STATUS (tst)
─────────────────────────────────────────────────────────────────────────────────────────  ────────────  ───────────  ───────────
Corvel_v1/Corvel-GW-SubmitClaimFeedService.AI_SERVICE_CONNECTOR                            found         found        found
Corvel_v1/Corvel-GetValidationService-v1.AI_SERVICE_CONNECTOR                              found         missing      found
Corvel_v1/GuideWire-CreateInvoice-v1.AI_SERVICE_CONNECTOR                                  found         missing      missing
```

## Root Cause

Two functions in `internal/output/table.go` use `utf8.RuneCountInString` to measure
string width:

### `computeColWidths` (`table.go:75-93`)

```go
l := utf8.RuneCountInString(extractField(row, col))
```

`extractField` calls `col.Func(row)` when a `Func` is set. `makeProfileStatusFunc` and
`targetStatusFunc` return ANSI-escaped strings such as:

```
\x1b[32;1mfound\x1b[0m    (visible width: 5, rune count: 15)
\x1b[38;5;208mmissing\x1b[0m  (visible width: 7, rune count: 18)
```

`utf8.RuneCountInString` counts every byte of the escape sequence as a rune. The
calculated column width becomes 15 or 18 instead of 5 or 7. The column separator line
(built from the inflated width) is correct by accident only for the first column - every
subsequent column inherits the wrong offset.

### `padRight` (`table.go:96-102`)

```go
l := utf8.RuneCountInString(s)
if l >= width {
    return s
}
return s + strings.Repeat(" ", width-l)
```

When `s` is an ANSI string with rune count 15 and `width` is 12 (correct visible width),
`l >= width` is true so no padding is added. The next column starts immediately after the
last ANSI byte with no gap.

## Fix

Add a `visibleLen` helper that strips ANSI SGR sequences before counting runes, and use
it in both `computeColWidths` and `padRight`.

### `visibleLen` helper

```go
// ansiEscape matches ANSI CSI escape sequences (colors, cursor moves, etc.).
var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

// visibleLen returns the number of visible runes in s, ignoring ANSI escape sequences.
func visibleLen(s string) int {
    return utf8.RuneCountInString(ansiEscape.ReplaceAllString(s, ""))
}
```

### Updated `computeColWidths`

```go
for _, row := range rows {
    for i, col := range columns {
        l := visibleLen(extractField(row, col))   // was: utf8.RuneCountInString(...)
        if l > widths[i] {
            widths[i] = l
        }
    }
}
```

### Updated `padRight`

```go
func padRight(s string, width int) string {
    l := visibleLen(s)                           // was: utf8.RuneCountInString(s)
    if l >= width {
        return s
    }
    return s + strings.Repeat(" ", width-l)
}
```

### Additional note - PATH column `Width` hint

The PATH column in `cmd/package.go` is registered with `Width: 90`. When actual
`path.type` strings are longer (real packages commonly produce 100-115 character IDs),
the fixed hint is redundant - `computeColWidths` already auto-sizes to content. Remove
the `Width: 90` hint from the PATH column definition in both single-profile and
multi-profile column builders so the column is always auto-sized to content.

## Files Affected

- `internal/output/table.go` - add `visibleLen`, update `computeColWidths` and `padRight`
- `cmd/package.go` - remove `Width: 90` from PATH column definitions

## Acceptance Criteria

- [ ] All STATUS columns align under their headers when colors are enabled
- [ ] All STATUS columns align under their headers when `--no-color` is used
- [ ] PATH column auto-sizes to the longest `path.type` value in the result set
- [ ] Single-profile `--target-profile` output is correctly aligned
- [ ] Multi-profile `--report dev,qa,tst` output is correctly aligned
- [ ] `go build ./... && go vet ./...` pass
- [ ] `golangci-lint run ./...` passes
