# CR-0040: Interactive role privilege editor and role cloning

## CR Type

- [x] Enhancement to existing command
- [x] New command

## Problem

`role add-privileges`, `role remove-privileges` and `role create` require
privileges to be typed with `--privilege` by name or ID. An organization has
600+ privileges (dev org: 625 across 19 services), so users must look up exact
privilege names with `iics privilege list` first. The existing numbered
multi-select prompt (`promptMultiSelect` in `cmd/user_prompt.go`) does not
scale to a list of this size.

There is also no way to create a role from an existing one; users must copy
every privilege manually.

## Proposed Solution

### Privilege grouping

Privilege names mostly follow `<action>.<object>` with action one of `view`,
`create`, `update`, `delete`, `execute`, `changeperm` (about 475 of 625 in the
dev org, for example `create.data.transfer.task`). Privileges are grouped:

1. By `service` (empty service grouped as `(no service)`).
2. Within a service, by object (`data.transfer.task`) with one column per
   action.
3. Names that do not match the pattern (`feature.mcp.*`, `PROFILE.*`,
   `oi.*`, ...) are listed individually in an "Other" section per service.

### Interactive editor (TUI)

Built with `charmbracelet/huh` for forms and a custom Bubble Tea model for the
privilege matrix.

1. Role picker (when `--id` / `--name` omitted) - filterable select.
2. Service picker - filterable select showing privilege count, assigned
   count and pending changes per service, plus `Review and apply` and
   `Cancel`.
3. Privilege matrix per service:

   ```text
   Role: CAI Viewer > DI (157)                      filter: /task
   OBJECT                    VIEW CREATE UPDATE DELETE EXEC PERM
   data.transfer.task         [x]  [+]    [ ]    [ ]    [ ]  [ ]
   file.listener              [x]  [ ]    [-]    [ ]    [x]  [ ]
   mapping                    [ ]  [ ]    [ ]    [ ]     .    .
   -- Other --
   edc.iics.discovery         [ ]
   create.data.transfer.task - Create Data Transfer Task      (Enabled)
   arrows move  space toggle  a row  c column  / filter  esc back  enter done
   ```

   - `[x]` assigned, `[+]` pending add, `[-]` pending remove, `.` action not
     available for the object.
   - `a` toggles all actions in the row; `c` toggles the column for all
     visible (filtered) rows.
   - Footer shows the focused privilege's full name, description and status.
4. Review - pending additions and removals grouped by service, then confirm.
   Applying is blocked if the role would be left with no privileges (API
   error `V3API_IDSError_088`).

### Commands

| Command | Interactive when | Behavior |
| ------- | ---------------- | -------- |
| `role edit [--id\|--name]` (new) | always (terminal required) | Full editor; applies the diff with add first, then remove |
| `role add-privileges` | `--privilege` omitted on a terminal | Editor in add-only mode (assigned cells locked) |
| `role remove-privileges` | `--privilege` omitted on a terminal | Editor in remove-only mode (only assigned privileges shown) |
| `role create` | `--name` or privileges missing on a terminal | Form: name, description, optional clone source role, then editor pre-checked with the source privileges |
| `role create --from-role <name\|id>` | never | Copies the source role's privileges, and description unless `--description` is given; `--privilege` values are added on top |

Without a terminal, behavior is unchanged: missing flags return an error.

## Implementation

- Dependencies: `github.com/charmbracelet/huh`,
  `github.com/charmbracelet/bubbletea`, `github.com/charmbracelet/bubbles`.
- `internal/client/privilegegroups.go` - `GroupPrivileges` (service, object,
  action grouping) and `DiffPrivileges` (add/remove sets); pure functions.
- `internal/tui/privmatrix/` - Bubble Tea privilege matrix model (navigation,
  toggles, row/column toggles, filter, add-only / remove-only modes).
- `internal/tui/` - editor orchestration: service picker, matrix, review and
  confirm; returns the desired privilege set.
- `cmd/role.go`, `cmd/role_prompt.go` - `role edit`, interactive branches for
  `add-privileges`, `remove-privileges`, `create`; `--from-role` flag;
  `pickRole` moved to a filterable select.
- No new API endpoints; reuses `GetRole`, `ListPrivileges`,
  `AddRolePrivileges`, `RemoveRolePrivileges`, `CreateRole`,
  `ResolvePrivileges`.
- Tests: `internal/client/privilegegroups_test.go`,
  `internal/tui/privmatrix/*_test.go` (model `Update` key handling).
- Docs: `docs/documentation/role.md` (`role edit`, interactive notes, key
  bindings, `--from-role`), `README.md` role row, `make completions`.

## Acceptance Criteria

- [ ] `role edit` adds and removes privileges across services and applies
      them in one confirmed step
- [ ] `role add-privileges` / `role remove-privileges` without `--privilege`
      open the editor in the matching mode
- [ ] `role create --from-role` creates a role with the source role's
      privileges
- [ ] Interactive `role create` supports cloning and editing before create
- [ ] Removing all privileges is blocked before any API call
- [ ] Non-terminal usage is unchanged
- [ ] `go build`, `go vet`, `golangci-lint`, `go test ./...` pass
