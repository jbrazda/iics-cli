# CR: [Short Title]

<!--
HOW TO USE THIS TEMPLATE
─────────────────────────
1. Fill in every section. Incomplete CRs lead to wrong assumptions and scope creep.
2. Save this file as docs/ChangeRequests/<YYYY-MM-DD>-<slug>.md.
3. When handing to Claude, say:
   "Implement the CR in docs/ChangeRequests/<file>.md. Read that file and docs/CLAUDE.md first."
4. Claude MUST read docs/CLAUDE.md before writing any code.
-->

---

## CR Type

> Tick exactly one. Determines which files need to change.

- [ ] **New resource** — add a brand-new `iics <resource>` command tree (all 4 files required)
- [ ] **New subcommand** — add a subcommand to an existing resource (client method + cmd wiring)
- [ ] **Enhancement** — change behaviour of an existing command (modify specific files)
- [ ] **Output change** — add/remove/rename columns, change default format, fix display
- [ ] **Flag / config change** — add/rename/remove a CLI flag or config field
- [ ] **Refactor** — internal restructuring with no behaviour change (rare; justify below)

---

## Problem

> One paragraph. What is missing or wrong today? Why does it matter?

---

## Desired Change

> Precise description of the new behaviour. Write as if specifying a contract:
> "Command `iics X Y` should do Z. Flag `--foo` controls W."
> Reference the Informatica API docs URL if this involves a new or changed API call.

**Informatica API docs (if applicable):**
`https://docs.informatica.com/cloud-common-services/administrator/current-version/rest-api-reference/`

---

## Scope

### Files to CREATE (new resource CRs only)

```text
internal/client/<resource>s.go          # struct + CRUD methods
internal/client/<resource>s_test.go     # tests for each client method
cmd/<resource>.go                       # thin command wiring
```

Also add `rootCmd.AddCommand(new<Resource>Cmd())` in `cmd/root.go init()`.

### Files to MODIFY

```text
# List each file and what changes — be specific
internal/client/<resource>s.go    # add X method / change Y field
cmd/<resource>.go                 # wire new subcommand / add flag
cmd/root.go                       # register new top-level command (new resource only)
```

### Files to READ (context only — do NOT modify)

```text
docs/CLAUDE.md                          # mandatory: patterns and rules
internal/client/client.go               # do() / doJSON() / doJSONWithQuery() signatures
internal/client/objects_test.go         # newTestClient() helper definition
internal/output/formatter.go            # Column struct, Formatter interface
cmd/root.go                             # getClient(), getFormatter(), global flags
```

> Add a reference implementation file when pattern-matching is useful:
> e.g. `internal/client/users.go` as a reference for a similar new resource.

### Forbidden (do NOT touch)

```text
# List dirs/files that must not change at all, e.g.:
internal/config/        # no config changes needed
internal/output/        # output layer is correct as-is
```

---

## API Details

> Required for any CR involving a new or changed API call.
> Wrong JSON tags are the #1 source of bugs — verify every field name against the API docs.

| Field          | Value                                             |
| -------------- | ------------------------------------------------- |
| API version    | V2 (`api/v2`) / V3 (`public/core/v3`)             |
| HTTP method    | GET / POST / PUT / PATCH / DELETE                 |
| Endpoint path  | e.g. `public/core/v3/widgets`                     |
| Session header | auto-detected by `do()` — no manual action needed |
| Request body   | yes / no                                          |
| Response type  | single object / array / empty 204                 |

**Request body fields (POST/PUT/PATCH):**

```json
{
  "fieldName": "type and description",
  "anotherField": "..."
}
```

**Response JSON (sample or schema):**

```json
{
  "id": "string — read-only",
  "name": "string — required",
  "description": "string — optional",
  "orgId": "string — read-only",
  "createTime": "string — read-only",
  "updateTime": "string — read-only"
}
```

> Use a separate unexported request struct when the response has read-only fields
> that must NOT be sent in the request body. See `folderRequest` in `internal/client/folders.go`.

---

## Implementation Instructions

> Step-by-step, ordered. Claude executes these in sequence.
> Be explicit; avoid "update as needed" — say exactly what to add or change.

### Step 1 — Client layer (`internal/client/`)

1. Define the Go struct(s) with JSON tags matching the API response exactly:
   - Required fields: no `omitempty`
   - Optional fields: add `omitempty`
   - Read-only fields present in response but not in request: use a separate request struct
   - Arrays of objects: use typed structs, never `[]string`
2. Define `<Resource>ListOptions` with `Limit int` and `Skip int` (and any filter fields).
3. Implement the following methods (tick which apply):
   - [ ] `List<Resource>s(ctx, opts) ([]Resource, error)` — uses `doJSONWithQuery`
   - [ ] `Get<Resource>(ctx, id) (*Resource, error)` — uses `doJSON` GET
   - [ ] `Create<Resource>(ctx, r) (*Resource, error)` — uses `doJSON` POST
   - [ ] `Update<Resource>(ctx, id, r) (*Resource, error)` — uses `doJSON` PUT or PATCH
   - [ ] `Delete<Resource>(ctx, id) error` — uses `doJSON` DELETE
   - [ ] `<Other>(ctx, ...) (..., error)` — describe: ___
4. Use `BaseAPIPathV2` or `BaseAPIPathV3` constant — not a hard-coded string.

### Step 2 — Tests (`internal/client/<resource>s_test.go`)

For each client method:

- Assert the HTTP method (`r.Method`)
- Assert the URL path (`r.URL.Path`)
- For query params: assert `r.URL.Query().Get("key")`
- For request bodies: decode and assert key fields
- Return a minimal valid JSON fixture
- Assert at least one field on the returned struct

Use `newTestClient(handler)` from `internal/client/objects_test.go`.

### Step 3 — Command layer (`cmd/<resource>.go`)

Follow the thin-cmd rule strictly:

1. `new<Resource>Cmd()` returns a parent `*cobra.Command` with `Use: "<resource>"`.
2. One `new<Resource><Verb>Cmd()` function per subcommand.
3. Each `RunE` does exactly: `getClient` → client method → `getFormatter` → `f.Format`.
4. Flags map 1:1 to `<Resource>ListOptions` fields.
5. Output columns: `[]output.Column` with `Field` matching the JSON tag (dot notation for nested).
6. Delete commands: require `--yes / -y` flag; prompt if absent.
7. Create/Update with many fields: use `--from-file` JSON file pattern.

### Step 4 — Register (new resource only)

In `cmd/root.go` `init()`:

```go
rootCmd.AddCommand(new<Resource>Cmd())
```

### Step 5 — Verify

```bash
/opt/local/bin/go build ./...
/opt/local/bin/go test ./...
/opt/local/bin/go vet ./...
golangci-lint run ./...
```

All must pass with zero new errors.

---

## Output Columns

> Define the table columns for `list` and `get` subcommands.
> `Field` must be the JSON tag of the struct field, not the Go field name.

| Header      | Field (JSON tag) | Width | Notes |
| ----------- | ---------------- | ----- | ----- |
| ID          | `id`             | 24    |       |
| NAME        | `name`           | 30    |       |
| DESCRIPTION | `description`    | 40    |       |
| UPDATED     | `updateTime`     | 22    |       |
| CREATED BY  | `createdBy`      | 20    |       |

> For nested fields use dot notation: `Field: "status.state"`.

---

## Acceptance Criteria

> Claude self-verifies each item before declaring done.

- [ ] `iics <resource> list` returns data in table, JSON, and CSV formats
- [ ] `iics <resource> get <id>` returns a single resource
- [ ] Create/update/delete subcommands behave as described
- [ ] All new client methods have tests; all assertions pass
- [ ] `go build ./...` succeeds with no errors
- [ ] `go test ./...` passes with no failures
- [ ] `go vet ./...` reports no issues
- [ ] `golangci-lint run ./...` reports no new issues
- [ ] No unrelated code was modified
- [ ] No new external dependencies were added
- [ ] Two-layer rule is respected: no API logic in `cmd/`, no Cobra in `internal/client/`

---

## Do NOT

- Refactor, reformat, or add comments to code outside the CR scope
- Modify test files other than the one for the new/changed resource
- Add error handling for scenarios that cannot happen
- Create helpers or abstractions used only once
- Use `os.Exit()` — return errors from `RunE`
- Use tablewriter v0.x API (`NewWriter`, `SetHeader`, `[]string` rows, `Render()` without error)
- Hard-code base API paths — use `BaseAPIPathV2` / `BaseAPIPathV3` constants
- Guess JSON field names — verify every tag against the API sample response above
- Add features beyond what is explicitly described in "Desired Change"
- Add `Co-Authored-By` trailers to commit messages

---

## Implementation Notes (filled in by Claude during implementation)

> Claude documents decisions made during implementation that deviate from or extend the instructions above.

-
