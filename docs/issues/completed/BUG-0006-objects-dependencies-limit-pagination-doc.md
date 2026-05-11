---
id: BUG-0006
title: objects dependencies - wrong default limit, no auto-pagination, wrong doc jq examples
status: new
priority: medium
affects: cmd/objects.go, internal/client/objects.go, docs/documentation/objects.md
---

# BUG-0006: objects dependencies - wrong default limit, no auto-pagination, wrong doc jq examples

## Summary

The `iics objects dependencies` subcommand has three correctness issues:

1. The `--limit` flag defaults to `200`, but the IICS v3 API default page size for
   `GET /public/core/v3/objects/{id}/references` is **50**.
2. There is no auto-pagination. When the result set exceeds one page the user must
   manually iterate with `--skip`, unlike `objects list` which fetches all pages
   automatically when `--limit` is `0`.
3. The doc examples that pipe `iics lookup` output into `jq` use the wrong filter
   expression (`.id` instead of `.[0].id`).

---

## Bug 1: Wrong default limit

### Observed behavior

```bash
iics objects dependencies --id <id> --ref-type uses
```

Sends `?limit=200` to the API. The API spec states the default page size is **50**
and the maximum is not documented to be 200.

### Expected behavior

Default fetches all dependencies automatically (no `--limit` needed for the common case),
in batches of 50, consistent with how `objects list` behaves when `--limit` is omitted.

### Root cause

`cmd/objects.go` line 273:

```go
cmd.Flags().IntVar(&limit, "limit", 200, "max results")
```

The default should be `0` (meaning "all pages") and the client method should use a
batch size of `50` to match the API default.

---

## Bug 2: No auto-pagination

### Observed behavior

`GetObjectDependencies()` in `internal/client/objects.go:141` is a single-page method.
There is no equivalent of `ListAllObjects()` for the references endpoint, so objects
with more than one page of dependencies are silently truncated.

### Expected behavior

When `--limit` is `0` (default), the command should loop through all pages and return
the complete dependency set, the same way `objects list` works.

### Root cause

`internal/client/objects.go` has no pagination loop for the references endpoint.

A new method `GetAllObjectDependencies()` must be added that loops with
`skip += 50` until a page returns fewer than 50 results, and `cmd/objects.go`
must call it when `limit == 0`.

---

## Bug 3: Wrong jq expression in documentation

### Observed behavior

`docs/documentation/objects.md` contains these examples (bash and PowerShell):

```bash
# Resolve ID first, then find dependencies
ID=$(iics lookup --path "Sales/ETL/LoadOrders" --type MTT --output json | jq -r '.id')
iics objects dependencies --id "$ID" --ref-type uses --output json
```

```powershell
$obj = iics lookup --path "Sales/ETL/LoadOrders" --type MTT --output json | ConvertFrom-Json
iics objects dependencies --id $obj.id --ref-type uses --output json
```

### Expected behavior

`iics lookup --output json` outputs a **JSON array** of result objects (`[]LookupResult`),
not a single object. The correct expressions are:

```bash
ID=$(iics lookup --path "Sales/ETL/LoadOrders" --type MTT --output json | jq -r '.[0].id')
```

```powershell
$obj = (iics lookup --path "Sales/ETL/LoadOrders" --type MTT --output json | ConvertFrom-Json)[0]
iics objects dependencies --id $obj.id --ref-type uses --output json
```

### Root cause

`cmd/lookup.go:66` calls `f.Format(resp.Objects, columns)` where `resp.Objects` is
`[]LookupResult`. The JSON formatter outputs a JSON array, not a bare object.
The documentation was written as if the output were a single object.

---

## Fix Guidance

### internal/client/objects.go

Add a new method that paginates automatically using batches of 50:

```go
// GetAllObjectDependencies fetches all dependency references for an object,
// auto-paginating in batches of 50 to match the API default page size.
func (c *Client) GetAllObjectDependencies(ctx context.Context, objectID string, refType string) (*ObjectDependenciesResponse, error) {
    const batchSize = 50
    var result ObjectDependenciesResponse
    skip := 0
    for {
        page, err := c.GetObjectDependencies(ctx, objectID, refType, batchSize, skip)
        if err != nil {
            return nil, err
        }
        result.Uses = append(result.Uses, page.Uses...)
        result.UsedBy = append(result.UsedBy, page.UsedBy...)
        if len(page.Uses)+len(page.UsedBy) < batchSize {
            break
        }
        skip += batchSize
    }
    return &result, nil
}
```

### cmd/objects.go

Change the `--limit` default to `0` and call the new method when `limit == 0`:

```go
cmd.Flags().IntVar(&limit, "limit", 0, "max results; 0 fetches all pages")
```

In `RunE`:

```go
var deps *client.ObjectDependenciesResponse
if limit == 0 {
    deps, err = c.GetAllObjectDependencies(ctx, objectID, refType)
} else {
    deps, err = c.GetObjectDependencies(ctx, objectID, refType, limit, skip)
}
```

### docs/documentation/objects.md

Fix lines 168-170 (bash) and 181-183 (PowerShell) as described above.

---

## Files Affected

- `internal/client/objects.go` - add `GetAllObjectDependencies()`
- `cmd/objects.go` - change `--limit` default to `0`, call new method when limit==0
- `docs/documentation/objects.md` - fix jq and PowerShell examples
