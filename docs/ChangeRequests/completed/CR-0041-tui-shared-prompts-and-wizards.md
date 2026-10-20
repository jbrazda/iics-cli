# CR-0041: Shared TUI prompts and interactive wizard refactor

## CR Type

- [x] Enhancement to existing command
- [x] Refactor

## Problem

Interactive mode across the CLI uses line-based helpers in `cmd/user_prompt.go`
(`promptSelect`, `promptMultiSelect`, `promptText`, `promptYesNo`). They print
numbered menus and read typed numbers, which does not scale to real org data:

- Group and role checklists list every group or role and require typing
  comma-separated numbers.
- The time zone prompt is a type-a-query-then-pick loop over ~600 zones.
- Pickers (`pickUserGroup`, `pickAgent`, `pickAgentService`,
  `promptUserSearch`) are numbered menus with no search.
- The runtime wizard loops through "Add agents / Remove agents / Done" menus.

CR-0040 introduced `charmbracelet/huh` and Bubble Tea for the role editor.
The rest of the CLI should use the same style.

## Proposed Solution

### Step 0 - shared prompt layer on huh

Reimplement the shared helpers on huh, keeping their signatures so existing
call sites change behavior without code changes:

| Helper | New behavior |
| ------ | ------------ |
| `promptSelect(label, options)` | `huh.Select`, `/` to filter, cursor keys; returns `-1` on Esc/Ctrl-C |
| `promptMultiSelect(label, options, defaults)` | `huh.MultiSelect` with defaults pre-checked, `/` to filter, `space` toggle, `ctrl+a` select all |
| `promptText(label, default)` | `huh.Input` with the default pre-filled |
| `promptYesNo(label, defaultYes)` | `huh.Confirm` |
| `promptPassword` / `promptPasswordConfirm` | `huh.Input` with password echo mode |

- Move the implementations to `internal/tui/prompt.go`; keep thin wrappers in
  `cmd/user_prompt.go` so call sites are unchanged.
- Add generic `tui.PickOne[T]` / `tui.PickMany[T]` helpers that take items and
  a label function, used by the pickers below.
- Non-terminal stdin or stderr (answers piped in by scripts, stderr
  redirected): keep the existing line-based prompts, moved to
  `internal/tui` with one shared buffered reader. huh accessible mode is not
  used here because its line protocol differs (for example a multi-select
  toggles one number per line), which would break scripted answers.
- `IICS_ACCESSIBLE=1` runs the huh forms in accessible mode on a terminal,
  for screen readers.
- Output goes to stderr, as today.

### 1. User wizard (`user create`, `user update`)

`runUserWizard` (18 prompts) becomes one huh form with pages:

1. **Identity** - authentication type (Native/SSO), first name, last name,
   user name, email (validated), phone, title, description.
2. **Account** - state, force password change, time zone (filterable select
   over the zone list, current value pre-selected).
3. **Membership** - groups and roles as filterable multi-selects with current
   membership pre-checked.
4. **Review** (update only) - changed fields and `+/-` group and role changes,
   then Apply / Back / Cancel.

### 2. User group wizard (`usergroup create`, `usergroup update`)

`runGroupWizard`: name and description inputs (create only, as today), roles
as a filterable multi-select with current roles pre-checked, and a review of
`+/-` role changes on update. Shares the role option builder with the user
wizard.

### 3. Runtime environment wizard (`runtime create`, `runtime update`)

`runRuntimeCreateWizard`: replace the add/remove menu loop with one filterable
multi-select of agents (assigned agents pre-checked; agents assigned to
another environment shown with a note and excluded), then a review of
`+/-` agents, then apply.

### 4. Pickers

`pickUserGroup`, `pickAgent`, `pickAgentService`, `promptUserSearch`,
`pickRole` use `tui.PickOne` with filtering. `promptUserSearch` lists users
directly (filterable) instead of the query loop.

### Out of scope

- Delete confirmations (`fmt.Scanln` in 7 commands) stay plain `[y/N]` so
  scripts can pipe answers; they are consolidated into one `confirmDelete`
  helper as a cleanup.
- `profile add` / `profile edit` (prompting lives in `internal/config`) -
  separate CR.
- `permission set` grid - CR-0042.

## Implementation

- `internal/tui/prompt.go` - huh-based `Select`, `MultiSelect`, `Input`,
  `Confirm`, `Password`, `PickOne[T]`, `PickMany[T]`, accessible mode
  detection.
- `cmd/user_prompt.go` - wrappers delegate to `internal/tui`; `promptTimezone`
  becomes a filterable select.
- `cmd/user.go` - `runUserWizard` rewritten as a paged form plus review.
- `cmd/usergroup_prompt.go` - `runGroupWizard`, `pickUserGroup`.
- `cmd/runtime_prompt.go` - agent multi-select plus review.
- `cmd/agent_service.go` - `pickAgent`, `pickAgentService`.
- `cmd/role_prompt.go` - `pickRole` uses `tui.PickOne`.
- New helper `confirmDelete` replacing duplicated `fmt.Scanln` blocks.
- Tests: `internal/tui/prompt_test.go` (accessible-mode line input for each
  helper, defaults, cancel), diff helpers for group/role/agent review.
- Docs: shared `docs/documentation/interactive.md` (keys, accessible mode,
  piped input format) linked from `user.md`, `group.md`, `environment.md`,
  `agent.md`, `role.md`; `IICS_ACCESSIBLE` in the README environment table;
  `make completions`.

## Acceptance Criteria

- [x] Existing interactive commands work with arrow keys and `/` filtering
      (Step 0)
- [x] Piped answers on non-terminal stdin still work (line-based fallback,
      Step 0)
- [x] User wizard: paged form, filterable time zone (update only; the
      create API has no time zone), group and role multi-selects
      pre-checked, review on update. Applying updates is blocked by the
      existing V2 update failure
      (`docs/issues/new/2026-09-30-user-update-v2-403.md`).
- [x] User group and runtime wizards: pre-checked multi-select plus review
      (runtime wizard is create-only, as before; agent checklist not
      exercised live because `dev` has no unassigned agents)
- [x] All pickers are filterable (`pickRole`, `pickUserGroup`, `pickAgent`,
      `pickAgentService` use `tui.PickOne`)
- [x] Delete confirmations unchanged in behavior, deduplicated into
      `confirmAction` (12 call sites; `user delete` writes to stderr and
      `login` asks `[Y/n]`, so both keep their own prompt)
- [x] Live test on `dev`: create/update/delete a throwaway user, group and
      runtime environment through the wizards (user update blocked by
      `docs/issues/new/2026-09-30-user-update-v2-403.md`)
- [x] `go build`, `go vet`, `golangci-lint`, `go test ./...` pass
