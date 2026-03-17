# BUG: objects list search filters are broken and do not match the API

---

## Symptoms

`iics objects list --tag <value>` returns no results or incorrect results.
Multi-filter queries (e.g. `--type MTT --tag production`) silently produce
wrong `q` expressions. The `--tag` flag should not exist as a separate
parameter - the API only accepts a single `q` filter string.

---

## Commands / Reproduction Steps

```bash
# Tag filter silently broken (wrong separator when combined):
iics objects list --type MTT --tag production --profile dev

# Wrong separator in raw query (documented example uses "and" not ";"):
iics objects list --query "type=='DTEMPLATE' and location=='Default/Sales'" --profile dev
```

---

## Expected Behaviour

- Filters combine correctly using `;` as separator.
- `--tag` flag does not exist; tag filtering is expressed via `--query "tag=='production'"`.
- All documented `q` filter fields are usable through `--query`.

---

## Actual Behaviour

- Multiple filters joined with `" and "` (wrong separator - API requires `";"`).
- A `--tag` flag exists as a separate CLI parameter, implying a standalone API param
  that does not exist. Tag filtering only works via the `q` parameter.
- The `--type` flag is also a separate parameter generating `type=='value'` in `q`,
  which works only as a single filter but fails silently when combined with `--query`
  or `--tag` via the wrong separator.

---

## Environment

| Field                      | Value         |
| -------------------------- | ------------- |
| OS                         | macOS 25.3.0  |
| `iics --version`           | dev build     |
| Go version                 | 1.25.0        |
| IICS region                | US            |
| Output format (`--output`) | table         |

---

## Architecture Layer

- [x] **`internal/client/`** - HTTP logic, API structs, request/response handling
- [x] **`cmd/`** - flag parsing, command wiring, output formatting

---

## Likely Affected Files

```text
internal/client/objects.go      - ObjectsListOptions, ListObjects, ListAllObjects
cmd/objects.go                  - flag definitions, --query example string
```

---

## API Details

| Field          | Value                               |
| -------------- | ----------------------------------- |
| API version    | V3 (`public/core/v3`)               |
| HTTP method    | GET                                 |
| Endpoint path  | `public/core/v3/objects`            |
| Session header | `INFA-SESSION-ID`                   |

**Docs reference:**
`https://docs.informatica.com/cloud-common-services/administrator/current-version/rest-api-reference/platform-rest-api-version-3-resources/objects/finding-an-asset.html`

### API query parameters

| Parameter | Type | Notes |
| --------- | ---- | ----- |
| `q`       | String | Only filter parameter. Multiple conditions separated by `;` |
| `limit`   | Int  | Max 200 per page |
| `skip`    | Int  | Pagination offset |

### Supported `q` filter fields and operators

| Field | Operators | Example |
| ----- | --------- | ------- |
| `type` | `==`, `!=` | `type=='MTT'` |
| `location` | `==` | `location=='Default/Sales'` |
| `updateTime` | `<`, `<=`, `==`, `>=`, `>` | `updateTime>="2024-01-01T00:00:00Z"` |
| `updatedBy` | `==`, `!=` | `updatedBy=='jsmith'` |
| `tag` | `==` | `tag=='production'` |
| `sourceControl.checkedOutBy` | `==`, `!=` | `sourceControl.checkedOutBy=='jsmith'` |
| `sourceControl.checkedOutTime` | `<`, `<=`, `==`, `>=`, `>` | |
| `sourceControl.sourceControlled` | `==`, `!=` | `sourceControl.sourceControlled==true` |
| `customAttributes.publishedBy` | `==`, `!=` | |
| `customAttributes.publicationDate` | `<`, `<=`, `==`, `>=`, `>` | |

**Sample API request (correct separator):**

```
GET /public/core/v3/objects?q=location=='Default/Sales';type=='MTT'&limit=200
```

### `Object` struct - missing fields

The API response also returns `sourceControl` and `customAttributes` nested objects
that are not mapped in the current `Object` struct:

```json
{
  "sourceControl": {
    "checkedOutBy": "string",
    "checkedOutTime": "date",
    "hash": "string",
    "lastCheckinBy": "string",
    "lastCheckinTime": "date",
    "lastPullTime": "date",
    "sourceControlled": "boolean"
  },
  "customAttributes": {
    "publishedBy": "string",
    "publicationDate": "date"
  }
}
```

---

## Fix Instructions

1. Read `docs/CLAUDE.md`, `internal/client/objects.go`, and `cmd/objects.go` before writing any code.
2. In `internal/client/objects.go`:
   a. Remove the `Tag` field from `ObjectsListOptions` and the corresponding filter
      building block in `ListObjects` and `ListAllObjects`.
   b. Remove the `Type` field from `ObjectsListOptions` and the corresponding
      filter building block - let users express type filtering through `Query`.
      **Note:** If removing `--type` / `--tag` is too disruptive, an alternative
      is to keep them but merge them into `Query` at the `cmd/` layer before
      passing to the client. Decide based on impact.
   c. Change the filter expression separator from `" and "` to `";"`.
   d. Add `SourceControl` and `CustomAttributes` typed nested structs to `Object`.
3. In `cmd/objects.go`:
   a. Remove `--tag` flag.
   b. Update `--type` handling accordingly (remove flag or merge into `--query`).
   c. Fix the `--query` example to use `;` not `and`.
   d. Update `--output-fields` documentation and `objectsColumnDefs` if new struct
      fields are exposed.
4. Update the command `Example` block and `Long` description with correct `q` filter syntax.
5. Add or update tests in `internal/client/objects_test.go`.
6. Run `/opt/local/bin/go test ./...` and `golangci-lint run ./...`.

---

## Acceptance Criteria

- [ ] `iics objects list --query "tag=='production'"` returns correct results
- [ ] `iics objects list --query "type=='MTT';tag=='production'"` works correctly
- [ ] No `--tag` flag on `objects list`
- [ ] `q` filter conditions are joined with `";"`
- [ ] All existing tests still pass
- [ ] `go vet ./...` and `golangci-lint run ./...` report no new issues

---

## Do NOT

- Refactor or rename unrelated code
- Change the dependencies subcommand
- Add features beyond what the bug describes
- Switch tablewriter to v0.x API patterns
- Call `os.Exit()` - always return errors from `RunE`
