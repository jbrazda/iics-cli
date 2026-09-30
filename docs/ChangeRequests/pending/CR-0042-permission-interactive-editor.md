# CR-0042: Interactive object permission editor

## CR Type

- [x] Enhancement to existing command
- [x] New command

## Dependencies

- Bug `docs/issues/pending/2026-09-30-permission-client-wrong-api-shape.md`
  (fixed in `089f26c`: correct ACL structs and per-ACL endpoints).
- Reuses the TUI foundation from CR-0040 and CR-0041.

## Problem

Object permissions (ACLs) can only be changed by writing JSON. An ACL grants a
user or group five permissions on one object (read, update, delete, execute,
change permission), which is naturally a grid of principals by permission
types - the same shape as the role privilege matrix from CR-0040. Sharing an
object with several groups currently takes several hand-written requests.

## Proposed Solution

### `permission edit [--object-id <id> | --path <project/folder/asset> --type <type>]`

1. **Object selection** - `--object-id`, or `--path` and `--type` resolved
   with the existing lookup client. Without either on a terminal, a filterable
   picker over projects and folders, then assets in the chosen location.
2. **ACL grid** - Bubble Tea model, generalized from
   `internal/tui/privmatrix`:

   ```text
   Object: Default/Sales/m_load_orders (DTEMPLATE)
   PRINCIPAL                      TYPE    READ  UPDATE  DELETE  EXEC  PERM
   > Everyone                     GROUP   [x]   [x]     [ ]     [x]   [ ]
     Data Engineering             GROUP   [x]   [+]     [ ]     [x]   [ ]
     larry@infa.com               USER    [x]   [x]     [x]     [x]   [x]
     analyst@infa.com (new)       USER    [+]   [ ]     [ ]     [ ]   [ ]
   n add principal  d remove principal  space toggle  a row  c column  enter done
   ```

   - `n` opens a filterable picker of users and user groups not yet in the
     ACL list; the new row starts with `read` checked.
   - `d` clears a principal's row; a principal with no permissions has its
     ACL deleted (shown as `[-]` cells).
   - Same marks as CR-0040: `[x]` unchanged, `[+]` grant, `[-]` revoke.
3. **Review** - per principal: create, update (changed permissions), or
   delete; then Apply / Back / Cancel.
4. **Apply** - `POST` for new principals, `PUT <ACL ID>` for changed ones,
   `DELETE <ACL ID>` for removed ones. Stops at the first error and reports
   which changes were applied.

### Guardrails

- Before applying, call `checkAccess` for the object; if the current user
  lacks `changePermission`, stop with an error before any change.
- Warn when the change removes the current user's own `changePermission`.

### Non-interactive equivalent

`permission set --object-id <id> --from-file acls.json` (from the bug fix)
applies the same diff declaratively, so the TUI and scripts share one
`DiffACLs` function.

## Implementation

- `internal/client/permissions.go` - `DiffACLs(current, desired)` returning
  create/update/delete sets (pure, tested).
- `internal/tui/grid/` - extract the reusable grid core (cursor, toggles,
  row/column toggle, filter, marks, scrolling) from `privmatrix`; `privmatrix`
  and the new ACL grid become thin adapters. Existing privmatrix tests keep
  passing.
- `internal/tui/acleditor.go` - object picker, grid, add-principal picker,
  review.
- `cmd/permission.go` - `permission edit`.
- Tests: `DiffACLs`, grid core, ACL adapter (add/remove principal rows).
- Docs: `docs/documentation/permission.md` (`permission edit`, keys,
  guardrails), `README.md` row, `make completions`.

## Acceptance Criteria

- [x] `permission edit` shows current ACLs as a grid and applies
      create/update/delete changes after review
- [x] Principals can be added from a filterable user/group list and removed
- [x] Missing `changePermission` is detected before any change
- [x] `permission set --from-file` and the editor share `DiffACLs`
- [x] Role privilege editor behavior unchanged after the grid extraction
- [x] Live test on `dev` against a throwaway object, restoring ACLs afterwards
- [x] `go build`, `go vet`, `golangci-lint`, `go test ./...` pass

## Implementation Notes

- `internal/tui/grid/` - generic grid core (cells, marks, cursor, row/column
  toggles, filter with section headers, scrolling, footer detail, clear-row
  key, extra action keys that end the grid, empty-state text).
- `internal/tui/privmatrix/` - now a thin adapter building grid rows from a
  privilege service; `role edit` behavior unchanged (tests rewritten against
  the public model API, same scenarios).
- `internal/tui/acleditor.go` - `EditACLs`: grid, `n` add-principal picker,
  review with `ACLPlanLines` and self-lockout warning; returns a
  `client.ACLPlan` from `client.DiffACLs(current, desired, prune=true)`.
- `cmd/permission_edit.go` - object resolution (`--object-id`,
  `--path/--type`, or project/folder/asset picker via
  `q=location=='<path>'`), `checkAccess` guard, apply with
  `applyACLPlan` (shared with `permission set`).
- Verified on `dev` with a throwaway project and folder: add principal,
  toggle, clear row, review, apply; object picker drill-down; no-change
  review. The location query also returns the location object itself, so
  it is filtered out of its own child list.
