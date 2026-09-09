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
  `config.IsTerminal()` gate. `resolveAgent` now returns the full `*Agent`.
- `internal/client/agents.go` - `SetAgentServiceState` documented as taking the
  agent's **federatedId** (see below).
- Docs: `docs/documentation/agent.md`; `make completions`.

## Bug fix folded in

`POST public/core/v3/agent/service` rejected the v2 agent `id` with
`AgentServiceV3APIError_003` "Invalid agent" (the caveat noted in CR-0030). The
`agentId` field must be the agent's **federatedId**. `cmd/agent.go` now resolves
the full agent and passes `agent.FederatedID`; a guard errors if the agent has
no federatedId. Start/stop verified end-to-end live.
