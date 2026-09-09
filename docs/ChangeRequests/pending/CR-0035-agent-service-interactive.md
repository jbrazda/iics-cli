# CR-0035: Interactive `agent start` / `agent stop`

## CR Type

- [x] Enhancement to existing command

## Problem

`agent start` / `agent stop` require `--id`/`--name`/`--hostname` and an exact
`--service` display name. Operators rarely remember the exact service name.

## Proposed Solution

Add `--interactive` / `-i` to both commands:

1. List the Secure Agents and prompt for one (skipped when `--id`/`--name`/
   `--hostname` was given).
2. Fetch that agent's service engines (`agent/details`) and prompt for one.
3. Ask for confirmation (`<Verb> service "<name>" on agent <label>`), default yes.
4. Call `POST public/core/v3/agent/service`.

Cancelling at any prompt exits cleanly with `Canceled.`. `--service` is still
required when `--interactive` is not used.

## Implementation

- `cmd/agent.go` - `--interactive` flag on `newAgentServiceCmd`; helpers
  `pickAgent` (reuses `ListAgents`) and `pickAgentService` (reuses
  `GetAgentDetails`), plus `promptSelect` / `promptYesNo` and the
  `config.IsTerminal()` gate.
- Docs: `docs/documentation/agent.md`; `make completions`.

No client changes.
