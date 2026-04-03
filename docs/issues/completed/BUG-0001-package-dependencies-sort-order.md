---
id: BUG-0001
title: Incorrect sort order for `package dependencies` with `--publish` flag; no `--order-by` support
status: new
priority: medium
affects: cmd/package.go
---

# BUG-0001: Incorrect sort order for `package dependencies`

## Summary

The `package dependencies` command has two related issues:

1. The `typePriority` map has `AI_CONNECTION` (priority 1) ranked before
   `AI_SERVICE_CONNECTOR` (priority 2), which is the reverse of the correct
   publish order.
2. The type-based sort is applied unconditionally, even when `--publish` is not
   set. Without `--publish`, the output should sort by `Path` only.
3. There is no `--order-by` flag to allow the user to sort the output by any
   field in the dependency list.

---

## Current Behavior

### Wrong publish sort order (`cmd/package.go:486-492`)

```go
var typePriority = map[string]int{
    "AI_CONNECTION":        1,   // WRONG - should be 2
    "AI_SERVICE_CONNECTOR": 2,   // WRONG - should be 1
    "PROCESS":              3,
    "GUIDE":                4,
    "TASKFLOW":             5,
}
```

The publish API requires dependencies to be applied in dependency order.
`AI_SERVICE_CONNECTOR` must exist before an `AI_CONNECTION` can reference it, so
`AI_SERVICE_CONNECTOR` must sort first.

### Type-based sort applied without `--publish` (`cmd/package.go:810-825`)

```go
sort.Slice(items, func(i, j int) bool {
    pi, pj := typePriority[items[i].Type], typePriority[items[j].Type]
    // type-priority comparison runs regardless of publishMode ...
    return items[i].Path < items[j].Path
})
```

When `--publish` is not set, users expect the list in simple alphabetical path
order, not grouped by publishable type.

### No `--order-by` flag

There is no way for the user to request sorting by `Source`, `TargetStatus`, or
any other `dependencyItem` field.

---

## Expected Behavior

### 1. Correct publish sort order

When `--publish` is set, assets must sort in the following order (primary), with
`Path` as the secondary (tiebreaker) sort key:

| Priority | Type                  |
|----------|-----------------------|
| 1        | `AI_SERVICE_CONNECTOR` |
| 2        | `AI_CONNECTION`        |
| 3        | `PROCESS`              |
| 4        | `GUIDE`                |
| 5        | `TASKFLOW`             |

Types not in the map (non-publishable) should sort after all known types.

### 2. Default sort order without `--publish`

When `--publish` is **not** set, sort by `Path` only (ascending, case-insensitive).

### 3. New `--order-by` flag

Add an `--order-by <field>` flag to `package dependencies` that accepts any
field name present in the dependency output:

| Field          | Notes                               |
|----------------|-------------------------------------|
| `path`         | Default when `--publish` is not set |
| `type`         | Sort by asset type string           |
| `source`       | `package` vs `external`            |
| `targetStatus` | `found`, `missing`, `unknown`       |
| `warning`      | Warning message text                |

- When `--publish` is set and `--order-by` is **not** provided, use the publish
  priority order (primary) + `path` (secondary) as described above.
- When `--order-by` is provided alongside `--publish`, the explicit `--order-by`
  field takes precedence and the type-priority sort is skipped.
- Invalid field names should return a descriptive error listing the valid fields.

---

## Fix Guidance

### 1. Swap priority values in `typePriority`

```go
var typePriority = map[string]int{
    "AI_SERVICE_CONNECTOR": 1,
    "AI_CONNECTION":        2,
    "PROCESS":              3,
    "GUIDE":                4,
    "TASKFLOW":             5,
}
```

### 2. Gate type-priority sort on `publishMode`

In `resolveDependencies`, pass `publishMode` through (it is already a parameter)
and apply the type-priority comparator only when `publishMode == true`. Otherwise
sort by `Path` only.

### 3. Add `--order-by` flag

- Declare `var orderBy string` in `newPackageDependenciesCmd`.
- Register: `cmd.Flags().StringVar(&orderBy, "order-by", "", "sort output by field: path, type, source, targetStatus, warning")`.
- Validate the value against the known field set and return an error for unknown
  values.
- Pass `orderBy` into the sort step; when non-empty it overrides the default
  comparator.

---

## Files Affected

- `cmd/package.go` - `typePriority` map, sort comparator in `resolveDependencies`,
  flag registration in `newPackageDependenciesCmd`
- `docs/documentation/package.md` - document new `--order-by` flag and corrected
  sort behaviour
