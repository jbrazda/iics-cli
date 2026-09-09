# CR-0032: Interactive `iics runtime create`

## CR Type

- [x] Enhancement to existing command

## Problem

`iics runtime create` only accepts `--from-file` and errors when it is omitted.
There is no guided way to create a Secure Agent group and pick its member
agents.

## Proposed Solution

Add an interactive wizard to `runtime create`, triggered when `--from-file` is
omitted on a TTY, or by an explicit `--interactive` / `-i`. With both
`--from-file` and `-i`, interactive wins and the file pre-fills the prompts.

Prompts:

1. `Environment Name:` (required, re-prompt if empty).
2. `Is shared [y/N]:`.
3. Agent management menu (`promptSelect`): `Add agents` / `Remove agents` /
   Done.
   - Add: lists unassigned agents (`ListAgents` with `IncludeUnassignedOnly`),
     multi-select, appended to the working set (deduped).
   - Remove: multi-select over the working set, removes the picks.

The selected agent IDs are sent in the `agents` array of the
`POST /api/v2/runtimeEnvironment` body. Verified live: the endpoint processes the
`agents` array, and each element must carry `"@type": "agent"` (without it the
whole request is rejected with `APP_20149`). `CreateRuntimeEnvironment` /
`UpdateRuntimeEnvironment` inject that discriminator. If the server rejects agent
assignment for another reason, the raw API error is surfaced (fail hard - no
silent degradation); the name/shared prompts still function.

Non-TTY without `--from-file` and without `-i` keeps erroring
(`--from-file or --interactive is required`).

## Out of scope

- Interactive `runtime update` (the v2 update PUT is separately broken, BUG-0015).

## Implementation

- `cmd/runtime.go` / `cmd/runtime_prompt.go` - `--interactive`/`-i` flag,
  `runRuntimeCreateWizard`, reusing `promptText` / `promptYesNo` /
  `promptSelect` / `promptMultiSelect` from `cmd/user_prompt.go` and the
  `config.IsTerminal()` gate (pattern: `runUserWizard`, `cmd/user.go`).
- `internal/client/runtimes.go` - only if live testing shows the create body
  needs a different agent shape.
- Docs: `docs/documentation/runtime.md`, `make completions`.
